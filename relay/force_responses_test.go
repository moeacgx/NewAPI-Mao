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
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForceResponsesRoutesConversationRequests(t *testing.T) {
	settings := model_setting.GetGlobalSettings()
	saved := *settings
	t.Cleanup(func() { *settings = saved })
	settings.ChatCompletionsToResponsesPolicy = model_setting.ChatCompletionsToResponsesPolicy{}

	tests := []struct {
		name    string
		path    string
		format  types.RelayFormat
		mode    int
		body    string
		request dto.Request
		helper  func(*gin.Context, *relaycommon.RelayInfo) *types.NewAPIError
	}{
		{"chat", "/v1/chat/completions", types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions,
			`{"model":"client-model","messages":[{"role":"user","content":"hello"}],"temperature":0,"service_tier":"priority"}`, &dto.GeneralOpenAIRequest{}, TextHelper},
		{"claude", "/v1/messages", types.RelayFormatClaude, relayconstant.RelayModeUnknown,
			`{"model":"client-model","messages":[{"role":"user","content":"hello"}],"max_tokens":32}`, &dto.ClaudeRequest{}, ClaudeHelper},
		{"gemini", "/v1beta/models/client-model:generateContent", types.RelayFormatGemini, relayconstant.RelayModeGemini,
			`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`, &dto.GeminiChatRequest{}, GeminiHelper},
		{"responses", "/v1/responses", types.RelayFormatOpenAIResponses, relayconstant.RelayModeResponses,
			`{"model":"client-model","input":"hello"}`, &dto.OpenAIResponsesRequest{}, ResponsesHelper},
		{"chat stream", "/v1/chat/completions", types.RelayFormatOpenAI, relayconstant.RelayModeChatCompletions,
			`{"model":"client-model","messages":[{"role":"user","content":"hello"}],"stream":true}`, &dto.GeneralOpenAIRequest{}, TextHelper},
		{"claude stream", "/v1/messages", types.RelayFormatClaude, relayconstant.RelayModeUnknown,
			`{"model":"client-model","messages":[{"role":"user","content":"hello"}],"max_tokens":32,"stream":true}`, &dto.ClaudeRequest{}, ClaudeHelper},
		{"gemini stream", "/v1beta/models/client-model:streamGenerateContent?alt=sse", types.RelayFormatGemini, relayconstant.RelayModeGemini,
			`{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}`, &dto.GeminiChatRequest{}, GeminiHelper},
	}
	for _, tt := range tests {
		for _, passthrough := range []bool{false, true} {
			t.Run(tt.name+map[bool]string{false: "", true: "/透传与反向转换同时开启"}[passthrough], func(t *testing.T) {
				settings.PassThroughRequestEnabled = passthrough
				var outbound map[string]any
				var path string
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					path = r.URL.Path
					body, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.NoError(t, common.Unmarshal(body, &outbound))
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest)
					_, _ = io.WriteString(w, `{"error":{"message":"upstream rejected","type":"invalid_request_error"}}`)
				}))
				t.Cleanup(server.Close)
				require.NoError(t, common.UnmarshalJsonStr(tt.body, tt.request))
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
				c.Request.Header.Set("Content-Type", "application/json")
				c.Set("model_mapping", `{"client-model":"upstream-model"}`)
				common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
				common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
				common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
				common.SetContextKey(c, constant.ContextKeyOriginalModel, "client-model")
				common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{ForceResponses: true, PassThroughBodyEnabled: passthrough})
				common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, dto.ChannelOtherSettings{ResponsesToChatEnabled: passthrough})
				info, err := relaycommon.GenRelayInfo(c, tt.format, tt.request, nil)
				require.NoError(t, err)
				assert.Equal(t, tt.mode, info.RelayMode)
				apiErr := tt.helper(c, info)
				require.NotNil(t, apiErr)
				assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
				assert.Equal(t, "/v1/responses", path)
				assert.Equal(t, "upstream-model", outbound["model"])
				assert.Contains(t, outbound, "input")
				if strings.Contains(tt.name, "stream") {
					assert.Equal(t, true, outbound["stream"])
				}
				assert.NotContains(t, outbound, "messages")
				assert.NotContains(t, outbound, "contents")
				assert.NotContains(t, outbound, "service_tier")
				if tt.name == "chat" {
					assert.Equal(t, float64(0), outbound["temperature"])
				}
				assert.Equal(t, tt.mode, info.RelayMode)
				assert.Equal(t, tt.path, info.RequestURLPath)
			})
		}
	}
}

func TestForceResponsesBridgePreservesClientFormatAndUsage(t *testing.T) {
	response := `{"id":"resp_test","model":"gpt-test","object":"response","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":2,"output_tokens":3,"total_tokens":5}}`
	events := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_test\",\"model\":\"gpt-test\"}}\n\n" +
		"data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\n"
	for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude, types.RelayFormatGemini} {
		for _, mode := range []string{"json", "stream", "buffered"} {
			t.Run(string(format)+"/"+mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/v1/responses", r.URL.Path)
					if mode == "json" {
						w.Header().Set("Content-Type", "application/json")
						_, _ = io.WriteString(w, response)
					} else {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = io.WriteString(w, events)
					}
				}))
				t.Cleanup(server.Close)
				recorder := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(recorder)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
				c.Request.Header.Set("Content-Type", "application/json")
				info := &relaycommon.RelayInfo{
					ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: server.URL, UpstreamModelName: "gpt-test", ChannelSetting: dto.ChannelSettings{ForceResponses: true}, ChannelOtherSettings: dto.ChannelOtherSettings{ResponsesToChatEnabled: true}},
					RelayMode:   relayconstant.RelayModeChatCompletions, RequestURLPath: "/v1/chat/completions", RelayFormat: format, IsStream: mode == "stream", ShouldIncludeUsage: true, DisablePing: true,
				}
				request := &dto.GeneralOpenAIRequest{Model: "gpt-test", Messages: []dto.Message{{Role: "user", Content: "hello"}}, Stream: common.GetPointer(mode == "stream")}
				adaptor := GetAdaptor(constant.APITypeOpenAI)
				adaptor.Init(info)
				info.FinalRequestRelayFormat = types.RelayFormatClaude
				usage, apiErr := chatCompletionsViaResponses(c, info, adaptor, request)
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				assert.Equal(t, types.RelayFormat(types.RelayFormatOpenAIResponses), info.GetFinalRequestRelayFormat())
				assert.Equal(t, 2, usage.PromptTokens)
				assert.Equal(t, 3, usage.CompletionTokens)
				assert.Equal(t, 5, usage.TotalTokens)
				assert.Contains(t, recorder.Body.String(), "hello")
				assert.Equal(t, mode == "stream", info.IsStream)
				if mode == "stream" {
					assert.Contains(t, recorder.Header().Get("Content-Type"), "text/event-stream")
					if format == types.RelayFormatClaude {
						assert.Contains(t, recorder.Body.String(), "message_stop")
					}
				} else {
					var body map[string]any
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &body))
					field := map[types.RelayFormat]string{types.RelayFormatOpenAI: "choices", types.RelayFormatClaude: "content", types.RelayFormatGemini: "candidates"}[format]
					assert.Contains(t, body, field)
				}
			})
		}
	}
}

func TestForceResponsesRestoresClientStreamAfterUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":{"message":"busy","type":"rate_limit_error"}}`)
	}))
	t.Cleanup(server.Close)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ChannelBaseUrl: server.URL, ChannelSetting: dto.ChannelSettings{ForceResponses: true}}, RelayMode: relayconstant.RelayModeChatCompletions, RequestURLPath: "/v1/chat/completions", RelayFormat: types.RelayFormatOpenAI}
	adaptor := GetAdaptor(constant.APITypeOpenAI)
	adaptor.Init(info)
	_, apiErr := chatCompletionsViaResponses(c, info, adaptor, &dto.GeneralOpenAIRequest{Model: "gpt-test", Messages: []dto.Message{{Role: "user", Content: "hello"}}})
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.False(t, info.IsStream)
	assert.Equal(t, relayconstant.RelayModeChatCompletions, info.RelayMode)
	assert.Equal(t, "/v1/chat/completions", info.RequestURLPath)
}
