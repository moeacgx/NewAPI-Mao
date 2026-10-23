package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ApplyTokenPromptText 只改写协议的系统指令字段，保留未知字段和原始数值精度。
// 调用方每次从客户端原始正文构造，避免重试重复装载或跨分组泄漏。
func ApplyTokenPromptText(body []byte, format types.RelayFormat, modelName string, texts []string) ([]byte, error) {
	if len(texts) == 0 {
		return body, nil
	}
	if format != types.RelayFormatOpenAI && format != types.RelayFormatOpenAIResponses &&
		format != types.RelayFormatClaude && format != types.RelayFormatGemini {
		return body, nil
	}
	text := strings.Join(texts, "\n\n")
	if len(text) > model.MaxTokenPromptBytes+2*(model.MaxTokenPrompts-1) {
		return nil, errors.New("装载提示词合计超出限制")
	}
	return appendSystemInstructionText(body, format, modelName, text)
}

// appendSystemInstructionText 只负责协议改写；全文装载与渐进式读取各自在调用边界验证容量。
func appendSystemInstructionText(body []byte, format types.RelayFormat, modelName, text string) ([]byte, error) {
	if format != types.RelayFormatOpenAI && format != types.RelayFormatOpenAIResponses &&
		format != types.RelayFormatClaude && format != types.RelayFormatGemini {
		return body, nil
	}
	if !gjson.ValidBytes(body) {
		return nil, errors.New("提示词装载需要有效 JSON 正文")
	}
	if format == types.RelayFormatOpenAIResponses {
		existing := gjson.GetBytes(body, "instructions")
		if existing.Type != gjson.Null && existing.Type != gjson.String {
			return nil, errors.New("instructions 必须为字符串")
		}
		if existing.String() != "" {
			text += "\n\n" + existing.String()
		}
		return sjson.SetBytes(body, "instructions", text)
	}
	path := "messages"
	var block any
	var existing []json.RawMessage
	switch format {
	case types.RelayFormatOpenAI:
		block = map[string]any{"role": (&dto.GeneralOpenAIRequest{Model: modelName}).GetSystemRoleName(), "content": text}
	case types.RelayFormatClaude:
		path = "system"
		block = map[string]any{"type": "text", "text": text}
	case types.RelayFormatGemini:
		// DTO 既有语义以 snake_case 优先，归一后移除别名，防止覆盖新装载内容。
		system := gjson.GetBytes(body, "systemInstruction")
		if snake := gjson.GetBytes(body, "system_instruction"); snake.Type != gjson.Null {
			system = snake
		}
		var err error
		if system.Type != gjson.Null {
			if !system.IsObject() {
				return nil, errors.New("systemInstruction 必须为对象")
			}
			body, err = sjson.SetRawBytes(body, "systemInstruction", []byte(system.Raw))
			if err != nil {
				return nil, err
			}
		}
		body, err = sjson.DeleteBytes(body, "system_instruction")
		if err != nil {
			return nil, err
		}
		path = "systemInstruction.parts"
		block = map[string]any{"text": text}
	}
	value := gjson.GetBytes(body, path)
	if format == types.RelayFormatClaude && value.Type == gjson.String {
		clientBlock, err := common.Marshal(map[string]any{"type": "text", "text": value.String()})
		if err != nil {
			return nil, err
		}
		existing = []json.RawMessage{clientBlock}
	} else if value.Type != gjson.Null {
		if !value.IsArray() {
			return nil, errors.New("系统指令消息必须为数组")
		}
		if err := common.Unmarshal([]byte(value.Raw), &existing); err != nil {
			return nil, err
		}
	}
	encoded, err := common.Marshal(block)
	if err != nil {
		return nil, err
	}
	merged := append([]json.RawMessage{encoded}, existing...)
	encoded, err = common.Marshal(merged)
	if err != nil {
		return nil, err
	}
	return sjson.SetRawBytes(body, path, encoded)
}
