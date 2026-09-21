package xai

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXAIHandlerMapsCacheUsageForBilling(t *testing.T) {
	tests := []struct {
		name         string
		usage        string
		cached       int
		cachePresent bool
	}{
		{
			name:         "standard cached tokens take precedence",
			usage:        `"prompt_tokens_details":{"cached_tokens":6},"prompt_cache_hit_tokens":9`,
			cached:       6,
			cachePresent: true,
		},
		{
			name:         "prompt cache hit alias is mapped",
			usage:        `"prompt_cache_hit_tokens":7`,
			cached:       7,
			cachePresent: true,
		},
		{
			name:         "missing cache fields remain compatible",
			usage:        ``,
			cached:       0,
			cachePresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			usageFields := ""
			if tt.usage != "" {
				usageFields = "," + tt.usage
			}
			body := []byte(`{"id":"chatcmpl-test","object":"chat.completion","model":"grok-test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12` + usageFields + `}}`)
			info := &relaycommon.RelayInfo{
				RelayFormat: types.RelayFormatOpenAI,
				ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, UpstreamModelName: "grok-test"},
			}

			usage, apiErr := xAIHandler(c, info, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			})

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.cached, usage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, tt.cachePresent, usage.PromptTokensDetails.HasCachedTokens)
		})
	}
}

func TestXAIStreamHandlerPreservesFinalUsageCacheFields(t *testing.T) {
	tests := []struct {
		name         string
		usage        string
		cached       int
		cachePresent bool
	}{
		{
			name:         "standard cached tokens take precedence",
			usage:        `"prompt_tokens_details":{"cached_tokens":4},"prompt_cache_hit_tokens":9`,
			cached:       4,
			cachePresent: true,
		},
		{
			name:         "prompt cache hit alias is mapped",
			usage:        `"prompt_cache_hit_tokens":5`,
			cached:       5,
			cachePresent: true,
		},
		{
			name:         "missing cache fields remain compatible",
			usage:        "",
			cached:       0,
			cachePresent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			info := &relaycommon.RelayInfo{
				RelayFormat:        types.RelayFormatOpenAI,
				ShouldIncludeUsage: true,
				ChannelMeta:        &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, UpstreamModelName: "grok-test"},
			}
			usageFields := ""
			if tt.usage != "" {
				usageFields = "," + tt.usage
			}
			body := "data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"grok-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n" +
				"data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"grok-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12" + usageFields + "}}\n\n" +
				"data: [DONE]\n\n"

			usage, apiErr := xAIStreamHandler(c, info, &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			})

			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tt.cached, usage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, tt.cachePresent, usage.PromptTokensDetails.HasCachedTokens)
			assert.Equal(t, 12, usage.TotalTokens)
			if tt.cachePresent {
				assert.Contains(t, recorder.Body.String(), `"usage"`)
			}
		})
	}
}

func TestXAIStreamHandlerMergesDetailOnlyUsageChunk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	info := &relaycommon.RelayInfo{
		RelayFormat:        types.RelayFormatOpenAI,
		ShouldIncludeUsage: true,
		ChannelMeta:        &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeXai, UpstreamModelName: "grok-test"},
	}
	body := "data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"model\":\"grok-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":20,\"total_tokens\":120}}\n\n" +
		"data: {\"id\":\"chatcmpl-test\",\"object\":\"chat.completion.chunk\",\"model\":\"grok-test\",\"choices\":[],\"usage\":{\"prompt_tokens_details\":{\"cached_tokens\":80}}}\n\n" +
		"data: [DONE]\n\n"

	usage, apiErr := xAIStreamHandler(c, info, &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	})

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 100, usage.PromptTokens)
	assert.Equal(t, 120, usage.TotalTokens)
	assert.Equal(t, 80, usage.PromptTokensDetails.CachedTokens)
}

func TestNormalizeXAIUsageBoundsCacheTokens(t *testing.T) {
	tests := []struct {
		name   string
		usage  string
		cached int
	}{
		{name: "negative cache", usage: `{"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":-1}}`, cached: 0},
		{name: "cache exceeds prompt", usage: `{"prompt_tokens":100,"prompt_tokens_details":{"cached_tokens":200}}`, cached: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usage dto.Usage
			require.NoError(t, common.UnmarshalJsonStr(tt.usage, &usage))
			normalizeXAIUsage(&usage)
			assert.Equal(t, tt.cached, usage.PromptTokensDetails.CachedTokens)
		})
	}
}

func TestXAIStreamCacheSourcesRemainConsistent(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
		cached int
	}{
		{"input details update", []string{`{"input_tokens_details":{"cached_tokens":128}}`, `{"input_tokens_details":{"cached_tokens":64000}}`}, 64000},
		{"input details lower update", []string{`{"input_tokens_details":{"cached_tokens":64000}}`, `{"input_tokens_details":{"cached_tokens":128}}`}, 128},
		{"standard survives later alias", []string{`{"prompt_tokens_details":{"cached_tokens":64000}}`, `{"prompt_cache_hit_tokens":128}`}, 64000},
		{"standard zero beats alias", []string{`{"prompt_tokens_details":{"cached_tokens":0}}`, `{"prompt_cache_hit_tokens":64000}`}, 0},
		{"input zero beats alias", []string{`{"input_tokens_details":{"cached_tokens":0}}`, `{"prompt_cache_hit_tokens":64000}`}, 0},
		{"input zero update", []string{`{"input_tokens_details":{"cached_tokens":64000}}`, `{"input_tokens_details":{"cached_tokens":0}}`}, 0},
		{"alias zero update", []string{`{"prompt_cache_hit_tokens":64000}`, `{"prompt_cache_hit_tokens":0}`}, 0},
		{"standard zero update", []string{`{"prompt_tokens_details":{"cached_tokens":64000}}`, `{"prompt_tokens_details":{"cached_tokens":0}}`}, 0},
		{"standard beats larger input", []string{`{"prompt_tokens_details":{"cached_tokens":128}}`, `{"input_tokens_details":{"cached_tokens":64000}}`}, 128},
		{"new standard beats old input", []string{`{"input_tokens_details":{"cached_tokens":128}}`, `{"prompt_tokens_details":{"cached_tokens":64000}}`}, 64000},
		{"clamp waits for final prompt", []string{`{"prompt_tokens":128,"prompt_tokens_details":{"cached_tokens":64000}}`}, 64000},
		{"clamp to final prompt", []string{`{"prompt_tokens":128,"prompt_tokens_details":{"cached_tokens":100000}}`}, 64213},
		{"input cache survives other detail", []string{`{"input_tokens_details":{"cached_tokens":64000}}`, `{"input_tokens_details":{"text_tokens":128}}`}, 64000},
		{"null leaves previous declaration", []string{`{"input_tokens_details":{"cached_tokens":64000}}`, `{"input_tokens_details":{"cached_tokens":null},"prompt_cache_hit_tokens":null}`}, 64000},
	}
	for _, tt := range tests {
		for _, format := range []types.RelayFormat{types.RelayFormatOpenAI, types.RelayFormatClaude} {
			t.Run(tt.name+"/"+string(format), func(t *testing.T) {
				c, recorder, info := newXAIClaudeResponseTestContext()
				info.RelayFormat = format
				var body strings.Builder
				body.WriteString("data: " + `{"id":"cache-test","model":"grok-4.6-build","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}` + "\n\n")
				for _, chunk := range append(append([]string{}, tt.chunks...), `{"prompt_tokens":64213,"completion_tokens":125,"total_tokens":64338}`) {
					body.WriteString(`data: {"id":"cache-test","choices":[],"usage":` + chunk + "}\n\n")
				}
				body.WriteString("data: [DONE]\n\n")
				usage, apiErr := xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String()))})
				require.Nil(t, apiErr)
				require.NotNil(t, usage)
				assert.Equal(t, 64213, usage.PromptTokens)
				assert.Equal(t, 125, usage.CompletionTokens)
				assert.Equal(t, tt.cached, usage.PromptTokensDetails.CachedTokens)
				if format == types.RelayFormatOpenAI {
					assert.Contains(t, recorder.Body.String(), "data: [DONE]")
					assert.NotContains(t, recorder.Body.String(), "message_delta")
					return
				}
				var finalUsage *dto.ClaudeUsage
				for _, event := range readXAIClaudeEvents(t, recorder.Body.String()) {
					if event.Type == "message_delta" {
						finalUsage = event.Usage
					}
				}
				require.NotNil(t, finalUsage)
				assert.Equal(t, tt.cached, finalUsage.CacheReadInputTokens)
				assert.Equal(t, 125, finalUsage.OutputTokens)
			})
		}
	}
}

func TestXAIClaudeJSONCacheMatchesBilling(t *testing.T) {
	for _, tt := range []struct {
		name   string
		detail string
		cached int
	}{
		{"standard priority", `"prompt_tokens_details":{"cached_tokens":64},"input_tokens_details":{"cached_tokens":12}`, 64},
		{"standard explicit zero", `"prompt_tokens_details":{"cached_tokens":0},"input_tokens_details":{"cached_tokens":64}`, 0},
		{"input cache clamped", `"input_tokens_details":{"cached_tokens":64000}`, 100},
		{"non-cache input details", `"input_tokens_details":{"text_tokens":100},"prompt_cache_hit_tokens":64`, 64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, recorder, info := newXAIClaudeResponseTestContext()
			body := `{"id":"cache-test","model":"grok-4.6-build","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":2,"total_tokens":102,` + tt.detail + `}}`
			usage, apiErr := xAIHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})
			require.Nil(t, apiErr)
			assert.Equal(t, tt.cached, usage.PromptTokensDetails.CachedTokens)
			var response dto.ClaudeResponse
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.NotNil(t, response.Usage)
			assert.Equal(t, tt.cached, response.Usage.CacheReadInputTokens)
		})
	}
}

func TestXAIStreamUsagePreservesPartialDetailsAndNativeChunks(t *testing.T) {
	c, recorder, info := newXAIClaudeResponseTestContext()
	info.RelayFormat = types.RelayFormatOpenAI
	chunks := []string{
		`{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12,"prompt_tokens_details":{"cached_tokens":8,"text_tokens":2,"audio_tokens":0,"image_tokens":1},"input_tokens_details":{"cached_tokens":6,"text_tokens":4},"completion_tokens_details":{"reasoning_tokens":5,"audio_tokens":2},"prompt_cache_hit_tokens":99}`,
		`{"prompt_tokens":64,"total_tokens":70,"prompt_tokens_details":{"text_tokens":0},"input_tokens_details":{"text_tokens":0},"completion_tokens_details":{"reasoning_tokens":0},"prompt_cache_hit_tokens":0}`,
		`{"prompt_tokens_details":{"audio_tokens":3}}`,
	}
	var body strings.Builder
	for _, chunk := range chunks {
		body.WriteString(`data: {"id":"partial-usage","choices":[],"usage":` + chunk + "}\n\n")
	}
	body.WriteString("data: [DONE]\n\n")
	usage, apiErr := xAIStreamHandler(c, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body.String()))})
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 64, usage.PromptTokens)
	assert.Equal(t, 6, usage.CompletionTokens)
	assert.Equal(t, 8, usage.PromptTokensDetails.CachedTokens)
	assert.Zero(t, usage.PromptTokensDetails.TextTokens)
	assert.Equal(t, 3, usage.PromptTokensDetails.AudioTokens)
	assert.Equal(t, 1, usage.PromptTokensDetails.ImageTokens)
	assert.Zero(t, usage.CompletionTokenDetails.ReasoningTokens)
	assert.Equal(t, 2, usage.CompletionTokenDetails.AudioTokens)
	require.NotNil(t, usage.InputTokensDetails)
	assert.Equal(t, 6, usage.InputTokensDetails.CachedTokens)
	assert.Zero(t, usage.InputTokensDetails.TextTokens)
	assert.True(t, usage.HasPromptCacheHitTokens)
	assert.Zero(t, usage.PromptCacheHitTokens)
	var wireUsage []dto.Usage
	for _, line := range strings.Split(recorder.Body.String(), "\n") {
		if !strings.HasPrefix(line, "data: {") {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.UnmarshalJsonStr(strings.TrimPrefix(line, "data: "), &chunk))
		require.NotNil(t, chunk.Usage)
		wireUsage = append(wireUsage, *chunk.Usage)
	}
	require.Len(t, wireUsage, 3)
	assert.Equal(t, 8, wireUsage[0].PromptTokensDetails.CachedTokens)
	assert.Equal(t, 99, wireUsage[0].PromptCacheHitTokens)
	assert.Zero(t, wireUsage[1].PromptTokensDetails.CachedTokens)
	assert.Zero(t, wireUsage[2].PromptTokensDetails.CachedTokens)
	assert.Equal(t, 1, strings.Count(recorder.Body.String(), "data: [DONE]"))
}
