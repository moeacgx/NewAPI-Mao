package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestClaudeThinkingLogMatchesOutboundRequest(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	settings := model_setting.GetGlobalSettings()
	oldPassThrough := settings.PassThroughRequestEnabled
	t.Cleanup(func() {
		gin.SetMode(oldMode)
		settings.PassThroughRequestEnabled = oldPassThrough
	})

	// 保留空白和扩展字段，验证日志采集不会重新序列化透传正文。
	budgetBody := "{\n  \"model\":\"claude-sonnet-4-6\", \"max_tokens\":8192,\n  \"messages\":[{\"role\":\"user\",\"content\":\"hello\"}],\n  \"thinking\":{\"type\":\"enabled\",\"budget_tokens\":4096}, \"provider_extension\":true\n}\n"
	adaptiveBody := `{"model":"claude-opus-4-6","max_tokens":8192,"messages":[{"role":"user","content":"hello"}],"thinking":{"type":"adaptive"},"output_config":{"effort":"max"}}`
	tests := []struct {
		name           string
		body           string
		format         types.RelayFormat
		channelType    int
		globalPass     bool
		channelPass    bool
		forceResponses bool
		override       string
		wantEffort     string
		wantType       string
		wantBudget     int
		wantFields     map[string]string
		absentFields   []string
	}{
		{
			name: "原生预算保留并记录", body: budgetBody,
			wantEffort: "thinking:4096", wantType: "enabled", wantBudget: 4096,
			wantFields: map[string]string{"thinking.type": `"enabled"`, "thinking.budget_tokens": "4096"},
		},
		{
			name: "自适应思考记录明确等级", body: adaptiveBody,
			wantEffort: "max", wantType: "adaptive",
			wantFields: map[string]string{"thinking.type": `"adaptive"`, "output_config.effort": `"max"`},
		},
		{
			name: "全局透传记录预算且正文逐字节不变", body: budgetBody, globalPass: true,
			wantEffort: "thinking:4096", wantType: "enabled", wantBudget: 4096,
		},
		{
			name: "渠道透传记录等级且正文逐字节不变", body: adaptiveBody, channelPass: true,
			wantEffort: "max", wantType: "adaptive",
		},
		{
			name: "参数覆盖后记录实际预算", body: budgetBody,
			override:   `{"operations":[{"path":"thinking.budget_tokens","mode":"set","value":2048}]}`,
			wantEffort: "thinking:2048", wantType: "enabled", wantBudget: 2048,
			wantFields: map[string]string{"thinking.budget_tokens": "2048"},
		},
		{
			name: "删除思考配置不记录旧预算", body: budgetBody,
			override:     `{"operations":[{"path":"thinking","mode":"delete"}]}`,
			absentFields: []string{"thinking"},
		},
		{
			name: "删除思考和等级不回退旧等级", body: adaptiveBody,
			override:     `{"operations":[{"path":"thinking","mode":"delete"},{"path":"output_config","mode":"delete"}]}`,
			absentFields: []string{"thinking", "output_config"},
		},
		{
			name: "OpenRouter转换预算仍保留日志", channelType: constant.ChannelTypeOpenRouter, format: types.RelayFormatOpenAI,
			body:       `{"model":"anthropic/claude-sonnet-4.6","max_tokens":8192,"messages":[{"role":"user","content":"hello"}],"thinking":{"type":"enabled","budget_tokens":4096}}`,
			wantEffort: "thinking:4096", wantType: "enabled", wantBudget: 4096,
			wantFields: map[string]string{"reasoning.enabled": "true", "reasoning.max_tokens": "4096"}, absentFields: []string{"thinking"},
		},
		{
			// 强制 Responses 在 Chat 中间请求应用覆盖，日志须记录最终发出的明确等级。
			name: "Claude强制Responses记录覆盖后的实际等级", body: adaptiveBody,
			channelType: constant.ChannelTypeOpenAI, forceResponses: true,
			override:   `{"operations":[{"path":"reasoning_effort","mode":"set","value":"high"}]}`,
			wantEffort: "high", wantFields: map[string]string{"reasoning.effort": `"high"`},
			absentFields: []string{"thinking", "output_config"},
		},
		{
			// 现有 Responses→Claude 将 high 映射为预算，max 不在该转换器支持范围内。
			name: "Responses转Claude记录实际思考预算", format: types.RelayFormatOpenAIResponses,
			body:       `{"model":"claude-opus-4-6","input":"hello","max_output_tokens":8192,"reasoning":{"effort":"high"}}`,
			wantEffort: "thinking:4096", wantType: "enabled", wantBudget: 4096,
			wantFields: map[string]string{"thinking.type": `"enabled"`, "thinking.budget_tokens": "4096"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings.PassThroughRequestEnabled = tt.globalPass
			capturedBody := make(chan []byte, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if !assert.NoError(t, err) {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				capturedBody <- body
				// 返回上游错误，在真实请求完成后终止，避免进入数据库扣费。
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"captured"}}`))
			}))
			t.Cleanup(server.Close)

			format := tt.format
			if format == "" {
				format = types.RelayFormatClaude
			}
			channelType := tt.channelType
			if channelType == 0 {
				channelType = constant.ChannelTypeAnthropic
			}
			path := "/v1/messages"
			var request dto.Request = &dto.ClaudeRequest{}
			handler := ClaudeHelper
			relayMode := relayconstant.RelayModeChatCompletions
			if format == types.RelayFormatOpenAI {
				path, request, handler = "/v1/chat/completions", &dto.GeneralOpenAIRequest{}, TextHelper
			} else if format == types.RelayFormatOpenAIResponses {
				path, request, handler = "/v1/responses", &dto.OpenAIResponsesRequest{}, ResponsesHelper
				relayMode = relayconstant.RelayModeResponses
			}
			require.NoError(t, common.Unmarshal([]byte(tt.body), request))
			model := gjson.Get(tt.body, "model").String()
			originalRequest, err := common.Marshal(request)
			require.NoError(t, err)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			t.Cleanup(func() { common.CleanupBodyStorage(ctx) })
			common.SetContextKey(ctx, constant.ContextKeyChannelType, channelType)
			common.SetContextKey(ctx, constant.ContextKeyChannelBaseUrl, server.URL)
			common.SetContextKey(ctx, constant.ContextKeyChannelKey, "test-key")
			common.SetContextKey(ctx, constant.ContextKeyOriginalModel, model)
			common.SetContextKey(ctx, constant.ContextKeyChannelSetting, dto.ChannelSettings{
				PassThroughBodyEnabled: tt.channelPass,
				ForceResponses:         tt.forceResponses,
			})
			if tt.override != "" {
				var override map[string]interface{}
				require.NoError(t, common.Unmarshal([]byte(tt.override), &override))
				common.SetContextKey(ctx, constant.ContextKeyChannelParamOverride, override)
			}
			info := &relaycommon.RelayInfo{
				OriginModelName: model, RelayFormat: format,
				RelayMode: relayMode, RequestURLPath: path, Request: request,
			}
			apiErr := handler(ctx, info)
			require.NotNil(t, apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.StatusCode, "%v", apiErr)
			var outbound []byte
			select {
			case outbound = <-capturedBody:
			default:
				t.Fatalf("请求未到达伪上游: %v", apiErr)
			}
			if tt.globalPass || tt.channelPass {
				assert.Equal(t, tt.body, string(outbound))
			}
			for path, want := range tt.wantFields {
				assert.JSONEq(t, want, gjson.GetBytes(outbound, path).Raw, "%s; 出站正文: %s", path, outbound)
			}
			for _, path := range tt.absentFields {
				assert.False(t, gjson.GetBytes(outbound, path).Exists(), path)
			}
			other := service.GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, 0, 1)
			for key, want := range map[string]string{"reasoning_effort": tt.wantEffort, "thinking_type": tt.wantType} {
				if want == "" {
					assert.NotContains(t, other, key)
				} else {
					assert.Equal(t, want, other[key], key)
				}
			}
			if tt.wantBudget == 0 {
				assert.NotContains(t, other, "thinking_budget_tokens")
			} else {
				assert.Equal(t, tt.wantBudget, other["thinking_budget_tokens"])
			}
			requestAfter, err := common.Marshal(request)
			require.NoError(t, err)
			assert.JSONEq(t, string(originalRequest), string(requestAfter), "日志采集及中继不得修改原始请求快照")
		})
	}
}
