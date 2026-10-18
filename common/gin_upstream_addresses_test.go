package common

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRequestScopedMaskingPreservesPublicUpstream(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://api.openai.com")
	message := `https://api.openai.com/v1/responses: unknown response.output_text.delta; see https://docs.example.org/errors?key=secret; dial tcp 10.0.0.1:443`
	assert.Equal(t, `https://api.openai.com/v1/responses: unknown response.output_text.delta; see https://docs.example.org/errors?key=***; dial tcp ***.***.***.***:443`, MaskSensitiveInfo(message, UpstreamAddresses(c)...))
	SetContextKey(c, constant.ContextKeyChannelBaseUrl, "https://private.example.com")
	assert.Equal(t, `https://***.com/***/***: see https://docs.example.org/errors`, MaskSensitiveInfo(`https://private.example.com/v1/responses: see https://docs.example.org/errors`, UpstreamAddresses(c)...))
}
