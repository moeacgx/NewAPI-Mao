package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPMoneyUserQueriesRealRoute(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	model.InitDBColumns()
	general := operation_setting.GetGeneralSetting()
	previous, unit, rate := *general, common.QuotaPerUnit, operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*general, common.QuotaPerUnit, operation_setting.USDExchangeRate = previous, unit, rate
	})
	general.QuotaDisplayType, general.AutoUSDExchangeRate = "CNY", false
	common.QuotaPerUnit, operation_setting.USDExchangeRate = 500000, 7.2
	target := model.User{Username: "money-user", DisplayName: "金额展示用户", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", Quota: 4600331893, UsedQuota: 1000000}
	require.NoError(t, model.DB.Create(&target).Error)
	names := []string{"newapi_users_search", "newapi_users_list", "newapi_user_get", "newapi_user_edit_get", "newapi_user_quota_adjust", "newapi_context"}
	key, secret, err := service.CreateMCPKey(f.user.ToBaseUser(), "金额展示验收", names, time.Now().Unix()+3600)
	require.NoError(t, err)
	f.key, f.secret = key, secret
	for _, name := range names[:4] {
		t.Run(name, func(t *testing.T) {
			arguments := map[string]any{}
			switch name {
			case "newapi_users_search":
				arguments["query"] = map[string]string{"keyword": target.Username}
			case "newapi_user_get", "newapi_user_edit_get":
				arguments["path"] = map[string]string{"id": strconv.Itoa(target.Id)}
			}
			out := callMCPAccountRoute(t, f, name, arguments)
			assert.Contains(t, string(out), `"quota":"¥66244.78"`)
			assert.Contains(t, string(out), `"quota":4600331893`)
			assert.Contains(t, string(out), `"display_type":"CNY"`)
			assert.NotContains(t, string(out), `"password"`)
			if name != "newapi_user_edit_get" {
				assert.Contains(t, string(out), `"used_quota":"¥14.40"`)
			}
		})
	}
	// 展示金额不会改变调整参数；100 个内部单位必须只增加 100。
	callMCPAccountRoute(t, f, "newapi_user_quota_adjust", map[string]any{
		"request_id": "money-adjust", "body": map[string]any{"id": target.Id, "action": "add_quota", "mode": "add", "value": 100},
	})
	var updated model.User
	require.NoError(t, model.DB.First(&updated, target.Id).Error)
	assert.Equal(t, int64(4600331993), updated.Quota)
	context := callMCPAccountRoute(t, f, "newapi_context", map[string]any{})
	assert.Contains(t, string(context), "money_display_instructions")
	response := f.rpc(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"money-test","version":"1"}}}`, nil)
	var initialized struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &initialized))
	assert.Contains(t, string(initialized.Result["instructions"]), "money_display")
}

func TestMCPMoneyWriteReceiptSurvivesDisplayFailureAndReplay(t *testing.T) {
	f := setupMCPAccountRoutes(t)
	key, _, err := service.CreateMCPKey(f.user.ToBaseUser(), "展示回执验收", []string{"newapi_affiliate_records"}, time.Now().Unix()+3600)
	require.NoError(t, err)
	identity := &service.MCPIdentity{User: f.user.ToBaseUser(), Key: key}
	general := operation_setting.GetGeneralSetting()
	previous, unit := *general, common.QuotaPerUnit
	t.Cleanup(func() { *general, common.QuotaPerUnit = previous, unit })
	common.QuotaPerUnit = 500000
	for _, tc := range []struct {
		name string
		rows int
	}{
		{"展示扩张超限", 800},
		{"历史原始回执", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			general.QuotaDisplayType = "USD"
			count := 0
			dispatch := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				count++
				_, _ = w.Write([]byte(`{"success":true,"data":[` + strings.TrimSuffix(strings.Repeat(`{"quota":500000},`, tc.rows), ",") + `]}`))
			})
			args, err := common.Marshal(map[string]any{"request_id": "money-receipt-" + strconv.Itoa(tc.rows)})
			require.NoError(t, err)
			first := service.ExecuteMCPTool(context.Background(), identity, "newapi_affiliate_records", args, dispatch)
			require.False(t, first.IsError)
			var record model.MCPWriteRequest
			require.NoError(t, model.DB.Where("key_id = ? AND request_id = ?", key.ID, "money-receipt-"+strconv.Itoa(tc.rows)).First(&record).Error)
			assert.Equal(t, "done", record.State)
			assert.NotContains(t, record.Result, "money_display", "持久化业务结果与展示无关")
			general.QuotaDisplayType, general.CustomCurrencySymbol, general.CustomCurrencyExchangeRate = "CUSTOM", "€", 2
			replay := service.ExecuteMCPTool(context.Background(), identity, "newapi_affiliate_records", args, dispatch)
			require.False(t, replay.IsError)
			assert.Equal(t, 1, count, "重放不得再次执行业务")
			if tc.rows == 1 {
				require.Len(t, replay.Content, 1)
				assert.Contains(t, replay.Content[0].Text, `"quota":"€2.00"`)
			} else {
				require.Len(t, replay.Content, 2)
				assert.Contains(t, replay.Content[1].Text, "金额展示暂不可用")
			}
		})
	}
}
