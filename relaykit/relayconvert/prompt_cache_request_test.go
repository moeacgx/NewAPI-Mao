package relayconvert

import (
	"context"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptCacheRequestConversion(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, fields := range []string{
			``,
			`,"prompt_cache_key":"session-世界","prompt_cache_retention":"24h","prompt_cache_options":{"enabled":false,"limit":0}`,
			`,"prompt_cache_options":{},"prompt_cache_retention":null`,
		} {
			t.Run(fmt.Sprintf("stream=%t/%s", stream, fields), func(t *testing.T) {
				raw := fmt.Sprintf(`{"model":"gpt-test","messages":[{"role":"user","content":"hello"}],"stream":%t%s}`, stream, fields)
				var request dto.GeneralOpenAIRequest
				require.NoError(t, kitutil.Unmarshal([]byte(raw), &request))
				var original map[string]any
				require.NoError(t, kitutil.Unmarshal([]byte(raw), &original))
				converted, err := ConvertRequest(context.Background(), nil, types.RelayFormatOpenAIResponses, &request)
				require.NoError(t, err)
				responses, ok := converted.Value.(*dto.OpenAIResponsesRequest)
				require.True(t, ok)
				back, err := ConvertRequest(context.Background(), nil, types.RelayFormatOpenAI, responses)
				require.NoError(t, err)
				for _, value := range []any{responses, back.Value} {
					encoded, err := kitutil.Marshal(value)
					require.NoError(t, err)
					var actual map[string]any
					require.NoError(t, kitutil.Unmarshal(encoded, &actual))
					assert.Equal(t, stream, actual["stream"])
					for _, field := range []string{"prompt_cache_key", "prompt_cache_options", "prompt_cache_retention"} {
						want, exists := original[field]
						if exists {
							assert.Contains(t, actual, field)
							assert.Equal(t, want, actual[field], field)
						} else {
							assert.NotContains(t, actual, field)
						}
					}
				}
			})
		}
	}
}
