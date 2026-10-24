package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
)

// GatewaySkillDiscoveryMaxOutputTokens 仅限制私有发现，每轮计费仍按实际用量结算。
func GatewaySkillDiscoveryMaxOutputTokens() uint {
	return uint(gatewaySkillPositiveEnv("GATEWAY_SKILLS_DISCOVERY_MAX_OUTPUT_TOKENS", 4096, dto.MaxTokensLimit-1))
}

func gatewaySkillBudgetError(message string) error {
	return types.NewErrorWithStatusCode(errors.New(message), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
}

// gatewaySkillOutputLimit 校验全部别名，按给定优先级选择客户端上限，保留显式零值。
func gatewaySkillOutputLimit(fields map[string]json.RawMessage, ceiling uint, names ...string) (uint, error) {
	limit := ceiling
	selected := false
	for _, name := range names {
		raw := fields[name]
		if len(raw) == 0 || common.GetJsonType(raw) == "null" {
			continue
		}
		var value uint64
		if err := common.Unmarshal(raw, &value); err != nil || value >= dto.MaxTokensLimit {
			return 0, gatewaySkillBudgetError(fmt.Sprintf("%s 必须为小于 %d 的非负整数", name, dto.MaxTokensLimit))
		}
		if !selected {
			limit = min(ceiling, uint(value))
			selected = true
		}
	}
	return limit, nil
}

func prepareGatewaySkillOutputBudget(request map[string]json.RawMessage, format types.RelayFormat, ceiling uint) error {
	switch format {
	case types.RelayFormatOpenAI:
		limit, err := gatewaySkillOutputLimit(request, ceiling, "max_completion_tokens", "max_tokens")
		if err != nil {
			return err
		}
		field := "max_tokens"
		if raw := request["max_completion_tokens"]; len(raw) > 0 && common.GetJsonType(raw) != "null" {
			field = "max_completion_tokens"
		}
		delete(request, "max_tokens")
		delete(request, "max_completion_tokens")
		request[field] = json.RawMessage(strconv.FormatUint(uint64(limit), 10))
		if err := boundGatewaySkillManualThinking(request, limit, false); err != nil {
			return err
		}
		return boundGatewaySkillReasoning(request, limit)
	case types.RelayFormatOpenAIResponses:
		limit, err := gatewaySkillOutputLimit(request, ceiling, "max_output_tokens")
		if err != nil {
			return err
		}
		request["max_output_tokens"] = json.RawMessage(strconv.FormatUint(uint64(limit), 10))
		return boundGatewaySkillReasoning(request, limit)
	case types.RelayFormatClaude:
		limit, err := gatewaySkillOutputLimit(request, ceiling, "max_tokens", "max_tokens_to_sample")
		if err != nil {
			return err
		}
		request["max_tokens"] = json.RawMessage(strconv.FormatUint(uint64(limit), 10))
		delete(request, "max_tokens_to_sample")
		return boundGatewaySkillManualThinking(request, limit, true)
	case types.RelayFormatGemini:
		var generation map[string]json.RawMessage
		if raw := request["generationConfig"]; len(raw) > 0 {
			if err := common.Unmarshal(raw, &generation); err != nil {
				return gatewaySkillBudgetError("generationConfig 必须为对象")
			}
		}
		if generation == nil {
			generation = make(map[string]json.RawMessage)
		}
		limit, err := gatewaySkillOutputLimit(generation, ceiling, "max_output_tokens", "maxOutputTokens")
		if err != nil {
			return err
		}
		generation["maxOutputTokens"] = json.RawMessage(strconv.FormatUint(uint64(limit), 10))
		delete(generation, "max_output_tokens")
		generation["candidateCount"] = json.RawMessage(`1`)
		for _, field := range []string{"candidate_count", "responseMimeType", "response_mime_type", "responseSchema", "response_schema", "responseJsonSchema", "response_json_schema"} {
			delete(generation, field)
		}
		// Gemini DTO 对非 null 的 snake_case 对象优先；不能留下会覆盖私有配置的旧别名。
		if raw := generation["thinking_config"]; len(raw) > 0 && common.GetJsonType(raw) != "null" {
			generation["thinkingConfig"] = raw
		}
		delete(generation, "thinking_config")
		if err := boundGatewaySkillGeminiThinking(generation, limit); err != nil {
			return err
		}
		request["generationConfig"], err = common.Marshal(generation)
		return err
	}
	return nil
}

// boundGatewaySkillManualThinking 保留思考模式与未知配置，只收紧已知的手动预算。
func boundGatewaySkillManualThinking(request map[string]json.RawMessage, limit uint, claude bool) error {
	raw := request["thinking"]
	if len(raw) == 0 || common.GetJsonType(raw) == "null" {
		return nil
	}
	var thinking map[string]json.RawMessage
	if err := common.Unmarshal(raw, &thinking); err != nil {
		if !claude {
			// Chat 兼容供应商还允许其他 thinking 形态，不能按 Claude 规则改写。
			return nil
		}
		return gatewaySkillBudgetError("thinking 必须为对象")
	}
	var mode string
	if err := common.Unmarshal(thinking["type"], &mode); err != nil || mode != "enabled" {
		return nil
	}
	budgetRaw := thinking["budget_tokens"]
	if len(budgetRaw) == 0 && !claude {
		return nil
	}
	var budget uint64
	if err := common.Unmarshal(budgetRaw, &budget); err != nil || budget < 1024 || budget >= dto.MaxTokensLimit {
		return gatewaySkillBudgetError("enabled thinking 的 budget_tokens 必须为不小于 1024 的有效整数")
	}
	if limit <= 1024 {
		return gatewaySkillBudgetError("私有技能发现的输出上限必须大于 enabled thinking 所需的最小预算 1024")
	}
	// 为工具调用留出最多 1024 token，同时兼容客户端本来就很小的合法上限。
	thinkingLimit := max(uint(1024), limit-1024)
	if budget <= uint64(thinkingLimit) {
		return nil
	}
	thinking["budget_tokens"] = json.RawMessage(strconv.FormatUint(uint64(thinkingLimit), 10))
	encoded, err := common.Marshal(thinking)
	if err == nil {
		request["thinking"] = encoded
	}
	return err
}

func boundGatewaySkillReasoning(request map[string]json.RawMessage, limit uint) error {
	raw := request["reasoning"]
	if len(raw) == 0 || common.GetJsonType(raw) != "object" {
		return nil
	}
	var reasoning map[string]json.RawMessage
	if err := common.Unmarshal(raw, &reasoning); err != nil {
		return err
	}
	if raw := reasoning["max_tokens"]; len(raw) == 0 || common.GetJsonType(raw) == "null" {
		return nil
	}
	budget, err := gatewaySkillOutputLimit(reasoning, limit, "max_tokens")
	if err != nil {
		return err
	}
	reasoning["max_tokens"] = json.RawMessage(strconv.FormatUint(uint64(budget), 10))
	request["reasoning"], err = common.Marshal(reasoning)
	return err
}

func boundGatewaySkillGeminiThinking(generation map[string]json.RawMessage, limit uint) error {
	raw := generation["thinkingConfig"]
	if len(raw) == 0 || common.GetJsonType(raw) == "null" {
		return nil
	}
	var thinking map[string]json.RawMessage
	if err := common.Unmarshal(raw, &thinking); err != nil || thinking == nil {
		return gatewaySkillBudgetError("thinkingConfig 必须为对象")
	}
	for _, names := range [][2]string{{"thinking_budget", "thinkingBudget"}, {"include_thoughts", "includeThoughts"}} {
		if raw := thinking[names[0]]; len(raw) > 0 && common.GetJsonType(raw) != "null" {
			thinking[names[1]] = raw
		}
		delete(thinking, names[0])
	}
	if raw := thinking["thinking_level"]; len(raw) > 0 && common.GetJsonType(raw) != "null" {
		var level string
		if err := common.Unmarshal(raw, &level); err != nil {
			return gatewaySkillBudgetError("thinking_level 必须为字符串")
		}
		if level != "" {
			thinking["thinkingLevel"] = raw
		}
	}
	delete(thinking, "thinking_level")
	if raw := thinking["thinkingBudget"]; len(raw) > 0 && common.GetJsonType(raw) != "null" {
		var budget int64
		if err := common.Unmarshal(raw, &budget); err != nil || budget < -1 || budget >= dto.MaxTokensLimit {
			return gatewaySkillBudgetError("thinkingBudget 必须为 -1 或有效的非负整数")
		}
		// 0 表示关闭、-1 表示动态思考；不改变其模式，也不凭空加入与 level 冲突的预算。
		if limit > 1024 && budget > int64(max(uint(1024), limit-1024)) {
			thinking["thinkingBudget"] = json.RawMessage(strconv.FormatUint(uint64(max(uint(1024), limit-1024)), 10))
		}
	}
	encoded, err := common.Marshal(thinking)
	if err == nil {
		generation["thinkingConfig"] = encoded
	}
	return err
}
