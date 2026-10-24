package service

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
)

// GatewaySkillRequest 保存单次私有发现的请求状态；不在请求之间共享，也不并发使用。
// 原始历史和已编码字段始终只读，续轮仅编码新增项，出站时再顺序复制各段。
type GatewaySkillRequest struct {
	format       types.RelayFormat
	historyKey   string
	prefix       []byte
	history      []json.RawMessage
	historyBytes int
}

func parseGatewaySkillRequest(body []byte, format types.RelayFormat) (*GatewaySkillRequest, map[string]json.RawMessage, error) {
	state := &GatewaySkillRequest{format: format, historyKey: "messages"}
	switch format {
	case types.RelayFormatOpenAI, types.RelayFormatClaude:
	case types.RelayFormatOpenAIResponses:
		state.historyKey = "input"
	case types.RelayFormatGemini:
		state.historyKey = "contents"
	default:
		return nil, nil, errors.New("该端点不支持渐进式技能")
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(body, &request); err != nil || request == nil {
		return nil, nil, errors.New("技能读取请求必须为 JSON 对象")
	}
	history := bytes.TrimSpace(request[state.historyKey])
	switch common.GetJsonType(history) {
	case "unknown", "null":
	case "array":
		// 根对象解析已验证完整 JSON；保留数组内部原文，无需逐条解析和复制。
		state.appendHistory(bytes.TrimSpace(history[1 : len(history)-1]))
	case "string":
		if format != types.RelayFormatOpenAIResponses {
			return nil, nil, errors.New("技能读取历史必须为数组")
		}
		item, err := common.Marshal(map[string]any{"role": "user", "content": json.RawMessage(history)})
		if err != nil {
			return nil, nil, err
		}
		state.appendHistory(item)
	default:
		return nil, nil, errors.New("技能读取历史必须为数组")
	}
	delete(request, state.historyKey)
	return state, request, nil
}

func restoreGatewaySkillRequest(body []byte, format types.RelayFormat) (*GatewaySkillRequest, error) {
	state, request, err := parseGatewaySkillRequest(body, format)
	if err != nil {
		return nil, err
	}
	if err := state.seal(request); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *GatewaySkillRequest) seal(request map[string]json.RawMessage) error {
	encoded, err := common.Marshal(request)
	if err != nil {
		return err
	}
	encoded = encoded[:len(encoded)-1]
	if len(request) > 0 {
		encoded = append(encoded, ',')
	}
	r.prefix = append(encoded, '"')
	r.prefix = append(r.prefix, r.historyKey...)
	r.prefix = append(r.prefix, '"', ':', '[')
	return nil
}

// Bytes 返回独立的出站正文；后续追加不会修改此前返回的正文。
func (r *GatewaySkillRequest) Bytes() []byte {
	body := make([]byte, 0, len(r.prefix)+r.historyBytes+2)
	body = append(body, r.prefix...)
	for i, part := range r.history {
		if i > 0 {
			body = append(body, ',')
		}
		body = append(body, part...)
	}
	return append(body, ']', '}')
}

func (r *GatewaySkillRequest) appendHistory(history json.RawMessage) {
	if len(history) == 0 {
		return
	}
	if len(r.history) > 0 {
		r.historyBytes++
	}
	r.historyBytes += len(history)
	r.history = append(r.history, history)
}

func (r *GatewaySkillRequest) prependInstruction(request map[string]json.RawMessage, modelName, instruction string) error {
	switch r.format {
	case types.RelayFormatOpenAI:
		item, err := common.Marshal(map[string]any{"role": (&dto.GeneralOpenAIRequest{Model: modelName}).GetSystemRoleName(), "content": instruction})
		if err != nil {
			return err
		}
		if len(r.history) > 0 {
			r.historyBytes++
		}
		r.historyBytes += len(item)
		r.history = append([]json.RawMessage{item}, r.history...)
	case types.RelayFormatOpenAIResponses:
		var existing string
		if raw := request["instructions"]; len(raw) > 0 {
			if err := common.Unmarshal(raw, &existing); err != nil {
				return errors.New("instructions 必须为字符串")
			}
		}
		if existing != "" {
			instruction += "\n\n" + existing
		}
		encoded, err := common.Marshal(instruction)
		if err != nil {
			return err
		}
		request["instructions"] = encoded
	case types.RelayFormatClaude:
		item, err := common.Marshal(map[string]any{"type": "text", "text": instruction})
		if err != nil {
			return err
		}
		request["system"], err = prependGatewaySkillInstruction(request["system"], item, true)
		return err
	case types.RelayFormatGemini:
		// 与 Gemini DTO 保持 snake_case 优先，并保留系统对象的未知字段。
		raw := request["systemInstruction"]
		if snake := request["system_instruction"]; len(snake) > 0 && common.GetJsonType(snake) != "null" {
			raw = snake
		}
		var system map[string]json.RawMessage
		if len(raw) > 0 {
			if err := common.Unmarshal(raw, &system); err != nil {
				return errors.New("systemInstruction 必须为对象")
			}
		}
		if system == nil {
			system = make(map[string]json.RawMessage)
		}
		item, err := common.Marshal(map[string]any{"text": instruction})
		if err != nil {
			return err
		}
		system["parts"], err = prependGatewaySkillInstruction(system["parts"], item, false)
		if err != nil {
			return err
		}
		request["systemInstruction"], err = common.Marshal(system)
		delete(request, "system_instruction")
		return err
	}
	return nil
}

func prependGatewaySkillInstruction(raw, instruction json.RawMessage, allowString bool) (json.RawMessage, error) {
	var existing json.RawMessage
	switch common.GetJsonType(raw) {
	case "unknown", "null":
	case "array":
		trimmed := bytes.TrimSpace(raw)
		existing = bytes.TrimSpace(trimmed[1 : len(trimmed)-1])
	case "string":
		if !allowString {
			return nil, errors.New("系统指令消息必须为数组")
		}
		var err error
		existing, err = common.Marshal(map[string]any{"type": "text", "text": raw})
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("系统指令消息必须为数组")
	}
	encoded := make([]byte, 0, len(instruction)+len(existing)+3)
	encoded = append(encoded, '[')
	encoded = append(encoded, instruction...)
	if len(existing) > 0 {
		encoded = append(encoded, ',')
		encoded = append(encoded, existing...)
	}
	return append(encoded, ']'), nil
}

// Refocus 只追加阶段纠偏，不记录意外客户端调用，也不伪造其执行结果。
func (r *GatewaySkillRequest) Refocus() error {
	instruction := "Private gateway discovery correction: the requested client tool was NOT executed. " +
		"You are still selecting skill context, not performing the original Agent task. " +
		"Use only newapi_skills_list, newapi_skills_read, and newapi_skills_files. " +
		"Read the relevant installed skill's SKILL.md before handing back to the Agent. " +
		"If no installed skill is relevant, reply SKILL_CONTEXT_READY without tools. " +
		"For catalogue questions, call newapi_skills_list. Original client tools will be restored for the actual assistant."
	message := map[string]any{"role": "user", "content": instruction}
	if r.format == types.RelayFormatGemini {
		message = map[string]any{"role": "user", "parts": []map[string]any{{"text": instruction}}}
	}
	item, err := common.Marshal(message)
	if err != nil {
		return err
	}
	r.appendHistory(item)
	return nil
}

// Continue 仅编码本轮新增助手项和工具结果，失败时不改变已有请求状态。
func (r *GatewaySkillRequest) Continue(turn *GatewaySkillTurn, results []GatewaySkillResult) error {
	if turn == nil || len(turn.Calls) == 0 || len(turn.ClientCalls) > 0 || len(turn.Calls) != len(results) {
		return errors.New("技能工具结果与调用不匹配")
	}
	for i, result := range results {
		if result.Call.ID != turn.Calls[i].ID || result.Call.Name != turn.Calls[i].Name {
			return errors.New("技能工具结果标识不匹配")
		}
	}
	var history []json.RawMessage
	switch r.format {
	case types.RelayFormatOpenAI:
		history = append(history, turn.Assistant)
		for _, result := range results {
			item, err := common.Marshal(map[string]any{"role": "tool", "tool_call_id": result.Call.ID, "content": string(result.Content)})
			if err != nil {
				return err
			}
			history = append(history, item)
		}
	case types.RelayFormatOpenAIResponses:
		history = append(history, turn.Output...)
		for _, result := range results {
			item, err := common.Marshal(map[string]any{"type": "function_call_output", "call_id": result.Call.ID, "output": string(result.Content)})
			if err != nil {
				return err
			}
			history = append(history, item)
		}
	case types.RelayFormatClaude:
		assistant, err := common.Marshal(map[string]any{"role": "assistant", "content": turn.Output})
		if err != nil {
			return err
		}
		history = append(history, assistant)
		blocks := make([]map[string]any, 0, len(results))
		for _, result := range results {
			blocks = append(blocks, map[string]any{"type": "tool_result", "tool_use_id": result.Call.ID, "content": string(result.Content)})
		}
		user, err := common.Marshal(map[string]any{"role": "user", "content": blocks})
		if err != nil {
			return err
		}
		history = append(history, user)
	case types.RelayFormatGemini:
		history = append(history, turn.Assistant)
		parts := make([]map[string]any, 0, len(results))
		for _, result := range results {
			response := map[string]any{"name": result.Call.Name, "response": result.Content}
			if result.Call.ID != "" {
				response["id"] = result.Call.ID
			}
			parts = append(parts, map[string]any{"functionResponse": response})
		}
		user, err := common.Marshal(map[string]any{"role": "user", "parts": parts})
		if err != nil {
			return err
		}
		history = append(history, user)
	}
	encoded, err := common.Marshal(history)
	if err != nil {
		return err
	}
	r.appendHistory(encoded[1 : len(encoded)-1])
	return nil
}
