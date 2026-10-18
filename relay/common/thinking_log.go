package common

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/tidwall/gjson"
)

// ThinkingLogSnapshot 只用于日志，不参与协议转换、请求改写或条件计费。
type ThinkingLogSnapshot struct {
	Effort       string
	Type         string
	BudgetTokens *int
}

func (snapshot *ThinkingLogSnapshot) DisplayEffort() string {
	if snapshot.Effort != "" {
		return snapshot.Effort
	}
	switch snapshot.Type {
	case "disabled", "adaptive":
		return snapshot.Type
	case "enabled":
		if snapshot.BudgetTokens != nil && *snapshot.BudgetTokens > 0 {
			return fmt.Sprintf("thinking:%d", *snapshot.BudgetTokens)
		}
		return "thinking"
	}
	return ""
}

func thinkingLogFromRequest(request dto.Request) *ThinkingLogSnapshot {
	var thinking *dto.Thinking
	var effort string
	switch req := request.(type) {
	case *dto.ClaudeRequest:
		if req == nil {
			return nil
		}
		thinking, effort = req.Thinking, req.GetEfforts()
	case *dto.GeneralOpenAIRequest:
		if req == nil || len(req.THINKING) == 0 {
			return nil
		}
		if err := common.Unmarshal(req.THINKING, &thinking); err != nil {
			return nil
		}
		effort = reasoningEffortFromRequest(req)
	default:
		return nil
	}
	snapshot := &ThinkingLogSnapshot{Effort: strings.TrimSpace(effort)}
	if thinking != nil {
		snapshot.Type = thinking.Type
		if thinking.BudgetTokens != nil {
			budget := *thinking.BudgetTokens
			snapshot.BudgetTokens = &budget
		}
	}
	return snapshot
}

// CaptureThinkingLogBody 读取最终出站参数；空快照用于阻止已删除参数回退到旧等级。
// 调用方必须在过滤、参数覆盖及转换完成后调用；不会修改或保存请求正文。
func (info *RelayInfo) CaptureThinkingLogBody(body []byte) {
	format := info.GetFinalRequestRelayFormat()
	thinking := gjson.GetBytes(body, "thinking")
	if info.ThinkingLog == nil && format != types.RelayFormatClaude && !thinking.IsObject() {
		return
	}
	effort, _ := extractReasoningEffortFromJSON(format, body)
	snapshot := &ThinkingLogSnapshot{Effort: effort}
	kind := thinking.Get("type")
	if kind.Type == gjson.String {
		snapshot.Type = kind.String()
	}
	budget := thinking.Get("budget_tokens")
	// OpenRouter 将 Anthropic thinking 预算转换为 reasoning.max_tokens。
	if !thinking.Exists() && format == types.RelayFormatOpenAI {
		reasoning := gjson.GetBytes(body, "reasoning")
		enabled := reasoning.Get("enabled")
		budget = reasoning.Get("max_tokens")
		if enabled.Type == gjson.False {
			snapshot.Type = "disabled"
		} else if enabled.Type == gjson.True || budget.Exists() {
			snapshot.Type = "enabled"
		}
	}
	if budget.Type == gjson.Number {
		if value, err := strconv.Atoi(budget.Raw); err == nil {
			snapshot.BudgetTokens = &value
		}
	}
	info.ThinkingLog = snapshot
}
