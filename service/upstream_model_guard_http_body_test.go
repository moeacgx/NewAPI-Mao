package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VerifyUpstreamModelGuardHTTPBodyIntegration 将真实 HTTP 和协议解析器接入守卫，避免测试包循环依赖。
func VerifyUpstreamModelGuardHTTPBodyIntegration(t *testing.T, stream bool,
	send func(*gin.Context, *http.Request, *relaycommon.RelayInfo) (*http.Response, error),
	parse func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError),
) {
	t.Helper()
	for _, test := range []struct {
		name, bodyModel, headerModel string
		wantDisabled                 bool
	}{
		{name: "正文匹配时忽略头部异常", bodyModel: "provider", headerModel: "unexpected-buffer"},
		{name: "正文异常时仍按原规则关闭", bodyModel: "wrong-provider", headerModel: "provider", wantDisabled: true},
		{name: "只有头部声明时跳过检测", headerModel: "unexpected-buffer"},
	} {
		t.Run(test.name, func(t *testing.T) {
			channel, info, _ := setupUpstreamModelGuardServiceTest(t)
			response := fmt.Sprintf(`{"id":"resp-guard","object":"response","model":%q,"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`, test.bodyModel)
			contentType := "application/json"
			if stream {
				contentType = "text/event-stream"
				response = "data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\ndata: [DONE]\n\n"
			}
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", contentType)
				w.Header().Set("X-Codex-Safety-Buffering-Faster-Model", test.headerModel)
				_, _ = io.WriteString(w, response)
			}))
			t.Cleanup(upstream.Close)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{}`))
			request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, upstream.URL, strings.NewReader(`{}`))
			require.NoError(t, err)
			info.IsStream = stream
			info.DisablePing = true
			info.RelayFormat = types.RelayFormatOpenAIResponses
			info.UpstreamModelName = "provider"
			finish := BindUpstreamModelGuard(info)
			resp, err := send(ctx, request, info)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })
			enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
			require.NoError(t, err)
			assert.True(t, enabled, "仅收到响应头不能关闭渠道")
			usage, apiErr := parse(ctx, info, resp)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 2, usage.TotalTokens)
			finish()
			assert.Equal(t, test.bodyModel, info.UpstreamResponseModelName)
			assert.Contains(t, recorder.Body.String(), `"text":"ok"`)
			enabled, err = model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
			require.NoError(t, err)
			assert.Equal(t, !test.wantDisabled, enabled)
			records, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
			require.NoError(t, err)
			if test.wantDisabled {
				require.EqualValues(t, 1, total)
				require.Len(t, records, 1)
				assert.Equal(t, test.bodyModel, records[0].ActualUpstreamModel)
			} else {
				assert.Zero(t, total)
			}
			other := GenerateTextOtherInfo(ctx, info, 1, 1, 1, 0, 0, 0, 1)
			adminInfo, ok := other["admin_info"].(map[string]interface{})
			require.True(t, ok)
			assert.NotContains(t, adminInfo, "codex_faster_model", "消费日志不再记录头部模型")
		})
	}
}
