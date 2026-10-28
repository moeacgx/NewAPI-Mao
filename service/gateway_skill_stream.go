package service

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

type gatewaySkillStreamTool struct {
	id        strings.Builder
	name      strings.Builder
	arguments strings.Builder
}

// ParseGatewaySkillChatStream 只聚合私有 Chat 发现响应；完整结束后才允许解析和执行工具。
// 参数覆盖或上游可能将非流式请求改成 SSE，此处保留分片助手消息并复用原有授权检查。
func ParseGatewaySkillChatStream(response []byte, clientTools map[string]bool) (*GatewaySkillTurn, error) {
	stream := strings.ReplaceAll(string(response), "\r\n", "\n")
	frames := strings.Split(stream, "\n\n")
	if strings.TrimSpace(frames[len(frames)-1]) != "" {
		return nil, errors.New("技能读取阶段流式响应帧不完整")
	}
	tools := make(map[int]*gatewaySkillStreamTool)
	texts := make(map[string]*strings.Builder)
	role, finishReason := "", ""
	choiceIndex := -1
	done := false
	for _, frame := range frames[:len(frames)-1] {
		var data []string
		for _, line := range strings.Split(frame, "\n") {
			field, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			if field == "event" && value == "error" {
				return nil, errors.New("技能读取阶段流式响应返回错误")
			}
			if field == "data" {
				data = append(data, value)
			}
		}
		if len(data) == 0 {
			continue
		}
		if done {
			return nil, errors.New("技能读取阶段流式响应结束后仍有数据")
		}
		payload := strings.Join(data, "\n")
		if strings.TrimSpace(payload) == "[DONE]" {
			done = true
			continue
		}
		var chunk struct {
			Error   json.RawMessage `json:"error"`
			Choices []struct {
				Index        *int            `json:"index"`
				Delta        json.RawMessage `json:"delta"`
				FinishReason *string         `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := common.UnmarshalJsonStr(payload, &chunk); err != nil || len(chunk.Error) > 0 || chunk.Choices == nil {
			return nil, errors.New("技能读取阶段流式响应数据无效")
		}
		if len(chunk.Choices) == 0 {
			// 用量尾帧不包含候选，计费已由正常中继链路完成。
			continue
		}
		if len(chunk.Choices) != 1 || finishReason != "" {
			return nil, errors.New("技能读取阶段流式响应候选不唯一或已结束")
		}
		choice := chunk.Choices[0]
		if choice.Index == nil || *choice.Index < 0 || (choiceIndex >= 0 && choiceIndex != *choice.Index) {
			return nil, errors.New("技能读取阶段流式响应候选标识无效")
		}
		choiceIndex = *choice.Index
		var delta map[string]json.RawMessage
		if err := common.Unmarshal(choice.Delta, &delta); err != nil || delta == nil {
			return nil, errors.New("技能读取阶段流式助手消息无效")
		}
		for _, field := range []string{"role", "content", "reasoning_content", "reasoning", "refusal"} {
			raw, exists := delta[field]
			if !exists || string(raw) == "null" {
				continue
			}
			var value string
			if err := common.Unmarshal(raw, &value); err != nil {
				return nil, errors.New("技能读取阶段流式助手字段无效")
			}
			if field == "role" {
				if value != "assistant" {
					return nil, errors.New("技能读取阶段流式响应不是助手消息")
				}
				role = value
				continue
			}
			if texts[field] == nil {
				texts[field] = &strings.Builder{}
			}
			texts[field].WriteString(value)
		}
		if raw := delta["tool_calls"]; len(raw) > 0 {
			var calls []struct {
				Index    *int   `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			}
			if err := common.Unmarshal(raw, &calls); err != nil {
				return nil, errors.New("技能读取阶段流式工具调用无效")
			}
			for _, call := range calls {
				if call.Index == nil || *call.Index < 0 || *call.Index >= GatewaySkillMaxCalls || (call.Type != "" && call.Type != "function") {
					return nil, errors.New("技能读取阶段流式工具标识或类型无效")
				}
				tool := tools[*call.Index]
				if tool == nil {
					tool = &gatewaySkillStreamTool{}
					tools[*call.Index] = tool
				}
				tool.id.WriteString(call.ID)
				tool.name.WriteString(call.Function.Name)
				tool.arguments.WriteString(call.Function.Arguments)
			}
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			finishReason = *choice.FinishReason
			if finishReason != "stop" && finishReason != "tool_calls" {
				return nil, errors.New("技能读取阶段流式响应未正常完成")
			}
		}
	}
	if !done || finishReason == "" || role != "assistant" || (finishReason == "tool_calls" && len(tools) == 0) {
		return nil, errors.New("技能读取阶段流式响应未完整结束")
	}
	message := map[string]any{"role": role, "content": nil}
	for field, value := range texts {
		message[field] = value.String()
	}
	indices := make([]int, 0, len(tools))
	for index := range tools {
		indices = append(indices, index)
	}
	slices.Sort(indices)
	calls := make([]map[string]any, 0, len(tools))
	for _, index := range indices {
		tool := tools[index]
		var arguments map[string]any
		if err := common.UnmarshalJsonStr(tool.arguments.String(), &arguments); err != nil || arguments == nil {
			return nil, errors.New("技能读取阶段流式工具参数不完整")
		}
		calls = append(calls, map[string]any{"id": tool.id.String(), "type": "function", "function": map[string]any{"name": tool.name.String(), "arguments": tool.arguments.String()}})
	}
	if len(calls) > 0 {
		message["tool_calls"] = calls
	}
	envelope, err := common.Marshal(map[string]any{"choices": []map[string]any{{"message": message}}})
	if err != nil {
		return nil, err
	}
	return ParseGatewaySkillTurn(envelope, types.RelayFormatOpenAI, clientTools)
}
