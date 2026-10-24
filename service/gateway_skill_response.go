package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	// GatewaySkillsResponseHeader 是网关实际装载技能的响应元数据头。
	GatewaySkillsResponseHeader = "X-New-API-Gateway-Skills"
	// GatewaySkillsDisplayHeader 允许客户端只显示元数据而不修改回复正文。
	GatewaySkillsDisplayHeader = "X-New-API-Skills-Display"

	gatewaySkillResponseHeaderLimit = 6 * 1024
	gatewaySkillMaxJSONBodyBytes    = 16 * 1024 * 1024
	gatewaySkillMaxSSEEventBytes    = 256 * 1024
)

const gatewaySkillFooterPrefix = "\n\n---\n☁ 网关云端 Skills · "

// GatewaySkillResponseUsageFromContext 返回当前请求的实际技能快照。
// 调用方只能得到没有正文的结构化元数据，适合响应头和客户端展示。
func GatewaySkillResponseUsageFromContext(c *gin.Context) *GatewaySkillResponseUsage {
	if c == nil {
		return nil
	}
	value, ok := c.Get(gatewaySkillResponseUsageKey)
	if !ok {
		return nil
	}
	usage, ok := value.(*GatewaySkillResponseUsage)
	if !ok || usage == nil || len(usage.Skills) == 0 {
		return nil
	}
	return usage
}

// ResetGatewaySkillResponse 清除本次尝试的技能响应信息。
// auto 重试换组时必须先调用，避免把上一组读取过的技能泄漏到最终回复。
func ResetGatewaySkillResponse(c *gin.Context) {
	if c == nil {
		return
	}
	c.Set(gatewaySkillResponseUsageKey, nil)
	if c.Writer != nil {
		c.Writer.Header().Del(GatewaySkillsResponseHeader)
	}
}

// BeginGatewaySkillResponse 在正式 relay 前安装响应封装。
// metadata 模式仅设置响应头，返回 nil；默认模式在有文本的成功回复中追加尾部标识。
func BeginGatewaySkillResponse(c *gin.Context, format types.RelayFormat) *GatewaySkillResponseWriter {
	usage := GatewaySkillResponseUsageFromContext(c)
	if c == nil || c.Writer == nil || usage == nil {
		return nil
	}
	if encoded := encodeGatewaySkillUsageHeader(usage); encoded != "" {
		c.Writer.Header().Set(GatewaySkillsResponseHeader, encoded)
	}
	if strings.EqualFold(strings.TrimSpace(c.GetHeader(GatewaySkillsDisplayHeader)), "metadata") {
		return nil
	}
	writer := &GatewaySkillResponseWriter{
		Context:        c,
		ResponseWriter: c.Writer,
		format:         format,
		footer:         gatewaySkillFooter(usage),
		status:         http.StatusOK,
		mode:           gatewaySkillResponseModeUnknown,
		visible:        make(map[int]bool),
		maxOutputIndex: -1,
	}
	c.Writer = writer
	return writer
}

func encodeGatewaySkillUsageHeader(usage *GatewaySkillResponseUsage) string {
	if usage == nil || len(usage.Skills) == 0 {
		return ""
	}
	all := make([]GatewaySkillResponseSkill, len(usage.Skills))
	copy(all, usage.Skills)
	for count := len(all); count >= 0; count-- {
		candidate := &GatewaySkillResponseUsage{Version: 1, Total: len(all), Skills: all[:count], Truncated: count != len(all)}
		encodedJSON, err := common.Marshal(candidate)
		if err != nil {
			return ""
		}
		encoded := base64.RawURLEncoding.EncodeToString(encodedJSON)
		if len(encoded) <= gatewaySkillResponseHeaderLimit {
			return encoded
		}
	}
	return ""
}

func gatewaySkillFooter(usage *GatewaySkillResponseUsage) string {
	if usage == nil || len(usage.Skills) == 0 {
		return ""
	}
	names := make([]string, 0, len(usage.Skills))
	for _, skill := range usage.Skills {
		title := safeGatewaySkillTitle(skill.Title)
		if title == "" {
			title = safeGatewaySkillTitle(skill.SkillID)
		}
		if title != "" {
			names = append(names, title)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return gatewaySkillFooterPrefix + strings.Join(names, "、")
}

func safeGatewaySkillTitle(title string) string {
	var builder strings.Builder
	for _, r := range title {
		if unicode.IsControl(r) {
			if r == '\n' || r == '\r' || r == '\t' {
				builder.WriteByte(' ')
			}
			continue
		}
		builder.WriteRune(r)
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

type gatewaySkillResponseMode uint8

const (
	gatewaySkillResponseModeUnknown gatewaySkillResponseMode = iota
	gatewaySkillResponseModeJSON
	gatewaySkillResponseModeSSE
	gatewaySkillResponseModePassThrough
)

// GatewaySkillResponseWriter 延迟非流式 JSON，流式响应按 SSE 事件增量处理。
// 它不改写请求用量或上游原始工具、思考和未知字段。
type GatewaySkillResponseWriter struct {
	gin.ResponseWriter
	Context     *gin.Context
	format      types.RelayFormat
	footer      string
	status      int
	mode        gatewaySkillResponseMode
	logicalSize int
	written     bool
	finished    bool

	jsonBody     bytes.Buffer
	jsonDisabled bool

	ssePending         bytes.Buffer
	sseDeferred        []string
	sseTerminal        bool
	sseRewriteDisabled bool
	sseSuppressFooter  bool
	claudeStopReason   string
	visible            map[int]bool
	maxOutputIndex     int
	sequenceSeen       bool
	lastSequence       int64
}

var _ gin.ResponseWriter = (*GatewaySkillResponseWriter)(nil)

func (w *GatewaySkillResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *GatewaySkillResponseWriter) WriteHeader(status int) {
	if status <= 0 || w.written {
		return
	}
	w.status = status
}

func (w *GatewaySkillResponseWriter) WriteHeaderNow() {
	if w.written {
		return
	}
	w.written = true
	if w.status <= 0 {
		w.status = http.StatusOK
	}
}

func (w *GatewaySkillResponseWriter) Write(data []byte) (int, error) {
	if w.finished {
		return w.ResponseWriter.Write(data)
	}
	w.WriteHeaderNow()
	w.logicalSize += len(data)
	if w.mode == gatewaySkillResponseModeUnknown {
		if looksLikeGatewaySSE(data) {
			w.mode = gatewaySkillResponseModeSSE
		} else {
			w.mode = gatewaySkillResponseModeJSON
		}
	}
	switch w.mode {
	case gatewaySkillResponseModeJSON:
		if !w.jsonDisabled && w.jsonBody.Len()+len(data) <= gatewaySkillMaxJSONBodyBytes {
			_, _ = w.jsonBody.Write(data)
			return len(data), nil
		}
		if !w.jsonDisabled {
			w.jsonDisabled = true
			if err := w.writeUnderlying(w.jsonBody.Bytes()); err != nil {
				return 0, err
			}
			w.jsonBody.Reset()
		}
		return len(data), w.writeUnderlying(data)
	case gatewaySkillResponseModeSSE:
		if w.sseRewriteDisabled {
			return len(data), w.writeUnderlying(data)
		}
		if _, err := w.ssePending.Write(data); err != nil {
			return 0, err
		}
		if w.ssePending.Len() > gatewaySkillMaxSSEEventBytes && !bytes.Contains(w.ssePending.Bytes(), []byte("\n\n")) && !bytes.Contains(w.ssePending.Bytes(), []byte("\r\n\r\n")) {
			w.sseRewriteDisabled = true
			pending := append([]byte(nil), w.ssePending.Bytes()...)
			w.ssePending.Reset()
			return len(data), w.writeUnderlying(pending)
		}
		return len(data), w.consumeSSEFrames(false)
	case gatewaySkillResponseModePassThrough:
		return len(data), w.writeUnderlying(data)
	default:
		return len(data), nil
	}
}

func (w *GatewaySkillResponseWriter) WriteString(value string) (int, error) {
	return w.Write([]byte(value))
}

func (w *GatewaySkillResponseWriter) Flush() {
	if w.finished {
		w.ResponseWriter.Flush()
		return
	}
	if w.mode == gatewaySkillResponseModeSSE {
		_ = w.consumeSSEFrames(false)
		if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}
	if w.mode == gatewaySkillResponseModePassThrough {
		if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

func (w *GatewaySkillResponseWriter) Status() int { return w.status }

func (w *GatewaySkillResponseWriter) Size() int { return w.logicalSize }

func (w *GatewaySkillResponseWriter) Written() bool { return w.written }

// Finish 提交封装结果并恢复 gin 原始 writer。调用失败或正文不完整时保留原始输出。
func (w *GatewaySkillResponseWriter) Finish(success bool) {
	if w == nil || w.finished {
		return
	}
	w.finished = true
	defer func() {
		if w.Context != nil {
			w.Context.Writer = w.ResponseWriter
		}
	}()
	if w.status <= 0 {
		w.status = http.StatusOK
	}
	// 非成功轮次不能提交封装层尚未转发的正文，否则自动重试时会把上一组
	// 的半成品输出混入下一组响应。已经因流式低延迟而透传到底层的字节无法
	// 撤回，这也是流式失败轮次只能做到“尽量不新增泄漏”的边界。
	if !success {
		w.jsonBody.Reset()
		w.ssePending.Reset()
		w.sseDeferred = nil
		return
	}
	switch w.mode {
	case gatewaySkillResponseModeJSON:
		if !w.jsonDisabled {
			body := append([]byte(nil), w.jsonBody.Bytes()...)
			if w.status >= http.StatusOK && w.status < http.StatusMultipleChoices {
				if modified, ok := appendGatewaySkillFooterJSON(body, w.format, w.footer); ok {
					body = modified
					w.Header().Del("Content-Length")
				}
			}
			_ = w.writeUnderlying(body)
		}
	case gatewaySkillResponseModeSSE:
		_ = w.consumeSSEFrames(true)
		if w.sseRewriteDisabled {
			if w.ssePending.Len() > 0 {
				_ = w.writeUnderlying(w.ssePending.Bytes())
				w.ssePending.Reset()
			}
			for _, frame := range w.sseDeferred {
				_ = w.writeUnderlying([]byte(frame))
			}
		} else {
			w.finishSSESuccess()
		}
	case gatewaySkillResponseModePassThrough:
		// 内容已经直接写入底层。
	case gatewaySkillResponseModeUnknown:
		_ = w.writeUnderlying(nil)
	}
}

func (w *GatewaySkillResponseWriter) writeUnderlying(data []byte) error {
	if w.status <= 0 {
		w.status = http.StatusOK
	}
	if !w.ResponseWriter.Written() {
		w.ResponseWriter.WriteHeader(w.status)
	}
	if len(data) == 0 {
		return nil
	}
	_, err := w.ResponseWriter.Write(data)
	return err
}

func looksLikeGatewaySSE(data []byte) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return false
	}
	first := trimmed
	if index := bytes.IndexByte(first, '\n'); index >= 0 {
		first = first[:index]
	}
	first = bytes.TrimSpace(first)
	return bytes.HasPrefix(first, []byte("event:")) || bytes.HasPrefix(first, []byte("data:")) || bytes.HasPrefix(first, []byte(":"))
}

func (w *GatewaySkillResponseWriter) consumeSSEFrames(final bool) error {
	for {
		data := w.ssePending.Bytes()
		end, delimiter := gatewaySSEFrameEnd(data)
		if end < 0 {
			break
		}
		frame := string(data[:end+delimiter])
		w.ssePending.Next(end + delimiter)
		if len(frame) > gatewaySkillMaxSSEEventBytes {
			w.sseRewriteDisabled = true
			if err := w.writeUnderlying([]byte(frame)); err != nil {
				return err
			}
			continue
		}
		if w.sseRewriteDisabled || w.sseTerminal {
			w.sseDeferred = append(w.sseDeferred, frame)
			continue
		}
		terminal, err := w.observeSSEFrame(frame)
		if err != nil {
			// 无法识别的事件保持原样，不能因展示标识破坏协议。
			if err := w.writeUnderlying([]byte(frame)); err != nil {
				return err
			}
			continue
		}
		if terminal {
			w.sseTerminal = true
			w.sseDeferred = append(w.sseDeferred, frame)
			continue
		}
		if err := w.writeUnderlying([]byte(frame)); err != nil {
			return err
		}
	}
	if final && w.ssePending.Len() > 0 {
		// 未闭合事件不满足协议完成条件，按原样透传。
		if err := w.writeUnderlying(w.ssePending.Bytes()); err != nil {
			return err
		}
		w.ssePending.Reset()
	}
	return nil
}

func gatewaySSEFrameEnd(data []byte) (int, int) {
	lf := bytes.Index(data, []byte("\n\n"))
	crlf := bytes.Index(data, []byte("\r\n\r\n"))
	if lf < 0 && crlf < 0 {
		return -1, 0
	}
	if lf >= 0 && (crlf < 0 || lf < crlf) {
		return lf, 2
	}
	return crlf, 4
}

type gatewaySSEFrameData struct {
	event string
	data  string
}

func parseGatewaySSEFrame(frame string) gatewaySSEFrameData {
	clean := strings.ReplaceAll(frame, "\r\n", "\n")
	clean = strings.TrimSuffix(clean, "\n\n")
	var parsed gatewaySSEFrameData
	for _, line := range strings.Split(clean, "\n") {
		switch {
		case strings.HasPrefix(line, "event:"):
			parsed.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if parsed.data != "" {
				parsed.data += "\n"
			}
			parsed.data += strings.TrimPrefix(line, "data:")
		}
	}
	parsed.data = strings.TrimPrefix(parsed.data, " ")
	return parsed
}

func gatewaySSEFrameWithData(frame string, payload []byte) string {
	clean := strings.ReplaceAll(frame, "\r\n", "\n")
	clean = strings.TrimSuffix(clean, "\n\n")
	lines := strings.Split(clean, "\n")
	dataWritten := false
	for i, line := range lines {
		if strings.HasPrefix(line, "data:") {
			if !dataWritten {
				lines[i] = "data: " + string(payload)
				dataWritten = true
			} else {
				lines[i] = ""
			}
		}
	}
	if !dataWritten {
		lines = append(lines, "data: "+string(payload))
	}
	return strings.Join(lines, "\n") + "\n\n"
}

func (w *GatewaySkillResponseWriter) observeSSEFrame(frame string) (bool, error) {
	parsed := parseGatewaySSEFrame(frame)
	if parsed.data == "" {
		return false, nil
	}
	if parsed.data == "[DONE]" {
		return w.sseTerminal, nil
	}
	if !gjson.Valid(parsed.data) {
		return false, fmt.Errorf("invalid SSE JSON")
	}
	data := []byte(parsed.data)
	if sequence := gjson.GetBytes(data, "sequence_number"); sequence.Type == gjson.Number {
		w.sequenceSeen = true
		if sequence.Int() > w.lastSequence {
			w.lastSequence = sequence.Int()
		}
	}
	switch w.format {
	case types.RelayFormatOpenAI:
		for _, choice := range gjson.GetBytes(data, "choices").Array() {
			index := int(choice.Get("index").Int())
			if choice.Get("delta.content").Type == gjson.String && choice.Get("delta.content").String() != "" {
				w.visible[index] = true
			}
			if reason := choice.Get("finish_reason"); reason.Type == gjson.String && reason.String() != "" {
				// 工具调用、内容过滤等终止原因也要停止继续等待，最终
				// 是否追加尾标由 appendGatewaySkillChatStream 再按语义判断。
				return true, nil
			}
		}
		return false, nil
	case types.RelayFormatOpenAIResponses:
		if outputIndex := gjson.GetBytes(data, "output_index"); outputIndex.Type == gjson.Number && int(outputIndex.Int()) > w.maxOutputIndex {
			w.maxOutputIndex = int(outputIndex.Int())
		}
		if parsed.event == "response.output_text.delta" || gjson.GetBytes(data, "type").String() == "response.output_text.delta" {
			if gjson.GetBytes(data, "delta").Type == gjson.String && gjson.GetBytes(data, "delta").String() != "" {
				outputIndex := 0
				if value := gjson.GetBytes(data, "output_index"); value.Type == gjson.Number {
					outputIndex = int(value.Int())
				}
				w.visible[outputIndex] = true
			}
		}
		if parsed.event == "response.completed" || parsed.event == "response.done" ||
			gjson.GetBytes(data, "type").String() == "response.completed" || gjson.GetBytes(data, "type").String() == "response.done" {
			return gjson.GetBytes(data, "response.status").String() == "completed", nil
		}
	case types.RelayFormatClaude:
		if parsed.event == "content_block_delta" || gjson.GetBytes(data, "type").String() == "content_block_delta" {
			if gjson.GetBytes(data, "delta.type").String() == "text_delta" && gjson.GetBytes(data, "delta.text").String() != "" {
				w.visible[int(gjson.GetBytes(data, "index").Int())] = true
			}
		}
		if parsed.event == "message_delta" || gjson.GetBytes(data, "type").String() == "message_delta" {
			stop := gjson.GetBytes(data, "delta.stop_reason")
			if stop.Type == gjson.String && stop.String() != "" {
				w.claudeStopReason = stop.String()
				if stop.String() == "tool_use" {
					w.sseSuppressFooter = true
				}
				return true, nil
			}
		}
		if parsed.event == "message_stop" || gjson.GetBytes(data, "type").String() == "message_stop" {
			// message_stop 没有 stop_reason，无法证明这是可展示的助手文本
			// 完成事件；保守地保留原始事件，不伪造技能调用尾标。
			if w.claudeStopReason == "" {
				w.sseSuppressFooter = true
			}
			return true, nil
		}
	case types.RelayFormatGemini:
		for i, candidate := range gjson.GetBytes(data, "candidates").Array() {
			for _, part := range candidate.Get("content.parts").Array() {
				if part.Get("text").Type == gjson.String && part.Get("text").String() != "" && !part.Get("thought").Bool() {
					w.visible[i] = true
				}
			}
			reason := candidate.Get("finishReason").String()
			if reason == "STOP" || reason == "MAX_TOKENS" {
				return true, nil
			}
		}
	}
	return false, nil
}

func (w *GatewaySkillResponseWriter) finishSSESuccess() {
	if w.ssePending.Len() > 0 {
		_ = w.writeUnderlying(w.ssePending.Bytes())
		w.ssePending.Reset()
	}
	if len(w.sseDeferred) == 0 {
		return
	}
	frames := w.sseDeferred
	if w.sseTerminal && !w.sseRewriteDisabled {
		first := frames[0]
		transformed := w.transformTerminalSSE(first)
		for _, frame := range transformed {
			_ = w.writeUnderlying([]byte(frame))
		}
		for _, frame := range frames[1:] {
			_ = w.writeUnderlying([]byte(frame))
		}
		return
	}
	for _, frame := range frames {
		_ = w.writeUnderlying([]byte(frame))
	}
}

func (w *GatewaySkillResponseWriter) transformTerminalSSE(frame string) []string {
	parsed := parseGatewaySSEFrame(frame)
	if parsed.data == "" || parsed.data == "[DONE]" || w.footer == "" {
		return []string{frame}
	}
	data := []byte(parsed.data)
	switch w.format {
	case types.RelayFormatOpenAI:
		modified, ok := appendGatewaySkillChatStream(data, w.footer, w.visible)
		if ok {
			return []string{gatewaySSEFrameWithData(frame, modified)}
		}
	case types.RelayFormatOpenAIResponses:
		return w.transformResponsesTerminalSSE(frame, data)
	case types.RelayFormatClaude:
		if len(w.visible) == 0 || w.sseSuppressFooter || !gatewaySkillClaudeFooterStopReason(w.claudeStopReason) {
			return []string{frame}
		}
		index := 0
		for candidate := range w.visible {
			if candidate >= index {
				index = candidate + 1
			}
		}
		return []string{
			gatewaySSEFrame("content_block_start", map[string]any{"type": "content_block_start", "index": index, "content_block": map[string]any{"type": "text", "text": ""}}),
			gatewaySSEFrame("content_block_delta", map[string]any{"type": "content_block_delta", "index": index, "delta": map[string]any{"type": "text_delta", "text": w.footer}}),
			gatewaySSEFrame("content_block_stop", map[string]any{"type": "content_block_stop", "index": index}),
			frame,
		}
	case types.RelayFormatGemini:
		modified, ok := appendGatewaySkillGeminiStream(data, w.footer, w.visible)
		if ok {
			return []string{gatewaySSEFrameWithData(frame, modified)}
		}
	}
	return []string{frame}
}

func (w *GatewaySkillResponseWriter) transformResponsesTerminalSSE(frame string, data []byte) []string {
	if gjson.GetBytes(data, "response.status").String() != "completed" || len(w.visible) == 0 {
		return []string{frame}
	}
	response := gjson.GetBytes(data, "response")
	if !response.IsObject() {
		return []string{frame}
	}
	output := make([]json.RawMessage, 0)
	if raw := response.Get("output"); raw.IsArray() {
		if err := common.Unmarshal([]byte(raw.Raw), &output); err != nil {
			return []string{frame}
		}
	}
	maxIndex := w.maxOutputIndex
	for _, item := range output {
		if index := int(gjson.GetBytes(item, "index").Int()); index > maxIndex {
			maxIndex = index
		}
	}
	if len(output)-1 > maxIndex {
		maxIndex = len(output) - 1
	}
	outputIndex := maxIndex + 1
	responseID := response.Get("id").String()
	if responseID == "" {
		responseID = "response"
	}
	itemID := "msg_gateway_skills_" + safeGatewayResponseID(responseID)
	part := map[string]any{"type": "output_text", "text": w.footer, "annotations": []any{}}
	item := map[string]any{"id": itemID, "type": "message", "role": "assistant", "status": "completed", "content": []any{part}}
	itemRaw, err := common.Marshal(item)
	if err != nil {
		return []string{frame}
	}
	output = append(output, json.RawMessage(itemRaw))
	updated, err := sjson.SetRawBytes(data, "response.output", mustGatewaySkillMarshal(output))
	if err != nil {
		return []string{frame}
	}

	sequence := int64(0)
	if original := gjson.GetBytes(data, "sequence_number"); original.Type == gjson.Number {
		sequence = original.Int()
	} else if w.sequenceSeen {
		sequence = w.lastSequence + 1
	}
	sequenceEnabled := w.sequenceSeen || gjson.GetBytes(data, "sequence_number").Type == gjson.Number
	// response.completed 已经占用了当前序号。把新增的六个事件插在该
	// 序号之前，并将原完成事件顺延到最后，确保客户端看到严格递增的
	// sequence_number（此前从 lastSequence 反推会把新增事件放到完成事件
	// 之后，导致回放器拒绝整段流）。
	next := sequence
	frames := make([]string, 0, 7)
	addEvent := func(event string, value map[string]any) {
		if sequenceEnabled {
			value["sequence_number"] = next
			next++
		}
		frames = append(frames, gatewaySSEFrame(event, value))
	}
	addEvent("response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": outputIndex, "item": map[string]any{"id": itemID, "type": "message", "role": "assistant", "status": "in_progress", "content": []any{}}})
	addEvent("response.content_part.added", map[string]any{"type": "response.content_part.added", "item_id": itemID, "output_index": outputIndex, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}})
	addEvent("response.output_text.delta", map[string]any{"type": "response.output_text.delta", "item_id": itemID, "output_index": outputIndex, "content_index": 0, "delta": w.footer})
	addEvent("response.output_text.done", map[string]any{"type": "response.output_text.done", "item_id": itemID, "output_index": outputIndex, "content_index": 0, "text": w.footer})
	addEvent("response.content_part.done", map[string]any{"type": "response.content_part.done", "item_id": itemID, "output_index": outputIndex, "content_index": 0, "part": part})
	addEvent("response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": outputIndex, "item": item})
	if sequenceEnabled {
		sequence = next
		w.lastSequence = sequence
		updated, err = sjson.SetRawBytes(updated, "sequence_number", mustGatewaySkillMarshal(sequence))
		if err != nil {
			return []string{frame}
		}
	}
	frames = append(frames, gatewaySSEFrameWithData(frame, updated))
	return frames
}

func safeGatewayResponseID(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "response"
	}
	return builder.String()
}

func gatewaySkillClaudeFooterStopReason(reason string) bool {
	switch reason {
	case "end_turn", "max_tokens", "stop_sequence":
		return true
	default:
		return false
	}
}

func mustGatewaySkillMarshal(value any) []byte {
	encoded, err := common.Marshal(value)
	if err != nil {
		return []byte("null")
	}
	return encoded
}

func gatewaySSEFrame(event string, value any) string {
	encoded, err := common.Marshal(value)
	if err != nil {
		return ""
	}
	return "event: " + event + "\ndata: " + string(encoded) + "\n\n"
}

// appendGatewaySkillFooterJSON 仅修改有可见助手文本且明确成功的 JSON 响应。
func appendGatewaySkillFooterJSON(body []byte, format types.RelayFormat, footer string) ([]byte, bool) {
	if footer == "" || !gjson.ValidBytes(body) || gjson.GetBytes(body, "error").Exists() {
		return body, false
	}
	switch format {
	case types.RelayFormatOpenAI:
		return appendGatewaySkillChatJSON(body, footer)
	case types.RelayFormatOpenAIResponses:
		return appendGatewaySkillResponsesJSON(body, footer)
	case types.RelayFormatClaude:
		return appendGatewaySkillClaudeJSON(body, footer)
	case types.RelayFormatGemini:
		return appendGatewaySkillGeminiJSON(body, footer)
	default:
		return body, false
	}
}

func appendGatewaySkillChatJSON(body []byte, footer string) ([]byte, bool) {
	choices := gjson.GetBytes(body, "choices").Array()
	modified := false
	for index, choice := range choices {
		reason := choice.Get("finish_reason").String()
		if reason == "" || reason == "content_filter" || reason == "tool_calls" || reason == "function_call" {
			continue
		}
		message := choice.Get("message")
		if message.Get("role").String() != "assistant" {
			continue
		}
		content := message.Get("content")
		path := fmt.Sprintf("choices.%d.message.content", index)
		if content.Type == gjson.String && content.String() != "" {
			var err error
			body, err = setGatewaySkillJSONValue(body, path, content.String()+footer)
			if err != nil {
				return body, modified
			}
			modified = true
			continue
		}
		if content.IsArray() {
			parts := make([]json.RawMessage, 0)
			if err := common.Unmarshal([]byte(content.Raw), &parts); err != nil {
				continue
			}
			hasText := false
			for _, part := range parts {
				if gjson.GetBytes(part, "type").String() == "text" && gjson.GetBytes(part, "text").String() != "" {
					hasText = true
				}
			}
			if !hasText {
				continue
			}
			encoded, err := common.Marshal(append(parts, gatewaySkillTextPart(footer)))
			if err != nil {
				continue
			}
			body, err = sjson.SetRawBytes(body, path, encoded)
			if err == nil {
				modified = true
			}
		}
	}
	return body, modified
}

func appendGatewaySkillChatStream(body []byte, footer string, visible map[int]bool) ([]byte, bool) {
	choices := gjson.GetBytes(body, "choices").Array()
	modified := false
	for index, choice := range choices {
		reason := choice.Get("finish_reason").String()
		if !visible[int(choice.Get("index").Int())] || reason == "" || reason == "content_filter" || reason == "tool_calls" || reason == "function_call" {
			continue
		}
		path := fmt.Sprintf("choices.%d.delta.content", index)
		content := choice.Get("delta.content")
		value := footer
		if content.Type == gjson.String {
			value = content.String() + footer
		}
		var err error
		body, err = setGatewaySkillJSONValue(body, path, value)
		if err == nil {
			modified = true
		}
	}
	return body, modified
}

func appendGatewaySkillResponsesJSON(body []byte, footer string) ([]byte, bool) {
	if gjson.GetBytes(body, "status").String() != "completed" {
		return body, false
	}
	output := gjson.GetBytes(body, "output").Array()
	modified := false
	for outputIndex, item := range output {
		if item.Get("type").String() != "message" || item.Get("role").String() != "assistant" {
			continue
		}
		content := item.Get("content")
		if !content.IsArray() {
			continue
		}
		parts := make([]json.RawMessage, 0)
		if err := common.Unmarshal([]byte(content.Raw), &parts); err != nil {
			continue
		}
		hasText := false
		for _, part := range parts {
			if gjson.GetBytes(part, "type").String() == "output_text" && gjson.GetBytes(part, "text").String() != "" {
				hasText = true
			}
		}
		if !hasText {
			continue
		}
		encoded, err := common.Marshal(append(parts, gatewaySkillOutputTextPart(footer)))
		if err != nil {
			continue
		}
		path := fmt.Sprintf("output.%d.content", outputIndex)
		body, err = sjson.SetRawBytes(body, path, encoded)
		if err == nil {
			modified = true
		}
	}
	return body, modified
}

func appendGatewaySkillClaudeJSON(body []byte, footer string) ([]byte, bool) {
	if gjson.GetBytes(body, "type").String() != "message" || gjson.GetBytes(body, "role").String() != "assistant" {
		return body, false
	}
	stop := gjson.GetBytes(body, "stop_reason")
	if stop.Type != gjson.String || !gatewaySkillClaudeFooterStopReason(stop.String()) {
		return body, false
	}
	content := gjson.GetBytes(body, "content")
	if !content.IsArray() {
		return body, false
	}
	blocks := make([]json.RawMessage, 0)
	if err := common.Unmarshal([]byte(content.Raw), &blocks); err != nil {
		return body, false
	}
	hasText := false
	for _, block := range blocks {
		if gjson.GetBytes(block, "type").String() == "text" && gjson.GetBytes(block, "text").String() != "" {
			hasText = true
		}
	}
	if !hasText {
		return body, false
	}
	encoded, err := common.Marshal(append(blocks, gatewaySkillTextPart(footer)))
	if err != nil {
		return body, false
	}
	body, err = sjson.SetRawBytes(body, "content", encoded)
	return body, err == nil
}

func appendGatewaySkillGeminiJSON(body []byte, footer string) ([]byte, bool) {
	return appendGatewaySkillGeminiStream(body, footer, nil)
}

func appendGatewaySkillGeminiStream(body []byte, footer string, visible map[int]bool) ([]byte, bool) {
	candidates := gjson.GetBytes(body, "candidates").Array()
	modified := false
	for index, candidate := range candidates {
		reason := candidate.Get("finishReason").String()
		if (reason != "STOP" && reason != "MAX_TOKENS") || (visible != nil && !visible[index]) {
			continue
		}
		partsResult := candidate.Get("content.parts")
		if visible == nil && !partsResult.IsArray() {
			continue
		}
		parts := make([]json.RawMessage, 0)
		if partsResult.IsArray() {
			if err := common.Unmarshal([]byte(partsResult.Raw), &parts); err != nil {
				continue
			}
		} else if visible == nil {
			continue
		}
		if visible == nil {
			hasText := false
			for _, part := range parts {
				if gjson.GetBytes(part, "text").String() != "" && !gjson.GetBytes(part, "thought").Bool() {
					hasText = true
				}
			}
			if !hasText {
				continue
			}
		}
		encoded, err := common.Marshal(append(parts, gatewaySkillGeminiTextPart(footer)))
		if err != nil {
			continue
		}
		path := fmt.Sprintf("candidates.%d.content.parts", index)
		body, err = sjson.SetRawBytes(body, path, encoded)
		if err == nil {
			modified = true
		}
	}
	return body, modified
}

func gatewaySkillTextPart(text string) json.RawMessage {
	encoded, _ := common.Marshal(map[string]any{"type": "text", "text": text})
	return json.RawMessage(encoded)
}

func gatewaySkillOutputTextPart(text string) json.RawMessage {
	encoded, _ := common.Marshal(map[string]any{"type": "output_text", "text": text, "annotations": []any{}})
	return json.RawMessage(encoded)
}

func gatewaySkillGeminiTextPart(text string) json.RawMessage {
	encoded, _ := common.Marshal(map[string]any{"text": text})
	return json.RawMessage(encoded)
}

func setGatewaySkillJSONValue(body []byte, path string, value any) ([]byte, error) {
	encoded, err := common.Marshal(value)
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(body, path, encoded)
}

// Ensure imports remain meaningful when a downstream adapter only uses the writer's io methods.
var _ io.Writer = (*GatewaySkillResponseWriter)(nil)
