package claude

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	rootconstant "github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cancellingResponseWriter struct {
	gin.ResponseWriter
	cancel       context.CancelFunc
	event        string
	disconnected chan struct{}
}

type failingUpstreamResponseBody struct {
	cancel context.CancelFunc
}

func (*failingUpstreamResponseBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func (r *failingUpstreamResponseBody) Close() error {
	r.cancel()
	return nil
}

func (w *cancellingResponseWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.event) {
		w.cancel()
		if w.disconnected != nil {
			select {
			case w.disconnected <- struct{}{}:
			default:
			}
		}
		return 0, context.Canceled
	}
	return w.ResponseWriter.Write(p)
}

func (w *cancellingResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func TestConvertOpenAIResponsesRequestUsesRelaykitAndNormalizesSampling(t *testing.T) {
	c, _ := newClaudeBridgeContext(t)
	maxTokens := uint(256)
	temperature := 0.7
	topP := 0.9
	request := dto.OpenAIResponsesRequest{
		Model:           "claude-sonnet-4-6",
		MaxOutputTokens: &maxTokens,
		Temperature:     &temperature,
		TopP:            &topP,
		Input: mustClaudeBridgeRaw(t, []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "input_text", "text": "cached prompt", "cache_control": map[string]any{"type": "ephemeral"}},
				},
			},
		}),
		Tools: mustClaudeBridgeRaw(t, []map[string]any{
			{"type": "function", "name": "lookup", "parameters": map[string]any{"type": "object", "properties": map[string]any{}}},
		}),
	}
	info := newClaudeBridgeInfo(false)

	converted, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, info, request)
	require.NoError(t, err)
	claudeRequest, ok := converted.(*dto.ClaudeRequest)
	require.True(t, ok)
	assert.Equal(t, &temperature, claudeRequest.Temperature)
	assert.Equal(t, &topP, claudeRequest.TopP)
	assert.Nil(t, claudeRequest.TopK)
	assert.Equal(t, types.RelayFormat(types.RelayFormatClaude), info.FinalRequestRelayFormat)
	require.Len(t, claudeRequest.GetTools(), 1)
	require.Len(t, claudeRequest.Messages, 1)
	parts, err := claudeRequest.Messages[0].ParseContent()
	require.NoError(t, err)
	require.Len(t, parts, 1)
	assert.JSONEq(t, `{"type":"ephemeral"}`, string(parts[0].CacheControl))
}

func TestConvertOpenAIResponsesRequestRejectsLossyCustomTool(t *testing.T) {
	c, _ := newClaudeBridgeContext(t)
	maxTokens := uint(256)
	_, err := (&Adaptor{}).ConvertOpenAIResponsesRequest(c, newClaudeBridgeInfo(false), dto.OpenAIResponsesRequest{
		Model:           "claude-sonnet-4-6",
		MaxOutputTokens: &maxTokens,
		Input: mustClaudeBridgeRaw(t, []map[string]any{
			{"type": "custom_tool_call", "call_id": "call_1", "name": "apply_patch", "input": "patch"},
		}),
	})
	require.ErrorContains(t, err, "cannot safely represent")
}

func TestClaudeMessagesToResponsesHandlerPreservesOrderedOutputsAndCacheUsage(t *testing.T) {
	c, recorder := newClaudeBridgeContext(t)
	response := map[string]any{
		"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-4-6",
		"content": []map[string]any{
			{"type": "thinking", "thinking": "reasoning"},
			{"type": "text", "text": "hello"},
			{"type": "tool_use", "id": "call_1", "name": "lookup", "input": map[string]any{"q": "x"}},
		},
		"stop_reason": "tool_use",
		"usage": map[string]any{
			"input_tokens": 6, "cache_creation_input_tokens": 3, "cache_read_input_tokens": 4, "output_tokens": 2,
			"cache_creation": map[string]any{"ephemeral_5m_input_tokens": 2, "ephemeral_1h_input_tokens": 1},
		},
	}
	body, err := common.Marshal(response)
	require.NoError(t, err)

	usage, relayErr := ClaudeMessagesToResponsesHandler(c, newClaudeBridgeInfo(false), &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	})
	require.Nil(t, relayErr)
	require.NotNil(t, usage)
	assert.Equal(t, 13, usage.PromptTokens)
	require.NotNil(t, usage.InputTokensDetails)
	assert.Equal(t, 4, usage.InputTokensDetails.CachedTokens)
	assert.Equal(t, 3, usage.PromptTokensDetails.CacheCreationTokensTotal())

	var result dto.OpenAIResponsesResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &result))
	assert.Equal(t, "response", result.Object)
	require.Len(t, result.Output, 3)
	assert.Equal(t, "message", result.Output[0].Type)
	assert.Equal(t, "hello", result.Output[0].Content[0].Text)
	assert.Equal(t, "reasoning", result.Output[1].Type)
	assert.Equal(t, "reasoning", result.Output[1].Content[0].Text)
	assert.Equal(t, "function_call", result.Output[2].Type)
	assert.Equal(t, "lookup", result.Output[2].Name)
	require.NotNil(t, result.Usage)
	assert.Equal(t, 13, result.Usage.InputTokens)
	assert.Equal(t, 2, result.Usage.OutputTokens)
	assert.Equal(t, 3, result.Usage.InputTokensDetails.CacheCreationTokensTotal())
}

func TestClaudeMessagesToResponsesStreamHandlerEmitsOrderedToolAndCacheEvents(t *testing.T) {
	c, recorder := newClaudeBridgeContext(t)
	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6","usage":{"input_tokens":6,"cache_creation_input_tokens":3,"cache_read_input_tokens":4,"output_tokens":0,"cache_creation":{"ephemeral_5m_input_tokens":2,"ephemeral_1h_input_tokens":1}}}}`,
		`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`,
		`data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"call_1","name":"lookup","input":{}}}`,
		`data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"q\":\"x\"}"}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":"opaque-stop"},"usage":{"output_tokens":2}}`,
		`data: {"type":"message_stop"}`,
		``,
	}, "\n")

	oldTimeout := rootconstant.StreamingTimeout
	rootconstant.StreamingTimeout = 30
	t.Cleanup(func() { rootconstant.StreamingTimeout = oldTimeout })
	usage, relayErr := ClaudeMessagesToResponsesStreamHandler(c, newClaudeBridgeInfo(true), &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	})
	require.Nil(t, relayErr)
	require.NotNil(t, usage)
	assert.Equal(t, 6, usage.PromptTokens)
	assert.Equal(t, 2, usage.CompletionTokens)
	assert.Equal(t, 4, usage.PromptTokensDetails.CachedTokens)
	assert.Equal(t, 3, usage.PromptTokensDetails.CacheCreationTokensTotal())

	output := recorder.Body.String()
	assert.Equal(t, "text/event-stream", recorder.Header().Get("Content-Type"))
	requireOrderedClaudeBridgeEvents(t, output,
		"event: response.created",
		"event: response.output_item.added",
		"event: response.output_text.delta",
		"event: response.output_item.added",
		"event: response.function_call_arguments.delta",
		"event: response.output_text.done",
		"event: response.completed",
	)
	assert.Contains(t, output, `"name":"lookup"`)
	assert.Contains(t, output, `"input_tokens":13`)
	assert.Contains(t, output, `"output_tokens":2`)
}

func TestClaudeResponsesDisconnectReturnsOnlyObservedUsage(t *testing.T) {
	const start = `data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6","usage":{"input_tokens":7,"output_tokens":0,"cache_read_input_tokens":4}}}` + "\n"
	const emptyStart = `data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6"}}` + "\n"
	const text = `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}` + "\n"
	const delta = `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}` + "\n"
	for _, tc := range []struct {
		name, stream, cancelEvent        string
		wantInput, wantCache, wantOutput int
	}{
		{name: "首帧写出取消保留输入和缓存", stream: start, cancelEvent: "event: response.created", wantInput: 7, wantCache: 4},
		{name: "内容写出取消不补估输出", stream: start + text, cancelEvent: "event: response.output_text.delta", wantInput: 7, wantCache: 4},
		{name: "无用量的内容写出取消不收费", stream: emptyStart + text, cancelEvent: "event: response.output_text.delta"},
		{name: "最终事件写出取消保留原始用量", stream: start + text, cancelEvent: "event: response.completed", wantInput: 7, wantCache: 4},
		{name: "最终事件写出取消保留已报告输出", stream: start + text + delta, cancelEvent: "event: response.completed", wantInput: 7, wantCache: 4, wantOutput: 3},
		{name: "最终事件写出取消不结算本地估算", stream: emptyStart + text, cancelEvent: "event: response.completed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newClaudeBridgeContext(t)
			requestCtx, cancel := context.WithCancel(ctx.Request.Context())
			t.Cleanup(cancel)
			ctx.Request = ctx.Request.WithContext(requestCtx)
			ctx.Writer = &cancellingResponseWriter{ResponseWriter: ctx.Writer, cancel: cancel, event: tc.cancelEvent}
			body := io.NopCloser(strings.NewReader(tc.stream))
			info := newClaudeBridgeInfo(true)
			info.SetEstimatePromptTokens(1200)

			usage, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, info, &http.Response{StatusCode: http.StatusOK, Body: body})

			require.ErrorIs(t, requestCtx.Err(), context.Canceled)
			require.Nil(t, apiErr)
			require.NotNil(t, usage)
			assert.Equal(t, tc.wantInput, usage.PromptTokens)
			assert.Equal(t, tc.wantCache, usage.PromptTokensDetails.CachedTokens)
			assert.Equal(t, tc.wantOutput, usage.CompletionTokens)
			assert.False(t, common.GetContextKeyBool(ctx, rootconstant.ContextKeyLocalCountTokens))
			assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
			assert.False(t, info.StreamStatus.HasErrors(), "客户端取消不应记成上游流错误")
			if tc.wantInput > 0 {
				require.NotNil(t, usage.BillingUsage)
				require.NotNil(t, usage.BillingUsage.ClaudeUsage)
				assert.Equal(t, tc.wantInput, usage.BillingUsage.ClaudeUsage.InputTokens)
				assert.Equal(t, tc.wantOutput, usage.BillingUsage.ClaudeUsage.OutputTokens)
			}
		})
	}
}

func TestClaudeMessagesToResponsesStreamHandlerDrainsFinalUsageAfterClientDisconnect(t *testing.T) {
	ctx, recorder := newClaudeBridgeContext(t)
	requestCtx, cancel := context.WithCancel(ctx.Request.Context())
	t.Cleanup(cancel)
	ctx.Request = ctx.Request.WithContext(requestCtx)
	disconnected := make(chan struct{}, 1)
	ctx.Writer = &cancellingResponseWriter{ResponseWriter: ctx.Writer, cancel: cancel, event: "event: response.created", disconnected: disconnected}
	reader, writer := io.Pipe()
	body := &http.Response{StatusCode: http.StatusOK, Body: reader}
	info := newClaudeBridgeInfo(true)
	result := make(chan struct {
		usage *dto.Usage
		err   *types.NewAPIError
	}, 1)
	go func() {
		usage, err := ClaudeMessagesToResponsesStreamHandler(ctx, info, body)
		result <- struct {
			usage *dto.Usage
			err   *types.NewAPIError
		}{usage: usage, err: err}
	}()

	_, err := io.WriteString(writer, `data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6","usage":{"input_tokens":7,"output_tokens":0}}}`+"\n")
	require.NoError(t, err)
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("downstream cancellation was not observed")
	}
	require.ErrorIs(t, requestCtx.Err(), context.Canceled)

	_, err = io.WriteString(writer, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`+"\n")
	require.NoError(t, err)
	_, err = io.WriteString(writer, `data: {"type":"message_stop"}`+"\n")
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	select {
	case outcome := <-result:
		require.Nil(t, outcome.err)
		require.NotNil(t, outcome.usage)
		assert.Equal(t, 7, outcome.usage.PromptTokens)
		assert.Equal(t, 3, outcome.usage.CompletionTokens)
		assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
		assert.False(t, common.GetContextKeyBool(ctx, rootconstant.ContextKeyLocalCountTokens))
	case <-time.After(2 * time.Second):
		t.Fatal("response handler did not finish after receiving final upstream usage")
	}
	assert.Contains(t, recorder.Body.String(), "response.created")
}

func TestClaudeMessagesToResponsesStreamHandlerDrainDeadlineDoesNotEstimateUsage(t *testing.T) {
	oldStreamingTimeout := rootconstant.StreamingTimeout
	rootconstant.StreamingTimeout = 1
	t.Cleanup(func() { rootconstant.StreamingTimeout = oldStreamingTimeout })
	ctx, _ := newClaudeBridgeContext(t)
	requestCtx, cancel := context.WithCancel(ctx.Request.Context())
	t.Cleanup(cancel)
	ctx.Request = ctx.Request.WithContext(requestCtx)
	disconnected := make(chan struct{}, 1)
	ctx.Writer = &cancellingResponseWriter{ResponseWriter: ctx.Writer, cancel: cancel, event: "event: response.created", disconnected: disconnected}
	reader, writer := io.Pipe()
	t.Cleanup(func() { _ = writer.Close() })
	info := newClaudeBridgeInfo(true)
	info.SetEstimatePromptTokens(1200)
	result := make(chan struct {
		usage *dto.Usage
		err   *types.NewAPIError
	}, 1)
	go func() {
		usage, err := ClaudeMessagesToResponsesStreamHandler(ctx, info, &http.Response{StatusCode: http.StatusOK, Body: reader})
		result <- struct {
			usage *dto.Usage
			err   *types.NewAPIError
		}{usage: usage, err: err}
	}()
	_, err := io.WriteString(writer, `data: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-4-6"}}`+"\n")
	require.NoError(t, err)
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("downstream cancellation was not observed")
	}
	select {
	case outcome := <-result:
		require.Nil(t, outcome.err)
		require.NotNil(t, outcome.usage)
		assert.Zero(t, outcome.usage.PromptTokens)
		assert.Zero(t, outcome.usage.CompletionTokens)
		assert.False(t, common.GetContextKeyBool(ctx, rootconstant.ContextKeyLocalCountTokens))
		assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
	case <-time.After(3 * time.Second):
		t.Fatal("usage drain did not stop at its streaming deadline")
	}
}

func TestClaudeResponsesUpstreamErrorsRemainErrors(t *testing.T) {
	for _, tc := range []struct {
		name, event string
		wantCode    string
	}{
		{name: "上游限流", event: `data: {"type":"error","error":{"type":"rate_limit_error","message":"rate limited"}}`, wantCode: "rate_limit_error"},
		{name: "上游无效事件", event: `data: {`, wantCode: string(types.ErrorCodeBadResponseBody)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := newClaudeBridgeContext(t)
			body := `data: {"type":"message_start","message":{"usage":{"input_tokens":7,"output_tokens":0}}}` + "\n" + tc.event + "\n"
			usage, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, newClaudeBridgeInfo(true), &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			})
			require.NotNil(t, apiErr)
			assert.Equal(t, tc.wantCode, string(apiErr.GetErrorCode()))
			assert.Nil(t, usage, "真正上游错误不能伪装成下游取消成功结算")
		})
	}
}

func TestClaudeResponsesCancellationAfterCompletionKeepsCompletedUsage(t *testing.T) {
	ctx, recorder := newClaudeBridgeContext(t)
	requestCtx, cancel := context.WithCancel(ctx.Request.Context())
	t.Cleanup(cancel)
	ctx.Request = ctx.Request.WithContext(requestCtx)
	info := newClaudeBridgeInfo(true)
	body := strings.Join([]string{
		`data: {"type":"message_start","message":{"id":"msg_1","role":"assistant","usage":{"input_tokens":7,"output_tokens":0}}}`,
		`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`,
		`data: {"type":"message_stop"}`,
		"",
	}, "\n")
	usage, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, info, &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))})
	require.Nil(t, apiErr)
	require.Contains(t, recorder.Body.String(), "event: response.completed")
	cancel()
	assert.Equal(t, 7, usage.PromptTokens)
	assert.Equal(t, 3, usage.CompletionTokens)
	assert.True(t, info.StreamStatus.IsNormalEnd())
	assert.False(t, info.StreamStatus.HasErrors())
}

func TestClaudeResponsesUpstreamReadFailureSurvivesClientCancellation(t *testing.T) {
	ctx, _ := newClaudeBridgeContext(t)
	requestCtx, cancel := context.WithCancel(ctx.Request.Context())
	t.Cleanup(cancel)
	ctx.Request = ctx.Request.WithContext(requestCtx)
	info := newClaudeBridgeInfo(true)
	_, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, info, &http.Response{
		StatusCode: http.StatusOK,
		Body:       &failingUpstreamResponseBody{cancel: cancel},
	})
	require.Nil(t, apiErr)
	require.ErrorIs(t, requestCtx.Err(), context.Canceled)
	assert.Equal(t, relaycommon.StreamEndReasonScannerErr, info.StreamStatus.EndReason)
	assert.ErrorIs(t, info.StreamStatus.EndError, io.ErrUnexpectedEOF)
}

func newClaudeBridgeContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	c.Set(common.RequestIdKey, "claude-bridge-test")
	return c, recorder
}

func newClaudeBridgeInfo(stream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		IsStream:        stream,
		RelayMode:       relayconstant.RelayModeResponses,
		RelayFormat:     types.RelayFormatOpenAIResponses,
		RequestURLPath:  "/v1/responses",
		DisablePing:     true,
		OriginModelName: "claude-sonnet-4-6",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "claude-sonnet-4-6",
		},
	}
}

func mustClaudeBridgeRaw(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := common.Marshal(value)
	require.NoError(t, err)
	return raw
}

func requireOrderedClaudeBridgeEvents(t *testing.T, body string, events ...string) {
	t.Helper()
	offset := 0
	for _, event := range events {
		index := strings.Index(body[offset:], event)
		require.NotEqualf(t, -1, index, "missing %q after byte %d", event, offset)
		offset += index + len(event)
	}
}

func TestClaudeResponsesHTTPDisconnectDrainsUsageUntilMessageStop(t *testing.T) {
	service.InitHttpClient()
	ctx, recorder := newClaudeBridgeContext(t)
	requestContext, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()
	ctx.Request = ctx.Request.WithContext(requestContext)
	disconnected := make(chan struct{}, 1)
	ctx.Writer = &cancellingResponseWriter{ResponseWriter: ctx.Writer, cancel: cancel, event: "event: response.created", disconnected: disconnected}
	releaseFinalUsage := make(chan struct{})
	upstreamClosed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, `data: {"type":"message_start","message":{"id":"msg_1","role":"assistant","usage":{"input_tokens":7,"cache_read_input_tokens":4,"output_tokens":0}}}`+"\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-releaseFinalUsage:
			_, _ = io.WriteString(w, `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`+"\n\n"+`data: {"type":"message_stop"}`+"\n\n")
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			close(upstreamClosed)
			return
		}
		// 终态事件后故意不发送 EOF，网关必须主动结束读取并释放连接。
		<-r.Context().Done()
		close(upstreamClosed)
	}))
	defer func() { server.CloseClientConnections(); server.Close() }()
	info := newClaudeBridgeInfo(true)
	info.ChannelBaseUrl = server.URL
	info.ApiType = rootconstant.APITypeAnthropic
	info.SetEstimatePromptTokens(1200)
	adaptor := &Adaptor{}
	_, err := adaptor.ConvertOpenAIResponsesRequest(ctx, info, dto.OpenAIResponsesRequest{Model: "claude-sonnet-4-6", Input: []byte(`"hi"`)})
	require.NoError(t, err)
	response, err := adaptor.DoRequest(ctx, info, strings.NewReader(`{}`))
	require.NoError(t, err)
	resp, ok := response.(*http.Response)
	require.True(t, ok)
	defer resp.Body.Close()
	result := make(chan struct {
		usage *dto.Usage
		err   *types.NewAPIError
	}, 1)
	go func() {
		usage, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, info, resp)
		result <- struct {
			usage *dto.Usage
			err   *types.NewAPIError
		}{usage, apiErr}
	}()
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("downstream did not disconnect")
	}
	close(releaseFinalUsage)
	select {
	case outcome := <-result:
		require.Nil(t, outcome.err)
		require.NotNil(t, outcome.usage)
		assert.Equal(t, 7, outcome.usage.PromptTokens)
		assert.Equal(t, 3, outcome.usage.CompletionTokens)
		assert.Equal(t, 4, outcome.usage.PromptTokensDetails.CachedTokens)
		assert.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
		assert.False(t, common.GetContextKeyBool(ctx, rootconstant.ContextKeyLocalCountTokens))
	case <-time.After(2 * time.Second):
		t.Fatal("message_stop did not finish bounded usage drain")
	}
	select {
	case <-upstreamClosed:
	case <-time.After(2 * time.Second):
		t.Fatal("completed usage drain did not release upstream connection")
	}
	assert.NotContains(t, recorder.Body.String(), "response.completed")
}

func TestClaudeResponsesUpstreamErrorAfterDisconnectRemainsError(t *testing.T) {
	ctx, _ := newClaudeBridgeContext(t)
	requestContext, cancel := context.WithCancel(ctx.Request.Context())
	defer cancel()
	ctx.Request = ctx.Request.WithContext(requestContext)
	disconnected := make(chan struct{}, 1)
	ctx.Writer = &cancellingResponseWriter{ResponseWriter: ctx.Writer, cancel: cancel, event: "event: response.created", disconnected: disconnected}
	reader, writer := io.Pipe()
	defer writer.Close()
	info := newClaudeBridgeInfo(true)
	result := make(chan *types.NewAPIError, 1)
	go func() {
		_, apiErr := ClaudeMessagesToResponsesStreamHandler(ctx, info, &http.Response{StatusCode: http.StatusOK, Body: reader})
		result <- apiErr
	}()
	_, err := io.WriteString(writer, `data: {"type":"message_start","message":{"id":"msg_1","role":"assistant","usage":{"input_tokens":7}}}`+"\n")
	require.NoError(t, err)
	select {
	case <-disconnected:
	case <-time.After(2 * time.Second):
		t.Fatal("downstream did not disconnect")
	}
	_, err = io.WriteString(writer, `data: {"type":"error","error":{"type":"overloaded_error","message":"upstream overloaded"}}`+"\n")
	require.NoError(t, err)
	select {
	case apiErr := <-result:
		require.NotNil(t, apiErr)
		assert.Equal(t, "overloaded_error", string(apiErr.GetErrorCode()))
		assert.True(t, info.StreamStatus.HasErrors())
	case <-time.After(2 * time.Second):
		t.Fatal("real upstream error was swallowed during usage drain")
	}
}
