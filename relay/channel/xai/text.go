package xai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func normalizeXAIUsage(usage *dto.Usage) {
	if usage == nil {
		return
	}
	cacheTokens := usage.PromptTokensDetails.CachedTokens
	cachePresent := usage.PromptTokensDetails.HasCachedTokens || cacheTokens != 0
	if !cachePresent && usage.InputTokensDetails != nil {
		cacheTokens = usage.InputTokensDetails.CachedTokens
		cachePresent = usage.InputTokensDetails.HasCachedTokens || cacheTokens != 0
	}
	if !cachePresent && (usage.HasPromptCacheHitTokens || usage.PromptCacheHitTokens != 0) {
		cacheTokens = usage.PromptCacheHitTokens
		cachePresent = true
	}
	if !cachePresent {
		return
	}
	if cacheTokens < 0 {
		cacheTokens = 0
	}
	if usage.HasPromptTokens && usage.PromptTokens >= 0 && cacheTokens > usage.PromptTokens {
		cacheTokens = usage.PromptTokens
	}
	usage.PromptTokensDetails.CachedTokens = cacheTokens
	usage.PromptTokensDetails.HasCachedTokens = true
}

func mergeXAIUsage(fields map[string]json.RawMessage, data string) (*dto.Usage, error) {
	var chunk struct {
		Usage map[string]json.RawMessage `json:"usage"`
	}
	if err := common.UnmarshalJsonStr(data, &chunk); err != nil {
		return nil, err
	}
	// 原始字段独立累计，避免把归一化补出的值当成上游声明的标准字段。
	for name, value := range chunk.Usage {
		if common.GetJsonType(value) == "null" {
			continue
		}
		switch name {
		case "prompt_tokens_details", "input_tokens_details", "completion_tokens_details", "output_tokens_details":
			details := make(map[string]json.RawMessage)
			if previous := fields[name]; len(previous) > 0 {
				if err := common.Unmarshal(previous, &details); err != nil {
					return nil, err
				}
			}
			var incoming map[string]json.RawMessage
			if err := common.Unmarshal(value, &incoming); err != nil {
				return nil, err
			}
			for key, detail := range incoming {
				if common.GetJsonType(detail) != "null" {
					details[key] = detail
				}
			}
			merged, err := common.Marshal(details)
			if err != nil {
				return nil, err
			}
			fields[name] = merged
		default:
			fields[name] = value
		}
	}
	merged, err := common.Marshal(fields)
	if err != nil {
		return nil, err
	}
	var usage dto.Usage
	if err := common.Unmarshal(merged, &usage); err != nil {
		return nil, err
	}
	return &usage, nil
}

func normalizeXAIClaudeUsage(usage *dto.Usage) {
	normalizeXAIUsage(usage)
	if usage == nil || usage.InputTokensDetails == nil {
		return
	}
	// 通用 Claude 转换器优先读取 input_tokens_details，xAI 边界统一为结算值。
	details := *usage.InputTokensDetails
	details.CachedTokens = usage.PromptTokensDetails.CachedTokens
	details.HasCachedTokens = usage.PromptTokensDetails.HasCachedTokens
	usage.InputTokensDetails = &details
}

func streamResponseXAI2OpenAI(xAIResp *dto.ChatCompletionsStreamResponse, usage *dto.Usage) *dto.ChatCompletionsStreamResponse {
	if xAIResp == nil {
		return nil
	}
	if xAIResp.Usage != nil {
		xAIResp.Usage.CompletionTokens = usage.CompletionTokens
	}
	openAIResp := &dto.ChatCompletionsStreamResponse{
		Id:      xAIResp.Id,
		Object:  xAIResp.Object,
		Created: xAIResp.Created,
		Model:   xAIResp.Model,
		Choices: xAIResp.Choices,
		Usage:   xAIResp.Usage,
	}

	return openAIResp
}

func xAIStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if info.RelayFormat == types.RelayFormatClaude {
		return xAIClaudeStreamHandler(c, info, resp)
	}
	usage := &dto.Usage{}
	usageFields := make(map[string]json.RawMessage)
	var responseTextBuilder strings.Builder
	var toolCount int
	var containStreamUsage bool

	helper.SetEventStreamHeaders(c)

	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		var xAIResp *dto.ChatCompletionsStreamResponse
		if err := common.UnmarshalJsonStr(data, &xAIResp); err != nil {
			common.SysLog("error unmarshalling stream response: " + err.Error())
			sr.Error(err)
			return
		}
		info.SetUpstreamResponseModelName(xAIResp.Model)

		// 把 xAI 的usage转换为 OpenAI 的usage
		if xAIResp.Usage != nil {
			containStreamUsage = true
			merged, err := mergeXAIUsage(usageFields, data)
			if err != nil {
				sr.Error(err)
				return
			}
			usage = merged
			usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
		}

		openaiResponse := streamResponseXAI2OpenAI(xAIResp, usage)
		_ = openai.ProcessStreamResponse(*openaiResponse, &responseTextBuilder, &toolCount)
		if err := helper.ObjectData(c, openaiResponse); err != nil {
			common.SysLog(err.Error())
			sr.Error(err)
		}
	})

	if !containStreamUsage {
		usage = service.ResponseText2Usage(c, responseTextBuilder.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
		usage.CompletionTokens += toolCount * 7
	}
	normalizeXAIUsage(usage)

	helper.Done(c)
	service.CloseResponseBodyGracefully(resp)
	return usage, nil
}

func xAIHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if info.RelayFormat == types.RelayFormatClaude {
		return xAIClaudeHandler(c, info, resp)
	}
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	var xaiResponse ChatCompletionResponse
	err = common.Unmarshal(responseBody, &xaiResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	info.SetUpstreamResponseModelName(xaiResponse.Model)
	if xaiResponse.Usage != nil {
		normalizeXAIUsage(xaiResponse.Usage)
		xaiResponse.Usage.CompletionTokens = xaiResponse.Usage.TotalTokens - xaiResponse.Usage.PromptTokens
		xaiResponse.Usage.CompletionTokenDetails.TextTokens = xaiResponse.Usage.CompletionTokens - xaiResponse.Usage.CompletionTokenDetails.ReasoningTokens
	}

	// new body
	encodeJson, err := common.Marshal(xaiResponse)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}

	service.IOCopyBytesGracefully(c, resp, encodeJson)

	return xaiResponse.Usage, nil
}

func xAIClaudeHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewError(errors.New("invalid xAI response"), types.ErrorCodeBadResponse)
	}
	defer service.CloseResponseBodyGracefully(resp)
	var response *dto.OpenAITextResponse
	if err := common.DecodeJson(resp.Body, &response); err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	if response == nil {
		return nil, types.NewError(errors.New("empty xAI response"), types.ErrorCodeBadResponseBody)
	}
	if response.Error != nil {
		openAIError := response.GetOpenAIError()
		if openAIError == nil {
			return nil, types.NewError(errors.New("invalid xAI error response"), types.ErrorCodeBadResponseBody)
		}
		return nil, types.WithOpenAIError(*openAIError, http.StatusBadGateway)
	}
	if len(response.Choices) == 0 {
		return nil, types.NewError(errors.New("xAI response has no choices"), types.ErrorCodeBadResponseBody)
	}
	info.SetUpstreamResponseModelName(response.Model)
	usage := &response.Usage
	if !usage.HasPromptTokens && !usage.HasCompletionTokens && !usage.HasTotalTokens {
		var text strings.Builder
		for _, choice := range response.Choices {
			text.WriteString(choice.Message.StringContent())
			text.WriteString(choice.Message.GetReasoningContent())
			for _, tool := range choice.Message.ParseToolCalls() {
				text.WriteString(tool.Function.Name)
				text.WriteString(tool.Function.Arguments)
			}
		}
		estimated := service.ResponseText2Usage(c, text.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
		usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens = estimated.PromptTokens, estimated.CompletionTokens, estimated.TotalTokens
	} else if usage.HasTotalTokens && usage.HasPromptTokens {
		usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
	}
	normalizeXAIClaudeUsage(usage)
	usage.CompletionTokenDetails.TextTokens = usage.CompletionTokens - usage.CompletionTokenDetails.ReasoningTokens
	converted, err := relayconvert.ConvertResponse(c, info, types.RelayFormatClaude, response)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	encoded, err := common.Marshal(converted.Value)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	service.IOCopyBytesGracefully(c, resp, encoded)
	return usage, nil
}

func xAIClaudeStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewError(errors.New("invalid xAI response"), types.ErrorCodeBadResponse)
	}
	defer service.CloseResponseBodyGracefully(resp)
	state, err := relayconvert.NewResponseStreamState(types.RelayFormatOpenAI, types.RelayFormatClaude, relayconvert.ResponseStreamOptions{})
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	usage := &dto.Usage{}
	usageFields := make(map[string]json.RawMessage)
	var responseText strings.Builder
	var toolCount int
	var streamErr *types.NewAPIError
	var sawChoices bool
	helper.StreamScannerHandler(c, resp, info, func(data string, sr *helper.StreamResult) {
		var envelope struct {
			*dto.ChatCompletionsStreamResponse
			Error any `json:"error"`
		}
		if err := common.UnmarshalJsonStr(data, &envelope); err != nil {
			streamErr = types.NewError(err, types.ErrorCodeBadResponseBody)
			sr.Stop(streamErr)
			return
		}
		if envelope.Error != nil {
			openAIError := dto.GetOpenAIError(envelope.Error)
			if openAIError == nil {
				streamErr = types.NewError(errors.New("invalid xAI error response"), types.ErrorCodeBadResponseBody)
				sr.Stop(streamErr)
				return
			}
			streamErr = types.WithOpenAIError(*openAIError, http.StatusBadGateway)
			sr.Stop(streamErr)
			return
		}
		chunk := envelope.ChatCompletionsStreamResponse
		if chunk == nil || (len(chunk.Choices) == 0 && chunk.Usage == nil) {
			streamErr = types.NewError(errors.New("empty xAI stream chunk"), types.ErrorCodeBadResponseBody)
			sr.Stop(streamErr)
			return
		}
		info.SetUpstreamResponseModelName(chunk.Model)
		if chunk.Usage != nil {
			merged, err := mergeXAIUsage(usageFields, data)
			if err != nil {
				streamErr = types.NewError(err, types.ErrorCodeBadResponseBody)
				sr.Stop(streamErr)
				return
			}
			usage = merged
			if usage.HasPromptTokens && usage.HasTotalTokens {
				usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
				usage.HasCompletionTokens = true
			}
			if usage.HasPromptTokens && usage.HasCompletionTokens {
				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				usage.HasTotalTokens = true
			}
		}
		if len(chunk.Choices) == 0 {
			return
		}
		sawChoices = true
		_ = openai.ProcessStreamResponse(*chunk, &responseText, &toolCount)
		// xAI 的缓存详情可能晚于结束原因到达，先转内容，收齐用量后再结束 Claude 流。
		if chunk.Choices[0].FinishReason != nil && *chunk.Choices[0].FinishReason != "" {
			info.EnsureClaudeConvertInfo().FinishReason = *chunk.Choices[0].FinishReason
		}
		for i := range chunk.Choices {
			chunk.Choices[i].FinishReason = nil
		}
		chunk.Usage = nil
		results, err := relayconvert.ConvertStreamResponseChunk(c, info, state, chunk)
		if err == nil {
			err = writeXAIClaudeStreamResults(c, results)
		}
		if err != nil {
			streamErr = types.NewError(err, types.ErrorCodeBadResponseBody)
			sr.Stop(streamErr)
		}
	})
	// 扫描结束后先规范已收到的用量，取消或错误返回也不能丢失缓存计费信息。
	normalizeXAIClaudeUsage(usage)
	if streamErr != nil {
		return usage, streamErr
	}
	if c.Request.Context().Err() != nil || (info.StreamStatus != nil && info.StreamStatus.EndReason == relaycommon.StreamEndReasonClientGone) {
		return usage, nil
	}
	if info.StreamStatus != nil && (!info.StreamStatus.IsNormalEnd() || info.StreamStatus.HasErrors()) {
		return usage, types.NewError(fmt.Errorf("xAI stream interrupted: %s", info.StreamStatus.Summary()), types.ErrorCodeBadResponseBody)
	}
	if !sawChoices {
		return usage, types.NewError(errors.New("xAI stream has no choices"), types.ErrorCodeBadResponseBody)
	}
	if !usage.HasPromptTokens && !usage.HasCompletionTokens && !usage.HasTotalTokens {
		estimated := service.ResponseText2Usage(c, responseText.String(), info.UpstreamModelName, info.GetEstimatePromptTokens())
		usage.PromptTokens, usage.CompletionTokens = estimated.PromptTokens, estimated.CompletionTokens+toolCount*7
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	} else if usage.HasTotalTokens && usage.HasPromptTokens {
		usage.CompletionTokens = usage.TotalTokens - usage.PromptTokens
	}
	normalizeXAIClaudeUsage(usage)
	info.EnsureClaudeConvertInfo().Usage = usage
	results, err := relayconvert.FinalizeStreamResponse(c, info, state)
	if err == nil {
		err = writeXAIClaudeStreamResults(c, results)
	}
	if err != nil {
		return usage, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	return usage, nil
}

func writeXAIClaudeStreamResults(c *gin.Context, results []relayconvert.ResponseResult) error {
	for _, result := range results {
		response, ok := result.Value.(*dto.ClaudeResponse)
		if !ok {
			return fmt.Errorf("expected Claude stream response, got %T", result.Value)
		}
		if err := helper.ClaudeData(c, *response); err != nil {
			return err
		}
	}
	return nil
}
