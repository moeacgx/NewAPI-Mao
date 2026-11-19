package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func mcpCommerceTestTool(t *testing.T, name string) MCPTool {
	t.Helper()
	for _, tool := range mcpFullCommerceTools() {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("未找到商务工具 %s", name)
	return MCPTool{}
}

// 每项数据库测试都重建内存数据库，禁止继承开发数据库、环境文件或外部连接。
func mcpCommerceMockDB(t *testing.T, tables ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedis, previousBatch := common.RedisEnabled, common.BatchUpdateEnabled
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled, common.BatchUpdateEnabled = false, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled, common.BatchUpdateEnabled = previousRedis, previousBatch
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(tables...))
	return db
}

func TestMCPCommerceRequestContracts(t *testing.T) {
	tests := []struct {
		name, input, path string
	}{
		{"newapi_subscription_plan_status_set", `{"path":{"id":"7"},"body":{"enabled":false},"request_id":"disable"}`, "/api/subscription/admin/plans/7"},
		{"newapi_subscription_bind", `{"body":{"user_id":2,"plan_id":3},"request_id":"bind"}`, "/api/subscription/admin/bind"},
		{"newapi_user_subscription_create", `{"path":{"id":"2"},"body":{"plan_id":3},"request_id":"bind"}`, "/api/subscription/admin/users/2/subscriptions"},
		{"newapi_user_subscriptions_reset", `{"path":{"id":"2"},"body":{"plan_id":3,"advance_reset_time":false},"request_id":"reset"}`, "/api/subscription/admin/users/2/subscriptions/reset"},
		{"newapi_subscription_plan_reset", `{"path":{"id":"3"},"body":{"advance_reset_time":false},"request_id":"reset"}`, "/api/subscription/admin/plans/3/subscriptions/reset"},
		{"newapi_user_subscription_cancel", `{"path":{"id":"4"},"body":{},"request_id":"cancel"}`, "/api/subscription/admin/user_subscriptions/4/invalidate"},
		{"newapi_topup_complete", `{"body":{"trade_no":"order-8"},"request_id":"complete"}`, "/api/user/topup/complete"},
		{"newapi_invoice_update", `{"path":{"id":"5"},"body":{"status":"closed","download_url":"","admin_remark":"重复申请"},"request_id":"close"}`, "/api/user/invoice/5"},
		{"newapi_redemptions_delete_batch", `{"body":{"ids":[1,2]},"request_id":"delete"}`, "/api/redemption/batch"},
		{"newapi_redemption_status_set", `{"query":{"status_only":"true"},"body":{"id":1,"status":2},"request_id":"disable"}`, "/api/redemption/?status_only=true"},
		{"newapi_promo_code_status_set", `{"query":{"status_only":"true"},"body":{"id":1,"status":1},"request_id":"enable"}`, "/api/promo_code/?status_only=true"},
		{"newapi_benefit_activity_terminate", `{"path":{"id":"1"},"body":{"mode":"all","confirm":true,"reason":"活动取消"},"request_id":"terminate"}`, "/api/benefit/admin/activities/1/terminate"},
		{"newapi_benefit_vouchers_void_batch", `{"body":{"ids":[1,2],"confirm":true,"reason":"活动取消"},"request_id":"void"}`, "/api/benefit/admin/vouchers/batch-void"},
		{"newapi_affiliate_inviter_bind", `{"body":{"user_id":1,"aff_code":"ABCD","force":false},"request_id":"bind"}`, "/api/affiliate/admin/bind-inviter"},
		{"newapi_affiliate_risk_apply", `{"path":{"user_id":"2"},"body":{"freeze_assets":true,"block_invite_code":true,"detach_invitees":false,"clear_assets":false,"reason":"待核查"},"request_id":"freeze"}`, "/api/affiliate/admin/risk-users/2/apply"},
		{"newapi_affiliate_risk_preview", `{"path":{"user_id":"2"},"body":{},"request_id":"preview"}`, "/api/affiliate/admin/risk-users/2/preview"},
		{"newapi_affiliate_fraud_scan", `{"query":{"days":"30"},"body":{},"request_id":"scan"}`, "/api/affiliate/admin/fraud-alerts/scan?days=30"},
		{"newapi_game_prediction_answer_set", `{"path":{"id":"1"},"body":{"option_id":2},"request_id":"answer"}`, "/api/game/admin/predictions/1/answer"},
		{"newapi_waffo_pancake_save", `{"body":{"merchant_id":"merchant-1","private_key":"","return_url":"https://example.invalid/return","store_id":"STORE_1","product_id":"PROD_1"},"request_id":"save"}`, "/api/option/waffo-pancake/save"},
		{"newapi_waffo_pancake_subscription_product_create", `{"body":{"name":"月套餐","amount":"19.990000"},"request_id":"create"}`, "/api/option/waffo-pancake/subscription-product"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := mcpCommerceTestTool(t, tt.name)
			arguments, path, body, err := PrepareMCPRequest(tool, []byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.path, path)
			assert.JSONEq(t, string(arguments.Body), string(body), "原始零值与标识必须保留")
			assert.True(t, tool.Write)
		})
	}
}

func TestMCPCommerceRejectsUnsafeInputs(t *testing.T) {
	tests := []struct{ name, input string }{
		{"newapi_subscription_bind", `{"body":{"user_id":0,"plan_id":1},"request_id":"x"}`},
		{"newapi_subscription_bind", `{"body":{"user_id":1,"plan_id":2147483648},"request_id":"x"}`},
		{"newapi_subscription_bind", `{"body":{"user_id":1,"plan_id":1,"quota":-1},"request_id":"x"}`},
		{"newapi_subscription_bind", `{"body":{"user_id":1,"plan_id":1}}`},
		{"newapi_subscription_plan_status_set", `{"path":{"id":"../1"},"body":{"enabled":false},"request_id":"x"}`},
		{"newapi_redemption_create", `{"body":{"name":"测试","quota":-1,"count":1,"expired_time":0,"max_redeem_count":1},"request_id":"x"}`},
		{"newapi_redemption_create", `{"body":{"name":"测试","quota":2147483648,"count":1,"expired_time":0,"max_redeem_count":1},"request_id":"x"}`},
		{"newapi_redemption_create", `{"body":{"name":"测试","quota":1,"count":101,"expired_time":0,"max_redeem_count":1},"request_id":"x"}`},
		{"newapi_redemption_status_set", `{"body":{"id":1,"status":2},"request_id":"x"}`},
		{"newapi_redemption_update", `{"query":{"status_only":"true"},"body":{"id":1,"name":"测试","quota":1,"expired_time":0,"max_redeem_count":1},"request_id":"x"}`},
		{"newapi_redemptions_delete_batch", `{"body":{"ids":[0]},"request_id":"x"}`},
		{"newapi_promo_codes_delete_batch", `{"body":{"ids":[]},"request_id":"x"}`},
		{"newapi_benefit_voucher_void", `{"path":{"id":"1"},"body":{"confirm":false,"reason":"原因"},"request_id":"x"}`},
		{"newapi_benefit_voucher_void", `{"path":{"id":"1"},"body":{"confirm":true,"reason":"   "},"request_id":"x"}`},
		{"newapi_affiliate_fraud_scan", `{"query":{"days":"0"},"body":{},"request_id":"x"}`},
		{"newapi_affiliate_risk_preview", `{"path":{"user_id":"1"}}`},
		{"newapi_game_prediction_answer_set", `{"path":{"id":"1"},"body":{"option_id":0},"request_id":"x"}`},
		{"newapi_waffo_pancake_save", `{"body":{"merchant_id":"a","private_key":"","return_url":"","store_id":"b","product_id":"c","callback_url":"https://example.invalid"},"request_id":"x"}`},
		{"newapi_waffo_pancake_subscription_product_create", `{"body":{"name":"套餐","amount":"-1"},"request_id":"x"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := PrepareMCPRequest(mcpCommerceTestTool(t, tt.name), []byte(tt.input))
			require.Error(t, err)
		})
	}
}

func TestMCPCommercePlanAndPromoPreserveEditingContract(t *testing.T) {
	plan := map[string]any{
		"title": "月套餐", "subtitle": "第一行\n第二行 https://example.invalid/说明", "price_amount": 0, "currency": "CNY", "duration_unit": "month", "duration_value": 1, "custom_seconds": 0,
		"enabled": false, "sort_order": 0, "allow_balance_pay": false, "allow_wallet_overflow": false, "stripe_price_id": "", "creem_product_id": "", "waffo_pancake_product_id": "PROD_1",
		"max_purchase_per_user": 0, "total_amount": 0, "upgrade_group": "", "downgrade_group": "", "quota_reset_period": "never", "quota_reset_custom_seconds": 0,
	}
	args, err := common.Marshal(map[string]any{"path": map[string]string{"id": "3"}, "body": map[string]any{"plan": plan}, "request_id": "plan"})
	require.NoError(t, err)
	_, _, body, err := PrepareMCPRequest(mcpCommerceTestTool(t, "newapi_subscription_plan_update"), args)
	require.NoError(t, err)
	var decoded struct {
		Plan model.SubscriptionPlan `json:"plan"`
	}
	require.NoError(t, common.Unmarshal(body, &decoded))
	require.NotNil(t, decoded.Plan.AllowBalancePay)
	assert.False(t, *decoded.Plan.AllowBalancePay)
	assert.Equal(t, "PROD_1", decoded.Plan.WaffoPancakeProductId)
	assert.Zero(t, decoded.Plan.TotalAmount)
	response, err := common.Marshal(map[string]any{"success": true, "data": []any{map[string]any{"plan": decoded.Plan}}})
	require.NoError(t, err)
	projected, handled, err := projectMCPFullCommerceResponse(MCPTool{Name: "newapi_subscription_plans"}, response)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, string(projected), `"subtitle":"第一行\n第二行 https://example.invalid/说明"`)
	assert.Contains(t, string(projected), `"allow_wallet_overflow":false`)

	promo := map[string]any{"id": 2, "name": "夏季优惠", "code": "SUMMER", "discount_type": "percent", "discount_value": 20, "applies_to_topup": false, "applies_to_all_subscription": false, "subscription_plan_ids": "1,2", "max_redeem_count": 0, "expired_time": 0}
	args, err = common.Marshal(map[string]any{"body": promo, "request_id": "promo"})
	require.NoError(t, err)
	_, _, body, err = PrepareMCPRequest(mcpCommerceTestTool(t, "newapi_promo_code_update"), args)
	require.NoError(t, err)
	var promoModel model.PromoCode
	require.NoError(t, common.Unmarshal(body, &promoModel))
	assert.Equal(t, "1,2", promoModel.SubscriptionPlanIds)
	response, err = common.Marshal(map[string]any{"success": true, "data": promoModel})
	require.NoError(t, err)
	projected, handled, err = projectMCPFullCommerceResponse(MCPTool{Name: "newapi_promo_code_get"}, response)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, string(projected), `"code":"SUMMER"`)
	assert.Contains(t, string(projected), `"subscription_plan_ids":"1,2"`)
}

func TestMCPCommerceResponseProjection(t *testing.T) {
	tests := []struct{ name, input, expected string }{
		{"newapi_topups_list", `{"success":true,"data":{"items":[{"id":1,"trade_no":"order-1","credited_quota":100,"invoice_tax_no":"private","provider_payload":"secret"}],"total":1}}`, `{"success":true,"data":{"items":[{"id":1,"trade_no":"order-1","credited_quota":100}],"total":1}}`},
		{"newapi_invoices_manage_list", `{"success":true,"data":{"items":[{"id":2,"title":"企业开票","tax_no":"tax-1","email":"a@example.invalid","admin_remark":"人工核查\n备注","download_url":"https://example.invalid/invoice.pdf","provider_payload":"secret","request_ip":"private"}]}}`, `{"success":true,"data":{"items":[{"id":2,"title":"企业开票","tax_no":"tax-1","email":"a@example.invalid","admin_remark":"人工核查\n备注","download_url":"https://example.invalid/invoice.pdf"}]}}`},
		{"newapi_affiliate_withdrawals_manage", `{"success":true,"data":{"items":[{"id":1,"status":"approved","payout_details":{"method":"usdt","usdt_address":"wallet-address","usdt_chain":"TRC20","private_key":"secret"},"payout_snapshot":"secret"}]}}`, `{"success":true,"data":{"items":[{"id":1,"status":"approved","payout_details":{"method":"usdt","usdt_address":"wallet-address","usdt_chain":"TRC20"}}]}}`},
		{"newapi_affiliate_fraud_alerts", `{"success":true,"data":{"items":[{"inviter_id":1,"shared_ips":["192.0.2.1"],"alerts":[{"id":2,"shared_ips":"[\"192.0.2.1\"]","admin_remark":"待核验"}]}]}}`, `{"success":true,"data":{"items":[{"inviter_id":1,"shared_ips":["192.0.2.1"],"alerts":[{"id":2,"shared_ips":"[\"192.0.2.1\"]","admin_remark":"待核验"}]}]}}`},
		{"newapi_affiliate_risk_preview", `{"success":true,"data":{"user":{"id":1,"status":1},"active_risk":{"status":"active","freeze_assets":true},"balance":{"available_quota":3},"clearable_quota":3}}`, `{"success":true,"data":{"user":{"id":1,"status":1},"active_risk":{"status":"active","freeze_assets":true},"balance":{"available_quota":3},"clearable_quota":3}}`},
		{"newapi_benefit_voucher_ledger", `{"success":true,"data":[{"id":1,"quota_delta":-5,"balance_after":10,"metadata":"secret","request_id":"req-1"}]}`, `{"success":true,"data":[{"id":1,"quota_delta":-5,"balance_after":10,"request_id":"req-1"}]}`},
		{"newapi_redemptions_delete_batch", `{"success":true,"data":{"deleted_ids":[1],"skipped":[{"id":2,"reason":"已使用"}]}}`, `{"success":true,"data":{"deleted_ids":[1],"skipped":[{"id":2,"reason":"已使用"}]}}`},
		{"newapi_invoice_delete", `{"success":true,"data":1}`, `{"success":true,"data":1}`},
		{"newapi_waffo_pancake_catalog", `{"message":"success","data":{"stores":[{"id":"STORE_1","name":"商店","status":"active","prodEnabled":true,"onetimeProducts":[{"id":"PROD_1","name":"套餐","status":"published","secret":"hidden"}]}]}}`, `{"success":true,"data":{"stores":[{"id":"STORE_1","name":"商店","status":"active","prodEnabled":true,"onetimeProducts":[{"id":"PROD_1","name":"套餐","status":"published"}]}]}}`},
		{"newapi_okpay_rate_preview", `{"success":true,"data":{"raw_rate":"7.2","adjusted_rate":"7.1","tier":2,"source":"okx","side":"buy","adjustment":"-0.1","adjustment_type":"fixed"}}`, `{"success":true,"data":{"raw_rate":"7.2","adjusted_rate":"7.1","tier":2,"source":"okx","side":"buy","adjustment":"-0.1","adjustment_type":"fixed"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projected, handled, err := projectMCPFullCommerceResponse(MCPTool{Name: tt.name, TypedArguments: true}, []byte(tt.input))
			require.NoError(t, err)
			require.True(t, handled)
			assert.JSONEq(t, tt.expected, string(projected))
		})
	}
	_, handled, err := projectMCPFullCommerceResponse(MCPTool{Name: "future_unknown_tool", Category: "billing"}, []byte(`{"data":{"key":"secret"}}`))
	require.NoError(t, err)
	assert.False(t, handled)
	_, handled, err = projectMCPFullCommerceResponse(MCPTool{Name: "newapi_affiliate_risk_preview"}, []byte(`{"data":{"clearable_quota":{"key":"secret"}}}`))
	assert.True(t, handled)
	require.Error(t, err, "数值字段不能承载任意对象")
}

func TestMCPCommerceGroupDisplayNames(t *testing.T) {
	db := mcpCommerceMockDB(t, &model.Group{}, &model.GroupAlias{})
	group := model.Group{Code: "vip-code", Name: "当前会员组"}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Create(&model.GroupAlias{GroupId: group.Id, Alias: "old-vip"}).Error)
	raw := []byte(`{"success":true,"data":{"upgrade_group":"old-vip","downgrade_group":"deleted-group"}}`)
	projected, handled, err := projectMCPFullCommerceResponse(MCPTool{Name: "newapi_subscription_plans"}, raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, string(projected), `"upgrade_group":"old-vip"`)
	assert.Contains(t, string(projected), `"upgrade_group_name":"当前会员组"`)
	assert.Contains(t, string(projected), `"downgrade_group_name":"deleted-group"`)
	require.NoError(t, db.Model(&group).Update("name", "改名会员组").Error)
	raw, err = common.Marshal(map[string]any{"success": true, "data": map[string]any{"group_id": group.Id, "group_code_snapshot": "old-vip", "group_name_snapshot": "旧名称"}})
	require.NoError(t, err)
	projected, _, err = projectMCPFullCommerceResponse(MCPTool{Name: "newapi_benefit_activity_get"}, raw)
	require.NoError(t, err)
	assert.Contains(t, string(projected), `"group_name":"改名会员组"`)
}

func TestMCPCommerceGameRealPersistence(t *testing.T) {
	mcpCommerceMockDB(t, &model.GamePrediction{}, &model.GamePredictionOption{})
	raw, err := common.Marshal(map[string]any{"request_id": "game-create", "body": map[string]any{"title": "验证预测", "description": "多行\n描述", "options": []string{"是", "否"}, "close_time": time.Now().Unix() + 3600, "settle_time": 0, "judge_mode": "manual"}})
	require.NoError(t, err)
	tool := mcpCommerceTestTool(t, "newapi_game_prediction_create")
	_, _, body, err := PrepareMCPRequest(tool, raw)
	require.NoError(t, err)
	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Options     []string `json:"options"`
		CloseTime   int64    `json:"close_time"`
		SettleTime  int64    `json:"settle_time"`
		JudgeMode   string   `json:"judge_mode"`
	}
	require.NoError(t, common.Unmarshal(body, &req))
	prediction, err := CreateGamePrediction(CreateGamePredictionRequest{Title: req.Title, Description: req.Description, Options: req.Options, CloseTime: req.CloseTime, SettleTime: req.SettleTime, JudgeMode: req.JudgeMode, CreatedBy: 1})
	require.NoError(t, err)
	require.Len(t, prediction.Options, 2)
	response, err := common.Marshal(map[string]any{"success": true, "data": prediction})
	require.NoError(t, err)
	projected, handled, err := projectMCPFullCommerceResponse(tool, response)
	require.NoError(t, err)
	require.True(t, handled)
	var result struct {
		Data model.GamePrediction `json:"data"`
	}
	require.NoError(t, common.Unmarshal(projected, &result))
	assert.Equal(t, prediction.Options, result.Data.Options)
	assert.Equal(t, prediction.Description, result.Data.Description)
	assert.NotContains(t, string(projected), "judge_result_json")
	assert.Equal(t, authz.GameAdminWrite, tool.Permission)
	assert.Equal(t, authz.GameAdminRead, mcpCommerceTestTool(t, "newapi_game_predictions").Permission)
}

func TestMCPCommerceRedemptionSecretFirstResponseOnly(t *testing.T) {
	db := mcpCommerceMockDB(t, &model.User{}, &model.MCPKey{}, &model.MCPWriteRequest{}, &model.MCPCallLog{})
	user := model.User{Username: "commerce-admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	key := model.MCPKey{UserID: user.Id, KeyHash: "commerce-test-hash", Tools: []string{"newapi_redemption_create"}, ExpiresAt: time.Now().Unix() + 3600}
	require.NoError(t, db.Create(&key).Error)
	identity := &MCPIdentity{User: user.ToBaseUser(), Key: &key}
	code := "0123456789abcdef0123456789abcdef"
	calls := 0
	dispatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/api/redemption/", r.URL.Path)
		payload, err := common.Marshal(map[string]any{"success": true, "data": []string{code}})
		require.NoError(t, err)
		_, err = w.Write(payload)
		require.NoError(t, err)
	})
	raw := []byte(`{"request_id":"create-redemption","body":{"name":"测试","quota":100,"count":1,"expired_time":0,"max_redeem_count":1}}`)
	result := ExecuteMCPTool(context.Background(), identity, "newapi_redemption_create", raw, dispatch)
	require.False(t, result.IsError, "%+v", result)
	require.Len(t, result.Content, 1)
	assert.Contains(t, result.Content[0].Text, code)
	replay := ExecuteMCPTool(context.Background(), identity, "newapi_redemption_create", raw, dispatch)
	require.False(t, replay.IsError)
	assert.NotContains(t, replay.Content[0].Text, code)
	assert.Contains(t, replay.Content[0].Text, "result_not_retained")
	assert.Equal(t, 1, calls)
	var writes []model.MCPWriteRequest
	require.NoError(t, db.Find(&writes).Error)
	require.Len(t, writes, 1)
	assert.NotContains(t, writes[0].Result, code)
	assert.True(t, mcpCommerceTestTool(t, "newapi_redemption_create").TransientResult)
}

func TestMCPCommerceLargeTypedResponseAndRootBoundary(t *testing.T) {
	tool := mcpCommerceTestTool(t, "newapi_benefit_activity_get")
	raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"description": strings.Repeat("说明", 10000)}})
	require.NoError(t, err)
	projected, handled, err := projectMCPFullCommerceResponse(tool, raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, string(raw), string(projected))
	_, _, err = projectMCPFullCommerceResponse(tool, []byte(strings.Repeat(" ", 4*1024*1024+1)))
	require.Error(t, err)
	for _, name := range []string{"newapi_okpay_rate_preview", "newapi_waffo_pancake_catalog", "newapi_waffo_pancake_pair_create", "newapi_waffo_pancake_save", "newapi_waffo_pancake_subscription_product_create", "newapi_waffo_pancake_subscription_product_options"} {
		tool := mcpCommerceTestTool(t, name)
		assert.Equal(t, common.RoleRootUser, tool.MinRole)
		assert.True(t, tool.Write, "主动请求第三方也要求 request_id")
	}
}

func TestMCPCommerceRootAndGamePermissionIsolation(t *testing.T) {
	db := mcpCommerceMockDB(t, &model.CasbinRule{})
	key := &model.MCPKey{Tools: []string{"newapi_waffo_pancake_save", "newapi_game_predictions", "newapi_game_prediction_create"}}
	admin := &model.UserBase{Id: 8, Role: common.RoleAdminUser}
	assert.Empty(t, MCPVisibleTools(admin, key), "普通管理员不能取得 Root 支付配置或未授权游戏管理工具")
	require.NoError(t, db.Create(&model.CasbinRule{Ptype: "p", V0: authz.UserSubject(8), V1: authz.GameAdminRead.Resource, V2: authz.GameAdminRead.Action, V3: "allow"}).Error)
	visible := MCPVisibleTools(admin, key)
	require.Len(t, visible, 1)
	assert.Equal(t, "newapi_game_predictions", visible[0].Name)
	admin.Role = common.RoleRootUser
	assert.Len(t, MCPVisibleTools(admin, key), 3)
}

func TestMCPCommercePaymentStringMerchantQuery(t *testing.T) {
	tool := mcpCommerceTestTool(t, "newapi_waffo_pancake_catalog")
	_, path, _, err := PrepareMCPRequest(tool, []byte(`{"query":{"merchant_id":"MERCHANT_1","private_key":"test-only-private-key"},"body":{},"request_id":"catalog"}`))
	require.NoError(t, err, "Waffo 的 merchant_id 是字符串，不能强制转整数")
	assert.Contains(t, path, "merchant_id=MERCHANT_1")
}

func TestMCPCommercePaymentBusinessFailure(t *testing.T) {
	tool := mcpCommerceTestTool(t, "newapi_waffo_pancake_pair_create")
	projected, success, err := ProjectMCPResponse(tool, []byte(`{"message":"error","data":{"error":"private provider response","store_id":"STORE_1","orphan_store":true}}`), http.StatusOK)
	require.NoError(t, err)
	assert.False(t, success, "Waffo 的 message=error 不能作为 HTTP 200 成功响应")
	assert.NotContains(t, string(projected), "private provider response")
}

func TestMCPCommerceBatchLimitsAndResetResult(t *testing.T) {
	ids := make([]int, 101)
	for i := range ids {
		ids[i] = i + 1
	}
	raw, err := common.Marshal(map[string]any{"request_id": "batch", "body": map[string]any{"ids": ids}})
	require.NoError(t, err)
	_, _, _, err = PrepareMCPRequest(mcpCommerceTestTool(t, "newapi_invoices_delete_batch"), raw)
	require.Error(t, err, "发票宿主单批上限是 100")
	_, _, _, err = PrepareMCPRequest(mcpCommerceTestTool(t, "newapi_redemptions_delete_batch"), raw)
	require.NoError(t, err, "兑换码单批上限是 500")
	result := model.SubscriptionResetResult{PlanId: 4, MatchedCount: 3, ResetCount: 2, UserCount: 1, AdvanceResetTime: false}
	raw, err = common.Marshal(map[string]any{"success": true, "data": result})
	require.NoError(t, err)
	projected, handled, err := projectMCPFullCommerceResponse(mcpCommerceTestTool(t, "newapi_subscription_plan_reset"), raw)
	require.NoError(t, err)
	require.True(t, handled)
	assert.JSONEq(t, string(raw), string(projected))
}

func TestMCPCommerceLegacyMetadataDoesNotGainSecrets(t *testing.T) {
	for _, tc := range []struct{ name, category, raw string }{
		{"newapi_redemptions_list", "marketing", `{"success":true,"data":[{"id":1,"quota":20,"key":"short-secret","code":"private-code"}]}`},
		{"newapi_redemption_get", "marketing", `{"success":true,"data":{"id":1,"key":"short-secret"}}`},
		{"newapi_promo_codes_list", "marketing", `{"success":true,"data":[{"id":1,"code":"private-code"}]}`},
		{"newapi_invoices_list", "billing", `{"success":true,"data":{"items":[{"id":1,"amount":2,"tax_no":"private-tax","email":"private-email","download_url":"private-file","admin_remark":"private-note"}]}}`},
		{"newapi_affiliate_withdrawals", "affiliate", `{"success":true,"data":{"id":1,"payout_details":{"usdt_address":"private-address"}}}`},
		{"newapi_affiliate_invitations", "affiliate", `{"success":true,"data":{"id":1,"inviter_aff_code":"private-code"}}`},
		{"newapi_benefit_activities", "marketing", `{"success":true,"data":{"items":[{"id":1,"description":"private-description","terminate_reason":"private-reason"}]}}`},
		{"newapi_benefit_report", "marketing", `{"success":true,"data":{"total_quota":100,"description":"private-description","terminate_reason":"private-reason"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projected, success, err := ProjectMCPResponse(MCPTool{Name: tc.name, Category: tc.category}, []byte(tc.raw), http.StatusOK)
			require.NoError(t, err)
			require.True(t, success)
			assert.NotContains(t, string(projected), "private-")
			assert.NotContains(t, string(projected), "short-secret")
			if tc.name == "newapi_invoices_list" {
				assert.Contains(t, string(projected), `"amount":2`)
			}
		})
	}
	projected, handled, err := projectMCPFullCommerceResponse(mcpCommerceTestTool(t, "newapi_redemption_secret_get"), []byte(`{"success":true,"data":{"id":1,"key":"0123456789abcdef0123456789abcdef"}}`))
	require.NoError(t, err)
	require.True(t, handled)
	assert.Contains(t, string(projected), "0123456789abcdef0123456789abcdef")
	visible := MCPVisibleTools(&model.UserBase{Id: 1, Role: common.RoleRootUser}, &model.MCPKey{Tools: []string{"newapi_redemption_get"}})
	require.Len(t, visible, 1)
	assert.Equal(t, "newapi_redemption_get", visible[0].Name, "旧授权不能自动发现新的秘密读取工具")
	benefit := mcpCommerceTestTool(t, "newapi_benefit_activities_manage")
	projected, success, err := ProjectMCPResponse(benefit, []byte(`{"success":true,"data":{"items":[{"id":1,"description":"private-description","terminate_reason":"private-reason"}]}}`), http.StatusOK)
	require.NoError(t, err)
	require.True(t, success)
	assert.Contains(t, string(projected), "private-description")
	assert.Contains(t, string(projected), "private-reason")
	assert.True(t, mcpCommerceTestTool(t, "newapi_benefit_report_manage").Write)
	visible = MCPVisibleTools(&model.UserBase{Id: 1, Role: common.RoleRootUser}, &model.MCPKey{Tools: []string{"newapi_benefit_activities", "newapi_benefit_report"}})
	require.Len(t, visible, 2)
	for _, tool := range visible {
		assert.NotContains(t, tool.Name, "_manage", "旧授权不能获得新增福利管理工具")
	}
}

func TestMCPCommercePaymentOptionSecretReview(t *testing.T) {
	for _, name := range []string{"OkpayMerchantToken", "BepusdtAuthToken", "SMTPToken", "WeChatServerToken"} {
		t.Run(name, func(t *testing.T) {
			projected := MCPProjectOption(name, "unmarked-private-credential")
			assert.True(t, projected.Redacted)
			assert.Empty(t, projected.Value)
		})
	}
}

func TestMCPCommerceMonetaryOptionReview(t *testing.T) {
	previous := common.OptionMap
	common.OptionMap = map[string]string{"QuotaPerUnit": "500000", "Price": "7", "ModelRatio": "{}"}
	t.Cleanup(func() { common.OptionMap = previous })
	var tool MCPTool
	for _, candidate := range MCPTools() {
		if candidate.Name == "newapi_setting_update" {
			tool = candidate
			break
		}
	}
	require.NotEmpty(t, tool.Name)
	for _, tc := range []struct{ key, value string }{
		{"QuotaPerUnit", "NaN"}, {"QuotaPerUnit", "0"}, {"Price", "-1"}, {"ModelRatio", `{"model-a":-1}`}, {"ModelRatio", `{"model-a":`},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			raw, err := common.Marshal(map[string]any{"request_id": "invalid-setting", "body": map[string]any{"key": tc.key, "value": tc.value}})
			require.NoError(t, err)
			_, _, _, err = PrepareMCPRequest(tool, raw)
			require.Error(t, err, "有副作用的路由之前必须拒绝无效计费配置")
		})
	}
}

func TestMCPCommerceTypedWriteReceiptKeepsOnlyMetadata(t *testing.T) {
	for _, size := range []int{64, 50 * 1024} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			db := mcpCommerceMockDB(t, &model.User{}, &model.MCPKey{}, &model.MCPWriteRequest{}, &model.MCPCallLog{})
			user := model.User{Username: "receipt-admin", Role: common.RoleAdminUser, Status: common.UserStatusEnabled}
			require.NoError(t, db.Create(&user).Error)
			key := model.MCPKey{UserID: user.Id, KeyHash: "receipt-test-hash", Tools: []string{"newapi_invoice_update"}, ExpiresAt: time.Now().Unix() + 3600}
			require.NoError(t, db.Create(&key).Error)
			identity := &MCPIdentity{User: user.ToBaseUser(), Key: &key}
			note := "private-invoice-note-" + strings.Repeat("x", size)
			payload, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"id": 7, "status": "issued", "admin_remark": note}})
			require.NoError(t, err)
			calls := 0
			dispatch := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, "/api/user/invoice/7", r.URL.Path)
				_, err := w.Write(payload)
				require.NoError(t, err)
			})
			raw := []byte(`{"path":{"id":"7"},"body":{"status":"issued","download_url":"","admin_remark":"测试"},"request_id":"receipt-invoice"}`)
			first := ExecuteMCPTool(context.Background(), identity, "newapi_invoice_update", raw, dispatch)
			require.False(t, first.IsError, "%+v", first)
			require.Len(t, first.Content, 1)
			assert.Contains(t, first.Content[0].Text, note, "首次返回完整投影，不因回执限制截断")
			replay := ExecuteMCPTool(context.Background(), identity, "newapi_invoice_update", raw, dispatch)
			require.False(t, replay.IsError, "%+v", replay)
			require.Len(t, replay.Content, 1)
			assert.Contains(t, replay.Content[0].Text, `"result_not_retained":true`)
			assert.Contains(t, replay.Content[0].Text, `"id":7`)
			assert.NotContains(t, replay.Content[0].Text, "private-invoice-note")
			assert.Equal(t, 1, calls, "重试不得重复修改发票")
			var record model.MCPWriteRequest
			require.NoError(t, db.First(&record).Error)
			assert.Equal(t, "done", record.State)
			assert.NotContains(t, record.Result, "private-invoice-note")
			assert.LessOrEqual(t, len(record.Result), 48*1024)
		})
	}
}

func TestMCPCommercePaymentReceiptRetainsCreatedResourceIDs(t *testing.T) {
	// 外部创建可能已成功而首轮连接丢失，回执必须能定位新建店铺和产品。
	tool := mcpCommerceTestTool(t, "newapi_waffo_pancake_pair_create")
	projected, success, err := ProjectMCPResponse(tool, []byte(`{"message":"success","data":{"store_id":"STORE_1","store_name":"商店私有名称","product_id":"PROD_1","product_name":"产品私有名称"}}`), http.StatusOK)
	require.NoError(t, err)
	require.True(t, success)
	receipt, err := mcpManagementReceipt(projected, "waffo-pair", true, false)
	require.NoError(t, err)
	require.Len(t, receipt.Content, 1)
	assert.Contains(t, receipt.Content[0].Text, `"store_id":"STORE_1"`)
	assert.Contains(t, receipt.Content[0].Text, `"product_id":"PROD_1"`)
	assert.NotContains(t, receipt.Content[0].Text, "私有名称")
}
