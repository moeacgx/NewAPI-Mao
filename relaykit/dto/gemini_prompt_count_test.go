package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiTokenCountIncludesLoadedSystemInstructions(t *testing.T) {
	var request GeminiChatRequest
	require.NoError(t, kitutil.Unmarshal([]byte(`{"systemInstruction":{"parts":[{"text":"loaded rules"},{"text":"client rules"}]},"contents":[{"parts":[{"text":"question"}]}]}`), &request))
	assert.Equal(t, "loaded rules\nclient rules\nquestion", request.GetTokenCountMeta().CombineText)
}
