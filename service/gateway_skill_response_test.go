package service

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newGatewaySkillResponseContext(t *testing.T, usage *GatewaySkillResponseUsage) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set(gatewaySkillResponseUsageKey, usage)
	return c, recorder
}

func TestGatewaySkillResponseJSONFooterAndHeader(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 7, SkillID: "review", Version: 2, Title: "代码审查"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	require.NotNil(t, writer)
	body := `{"id":"chat-1","choices":[{"index":0,"message":{"role":"assistant","content":"答案"},"finish_reason":"stop"}],"usage":{"total_tokens":12},"unknown":{"keep":true}}`
	_, err := writer.Write([]byte(body))
	require.NoError(t, err)
	writer.Finish(true)

	assert.Same(t, writer.ResponseWriter, c.Writer)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, gjson.Get(recorder.Body.String(), "choices.0.message.content").String(), "答案\n\n---\n☁ 网关云端 Skills · 代码审查")
	assert.Contains(t, recorder.Body.String(), `"unknown":{"keep":true}`)
	assert.Equal(t, `{"total_tokens":12}`, gjson.Get(recorder.Body.String(), "usage").Raw)
	decoded, err := base64.RawURLEncoding.DecodeString(recorder.Header().Get(GatewaySkillsResponseHeader))
	require.NoError(t, err)
	assert.Equal(t, "代码审查", gjson.GetBytes(decoded, "skills.0.title").String())
}

func TestGatewaySkillResponseMetadataModeDoesNotChangeBody(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	c.Request.Header.Set(GatewaySkillsDisplayHeader, "metadata")
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	assert.Nil(t, writer)
	_, err := c.Writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"答案"},"finish_reason":"stop"}]}`))
	require.NoError(t, err)
	assert.NotContains(t, recorder.Body.String(), "网关云端 Skills")
	assert.NotEmpty(t, recorder.Header().Get(GatewaySkillsResponseHeader))
}

func TestGatewaySkillResponseSSEChatFooterPreservesUsageAndDone(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	require.NotNil(t, writer)
	chunks := []string{
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"分\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"段\"},\"finish_reason\":\"stop\"}],\"usage\":{\"total_tokens\":8}}\n\n",
		"data: [DONE]\n\n",
	}
	for _, chunk := range chunks {
		_, err := writer.Write([]byte(chunk))
		require.NoError(t, err)
	}
	writer.Finish(true)
	result := recorder.Body.String()
	assert.Contains(t, result, "分")
	assert.Contains(t, result, "网关云端 Skills · A")
	assert.Contains(t, result, "data: [DONE]")
	assert.Contains(t, result, `"total_tokens":8`)
}

func TestGatewaySkillResponseGeminiStreamUsesPreviousVisibleText(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatGemini)
	require.NotNil(t, writer)
	chunks := []string{
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"先回答\"}]}}]}\n\n",
		"data: {\"candidates\":[{\"finishReason\":\"STOP\"}]}\n\n",
	}
	for _, chunk := range chunks {
		_, err := writer.Write([]byte(chunk))
		require.NoError(t, err)
	}
	writer.Finish(true)
	result := recorder.Body.String()
	assert.Contains(t, result, "先回答")
	assert.Contains(t, result, "网关云端 Skills · A")
	assert.Contains(t, result, `"finishReason":"STOP"`)
}

func TestGatewaySkillResponseDoesNotMarkToolOnlyOrFailure(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "tool", body: `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"1"}]},"finish_reason":"tool_calls"}]}`},
		{name: "content filter", body: `{"choices":[{"message":{"role":"assistant","content":"截断"},"finish_reason":"content_filter"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, recorder := newGatewaySkillResponseContext(t, usage)
			writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
			require.NotNil(t, writer)
			_, err := writer.Write([]byte(test.body))
			require.NoError(t, err)
			writer.Finish(true)
			assert.NotContains(t, recorder.Body.String(), "网关云端 Skills")
		})
	}
}

func TestGatewaySkillResponseFailedJSONRoundDoesNotLeakIntoRetry(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	require.NotNil(t, writer)
	_, err := writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"失败轮次"},"finish_reason":"stop"}]}`))
	require.NoError(t, err)
	writer.Finish(false)
	assert.Empty(t, recorder.Body.String())

	// 重试使用原始 writer，失败轮次缓存的正文和尾标均不应残留。
	_, err = c.Writer.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"重试成功"},"finish_reason":"stop"}]}`))
	require.NoError(t, err)
	assert.Equal(t, `{"choices":[{"message":{"role":"assistant","content":"重试成功"},"finish_reason":"stop"}]}`, recorder.Body.String())
}

func TestGatewaySkillResponseClaudeToolUseStreamDoesNotAddFooter(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatClaude)
	require.NotNil(t, writer)
	chunks := []string{
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"先回答\"}}\n\n",
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"}}\n\n",
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
	}
	for _, chunk := range chunks {
		_, err := writer.Write([]byte(chunk))
		require.NoError(t, err)
	}
	writer.Finish(true)
	assert.Contains(t, recorder.Body.String(), "先回答")
	assert.NotContains(t, recorder.Body.String(), "网关云端 Skills")
}

func TestGatewaySkillResponseResponsesStreamKeepsSequenceOrder(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAIResponses)
	require.NotNil(t, writer)
	chunks := []string{
		"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"sequence_number\":0,\"output_index\":0,\"delta\":\"原文\"}\n\n",
		"event: response.completed\ndata: {\"type\":\"response.completed\",\"sequence_number\":1,\"response\":{\"id\":\"resp-1\",\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"role\":\"assistant\",\"content\":[{\"type\":\"output_text\",\"text\":\"原文\"}]}]}}\n\n",
	}
	for _, chunk := range chunks {
		_, err := writer.Write([]byte(chunk))
		require.NoError(t, err)
	}
	writer.Finish(true)

	var sequences []int64
	for _, frame := range strings.Split(recorder.Body.String(), "\n\n") {
		parsed := parseGatewaySSEFrame(frame + "\n\n")
		if parsed.data == "" {
			continue
		}
		sequence := gjson.Get(parsed.data, "sequence_number")
		if sequence.Exists() {
			sequences = append(sequences, sequence.Int())
		}
	}
	require.Len(t, sequences, 8)
	for index, sequence := range sequences {
		assert.Equal(t, int64(index), sequence)
	}
	assert.Contains(t, recorder.Body.String(), `"output_index":1`)
}

func TestGatewaySkillResponseHeaderTruncatesCompleteEntries(t *testing.T) {
	skills := make([]GatewaySkillResponseSkill, 0, 200)
	for i := 0; i < 200; i++ {
		skills = append(skills, GatewaySkillResponseSkill{TemplateID: i + 1, SkillID: strings.Repeat("skill", 12), Version: 1, Title: strings.Repeat("标题", 30)})
	}
	usage := &GatewaySkillResponseUsage{Version: 1, Total: len(skills), Skills: skills}
	c, _ := newGatewaySkillResponseContext(t, usage)
	BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	header := c.Writer.Header().Get(GatewaySkillsResponseHeader)
	assert.LessOrEqual(t, len(header), gatewaySkillResponseHeaderLimit)
	decoded, err := base64.RawURLEncoding.DecodeString(header)
	require.NoError(t, err)
	assert.Equal(t, len(skills), int(gjson.GetBytes(decoded, "total").Int()))
	assert.True(t, gjson.GetBytes(decoded, "truncated").Bool())
	assert.Greater(t, len(gjson.GetBytes(decoded, "skills").Array()), 0)
}

func TestGatewaySkillResponseResponsesAddsOutputTextPart(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	c.Request.URL.Path = "/v1/responses"
	writer := BeginGatewaySkillResponse(c, types.RelayFormatOpenAIResponses)
	require.NotNil(t, writer)
	body := `{"status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"完成"}]}]}`
	_, err := writer.Write([]byte(body))
	require.NoError(t, err)
	writer.Finish(true)
	result := recorder.Body.String()
	assert.Contains(t, result, "完成")
	assert.Contains(t, result, "网关云端 Skills · A")
	assert.Contains(t, result, `"type":"output_text"`)
}

func TestGatewaySkillResponseJSONFooterSupportsClaudeAndGemini(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	for _, test := range []struct {
		name   string
		format types.RelayFormat
		body   string
		path   string
	}{
		{
			name:   "claude",
			format: types.RelayFormatClaude,
			body:   `{"type":"message","role":"assistant","content":[{"type":"text","text":"完成"}],"stop_reason":"end_turn"}`,
			path:   "content.1.text",
		},
		{
			name:   "gemini",
			format: types.RelayFormatGemini,
			body:   `{"candidates":[{"content":{"parts":[{"text":"完成"}]},"finishReason":"STOP"}]}`,
			path:   "candidates.0.content.parts.1.text",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, recorder := newGatewaySkillResponseContext(t, usage)
			writer := BeginGatewaySkillResponse(c, test.format)
			require.NotNil(t, writer)
			_, err := writer.Write([]byte(test.body))
			require.NoError(t, err)
			writer.Finish(true)

			assert.Contains(t, gjson.Get(recorder.Body.String(), test.path).String(), "网关云端 Skills · A")
		})
	}
}

func TestGatewaySkillResponseClaudeJSONDoesNotMarkUnsupportedStopReason(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	c, recorder := newGatewaySkillResponseContext(t, usage)
	writer := BeginGatewaySkillResponse(c, types.RelayFormatClaude)
	require.NotNil(t, writer)
	body := `{"type":"message","role":"assistant","content":[{"type":"text","text":"暂存"}],"stop_reason":"pause_turn"}`
	_, err := writer.Write([]byte(body))
	require.NoError(t, err)
	writer.Finish(true)
	assert.NotContains(t, recorder.Body.String(), "网关云端 Skills")
}

func TestGatewaySkillResponseHeaderUsesCommonJSONAndResets(t *testing.T) {
	usage := &GatewaySkillResponseUsage{Version: 1, Total: 1, Skills: []GatewaySkillResponseSkill{{TemplateID: 1, SkillID: "a", Version: 1, Title: "A"}}}
	encoded, err := common.Marshal(usage)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"version":1`)
	c, _ := newGatewaySkillResponseContext(t, usage)
	BeginGatewaySkillResponse(c, types.RelayFormatOpenAI)
	assert.NotEmpty(t, c.Writer.Header().Get(GatewaySkillsResponseHeader))
	ResetGatewaySkillResponse(c)
	assert.Empty(t, c.Writer.Header().Get(GatewaySkillsResponseHeader))
	assert.Nil(t, GatewaySkillResponseUsageFromContext(c))
}
