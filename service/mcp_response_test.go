package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPResponseBusinessSamples(t *testing.T) {
	for _, tc := range []struct {
		name, category, raw string
		present, absent     []string
	}{
		{"newapi_channel_models", "channels", `{"success":true,"data":["gpt-4.1","vendor/model-1"]}`, []string{`"gpt-4.1"`, `"vendor/model-1"`}, []string{`null`}},
		{"newapi_channel_models_enabled", "channels", `{"success":true,"data":["gemini-2.5-pro"]}`, []string{`"gemini-2.5-pro"`}, []string{`null`}},
		{"newapi_channel_get", "channels", `{"success":true,"data":{"id":1,"models":"gpt-4.1,gemini-2.5-pro","group":"internal-code","group_name":"显示组","setting":"sensitive","base_url":"https://private.example","key_preview":"short"}}`, []string{`"models":"gpt-4.1,gemini-2.5-pro"`, `"group_name":"显示组"`}, []string{"sensitive", "private.example", "short"}},
		{"newapi_ticket_stats", "tickets", `{"success":true,"data":{"open":1,"processing":2,"resolved":3,"closed":4}}`, []string{`"processing":2`}, nil},
		{"newapi_users_list", "users", `{"success":true,"data":{"items":[{"id":7,"username":"admin","email":"private@example.com","github_id":"private-id","remark":"private-note","quota":50}]}}`, []string{`"id":7`, `"quota":50`}, []string{"private"}},
		{"newapi_invoices_list", "billing", `{"success":true,"data":{"items":[{"id":8,"status":"pending","amount":2,"tax_number":"private-tax","address":"private-address","file_url":"https://private.example"}]}}`, []string{`"status":"pending"`, `"amount":2`}, []string{"private"}},
		{"newapi_affiliate_withdrawals", "affiliate", `{"success":true,"data":{"id":9,"status":"paid","payout_account":"private-account","payment_proof":"private-proof"}}`, []string{`"status":"paid"`}, []string{"private"}},
		{"newapi_redemptions_list", "marketing", `{"success":true,"data":[{"id":1,"name":"兑换","quota":50,"key":"short-secret","code":"also-private"}]}`, []string{`"quota":50`}, []string{"short-secret", "also-private"}},
		{"newapi_tasks", "tasks", `{"success":true,"data":{"id":1,"status":"failed","properties":{"key":"raw-secret"},"fail_reason":"raw-secret","data":"raw-secret"}}`, []string{`"status":"failed"`}, []string{"raw-secret"}},
		{"newapi_notification_tasks", "notifications", `{"success":true,"data":{"id":1,"status":"pending","target":"private-target","template":"private-text","bot_token":"private-token"}}`, []string{`"status":"pending"`}, []string{"private"}},
		{"newapi_plugins_list", "extensions", `{"success":true,"data":[{"id":1,"name":"插件","version":"1.0","source":"private-source","config":{"name":"private-config"}}]}`, []string{`"version":"1.0"`}, []string{"private"}},
		{"newapi_group_tags_list", "groups", `{"success":true,"data":[{"id":1,"name":"标签","group_ids":[2],"sort_order":3,"description":"标签说明","icons":["https://cdn.example/icon.svg"],"match_keywords":["模型"]}]}`, []string{`"group_ids":[2]`, `"sort_order":3`, `"description":"标签说明"`, `"match_keywords":["模型"]`, `https://cdn.example/icon.svg`}, nil},
		{"newapi_model_get", "models", `{"success":true,"data":{"id":1,"model_name":"gpt-4.1","vendor_id":2,"status":1,"description":"模型说明","icon":"OpenAI","tags":"chat","name_rule":0,"sync_official":0,"endpoints":"[\"openai\"]"}}`, []string{`"model_name":"gpt-4.1"`, `"vendor_id":2`, `"description":"模型说明"`, `"sync_official":0`, `"name_rule":0`, `"endpoints":"[\"openai\"]"`}, nil},
		{"newapi_vendor_get", "models", `{"success":true,"data":{"id":1,"name":"供应商","status":1,"description":"供应商说明","icon":"OpenAI"}}`, []string{`"name":"供应商"`, `"description":"供应商说明"`, `"icon":"OpenAI"`}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projected, success, err := ProjectMCPResponse(MCPTool{Name: tc.name, Category: tc.category}, []byte(tc.raw), 200)
			require.NoError(t, err)
			require.True(t, success)
			for _, value := range tc.present {
				assert.Contains(t, string(projected), value)
			}
			for _, value := range tc.absent {
				assert.NotContains(t, string(projected), value)
			}
		})
	}
	projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_users_list", Category: "users"}, []byte(`{"success":true,"data":["raw-secret"]}`), 200)
	require.NoError(t, err)
	assert.NotContains(t, string(projected), "raw-secret", "只对模型目录工具允许 data 字符串列表")
}

func TestMCPStatisticsTokenFieldsAcceptOnlyNumericSamples(t *testing.T) {
	raw := []byte(`{"success":true,"data":{"input_tokens_total":123,"cache_read_tokens":null,"cache_write_tokens":"unmarked-private-credential","cache_token_hit_rate":{"value":"private-credential"},"access_token":321,"secret":456,"key":789}}`)
	projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_analytics_summary", Category: "channels"}, raw, 200)
	require.NoError(t, err)
	require.True(t, success)
	assert.JSONEq(t, `{"success":true,"data":{"input_tokens_total":123,"cache_read_tokens":null}}`, string(projected))
	projected, _, err = ProjectMCPResponse(MCPTool{Name: "newapi_channel_get", Category: "channels"}, raw, 200)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":{}}`, string(projected), "统计数值例外不能扩散到其它工具")
}

func TestMCPCanonicalRequestDoesNotRoundLargeNumbers(t *testing.T) {
	left, err := canonicalMCPJSON([]byte(`{"b":{"count":9007199254740992},"a":1}`))
	require.NoError(t, err)
	reordered, err := canonicalMCPJSON([]byte(`{ "a": 1, "b": { "count": 9007199254740992 } }`))
	require.NoError(t, err)
	right, err := canonicalMCPJSON([]byte(`{"a":1,"b":{"count":9007199254740993}}`))
	require.NoError(t, err)
	assert.Equal(t, left, reordered)
	assert.NotEqual(t, left, right)
}

func TestMCPPerKeyConcurrencyAndRateLimit(t *testing.T) {
	mcpLimits.Lock()
	delete(mcpLimits.keys, 90123)
	delete(mcpLimits.keys, 90124)
	mcpLimits.Unlock()
	var releases []func()
	for i := 0; i < 4; i++ {
		release, ok := AcquireMCPLimit(90123)
		require.True(t, ok)
		releases = append(releases, release)
	}
	_, ok := AcquireMCPLimit(90123)
	assert.False(t, ok)
	for _, release := range releases {
		release()
		release()
	}
	for i := 4; i < 60; i++ {
		release, ok := AcquireMCPLimit(90123)
		require.True(t, ok)
		release()
	}
	_, ok = AcquireMCPLimit(90123)
	assert.False(t, ok)
	release, ok := AcquireMCPLimit(90124)
	require.True(t, ok, "各密钥独立限流")
	release()
}
