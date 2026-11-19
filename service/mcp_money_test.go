package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPMoneyDisplay(t *testing.T) {
	general := operation_setting.GetGeneralSetting()
	original, quotaPerUnit, rate := *general, common.QuotaPerUnit, operation_setting.USDExchangeRate
	t.Cleanup(func() {
		*general, common.QuotaPerUnit, operation_setting.USDExchangeRate = original, quotaPerUnit, rate
	})
	common.QuotaPerUnit = 500000
	general.AutoUSDExchangeRate = false
	operation_setting.USDExchangeRate = 7.2
	general.CustomCurrencySymbol = "€"
	general.CustomCurrencyExchangeRate = 0.9
	for _, tc := range []struct {
		mode, balance, used string
	}{
		{"USD", "$9200.66", "$27780.87"},
		{"CNY", "¥66244.78", "¥200022.23"},
		{"CUSTOM", "€8280.60", "€25002.78"},
		{"TOKENS", "4600331893 TOKENS", "13890432541 TOKENS"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			general.QuotaDisplayType = tc.mode
			raw := []byte(`{"success":true,"data":{"quota":4600331893,"used_quota":13890432541,"request_count":917513}}`)
			out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: "newapi_user_get"}, raw)
			require.NoError(t, err)
			var result struct {
				Data struct {
					Quota   json.RawMessage   `json:"quota"`
					Display map[string]string `json:"money_display"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(out, &result))
			assert.Equal(t, tc.balance, result.Data.Display["quota"])
			assert.Equal(t, tc.used, result.Data.Display["used_quota"])
			assert.NotContains(t, result.Data.Display, "request_count")
			assert.Equal(t, "4600331893", string(result.Data.Quota))
		})
	}
	general.QuotaDisplayType = "USD"
	for _, tc := range []struct{ raw, display string }{
		{`0`, "$0.00"},
		{`1`, "$0.01"},
		{`-1`, "$-0.01"},
		{`-500000`, "$-1.00"},
		{`9223372036854775807`, "$18446744073709.55"},
		{`null`, ""},
	} {
		out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: "newapi_user_get"}, []byte(`{"success":true,"data":{"quota":`+tc.raw+`}}`))
		require.NoError(t, err)
		assert.Contains(t, string(out), `"quota":`+tc.raw)
		if tc.display != "" {
			assert.Contains(t, string(out), `"quota":"`+tc.display+`"`)
		}
	}
	out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: "newapi_my_token_get"}, []byte(`{"data":{"remain_quota":500000,"unlimited_quota":true,"used_quota":250000}}`))
	require.NoError(t, err)
	assert.Contains(t, string(out), `"remain_quota":"Unlimited"`)
	assert.Contains(t, string(out), `"used_quota":"$0.50"`)
}

func TestMCPMoneyFieldUnits(t *testing.T) {
	general := operation_setting.GetGeneralSetting()
	original, quotaPerUnit := *general, common.QuotaPerUnit
	t.Cleanup(func() { *general, common.QuotaPerUnit = original, quotaPerUnit })
	general.QuotaDisplayType, common.QuotaPerUnit = "USD", 500000
	for _, tc := range []struct{ tool, raw, converted, untouched string }{
		{"newapi_subscription_plans", `{"data":[{"plan":{"total_amount":500000,"price_amount":12.50}}]}`, `"total_amount":"$1.00"`, `"price_amount":12.50`},
		{"newapi_user_subscriptions", `{"data":[{"amount_total":500000,"amount_used":250000}]}`, `"amount_used":"$0.50"`, `"amount_total":500000`},
		{"newapi_invoices_manage_list", `{"data":{"total_amount":12.50}}`, "", `"total_amount":12.50`},
		{"newapi_channel_get", `{"data":{"balance":123.45}}`, "", `"balance":123.45`},
		{"newapi_topups_list", `{"data":{"amount":10,"money":72,"credited_quota":5000000}}`, `"credited_quota":"$10.00"`, `"money":72`},
		{"newapi_benefit_voucher_ledger", `{"data":{"balance_after":500000}}`, `"balance_after":"$1.00"`, `"balance_after":500000`},
		{"newapi_revenue", `{"data":{"redemption_quota":500000,"online_money":72}}`, `"redemption_quota":"$1.00"`, `"online_money":72`},
		{"newapi_ratio_fetch", `{"data":{"differences":{"model":{"ratio":{"upstreams":{"quota":12.50,"money_display":1.5}}}}}}`, "", `"quota":12.50`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: tc.tool}, []byte(tc.raw))
			require.NoError(t, err)
			assert.Contains(t, string(out), tc.untouched)
			if tc.converted != "" {
				assert.Contains(t, string(out), tc.converted)
			} else {
				assert.JSONEq(t, tc.raw, string(out))
			}
		})
	}
}

func TestMCPMoneyUnlimitedSubscriptions(t *testing.T) {
	general := operation_setting.GetGeneralSetting()
	original := *general
	t.Cleanup(func() { *general = original })
	for _, mode := range []string{"USD", "TOKENS"} {
		general.QuotaDisplayType = mode
		for _, tc := range []struct{ field, value string }{
			{"total_amount", "0"}, {"amount_total", "0"}, {"amount_total", "-1"},
		} {
			out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: "newapi_user_subscriptions"}, []byte(`{"data":{"`+tc.field+`":`+tc.value+`,"amount_used":0}}`))
			require.NoError(t, err)
			assert.Contains(t, string(out), `"`+tc.field+`":"Unlimited"`)
			assert.NotContains(t, string(out), `"amount_used":"Unlimited"`)
		}
	}
}

func TestMCPMoneyUsesSiteDisplayRateAndUnit(t *testing.T) {
	general := operation_setting.GetGeneralSetting()
	original, unit, configuredRate := *general, common.QuotaPerUnit, operation_setting.USDExchangeRate
	displayExchangeRateMu.Lock()
	previousCache := displayExchangeRateCache
	displayExchangeRateCache = displayExchangeRateCacheEntry{
		Endpoint: coinGeckoUSDTToCNYURL, Rate: 6.76, ExpiresAt: time.Now().Add(time.Minute),
	}
	displayExchangeRateMu.Unlock()
	t.Cleanup(func() {
		*general, common.QuotaPerUnit, operation_setting.USDExchangeRate = original, unit, configuredRate
		displayExchangeRateMu.Lock()
		displayExchangeRateCache = previousCache
		displayExchangeRateMu.Unlock()
	})
	general.QuotaDisplayType, general.AutoUSDExchangeRate = "CNY", true
	common.QuotaPerUnit, operation_setting.USDExchangeRate = 1000000, 7.2
	out, err := formatMCPMoneyResponse(context.Background(), MCPTool{Name: "newapi_user_get"}, []byte(`{"data":{"quota":1000000}}`))
	require.NoError(t, err)
	assert.Contains(t, string(out), `"quota":"¥6.76"`)
	assert.Contains(t, string(out), `"exchange_rate_source":"coingecko"`)
	assert.Contains(t, string(out), `"quota_per_unit":"1000000"`)
}
