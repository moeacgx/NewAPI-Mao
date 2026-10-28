package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// occupyChannelAfterAvailabilityHook 在首组渠道的可用性 Lua 返回后，
// 用第二个 Redis 客户端真实执行一次抢租 Lua，确定性制造预检查与抢租之间的竞争。
type occupyChannelAfterAvailabilityHook struct {
	client    *redis.Client
	channelID int
	occupied  atomic.Bool
	token     string
	lease     *model.ChannelConcurrencyLease
	err       error
}

func (hook *occupyChannelAfterAvailabilityHook) BeforeProcess(ctx context.Context, _ redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (hook *occupyChannelAfterAvailabilityHook) AfterProcess(_ context.Context, cmd redis.Cmder) error {
	if hook.occupied.Load() || cmd.Err() != nil || cmd.Name() != "eval" && cmd.Name() != "evalsha" {
		return nil
	}
	args := cmd.Args()
	if len(args) != 5 {
		return nil
	}
	redisCmd, ok := cmd.(*redis.Cmd)
	if !ok {
		return nil
	}
	result, err := redisCmd.Int()
	if err != nil || result != 1 {
		return nil
	}
	key, ok := args[3].(string)
	if !ok || key != fmt.Sprintf("new-api:channel-concurrency:%d", hook.channelID) {
		return nil
	}
	// hook 在预检查结果返回后同步调用生产抢租路径。
	previousClient := common.RDB
	common.RDB = hook.client
	limit := 1
	var acquired bool
	hook.lease, acquired = model.TryAcquireChannelConcurrencyLease(&model.Channel{
		Id: hook.channelID, ConcurrencyLimit: &limit,
	})
	common.RDB = previousClient
	if !acquired || hook.lease == nil {
		hook.err = fmt.Errorf("second Redis client could not acquire the competing channel lease")
		return nil
	}
	hook.token = hook.lease.Token
	hook.occupied.Store(true)
	return nil
}

func (hook *occupyChannelAfterAvailabilityHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (hook *occupyChannelAfterAvailabilityHook) AfterProcessPipeline(context.Context, []redis.Cmder) error {
	return nil
}

func TestDistributeRedisConcurrentPrecheckFallsBackWithZeroRetryBudget(t *testing.T) {
	redisServer := miniredis.RunT(t)
	clientA := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	clientB := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		_ = clientA.Close()
		_ = clientB.Close()
		redisServer.Close()
	})

	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMemoryCache, originalRedis, originalRDB, originalRetry := common.MemoryCacheEnabled, common.RedisEnabled, common.RDB, common.RetryTimes
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Group{}))
	model.DB, model.LOG_DB = db, nil
	common.MemoryCacheEnabled, common.RedisEnabled, common.RDB, common.RetryTimes = true, true, clientA, 0
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.MemoryCacheEnabled, common.RedisEnabled, common.RDB, common.RetryTimes = originalMemoryCache, originalRedis, originalRDB, originalRetry
		if originalMemoryCache && originalDB != nil && originalDB.Migrator().HasTable(&model.Channel{}) && originalDB.Migrator().HasTable(&model.Ability{}) {
			model.InitChannelCache()
		}
		sqlDB, closeErr := db.DB()
		if closeErr == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, i18n.Init())

	const (
		userID    = 999100
		modelName = "redis-race-fallback-model"
		firstID   = 2501
		secondID  = 2502
	)
	groups := []model.Group{
		{Id: 999101, Code: "redis-race-first", Name: "首组", SingleUserConcurrencyLimit: 1},
		{Id: 999102, Code: "redis-race-second", Name: "后备组", SingleUserConcurrencyLimit: 1},
	}
	require.NoError(t, db.Create(&groups).Error)
	for _, item := range []struct {
		id    int
		group string
	}{
		{firstID, groups[0].Code},
		{secondID, groups[1].Code},
	} {
		limit, priority, weight := 1, int64(100), uint(100)
		channel := &model.Channel{
			Id: item.id, Type: constant.ChannelTypeOpenAI, Key: fmt.Sprintf("key-%d", item.id),
			Status: common.ChannelStatusEnabled, Name: item.group, Group: item.group, Models: modelName,
			Weight: &weight, Priority: &priority, ConcurrencyLimit: &limit,
		}
		require.NoError(t, db.Create(channel).Error)
		require.NoError(t, db.Create(&model.Ability{
			Group: item.group, Model: modelName, ChannelId: item.id, Enabled: true,
			Priority: &priority, Weight: weight,
		}).Error)
	}
	model.InitChannelCache()
	hook := &occupyChannelAfterAvailabilityHook{client: clientB, channelID: firstID}
	t.Cleanup(func() {
		if hook.lease != nil {
			key := fmt.Sprintf("new-api:channel-concurrency:%d", firstID)
			removed, cleanupErr := clientB.ZRem(context.Background(), key, hook.lease.Token).Result()
			require.NoError(t, cleanupErr)
			require.Equal(t, int64(1), removed, "清理必须精确释放竞争客户端创建的租约")
		}
	})
	clientA.AddHook(hook)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var requestContext *gin.Context
	relayEntered := false
	authorizedGroups := []string{groups[0].Code, groups[1].Code}
	tokenGroup := strings.Join(authorizedGroups, ",")
	router.Use(func(c *gin.Context) {
		requestContext = c
		c.Set("id", userID)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, tokenGroup)
		common.SetContextKey(c, constant.ContextKeyTokenGroup, tokenGroup)
		common.SetContextKey(c, constant.ContextKeyTokenGroups, authorizedGroups)
	})
	router.POST("/v1/chat/completions", Distribute(), func(c *gin.Context) {
		relayEntered = true
		assert.Equal(t, secondID, common.GetContextKeyInt(c, constant.ContextKeyChannelId))
		assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
		assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeySelectedChannelGroup))
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"redis-race-fallback-model"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.True(t, hook.occupied.Load(), "应在首组预检查后由第二 Redis 客户端占满首组渠道")
	require.NoError(t, hook.err)
	require.NotNil(t, requestContext)
	// RetryTimes=0 仍应在原子抢租失败后继续后备授权组。
	assert.Equal(t, http.StatusNoContent, recorder.Code, recorder.Body.String())
	assert.True(t, relayEntered)
	assert.Equal(t, groups[1].Code, common.GetContextKeyString(requestContext, constant.ContextKeyUsingGroup))
	assert.Equal(t, groups[1].Code, common.GetContextKeyString(requestContext, constant.ContextKeySelectedChannelGroup))

	// 请求只能释放自己的后备组租约，竞争客户端持有的首组槽位必须仍然存在。
	firstKey := fmt.Sprintf("new-api:channel-concurrency:%d", firstID)
	assert.Equal(t, int64(1), clientB.ZCard(context.Background(), firstKey).Val())
	secondLease, secondAvailable := model.TryAcquireChannelConcurrencyLease(&model.Channel{Id: secondID, ConcurrencyLimit: common.GetPointer(1)})
	assert.True(t, secondAvailable)
	if secondLease != nil {
		assert.True(t, model.ReleaseChannelConcurrencyLease(secondLease))
	}
	assert.Equal(t, int64(1), clientB.ZCard(context.Background(), firstKey).Val())
}
