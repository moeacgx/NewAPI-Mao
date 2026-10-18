package channel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoRequestCapturesOnlyUpstreamCodexModelEvidence(t *testing.T) {
	for _, test := range []struct {
		name, model     string
		status          int
		wantObservation bool
	}{
		{"上游成功声明", "gpt-5.6-luna", http.StatusOK, true},
		{"只有客户端伪造头", "", http.StatusOK, false},
		{"错误响应不推断模型", "gpt-5.6-luna", http.StatusServiceUnavailable, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			const body = `{"model":"gpt-5.6-sol","output":[]}`
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.model != "" {
					w.Header().Set("X-Codex-Safety-Buffering-Faster-Model", test.model)
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, body)
			}))
			t.Cleanup(upstream.Close)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))
			ctx.Request.Header.Set("X-Codex-Safety-Buffering-Faster-Model", "client-forged-model")
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, upstream.URL, strings.NewReader(`{}`))
			require.NoError(t, err)
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
			var observations []relaycommon.UpstreamModelEvidence
			info.OnUpstreamModelEvidence = func(_ *relaycommon.RelayInfo, evidence relaycommon.UpstreamModelEvidence) {
				observations = append(observations, evidence)
			}
			response, err := DoRequest(ctx, request, info)
			require.NoError(t, err)
			t.Cleanup(func() { _ = response.Body.Close() })
			if test.wantObservation {
				require.Equal(t, []relaycommon.UpstreamModelEvidence{{Model: test.model, Source: constant.UpstreamModelSourceCodexFasterModel}}, observations)
			} else {
				require.Empty(t, observations)
			}
			assert.Empty(t, info.UpstreamResponseModelName)
			data, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, body, string(data), "采集头部不能消费或改写响应正文")
		})
	}
}
