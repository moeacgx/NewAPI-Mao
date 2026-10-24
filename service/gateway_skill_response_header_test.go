package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIOCopyBytesGracefullyKeepsLocalGatewaySkillsHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	context.Writer.Header().Set(GatewaySkillsResponseHeader, "local-metadata")
	upstream := &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type":              []string{"application/json"},
			GatewaySkillsResponseHeader: []string{"upstream-forgery"},
		},
	}

	IOCopyBytesGracefully(context, upstream, []byte(`{"choices":[]}`))

	require.Equal(t, "local-metadata", recorder.Header().Get(GatewaySkillsResponseHeader))
}
