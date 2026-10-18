package oaichat

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"path"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
)

// ResponseOpenAI2Gemini 将 OpenAI 响应转换为 Gemini 格式
func ResponseOpenAI2Gemini(openAIResponse *dto.OpenAITextResponse, info convmeta.Meta) *dto.GeminiChatResponse {
	totalTokens := openAIResponse.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = openAIResponse.Usage.PromptTokens + openAIResponse.Usage.CompletionTokens
	}
	geminiResponse := &dto.GeminiChatResponse{
		Candidates:       make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		HasUsageMetadata: true,
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     openAIResponse.Usage.PromptTokens,
			CandidatesTokenCount: openAIResponse.Usage.CompletionTokens,
			TotalTokenCount:      totalTokens,
			BillingUsage:         openAIBillingUsageFromUsage(&openAIResponse.Usage),
		},
	}
	if metadata, ok := geminiBillingMetadataFromOpenAIUsage(&openAIResponse.Usage); ok {
		geminiResponse.UsageMetadata = metadata
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		// 设置结束原因
		var finishReason string
		switch choice.FinishReason {
		case "stop":
			finishReason = "STOP"
		case "length":
			finishReason = "MAX_TOKENS"
		case "content_filter":
			finishReason = "SAFETY"
		case "tool_calls":
			finishReason = "STOP"
		default:
			finishReason = "STOP"
		}
		candidate.FinishReason = &finishReason

		// 转换消息内容
		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		content.Parts = appendOpenAIMessageContent(content.Parts, &choice.Message)

		toolCalls := choice.Message.ParseToolCalls()
		for _, toolCall := range toolCalls {
			var args map[string]interface{}
			if toolCall.Function.Arguments != "" {
				if err := kitutil.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					args = map[string]interface{}{"arguments": toolCall.Function.Arguments}
				}
			} else {
				args = make(map[string]interface{})
			}

			part := dto.GeminiPart{
				FunctionCall: &dto.FunctionCall{
					FunctionName: toolCall.Function.Name,
					Arguments:    args,
				},
			}
			content.Parts = append(content.Parts, part)
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

// StreamResponseOpenAI2Gemini 将 OpenAI 流式响应转换为 Gemini 格式
func StreamResponseOpenAI2Gemini(openAIResponse *dto.ChatCompletionsStreamResponse, info convmeta.Meta) *dto.GeminiChatResponse {
	// 检查是否有实际内容或结束标志
	hasContent := false
	hasFinishReason := false
	for _, choice := range openAIResponse.Choices {
		if len(choice.Delta.GetContentString()) > 0 || (choice.Delta.ToolCalls != nil && len(choice.Delta.ToolCalls) > 0) {
			hasContent = true
		}
		if choice.FinishReason != nil {
			hasFinishReason = true
		}
	}

	// 如果没有实际内容且没有结束标志，跳过。主要针对 openai 流响应开头的空数据
	if !hasContent && !hasFinishReason {
		return nil
	}

	estimatePromptTokens := 0
	if info != nil {
		estimatePromptTokens = info.GetEstimatePromptTokens()
	}
	geminiResponse := &dto.GeminiChatResponse{
		Candidates:       make([]dto.GeminiChatCandidate, 0, len(openAIResponse.Choices)),
		HasUsageMetadata: true,
		UsageMetadata: dto.GeminiUsageMetadata{
			PromptTokenCount:     estimatePromptTokens,
			CandidatesTokenCount: 0, // 流式响应中可能没有完整的 usage 信息
			TotalTokenCount:      estimatePromptTokens,
		},
	}

	if openAIResponse.Usage != nil {
		geminiResponse.UsageMetadata.PromptTokenCount = openAIResponse.Usage.PromptTokens
		geminiResponse.UsageMetadata.CandidatesTokenCount = openAIResponse.Usage.CompletionTokens
		geminiResponse.UsageMetadata.TotalTokenCount = openAIResponse.Usage.TotalTokens
		geminiResponse.UsageMetadata.BillingUsage = openAIBillingUsageFromUsage(openAIResponse.Usage)
		if metadata, ok := geminiBillingMetadataFromOpenAIUsage(openAIResponse.Usage); ok {
			geminiResponse.UsageMetadata = metadata
		}
	}

	for _, choice := range openAIResponse.Choices {
		candidate := dto.GeminiChatCandidate{
			Index:         int64(choice.Index),
			SafetyRatings: []dto.GeminiChatSafetyRating{},
		}

		// 设置结束原因
		if choice.FinishReason != nil {
			var finishReason string
			switch *choice.FinishReason {
			case "stop":
				finishReason = "STOP"
			case "length":
				finishReason = "MAX_TOKENS"
			case "content_filter":
				finishReason = "SAFETY"
			case "tool_calls":
				finishReason = "STOP"
			default:
				finishReason = "STOP"
			}
			candidate.FinishReason = &finishReason
		}

		// 转换消息内容
		content := dto.GeminiChatContent{
			Role:  "model",
			Parts: make([]dto.GeminiPart, 0),
		}

		// 处理工具调用
		if choice.Delta.ToolCalls != nil {
			for _, toolCall := range choice.Delta.ToolCalls {
				// 解析参数
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					if err := kitutil.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						args = map[string]interface{}{"arguments": toolCall.Function.Arguments}
					}
				} else {
					args = make(map[string]interface{})
				}

				part := dto.GeminiPart{
					FunctionCall: &dto.FunctionCall{
						FunctionName: toolCall.Function.Name,
						Arguments:    args,
					},
				}
				content.Parts = append(content.Parts, part)
			}
		} else {
			// 处理文本内容
			textContent := choice.Delta.GetContentString()
			if textContent != "" {
				content.Parts = append(content.Parts, dto.GeminiPart{Text: textContent})
			}
		}

		candidate.Content = content
		geminiResponse.Candidates = append(geminiResponse.Candidates, candidate)
	}

	return geminiResponse
}

func appendOpenAIMessageContent(parts []dto.GeminiPart, message *dto.Message) []dto.GeminiPart {
	for _, media := range message.ParseContent() {
		if media.Type == dto.ContentTypeText {
			parts = appendOpenAITextContent(parts, media.Text)
			continue
		}
		if media.Type == dto.ContentTypeImageURL {
			if image := media.GetImageMedia(); image != nil {
				if part, ok := geminiImagePart(image.Url, image.MimeType); ok {
					parts = append(parts, part)
				}
			}
		}
	}
	return appendOpenAIImageValues(parts, message.Images)
}

func appendOpenAITextContent(parts []dto.GeminiPart, text string) []dto.GeminiPart {
	if text == "" {
		return parts
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == text && strings.HasPrefix(strings.ToLower(trimmed), "data:image/") {
		if part, ok := geminiImagePart(trimmed, ""); ok {
			return append(parts, part)
		}
	}
	for len(text) > 0 {
		start := strings.Index(text, "![")
		if start < 0 {
			return append(parts, dto.GeminiPart{Text: text})
		}
		if start > 0 {
			parts = append(parts, dto.GeminiPart{Text: text[:start]})
		}
		closeAlt := strings.Index(text[start+2:], "](")
		if closeAlt < 0 {
			return append(parts, dto.GeminiPart{Text: text[start:]})
		}
		closeAlt += start + 2
		closeURL := strings.IndexByte(text[closeAlt+2:], ')')
		if closeURL < 0 {
			return append(parts, dto.GeminiPart{Text: text[start:]})
		}
		closeURL += closeAlt + 2
		imageURL := strings.TrimSpace(text[closeAlt+2 : closeURL])
		part, ok := geminiImagePart(imageURL, "")
		if !ok {
			parts = append(parts, dto.GeminiPart{Text: text[start : closeURL+1]})
		} else {
			parts = append(parts, part)
		}
		text = text[closeURL+1:]
	}
	return parts
}

func appendOpenAIImageValues(parts []dto.GeminiPart, raw json.RawMessage) []dto.GeminiPart {
	if len(raw) == 0 {
		return parts
	}
	var values []json.RawMessage
	if kitutil.Unmarshal(raw, &values) != nil {
		values = []json.RawMessage{raw}
	}
	for _, value := range values {
		var text string
		if kitutil.Unmarshal(value, &text) == nil {
			if part, ok := geminiImagePart(text, ""); ok {
				parts = append(parts, part)
			}
			continue
		}
		var item map[string]any
		if kitutil.Unmarshal(value, &item) != nil {
			continue
		}
		mimeType := stringValue(item, "mime_type", "mimeType")
		if encoded := stringValue(item, "b64_json"); encoded != "" {
			if part, ok := geminiInlineImagePart(defaultImageMime(mimeType), encoded); ok {
				parts = append(parts, part)
			}
			continue
		}
		for _, key := range []string{"image_url", "url", "data"} {
			value := item[key]
			if nested, ok := value.(map[string]any); ok {
				value = nested["url"]
			}
			if text, ok := value.(string); ok {
				if part, ok := geminiImagePart(text, mimeType); ok {
					parts = append(parts, part)
					break
				}
			}
		}
	}
	return parts
}

func geminiImagePart(rawURL string, mimeType string) (dto.GeminiPart, bool) {
	rawURL = strings.TrimSpace(rawURL)
	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "data:image/") {
		comma := strings.IndexByte(rawURL, ',')
		if comma < 0 {
			return dto.GeminiPart{}, false
		}
		metadata := rawURL[len("data:"):comma]
		segments := strings.Split(metadata, ";")
		if len(segments) < 2 || !strings.EqualFold(segments[len(segments)-1], "base64") {
			return dto.GeminiPart{}, false
		}
		meta := segments[0]
		if !strings.HasPrefix(strings.ToLower(meta), "image/") {
			return dto.GeminiPart{}, false
		}
		return geminiInlineImagePart(meta, rawURL[comma+1:])
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || parsed.Hostname() == "" ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") {
			return dto.GeminiPart{}, false
		}
		return dto.GeminiPart{FileData: &dto.GeminiFileData{MimeType: defaultImageMime(mimeTypeFromURL(rawURL, mimeType)), FileUri: rawURL}}, true
	}
	return dto.GeminiPart{}, false
}

func geminiInlineImagePart(mimeType string, encoded string) (dto.GeminiPart, bool) {
	if encoded == "" || !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return dto.GeminiPart{}, false
	}
	if _, err := base64.StdEncoding.DecodeString(encoded); err != nil {
		if _, rawErr := base64.RawStdEncoding.DecodeString(encoded); rawErr != nil {
			return dto.GeminiPart{}, false
		}
	}
	return dto.GeminiPart{InlineData: &dto.GeminiInlineData{MimeType: mimeType, Data: encoded}}, true
}

func mimeTypeFromURL(rawURL string, fallback string) string {
	if fallback != "" {
		return fallback
	}
	parsed, err := url.Parse(rawURL)
	if err == nil {
		switch strings.ToLower(path.Ext(parsed.Path)) {
		case ".jpg", ".jpeg":
			return "image/jpeg"
		case ".webp":
			return "image/webp"
		case ".gif":
			return "image/gif"
		}
	}
	return "image/png"
}

func defaultImageMime(value string) string {
	if strings.HasPrefix(strings.ToLower(value), "image/") {
		return value
	}
	return "image/png"
}

func stringValue(item map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := item[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}

func geminiBillingMetadataFromOpenAIUsage(usage *dto.Usage) (dto.GeminiUsageMetadata, bool) {
	if usage == nil || usage.BillingUsage == nil || usage.BillingUsage.GeminiUsageMetadata == nil {
		return dto.GeminiUsageMetadata{}, false
	}
	if usage.BillingUsage.Source != dto.BillingUsageSourceGeminiChat && usage.BillingUsage.Semantic != dto.BillingUsageSemanticGemini {
		return dto.GeminiUsageMetadata{}, false
	}
	billingUsage := dto.CloneBillingUsage(usage.BillingUsage)
	if billingUsage == nil || billingUsage.GeminiUsageMetadata == nil {
		return dto.GeminiUsageMetadata{}, false
	}
	return *billingUsage.GeminiUsageMetadata, true
}

func openAIBillingUsageFromUsage(usage *dto.Usage) *dto.BillingUsage {
	if usage == nil {
		return nil
	}
	if existingBillingUsage := dto.CloneBillingUsage(usage.BillingUsage); existingBillingUsage != nil && existingBillingUsage.OpenAIUsage != nil {
		if existingBillingUsage.Source == dto.BillingUsageSourceOAIChat ||
			existingBillingUsage.Source == dto.BillingUsageSourceOAIResponses ||
			existingBillingUsage.Semantic == dto.BillingUsageSemanticOpenAI {
			return existingBillingUsage
		}
	}
	return dto.NewOpenAIChatBillingUsage(usage)
}
