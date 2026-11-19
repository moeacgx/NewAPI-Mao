package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// projectMCPFullCommerceResponse 只在共享层确认业务成功后调用；第二返回值表示命中目录。
// 兑换码创建密钥供首次返回，核心执行器必须在持久化幂等结果前替换为脱敏回执。
func projectMCPFullCommerceResponse(tool MCPTool, raw []byte) ([]byte, bool, error) {
	// 旧密钥已获授权的摘要工具不能被隐式升级为秘密或个人资料读取。
	switch tool.Name {
	case "newapi_invoices_list", "newapi_redemptions_list", "newapi_redemption_get", "newapi_promo_codes_list", "newapi_affiliate_records", "newapi_affiliate_invitations", "newapi_affiliate_withdrawals", "newapi_benefit_activities", "newapi_benefit_report":
		return nil, false, nil
	}
	family := mcpCommerceResponseFamily(tool.Name)
	if family == "" {
		return nil, false, nil
	}
	if len(raw) > mcpResponseLimit(tool) {
		return nil, true, errors.New("commerce response too large")
	}
	var envelope map[string]json.RawMessage
	if err := common.Unmarshal(raw, &envelope); err != nil || envelope == nil {
		return nil, true, errors.New("invalid commerce response")
	}
	out := map[string]json.RawMessage{"success": json.RawMessage(`true`)}
	data, exists := envelope["data"]
	if !exists {
		encoded, err := common.Marshal(out)
		return encoded, true, err
	}
	if tool.Name == "newapi_redemption_create" {
		var codes []string
		if common.Unmarshal(data, &codes) != nil || len(codes) == 0 || len(codes) > 100 {
			return nil, true, errors.New("invalid generated redemption codes")
		}
		for _, code := range codes {
			// common.GetUUID 生成 32 位小写十六进制，不能把任意响应正文当作新密钥。
			if len(code) != 32 || strings.Trim(code, "0123456789abcdef") != "" {
				return nil, true, errors.New("invalid generated redemption code")
			}
		}
		out["data"] = data
		encoded, err := common.Marshal(out)
		return encoded, true, err
	}
	fields := mcpCommerceResponseFields(family)
	var groupNames map[string]string
	var groupsByID map[int]string
	projected, err := projectMCPCommerceData(data, fields, "data", 0, &groupNames, &groupsByID)
	if err != nil {
		return nil, true, err
	}
	out["data"] = projected
	encoded, err := common.Marshal(out)
	if len(encoded) > mcpResponseLimit(tool) {
		return nil, true, errors.New("projected commerce response too large")
	}
	return encoded, true, err
}

func mcpCommerceResponseFamily(name string) string {
	// 精确匹配已有与本次新增工具，不能按 category 给未来未知工具开放字段。
	switch name {
	case "newapi_subscription_plans", "newapi_user_subscriptions", "newapi_subscription_plan_create", "newapi_subscription_plan_update", "newapi_subscription_plan_status_set", "newapi_subscription_bind", "newapi_user_subscription_create", "newapi_user_subscriptions_reset", "newapi_subscription_plan_reset", "newapi_user_subscription_cancel", "newapi_user_subscription_delete":
		return "subscription"
	case "newapi_topups_list", "newapi_topup_complete":
		return "topup"
	case "newapi_invoices_manage_list", "newapi_invoice_update", "newapi_invoice_delete", "newapi_invoices_delete_batch":
		return "invoice"
	case "newapi_redemptions_search", "newapi_redemption_secret_get", "newapi_redemption_create", "newapi_redemption_update", "newapi_redemption_status_set", "newapi_redemption_delete", "newapi_redemptions_delete_batch", "newapi_redemptions_delete_invalid":
		return "redemption"
	case "newapi_promo_codes_search", "newapi_promo_code_get", "newapi_promo_code_create", "newapi_promo_code_update", "newapi_promo_code_status_set", "newapi_promo_code_delete", "newapi_promo_codes_delete_batch", "newapi_promo_codes_delete_invalid":
		return "promo"
	case "newapi_benefit_activities_manage", "newapi_benefit_report_manage", "newapi_benefit_activity_get", "newapi_benefit_activity_create", "newapi_benefit_activity_update", "newapi_benefit_activity_delete", "newapi_benefit_activities_delete_batch", "newapi_benefit_activity_publish", "newapi_benefit_activity_pause", "newapi_benefit_activity_resume", "newapi_benefit_activity_end", "newapi_benefit_activity_terminate", "newapi_benefit_vouchers", "newapi_benefit_voucher_ledger", "newapi_benefit_voucher_void", "newapi_benefit_vouchers_void_batch":
		return "benefit"
	case "newapi_affiliate_records_manage", "newapi_affiliate_invitations_manage", "newapi_affiliate_withdrawals_manage", "newapi_affiliate_applications", "newapi_affiliate_application_approve", "newapi_affiliate_application_reject", "newapi_affiliate_application_revoke", "newapi_affiliate_withdrawal_approve", "newapi_affiliate_withdrawal_reject", "newapi_affiliate_withdrawal_paid", "newapi_affiliate_inviter_bind", "newapi_affiliate_inviter_unbind", "newapi_affiliate_access_grant", "newapi_affiliate_risk_users", "newapi_affiliate_risk_preview", "newapi_affiliate_risk_apply", "newapi_affiliate_risk_remove", "newapi_affiliate_fraud_alerts", "newapi_affiliate_fraud_scan", "newapi_affiliate_fraud_scan_deep", "newapi_affiliate_fraud_unbind", "newapi_affiliate_fraud_clawback", "newapi_affiliate_fraud_dismiss", "newapi_affiliate_fraud_delete":
		return "affiliate"
	case "newapi_game_predictions", "newapi_game_prediction_create", "newapi_game_prediction_answer_set", "newapi_game_prediction_settle":
		return "game"
	case "newapi_okpay_rate_preview", "newapi_waffo_pancake_catalog", "newapi_waffo_pancake_pair_create", "newapi_waffo_pancake_save", "newapi_waffo_pancake_subscription_product_create", "newapi_waffo_pancake_subscription_product_options":
		return "payment"
	default:
		return ""
	}
}

// 字段同时限定值类型。自由文本只在明确编辑字段保留，不对所有字符串开放。
func mcpCommerceResponseFields(family string) map[string]string {
	fields := map[string]string{}
	sets := map[string]string{
		"number":    "id user_id total page page_size created_at updated_at count",
		"string":    "username display_name name reason",
		"container": "data items skipped",
		"numbers":   "deleted_ids updated_ids",
	}
	switch family {
	case "subscription":
		sets["number"] += " plan_id price_amount duration_value custom_seconds sort_order max_purchase_per_user total_amount quota_reset_custom_seconds amount_total amount_used start_time end_time last_reset_time next_reset_time matched_count reset_count user_count"
		sets["string"] += " title subtitle currency duration_unit stripe_price_id creem_product_id waffo_pancake_product_id upgrade_group downgrade_group prev_user_group quota_reset_period status source plan_title"
		sets["boolean"] = "enabled allow_balance_pay allow_wallet_overflow advance_reset_time"
		sets["container"] += " plan subscription"
		sets["numbers"] += " affected_user_ids"
	case "topup":
		sets["number"] += " amount money original_money discount_money actual_money paid_amount_cny promo_code_id affiliate_source_quota credited_quota invoice_base_amount invoice_fee_amount create_time complete_time"
		sets["string"] += " trade_no payment_method payment_provider provider_order_id provider_amount provider_currency status invoice_status"
		sets["boolean"] = "invoice_required invoice_discount_disabled"
	case "invoice":
		sets["number"] += " base_amount fee_amount total_amount payment_amount_minor paid_time create_time update_time issued_time invoice_id"
		sets["string"] += " source_type source_id payment_method payment_provider payment_status provider_order_id provider_amount provider_currency invoice_type invoice_kind title tax_no email phone remark download_url admin_remark cancel_reason status trade_no currency"
		sets["container"] += " orders"
	case "redemption":
		sets["number"] += " status quota redeemed_count max_redeem_count created_time redeemed_time expired_time used_user_id"
		sets["string"] += " key"
	case "promo":
		sets["number"] += " status discount_value max_redeem_count redeemed_count reserved_count created_time updated_time expired_time"
		sets["string"] += " code discount_type subscription_plan_ids"
		sets["boolean"] = "applies_to_topup applies_to_all_subscription"
	case "benefit":
		sets["number"] += " group_id total_amount total_amount_cents total_quota total_count fixed_amount fixed_amount_cents fixed_quota min_amount min_amount_cents min_quota max_amount max_amount_cents max_quota claim_paid_threshold claim_paid_threshold_cents personal_valid_hours starts_at ends_at published_at created_by updated_by terminated_by terminated_at activity_id share_id original_amount original_amount_cents original_quota remaining_quota used_quota claimed_at expires_at voided_at voucher_id log_id quota_delta balance_after undistributed_quota distributed_quota expired_unused_quota distributed_count used_count expired_count"
		sets["string"] += " description status amount_mode amount_display_type amount_display_type_snapshot amount_display_rate_snapshot quota_per_unit_snapshot group_code_snapshot group_name_snapshot terminate_mode terminate_reason void_reason activity_name request_id type"
	case "affiliate":
		sets["number"] += " inviter_id invitee_id previous_inviter_id level source_quota reward_quota ratio balance_after_quota available_time settled_time pending_quota available_quota frozen_quota risk_frozen_quota confiscated_quota withdrawn_quota transferred_quota total_quota quota display_amount payout_fiat_amount payout_rate payout_rate_at admin_id approved_time paid_time rejected_time agreed_at topup_count topup_quota recharge_amount last_topup_time commission_quota matched_inviter_count matched_invitee_count invitee_status invitee_created_at plan_id redemption_id original_amount discount_amount paid_amount shared_ip_count clawback_quota detected_at resolved_at alert_count invitee_count latest_detected_at cleared_quota removed_by removed_at direct_invitee_count restorable_invitee_count clearable_quota detached_count rejected_withdrawals restored_invitees unfrozen_quota new_alerts"
		sets["string"] += " source_type source_id display_currency payout_fiat_currency payout_rate_source method status admin_remark rejected_reason inviter_username inviter_name inviter_aff_code invitee_username invitee_name title plan_title redemption_name payment_provider payment_method aff_code resolved_action remove_remark"
		sets["boolean"] = "payout_rate_fallback updated freeze_assets block_invite_code detached_invitees"
		sets["container"] += " inviter invitee detail balance summary user generated_topup active_risk risk_user alerts payout_details"
		// 收款详情是完成管理员人工打款所需的明确字段，不开放原始 payout_snapshot。
		sets["string"] += " usdt_address usdt_chain alipay_account alipay_name alipay_qr_path wechat_account wechat_name wechat_qr_path"
		sets["string-or-strings"] = "shared_ips"
		sets["string-or-number"] = "status"
	case "game":
		sets["number"] += " prediction_id close_time settle_time answer_option_id answer_set_by answered_at settled_at settled_by total_pool winning_pool total_payout total_fee winner_count created_by index pool_amount bet_count settled_bets refunded_bets"
		sets["string"] += " title description status judge_mode"
		sets["container"] += " options prediction"
	case "payment":
		sets["string"] += " id status store_id store_name product_id product_name raw_rate adjusted_rate source side adjustment_type adjustment"
		sets["number"] += " tier"
		sets["boolean"] = "prodEnabled orphan_store"
		sets["container"] += " stores products onetimeProducts"
	}
	for typ, names := range sets {
		for _, name := range strings.Fields(names) {
			fields[name] = typ
		}
	}
	// 同名字段在不同层级有已知的混合类型，覆盖顺序不能依赖 map 遍历。
	if family == "affiliate" {
		fields["status"] = "string-or-number"
	}
	if family == "payment" {
		fields["id"] = "string"
	}
	return fields
}

func projectMCPCommerceData(raw json.RawMessage, fields map[string]string, field string, depth int, groupNames *map[string]string, groupsByID *map[int]string) (json.RawMessage, error) {
	if depth > 24 {
		return nil, errors.New("commerce response nesting too deep")
	}
	if common.GetJsonType(raw) == "null" {
		return raw, nil
	}
	kind := fields[field]
	valueType := common.GetJsonType(raw)
	if kind == "container" && valueType == "object" {
		var object map[string]json.RawMessage
		if err := common.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		out := map[string]json.RawMessage{}
		for key, value := range object {
			if _, exists := fields[key]; !exists {
				continue
			}
			child, err := projectMCPCommerceData(value, fields, key, depth+1, groupNames, groupsByID)
			if err != nil {
				return nil, err
			}
			out[key] = child
		}
		// 分组 code 保留为编辑标识，另给当前显示名称；同一响应最多各批量查询一次。
		for _, key := range []string{"upgrade_group", "downgrade_group", "prev_user_group"} {
			var code string
			if value, exists := out[key]; !exists || common.Unmarshal(value, &code) != nil || code == "" {
				continue
			}
			if *groupNames == nil {
				if model.DB == nil {
					return nil, errors.New("group names unavailable")
				}
				resolved, err := model.GetGroupDisplayNameMap()
				if err != nil {
					return nil, errors.New("group names unavailable")
				}
				*groupNames = resolved
			}
			out[key+"_name"], _ = common.Marshal(model.FormatGroupDisplayNames(code, *groupNames))
		}
		var groupID int
		if value, exists := out["group_id"]; exists && common.Unmarshal(value, &groupID) == nil && groupID > 0 {
			if *groupsByID == nil {
				if model.DB == nil {
					return nil, errors.New("group names unavailable")
				}
				var groups []model.Group
				if err := model.DB.Select("id", "name").Find(&groups).Error; err != nil {
					return nil, errors.New("group names unavailable")
				}
				*groupsByID = make(map[int]string, len(groups))
				for _, group := range groups {
					(*groupsByID)[group.Id] = group.Name
				}
			}
			name := (*groupsByID)[groupID]
			if name == "" {
				_ = common.Unmarshal(out["group_name_snapshot"], &name)
			}
			if name == "" {
				_ = common.Unmarshal(out["group_code_snapshot"], &name)
			}
			out["group_name"], _ = common.Marshal(name)
		}
		return common.Marshal(out)
	}
	if valueType == "array" && (kind == "container" || kind == "numbers" || kind == "string-or-strings") {
		var items []json.RawMessage
		if err := common.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		for i, item := range items {
			if kind == "numbers" && common.GetJsonType(item) != "number" || kind == "string-or-strings" && common.GetJsonType(item) != "string" {
				return nil, errors.New("invalid commerce array item")
			}
			if kind == "container" {
				child, err := projectMCPCommerceData(item, fields, field, depth+1, groupNames, groupsByID)
				if err != nil {
					return nil, err
				}
				items[i] = child
			}
		}
		return common.Marshal(items)
	}
	if kind == valueType && (kind == "string" || kind == "number" || kind == "boolean") {
		return raw, nil
	}
	if kind == "string-or-number" && (valueType == "string" || valueType == "number") || kind == "string-or-strings" && valueType == "string" {
		return raw, nil
	}
	// 删除接口返回数量或 ID 数组；普通字符串不能冒充 data 容器返回。
	if field == "data" && valueType == "number" {
		return raw, nil
	}
	return nil, errors.New("invalid commerce response field type")
}
