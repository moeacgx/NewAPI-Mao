package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPEditableMetadataRoundTrip(t *testing.T) {
	for _, tc := range []struct{ tool, raw string }{
		{"newapi_group_tags_list", `{"success":true,"data":[{"id":1,"name":"标签","description":"公开说明","icons":["OpenAI","https://cdn.example/icon.svg"],"match_keywords":["gpt","推理"],"group_ids":[2,3],"sort_order":-5}]}`},
		{"newapi_model_get", `{"success":true,"data":{"id":1,"model_name":"gpt-4.1","description":"公开说明","icon":"OpenAI","tags":"chat,tools","vendor_id":2,"endpoints":"{\"openai\":{\"path\":\"/v1/chat/completions\",\"method\":\"POST\"}}","status":0,"sync_official":0,"name_rule":2}}`},
		{"newapi_vendor_get", `{"success":true,"data":{"id":1,"name":"供应商","description":"公开说明\n第二行","icon":"OpenAI","status":1,"created_time":10,"updated_time":20}}`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			projected, success, err := ProjectMCPResponse(MCPTool{Name: tc.tool}, []byte(tc.raw), 200)
			require.NoError(t, err)
			require.True(t, success)
			assert.JSONEq(t, tc.raw, string(projected), "编辑字段必须完整保留，不能静默丢失")
		})
	}
}

func TestMCPEditableMetadataFailsClosedInsteadOfPartialObject(t *testing.T) {
	for _, data := range []map[string]any{
		{"description": "Bearer sensitive-value"},
		{"description": "api_key=sensitive-value"},
		{"description": strings.Repeat("x", 8193)},
		{"icon": "https://user:pass@cdn.example/icon.svg"},
		{"icon": "https://cdn.example/icon.svg?token=sensitive"},
		{"icon": "data:image/svg+xml;sensitive"},
		{"endpoints": `{"openai":{"path":"https://private.example","method":"POST"}}`},
		{"endpoints": `{"openai":{"path":"/v1/chat","method":"POST","key":"hidden"}}`},
	} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": data})
		require.NoError(t, err)
		projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_model_get"}, raw, 200)
		require.Error(t, err)
		assert.Empty(t, projected)
	}
	raw := []byte(`{"success":true,"data":{"id":1,"description":"公开说明","icon":"OpenAI"}}`)
	projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get", Category: "channels"}, raw, 200)
	require.NoError(t, err)
	assert.NotContains(t, string(projected), "description", "专用白名单不能扩大其他工具的自由文本暴露")
}
