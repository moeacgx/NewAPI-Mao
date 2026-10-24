package service

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewaySkillDiscoveryOutputLimits(t *testing.T) {
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", "")
	for _, test := range []struct {
		name    string
		format  types.RelayFormat
		request string
		field   string
		limit   string
		removed string
	}{
		{"chat-default", types.RelayFormatOpenAI, `{"messages":[]}`, "max_tokens", "4096", "max_completion_tokens"},
		{"chat-existing", types.RelayFormatOpenAI, `{"messages":[],"max_tokens":32768}`, "max_tokens", "4096", "max_completion_tokens"},
		{"chat-small", types.RelayFormatOpenAI, `{"messages":[],"max_tokens":512}`, "max_tokens", "512", "max_completion_tokens"},
		{"chat-completion", types.RelayFormatOpenAI, `{"messages":[],"max_completion_tokens":32768}`, "max_completion_tokens", "4096", "max_tokens"},
		{"chat-completion-priority", types.RelayFormatOpenAI, `{"messages":[],"max_tokens":128,"max_completion_tokens":32768}`, "max_completion_tokens", "4096", "max_tokens"},
		{"chat-completion-zero", types.RelayFormatOpenAI, `{"messages":[],"max_tokens":32768,"max_completion_tokens":0}`, "max_completion_tokens", "0", "max_tokens"},
		{"responses-default", types.RelayFormatOpenAIResponses, `{"input":"task"}`, "max_output_tokens", "4096", ""},
		{"responses-small", types.RelayFormatOpenAIResponses, `{"input":"task","max_output_tokens":512}`, "max_output_tokens", "512", ""},
		{"responses-zero", types.RelayFormatOpenAIResponses, `{"input":"task","max_output_tokens":0}`, "max_output_tokens", "0", ""},
		{"claude-default", types.RelayFormatClaude, `{"messages":[]}`, "max_tokens", "4096", "max_tokens_to_sample"},
		{"claude-alias", types.RelayFormatClaude, `{"messages":[],"max_tokens_to_sample":32768}`, "max_tokens", "4096", "max_tokens_to_sample"},
		{"claude-canonical-priority", types.RelayFormatClaude, `{"messages":[],"max_tokens":2048,"max_tokens_to_sample":32768}`, "max_tokens", "2048", "max_tokens_to_sample"},
		{"claude-zero", types.RelayFormatClaude, `{"messages":[],"max_tokens":0}`, "max_tokens", "0", "max_tokens_to_sample"},
		{"gemini-default", types.RelayFormatGemini, `{"contents":[]}`, "generationConfig.maxOutputTokens", "4096", "generationConfig.max_output_tokens"},
		{"gemini-snake-priority", types.RelayFormatGemini, `{"contents":[],"generationConfig":{"maxOutputTokens":32768,"max_output_tokens":512}}`, "generationConfig.maxOutputTokens", "512", "generationConfig.max_output_tokens"},
		{"gemini-snake-zero", types.RelayFormatGemini, `{"contents":[],"generationConfig":{"maxOutputTokens":32768,"max_output_tokens":0}}`, "generationConfig.maxOutputTokens", "0", "generationConfig.max_output_tokens"},
		{"gemini-null-alias", types.RelayFormatGemini, `{"contents":[],"generationConfig":{"maxOutputTokens":32768,"max_output_tokens":null}}`, "generationConfig.maxOutputTokens", "4096", "generationConfig.max_output_tokens"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := []byte(test.request)
			request, err := NewGatewaySkillRequest(original, test.format, "test", "")
			require.NoError(t, err)
			private := request.Bytes()
			assert.Equal(t, test.limit, gjson.GetBytes(private, test.field).Raw)
			if test.removed != "" {
				assert.False(t, gjson.GetBytes(private, test.removed).Exists())
			}
			assert.Equal(t, test.request, string(original), "正式请求继续使用客户端原始上限")
		})
	}
}

func TestGatewaySkillDiscoveryConfiguredBudget(t *testing.T) {
	for _, config := range []struct {
		value string
		limit string
	}{
		{"1536", "1536"}, {"0", "4096"}, {"-1", "4096"}, {"invalid", "4096"}, {strconv.Itoa(dto.MaxTokensLimit), "4096"},
	} {
		t.Run(config.value, func(t *testing.T) {
			t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", config.value)
			for _, protocol := range []struct {
				format  types.RelayFormat
				request string
				field   string
			}{
				{types.RelayFormatOpenAI, `{"messages":[],"max_tokens":32768}`, "max_tokens"},
				{types.RelayFormatOpenAIResponses, `{"input":"task","max_output_tokens":32768}`, "max_output_tokens"},
				{types.RelayFormatClaude, `{"messages":[],"max_tokens":32768}`, "max_tokens"},
				{types.RelayFormatGemini, `{"contents":[],"generationConfig":{"maxOutputTokens":32768}}`, "generationConfig.maxOutputTokens"},
			} {
				request, err := NewGatewaySkillRequest([]byte(protocol.request), protocol.format, "test", "")
				require.NoError(t, err)
				assert.Equal(t, config.limit, gjson.GetBytes(request.Bytes(), protocol.field).Raw)
			}
		})
	}
}

func TestGatewaySkillDiscoveryBudgetDoesNotHideInvalidClientLimits(t *testing.T) {
	for _, test := range []struct {
		format types.RelayFormat
		body   string
	}{
		{types.RelayFormatOpenAI, `{"messages":[],"max_tokens":-1,"max_completion_tokens":4096}`},
		{types.RelayFormatOpenAI, `{"messages":[],"max_tokens":4096,"max_completion_tokens":18446744073709551615}`},
		{types.RelayFormatOpenAIResponses, `{"input":"task","max_output_tokens":-1}`},
		{types.RelayFormatClaude, `{"messages":[],"max_tokens":4096,"max_tokens_to_sample":-1}`},
		{types.RelayFormatGemini, `{"contents":[],"generationConfig":{"maxOutputTokens":18446744073709551615,"max_output_tokens":4096}}`},
		{types.RelayFormatGemini, `{"contents":[],"generationConfig":{"max_output_tokens":-1}}`},
	} {
		t.Run(string(test.format)+test.body, func(t *testing.T) {
			_, err := NewGatewaySkillRequest([]byte(test.body), test.format, "test", "")
			require.Error(t, err)
			var apiErr *types.NewAPIError
			require.True(t, errors.As(err, &apiErr))
			assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
			assert.Equal(t, types.ErrorCodeInvalidRequest, apiErr.GetErrorCode())
		})
	}
}

func TestGatewaySkillDiscoveryPreservesThinkingModes(t *testing.T) {
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", "4096")
	for _, test := range []struct {
		name     string
		thinking string
		limit    int
		expected string
	}{
		{"manual-capped", `{"type":"enabled","budget_tokens":16384,"display":"omitted","vendor":false}`, 32768, `{"type":"enabled","budget_tokens":3072,"display":"omitted","vendor":false}`},
		{"manual-small", `{"type":"enabled","budget_tokens":1024,"vendor":0}`, 32768, `{"type":"enabled","budget_tokens":1024,"vendor":0}`},
		{"manual-small-total", `{"type":"enabled","budget_tokens":1024}`, 1500, `{"type":"enabled","budget_tokens":1024}`},
		{"adaptive", `{"type":"adaptive","display":"summarized","vendor":false}`, 32768, `{"type":"adaptive","display":"summarized","vendor":false}`},
		{"disabled", `{"type":"disabled","vendor":0}`, 32768, `{"type":"disabled","vendor":0}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := fmt.Sprintf(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"history","signature":"must-remain-signed"}]}],"max_tokens":%d,"thinking":%s}`, test.limit, test.thinking)
			request, err := NewGatewaySkillRequest([]byte(original), types.RelayFormatClaude, "test", "")
			require.NoError(t, err)
			private := request.Bytes()
			assert.JSONEq(t, test.expected, gjson.GetBytes(private, "thinking").Raw)
			assert.Equal(t, "must-remain-signed", gjson.GetBytes(private, "messages.0.content.0.signature").String())
			assert.Equal(t, strconv.Itoa(min(test.limit, 4096)), gjson.GetBytes(private, "max_tokens").Raw)
		})
	}
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", "1024")
	_, err := NewGatewaySkillRequest([]byte(`{"messages":[],"max_tokens":32768,"thinking":{"type":"enabled","budget_tokens":16384}}`), types.RelayFormatClaude, "test", "")
	require.ErrorContains(t, err, "最小预算 1024")
	var apiErr *types.NewAPIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
}

func TestGatewaySkillDiscoveryGeminiAliasesCannotOverridePrivateConfiguration(t *testing.T) {
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", "4096")
	original := []byte(`{"contents":[{"role":"model","parts":[{"thoughtSignature":"signed-history"}]}],"generationConfig":{"candidateCount":3,"candidate_count":4,"maxOutputTokens":32768,"max_output_tokens":8192,"responseMimeType":"application/json","response_mime_type":"application/json","responseSchema":{"type":"object"},"response_schema":{"type":"string"},"responseJsonSchema":{"type":"object"},"response_json_schema":{"type":"string"},"thinkingConfig":{"thinkingBudget":1024,"vendor":"camel"},"thinking_config":{"thinkingBudget":8192,"thinking_budget":16384,"includeThoughts":true,"include_thoughts":false,"vendor":false},"temperature":0,"unknown":9007199254740993}}`)
	request, err := NewGatewaySkillRequest(original, types.RelayFormatGemini, "test", "")
	require.NoError(t, err)
	private := request.Bytes()
	assert.Equal(t, "1", gjson.GetBytes(private, "generationConfig.candidateCount").Raw)
	assert.Equal(t, "4096", gjson.GetBytes(private, "generationConfig.maxOutputTokens").Raw)
	assert.JSONEq(t, `{"thinkingBudget":3072,"includeThoughts":false,"vendor":false}`, gjson.GetBytes(private, "generationConfig.thinkingConfig").Raw)
	assert.Equal(t, "0", gjson.GetBytes(private, "generationConfig.temperature").Raw)
	assert.Equal(t, "9007199254740993", gjson.GetBytes(private, "generationConfig.unknown").Raw)
	assert.Equal(t, "signed-history", gjson.GetBytes(private, "contents.0.parts.0.thoughtSignature").String())
	for _, field := range []string{"candidate_count", "max_output_tokens", "responseMimeType", "response_mime_type", "responseSchema", "response_schema", "responseJsonSchema", "response_json_schema", "thinking_config"} {
		assert.False(t, gjson.GetBytes(private, "generationConfig."+field).Exists(), field)
	}
	assert.Equal(t, "16384", gjson.GetBytes(original, "generationConfig.thinking_config.thinking_budget").Raw)
	for _, thinking := range []struct {
		input    string
		expected string
	}{
		{`{"thinkingBudget":8192,"thinking_budget":0,"include_thoughts":false}`, `{"thinkingBudget":0,"includeThoughts":false}`},
		{`{"thinkingBudget":8192,"thinking_budget":-1}`, `{"thinkingBudget":-1}`},
		{`{"thinkingLevel":"high","thinking_level":"low","include_thoughts":false}`, `{"thinkingLevel":"low","includeThoughts":false}`},
		{`{"thinkingLevel":"high","thinking_level":""}`, `{"thinkingLevel":"high"}`},
	} {
		request, err := NewGatewaySkillRequest([]byte(fmt.Sprintf(`{"contents":[],"generationConfig":{"thinking_config":%s}}`, thinking.input)), types.RelayFormatGemini, "test", "")
		require.NoError(t, err)
		assert.JSONEq(t, thinking.expected, gjson.GetBytes(request.Bytes(), "generationConfig.thinkingConfig").Raw)
	}
}
