package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"slices"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/tidwall/gjson"
)

const GatewaySkillsListTool = "newapi_skills_list"
const GatewaySkillsReadTool = "newapi_skills_read"
const GatewaySkillsFilesTool = "newapi_skills_files"

// GatewaySkillCall 描述私有阶段返回的工具调用；客户端调用仅用于交接，不执行。
type GatewaySkillCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type GatewaySkillTurn struct {
	Calls                   []GatewaySkillCall
	ClientCalls             []GatewaySkillCall
	UnrecognizedClientCalls bool
	Assistant               json.RawMessage
	Output                  []json.RawMessage
}

type GatewaySkillResult struct {
	Call    GatewaySkillCall
	Content json.RawMessage
}

// PrepareGatewaySkillRequest 创建独立读取阶段，最终生成仍使用原始客户端工具配置。
func PrepareGatewaySkillRequest(body []byte, format types.RelayFormat, modelName, catalogue string) ([]byte, error) {
	request, err := NewGatewaySkillRequest(body, format, modelName, catalogue)
	if err != nil {
		return nil, err
	}
	return request.Bytes(), nil
}

// NewGatewaySkillRequest 只解析一次客户端正文，续轮复用不可变历史与协议字段。
func NewGatewaySkillRequest(body []byte, format types.RelayFormat, modelName, catalogue string) (*GatewaySkillRequest, error) {
	instruction := "You are completing a private gateway skill-discovery phase before the actual assistant answers. " +
		"Select only installed skills relevant to the current user task. Read a skill's SKILL.md first, then read referenced text resources only when needed. " +
		"Use next_offset to read further pages when needed, and newapi_skills_files to search or page through resource paths after reading SKILL.md. " +
		"Do not execute scripts, perform the user's task, or call any client tools during this phase. " +
		"If the user asks which gateway cloud Skills are installed, or asks to list available skills, you MUST call newapi_skills_list with an empty query before finishing, even when no file content is needed. " +
		"When the useful instructions have been read, or no skill is relevant, reply SKILL_CONTEXT_READY without tool calls. " +
		"The gateway will give successful catalogue metadata and authorized file pages to the actual assistant together with its original tools; catalogue metadata means discovery only and does not mean that a skill was used. " +
		"You have at most 8 discovery rounds and a bounded reading budget.\nAvailable installed skills:\n" + catalogue
	state, request, err := parseGatewaySkillRequest(body, format)
	if err != nil {
		return nil, err
	}
	if err := state.prependInstruction(request, modelName, instruction); err != nil {
		return nil, err
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
	if err := prepareGatewaySkillOutputBudget(request, format, GatewaySkillDiscoveryMaxOutputTokens()); err != nil {
		return nil, err
	}
	if err := state.seal(request); err != nil {
		return nil, err
	}
	return state, nil
}

// GatewaySkillClientTools 读取原请求直接声明的工具；客户端适配层可能使用其他封装。
func GatewaySkillClientTools(body []byte, format types.RelayFormat) map[string]bool {
	names := make(map[string]bool)
	for _, tool := range gjson.GetBytes(body, "tools").Array() {
		var name string
		switch format {
		case types.RelayFormatOpenAI:
			if tool.Get("type").String() == "function" {
				name = tool.Get("function.name").String()
			}
		case types.RelayFormatOpenAIResponses:
			if tool.Get("type").String() == "function" || tool.Get("type").String() == "custom" {
				name = tool.Get("name").String()
			}
		case types.RelayFormatClaude:
			name = tool.Get("name").String()
		case types.RelayFormatGemini:
			declarations := tool.Get("functionDeclarations")
			if !declarations.Exists() {
				declarations = tool.Get("function_declarations")
			}
			for _, declaration := range declarations.Array() {
				if name := declaration.Get("name").String(); name != "" {
					names[name] = true
				}
			}
		}
		if name != "" {
			names[name] = true
		}
	}
	return names
}

// ParseGatewaySkillTurn 保留私有助手内容，将网关读取与其余调用分开；私有阶段不执行客户端工具。
// 未匹配原请求声明的工具只标记为未识别调用，由上层纠偏或结束私有发现。
func ParseGatewaySkillTurn(response []byte, format types.RelayFormat, clientTools map[string]bool) (*GatewaySkillTurn, error) {
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
				Type      string          `json:"type"`
				Name      string          `json:"name"`
				CallID    string          `json:"call_id"`
				Arguments string          `json:"arguments"`
				Input     json.RawMessage `json:"input"`
			}
			if err := common.Unmarshal(item, &output); err != nil {
				return nil, err
			}
			if output.Type == "function_call" {
				turn.Calls = append(turn.Calls, GatewaySkillCall{ID: output.CallID, Name: output.Name, Arguments: json.RawMessage(output.Arguments)})
			} else if output.Type == "custom_tool_call" {
				turn.Calls = append(turn.Calls, GatewaySkillCall{ID: output.CallID, Name: output.Name, Arguments: output.Input})
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
	if len(turn.Calls) > GatewaySkillMaxCalls {
		return nil, errors.New("技能读取阶段单轮工具调用超过 8 项")
	}
	seen := make(map[string]bool, len(turn.Calls))
	internalCalls := make([]GatewaySkillCall, 0, len(turn.Calls))
	for _, call := range turn.Calls {
		internal := call.Name == GatewaySkillsListTool || call.Name == GatewaySkillsReadTool || call.Name == GatewaySkillsFilesTool
		if call.Name == "" {
			return nil, errors.New("技能读取阶段工具名称为空")
		}
		if !internal && !clientTools[call.Name] {
			if format != types.RelayFormatOpenAIResponses {
				return nil, errors.New("技能读取阶段返回未声明工具")
			}
			// 不同客户端封装可能不会把工具声明原样传到此 API 层。
			// 未匹配声明的调用仍须隔离，不能在私有阶段执行。
			turn.UnrecognizedClientCalls = true
		}
		if (format != types.RelayFormatGemini && call.ID == "") || (call.ID != "" && seen[call.ID]) {
			return nil, errors.New("技能读取工具调用标识缺失或重复")
		}
		seen[call.ID] = true
		if internal {
			internalCalls = append(internalCalls, call)
		} else {
			turn.ClientCalls = append(turn.ClientCalls, call)
		}
	}
	turn.Calls = internalCalls
	return turn, nil
}

// RefocusGatewaySkillRequest 只追加阶段纠偏，不回填意外客户端调用或伪造执行结果。
func RefocusGatewaySkillRequest(body []byte, format types.RelayFormat) ([]byte, error) {
	request, err := restoreGatewaySkillRequest(body, format)
	if err != nil {
		return nil, err
	}
	if err := request.Refocus(); err != nil {
		return nil, err
	}
	return request.Bytes(), nil
}

// ContinueGatewaySkillRequest 只在私有读取阶段回填结果，绝不改写客户端原始历史。
func ContinueGatewaySkillRequest(body []byte, format types.RelayFormat, turn *GatewaySkillTurn, results []GatewaySkillResult) ([]byte, error) {
	request, err := restoreGatewaySkillRequest(body, format)
	if err != nil {
		return nil, err
	}
	if err := request.Continue(turn, results); err != nil {
		return nil, err
	}
	return request.Bytes(), nil
}
