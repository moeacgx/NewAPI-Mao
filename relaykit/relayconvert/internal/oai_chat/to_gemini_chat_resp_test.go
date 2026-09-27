package oaichat

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseOpenAI2GeminiMapsTextToolFinishReasonAndUsage(t *testing.T) {
	msg := dto.Message{
		Role:    "assistant",
		Content: "hello",
	}
	msg.SetToolCalls([]dto.ToolCallRequest{
		{
			ID:   "call_1",
			Type: "function",
			Function: dto.FunctionRequest{
				Name:      "lookup",
				Arguments: `{"q":"x"}`,
			},
		},
	})

	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Model: "gpt-test",
		Choices: []dto.OpenAITextResponseChoice{
			{
				Index:        2,
				Message:      msg,
				FinishReason: "length",
			},
		},
		Usage: dto.Usage{
			PromptTokens:     11,
			CompletionTokens: 5,
			TotalTokens:      16,
		},
	}, nil)

	assert.Equal(t, 11, resp.UsageMetadata.PromptTokenCount)
	assert.Equal(t, 5, resp.UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, 16, resp.UsageMetadata.TotalTokenCount)
	require.NotNil(t, resp.UsageMetadata.BillingUsage)
	require.NotNil(t, resp.UsageMetadata.BillingUsage.OpenAIUsage)
	assert.Equal(t, dto.BillingUsageSourceOAIChat, resp.UsageMetadata.BillingUsage.Source)
	assert.Equal(t, dto.BillingUsageSemanticOpenAI, resp.UsageMetadata.BillingUsage.Semantic)
	assert.Equal(t, 11, resp.UsageMetadata.BillingUsage.OpenAIUsage.PromptTokens)
	assert.Equal(t, 5, resp.UsageMetadata.BillingUsage.OpenAIUsage.CompletionTokens)
	assert.Equal(t, 16, resp.UsageMetadata.BillingUsage.OpenAIUsage.TotalTokens)
	assert.Nil(t, resp.UsageMetadata.BillingUsage.OpenAIUsage.BillingUsage)
	require.Len(t, resp.Candidates, 1)
	assert.Equal(t, int64(2), resp.Candidates[0].Index)
	require.NotNil(t, resp.Candidates[0].FinishReason)
	assert.Equal(t, "MAX_TOKENS", *resp.Candidates[0].FinishReason)
	require.Len(t, resp.Candidates[0].Content.Parts, 2)
	assert.Equal(t, "hello", resp.Candidates[0].Content.Parts[0].Text)
	require.NotNil(t, resp.Candidates[0].Content.Parts[1].FunctionCall)
	assert.Equal(t, "lookup", resp.Candidates[0].Content.Parts[1].FunctionCall.FunctionName)
	assert.Equal(t, map[string]interface{}{"q": "x"}, resp.Candidates[0].Content.Parts[1].FunctionCall.Arguments)
}

func TestResponseOpenAI2GeminiPreservesImageParts(t *testing.T) {
	msg := dto.Message{}
	msg.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "before"},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "data:image/png;base64,aW1hZ2U="}},
		{Type: dto.ContentTypeImageURL, ImageUrl: &dto.MessageImageUrl{Url: "https://cdn.invalid/image.png"}},
		{Type: dto.ContentTypeText, Text: "![remote](https://cdn.invalid/markdown.jpg) after"},
	})

	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{Message: msg}},
	}, nil)

	require.Len(t, resp.Candidates, 1)
	parts := resp.Candidates[0].Content.Parts
	require.Len(t, parts, 5)
	assert.Equal(t, "before", parts[0].Text)
	assert.Equal(t, "image/png", parts[1].InlineData.MimeType)
	assert.Equal(t, "aW1hZ2U=", parts[1].InlineData.Data)
	assert.Equal(t, "https://cdn.invalid/image.png", parts[2].FileData.FileUri)
	assert.Equal(t, "image/png", parts[2].FileData.MimeType)
	assert.Equal(t, "https://cdn.invalid/markdown.jpg", parts[3].FileData.FileUri)
	assert.Equal(t, " after", parts[4].Text)
}

func TestResponseOpenAI2GeminiPreservesMessageImages(t *testing.T) {
	msg := dto.Message{Images: []byte(`["data:image/jpeg;base64,/9j/",{"image_url":{"url":"https://cdn.invalid/out.webp"}}]`)}
	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{Message: msg}},
	}, nil)

	require.Len(t, resp.Candidates, 1)
	require.Len(t, resp.Candidates[0].Content.Parts, 2)
	assert.Equal(t, "image/jpeg", resp.Candidates[0].Content.Parts[0].InlineData.MimeType)
	assert.Equal(t, "https://cdn.invalid/out.webp", resp.Candidates[0].Content.Parts[1].FileData.FileUri)
	assert.Equal(t, "image/webp", resp.Candidates[0].Content.Parts[1].FileData.MimeType)
}

func TestResponseOpenAI2GeminiRejectsInvalidImageCarriers(t *testing.T) {
	msg := dto.Message{}
	msg.SetMediaContent([]dto.MediaContent{
		{Type: dto.ContentTypeText, Text: "![raw](data:image/png,not-base64)"},
		{Type: dto.ContentTypeText, Text: "![empty](data:image/png;base64,)"},
		{Type: dto.ContentTypeText, Text: "![bad-url](https:///missing-host.png)"},
		{Type: dto.ContentTypeText, Text: "ordinary https://example.com/page"},
	})

	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{Message: msg}},
	}, nil)

	require.Len(t, resp.Candidates, 1)
	require.Len(t, resp.Candidates[0].Content.Parts, 4)
	for _, part := range resp.Candidates[0].Content.Parts {
		assert.Nil(t, part.InlineData)
		assert.Nil(t, part.FileData)
	}
}

func TestResponseOpenAI2GeminiPreservesPlainDataImageContent(t *testing.T) {
	msg := dto.Message{Content: "data:image/png;base64,aW1hZ2U="}
	resp := ResponseOpenAI2Gemini(&dto.OpenAITextResponse{
		Choices: []dto.OpenAITextResponseChoice{{Message: msg}},
	}, nil)

	require.Len(t, resp.Candidates, 1)
	require.Len(t, resp.Candidates[0].Content.Parts, 1)
	assert.Equal(t, "image/png", resp.Candidates[0].Content.Parts[0].InlineData.MimeType)
	assert.Equal(t, "aW1hZ2U=", resp.Candidates[0].Content.Parts[0].InlineData.Data)
}

func TestStreamResponseOpenAI2GeminiKeepsMarkdownAsText(t *testing.T) {
	text := "![image](data:image/png;base64,aW1hZ2U=)"
	resp := StreamResponseOpenAI2Gemini(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{{
			Delta: dto.ChatCompletionsStreamResponseChoiceDelta{Content: &text},
		}},
	}, nil)

	require.Len(t, resp.Candidates, 1)
	require.Len(t, resp.Candidates[0].Content.Parts, 1)
	assert.Equal(t, text, resp.Candidates[0].Content.Parts[0].Text)
	assert.Nil(t, resp.Candidates[0].Content.Parts[0].InlineData)
}

func TestStreamResponseOpenAI2GeminiMapsToolCallFinishReasonAndUsage(t *testing.T) {
	resp := StreamResponseOpenAI2Gemini(&dto.ChatCompletionsStreamResponse{
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Index:        1,
				FinishReason: geminiRespPtr("tool_calls"),
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					ToolCalls: []dto.ToolCallResponse{
						{
							Type: "function",
							Function: dto.FunctionResponse{
								Name:      "lookup",
								Arguments: `{"q":"x"}`,
							},
						},
					},
				},
			},
		},
		Usage: &dto.Usage{
			PromptTokens:     13,
			CompletionTokens: 8,
			TotalTokens:      21,
		},
	}, &convmeta.Values{})

	require.NotNil(t, resp)
	assert.Equal(t, 13, resp.UsageMetadata.PromptTokenCount)
	assert.Equal(t, 8, resp.UsageMetadata.CandidatesTokenCount)
	assert.Equal(t, 21, resp.UsageMetadata.TotalTokenCount)
	require.NotNil(t, resp.UsageMetadata.BillingUsage)
	require.NotNil(t, resp.UsageMetadata.BillingUsage.OpenAIUsage)
	assert.Equal(t, 13, resp.UsageMetadata.BillingUsage.OpenAIUsage.PromptTokens)
	assert.Equal(t, 8, resp.UsageMetadata.BillingUsage.OpenAIUsage.CompletionTokens)
	require.Len(t, resp.Candidates, 1)
	assert.Equal(t, int64(1), resp.Candidates[0].Index)
	require.NotNil(t, resp.Candidates[0].FinishReason)
	assert.Equal(t, "STOP", *resp.Candidates[0].FinishReason)
	require.Len(t, resp.Candidates[0].Content.Parts, 1)
	require.NotNil(t, resp.Candidates[0].Content.Parts[0].FunctionCall)
	assert.Equal(t, "lookup", resp.Candidates[0].Content.Parts[0].FunctionCall.FunctionName)
	assert.Equal(t, map[string]interface{}{"q": "x"}, resp.Candidates[0].Content.Parts[0].FunctionCall.Arguments)
}

func geminiRespPtr[T any](value T) *T {
	return &value
}
