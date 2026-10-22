package claude

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func ClaudeMessagesToResponsesHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	logger.LogDebug(c, "Claude responses response body: %s", body)

	var claudeResponse dto.ClaudeResponse
	if err := common.Unmarshal(body, &claudeResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	info.SetUpstreamResponseModelName(claudeResponse.Model)
	if claudeResponse.Message != nil {
		info.SetUpstreamResponseModelName(claudeResponse.Message.Model)
	}
	if claudeError := claudeResponse.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
		return nil, types.WithClaudeError(*claudeError, resp.StatusCode)
	}
	maybeMarkClaudeRefusal(c, claudeResponse.StopReason)
	recordClaudeResponseTools(c, info, &claudeResponse)

	converted, err := relayconvert.ConvertResponse(c, info, types.RelayFormatOpenAIResponses, &claudeResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	responsesResponse, ok := converted.Value.(*dto.OpenAIResponsesResponse)
	if !ok {
		return nil, types.NewOpenAIError(fmt.Errorf("expected OpenAI responses response, got %T", converted.Value), types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	usage := converted.Usage
	if usage == nil || usage.TotalTokens == 0 {
		usage = service.ResponseText2Usage(c, responsesOutputText(responsesResponse), info.UpstreamModelName, info.GetEstimatePromptTokens())
		usage.UsageSemantic = "anthropic"
		usage.UsageSource = "anthropic"
		responsesResponse.Usage = relayconvert.UsageFromChatUsage(usage)
	}

	responseBody, err := common.Marshal(responsesResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
	}
	if usageError := service.TextUsageError(c, info, usage); usageError != nil {
		return usage, usageError
	}
	service.IOCopyBytesGracefully(c, resp, responseBody)
	return usage, nil
}

func ClaudeMessagesToResponsesStreamHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	created := common.GetTimestamp()
	state, err := relayconvert.NewResponseStreamState(types.RelayFormatClaude, types.RelayFormatOpenAIResponses, relayconvert.ResponseStreamOptions{
		ID:      helper.GetResponseID(c),
		Model:   info.UpstreamModelName,
		Created: created,
	})
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	claudeInfo := &ClaudeResponseInfo{
		ResponseId:   helper.GetResponseID(c),
		Created:      created,
		Model:        info.UpstreamModelName,
		ResponseText: strings.Builder{},
		Usage:        &dto.Usage{},
	}
	var streamErr *types.NewAPIError
	clientWriteFailed := false

	sendEvent := func(event relayconvert.ChatToResponsesStreamEvent) bool {
		data, err := common.Marshal(event.Payload)
		if err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeJsonMarshalFailed, http.StatusInternalServerError)
			return false
		}
		if err := helper.ResponseChunkData(c, dto.ResponsesStreamResponse{Type: event.Type}, string(data)); err != nil {
			if errors.Is(err, context.Canceled) {
				clientWriteFailed = true
				return false
			}
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			return false
		}
		return true
	}

	helper.StreamScannerHandlerWithClientDisconnectDrain(c, resp, info, channel.ClaudeResponsesUsageDrainTimeout, func(data string, result *helper.StreamResult) {
		var claudeResponse dto.ClaudeResponse
		if err := common.UnmarshalJsonStr(data, &claudeResponse); err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
			result.Stop(streamErr)
			return
		}
		info.SetUpstreamResponseModelName(claudeResponse.Model)
		if claudeResponse.Message != nil {
			info.SetUpstreamResponseModelName(claudeResponse.Message.Model)
		}
		if claudeError := claudeResponse.GetClaudeError(); claudeError != nil && claudeError.Type != "" {
			streamErr = types.WithClaudeError(*claudeError, resp.StatusCode)
			result.Stop(streamErr)
			return
		}
		if claudeResponse.StopReason != "" {
			maybeMarkClaudeRefusal(c, claudeResponse.StopReason)
		}
		if claudeResponse.Delta != nil && claudeResponse.Delta.StopReason != nil {
			maybeMarkClaudeRefusal(c, *claudeResponse.Delta.StopReason)
		}
		clientDisconnected := clientWriteFailed || errors.Is(c.Request.Context().Err(), context.Canceled)
		if !clientDisconnected || claudeResponse.Type != "content_block_delta" {
			FormatClaudeResponseInfo(&claudeResponse, nil, claudeInfo)
		}
		countClaudeStreamBillableTools(c, info, &claudeResponse)
		if clientDisconnected {
			if claudeResponse.Type == "message_stop" {
				result.Done()
			}
			return
		}

		converted, err := relayconvert.ConvertStreamResponseChunk(c, info, state, &claudeResponse)
		if err != nil {
			streamErr = types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
			result.Stop(streamErr)
			return
		}
		for _, convertedEvent := range converted {
			event, ok := convertedEvent.Value.(relayconvert.ChatToResponsesStreamEvent)
			if !ok {
				streamErr = types.NewOpenAIError(fmt.Errorf("expected OpenAI responses stream event, got %T", convertedEvent.Value), types.ErrorCodeBadResponse, http.StatusInternalServerError)
				result.Stop(streamErr)
				return
			}
			if !sendEvent(event) {
				if clientWriteFailed {
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, context.Canceled)
				} else {
					result.Stop(streamErr)
				}
				return
			}
		}
		if claudeResponse.Type == "message_stop" {
			result.Done()
		}
	})
	// 真正的上游或转换错误继续交给外层处理，不能因并发客户端取消而吞掉。
	if streamErr != nil {
		return nil, streamErr
	}
	if isClaudeResponsesClientGone(c, info) {
		return claudeResponsesDisconnectedUsage(c, info, claudeInfo.Usage), nil
	}

	// 先冻结上游用量；最终事件写出时仍可能断开，此时不能结算随后补入的估算量。
	observedUsage := *claudeInfo.Usage
	observedUsage.BillingUsage = dto.CloneBillingUsage(claudeInfo.Usage.BillingUsage)
	HandleStreamFinalResponse(c, info, claudeInfo)
	openAIUsage := buildOpenAIStyleUsageFromClaudeUsage(claudeInfo.Usage)
	state.SetUsage(&openAIUsage)
	finalEvents, err := relayconvert.FinalizeStreamResponse(c, info, state)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	for _, convertedEvent := range finalEvents {
		event, ok := convertedEvent.Value.(relayconvert.ChatToResponsesStreamEvent)
		if !ok {
			return nil, types.NewOpenAIError(fmt.Errorf("expected OpenAI responses stream event, got %T", convertedEvent.Value), types.ErrorCodeBadResponse, http.StatusInternalServerError)
		}
		if !sendEvent(event) {
			if clientWriteFailed {
				return claudeResponsesDisconnectedUsage(c, info, &observedUsage), nil
			}
			return nil, streamErr
		}
	}
	return claudeInfo.Usage, nil
}

// claudeResponsesDisconnectedUsage 只在扫描器及写入协程全部结束后调用。
// EOF 只代表上游读取结束；最终下游写入取消时仍须以 client_gone 结束本次传输，
// 保留已观测用量交给统一结算，并避免客户端取消污染上游质量指标。
func claudeResponsesDisconnectedUsage(c *gin.Context, info *relaycommon.RelayInfo, usage *dto.Usage) *dto.Usage {
	previous := info.StreamStatus
	// 上游读失败或超时先发生时，后来的客户端取消不能覆盖原有故障证据。
	if previous == nil || (previous.EndError == nil && !previous.HasErrors() &&
		(previous.EndReason == relaycommon.StreamEndReasonNone || previous.IsNormalEnd())) {
		info.StreamStatus = relaycommon.NewStreamStatus()
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, context.Canceled)
	}
	common.SetContextKey(c, constant.ContextKeyLocalCountTokens, false)
	return usage
}

func isClaudeResponsesClientGone(c *gin.Context, info *relaycommon.RelayInfo) bool {
	if info != nil && info.StreamStatus != nil && info.StreamStatus.EndReason == relaycommon.StreamEndReasonClientGone {
		return true
	}
	return c != nil && c.Request != nil && errors.Is(c.Request.Context().Err(), context.Canceled)
}

func recordClaudeResponseTools(c *gin.Context, info *relaycommon.RelayInfo, response *dto.ClaudeResponse) {
	if response == nil {
		return
	}
	if response.Usage != nil && response.Usage.ServerToolUse != nil && response.Usage.ServerToolUse.WebSearchRequests > 0 {
		c.Set("claude_web_search_requests", response.Usage.ServerToolUse.WebSearchRequests)
	}
	for i := range response.Content {
		if response.Content[i].Type == "tool_use" {
			info.CountBillableToolCall(dto.BuildInCallToolUse, response.Content[i].Name)
		}
	}
}

func responsesOutputText(response *dto.OpenAIResponsesResponse) string {
	if response == nil {
		return ""
	}
	var text strings.Builder
	for i := range response.Output {
		for j := range response.Output[i].Content {
			text.WriteString(response.Output[i].Content[j].Text)
		}
	}
	return text.String()
}
