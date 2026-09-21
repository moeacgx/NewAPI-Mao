package xai

import (
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

func mergeXAIUsage(dst, src *dto.Usage) {
	if dst == nil || src == nil {
		return
	}
	previous := *dst
	previousPromptDetails := dst.PromptTokensDetails
	previousInputDetails := dst.InputTokensDetails
	*dst = *src
	if !src.HasPromptTokens && previous.HasPromptTokens {
		dst.PromptTokens = previous.PromptTokens
		dst.HasPromptTokens = true
	}
	if !src.HasCompletionTokens && previous.HasCompletionTokens {
		dst.CompletionTokens = previous.CompletionTokens
		dst.HasCompletionTokens = true
	}
	if !src.HasTotalTokens && previous.HasTotalTokens {
		dst.TotalTokens = previous.TotalTokens
		dst.HasTotalTokens = true
	}
	if !src.HasPromptCacheHitTokens && !src.PromptTokensDetails.HasCachedTokens && src.PromptTokensDetails.CachedTokens == 0 &&
		(previous.HasPromptCacheHitTokens || previousPromptDetails.HasCachedTokens || previousPromptDetails.CachedTokens != 0) {
		dst.PromptCacheHitTokens = previous.PromptCacheHitTokens
		dst.HasPromptCacheHitTokens = previous.HasPromptCacheHitTokens
		dst.PromptTokensDetails = previousPromptDetails
	}
	if src.InputTokensDetails == nil && previousInputDetails != nil {
		dst.InputTokensDetails = previousInputDetails
	}
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
			mergeXAIUsage(usage, xAIResp.Usage)
			normalizeXAIUsage(usage)
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
	normalizeXAIUsage(usage)
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
			mergeXAIUsage(usage, chunk.Usage)
			normalizeXAIUsage(usage)
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
	normalizeXAIUsage(usage)
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
