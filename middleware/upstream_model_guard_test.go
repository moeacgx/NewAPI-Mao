package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/extension"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupUpstreamModelGuardMiddlewareTest(t *testing.T) *model.Channel {
	t.Helper()
	originalManager, originalVersion := extension.DefaultManager, common.Version
	originalDB, originalRedis, originalMemory := model.DB, common.RedisEnabled, common.MemoryCacheEnabled
	originalDatabaseType := common.MainDatabaseType()
	t.Setenv("EXTENSIONS_ROOT", t.TempDir())
	common.Version = "v1.0.0-rc.99.0.0.0"
	t.Cleanup(func() {
		extension.DefaultManager, common.Version = originalManager, originalVersion
		model.DB, common.RedisEnabled, common.MemoryCacheEnabled = originalDB, originalRedis, originalMemory
		common.SetMainDatabaseType(originalDatabaseType)
	})
	require.NoError(t, extension.Init())
	require.NoError(t, os.CopyFS(filepath.Join(extension.DefaultManager.RootDir(), service.UpstreamModelGuardModuleID), os.DirFS("../extensions/upstream-model-guard")))
	require.NoError(t, extension.DefaultManager.Scan())
	_, err := extension.DefaultManager.SetEnabled(service.UpstreamModelGuardModuleID, true)
	require.NoError(t, err)
	require.True(t, service.IsUpstreamModelGuardModuleEnabled())
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "guard-distributor.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	model.DB, common.RedisEnabled, common.MemoryCacheEnabled = db, false, true
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.Group{}, &model.GroupAlias{}))
	require.NoError(t, db.Create(&model.Group{Code: "guard-route", Name: "Guard route", Status: model.GroupStatusActive}).Error)
	channel := &model.Channel{
		Id: 991001, Name: "guard-channel", Type: constant.ChannelTypeOpenAI,
		Key: "guard-key-0\nguard-key-1", Models: "client-model", Group: "guard-route",
		Status: common.ChannelStatusEnabled, Priority: common.GetPointer(int64(10)),
		ChannelInfo: model.ChannelInfo{IsMultiKey: true, MultiKeyMode: constant.MultiKeyModePolling},
	}
	require.NoError(t, db.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
	model.InitChannelCache()
	return channel
}

func TestUpstreamModelGuardRejectsStaleEnabledChannelBeforeKeySelection(t *testing.T) {
	channel := setupUpstreamModelGuardMiddlewareTest(t)
	cached, err := model.CacheGetChannel(channel.Id)
	require.NoError(t, err)
	require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
	require.Equal(t, common.ChannelStatusEnabled, cached.Status)
	ctx := newSelectedChannelContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(ctx, constant.ContextKeyChannelPreferredMultiKeyIndex, 1)
	pollingIndex := cached.ChannelInfo.MultiKeyPollingIndex

	apiErr := SetupContextForSelectedChannel(ctx, cached, "client-model")

	require.NotNil(t, apiErr)
	assert.Contains(t, apiErr.Error(), "停用")
	assert.Empty(t, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	assert.Equal(t, pollingIndex, cached.ChannelInfo.MultiKeyPollingIndex)
}

func TestUpstreamModelGuardAllowsEnabledChannelAndDisabledModule(t *testing.T) {
	channel := setupUpstreamModelGuardMiddlewareTest(t)
	ctx := newSelectedChannelContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	require.Nil(t, SetupContextForSelectedChannel(ctx, channel, "client-model"))
	assert.NotEmpty(t, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	ReleaseChannelConcurrencyForContext(ctx)
	_, err := extension.DefaultManager.SetEnabled(service.UpstreamModelGuardModuleID, false)
	require.NoError(t, err)
	require.False(t, service.IsUpstreamModelGuardModuleEnabled())
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	ctx = newSelectedChannelContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	require.Nil(t, SetupContextForSelectedChannel(ctx, &model.Channel{Id: 991003, Key: "module-disabled-key"}, "client-model"))
	assert.Equal(t, "module-disabled-key", common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
}

func TestUpstreamModelGuardRejectsMissingChannelAndDatabaseFailure(t *testing.T) {
	channel := setupUpstreamModelGuardMiddlewareTest(t)
	ctx := newSelectedChannelContext()
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	missing := *channel
	missing.Id = 991099
	require.NotNil(t, SetupContextForSelectedChannel(ctx, &missing, "client-model"))
	assert.Empty(t, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	apiErr := SetupContextForSelectedChannel(ctx, channel, "client-model")
	require.NotNil(t, apiErr)
	assert.Empty(t, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	assert.NotContains(t, apiErr.Error(), "guard-key")
}

func TestUpstreamModelGuardDistributorFallsBackFromStaleCachedChannel(t *testing.T) {
	for _, route := range []string{"普通选渠", "亲和选渠", "指定渠道"} {
		t.Run(route, func(t *testing.T) {
			channel := setupUpstreamModelGuardMiddlewareTest(t)
			require.NoError(t, i18n.Init())
			originalRetry, originalErrorLog := common.RetryTimes, constant.ErrorLogEnabled
			common.RetryTimes, constant.ErrorLogEnabled = 1, false
			t.Cleanup(func() { common.RetryTimes, constant.ErrorLogEnabled = originalRetry, originalErrorLog })
			fallback := &model.Channel{Id: 991002, Name: "fallback", Key: "fallback-key", Type: constant.ChannelTypeOpenAI, Models: "client-model", Group: "guard-route", Status: common.ChannelStatusEnabled}
			require.NoError(t, model.DB.Create(fallback).Error)
			require.NoError(t, fallback.AddAbilities(nil))
			model.InitChannelCache()
			if route == "亲和选渠" {
				setting := operation_setting.GetChannelAffinitySetting()
				original := *setting
				*setting = operation_setting.ChannelAffinitySetting{
					Enabled: true, DefaultTTLSeconds: 60,
					Rules: []operation_setting.ChannelAffinityRule{{
						Name: "guard-affinity-test", ModelRegex: []string{"^client-model$"},
						KeySources:        []operation_setting.ChannelAffinityKeySource{{Type: "request_header", Key: "X-Guard-Affinity"}},
						BindMultiKeyIndex: true, IncludeRuleName: true,
					}},
				}
				ctx := newSelectedChannelContext()
				ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				ctx.Request.Header.Set("X-Guard-Affinity", t.Name())
				service.GetPreferredChannelAffinityBinding(ctx, "client-model", "guard-route")
				common.SetContextKey(ctx, constant.ContextKeyChannelId, channel.Id)
				common.SetContextKey(ctx, constant.ContextKeyChannelIsMultiKey, true)
				common.SetContextKey(ctx, constant.ContextKeyChannelMultiKeyIndex, 1)
				service.RecordChannelAffinity(ctx, channel.Id)
				boundID, found := service.GetPreferredChannelByAffinity(ctx, "client-model", "guard-route")
				require.True(t, found)
				require.Equal(t, channel.Id, boundID)
				t.Cleanup(func() {
					service.ClearCurrentChannelAffinityCache(ctx)
					*setting = original
				})
			}
			require.NoError(t, model.DB.Model(&model.Channel{}).Where("id = ?", channel.Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
			engine := gin.New()
			engine.Use(func(c *gin.Context) {
				common.SetContextKey(c, constant.ContextKeyUsingGroup, "guard-route")
				common.SetContextKey(c, constant.ContextKeyTokenGroup, "guard-route")
				if route == "指定渠道" {
					common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(channel.Id))
				}
			})
			engine.POST("/v1/chat/completions", BodyStorageCleanup(), Distribute(), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"channel_id": common.GetContextKeyInt(c, constant.ContextKeyChannelId)})
			})
			request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"client-model"}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Guard-Affinity", t.Name())
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			if route == "指定渠道" {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
				return
			}
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var response struct {
				ChannelID int `json:"channel_id"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.Equal(t, fallback.Id, response.ChannelID)
		})
	}
}
