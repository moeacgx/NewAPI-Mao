package model

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopupLogWriterOmitsIPAudit(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	RecordLogWithAdminInfo(1, LogTypeTopup, "充值成功", map[string]interface{}{
		"caller_ip": "203.0.113.42", "request_ip": "203.0.113.42",
		"callback_ip": "192.0.2.1", "server_ip": "172.16.0.1",
		"balance_before": 5, "balance_after": 15, "credited_quota": 10,
		"trade_no": "privacy-order", "payment_method": "alipay",
	})
	var log Log
	require.NoError(t, db.First(&log).Error)
	assert.Empty(t, log.Ip)
	audit := readTopUpAuditInfo(t, "privacy-order")
	for _, key := range []string{"caller_ip", "request_ip", "callback_ip", "server_ip"} {
		assert.NotContains(t, audit, key)
	}
	assert.Equal(t, float64(5), audit["balance_before"])
	assert.Equal(t, float64(15), audit["balance_after"])
	assert.Equal(t, float64(10), audit["credited_quota"])
	assert.Equal(t, "alipay", audit["payment_method"])
}

func TestTopupLogPrivacyPreservesExactAmounts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		other string
		want  string
	}{
		{
			name:  "wallet bigint",
			other: `{"admin_info":{"balance_before":9007199254740993,"request_ip":"203.0.113.42"}}`,
			want:  `{"admin_info":{"balance_before":9007199254740993}}`,
		},
		{
			name:  "malformed legacy admin info",
			other: `{"admin_info":"203.0.113.42","note":"保留"}`,
			want:  `{"note":"保留"}`,
		},
		{name: "malformed legacy json", other: `{"admin_info":{"request_ip":"203.0.113.42"`, want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := &Log{Type: LogTypeTopup, Ip: "203.0.113.42", Other: tc.other}
			stripTopupLogIP(log)
			assert.Empty(t, log.Ip)
			if tc.want == "" {
				assert.Empty(t, log.Other)
				return
			}
			var got, want map[string]json.RawMessage
			require.NoError(t, common.UnmarshalJsonStr(log.Other, &got))
			require.NoError(t, common.UnmarshalJsonStr(tc.want, &want))
			assert.Equal(t, want, got)
		})
	}
}

func TestSubscriptionTopupLogOmitsIP(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	RecordTopupLog(1, "订阅购买成功", PaymentMethodBalance, PaymentProviderBalance)
	var log Log
	require.NoError(t, db.First(&log).Error)
	assert.Empty(t, log.Ip)
	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	audit, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok)
	for _, key := range []string{"caller_ip", "request_ip", "callback_ip", "server_ip"} {
		assert.NotContains(t, audit, key)
	}
	assert.Equal(t, PaymentMethodBalance, audit["payment_method"])
}

func TestHistoricalTopupLogQueriesOmitIP(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	require.NoError(t, db.AutoMigrate(&Group{}))
	oldMax := common.MaxRecentItems
	common.MaxRecentItems = 20
	t.Cleanup(func() { common.MaxRecentItems = oldMax })
	legacy := &Log{
		UserId: 1, Username: "audit-admin", TokenId: 123, Type: LogTypeTopup,
		CreatedAt: common.GetTimestamp(), Ip: "172.71.24.199", Content: "充值成功",
		Other: common.MapToJsonStr(map[string]interface{}{
			"admin_info": map[string]interface{}{
				"caller_ip": "203.0.113.42", "request_ip": "203.0.113.42",
				"callback_ip": "192.0.2.1", "server_ip": "172.16.0.1",
				"trade_no": "legacy-order", "balance_before": 5, "balance_after": 15,
				"credited_quota": 10, "paid_amount_cny": 2,
				"payment_method": "alipay", "node_name": "node-a", "version": "test",
			},
			"note": "保留非 IP 业务信息",
		}),
	}
	// 直接写入历史夹具，模拟修复前持久化的数据。
	require.NoError(t, db.Create(legacy).Error)
	RecordLoginLog(1, "audit-admin", "登录成功", "203.0.113.9", "login", nil, nil)
	RecordOperationAuditLog(1, "管理员补单", "203.0.113.10", "topup.complete", nil, nil, nil)

	adminLogs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	for _, log := range adminLogs {
		switch log.Type {
		case LogTypeTopup:
			assert.Empty(t, log.Ip)
			other, err := common.StrToMap(log.Other)
			require.NoError(t, err)
			audit, ok := other["admin_info"].(map[string]interface{})
			require.True(t, ok)
			for _, key := range []string{"caller_ip", "request_ip", "callback_ip", "server_ip"} {
				assert.NotContains(t, audit, key)
			}
			assert.Equal(t, "legacy-order", audit["trade_no"])
			assert.Equal(t, float64(15), audit["balance_after"])
			assert.Equal(t, float64(2), audit["paid_amount_cny"])
			assert.Equal(t, "node-a", audit["node_name"])
			assert.Equal(t, "保留非 IP 业务信息", other["note"])
		case LogTypeLogin:
			assert.Equal(t, "203.0.113.9", log.Ip)
		case LogTypeManage:
			assert.Equal(t, "203.0.113.10", log.Ip)
		}
	}
	userLogs, total, err := GetUserLogs(1, LogTypeTopup, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	assert.Empty(t, userLogs[0].Ip)
	assert.NotContains(t, userLogs[0].Other, "admin_info")
	tokenLogs, err := GetLogByTokenId(123)
	require.NoError(t, err)
	require.Len(t, tokenLogs, 1)
	assert.Empty(t, tokenLogs[0].Ip)

	var persisted Log
	require.NoError(t, db.First(&persisted, legacy.Id).Error)
	assert.Equal(t, legacy.Ip, persisted.Ip, "读取脱敏不改写历史数据库")
}
