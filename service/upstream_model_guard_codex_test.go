package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// VerifyUpstreamModelGuardCodexHTTPIntegration 供外部测试接入真实 HTTP 与协议解析器，避免测试包循环依赖。
func VerifyUpstreamModelGuardCodexHTTPIntegration(t *testing.T, stream bool,
	send func(*gin.Context, *http.Request, *relaycommon.RelayInfo) (*http.Response, error),
	parse func(*gin.Context, *relaycommon.RelayInfo, *http.Response) (*dto.Usage, *types.NewAPIError),
) {
	t.Helper()
	channel, info, _ := setupUpstreamModelGuardServiceTest(t)
	response := `{"id":"resp-guard","object":"response","model":"provider","status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
	contentType := "application/json"
	if stream {
		contentType = "text/event-stream"
		response = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp-guard\",\"model\":\"provider\"}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\ndata: [DONE]\n\n"
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("X-Codex-Safety-Buffering-Faster-Model", "unexpected-buffer")
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
	assert.False(t, enabled, "主程序接收头部后、解析正文前应已按现有规则禁用渠道")
	usage, apiErr := parse(ctx, info, resp)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 2, usage.TotalTokens)
	finish()
	assert.Equal(t, "provider", info.UpstreamResponseModelName)
	assert.Contains(t, recorder.Body.String(), `"model":"provider"`, "当前响应中的正文模型保持原样")
	records, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	assert.Equal(t, "unexpected-buffer", records[0].ActualUpstreamModel)
	assert.Equal(t, constant.UpstreamModelSourceCodexFasterModel, records[0].DetectionSource)
}

func TestUpstreamModelGuardCodexHeaderMismatchSurvivesMatchingBodyAndCountsOnce(t *testing.T) {
	channel, info, config := setupUpstreamModelGuardServiceTest(t)
	_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{
		ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(2),
	}, 42)
	require.NoError(t, err)
	for attempt := 1; attempt <= 2; attempt++ {
		info.RequestId = fmt.Sprintf("codex-%d", attempt)
		finish := BindUpstreamModelGuard(info)
		info.CaptureCodexResponseHeaders(&http.Response{StatusCode: http.StatusOK, Header: http.Header{
			"X-Codex-Safety-Buffering-Faster-Model": {"unexpected-buffer"},
		}})
		info.SetUpstreamResponseModelName("provider")
		info.SetUpstreamResponseModelName("other-wrong-body")
		info.SetUpstreamResponseModelName("provider")
		finish()
		records, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
		require.NoError(t, err)
		require.EqualValues(t, attempt, total)
		assert.Equal(t, attempt, records[0].ConsecutiveMismatches)
		assert.Equal(t, constant.UpstreamModelSourceCodexFasterModel, records[0].DetectionSource)
		assert.Equal(t, "unexpected-buffer", records[0].ActualUpstreamModel)
		assert.Contains(t, records[0].Reason, "Codex faster-model 响应头")
		assert.Equal(t, "provider", info.UpstreamResponseModelName)
		enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
		require.NoError(t, err)
		assert.Equal(t, attempt < 2, enabled)
	}
}

func TestUpstreamModelGuardCodexHeaderRespectsExistingPolicy(t *testing.T) {
	for _, name := range []string{"白名单", "停用模块", "停用配置", "停用规则", "其他分组", "其他请求模型", "允许的模型"} {
		t.Run(name, func(t *testing.T) {
			channel, info, config := setupUpstreamModelGuardServiceTest(t)
			update := UpstreamModelGuardConfigUpdate{ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules}
			faster := "unexpected-buffer"
			switch name {
			case "白名单":
				ids := []int{channel.Id}
				update.ExcludedChannelIDs = &ids
			case "停用模块":
				RegisterUpstreamModelGuardModuleEnabledCheck(func() bool { return false })
			case "停用配置":
				update.Enabled = false
			case "停用规则":
				update.Rules[0].Enabled = false
			case "其他分组":
				info.UsingGroup = "other"
			case "其他请求模型":
				info.OriginModelName = "other"
			case "允许的模型":
				faster = "provider-v2"
			}
			_, err := SaveUpstreamModelGuardConfig(t.Context(), update, 42)
			require.NoError(t, err)
			finish := BindUpstreamModelGuard(info)
			info.CaptureCodexResponseHeaders(&http.Response{StatusCode: http.StatusOK, Header: http.Header{
				"X-Codex-Safety-Buffering-Faster-Model": {faster},
			}})
			info.SetUpstreamResponseModelName("provider")
			finish()
			enabled, err := model.IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
			require.NoError(t, err)
			assert.True(t, enabled)
			_, total, err := model.ListUpstreamModelGuardRecords(t.Context(), 1, 50)
			require.NoError(t, err)
			assert.Zero(t, total)
		})
	}
}

func TestUpstreamModelGuardCodexMatchingHeaderNeedsBodyToReset(t *testing.T) {
	for _, hasBody := range []bool{false, true} {
		t.Run(fmt.Sprintf("正文匹配=%t", hasBody), func(t *testing.T) {
			channel, info, config := setupUpstreamModelGuardServiceTest(t)
			_, err := SaveUpstreamModelGuardConfig(t.Context(), UpstreamModelGuardConfigUpdate{
				ExpectedVersion: config.ConfigVersion, Enabled: true, Rules: config.Rules, FailureThreshold: common.GetPointer(3),
			}, 42)
			require.NoError(t, err)
			seedFinish := BindUpstreamModelGuard(info)
			info.SetUpstreamResponseModelName("wrong")
			seedFinish()
			next := &relaycommon.RelayInfo{ChannelMeta: info.ChannelMeta, UsingGroup: info.UsingGroup, OriginModelName: info.OriginModelName, RequestId: "matching-header"}
			finish := BindUpstreamModelGuard(next)
			next.CaptureCodexResponseHeaders(&http.Response{StatusCode: http.StatusOK, Header: http.Header{
				"X-Codex-Safety-Buffering-Faster-Model": {"provider-v2"},
			}})
			if hasBody {
				next.SetUpstreamResponseModelName("provider")
			}
			finish()
			var streaks []model.UpstreamModelGuardStreak
			require.NoError(t, model.DB.Where("channel_id = ?", channel.Id).Find(&streaks).Error)
			if hasBody {
				assert.Empty(t, streaks)
			} else {
				require.Len(t, streaks, 1)
				assert.Equal(t, 1, streaks[0].ConsecutiveMismatches)
			}
		})
	}
}
