package controller

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// occupyControllerChannelHook 在首组可用性检查返回后，用第二个 Redis 客户端占用首组渠道。
// 这样可以确定性覆盖 getChannel 的预检查与原子抢租之间的竞争窗口。
type occupyControllerChannelHook struct {
	client    *redis.Client
	channelID int
	fired     atomic.Bool
	lease     *model.ChannelConcurrencyLease
	err       error
}

func (hook *occupyControllerChannelHook) BeforeProcess(ctx context.Context, _ redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (hook *occupyControllerChannelHook) AfterProcess(_ context.Context, cmd redis.Cmder) error {
	if hook.fired.Load() || cmd.Err() != nil || (cmd.Name() != "eval" && cmd.Name() != "evalsha") || len(cmd.Args()) != 5 {
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
	key, ok := cmd.Args()[3].(string)
	if !ok || key != fmt.Sprintf("new-api:channel-concurrency:%d", hook.channelID) || !hook.fired.CompareAndSwap(false, true) {
		return nil
	}
	previousClient := common.RDB
	common.RDB = hook.client
	limit := 1
	var acquired bool
	hook.lease, acquired = model.TryAcquireChannelConcurrencyLease(&model.Channel{Id: hook.channelID, ConcurrencyLimit: &limit})
	common.RDB = previousClient
	if !acquired || hook.lease == nil {
		hook.err = fmt.Errorf("competing Redis client could not acquire channel lease")
	}
	return nil
}

func (*occupyControllerChannelHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (*occupyControllerChannelHook) AfterProcessPipeline(context.Context, []redis.Cmder) error {
	return nil
}

func TestGetChannelFallsBackAfterConcurrentSetupFailureWithoutRetryBudget(t *testing.T) {
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
		if sqlDB, closeErr := db.DB(); closeErr == nil {
			_ = sqlDB.Close()
		}
	})

	const modelName = "controller-race-fallback-model"
	groups := []model.Group{
		{Id: 999201, Code: "controller-race-first", Name: "首组", SingleUserConcurrencyLimit: 1},
		{Id: 999202, Code: "controller-race-second", Name: "后备组", SingleUserConcurrencyLimit: 1},
	}
	require.NoError(t, db.Create(&groups).Error)
	for index, group := range groups {
		channelID := 2511 + index
		limit, priority, weight := 1, int64(100), uint(100)
		require.NoError(t, db.Create(&model.Channel{
			Id: channelID, Type: constant.ChannelTypeOpenAI, Key: fmt.Sprintf("key-%d", channelID),
			Status: common.ChannelStatusEnabled, Name: group.Code, Group: group.Code, Models: modelName,
			Weight: &weight, Priority: &priority, ConcurrencyLimit: &limit,
		}).Error)
		require.NoError(t, db.Create(&model.Ability{Group: group.Code, Model: modelName, ChannelId: channelID, Enabled: true, Priority: &priority, Weight: weight}).Error)
	}
	model.InitChannelCache()

	hook := &occupyControllerChannelHook{client: clientB, channelID: 2511}
	clientA.AddHook(hook)
	t.Cleanup(func() {
		if hook.lease != nil {
			assert.Equal(t, int64(1), clientB.ZRem(context.Background(), fmt.Sprintf("new-api:channel-concurrency:%d", hook.channelID), hook.lease.Token).Val())
		}
	})

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	t.Cleanup(func() {
		middleware.ReleaseChannelConcurrencyForContext(c)
		middleware.ReleaseGroupUserConcurrencyForContext(c)
	})
	c.Set("id", 999200)
	request := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	c.Request = request
	authorizedGroups := []string{groups[0].Code, groups[1].Code}
	tokenGroup := strings.Join(authorizedGroups, ",")
	common.SetContextKey(c, constant.ContextKeyUsingGroup, tokenGroup)
	common.SetContextKey(c, constant.ContextKeyTokenGroup, tokenGroup)
	common.SetContextKey(c, constant.ContextKeyTokenGroups, authorizedGroups)
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{}, TokenGroup: tokenGroup, UserGroup: groups[0].Code,
		UsingGroup: groups[0].Code, OriginModelName: modelName,
	}
	retry := &service.RetryParam{Ctx: c, TokenGroup: tokenGroup, ModelName: modelName, RequestPath: request.URL.Path, Retry: common.GetPointer(0)}

	channel, apiErr := getChannel(c, info, retry)
	require.True(t, hook.fired.Load(), "应在首组预检查后占用首组渠道")
	require.NoError(t, hook.err)
	require.Nil(t, apiErr)
	require.NotNil(t, channel)
	assert.Equal(t, 2512, channel.Id)
	assert.Equal(t, groups[1].Code, info.UsingGroup)
	assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeySelectedChannelGroup))
	assert.Equal(t, groups[1].Code, common.GetContextKeyString(c, constant.ContextKeyUsingGroup))
	assert.False(t, model.IsChannelConcurrencyAvailable(&model.Channel{Id: 2512, ConcurrencyLimit: common.GetPointer(1)}))
}
