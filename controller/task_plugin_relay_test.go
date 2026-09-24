package controller

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	kitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPluginPublicViewAndAuthenticatedInlineResources(t *testing.T) {
	setupCanvasControllerDB(t)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}, &model.Ability{}))
	require.NoError(t, service.InitTaskPlugins())
	row, err := model.GetTaskPluginVersion("sora", "")
	require.NoError(t, err)
	group := model.Group{Code: "default", Name: "Default", Status: model.GroupStatusActive, UserSelectable: true, Ratio: 1}
	require.NoError(t, model.DB.Create(&group).Error)
	ch := model.Channel{Id: 8201, Type: constant.ChannelTypeTaskPlugin, Status: common.ChannelStatusEnabled, Group: group.Code, Models: "sora-2", Setting: common.GetPointer(`{"task_plugin_key":"sora"}`)}
	require.NoError(t, model.DB.Create(&ch).Error)
	require.NoError(t, model.DB.Create(&model.ChannelGroupBinding{ChannelId: ch.Id, GroupId: group.Id}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{ChannelId: ch.Id, Group: group.Code, GroupId: group.Id, Model: "sora-2", Enabled: true}).Error)
	task := model.Task{TaskID: "task_public_resource", Platform: "sora", UserId: 8201, ChannelId: ch.Id, Group: group.Code, Status: model.TaskStatusSuccess, Progress: "100%", Properties: model.Properties{OriginModelName: "sora-2"}, PrivateData: model.TaskPrivateData{Key: "PRIVATE_AUTHORIZATION", UpstreamTaskID: "PRIVATE_UPSTREAM_ID", PluginData: []byte(`{"signed_url":"https://example.invalid?access=PRIVATE_ACCESS"}`), PluginResultURL: "data:image/png;base64,iVBORw==", Execution: &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, SourceHash: row.SourceHash, SourceKind: "builtin", APIVersion: row.APIVersion}}}}
	require.NoError(t, task.Insert())
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		id, _ := strconv.Atoi(c.GetHeader("X-Test-User"))
		c.Set("id", id)
		common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
		common.SetContextKey(c, constant.ContextKeyTokenGroupMode, model.TokenGroupModeInherit)
		c.Next()
	})
	engine.GET("/v1/task/plugins/:plugin_key/:task_id", GetOfficialPluginTask)
	engine.GET("/v1/task/plugins/:plugin_key/:task_id/artifacts/:artifact_key", GetOfficialPluginArtifacts)
	path := "/v1/task/plugins/sora/task_public_resource"
	for _, tc := range []struct {
		user, path string
		status     int
	}{
		{"", path, 404}, {"8202", path, 404}, {"8201", path, 200},
		{"", path + "/artifacts/result?access=PRIVATE_ACCESS", 404}, {"8202", path + "/artifacts/result", 404}, {"8201", path + "/artifacts/result", 200},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req.Header.Set("X-Test-User", tc.user)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		assert.Equal(t, tc.status, rec.Code, rec.Body.String())
		for _, marker := range []string{"PRIVATE_AUTHORIZATION", "PRIVATE_UPSTREAM_ID", "PRIVATE_ACCESS", "signed_url"} {
			assert.NotContains(t, rec.Body.String(), marker)
		}
		assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		if tc.status == 200 && strings.Contains(tc.path, "artifacts") {
			assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, rec.Body.Bytes())
			assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		}
	}
	for _, url := range []string{"https://example.invalid?access=PRIVATE_ACCESS", "data:image/svg+xml;base64,PHN2Zz4="} {
		task.PrivateData.PluginResultURL = url
		require.NoError(t, model.DB.Save(&task).Error)
		req := httptest.NewRequest(http.MethodGet, path+"/artifacts/result", nil)
		req.Header.Set("X-Test-User", "8201")
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if strings.HasPrefix(url, "https") {
			assert.Equal(t, 501, rec.Code)
		} else {
			assert.Equal(t, 415, rec.Code)
		}
		assert.NotContains(t, rec.Body.String(), "PRIVATE_ACCESS")
	}
	require.NoError(t, model.DB.Model(&group).Update("status", model.GroupStatusDisabled).Error)
	req := httptest.NewRequest(http.MethodGet, path+"/artifacts/result", nil)
	req.Header.Set("X-Test-User", "8201")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, 404, rec.Code)
}

func TestTaskPluginPollingErrorsDoNotExposeProviderPayload(t *testing.T) {
	setupCanvasControllerDB(t)
	model.InitDBColumns()
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, model.DB.AutoMigrate(&model.Option{}, &model.TaskPlugin{}, &model.Channel{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}))
	require.NoError(t, service.InitTaskPlugins())
	row, err := model.GetTaskPluginVersion("hailuo", "")
	require.NoError(t, err)
	pin := &model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, APIVersion: row.APIVersion, SourceHash: row.SourceHash, SourceKind: "builtin"}
	const payload = `{"error":{"http_code":429,"message":"REVIEW_FAKE_SECRET signed_url=example.invalid?access=FAKE Authorization=REVIEW_SECRET"}}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, payload)
	}))
	defer server.Close()
	service.InitHttpClient()
	task := model.Task{TaskID: "task_public_poll", Platform: "hailuo", ChannelId: 8101, Status: model.TaskStatusInProgress, Progress: "30%", Properties: model.Properties{OriginModelName: "MiniMax-H3"}, PrivateData: model.TaskPrivateData{UpstreamTaskID: "provider-private-id", Key: "provider-secret", Execution: &model.TaskExecutionSnapshot{TaskPlugin: pin}, PluginData: []byte(`{}`)}}
	require.NoError(t, model.DB.Create(&task).Error)
	ch := model.Channel{Id: task.ChannelId, Type: constant.ChannelTypeTaskPlugin, Name: "error-channel", Status: common.ChannelStatusEnabled, BaseURL: &server.URL, Key: "provider-secret", Setting: common.GetPointer(`{"task_plugin_key":"hailuo"}`)}
	require.NoError(t, model.DB.Create(&ch).Error)
	adaptor, err := relay.GetPinnedTaskAdaptor(&task)
	require.NoError(t, err)
	adaptor.Init(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: server.URL, ApiKey: ch.Key}})
	resp, err := adaptor.FetchTask(server.URL, ch.Key, nil, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	_, err = adaptor.ParseTaskResult(body)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stage=parse_failed")
	assert.Contains(t, err.Error(), "task_public_poll")
	for _, marker := range []string{"REVIEW_FAKE_SECRET", "signed_url", "access=FAKE", "Authorization", "provider-private-id"} {
		assert.NotContains(t, err.Error(), marker)
	}
	oldCache, oldDebug := common.MemoryCacheEnabled, common.DebugEnabled
	oldFactory, oldPinned := service.GetTaskAdaptorFunc, service.GetPinnedTaskAdaptorFunc
	var logs bytes.Buffer
	common.LogWriterMu.Lock()
	oldWriter, oldError := gin.DefaultWriter, gin.DefaultErrorWriter
	gin.DefaultWriter = &logs
	gin.DefaultErrorWriter = &logs
	common.LogWriterMu.Unlock()
	common.MemoryCacheEnabled = false
	common.DebugEnabled = true
	service.GetTaskAdaptorFunc = func(p constant.TaskPlatform) service.TaskPollingAdaptor { return relay.GetTaskAdaptor(p) }
	service.GetPinnedTaskAdaptorFunc = func(task *model.Task) (service.TaskPollingAdaptor, error) { return relay.GetPinnedTaskAdaptor(task) }
	t.Cleanup(func() {
		common.MemoryCacheEnabled = oldCache
		common.DebugEnabled = oldDebug
		service.GetTaskAdaptorFunc = oldFactory
		service.GetPinnedTaskAdaptorFunc = oldPinned
		common.LogWriterMu.Lock()
		gin.DefaultWriter = oldWriter
		gin.DefaultErrorWriter = oldError
		common.LogWriterMu.Unlock()
	})
	require.NoError(t, service.UpdateVideoTasks(context.Background(), "hailuo", map[int][]string{ch.Id: {task.GetUpstreamTaskID()}}, map[string]*model.Task{task.GetUpstreamTaskID(): &task}))
	assert.Contains(t, logs.String(), "stage=parse_failed")
	assert.Contains(t, logs.String(), "plugin=\"hailuo\"")
	assert.Contains(t, logs.String(), "version=\""+pin.Version+"\"")
	for _, marker := range []string{"REVIEW_FAKE_SECRET", "signed_url", "access=FAKE", "Authorization", "provider-private-id"} {
		assert.NotContains(t, logs.String(), marker)
	}
}

func TestTaskPluginSubmitPinnedPollingAndRefund(t *testing.T) {
	setupCanvasControllerDB(t)
	model.InitDBColumns()
	sqlDB, err := model.DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousLogDB := model.LOG_DB
	model.LOG_DB = model.DB
	previousCache, previousBatch, previousLog := common.MemoryCacheEnabled, common.BatchUpdateEnabled, common.LogConsumeEnabled
	previousFactory, previousPinned := service.GetTaskAdaptorFunc, service.GetPinnedTaskAdaptorFunc
	previousLimit := constant.TaskQueryLimit
	constant.TaskQueryLimit = 100
	t.Cleanup(func() { constant.TaskQueryLimit = previousLimit })
	common.MemoryCacheEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true
	previousPrices := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		model.LOG_DB = previousLogDB
		common.MemoryCacheEnabled = previousCache
		common.BatchUpdateEnabled = previousBatch
		common.LogConsumeEnabled = previousLog
		service.GetTaskAdaptorFunc = previousFactory
		service.GetPinnedTaskAdaptorFunc = previousPinned
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrices))
	})
	require.NoError(t, model.DB.AutoMigrate(&model.TaskPlugin{}, &model.Option{}, &model.Channel{}, &model.Token{}, &model.Log{}, &model.Group{}, &model.GroupAlias{}, &model.ChannelGroupBinding{}, &model.Ability{}, &model.UserSubscription{}, &model.SubscriptionPreConsumeRecord{}))
	require.NoError(t, service.InitTaskPlugins())
	row, err := model.GetTaskPluginVersion("sora", "")
	require.NoError(t, err)
	require.NoError(t, model.ActivateTaskPlugin("sora", row.Version))
	require.NoError(t, model.DB.Create(&model.Option{Key: "TaskPluginEnabled", Value: "true"}).Error)
	loaded, pin, err := service.ActiveTaskPlugin("sora")
	require.NoError(t, err)
	var submits, polls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer upstream-secret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			submits.Add(1)
			assert.Equal(t, "/v1/videos", r.URL.Path)
			var body map[string]any
			require.NoError(t, common.DecodeJson(r.Body, &body))
			assert.Equal(t, "sora-2", body["model"])
			assert.Equal(t, float64(8), body["seconds"])
			_, _ = io.WriteString(w, `{"id":"provider-original","status":"queued","secret":"hidden-provider-value"}`)
			return
		}
		polls.Add(1)
		assert.Equal(t, "/v1/videos/provider-original", r.URL.Path)
		_, _ = io.WriteString(w, `{"id":"provider-original","status":"failed","error":{"message":"hidden-provider-value"}}`)
	}))
	defer server.Close()
	service.InitHttpClient()
	require.NoError(t, model.DB.Create(&model.Group{Code: "default", Name: "Default", Status: model.GroupStatusActive, UserSelectable: true, Ratio: 1}).Error)
	user := model.User{Id: 7101, Username: "plugin-wallet", Quota: 1000000, Status: common.UserStatusEnabled, Group: "default", Setting: `{"billing_preference":"wallet_only"}`}
	require.NoError(t, model.DB.Create(&user).Error)
	token := model.Token{Id: 7101, UserId: user.Id, Key: "plugin-token", RemainQuota: 1000000, Status: common.TokenStatusEnabled}
	require.NoError(t, model.DB.Create(&token).Error)
	ch := model.Channel{Id: 7101, Type: constant.ChannelTypeTaskPlugin, Name: "plugin-channel", Key: "upstream-secret", Status: common.ChannelStatusEnabled, BaseURL: &server.URL, Setting: common.GetPointer(`{"task_plugin_key":"sora"}`)}
	require.NoError(t, model.DB.Create(&ch).Error)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"sora-2":0.01}`))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/task/plugins/sora", strings.NewReader(`{"model":"sora-2","prompt":"test","seconds":8}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("id", user.Id)
	c.Set("username", user.Username)
	c.Set("token_name", "plugin-token")
	c.Set("official_task_plugin", loaded)
	c.Set("task_plugin_snapshot", pin)
	c.Set("platform", "sora")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeTaskPlugin)
	common.SetContextKey(c, constant.ContextKeyChannelId, ch.Id)
	common.SetContextKey(c, constant.ContextKeyChannelKey, ch.Key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, ch.GetSetting())
	c.Set(common.RequestIdKey, "plugin-real-lifecycle")
	info := &relaycommon.RelayInfo{UserId: user.Id, TokenId: token.Id, TokenKey: token.Key, RequestId: "plugin-real-lifecycle", UserGroup: "default", UsingGroup: "default", OriginModelName: "sora-2", TaskRelayInfo: &relaycommon.TaskRelayInfo{}, UserSetting: kitdto.UserSetting{BillingPreference: "wallet_only"}}
	result, taskErr := relay.RelayTaskSubmit(c, info)
	require.Nil(t, taskErr)
	require.NotNil(t, result.PluginResponse)
	assert.Equal(t, 1, int(submits.Load()))
	assert.Equal(t, 5000, result.Quota, "8 秒用量不得被错误当作按次价格的倍率")
	require.NoError(t, persistOfficialPluginTask(c, info, result))
	require.NoError(t, service.SettleBilling(c, info, result.Quota))
	service.LogTaskConsumption(c, info, nil)
	var task model.Task
	require.NoError(t, model.DB.Where("task_id = ?", info.PublicTaskID).First(&task).Error)
	assert.Equal(t, pin.SourceHash, task.PrivateData.Execution.TaskPlugin.SourceHash)
	assert.NotContains(t, string(task.Data), "hidden-provider-value")
	assert.Contains(t, string(task.PrivateData.PluginData), "hidden-provider-value")
	var charged model.User
	require.NoError(t, model.DB.First(&charged, user.Id).Error)
	assert.Equal(t, int64(995000), charged.Quota)
	// 激活更高版本且关闭总开关，旧任务仍必须使用原查询路径和失败解释。
	newSource := strings.ReplaceAll(row.Source, `version: "`+row.Version+`"`, `version: "99.0.0"`)
	newSource = strings.ReplaceAll(newSource, `"/v1/videos/"`, `"/wrong-version/"`)
	hash := sha256.Sum256([]byte(newSource))
	newRow := model.TaskPlugin{Key: "sora", Version: "99.0.0", APIVersion: 1, Source: newSource, SourceHash: fmt.Sprintf("%x", hash)}
	require.NoError(t, model.DB.Create(&newRow).Error)
	require.NoError(t, model.ActivateTaskPlugin("sora", "99.0.0"))
	require.NoError(t, model.DB.Model(&model.Option{}).Where(&model.Option{Key: "TaskPluginEnabled"}).Update("value", "false").Error)
	service.GetTaskAdaptorFunc = func(p constant.TaskPlatform) service.TaskPollingAdaptor { return relay.GetTaskAdaptor(p) }
	service.GetPinnedTaskAdaptorFunc = func(task *model.Task) (service.TaskPollingAdaptor, error) { return relay.GetPinnedTaskAdaptor(task) }
	service.RunTaskPollingOnce(context.Background(), nil)
	service.RunTaskPollingOnce(context.Background(), nil)
	require.NoError(t, model.DB.First(&task, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), task.Status)
	assert.Zero(t, task.Quota)
	assert.Equal(t, 1, int(polls.Load()))
	require.NoError(t, model.DB.First(&charged, user.Id).Error)
	assert.Equal(t, int64(1000000), charged.Quota)
	require.NoError(t, model.DB.First(&token, token.Id).Error)
	assert.Equal(t, 1000000, token.RemainQuota)
	var refunds int64
	require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("type = ?", model.LogTypeRefund).Count(&refunds).Error)
	assert.Equal(t, int64(1), refunds)
	_, err = model.DeleteTaskPluginVersion("sora", row.Version)
	assert.Error(t, err)
	// 缓存不能绕过持久化源码完整性检查。
	require.NoError(t, model.DB.Model(&model.TaskPlugin{}).Where(&model.TaskPlugin{Key: "sora", Version: row.Version}).Update("source", row.Source+"\n// tampered").Error)
	_, err = relay.GetPinnedTaskAdaptor(&task)
	assert.Error(t, err)
}
