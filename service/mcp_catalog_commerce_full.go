package service

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/authz"
)

// 商务目录只描述已有管理路由，不能绕过控制器的交易、合规与状态机校验。
func mcpFullCommerceTools() []MCPTool {
	tools := []MCPTool{
		{Name: "newapi_invoices_manage_list", Category: "billing", Path: "/api/user/invoice", Description: "敏感管理读取：分页查询发票完整开票资料、税号、联系方式、管理员备注及文件链接。必须单独授权，旧元数据工具不会返回这些资料。"},
		{Name: "newapi_redemption_secret_get", Category: "marketing", Path: "/api/redemption/:id", Description: "敏感管理读取：按 ID 返回完整兑换凭据 key 及编辑字段，必须单独授权；旧列表和详情保持脱敏。"},
		{Name: "newapi_affiliate_records_manage", Category: "affiliate", Path: "/api/affiliate/admin/records", Write: true, Description: "完整管理返佣记录与来源；调用前会结算所有成熟返佣，必须 request_id。邀请信息可含商业邀请码，需单独授权。"},
		{Name: "newapi_affiliate_invitations_manage", Category: "affiliate", Path: "/api/affiliate/admin/invitations", Description: "敏感管理读取：分页查询邀请关系、商业邀请码及返佣统计；需单独授权。"},
		{Name: "newapi_affiliate_withdrawals_manage", Category: "affiliate", Path: "/api/affiliate/admin/withdrawals", Description: "敏感管理读取：分页查询提现详情及实际收款账户、二维码路径，供人工打款核验；需单独授权。"},
		{Name: "newapi_benefit_activities_manage", Category: "marketing", Path: "/api/benefit/admin/activities", Description: "敏感管理读取：分页查询福利活动完整编辑字段、说明与终止原因；需单独授权，旧活动清单保持摘要。"},
		{Name: "newapi_benefit_report_manage", Category: "marketing", Path: "/api/benefit/admin/activities/:id/report", Write: true, Description: "查询福利活动完整额度及份数统计；宿主会处理过期券，必须 request_id。"},
		{Name: "newapi_subscription_plan_create", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/plans", Description: "创建订阅计划。body.plan 为完整配置，total_amount 使用内部 quota，0 表示不限；需已确认支付合规。"},
		{Name: "newapi_subscription_plan_update", Category: "billing", Method: http.MethodPut, Path: "/api/subscription/admin/plans/:id", Description: "完整保存订阅计划；先读取现有计划，所有编辑字段均需提交，升级和降级分组使用 code。"},
		{Name: "newapi_subscription_plan_status_set", Category: "billing", Method: http.MethodPatch, Path: "/api/subscription/admin/plans/:id", Description: "启停订阅计划。宿主没有删除计划接口；停用不取消已有订阅。"},
		{Name: "newapi_subscription_bind", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/bind", Description: "给指定用户绑定订阅计划，不发起支付。"},
		{Name: "newapi_user_subscription_create", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/users/:id/subscriptions", Description: "为路径指定用户创建订阅权益，不发起支付。"},
		{Name: "newapi_user_subscriptions_reset", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/users/:id/subscriptions/reset", Description: "重置指定用户某计划的有效订阅额度。advance_reset_time 显式决定是否推进下次重置时间。"},
		{Name: "newapi_subscription_plan_reset", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/plans/:id/subscriptions/reset", Description: "重置计划下所有有效用户订阅额度，影响多用户。advance_reset_time 显式决定是否推进下次重置时间。"},
		{Name: "newapi_user_subscription_cancel", Category: "billing", Method: http.MethodPost, Path: "/api/subscription/admin/user_subscriptions/:id/invalidate", Description: "立即取消指定用户订阅，按宿主规则恢复分组。路径 id 是订阅 ID。"},
		{Name: "newapi_user_subscription_delete", Category: "billing", Method: http.MethodDelete, Path: "/api/subscription/admin/user_subscriptions/:id", Description: "永久删除指定用户订阅，路径 id 是订阅 ID。"},
		{Name: "newapi_topup_complete", Category: "billing", Method: http.MethodPost, Path: "/api/user/topup/complete", Description: "按 trade_no 管理员补单，可能增加余额和生成返佣。先核对订单真实付款状态。"},
		{Name: "newapi_invoice_update", Category: "billing", Method: http.MethodPut, Path: "/api/user/invoice/:id", Description: "更新发票状态、下载链接及管理员备注；关闭会释放开票订单占用，付款与退款状态仍由宿主校验。"},
		{Name: "newapi_invoice_delete", Category: "billing", Method: http.MethodDelete, Path: "/api/user/invoice/:id", Description: "删除指定发票申请；宿主拒绝未满足删除条件的记录。"},
		{Name: "newapi_invoices_delete_batch", Category: "billing", Method: http.MethodPost, Path: "/api/user/invoice/batch", Description: "批量删除发票申请，返回删除数量。"},
		{Name: "newapi_redemptions_search", Category: "marketing", Path: "/api/redemption/search", Description: "按名称关键字和状态搜索兑换码，返回管理所需兑换凭据；不要转发结果到公开日志。"},
		{Name: "newapi_redemption_create", Category: "marketing", Method: http.MethodPost, Path: "/api/redemption/", Description: "生成 1–100 个兑换码。quota 使用内部额度，首次返回新密钥；重试只返回脱敏回执，不会重新生成。"},
		{Name: "newapi_redemption_update", Category: "marketing", Method: http.MethodPut, Path: "/api/redemption/", Description: "完整更新兑换码名称、额度、到期时间和最大兑换人数；状态使用独立工具。"},
		{Name: "newapi_redemption_status_set", Category: "marketing", Method: http.MethodPut, Path: "/api/redemption/", Description: "通过 query.status_only=true 修改兑换码状态，1 启用、2 禁用。"},
		{Name: "newapi_redemption_delete", Category: "marketing", Method: http.MethodDelete, Path: "/api/redemption/:id", Description: "删除指定兑换码。"},
		{Name: "newapi_redemptions_delete_batch", Category: "marketing", Method: http.MethodDelete, Path: "/api/redemption/batch", Description: "按 body.ids 批量删除兑换码，返回 deleted_ids/skipped。"},
		{Name: "newapi_redemptions_delete_invalid", Category: "marketing", Method: http.MethodDelete, Path: "/api/redemption/invalid", Description: "清理已失效兑换码，影响全部符合宿主条件的记录。"},
		{Name: "newapi_promo_codes_search", Category: "marketing", Path: "/api/promo_code/search", Description: "搜索优惠码及其适用范围、使用量和商业凭据。"},
		{Name: "newapi_promo_code_get", Category: "marketing", Path: "/api/promo_code/:id", Description: "读取优惠码完整编辑字段，包括 code 商业凭据。"},
		{Name: "newapi_promo_code_create", Category: "marketing", Method: http.MethodPost, Path: "/api/promo_code/", Description: "创建优惠码。code 由操作者指定；percent 的 discount_value 为百分数，fixed 为最小货币单位；适用范围至少启用一项。"},
		{Name: "newapi_promo_code_update", Category: "marketing", Method: http.MethodPut, Path: "/api/promo_code/", Description: "完整保存优惠码，先读取详情；subscription_plan_ids 是逗号分隔的正整数 ID 字符串。"},
		{Name: "newapi_promo_code_status_set", Category: "marketing", Method: http.MethodPut, Path: "/api/promo_code/", Description: "通过 query.status_only=true 修改优惠码状态，1 启用、2 禁用。"},
		{Name: "newapi_promo_code_delete", Category: "marketing", Method: http.MethodDelete, Path: "/api/promo_code/:id", Description: "删除指定优惠码，受未结算使用预留限制。"},
		{Name: "newapi_promo_codes_delete_batch", Category: "marketing", Method: http.MethodDelete, Path: "/api/promo_code/batch", Description: "按 body.ids 批量删除优惠码，返回 deleted_ids/skipped。"},
		{Name: "newapi_promo_codes_delete_invalid", Category: "marketing", Method: http.MethodDelete, Path: "/api/promo_code/invalid", Description: "清理宿主判定失效且允许删除的优惠码。"},
		{Name: "newapi_benefit_activity_get", Category: "marketing", Path: "/api/benefit/admin/activities/:id", Description: "查询福利活动完整配置，金额按 amount_display_type 返回。"},
		{Name: "newapi_benefit_activity_create", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities", Description: "创建福利活动草稿。group_id 是稳定分组 ID；金额单位必须匹配当前 amount_display_type，固定面额乘份数必须等于总预算。"},
		{Name: "newapi_benefit_activity_update", Category: "marketing", Method: http.MethodPut, Path: "/api/benefit/admin/activities/:id", Description: "完整保存福利活动；草稿可改预算及范围，发布后宿主仅更新名称和说明。"},
		{Name: "newapi_benefit_activity_delete", Category: "marketing", Method: http.MethodDelete, Path: "/api/benefit/admin/activities/:id", Description: "删除符合条件的福利活动，返回 deleted_ids/skipped。"},
		{Name: "newapi_benefit_activities_delete_batch", Category: "marketing", Method: http.MethodDelete, Path: "/api/benefit/admin/activities/batch", Description: "按 body.ids 批量删除符合条件的福利活动。"},
		{Name: "newapi_benefit_activity_publish", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities/:id/publish", Description: "发布草稿福利活动并生成券份额。"},
		{Name: "newapi_benefit_activity_pause", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities/:id/pause", Description: "暂停福利活动领取。"},
		{Name: "newapi_benefit_activity_resume", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities/:id/resume", Description: "恢复暂停的福利活动。"},
		{Name: "newapi_benefit_activity_end", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities/:id/end", Description: "结束福利活动领取，已领取券沿用宿主有效期规则。"},
		{Name: "newapi_benefit_activity_terminate", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/activities/:id/terminate", Description: "强制终止福利活动，mode=unused 或 all，必须 confirm=true 并说明原因。"},
		{Name: "newapi_benefit_vouchers", Category: "marketing", Path: "/api/benefit/admin/activities/:id/vouchers", Description: "分页查询活动下用户福利券，支持 keyword、status。"},
		{Name: "newapi_benefit_voucher_ledger", Category: "marketing", Path: "/api/benefit/admin/vouchers/:id/ledger", Description: "查询指定福利券额度流水，排除原始 metadata。"},
		{Name: "newapi_benefit_voucher_void", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/vouchers/:id/void", Description: "作废单张福利券，必须 confirm=true 并说明原因。"},
		{Name: "newapi_benefit_vouchers_void_batch", Category: "marketing", Method: http.MethodPost, Path: "/api/benefit/admin/vouchers/batch-void", Description: "批量作废福利券，必须 confirm=true 并说明原因。"},
		{Name: "newapi_affiliate_applications", Category: "affiliate", Path: "/api/affiliate/admin/applications", Description: "分页查询返佣资格申请及审核理由。"},
		{Name: "newapi_affiliate_application_approve", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/applications/:id/approve", Description: "批准返佣资格申请，可填写 remark。"},
		{Name: "newapi_affiliate_application_reject", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/applications/:id/reject", Description: "拒绝返佣资格申请，填写 reason。"},
		{Name: "newapi_affiliate_application_revoke", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/applications/:id/revoke", Description: "撤销已批准的返佣资格。"},
		{Name: "newapi_affiliate_withdrawal_approve", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/withdrawals/:id/approve", Description: "批准提现申请，填写管理员备注；不自动向外部支付服务打款。"},
		{Name: "newapi_affiliate_withdrawal_reject", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/withdrawals/:id/reject", Description: "拒绝提现申请，按宿主规则解冻额度。"},
		{Name: "newapi_affiliate_withdrawal_paid", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/withdrawals/:id/paid", Description: "确认已人工打款并完成提现账务；先核对实际付款，不调用外部支付。"},
		{Name: "newapi_affiliate_inviter_bind", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/bind-inviter", Description: "按目标 user_id 与邀请人 aff_code 绑定邀请关系；force=true 允许覆盖旧关系。"},
		{Name: "newapi_affiliate_inviter_unbind", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/unbind-inviter", Description: "解绑目标 user_id 的邀请关系。"},
		{Name: "newapi_affiliate_access_grant", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/grant-access", Description: "向目标 user_id 授予返佣邀请资格。"},
		{Name: "newapi_affiliate_risk_users", Category: "affiliate", Path: "/api/affiliate/admin/risk-users", Description: "查询返佣风控用户及风险标记。"},
		{Name: "newapi_affiliate_risk_preview", Category: "affiliate", Path: "/api/affiliate/admin/risk-users/:user_id/preview", Description: "预览风控用户余额、邀请人数量及可清算额度，不修改数据。"},
		{Name: "newapi_affiliate_risk_apply", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/risk-users/:user_id/apply", Description: "施加返佣风控：显式选择冻结资产、封禁邀请码、解绑下级及清算资产。执行前先预览。"},
		{Name: "newapi_affiliate_risk_remove", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/risk-users/:user_id/remove", Description: "移除返佣风控，可选择恢复被解绑下级；已清算资产不自动返还。"},
		{Name: "newapi_affiliate_fraud_alerts", Category: "affiliate", Path: "/api/affiliate/admin/fraud-alerts", Description: "分页查询返佣欺诈告警。flat=true 返回逐条告警，否则按邀请人分组。"},
		{Name: "newapi_affiliate_fraud_scan", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/fraud-alerts/scan", Description: "扫描并写入返佣欺诈告警，query.days 为回溯天数，默认 30。"},
		{Name: "newapi_affiliate_fraud_scan_deep", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/fraud-alerts/scan-deep", Description: "执行深度返佣欺诈扫描并写入告警，可能较耗时。"},
		{Name: "newapi_affiliate_fraud_unbind", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/fraud-alerts/:id/unbind", Description: "根据欺诈告警解绑邀请关系，不追回历史佣金。"},
		{Name: "newapi_affiliate_fraud_clawback", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/fraud-alerts/:id/clawback", Description: "根据欺诈告警解绑邀请关系并追回佣金。"},
		{Name: "newapi_affiliate_fraud_dismiss", Category: "affiliate", Method: http.MethodPost, Path: "/api/affiliate/admin/fraud-alerts/:id/dismiss", Description: "忽略欺诈告警并保存处理备注。"},
		{Name: "newapi_affiliate_fraud_delete", Category: "affiliate", Method: http.MethodDelete, Path: "/api/affiliate/admin/fraud-alerts/:id", Description: "删除欺诈告警记录。"},
		{Name: "newapi_game_predictions", Category: "game", Path: "/api/game/admin/predictions", Permission: authz.GameAdminRead, Description: "分页查询管理员竞猜列表，包含答案与结算统计。"},
		{Name: "newapi_game_prediction_create", Category: "game", Method: http.MethodPost, Path: "/api/game/admin/predictions", Permission: authz.GameAdminWrite, Description: "创建人工判定竞猜，options 必须恰好两个非空选项，close_time/settle_time 使用 Unix 秒。"},
		{Name: "newapi_game_prediction_answer_set", Category: "game", Method: http.MethodPut, Path: "/api/game/admin/predictions/:id/answer", Permission: authz.GameAdminWrite, Description: "按 option_id 设置竞猜答案，先读取管理员竞猜列表取得选项 ID。"},
		{Name: "newapi_game_prediction_settle", Category: "game", Method: http.MethodPost, Path: "/api/game/admin/predictions/:id/settle", Permission: authz.GameAdminWrite, Description: "结算竞猜并改变游戏钱包余额；受截止时间和答案状态限制。"},
		{Name: "newapi_okpay_rate_preview", Category: "billing", Path: "/api/option/okpay/rate-preview", MinRole: common.RoleRootUser, Write: true, Description: "主动请求 OKPay 汇率来源，返回原始与调整后汇率；只读外部报价，但需 request_id 管理主动调用。"},
		{Name: "newapi_waffo_pancake_catalog", Category: "billing", Path: "/api/option/waffo-pancake/catalog", MinRole: common.RoleRootUser, Write: true, Description: "主动查询 Waffo 店铺和一次性产品目录。省略 query 凭据使用已保存配置；临时凭据只用于本次验证，不保存。"},
		{Name: "newapi_waffo_pancake_pair_create", Category: "billing", Method: http.MethodPost, Path: "/api/option/waffo-pancake/pair", MinRole: common.RoleRootUser, Description: "在 Waffo 创建店铺与一次性充值产品并发布，具有外部副作用；不会保存本地配置，失败或不确定时先查询目录，不换 request_id 重试。"},
		{Name: "newapi_waffo_pancake_save", Category: "billing", Method: http.MethodPost, Path: "/api/option/waffo-pancake/save", MinRole: common.RoleRootUser, Description: "原子保存 Waffo 商户、私钥、回跳地址、店铺与产品配置；private_key 为空表示保留现有私钥，不返回私钥。"},
		{Name: "newapi_waffo_pancake_subscription_product_create", Category: "billing", Method: http.MethodPost, Path: "/api/option/waffo-pancake/subscription-product", MinRole: common.RoleRootUser, Description: "使用已保存凭据和店铺创建并发布订阅计划使用的一次性产品。amount 是美元十进制字符串；不会自动保存订阅计划。"},
		{Name: "newapi_waffo_pancake_subscription_product_options", Category: "billing", Path: "/api/option/waffo-pancake/subscription-product-options", MinRole: common.RoleRootUser, Write: true, Description: "主动查询已保存 Waffo 店铺下可用于订阅计划的一次性产品。"},
	}
	for i := range tools {
		tool := &tools[i]
		if tool.MinRole == 0 {
			tool.MinRole = common.RoleAdminUser
		}
		tool.TypedArguments = true
		if tool.Name == "newapi_waffo_pancake_catalog" {
			tool.StringQueryIDs = []string{"merchant_id"}
		}
		if tool.Method == "" {
			tool.Method = http.MethodGet
		}
		tool.Write = tool.Write || tool.Method != http.MethodGet || tool.Name == "newapi_affiliate_risk_preview"
		// 预览会惰性初始化余额并同步旧返佣字段，也必须通过写幂等门禁。
		if tool.Name == "newapi_affiliate_risk_preview" {
			tool.Description = "预览风控用户余额、邀请人数量及可清算额度；宿主可能初始化余额并同步旧字段，因此要求 request_id。"
		}
		switch tool.Name {
		case "newapi_redemption_create", "newapi_redemption_update", "newapi_redemption_status_set", "newapi_promo_code_create", "newapi_promo_code_update", "newapi_promo_code_status_set":
			tool.TransientResult = true
		}
		tool.InputSchema = mcpToolInputSchema(*tool)
		properties := tool.InputSchema["properties"].(map[string]any)
		properties["query"] = mcpCommerceQuerySchema(tool.Name)
		if tool.Write {
			// DELETE 批量接口也读 JSON body，不能依赖共享构造器只处理 POST/PUT。
			properties["body"] = mcpCommerceBodySchema(tool.Name)
			required := []string{"request_id", "body"}
			if _, ok := properties["path"]; ok {
				required = append(required, "path")
			}
			if strings.HasSuffix(tool.Name, "_status_set") && tool.Category == "marketing" {
				required = append(required, "query")
			}
			tool.InputSchema["required"] = required
		}
		tool.Annotations = map[string]bool{"readOnlyHint": !tool.Write, "destructiveHint": tool.Write, "idempotentHint": true, "openWorldHint": tool.MinRole == common.RoleRootUser && tool.Name != "newapi_waffo_pancake_save"}
	}
	return tools
}

func mcpCommerceObject(fields map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
}

func mcpCommerceString(min, max int) map[string]any {
	return map[string]any{"type": "string", "minLength": min, "maxLength": max}
}

func mcpCommerceInteger(min, max int64) map[string]any {
	return map[string]any{"type": "integer", "minimum": min, "maximum": max}
}

func mcpCommerceEnum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func mcpCommerceQuerySchema(name string) map[string]any {
	fields := map[string]any{}
	switch name {
	case "newapi_redemptions_search", "newapi_promo_codes_search", "newapi_benefit_vouchers", "newapi_affiliate_applications", "newapi_affiliate_risk_users", "newapi_affiliate_fraud_alerts", "newapi_game_predictions", "newapi_invoices_manage_list", "newapi_affiliate_records_manage", "newapi_affiliate_invitations_manage", "newapi_affiliate_withdrawals_manage", "newapi_benefit_activities_manage":
		fields["p"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,4}$", "maxLength": 5}
		fields["page_size"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,2}$", "maxLength": 3}
	}
	switch name {
	case "newapi_redemptions_search", "newapi_promo_codes_search", "newapi_benefit_vouchers", "newapi_affiliate_risk_users", "newapi_affiliate_fraud_alerts", "newapi_affiliate_records_manage", "newapi_affiliate_invitations_manage":
		fields["keyword"] = mcpCommerceString(0, 256)
	}
	switch name {
	case "newapi_redemptions_search", "newapi_benefit_vouchers", "newapi_affiliate_applications", "newapi_affiliate_risk_users", "newapi_affiliate_fraud_alerts", "newapi_invoices_manage_list", "newapi_affiliate_records_manage", "newapi_affiliate_withdrawals_manage":
		fields["status"] = mcpCommerceString(0, 32)
	case "newapi_redemption_status_set", "newapi_promo_code_status_set":
		fields["status_only"] = mcpCommerceEnum("true")
		return mcpCommerceObject(fields, "status_only")
	case "newapi_affiliate_fraud_scan", "newapi_affiliate_fraud_scan_deep":
		fields["days"] = map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,2}$", "description": "回溯 1–999 天，省略默认 30。"}
	}
	if name == "newapi_affiliate_fraud_alerts" {
		fields["flat"] = mcpCommerceEnum("true", "false")
		fields["ip"] = mcpCommerceString(0, 64)
	}
	if name == "newapi_waffo_pancake_catalog" {
		fields["merchant_id"] = mcpCommerceString(0, 128)
		fields["private_key"] = mcpCommerceString(0, 2048)
	}
	if name == "newapi_affiliate_records_manage" {
		fields["source_type"] = mcpCommerceEnum("", "topup", "subscription", "redemption")
	}
	return mcpCommerceObject(fields)
}

func mcpCommerceBodySchema(name string) map[string]any {
	id := mcpCommerceInteger(1, 2147483647)
	quota := mcpCommerceInteger(0, 2147483647)
	timestamp := mcpCommerceInteger(0, 253402300799)
	boolean := map[string]any{"type": "boolean"}
	remark := mcpCommerceString(0, 500)
	reason := map[string]any{"type": "string", "minLength": 1, "maxLength": 500, "pattern": "\\S"}
	ids := map[string]any{"type": "array", "items": id, "minItems": 1, "maxItems": 500}
	confirm := map[string]any{"type": "boolean", "enum": []bool{true}}
	switch name {
	case "newapi_subscription_plan_create", "newapi_subscription_plan_update":
		plan := mcpCommerceObject(map[string]any{
			"title": mcpCommerceString(1, 128), "subtitle": mcpCommerceString(0, 255),
			"price_amount": map[string]any{"type": "number", "minimum": 0, "maximum": 9999}, "currency": mcpCommerceEnum("USD", "CNY"),
			"duration_unit": mcpCommerceEnum("year", "month", "day", "hour", "custom"), "duration_value": mcpCommerceInteger(0, 10000), "custom_seconds": mcpCommerceInteger(0, 2147483647),
			"enabled": boolean, "sort_order": mcpCommerceInteger(-2147483648, 2147483647), "allow_balance_pay": boolean, "allow_wallet_overflow": boolean,
			"stripe_price_id": mcpCommerceString(0, 128), "creem_product_id": mcpCommerceString(0, 128), "waffo_pancake_product_id": mcpCommerceString(0, 128),
			"max_purchase_per_user": quota, "total_amount": quota, "upgrade_group": mcpCommerceString(0, 64), "downgrade_group": mcpCommerceString(0, 64),
			"quota_reset_period": mcpCommerceEnum("never", "daily", "weekly", "monthly", "custom"), "quota_reset_custom_seconds": mcpCommerceInteger(0, 2147483647),
		}, "title", "subtitle", "price_amount", "currency", "duration_unit", "duration_value", "custom_seconds", "enabled", "sort_order", "allow_balance_pay", "allow_wallet_overflow", "stripe_price_id", "creem_product_id", "waffo_pancake_product_id", "max_purchase_per_user", "total_amount", "upgrade_group", "downgrade_group", "quota_reset_period", "quota_reset_custom_seconds")
		return mcpCommerceObject(map[string]any{"plan": plan}, "plan")
	case "newapi_subscription_plan_status_set":
		return mcpCommerceObject(map[string]any{"enabled": boolean}, "enabled")
	case "newapi_subscription_bind":
		return mcpCommerceObject(map[string]any{"user_id": id, "plan_id": id}, "user_id", "plan_id")
	case "newapi_user_subscription_create":
		return mcpCommerceObject(map[string]any{"plan_id": id}, "plan_id")
	case "newapi_user_subscriptions_reset":
		return mcpCommerceObject(map[string]any{"plan_id": id, "advance_reset_time": boolean}, "plan_id", "advance_reset_time")
	case "newapi_subscription_plan_reset":
		return mcpCommerceObject(map[string]any{"advance_reset_time": boolean}, "advance_reset_time")
	case "newapi_topup_complete":
		return mcpCommerceObject(map[string]any{"trade_no": mcpCommerceString(1, 255)}, "trade_no")
	case "newapi_invoice_update":
		return mcpCommerceObject(map[string]any{"download_url": mcpCommerceString(0, 2048), "status": mcpCommerceEnum("pending", "issued", "closed"), "admin_remark": remark}, "download_url", "status", "admin_remark")
	case "newapi_invoices_delete_batch":
		ids["maxItems"] = 100
		return mcpCommerceObject(map[string]any{"ids": ids}, "ids")
	case "newapi_redemptions_delete_batch", "newapi_promo_codes_delete_batch", "newapi_benefit_activities_delete_batch":
		return mcpCommerceObject(map[string]any{"ids": ids}, "ids")
	case "newapi_redemption_create", "newapi_redemption_update":
		fields := map[string]any{"name": mcpCommerceString(1, 20), "quota": quota, "expired_time": timestamp, "max_redeem_count": mcpCommerceInteger(1, 100000)}
		required := []string{"name", "quota", "expired_time", "max_redeem_count"}
		if name == "newapi_redemption_create" {
			fields["count"] = mcpCommerceInteger(1, 100)
			required = append(required, "count")
		} else {
			fields["id"] = id
			required = append(required, "id")
		}
		return mcpCommerceObject(fields, required...)
	case "newapi_redemption_status_set", "newapi_promo_code_status_set":
		return mcpCommerceObject(map[string]any{"id": id, "status": map[string]any{"type": "integer", "enum": []int{1, 2}}}, "id", "status")
	case "newapi_promo_code_create", "newapi_promo_code_update":
		// CSV 中的每个订阅 ID 同样限制为 int32 正整数，不能只限制字符串长度。
		planIDPattern := "(?:[1-9][0-9]{0,8}|1[0-9]{9}|20[0-9]{8}|21[0-3][0-9]{7}|214[0-6][0-9]{6}|2147[0-3][0-9]{5}|21474[0-7][0-9]{4}|214748[0-2][0-9]{3}|2147483[0-5][0-9]{2}|21474836[0-3][0-9]|214748364[0-7])"
		fields := map[string]any{
			"name": mcpCommerceString(1, 40), "code": mcpCommerceString(1, 64), "discount_type": mcpCommerceEnum("percent", "fixed"), "discount_value": mcpCommerceInteger(1, 2147483647),
			"applies_to_topup": boolean, "applies_to_all_subscription": boolean,
			"subscription_plan_ids": map[string]any{"type": "string", "maxLength": 4096, "pattern": "^(" + planIDPattern + "(," + planIDPattern + ")*)?$"},
			"max_redeem_count":      quota, "expired_time": timestamp,
		}
		required := []string{"name", "code", "discount_type", "discount_value", "applies_to_topup", "applies_to_all_subscription", "subscription_plan_ids", "max_redeem_count", "expired_time"}
		if name == "newapi_promo_code_update" {
			fields["id"] = id
			required = append(required, "id")
		}
		return mcpCommerceObject(fields, required...)
	case "newapi_benefit_activity_create", "newapi_benefit_activity_update":
		amount := map[string]any{"type": "number", "minimum": 0, "maximum": 2147483647}
		return mcpCommerceObject(map[string]any{
			"name": mcpCommerceString(1, 128), "description": mcpCommerceString(0, 8192), "group_id": id, "amount_mode": mcpCommerceEnum("fixed", "random"),
			"amount_display_type": mcpCommerceEnum("USD", "CNY", "TOKENS", "CUSTOM"), "total_amount": amount, "total_count": mcpCommerceInteger(1, 100000),
			"fixed_amount": amount, "min_amount": amount, "max_amount": amount, "claim_paid_threshold": amount,
			"personal_valid_hours": map[string]any{"type": "number", "minimum": 1.0 / 3600, "maximum": 596523}, "starts_at": timestamp, "ends_at": timestamp,
		}, "name", "description", "group_id", "amount_mode", "amount_display_type", "total_amount", "total_count", "fixed_amount", "min_amount", "max_amount", "claim_paid_threshold", "personal_valid_hours", "starts_at", "ends_at")
	case "newapi_benefit_activity_terminate":
		return mcpCommerceObject(map[string]any{"mode": mcpCommerceEnum("unused", "all"), "confirm": confirm, "reason": reason}, "mode", "confirm", "reason")
	case "newapi_benefit_voucher_void":
		return mcpCommerceObject(map[string]any{"confirm": confirm, "reason": reason}, "confirm", "reason")
	case "newapi_benefit_vouchers_void_batch":
		return mcpCommerceObject(map[string]any{"confirm": confirm, "reason": reason, "ids": ids}, "confirm", "reason", "ids")
	case "newapi_affiliate_application_approve", "newapi_affiliate_withdrawal_approve", "newapi_affiliate_withdrawal_reject", "newapi_affiliate_withdrawal_paid", "newapi_affiliate_fraud_dismiss":
		return mcpCommerceObject(map[string]any{"remark": remark})
	case "newapi_affiliate_application_reject":
		return mcpCommerceObject(map[string]any{"reason": reason}, "reason")
	case "newapi_affiliate_inviter_bind":
		return mcpCommerceObject(map[string]any{"user_id": id, "aff_code": mcpCommerceString(1, 64), "force": boolean}, "user_id", "aff_code", "force")
	case "newapi_affiliate_inviter_unbind":
		return mcpCommerceObject(map[string]any{"user_id": id}, "user_id")
	case "newapi_affiliate_access_grant":
		return mcpCommerceObject(map[string]any{"user_id": id, "remark": remark}, "user_id")
	case "newapi_affiliate_risk_apply":
		return mcpCommerceObject(map[string]any{"freeze_assets": boolean, "block_invite_code": boolean, "detach_invitees": boolean, "clear_assets": boolean, "reason": reason}, "freeze_assets", "block_invite_code", "detach_invitees", "clear_assets", "reason")
	case "newapi_affiliate_risk_remove":
		return mcpCommerceObject(map[string]any{"restore_detached_invitees": boolean, "remark": remark}, "restore_detached_invitees", "remark")
	case "newapi_game_prediction_create":
		return mcpCommerceObject(map[string]any{"title": mcpCommerceString(1, 255), "description": mcpCommerceString(0, 4096), "options": map[string]any{"type": "array", "minItems": 2, "maxItems": 2, "items": mcpCommerceString(1, 255)}, "close_time": timestamp, "settle_time": timestamp, "judge_mode": mcpCommerceEnum("manual")}, "title", "description", "options", "close_time", "settle_time", "judge_mode")
	case "newapi_game_prediction_answer_set":
		return mcpCommerceObject(map[string]any{"option_id": id}, "option_id")
	case "newapi_waffo_pancake_pair_create":
		return mcpCommerceObject(map[string]any{"merchant_id": mcpCommerceString(0, 128), "private_key": mcpCommerceString(0, 16384), "return_url": mcpCommerceString(0, 2048)})
	case "newapi_waffo_pancake_save":
		return mcpCommerceObject(map[string]any{"merchant_id": mcpCommerceString(1, 128), "private_key": mcpCommerceString(0, 16384), "return_url": mcpCommerceString(0, 2048), "store_id": mcpCommerceString(1, 128), "product_id": mcpCommerceString(1, 128)}, "merchant_id", "private_key", "return_url", "store_id", "product_id")
	case "newapi_waffo_pancake_subscription_product_create":
		return mcpCommerceObject(map[string]any{"name": mcpCommerceString(1, 128), "amount": map[string]any{"type": "string", "maxLength": 11, "pattern": "^(0|[1-9][0-9]{0,3})(\\.[0-9]{1,6})?$"}}, "name", "amount")
	default:
		// 无请求体的操作显式接受空对象，不能扩展为任意字段透传。
		return mcpCommerceObject(map[string]any{})
	}
}
