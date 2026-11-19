package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type MCPToolResult struct {
	Content []MCPTextContent `json:"content"`
	IsError bool             `json:"isError"`
}

type MCPTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func MCPToolError(message string) MCPToolResult {
	return MCPToolResult{Content: []MCPTextContent{{Type: "text", Text: message}}, IsError: true}
}

type mcpBoundedResponse struct {
	header   http.Header
	body     bytes.Buffer
	status   int
	exceeded bool
	limit    int
}

func (w *mcpBoundedResponse) Header() http.Header { return w.header }
func (w *mcpBoundedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *mcpBoundedResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	limit := w.limit
	if limit == 0 {
		limit = MCPMaxResponseBytes
	}
	if w.exceeded || w.body.Len()+len(body) > limit {
		w.exceeded = true
		return 0, errors.New("MCP response limit exceeded")
	}
	return w.body.Write(body)
}

// ExecuteMCPTool 仅将目录路由重新交给真实 HTTP 路由器，绝不直接调用 controller。
func ExecuteMCPTool(ctx context.Context, identity *MCPIdentity, name string, raw json.RawMessage, dispatch http.Handler) (result MCPToolResult) {
	start := time.Now()
	log := model.MCPCallLog{UserID: identity.User.Id, KeyID: identity.Key.ID}
	// 工具名来自目录才可记入审计，避免攻击者把密钥塞进未知工具名。
	for _, tool := range MCPTools() {
		if tool.Name == name {
			log.ToolName = name
			break
		}
	}
	if log.ToolName == "" {
		log.ToolName = "unknown"
	}
	if err := model.DB.Create(&log).Error; err != nil {
		return MCPToolError("Cannot persist audit record")
	}
	defer func() {
		if recover() != nil {
			result = MCPToolError("Request outcome is uncertain; do not execute again")
		}
		log.Success = !result.IsError && len(result.Content) != 0
		log.DurationMS = time.Since(start).Milliseconds()
		if err := model.DB.Model(&log).Updates(map[string]any{"success": log.Success, "duration_ms": log.DurationMS, "request_id": log.RequestID}).Error; err != nil {
			common.SysError("MCP audit finalization failed")
		}
	}()
	// 审计起始持久化之后再重新验证，撤销/降权不会沿用 HTTP 入口时的快照。
	fresh, err := ValidateMCPIdentity(identity.Key.ID)
	if err != nil {
		return MCPToolError("Credential no longer authorized")
	}
	identity = fresh
	var selected *MCPTool
	for _, tool := range MCPVisibleTools(identity.User, identity.Key) {
		if tool.Name == name {
			selected = &tool
			break
		}
	}
	if selected == nil {
		return MCPToolError("Tool is unavailable or unauthorized")
	}
	arguments, path, body, err := PrepareMCPRequest(*selected, raw)
	if err != nil {
		return MCPToolError(err.Error())
	}
	// 金额仅用于展示：业务结果先完成持久化；历史回执重放也走同一展示入口。
	defer func() {
		if result.IsError || len(result.Content) != 1 {
			return
		}
		formatted, displayErr := formatMCPMoneyResponse(ctx, *selected, []byte(result.Content[0].Text))
		if displayErr != nil {
			result.Content = append(result.Content, MCPTextContent{Type: "text", Text: "业务操作已完成，但站内金额展示暂不可用。不要把原始 quota 当作余额报告；请在后台核对余额，不要重复执行写操作。"})
			return
		}
		result.Content[0].Text = string(formatted)
	}()
	log.RequestID = arguments.RequestID
	if err = model.DB.Model(&log).Update("request_id", log.RequestID).Error; err != nil {
		return MCPToolError("Cannot persist audit record")
	}
	var writeID int64
	if selected.Write {
		canonical, err := canonicalMCPJSON(body)
		if err != nil {
			return MCPToolError("Cannot normalize request")
		}
		fingerprint, _ := common.Marshal([]string{selected.Name, selected.Method, path, string(canonical)})
		record, claimed, err := model.ClaimMCPWriteRequest(identity.Key.ID, arguments.RequestID, MCPKeyDigest(string(fingerprint)))
		if err != nil {
			return MCPToolError("Cannot reserve request_id")
		}
		if !claimed {
			if record.RequestHash != MCPKeyDigest(string(fingerprint)) {
				return MCPToolError("request_id already belongs to different arguments")
			}
			if record.State != "done" {
				return MCPToolError("Request outcome is pending or uncertain; do not execute again")
			}
			if err := common.UnmarshalJsonStr(record.Result, &result); err != nil {
				return MCPToolError("Stored result is unavailable; do not execute again")
			}
			return result
		}
		writeID = record.ID
	}
	if selected.Name == "newapi_context" {
		names := make([]string, 0)
		for _, tool := range MCPVisibleTools(identity.User, identity.Key) {
			names = append(names, tool.Name)
		}
		payload, _ := common.Marshal(map[string]any{"user_id": identity.User.Id, "username": identity.User.Username, "role": identity.User.Role, "key_id": identity.Key.ID, "tools": names, "scope": "NewAPI-Mao management", "money_display_instructions": MCPMoneyInstructions})
		return MCPToolResult{Content: []MCPTextContent{{Type: "text", Text: string(payload)}}}
	}
	executionContext, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	contentType := "application/json"
	if selected.UploadField != "" {
		body, contentType, err = mcpUploadBody(selected.UploadField, body)
		if err != nil {
			return MCPToolError("Cannot construct file upload")
		}
	}
	request, err := http.NewRequestWithContext(MCPInternalContext(executionContext, identity.Key.ID), selected.Method, path, bytes.NewReader(body))
	if err != nil {
		return MCPToolError("Cannot construct internal request")
	}
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Accept", "application/json")
	request.RemoteAddr = "127.0.0.1:0"
	writer := &mcpBoundedResponse{header: make(http.Header), limit: mcpResponseLimit(*selected)}
	// 同步执行：不以超时启动第二个 goroutine，避免超时释放并发槽后业务仍在写入。
	dispatch.ServeHTTP(writer, request)
	if writer.status == 0 {
		writer.status = http.StatusOK
	}
	if writer.exceeded {
		return MCPToolError("Response exceeded limit; write outcome may be uncertain")
	}
	if writer.status >= 500 || executionContext.Err() != nil {
		return MCPToolError("Request outcome is uncertain; inspect business state before further writes")
	}
	projected, success, err := ProjectMCPResponse(*selected, writer.body.Bytes(), writer.status)
	if err != nil {
		if !selected.Write {
			return MCPToolError("Response cannot be safely returned in full; inspect metadata in the dashboard")
		}
		return MCPToolError("Response unavailable; write outcome may be uncertain")
	}
	result = MCPToolResult{Content: []MCPTextContent{{Type: "text", Text: string(projected)}}, IsError: !success}
	if writeID != 0 {
		encoded, err := common.Marshal(result)
		if err == nil && (selected.TypedArguments || selected.TransientResult) {
			receipt, receiptErr := mcpManagementReceipt(projected, arguments.RequestID, success, selected.TransientResult, selected.Name)
			if receiptErr != nil {
				return MCPToolError("Result persistence failed; do not execute again")
			}
			encoded, err = common.Marshal(receipt)
			if err == nil && len(encoded) > 48*1024 {
				receipt, err = mcpManagementReceipt(nil, arguments.RequestID, success, true)
				if err == nil {
					encoded, err = common.Marshal(receipt)
				}
			}
		}
		if err != nil || model.CompleteMCPWriteRequest(writeID, string(encoded), success) != nil {
			return MCPToolError("Result persistence failed; do not execute again")
		}
	}
	return result
}

// 使用 RawMessage 递归排序，保留数值文本精度；不同的大整数不会因 float64 舍入共享摘要。
func canonicalMCPJSON(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		for key, value := range object {
			normalized, err := canonicalMCPJSON(value)
			if err != nil {
				return nil, err
			}
			object[key] = normalized
		}
		return common.Marshal(object)
	case "array":
		var array []json.RawMessage
		if err := common.Unmarshal(raw, &array); err != nil {
			return nil, err
		}
		for i, value := range array {
			normalized, err := canonicalMCPJSON(value)
			if err != nil {
				return nil, err
			}
			array[i] = normalized
		}
		return common.Marshal(array)
	default:
		return common.Marshal(raw)
	}
}
