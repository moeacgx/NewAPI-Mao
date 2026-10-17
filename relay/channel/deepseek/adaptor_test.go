package deepseek

import (
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDeepSeekSuffixFields(t *testing.T) {
	cases := []struct{ model, base, thinking, effort string }{
		{"deepseek-flash-none", "deepseek-flash", "disabled", ""},
		{"deepseek-flash-low", "deepseek-flash", "enabled", "low"},
		{"deepseek-v4-pro-max", "deepseek-v4-pro", "enabled", "max"},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			info := &relaycommon.RelayInfo{OriginModelName: tc.model, ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: tc.model}}
			req := &dto.GeneralOpenAIRequest{Model: tc.model}
			got, err := (&Adaptor{}).ConvertOpenAIRequest(nil, info, req)
			require.NoError(t, err)
			out := got.(*dto.GeneralOpenAIRequest)
			require.Equal(t, tc.base, out.Model)
			require.JSONEq(t, `{"type":"`+tc.thinking+`"}`, string(out.THINKING))
			require.Equal(t, tc.effort, out.ReasoningEffort)
		})
	}
}

func TestDeepSeekToolEndpointURLs(t *testing.T) {
	a := &Adaptor{}
	for mode, path := range map[int]string{constant.RelayModeToolsSearch: "/v1/tools/search", constant.RelayModeToolsFetch: "/v1/tools/fetch"} {
		u, err := a.GetRequestURL(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.deepseek.com"}, RelayMode: mode, RelayFormat: types.RelayFormatDeepSeekTools})
		require.NoError(t, err)
		require.Equal(t, "https://api.deepseek.com"+path, u)
	}
}
