package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	deepseekchannel "github.com/QuantumNous/new-api/relay/channel/deepseek"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDeepSeekStandaloneToolsUseRequestPriceAndRecordAccounting(t *testing.T) {
	service.InitHttpClient()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousBatch, previousLog, previousRedis := common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled
	previousPrice := ratio_setting.ModelPrice2JSONString()
	previousFreePreConsume := operation_setting.GetQuotaSetting().EnableFreeModelPreConsume
	operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = false
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "deepseek.db")), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Channel{}, &model.ChannelGroupBinding{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db
	common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled = false, true, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.BatchUpdateEnabled, common.LogConsumeEnabled, common.RedisEnabled = previousBatch, previousLog, previousRedis
		operation_setting.GetQuotaSetting().EnableFreeModelPreConsume = previousFreePreConsume
		require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(previousPrice))
		sqlDB, closeErr := db.DB()
		require.NoError(t, closeErr)
		require.NoError(t, sqlDB.Close())
	})

	for index, tc := range []struct {
		name, endpoint, response string
		price                    float64
		status                   int
		success                  bool
		readFailure              bool
	}{
		{"搜索自定义价", "search", `{"results":[{"title":"result"}]}`, 0.012, 200, true, false},
		{"抓取自定义价", "fetch", `{"content":"page"}`, 0.004, 200, true, false},
		{"搜索免费价", "search", `{"results":[{"title":"result"}]}`, 0, 200, true, false},
		{"抓取免费价", "fetch", `{"content":"page"}`, 0, 200, true, false},
		{"搜索空结果", "search", `{"results":[]}`, 0.012, 200, false, false},
		{"抓取无内容", "fetch", `{"content":""}`, 0.004, 200, false, false},
		{"搜索上游失败", "search", `{"error":{"message":"unavailable","type":"server_error"}}`, 0.012, 503, false, false},
		{"抓取上游失败", "fetch", `{"error":{"message":"missing","type":"not_found_error"}}`, 0.004, 404, false, false},
		{"搜索回包截断", "search", `{"results":[{"title":"result"}]}`, 0.012, 200, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			routingName := "deepseek-" + tc.endpoint
			prices, marshalErr := common.Marshal(map[string]float64{routingName: tc.price})
			require.NoError(t, marshalErr)
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(string(prices)))
			// 独立端点不得使用另一个价格体系的聊天工具附加费。
			toolKey := "deepseek_" + tc.endpoint
			operation_setting.SetToolPriceForTest(toolKey, 999)
			t.Cleanup(func() { operation_setting.DeleteToolPriceForTest(toolKey) })

			id := 98200 + index
			user := model.User{Id: id, Username: tc.name, AffCode: tc.name, Quota: 100000}
			channel := model.Channel{Id: id, Name: tc.name}
			require.NoError(t, db.Create(&user).Error)
			require.NoError(t, db.Create(&channel).Error)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if tc.readFailure {
					w.Header().Set("Content-Length", "1000")
				}
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.response)
			}))
			t.Cleanup(upstream.Close)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			path := "/v1/tools/" + tc.endpoint
			ctx.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"query":"test","url":"https://example.com"}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set(common.RequestIdKey, tc.name)
			common.SetContextKey(ctx, constant.ContextKeyOriginalModel, routingName)
			common.SetContextKey(ctx, constant.ContextKeyUsingGroup, "default")
			common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
			common.SetContextKey(ctx, constant.ContextKeyChannelType, constant.ChannelTypeDeepSeek)
			common.SetContextKey(ctx, constant.ContextKeyChannelId, id)
			common.SetContextKey(ctx, constant.ContextKeyUserId, id)
			common.SetContextKey(ctx, constant.ContextKeyChannelBaseUrl, upstream.URL)
			common.SetContextKey(ctx, constant.ContextKeyChannelKey, "sk-test")
			request, validateErr := helper.GetAndValidateRequest(ctx, types.RelayFormatDeepSeekTools)
			require.NoError(t, validateErr)
			info, infoErr := relaycommon.GenRelayInfo(ctx, types.RelayFormatDeepSeekTools, request, nil)
			require.NoError(t, infoErr)
			price, priceErr := helper.ModelPriceHelper(ctx, info, 0, request.GetTokenCountMeta())
			require.NoError(t, priceErr)
			expectedQuota := common.QuotaFromFloat(tc.price * common.QuotaPerUnit)
			assert.Equal(t, expectedQuota, price.QuotaToPreConsume)
			assert.Equal(t, tc.price == 0, price.FreeModel)
			settlement := &recordedSettlement{}
			info.Billing = settlement
			apiErr := DeepSeekToolsHelper(ctx, info)
			if tc.readFailure {
				require.NotNil(t, apiErr)
				assert.Equal(t, types.ErrorCodeReadResponseBodyFailed, apiErr.GetErrorCode())
			} else if tc.status >= 300 {
				require.NotNil(t, apiErr)
				assert.Equal(t, tc.status, apiErr.StatusCode)
			} else {
				require.Nil(t, apiErr)
			}

			var logs []model.Log
			require.NoError(t, db.Where("request_id = ?", tc.name).Find(&logs).Error)
			require.NoError(t, db.First(&user, id).Error)
			require.NoError(t, db.First(&channel, id).Error)
			if !tc.success {
				assert.Empty(t, settlement.settled)
				assert.Equal(t, tc.status == 200 && !tc.readFailure, settlement.refunded)
				assert.Empty(t, logs)
				assert.Zero(t, user.UsedQuota)
				assert.Zero(t, user.RequestCount)
				assert.Zero(t, channel.UsedQuota)
				return
			}
			assert.Equal(t, []int{expectedQuota}, settlement.settled)
			require.Len(t, logs, 1)
			log := logs[0]
			assert.Equal(t, model.LogTypeConsume, log.Type)
			assert.Equal(t, routingName, log.ModelName)
			assert.Equal(t, id, log.ChannelId)
			assert.Equal(t, expectedQuota, log.Quota)
			assert.Zero(t, log.PromptTokens)
			assert.Zero(t, log.CompletionTokens)
			assert.EqualValues(t, expectedQuota, user.UsedQuota)
			assert.Equal(t, 1, user.RequestCount)
			assert.EqualValues(t, expectedQuota, channel.UsedQuota)
			var metadata map[string]any
			require.NoError(t, common.UnmarshalJsonStr(log.Other, &metadata))
			assert.Equal(t, tc.price, metadata["model_price"])
			assert.Equal(t, path, metadata["request_path"])
			assert.NotContains(t, metadata, "tool_surcharges")
		})
	}
}

func TestDeepSeekHostedToolsCountOnce(t *testing.T) {
	previousBatch, previousLog := common.BatchUpdateEnabled, common.LogConsumeEnabled
	common.BatchUpdateEnabled, common.LogConsumeEnabled = true, false
	t.Cleanup(func() { common.BatchUpdateEnabled, common.LogConsumeEnabled = previousBatch, previousLog })
	for name, price := range map[string]float64{
		deepseekchannel.SearchToolPriceKey: 0.75, deepseekchannel.FetchToolPriceKey: 1.5,
		dto.BuildInToolWebSearch: 100, dto.BuildInToolWebSearchPreview: 100,
	} {
		operation_setting.SetToolPriceForTest(name, price)
		t.Cleanup(func() { operation_setting.DeleteToolPriceForTest(name) })
	}
	const claudeBody = `{"id":"message-1","type":"message","model":"deepseek-flash","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1,"server_tool_use":{"web_search_requests":3}}}`
	const output = `[{"id":"search-1","type":"web_search_call","status":"completed","action":{"type":"search"}},{"id":"fetch-1","type":"web_search_call","status":"completed","action":{"type":"open_page"}},{"id":"search-failed","type":"web_search_call","status":"failed","action":{"type":"search"}}]`
	const responseBody = `{"id":"response-1","object":"response","status":"completed","output":` + output + `,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
	const streamBody = "data: " + `{"type":"response.output_item.done","output_index":0,"item":{"id":"search-1","type":"web_search_call","status":"completed","action":{"type":"search"}}}` + "\n\n" +
		"data: " + `{"type":"response.output_item.done","output_index":1,"item":{"id":"fetch-1","type":"web_search_call","status":"completed","action":{"type":"open_page"}}}` + "\n\n" +
		"data: " + `{"type":"response.completed","response":` + responseBody + `}` + "\n\n"
	for _, tc := range []struct {
		name, body string
		format     types.RelayFormat
		mode       int
		stream     bool
		search     int
		fetch      int
	}{
		{"Claude 搜索只收费一次", claudeBody, types.RelayFormatClaude, relayconstant.RelayModeChatCompletions, false, 3, 0},
		{"Responses 供应商计数替换通用计数", responseBody, types.RelayFormatOpenAIResponses, relayconstant.RelayModeResponses, false, 1, 1},
		{"Responses 流事件与终态不重复", streamBody, types.RelayFormatOpenAIResponses, relayconstant.RelayModeResponses, true, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			info := &relaycommon.RelayInfo{
				OriginModelName: "deepseek-flash", RelayFormat: tc.format, RelayMode: tc.mode,
				StartTime: time.Now(), IsStream: tc.stream,
				ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeDeepSeek},
				PriceData:   hosttypes.PriceData{UsePrice: true, GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
			}
			adaptor := &deepseekchannel.Adaptor{}
			adaptor.Init(info)
			resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}
			if tc.stream {
				resp.Header.Set("Content-Type", "text/event-stream")
			}
			usage, apiErr := adaptor.DoResponse(ctx, resp, info)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Zero(t, ctx.GetInt("claude_web_search_requests"))
			require.NotNil(t, info.ResponsesUsageInfo)
			assert.Equal(t, tc.search, info.ResponsesUsageInfo.BuiltInTools[deepseekchannel.SearchToolPriceKey].CallCount)
			if tc.fetch > 0 {
				assert.Equal(t, tc.fetch, info.ResponsesUsageInfo.BuiltInTools[deepseekchannel.FetchToolPriceKey].CallCount)
			}
			assert.NotContains(t, info.ResponsesUsageInfo.BuiltInTools, dto.BuildInToolWebSearch)
			assert.NotContains(t, info.ResponsesUsageInfo.BuiltInTools, dto.BuildInToolWebSearchPreview)
			settlement := &recordedSettlement{}
			info.Billing = settlement
			require.Nil(t, service.PostTextConsumeQuota(ctx, info, usage.(*dto.Usage), nil))
			assert.Equal(t, []int{1125}, settlement.settled)
		})
	}
}
