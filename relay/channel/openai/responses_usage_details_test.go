package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOaiResponsesPreservesDetailsForExpressionBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const response = `{"id":"resp_details","model":"actual-model","status":"completed","output":[],"usage":{"input_tokens":100,"output_tokens":20,"total_tokens":120,"input_tokens_details":{"cached_tokens":30,"cache_write_tokens":4,"image_tokens":10,"audio_tokens":5},"completion_tokens_details":{"audio_tokens":2,"image_tokens":3,"reasoning_tokens":8}}}`
	for _, stream := range []bool{false, true} {
		name := "HTTP"
		if stream {
			name = "SSE"
		}
		t.Run(name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(writer)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:     types.RelayFormatOpenAIResponses,
				OriginModelName: "requested-model",
				ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "mapped-model"},
			}
			body := response
			if stream {
				body = "data: {\"type\":\"response.completed\",\"response\":" + response + "}\n\ndata: [DONE]\n\n"
			}
			resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
			var usage *dto.Usage
			var apiErr *types.NewAPIError
			if stream {
				usage, apiErr = OaiResponsesStreamHandler(c, info, resp)
			} else {
				usage, apiErr = OaiResponsesHandler(c, info, resp)
			}
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Nil(t, usage.BillingUsage, "原生传输不应生成协议转换计费快照")
			assert.True(t, usage.HasInputTokens)
			assert.True(t, usage.PromptTokensDetails.HasCachedTokens)
			assert.Equal(t, 8, usage.CompletionTokenDetails.ReasoningTokens)
			assert.Equal(t, "actual-model", info.UpstreamResponseModelName)
			if stream {
				assert.Contains(t, writer.Body.String(), response)
			} else {
				assert.Equal(t, response, writer.Body.String())
			}
			params := service.BuildTieredTokenParams(usage, false, map[string]bool{
				"cr": true, "cc": true, "img": true, "ai": true, "ao": true, "img_o": true,
			})
			assert.Equal(t, billingexpr.TokenParams{P: 51, C: 15, Len: 100, CR: 30, CC: 4, Img: 10, AI: 5, AO: 2, ImgO: 3}, params)
			cost, _, err := billingexpr.RunExpr("p*2+cr*0.5+cc*3+img*4+ai*5+c*6+ao*7+img_o*8", params)
			require.NoError(t, err)
			assert.Equal(t, float64(322), cost)
		})
	}
}

func TestOaiResponsesNativeUsageIgnoresWireBillingMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name      string
		response  string
		wantError bool
	}{
		{name: "零快照不覆盖文本回退", response: `{"id":"resp_fallback","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}],"usage":{"input_tokens":0,"output_tokens":0,"usage_semantic":"anthropic","usage_source":"untrusted","cost":9,"billing_usage":{"semantic":"openai","openai_usage":{}}}}`},
		{name: "正快照不能伪造实际产出", response: `{"id":"resp_empty","output":[],"usage":{"input_tokens":0,"output_tokens":0,"billing_usage":{"semantic":"openai","openai_usage":{"prompt_tokens":100,"completion_tokens":10}}}}`, wantError: true},
	} {
		for _, stream := range []bool{false, true} {
			transport := "HTTP"
			if stream {
				transport = "SSE"
			}
			t.Run(tc.name+"/"+transport, func(t *testing.T) {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
				info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAIResponses, OriginModelName: "gpt-4o", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
				info.SetEstimatePromptTokens(25)
				body := tc.response
				if stream {
					body = "data: {\"type\":\"response.completed\",\"response\":" + body + "}\n\ndata: [DONE]\n\n"
				}
				resp := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
				var usage *dto.Usage
				var apiErr *types.NewAPIError
				if stream {
					usage, apiErr = OaiResponsesStreamHandler(c, info, resp)
					if apiErr == nil {
						// SSE 的零用量检查由外层结算执行，在此复用其纯分类入口。
						apiErr = service.TextUsageError(c, info, usage)
					}
				} else {
					usage, apiErr = OaiResponsesHandler(c, info, resp)
				}
				if tc.wantError {
					require.NotNil(t, apiErr)
					assert.Equal(t, types.ErrorCodeEmptyResponse, apiErr.GetErrorCode())
					assert.False(t, c.Writer.Written())
					return
				}
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				assert.Nil(t, usage.BillingUsage)
				assert.Empty(t, usage.UsageSemantic)
				assert.Empty(t, usage.UsageSource)
				assert.Nil(t, usage.Cost)
				assert.Equal(t, 25, usage.PromptTokens)
				assert.Positive(t, usage.CompletionTokens)
			})
		}
	}
}

func TestOaiResponsesSparseTerminalRetainsCacheDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name      string
		details   string
		wantCache int
		wantWrite int
		wantAudio int
	}{
		{name: "缺失不擦除", wantCache: 8, wantWrite: 3, wantAudio: 2},
		{name: "显式零覆盖", details: `,"input_tokens_details":{"cached_tokens":0,"cache_write_tokens":0}`, wantAudio: 2},
		{name: "详情仅补充图片", details: `,"input_tokens_details":{"image_tokens":1}`, wantCache: 8, wantWrite: 3, wantAudio: 2},
		{name: "输出显式零覆盖", details: `,"completion_tokens_details":{"audio_tokens":0}`, wantCache: 8, wantWrite: 3},
		{name: "顶层缓存创建零覆盖", details: `,"cache_write_tokens":0`, wantCache: 8, wantAudio: 2},
		{name: "顶层缓存创建正数覆盖", details: `,"cache_write_tokens":5`, wantCache: 8, wantWrite: 5, wantAudio: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAIResponses, OriginModelName: "gpt-4o", ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gpt-4o"}}
			body := "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_sparse\",\"usage\":{\"input_tokens\":20,\"output_tokens\":5,\"input_tokens_details\":{\"cached_tokens\":8,\"cache_write_tokens\":3},\"completion_tokens_details\":{\"audio_tokens\":2}}}}\n\n" +
				"data: {\"type\":\"response.done\",\"response\":{\"id\":\"resp_sparse\",\"usage\":{\"input_tokens\":20,\"output_tokens\":6" + tc.details + "}}}\n\ndata: [DONE]\n\n"
			usage, apiErr := OaiResponsesStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))})
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, 20, usage.PromptTokens)
			assert.Equal(t, 6, usage.CompletionTokens)
			assert.Equal(t, tc.wantCache, usage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, tc.wantWrite, usage.GetCacheCreationTokens())
			assert.Equal(t, tc.wantAudio, usage.CompletionTokenDetails.AudioTokens)
		})
	}
}
