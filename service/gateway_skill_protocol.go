package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

const GatewaySkillsListTool = "newapi_skills_list"
const GatewaySkillsReadTool = "newapi_skills_read"
const GatewaySkillsFilesTool = "newapi_skills_files"

// GatewaySkillCall 仅描述网关读取阶段声明的工具，不执行客户端工具。
type GatewaySkillCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type GatewaySkillTurn struct {
	Calls     []GatewaySkillCall
	Assistant json.RawMessage
	Output    []json.RawMessage
}

type GatewaySkillResult struct {
	Call    GatewaySkillCall
	Content json.RawMessage
}

// PrepareGatewaySkillRequest 创建独立读取阶段，最终生成仍使用原始客户端工具配置。
func PrepareGatewaySkillRequest(body []byte, format types.RelayFormat, modelName, catalogue string) ([]byte, error) {
	instruction := "You are completing a private gateway skill-discovery phase before the actual assistant answers. " +
		"Select only installed skills relevant to the current user task. Read a skill's SKILL.md first, then read referenced text resources only when needed. " +
		"Use next_offset to read further pages when needed, and newapi_skills_files to search or page through resource paths after reading SKILL.md. " +
		"Do not execute scripts, perform the user's task, or call any client tools during this phase. " +
		"When the useful instructions have been read, or no skill is relevant, reply SKILL_CONTEXT_READY without tool calls. " +
		"The gateway will give the authorized file pages to the actual assistant together with its original tools. " +
		"You have at most 8 discovery rounds and a bounded reading budget.\nAvailable installed skills:\n" + catalogue
	prepared, err := appendSystemInstructionText(body, format, modelName, instruction)
	if err != nil {
		return nil, err
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(prepared, &request); err != nil || request == nil {
		return nil, errors.New("技能读取请求必须为 JSON 对象")
	}
	listSchema := map[string]any{
		"type": "object", "properties": map[string]any{
			"query":  map[string]any{"type": "string", "description": "Optional name or description search."},
			"offset": map[string]any{"type": "integer", "minimum": 0},
		},
	}
	readSchema := map[string]any{
		"type": "object", "properties": map[string]any{
			"template_id": map[string]any{"type": "integer", "minimum": 1},
			"skill_id":    map[string]any{"type": "string"},
			"version":     map[string]any{"type": "integer", "minimum": 1},
			"path":        map[string]any{"type": "string", "description": "SKILL.md or an exact resource path returned by reading SKILL.md."},
			"offset":      map[string]any{"type": "integer", "minimum": 0, "description": "UTF-8 byte offset; use next_offset from the previous page."},
		}, "required": []string{"template_id", "skill_id", "version"},
	}
	filesSchema := map[string]any{
		"type": "object", "properties": map[string]any{
			"template_id": map[string]any{"type": "integer", "minimum": 1},
			"skill_id":    map[string]any{"type": "string"},
			"version":     map[string]any{"type": "integer", "minimum": 1},
			"query":       map[string]any{"type": "string", "description": "Optional resource path search."},
			"offset":      map[string]any{"type": "integer", "minimum": 0},
		}, "required": []string{"template_id", "skill_id", "version"},
	}
	definitions := []map[string]any{
		{"name": GatewaySkillsListTool, "description": "Search or page through installed skill names and descriptions. Returns metadata only, never file content.", "parameters": listSchema},
		{"name": GatewaySkillsReadTool, "description": "Read one authorized installed skill file page (up to 24 KiB). Read SKILL.md first to see resource paths. Large files remain available through next_offset; no 32 KiB file limit.", "parameters": readSchema},
		{"name": GatewaySkillsFilesTool, "description": "Search or page through a selected skill's resource paths, after reading SKILL.md. Returns metadata only, never file content. Use next_offset for the next directory page.", "parameters": filesSchema},
	}
	var tools any
	switch format {
	case types.RelayFormatOpenAI:
		entries := make([]map[string]any, 0, len(definitions))
		for _, definition := range definitions {
			entries = append(entries, map[string]any{"type": "function", "function": definition})
		}
		tools = entries
		request["tool_choice"] = json.RawMessage(`"auto"`)
		request["parallel_tool_calls"] = json.RawMessage(`false`)
		request["n"] = json.RawMessage(`1`)
		delete(request, "functions")
		delete(request, "function_call")
		delete(request, "response_format")
		delete(request, "stream_options")
	case types.RelayFormatOpenAIResponses:
		// store=false 不会阻止 conversation 保存工具结果，不能复用客户端持久会话。
		if conversation := bytes.TrimSpace(request["conversation"]); len(conversation) > 0 && !bytes.Equal(conversation, []byte("null")) {
			return nil, types.NewErrorWithStatusCode(errors.New("渐进式技能暂不支持 Responses conversation，请使用包含完整对话历史的 input"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		for _, definition := range definitions {
			definition["type"] = "function"
			definition["strict"] = false
		}
		tools = definitions
		request["tool_choice"] = json.RawMessage(`"auto"`)
		request["parallel_tool_calls"] = json.RawMessage(`false`)
		request["store"] = json.RawMessage(`false`)
		request["background"] = json.RawMessage(`false`)
		// 无状态续轮必须带回加密推理项，不能丢失 reasoning 与工具调用的关联。
		var include []string
		if len(request["include"]) > 0 {
			if err := common.Unmarshal(request["include"], &include); err != nil {
				return nil, err
			}
		}
		if !slices.Contains(include, "reasoning.encrypted_content") {
			include = append(include, "reasoning.encrypted_content")
		}
		request["include"], err = common.Marshal(include)
		if err != nil {
			return nil, err
		}
		delete(request, "text")
	case types.RelayFormatClaude:
		entries := make([]map[string]any, 0, len(definitions))
		for _, definition := range definitions {
			entries = append(entries, map[string]any{"name": definition["name"], "description": definition["description"], "input_schema": definition["parameters"]})
		}
		tools = entries
		request["tool_choice"] = json.RawMessage(`{"type":"auto","disable_parallel_tool_use":true}`)
		delete(request, "output_config")
		delete(request, "mcp_servers")
	case types.RelayFormatGemini:
		// 缓存绑定的指令和工具不能被私有阶段替换，也不能删除引用而丢失客户端上下文。
		if raw := request["cachedContent"]; len(raw) > 0 {
			var cachedContent string
			if err := common.Unmarshal(raw, &cachedContent); err != nil || cachedContent != "" {
				return nil, types.NewErrorWithStatusCode(errors.New("渐进式技能暂不支持 Gemini cachedContent，请使用包含完整上下文的 contents"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
		}
		tools = []map[string]any{{"functionDeclarations": definitions}}
		request["toolConfig"] = json.RawMessage(`{"functionCallingConfig":{"mode":"AUTO"}}`)
		delete(request, "tool_config")
		var generation map[string]json.RawMessage
		if len(request["generationConfig"]) > 0 {
			if err := common.Unmarshal(request["generationConfig"], &generation); err != nil {
				return nil, err
			}
		}
		if generation == nil {
			generation = make(map[string]json.RawMessage)
		}
		delete(generation, "responseMimeType")
		delete(generation, "responseSchema")
		delete(generation, "responseJsonSchema")
		generation["candidateCount"] = json.RawMessage(`1`)
		request["generationConfig"], err = common.Marshal(generation)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("该端点不支持渐进式技能")
	}
	request["tools"], err = common.Marshal(tools)
	if err != nil {
		return nil, err
	}
	if format != types.RelayFormatGemini {
		request["stream"] = json.RawMessage(`false`)
	}
	return common.Marshal(request)
}

// ParseGatewaySkillTurn 保留原始助手内容，含推理签名与未知字段，供同渠道续轮使用。
func ParseGatewaySkillTurn(response []byte, format types.RelayFormat) (*GatewaySkillTurn, error) {
	turn := &GatewaySkillTurn{}
	switch format {
	case types.RelayFormatOpenAI:
		var envelope struct {
			Choices []struct {
				Message json.RawMessage `json:"message"`
			} `json:"choices"`
		}
		if err := common.Unmarshal(response, &envelope); err != nil || len(envelope.Choices) != 1 {
			return nil, errors.New("技能读取阶段未返回唯一助手消息")
		}
		turn.Assistant = envelope.Choices[0].Message
		var message struct {
			Role      string `json:"role"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		}
		if err := common.Unmarshal(turn.Assistant, &message); err != nil || message.Role != "assistant" {
			return nil, errors.New("技能读取阶段助手消息无效")
		}
		for _, call := range message.ToolCalls {
			turn.Calls = append(turn.Calls, GatewaySkillCall{ID: call.ID, Name: call.Function.Name, Arguments: json.RawMessage(call.Function.Arguments)})
		}
	case types.RelayFormatOpenAIResponses:
		var envelope struct {
			Output []json.RawMessage `json:"output"`
			Status string            `json:"status"`
		}
		if err := common.Unmarshal(response, &envelope); err != nil || envelope.Output == nil || envelope.Status == "failed" || envelope.Status == "incomplete" {
			return nil, errors.New("技能读取阶段 Responses 输出不完整")
		}
		turn.Output = envelope.Output
		for _, item := range envelope.Output {
			var output struct {
				Type      string `json:"type"`
				Name      string `json:"name"`
				CallID    string `json:"call_id"`
				Arguments string `json:"arguments"`
			}
			if err := common.Unmarshal(item, &output); err != nil {
				return nil, err
			}
			if output.Type == "function_call" {
				turn.Calls = append(turn.Calls, GatewaySkillCall{ID: output.CallID, Name: output.Name, Arguments: json.RawMessage(output.Arguments)})
			}
		}
	case types.RelayFormatClaude:
		var envelope struct {
			Content []json.RawMessage `json:"content"`
			Role    string            `json:"role"`
		}
		if err := common.Unmarshal(response, &envelope); err != nil || envelope.Role != "assistant" || envelope.Content == nil {
			return nil, errors.New("技能读取阶段 Claude 输出无效")
		}
		turn.Output = envelope.Content
		for _, item := range envelope.Content {
			var block struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			}
			if err := common.Unmarshal(item, &block); err != nil {
				return nil, err
			}
			if block.Type == "tool_use" {
				turn.Calls = append(turn.Calls, GatewaySkillCall{ID: block.ID, Name: block.Name, Arguments: block.Input})
			}
		}
	case types.RelayFormatGemini:
		var envelope struct {
			Candidates []struct {
				Content json.RawMessage `json:"content"`
			} `json:"candidates"`
		}
		if err := common.Unmarshal(response, &envelope); err != nil || len(envelope.Candidates) != 1 {
			return nil, errors.New("技能读取阶段 Gemini 未返回唯一候选")
		}
		turn.Assistant = envelope.Candidates[0].Content
		var content struct {
			Parts []struct {
				FunctionCall *struct {
					ID   string          `json:"id"`
					Name string          `json:"name"`
					Args json.RawMessage `json:"args"`
				} `json:"functionCall"`
			} `json:"parts"`
		}
		if err := common.Unmarshal(turn.Assistant, &content); err != nil || content.Parts == nil {
			return nil, errors.New("技能读取阶段 Gemini 内容无效")
		}
		for _, part := range content.Parts {
			if call := part.FunctionCall; call != nil {
				turn.Calls = append(turn.Calls, GatewaySkillCall{ID: call.ID, Name: call.Name, Arguments: call.Args})
			}
		}
	default:
		return nil, errors.New("该端点不支持渐进式技能")
	}
	seen := make(map[string]bool, len(turn.Calls))
	for _, call := range turn.Calls {
		if call.Name != GatewaySkillsListTool && call.Name != GatewaySkillsReadTool && call.Name != GatewaySkillsFilesTool {
			return nil, fmt.Errorf("技能读取阶段返回未声明工具 %q", call.Name)
		}
		if (format != types.RelayFormatGemini && call.ID == "") || (call.ID != "" && seen[call.ID]) {
			return nil, errors.New("技能读取工具调用标识缺失或重复")
		}
		seen[call.ID] = true
	}
	return turn, nil
}

// ContinueGatewaySkillRequest 只在私有读取阶段回填结果，绝不改写客户端原始历史。
func ContinueGatewaySkillRequest(body []byte, format types.RelayFormat, turn *GatewaySkillTurn, results []GatewaySkillResult) ([]byte, error) {
	if turn == nil || len(turn.Calls) == 0 || len(turn.Calls) != len(results) {
		return nil, errors.New("技能工具结果与调用不匹配")
	}
	for i, result := range results {
		if result.Call.ID != turn.Calls[i].ID || result.Call.Name != turn.Calls[i].Name {
			return nil, errors.New("技能工具结果标识不匹配")
		}
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(body, &request); err != nil || request == nil {
		return nil, errors.New("技能读取请求必须为 JSON 对象")
	}
	key := "messages"
	if format == types.RelayFormatGemini {
		key = "contents"
	} else if format == types.RelayFormatOpenAIResponses {
		key = "input"
	}
	var history []json.RawMessage
	if format == types.RelayFormatOpenAIResponses && common.GetJsonType(request[key]) == "string" {
		item, err := common.Marshal(map[string]any{"role": "user", "content": request[key]})
		if err != nil {
			return nil, err
		}
		history = append(history, item)
	} else if len(request[key]) > 0 {
		if err := common.Unmarshal(request[key], &history); err != nil {
			return nil, err
		}
	}
	switch format {
	case types.RelayFormatOpenAI:
		history = append(history, turn.Assistant)
		for _, result := range results {
			item, err := common.Marshal(map[string]any{"role": "tool", "tool_call_id": result.Call.ID, "content": string(result.Content)})
			if err != nil {
				return nil, err
			}
			history = append(history, item)
		}
	case types.RelayFormatOpenAIResponses:
		history = append(history, turn.Output...)
		for _, result := range results {
			item, err := common.Marshal(map[string]any{"type": "function_call_output", "call_id": result.Call.ID, "output": string(result.Content)})
			if err != nil {
				return nil, err
			}
			history = append(history, item)
		}
	case types.RelayFormatClaude:
		assistant, err := common.Marshal(map[string]any{"role": "assistant", "content": turn.Output})
		if err != nil {
			return nil, err
		}
		history = append(history, assistant)
		blocks := make([]map[string]any, 0, len(results))
		for _, result := range results {
			blocks = append(blocks, map[string]any{"type": "tool_result", "tool_use_id": result.Call.ID, "content": string(result.Content)})
		}
		user, err := common.Marshal(map[string]any{"role": "user", "content": blocks})
		if err != nil {
			return nil, err
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
			return nil, err
		}
		history = append(history, user)
	default:
		return nil, errors.New("该端点不支持渐进式技能")
	}
	encoded, err := common.Marshal(history)
	if err != nil {
		return nil, err
	}
	request[key] = encoded
	return common.Marshal(request)
}
