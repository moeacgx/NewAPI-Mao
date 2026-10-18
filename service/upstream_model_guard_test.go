package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpstreamModelGuardServiceTest(t *testing.T) (*model.Channel, *relaycommon.RelayInfo, *UpstreamModelGuardConfig) {
	t.Helper()
	setupNotificationServiceTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.Channel{}, &model.Ability{}, &model.UpstreamModelGuardConfig{}, &model.UpstreamModelGuardRecord{}, &model.UpstreamModelGuardStreak{}))
	for _, group := range []model.Group{{Code: "default", Name: "默认", Status: model.GroupStatusActive}, {Code: "other", Name: "其他", Status: model.GroupStatusActive}} {
		require.NoError(t, model.DB.Create(&group).Error)
	}
	originalMemory := common.MemoryCacheEnabled
	originalAutoEnable := common.AutomaticEnableChannelEnabled
	upstreamModelGuardModuleState.RLock()
	originalCheck := upstreamModelGuardModuleState.enabled
	upstreamModelGuardModuleState.RUnlock()
	common.MemoryCacheEnabled = false
	common.AutomaticEnableChannelEnabled = true
	RegisterUpstreamModelGuardModuleEnabledCheck(func() bool { return true })
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemory
		common.AutomaticEnableChannelEnabled = originalAutoEnable
		RegisterUpstreamModelGuardModuleEnabledCheck(originalCheck)
	})
	channel := &model.Channel{Name: "模型校验渠道", Key: "secret", Status: common.ChannelStatusEnabled, Models: "client", Group: "default"}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
	config, err := GetUpstreamModelGuardConfig(context.Background())
	require.NoError(t, err)
	config, err = SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{
		ExpectedVersion: config.ConfigVersion, Enabled: true,
		FailureThreshold: common.GetPointer(1),
		Rules:            []UpstreamModelGuardRule{{Enabled: true, GroupCodes: []string{"default"}, Model: "client", UpstreamModels: []string{"provider", "provider-v2"}}},
	}, 42)
	require.NoError(t, err)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: channel.Id}, UsingGroup: "default", OriginModelName: "client", RequestId: "request-guard"}
	return channel, info, config
}

func TestUpstreamModelGuardMatchesOnlyEnabledExactRule(t *testing.T) {
	for _, test := range []struct {
		name, actual, group, requested                          string
		disableModule, disableRule, disableConfig, wantDisabled bool
	}{
		{name: "允许模型", actual: " provider-v2 "},
		{name: "空模型跳过", actual: "  "},
		{name: "其他分组", actual: "wrong", group: "other"},
		{name: "其他请求模型", actual: "wrong", requested: "Client"},
		{name: "停用模块", actual: "wrong", disableModule: true},
		{name: "停用规则", actual: "wrong", disableRule: true},
		{name: "停用配置", actual: "wrong", disableConfig: true},
		{name: "区分大小写", actual: "Provider", wantDisabled: true},
		{name: "非预期模型", actual: "wrong", wantDisabled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			channel, info, config := setupUpstreamModelGuardServiceTest(t)
			if test.group != "" {
				info.UsingGroup = test.group
			}
			if test.requested != "" {
				info.OriginModelName = test.requested
			}
			if test.disableModule {
				RegisterUpstreamModelGuardModuleEnabledCheck(func() bool { return false })
			}
			if test.disableRule || test.disableConfig {
				config.Rules[0].Enabled = !test.disableRule
				_, err := SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: !test.disableConfig, Rules: config.Rules}, 42)
				require.NoError(t, err)
			}
			BindUpstreamModelGuard(info)
			info.SetUpstreamResponseModelName(test.actual)
			var stored model.Channel
			require.NoError(t, model.DB.First(&stored, channel.Id).Error)
			assert.Equal(t, test.wantDisabled, stored.Status == common.ChannelStatusManuallyDisabled)
			assert.False(t, ShouldEnableChannel(nil, common.ChannelStatusManuallyDisabled))
		})
	}
}

func TestUpstreamModelGuardStreamModelChangeDisablesBeforeCallbackReturns(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("provider")
	info.SetUpstreamResponseModelName("wrong")
	info.SetUpstreamResponseModelName("another-wrong")
	available, err := model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.False(t, available)
	records, total, err := model.ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	assert.Equal(t, "wrong", records[0].ActualUpstreamModel)
}

func TestUpstreamModelGuardConfigRejectsInvalidRulesAndStaleWrites(t *testing.T) {
	_, _, config := setupUpstreamModelGuardServiceTest(t)
	valid := config.Rules[0]
	for _, test := range []struct {
		name  string
		rules []UpstreamModelGuardRule
	}{
		{name: "启用规则重叠", rules: []UpstreamModelGuardRule{valid, valid}},
		{name: "未知分组", rules: []UpstreamModelGuardRule{{Enabled: true, GroupCodes: []string{"unknown"}, Model: "client", UpstreamModels: []string{"provider"}}}},
		{name: "空模型", rules: []UpstreamModelGuardRule{{GroupCodes: []string{"default"}, Model: " ", UpstreamModels: []string{"provider"}}}},
		{name: "空允许列表", rules: []UpstreamModelGuardRule{{GroupCodes: []string{"default"}, Model: "client"}}},
		{name: "超长模型", rules: []UpstreamModelGuardRule{{GroupCodes: []string{"default"}, Model: strings.Repeat("a", 256), UpstreamModels: []string{"provider"}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: test.rules}, 42)
			require.Error(t, err)
			if test.name == "启用规则重叠" {
				assert.Contains(t, err.Error(), "分组 \"默认\"")
				assert.NotContains(t, err.Error(), "default")
			}
		})
	}
	valid.Model = " client "
	valid.GroupCodes = []string{" default ", "default"}
	valid.UpstreamModels = []string{" provider ", "provider", "Provider"}
	updated, err := SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: []UpstreamModelGuardRule{valid}}, 43)
	require.NoError(t, err)
	assert.Equal(t, []string{"default"}, updated.Rules[0].GroupCodes)
	assert.Equal(t, []string{"provider", "Provider"}, updated.Rules[0].UpstreamModels)
	assert.Equal(t, "client", updated.Rules[0].Model)
	assert.Equal(t, 43, updated.UpdatedBy)
	_, err = SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Rules: []UpstreamModelGuardRule{}}, 43)
	require.ErrorIs(t, err, model.ErrUpstreamModelGuardConfigConflict)
}

func TestUpstreamModelGuardRetainsClientModelAfterProviderRewrite(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	BindUpstreamModelGuard(info)
	// Gemini 等适配器会为了计费修改 OriginModelName，策略仍需匹配客户端请求模型。
	info.OriginModelName = "client-nothinking"
	info.SetUpstreamResponseModelName("wrong-provider")
	enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.False(t, enabled)
	records, _, err := model.ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "client", records[0].RequestedModel)
}

func TestUpstreamModelGuardUsesStableGroupIdentityAfterRename(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	require.NoError(t, model.DB.Model(&model.Group{}).Where("code = ?", "default").Update("code", "renamed").Error)
	config, err := GetUpstreamModelGuardConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"renamed"}, config.Rules[0].GroupCodes)
	info.UsingGroup = "renamed"
	BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong")
	available, err := model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.False(t, available)
}

func TestUpstreamModelGuardConfigDisableCancelsBoundDetection(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	BindUpstreamModelGuard(info)
	_, err := SaveUpstreamModelGuardConfig(context.Background(), UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: false, Rules: config.Rules}, 42)
	require.NoError(t, err)
	info.SetUpstreamResponseModelName("wrong")
	available, err := model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.True(t, available)
}

func TestUpstreamModelGuardDoesNotRetargetDeletedGroupCode(t *testing.T) {
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	require.NoError(t, model.DB.Where("code = ?", "default").Delete(&model.Group{}).Error)
	require.NoError(t, model.DB.Create(&model.Group{Code: "default", Name: "新分组", Status: model.GroupStatusActive}).Error)
	BindUpstreamModelGuard(info)
	info.SetUpstreamResponseModelName("wrong")
	available, err := model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.True(t, available)
}

type upstreamModelGuardStreamRecorder struct {
	*httptest.ResponseRecorder
	channelID            int
	observedMismatch     bool
	enabledWhenForwarded bool
	stateError           error
}

func (recorder *upstreamModelGuardStreamRecorder) Write(data []byte) (int, error) {
	if !recorder.observedMismatch && bytes.Contains(data, []byte("wrong-provider")) {
		recorder.observedMismatch = true
		recorder.enabledWhenForwarded, recorder.stateError = model.IsChannelEnabledAfterUpstreamModelGuard(context.Background(), recorder.channelID)
	}
	return recorder.ResponseRecorder.Write(data)
}

// 外部测试包传入真实解析器，避免 OpenAI 适配器与 service 测试包形成循环依赖。
func VerifyUpstreamModelGuardStreamIntegration(t *testing.T, parseStream func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError)) {
	t.Helper()
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	channel.Name = "<模型校验渠道>"
	require.NoError(t, model.DB.Model(channel).Update("name", channel.Name).Error)
	bot := &model.NotificationBot{Name: "existing-local-bot", Token: "guard-local-token", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(bot))
	task := &model.NotificationTask{
		Name: "上游模型校验通知", EventType: model.UpstreamModelGuardNotificationEvent, BotId: bot.Id,
		Template: "{{mention}} 渠道「{{channel_name}}」（#{{channel_id}}）已关闭\n{{comparison}}", Enabled: true,
	}
	require.NoError(t, model.CreateNotificationTask(task))
	require.NoError(t, model.CreateNotificationTarget(&model.NotificationTarget{
		TaskId: task.Id, ChatId: "-10001", MentionUserId: "42", MentionName: "<管理员>", Enabled: true,
	}))
	type telegramCall struct {
		method, path string
		message      telegramMessageRequest
		err          error
	}
	calls := make(chan telegramCall, 2)
	useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		call := telegramCall{method: request.Method, path: request.URL.Path}
		data, err := io.ReadAll(request.Body)
		call.err = err
		if err == nil {
			call.err = common.Unmarshal(data, &call.message)
		}
		calls <- call
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
	}))
	originalTimeout := constant.StreamingTimeout
	constant.StreamingTimeout = 30
	t.Cleanup(func() { constant.StreamingTimeout = originalTimeout })
	recorder := &upstreamModelGuardStreamRecorder{ResponseRecorder: httptest.NewRecorder(), channelID: channel.Id}
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info.UpstreamModelName = "provider"
	info.RelayFormat = types.RelayFormatOpenAI
	info.StartTime = time.Now()
	info.IsStream = true
	BindUpstreamModelGuard(info)
	stream := "data: {\"id\":\"chatcmpl-guard\",\"model\":\"provider\",\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n" +
		"data: {\"id\":\"chatcmpl-guard\",\"model\":\"wrong-provider\",\"choices\":[{\"delta\":{\"content\":\"second\"}}]}\n\n" +
		"data: {\"id\":\"chatcmpl-guard\",\"model\":\"wrong-provider\",\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n" +
		"data: [DONE]\n\n"
	usage, apiErr := parseStream(ctx, info, &http.Response{
		StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream)),
	})
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 5, usage.TotalTokens)
	assert.Equal(t, "wrong-provider", info.UpstreamResponseModelName)
	require.True(t, recorder.observedMismatch)
	require.NoError(t, recorder.stateError)
	assert.False(t, recorder.enabledWhenForwarded, "不匹配的数据转发到客户端前必须已持久化停用渠道")
	assert.Contains(t, recorder.Body.String(), "data: [DONE]", "当前在途响应应正常完成")
	records, total, err := model.ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, "wrong-provider", records[0].ActualUpstreamModel)
	assert.Equal(t, []string{"provider", "provider-v2"}, records[0].ExpectedUpstreamModels)
	select {
	case <-calls:
		t.Fatal("响应解析不能直接发送 Bot HTTP 请求")
	default:
	}
	dispatched, err := RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	assert.Equal(t, NotificationDispatchResult{Claimed: 1, Processed: 1}, dispatched)
	var call telegramCall
	select {
	case call = <-calls:
	default:
		t.Fatal("本地 Telegram 端点没有收到通知")
	}
	require.NoError(t, call.err)
	assert.Equal(t, http.MethodPost, call.method)
	assert.Equal(t, "/botguard-local-token/sendMessage", call.path)
	assert.Equal(t, telegramMessageRequest{
		ChatID: "-10001", ParseMode: "HTML", DisableWebPagePreview: true,
		Text: fmt.Sprintf(`<a href="tg://user?id=42">&lt;管理员&gt;</a> 渠道「&lt;模型校验渠道&gt;」（#%d）已关闭`+"\n分组: 默认\n请求模型: client\n允许上游模型: provider, provider-v2\n实际上游模型: wrong-provider\n连续不匹配: 1/1", channel.Id),
	}, call.message)
	var delivery model.NotificationDelivery
	require.NoError(t, model.DB.First(&delivery).Error)
	assert.Equal(t, model.NotificationDeliverySuccess, delivery.Status)
	assert.Equal(t, 1, delivery.AttemptCount)
	dispatched, err = RunNotificationDispatcherPass(context.Background())
	require.NoError(t, err)
	assert.Zero(t, dispatched.Processed, "重复的上游模型帧只能产生一条通知")
}
