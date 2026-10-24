package dto

import (
	"testing"

	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 工具续轮必须原样回送调用身份和思考签名，才能与模型此前生成的调用关联。
func TestGeminiRequestPreservesFunctionCallIdentityAndThoughtSignature(t *testing.T) {
	raw := []byte(`{"contents":[
		{"role":"model","parts":[{"functionCall":{"id":"call_read_1","name":"read_skill","args":{"path":"SKILL.md"}},"thoughtSignature":"c2lnbmF0dXJl"}]},
		{"role":"user","parts":[{"functionResponse":{"id":"call_read_1","name":"read_skill","response":{"content":"skill instructions"}}}]}
	]}`)
	var request GeminiChatRequest
	require.NoError(t, kitutil.Unmarshal(raw, &request))
	encoded, err := kitutil.Marshal(request)
	require.NoError(t, err)
	var restored struct {
		Contents []struct {
			Parts []struct {
				FunctionCall struct {
					ID string `json:"id"`
				} `json:"functionCall"`
				FunctionResponse struct {
					ID string `json:"id"`
				} `json:"functionResponse"`
				ThoughtSignature string `json:"thoughtSignature"`
			} `json:"parts"`
		} `json:"contents"`
	}
	require.NoError(t, kitutil.Unmarshal(encoded, &restored))
	require.Len(t, restored.Contents, 2)
	require.Len(t, restored.Contents[0].Parts, 1)
	require.Len(t, restored.Contents[1].Parts, 1)
	assert.Equal(t, "call_read_1", restored.Contents[0].Parts[0].FunctionCall.ID)
	assert.Equal(t, "c2lnbmF0dXJl", restored.Contents[0].Parts[0].ThoughtSignature)
	assert.Equal(t, "call_read_1", restored.Contents[1].Parts[0].FunctionResponse.ID)
}
