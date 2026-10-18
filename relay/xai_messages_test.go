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
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIClaudeHelperRoutesConvertedRequests(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	saved := *settings
	t.Cleanup(func() { *settings = saved })
	settings.PassThroughRequestEnabled = false
	settings.ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}

	for _, tt := range []struct {
		name, mappedModel, wantModel, effort string
		stream, search, override             bool
	}{
		{name: "普通请求", mappedModel: "grok-4.6", wantModel: "grok-4.6"},
		{name: "流式请求", mappedModel: "grok-4.6", wantModel: "grok-4.6", stream: true},
		{name: "搜索后缀和转换链", mappedModel: "grok-4.6-search", wantModel: "grok-4.6", search: true},
		{name: "推理后缀", mappedModel: "grok-3-mini-high", wantModel: "grok-3-mini", effort: "high"},
		{name: "参数覆盖", mappedModel: "grok-4.6", wantModel: "grok-4.6", override: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			type upstreamRequest struct {
				path, authorization string
				body                map[string]any
			}
			captured := make(chan upstreamRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				var payload map[string]any
				assert.NoError(t, common.Unmarshal(body, &payload))
				captured <- upstreamRequest{r.URL.Path, r.Header.Get("Authorization"), payload}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"message":"captured","type":"invalid_request_error"}}`)
			}))
			t.Cleanup(server.Close)
			body := `{"model":"client-grok","max_tokens":64,"temperature":0,"top_p":0,"system":"client system","messages":[{"role":"user","content":"hello"}],"tools":[{"name":"lookup","description":"find item","input_schema":{"type":"object","properties":{"id":{"type":"string"}}}}],"tool_choice":{"type":"tool","name":"lookup"}}`
			request := &dto.ClaudeRequest{}
			require.NoError(t, common.UnmarshalJsonStr(body, request))
			request.Stream = common.GetPointer(tt.stream)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set("model_mapping", `{"client-grok":"`+tt.mappedModel+`"}`)
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeXai)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "client-grok")
			common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{SystemPrompt: "channel system", SystemPromptOverride: true})
			if tt.override {
				common.SetContextKey(c, constant.ContextKeyChannelParamOverride, map[string]any{"temperature": 0.25})
			}
			info, err := relaycommon.GenRelayInfo(c, types.RelayFormatClaude, request, nil)
			require.NoError(t, err)
			apiErr := ClaudeHelper(c, info)
			require.NotNil(t, apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.StatusCode, apiErr.Error())
			out := <-captured
			assert.Equal(t, "/v1/chat/completions", out.path)
			assert.Equal(t, "Bearer test-key", out.authorization)
			assert.Equal(t, tt.wantModel, out.body["model"])
			assert.Equal(t, tt.wantModel, info.UpstreamModelName)
			wantTemperature := 0.0
			if tt.override {
				wantTemperature = 0.25
			}
			assert.Equal(t, wantTemperature, out.body["temperature"])
			assert.Equal(t, float64(0), out.body["top_p"])
			assert.NotContains(t, out.body, "system")
			messages, ok := out.body["messages"].([]any)
			require.True(t, ok)
			require.Len(t, messages, 2)
			assert.Equal(t, map[string]any{"role": "system", "content": "channel system\nclient system"}, messages[0])
			assert.Equal(t, map[string]any{"role": "user", "content": "hello"}, messages[1])
			tools, ok := out.body["tools"].([]any)
			require.True(t, ok)
			require.Len(t, tools, 1)
			tool := tools[0].(map[string]any)
			assert.Equal(t, "function", tool["type"])
			assert.Equal(t, "lookup", tool["function"].(map[string]any)["name"])
			assert.Equal(t, map[string]any{"type": "function", "function": map[string]any{"name": "lookup"}}, out.body["tool_choice"])
			if tt.stream {
				assert.Equal(t, true, out.body["stream"])
				assert.Equal(t, map[string]any{"include_usage": true}, out.body["stream_options"])
			} else {
				assert.NotContains(t, out.body, "stream_options")
			}
			if tt.search {
				assert.Equal(t, map[string]any{"mode": "on"}, out.body["search_parameters"])
			}
			if tt.effort != "" {
				assert.Equal(t, tt.effort, out.body["reasoning_effort"])
				assert.Equal(t, tt.effort, info.ReasoningEffort)
				assert.Equal(t, float64(64), out.body["max_completion_tokens"])
				assert.NotContains(t, out.body, "max_tokens")
			}
			assert.Equal(t, []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatOpenAI}, info.RequestConversionChain)
			assert.Equal(t, types.RelayFormat(types.RelayFormatOpenAI), info.GetFinalRequestRelayFormat())
			assert.Equal(t, "/v1/messages", info.RequestURLPath)
			assert.Equal(t, "client-grok", request.Model)
		})
	}
}

func TestXAIClaudeHelperPreservesProtocolPolicies(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	saved := *settings
	t.Cleanup(func() { *settings = saved })
	settings.ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}
	for _, tt := range []struct {
		name                           string
		globalPass, channelPass, force bool
		wantPath                       string
	}{
		{"渠道透传", false, true, false, "/v1/messages"},
		{"全局透传", true, false, false, "/v1/messages"},
		{"强制Responses", false, false, true, "/v1/responses"},
		{"强制Responses优先透传", true, true, true, "/v1/responses"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			settings.PassThroughRequestEnabled = tt.globalPass
			var path string
			var outbound map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				assert.NoError(t, common.DecodeJson(r.Body, &outbound))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"message":"captured","type":"invalid_request_error"}}`)
			}))
			t.Cleanup(server.Close)
			body := `{"model":"grok-4.6","max_tokens":64,"system":"system","messages":[{"role":"user","content":"hello"}]}`
			request := &dto.ClaudeRequest{}
			require.NoError(t, common.UnmarshalJsonStr(body, request))
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeXai)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
			common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
			common.SetContextKey(c, constant.ContextKeyOriginalModel, "grok-4.6")
			common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{ForceResponses: tt.force, PassThroughBodyEnabled: tt.channelPass})
			info, err := relaycommon.GenRelayInfo(c, types.RelayFormatClaude, request, nil)
			require.NoError(t, err)
			apiErr := ClaudeHelper(c, info)
			require.NotNil(t, apiErr)
			require.Equal(t, http.StatusBadRequest, apiErr.StatusCode, apiErr.Error())
			assert.Equal(t, tt.wantPath, path)
			if tt.force {
				assert.Contains(t, outbound, "input")
				assert.NotContains(t, outbound, "messages")
				assert.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
			} else {
				var original map[string]any
				require.NoError(t, common.UnmarshalJsonStr(body, &original))
				assert.Equal(t, original, outbound)
				assert.Equal(t, []types.RelayFormat{types.RelayFormatClaude}, info.RequestConversionChain)
			}
		})
	}
}
