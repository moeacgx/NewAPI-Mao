package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestTokenPromptBodyPreservesClientFieldsAcrossProtocols(t *testing.T) {
	tests := []struct {
		name     string
		format   types.RelayFormat
		body     string
		path     string
		expected string
	}{
		{"chat", types.RelayFormatOpenAI, `{"model":"gpt-5","temperature":0,"messages":[{"role":"system","content":"client"},{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}],"vendor":{"on":false}}`, "messages.0.role", "developer"},
		{"responses", types.RelayFormatOpenAIResponses, `{"model":"gpt-5","instructions":"client","input":[{"type":"reasoning","encrypted_content":"opaque"}],"temperature":0,"vendor":{"on":false}}`, "instructions", "first\n\nsecond\n\nclient"},
		{"claude_string", types.RelayFormatClaude, `{"model":"claude","system":"client","messages":[{"role":"user","content":"hello"}],"temperature":0,"vendor":{"on":false}}`, "system.1.text", "client"},
		{"claude_blocks", types.RelayFormatClaude, `{"model":"claude","system":[{"type":"text","text":"client","cache_control":{"type":"ephemeral"}}],"messages":[],"temperature":0,"vendor":{"on":false}}`, "system.1.cache_control.type", "ephemeral"},
		{"gemini_snake", types.RelayFormatGemini, `{"system_instruction":{"parts":[{"text":"client"}],"role":"system"},"contents":[{"parts":[{"text":"hello"}]}],"temperature":0,"vendor":{"on":false}}`, "systemInstruction.parts.1.text", "client"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := ApplyTokenPromptText([]byte(tt.body), tt.format, "gpt-5", []string{"first", "second"})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, gjson.GetBytes(out, tt.path).String())
			assert.Equal(t, "0", gjson.GetBytes(out, "temperature").Raw)
			assert.Equal(t, "false", gjson.GetBytes(out, "vendor.on").Raw)
			if tt.name == "chat" {
				assert.Equal(t, "client", gjson.GetBytes(out, "messages.1.content").String())
				assert.Equal(t, "https://example.com/a.png", gjson.GetBytes(out, "messages.2.content.1.image_url.url").String())
			}
			if tt.name == "responses" {
				assert.Equal(t, "opaque", gjson.GetBytes(out, "input.0.encrypted_content").String())
			}
			if tt.name == "gemini_snake" {
				assert.False(t, gjson.GetBytes(out, "system_instruction").Exists())
			}
		})
	}
}

func TestTokenPromptBodyWithoutMatchesRemainsIdentical(t *testing.T) {
	body := []byte(`{ "messages":[], "unknown": 0 }`)
	out, err := ApplyTokenPromptText(body, types.RelayFormatOpenAI, "gpt-4", nil)
	require.NoError(t, err)
	assert.Equal(t, body, out)
	out, err = ApplyTokenPromptText(body, types.RelayFormatEmbedding, "embed", []string{"rules"})
	require.NoError(t, err)
	assert.Equal(t, body, out)
}

func TestTokenPromptBodyRejectsInvalidInstructionShape(t *testing.T) {
	_, err := ApplyTokenPromptText([]byte(`{"instructions":{"secret":"invalid"}}`), types.RelayFormatOpenAIResponses, "gpt-5", []string{"rules"})
	require.Error(t, err)
}

func TestTokenPromptBodyNormalizesGeminiAliasWithoutLosingSystemFields(t *testing.T) {
	out, err := ApplyTokenPromptText([]byte(`{"systemInstruction":{"parts":[{"text":"unused"}]},"system_instruction":{"parts":[{"text":"effective"}],"role":"system","vendor":7}}`), types.RelayFormatGemini, "gemini", []string{"rules"})
	require.NoError(t, err)
	assert.False(t, gjson.GetBytes(out, "system_instruction").Exists())
	assert.Equal(t, "effective", gjson.GetBytes(out, "systemInstruction.parts.1.text").String())
	assert.Equal(t, "7", gjson.GetBytes(out, "systemInstruction.vendor").Raw)
}

func TestTokenPromptBodyDoesNotMutateSourceAcrossRetries(t *testing.T) {
	source := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"question"}]}`)
	first, err := ApplyTokenPromptText(source, types.RelayFormatOpenAI, "gpt-4", []string{"allowed rules"})
	require.NoError(t, err)
	second, err := ApplyTokenPromptText(source, types.RelayFormatOpenAI, "gpt-4", nil)
	require.NoError(t, err)
	assert.Equal(t, source, second)
	assert.Equal(t, "allowed rules", gjson.GetBytes(first, "messages.0.content").String())
	var parsed map[string]any
	require.NoError(t, common.Unmarshal(source, &parsed))
	assert.Len(t, parsed["messages"], 1)
}
