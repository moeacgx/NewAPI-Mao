package xai

import (
	"context"
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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newXAIClaudeResponseTestContext() (*gin.Context, *httptest.ResponseRecorder, *relaycommon.RelayInfo) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat: types.RelayFormatClaude,
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, UpstreamModelName: "grok-4.6"},
		DisablePing: true,
	}
	info.SetEstimatePromptTokens(13)
	return c, recorder, info
}

func readXAIClaudeEvents(t *testing.T, body string) []dto.ClaudeResponse {
	t.Helper()
	var events []dto.ClaudeResponse
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event dto.ClaudeResponse
		require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &event))
		events = append(events, event)
	}
	return events
}

func TestXAIClaudeJSONResponsePreservesToolsAndUsage(t *testing.T) {
	c, recorder, info := newXAIClaudeResponseTestContext()
	body := `{"id":"chat-test","model":"grok-actual","choices":[{"index":0,"message":{"role":"assistant","content":"checking","tool_calls":[{"id":"call-1","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":100,"completion_tokens":1,"total_tokens":120,"prompt_cache_hit_tokens":80,"completion_tokens_details":{"reasoning_tokens":5}}}`
	usage, apiErr := xAIHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 20, usage.CompletionTokens)
	assert.Equal(t, 15, usage.CompletionTokenDetails.TextTokens)
	assert.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, "grok-actual", info.UpstreamResponseModelName)
	var response dto.ClaudeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "message", response.Type)
	assert.Equal(t, "tool_use", response.StopReason)
	require.Len(t, response.Content, 2)
	assert.Equal(t, "checking", response.Content[0].GetText())
	assert.Equal(t, "weather", response.Content[1].Name)
	assert.Equal(t, map[string]any{"city": "Paris"}, response.Content[1].Input)
	require.NotNil(t, response.Usage)
	assert.Equal(t, 20, response.Usage.OutputTokens)
	assert.Equal(t, 80, response.Usage.CacheReadInputTokens)
}

func TestXAIClaudeStreamWaitsForFinalUsage(t *testing.T) {
	c, recorder, info := newXAIClaudeResponseTestContext()
	chunks := []string{
		`{"id":"chat-test","model":"grok-actual","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"think"}}]}`,
		`{"id":"chat-test","model":"grok-actual","choices":[{"index":0,"delta":{"content":"answer"}}]}`,
		`{"id":"chat-test","model":"grok-actual","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"weather","arguments":"{\"city\":"}}]}}]}`,
		`{"id":"chat-test","model":"grok-actual","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Paris\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":100,"completion_tokens":1,"total_tokens":120}}`,
		`{"id":"chat-test","choices":[],"usage":{"prompt_tokens_details":{"cached_tokens":80}}}`,
		`[DONE]`,
	}
	body := "data: " + strings.Join(chunks, "\n\ndata: ") + "\n\n"
	usage, apiErr := xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 20, usage.CompletionTokens)
	assert.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, "grok-actual", info.UpstreamResponseModelName)
	assert.NotContains(t, recorder.Body.String(), "[DONE]")
	events := readXAIClaudeEvents(t, recorder.Body.String())
	require.NotEmpty(t, events)
	assert.Equal(t, "message_start", events[0].Type)
	var stopCount int
	var thinking, text, arguments strings.Builder
	var finalUsage *dto.ClaudeUsage
	for _, event := range events {
		if event.Type == "message_stop" {
			stopCount++
		}
		if event.Type == "message_delta" {
			finalUsage = event.Usage
			require.NotNil(t, event.Delta)
			require.NotNil(t, event.Delta.StopReason)
			assert.Equal(t, "tool_use", *event.Delta.StopReason)
		}
		if event.Delta == nil {
			continue
		}
		if event.Delta.Thinking != nil {
			thinking.WriteString(*event.Delta.Thinking)
		}
		if event.Delta.Text != nil {
			text.WriteString(*event.Delta.Text)
		}
		if event.Delta.PartialJson != nil {
			arguments.WriteString(*event.Delta.PartialJson)
		}
	}
	assert.Equal(t, 1, stopCount)
	assert.Equal(t, "think", thinking.String())
	assert.Equal(t, "answer", text.String())
	assert.JSONEq(t, `{"city":"Paris"}`, arguments.String())
	require.NotNil(t, finalUsage)
	assert.Equal(t, 80, finalUsage.CacheReadInputTokens)
	assert.Equal(t, 20, finalUsage.OutputTokens)
}

func TestXAIClaudeStreamWithoutUsage(t *testing.T) {
	for _, terminator := range []string{"\n\ndata: [DONE]\n\n", "\n\n"} {
		t.Run(terminator, func(t *testing.T) {
			c, recorder, info := newXAIClaudeResponseTestContext()
			body := `data: {"id":"chat-test","model":"grok-actual","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":"length"}]}` + terminator
			usage, apiErr := xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 13, usage.PromptTokens)
			assert.Positive(t, usage.CompletionTokens)
			events := readXAIClaudeEvents(t, recorder.Body.String())
			require.GreaterOrEqual(t, len(events), 2)
			final := events[len(events)-2]
			assert.Equal(t, "message_delta", final.Type)
			require.NotNil(t, final.Usage)
			assert.Equal(t, usage.CompletionTokens, final.Usage.OutputTokens)
			require.NotNil(t, final.Delta.StopReason)
			assert.Equal(t, "max_tokens", *final.Delta.StopReason)
			assert.Equal(t, "message_stop", events[len(events)-1].Type)
		})
	}
}

func TestXAIClaudeRejectsInvalidResponses(t *testing.T) {
	for _, body := range []string{`null`, `{`, `{}`, `{"error":{"message":"upstream unavailable","type":"server_error"}}`} {
		for _, stream := range []bool{false, true} {
			t.Run(body+string(rune('0'+map[bool]int{false: 0, true: 1}[stream])), func(t *testing.T) {
				c, recorder, info := newXAIClaudeResponseTestContext()
				payload := body
				if stream {
					payload = "data: " + payload + "\n\ndata: [DONE]\n\n"
				}
				resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload))}
				var apiErr *types.NewAPIError
				require.NotPanics(t, func() {
					if stream {
						_, apiErr = xAIStreamHandler(c, info, resp)
					} else {
						_, apiErr = xAIHandler(c, info, resp)
					}
				})
				assert.NotNil(t, apiErr)
				assert.NotContains(t, recorder.Body.String(), "message_stop")
				assert.NotContains(t, recorder.Body.String(), "[DONE]")
			})
		}
	}
}

func TestXAIClaudeCanceledStreamDoesNotFinish(t *testing.T) {
	c, recorder, info := newXAIClaudeResponseTestContext()
	ctx, cancel := context.WithCancel(c.Request.Context())
	cancel()
	c.Request = c.Request.WithContext(ctx)
	_, _ = xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))})
	assert.NotContains(t, recorder.Body.String(), "message_stop")
}
