package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

const MCPMoneyInstructions = "向用户展示余额、累计消费和额度时，必须优先使用同一对象 money_display 中的已换算文本；不要把原始 quota 数字作为余额，也不要再次换算。money_display_settings 是本次站点展示配置。原始数值仅供精确编辑，写工具的 quota 参数仍使用内部单位。TOKENS 模式是站点选择的额度单位，不是实际模型 token 用量。"

type mcpMoneyFormatter struct {
	ctx      context.Context
	tool     MCPTool
	settings map[string]any
	mode     string
	symbol   string
	unit     decimal.Decimal
	rate     decimal.Decimal
}

// 仅接收已经过安全投影的成功响应，不改变原字段或写入参数的单位。
func formatMCPMoneyResponse(ctx context.Context, tool MCPTool, raw []byte) ([]byte, error) {
	// 只有已核对为固定业务对象的工具进入转换；价格来源、模型名等动态字典不参与。
	family := mcpCommerceResponseFamily(tool.Name)
	supported := family == "subscription" || family == "topup" || family == "redemption" || family == "benefit" || family == "affiliate"
	switch tool.Name {
	case "newapi_users_list", "newapi_users_search", "newapi_user_get", "newapi_user_edit_get", "newapi_user_permissions_get",
		"newapi_my_tokens_list", "newapi_my_tokens_search", "newapi_my_token_get", "newapi_my_token_edit_get",
		"newapi_logs_list", "newapi_usage_stats", "newapi_quota_history", "newapi_quota_by_user", "newapi_flow_history", "newapi_revenue",
		"newapi_tasks_list", "newapi_midjourney_tasks", "newapi_channels_list", "newapi_channels_search", "newapi_channel_get",
		"newapi_channel_analytics_summary", "newapi_channel_analytics_channels", "newapi_channel_analytics_models",
		"newapi_channel_analytics_trend", "newapi_channel_stability", "newapi_channel_status_codes", "newapi_channel_failures",
		"newapi_redemptions_list", "newapi_redemption_get", "newapi_affiliate_records", "newapi_affiliate_invitations",
		"newapi_affiliate_withdrawals", "newapi_benefit_activities", "newapi_benefit_report":
		supported = true
	}
	if !supported {
		return raw, nil
	}
	formatter := mcpMoneyFormatter{ctx: ctx, tool: tool}
	projected, err := formatter.visit(raw, 0)
	if err != nil || formatter.settings == nil {
		return raw, err
	}
	var object map[string]json.RawMessage
	if err = common.Unmarshal(projected, &object); err != nil || object == nil {
		return nil, errors.New("invalid money response")
	}
	object["money_display_settings"], err = common.Marshal(formatter.settings)
	if err != nil {
		return nil, err
	}
	projected, err = common.Marshal(object)
	// 展示增加后仍保留明确响应预算；超限通过原执行器提示缩小查询。
	if len(projected) > mcpResponseLimit(tool) {
		return nil, errors.New("money response too large")
	}
	return projected, err
}

func (f *mcpMoneyFormatter) visit(raw json.RawMessage, depth int) (json.RawMessage, error) {
	if depth > 32 {
		return nil, errors.New("money response nesting too deep")
	}
	switch common.GetJsonType(raw) {
	case "array":
		var values []json.RawMessage
		if err := common.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for i, value := range values {
			child, err := f.visit(value, depth+1)
			if err != nil {
				return nil, err
			}
			values[i] = child
		}
		return common.Marshal(values)
	case "object":
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		display := map[string]string{}
		for field, value := range object {
			isQuota := strings.Contains(" quota used_quota remain_quota remaining_quota total_quota charged_quota redemption_quota total_redemption_quota affiliate_source_quota credited_quota fixed_quota min_quota max_quota original_quota quota_delta undistributed_quota distributed_quota expired_unused_quota source_quota reward_quota balance_after_quota pending_quota available_quota frozen_quota risk_frozen_quota confiscated_quota withdrawn_quota transferred_quota topup_quota commission_quota clawback_quota cleared_quota clearable_quota unfrozen_quota ", " "+field+" ")
			family := mcpCommerceResponseFamily(f.tool.Name)
			isQuota = isQuota || family == "subscription" && (field == "total_amount" || field == "amount_total" || field == "amount_used")
			isQuota = isQuota || family == "benefit" && field == "balance_after"
			if isQuota && common.GetJsonType(value) == "number" {
				if f.settings == nil {
					general := operation_setting.GetGeneralSetting()
					f.mode, f.symbol = general.QuotaDisplayType, operation_setting.GetCurrencySymbol()
					unit := common.QuotaPerUnit
					if unit <= 0 || math.IsNaN(unit) || math.IsInf(unit, 0) {
						return nil, errors.New("invalid quota display unit")
					}
					rate := 1.0
					source := "configured"
					if f.mode == operation_setting.QuotaDisplayTypeCNY {
						resolved := ResolveDisplayExchangeRate(f.ctx, general.AutoUSDExchangeRate, operation_setting.USDExchangeRate)
						rate, source = resolved.Rate, resolved.Source
					} else if f.mode == operation_setting.QuotaDisplayTypeCustom {
						rate = general.CustomCurrencyExchangeRate
						if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
							rate = 1
						}
					} else if f.mode != operation_setting.QuotaDisplayTypeTokens {
						f.mode, f.symbol = operation_setting.QuotaDisplayTypeUSD, "$"
					}
					f.unit, f.rate = decimal.NewFromFloat(unit), decimal.NewFromFloat(rate)
					f.settings = map[string]any{
						"display_type": f.mode, "symbol": f.symbol,
						"quota_per_unit": f.unit.String(), "usd_to_display_rate": f.rate.String(),
						"exchange_rate_source": source, "instructions": MCPMoneyInstructions,
					}
				}
				if len(value) > 64 {
					return nil, errors.New("invalid quota amount")
				}
				quota, err := decimal.NewFromString(string(value))
				if err != nil || quota.Exponent() < -18 || quota.Exponent() > 18 {
					return nil, errors.New("invalid quota amount")
				}
				if field == "remain_quota" && string(object["unlimited_quota"]) == "true" ||
					family == "subscription" && (field == "total_amount" || field == "amount_total") && !quota.IsPositive() {
					display[field] = "Unlimited"
				} else if f.mode == operation_setting.QuotaDisplayTypeTokens {
					display[field] = quota.String() + " TOKENS"
				} else {
					amount := quota.Mul(f.rate).Div(f.unit)
					rounded := amount.Round(2)
					if rounded.IsZero() && !amount.IsZero() {
						rounded = decimal.NewFromInt(int64(amount.Sign())).Shift(-2)
					}
					display[field] = f.symbol + rounded.StringFixed(2)
				}
				continue
			}
			child, err := f.visit(value, depth+1)
			if err != nil {
				return nil, err
			}
			object[field] = child
		}
		if len(display) > 0 {
			value, err := common.Marshal(display)
			if err != nil {
				return nil, err
			}
			object["money_display"] = value
		}
		return common.Marshal(object)
	default:
		return raw, nil
	}
}
