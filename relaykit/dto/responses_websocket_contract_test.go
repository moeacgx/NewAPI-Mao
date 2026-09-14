package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponsesWebSocketSettingDefaultsOff(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{name: "空配置", body: `{}`},
		{name: "旧配置", body: `{"proxy":"http://localhost:8080","http_protocol":"http1"}`},
		{name: "显式空值", body: `{"responses_websocket_enabled":null}`},
		{name: "显式关闭", body: `{"responses_websocket_enabled":false}`},
		{name: "显式启用", body: `{"responses_websocket_enabled":true}`, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var settings ChannelSettings
			require.NoError(t, kitutil.Unmarshal([]byte(tc.body), &settings))
			assert.Equal(t, tc.want, settings.ResponsesWebSocketEnabled)
			encoded, err := kitutil.Marshal(settings)
			require.NoError(t, err)
			var roundTrip ChannelSettings
			require.NoError(t, kitutil.Unmarshal(encoded, &roundTrip))
			assert.Equal(t, settings, roundTrip)
			var fields map[string]any
			require.NoError(t, kitutil.Unmarshal(encoded, &fields))
			if tc.want {
				assert.Equal(t, true, fields["responses_websocket_enabled"])
			} else {
				assert.NotContains(t, fields, "responses_websocket_enabled")
			}
		})
	}
}

func TestResponsesTerminalUsagePreservesOutputDetailsAndPresence(t *testing.T) {
	const body = `{"type":"response.completed","response":{"id":"resp_contract","usage":{"input_tokens":0,"output_tokens":7,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":5,"text_tokens":2}}}}`
	var event ResponsesStreamResponse
	require.NoError(t, kitutil.Unmarshal([]byte(body), &event))
	require.NotNil(t, event.Response)
	require.NotNil(t, event.Response.Usage)
	usage := event.Response.Usage
	assert.True(t, usage.HasInputTokens)
	assert.True(t, usage.HasOutputTokens)
	assert.False(t, usage.HasTotalTokens)
	assert.Zero(t, usage.InputTokens)
	assert.Equal(t, 7, usage.OutputTokens)
	require.NotNil(t, usage.InputTokensDetails)
	assert.True(t, usage.InputTokensDetails.HasCachedTokens)
	require.NotNil(t, usage.OutputTokensDetails)
	assert.Equal(t, 5, usage.OutputTokensDetails.ReasoningTokens)
	assert.Equal(t, 2, usage.OutputTokensDetails.TextTokens)
	encoded, err := kitutil.Marshal(event)
	require.NoError(t, err)
	var roundTrip ResponsesStreamResponse
	require.NoError(t, kitutil.Unmarshal(encoded, &roundTrip))
	require.NotNil(t, roundTrip.Response)
	require.NotNil(t, roundTrip.Response.Usage)
	assert.Equal(t, usage.OutputTokensDetails, roundTrip.Response.Usage.OutputTokensDetails)
}

func TestResponsesOutputDetailsDistinguishesAbsentFromExplicitZero(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		present bool
	}{
		{name: "缺失", body: `{}`},
		{name: "空值", body: `{"output_tokens_details":null}`},
		{name: "显式零", body: `{"output_tokens_details":{"reasoning_tokens":0}}`, present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var usage Usage
			require.NoError(t, kitutil.Unmarshal([]byte(tc.body), &usage))
			assert.Equal(t, tc.present, usage.OutputTokensDetails != nil)
			encoded, err := kitutil.Marshal(usage)
			require.NoError(t, err)
			var fields map[string]any
			require.NoError(t, kitutil.Unmarshal(encoded, &fields))
			if tc.present {
				require.NotNil(t, usage.OutputTokensDetails)
				assert.Zero(t, usage.OutputTokensDetails.ReasoningTokens)
				assert.Contains(t, fields, "output_tokens_details")
			} else {
				assert.NotContains(t, fields, "output_tokens_details")
			}
		})
	}
}
