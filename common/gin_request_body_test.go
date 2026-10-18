package common

import (
	"bytes"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type requestBodyErrorReader struct{}

func (requestBodyErrorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestGetRequestBodyPreservesFirstReadError(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", requestBodyErrorReader{})

	_, firstErr := GetRequestBody(c)
	require.ErrorIs(t, firstErr, io.ErrUnexpectedEOF)

	_, secondErr := GetRequestBody(c)
	require.ErrorIs(t, secondErr, io.ErrUnexpectedEOF)
	require.Error(t, secondErr)
}

func TestGetRequestBodyMarksOversizedPayload(t *testing.T) {
	previous := constant.MaxRequestBodyMB
	constant.MaxRequestBodyMB = 1
	t.Cleanup(func() { constant.MaxRequestBodyMB = previous })

	payload := make([]byte, 1<<20+1)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewReader(payload))

	_, firstErr := GetRequestBody(c)
	require.Error(t, firstErr)
	require.True(t, IsRequestBodyTooLargeError(firstErr))
	_, secondErr := GetRequestBody(c)
	require.Error(t, secondErr)
	require.True(t, IsRequestBodyTooLargeError(secondErr))
}
