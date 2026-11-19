package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPMetadataWriteContracts(t *testing.T) {
	for _, tc := range []struct {
		name, tool, arguments string
		valid                 bool
	}{
		{"新增模型", "newapi_model_create", `{"body":{"model_name":"local-model","status":1,"sync_official":0,"name_rule":0},"request_id":"model-create"}`, true},
		{"模型名必填", "newapi_model_create", `{"body":{"description":"仅有说明"},"request_id":"model-create"}`, false},
		{"模型字段类型", "newapi_model_create", `{"body":{"model_name":"local-model","status":"1"},"request_id":"model-create"}`, false},
		{"模型规则枚举", "newapi_model_create", `{"body":{"model_name":"local-model","name_rule":9},"request_id":"model-create"}`, false},
		{"模型更新不能只传名称", "newapi_model_update", `{"body":{"id":1,"model_name":"renamed"},"request_id":"model-update"}`, false},
		{"新增供应商", "newapi_vendor_create", `{"body":{"name":"本地供应商","status":1},"request_id":"vendor-create"}`, true},
		{"供应商名必填", "newapi_vendor_create", `{"body":{"status":1},"request_id":"vendor-create"}`, false},
		{"供应商更新不能清空其他字段", "newapi_vendor_update", `{"body":{"id":1,"name":"新名称"},"request_id":"vendor-update"}`, false},
		{"供应商完整更新", "newapi_vendor_update", `{"body":{"id":1,"name":"新名称","description":"","icon":"","status":1},"request_id":"vendor-update"}`, true},
		{"标签分组必须数组", "newapi_group_tag_create", `{"body":{"name":"标签","group_ids":"1"},"request_id":"tag-create"}`, false},
		{"标签完整更新", "newapi_group_tag_update", `{"path":{"id":"1"},"body":{"name":"标签","description":"","icons":[],"sort_order":0,"group_ids":[],"match_keywords":[]},"request_id":"tag-update"}`, true},
		{"标签更新防止清空绑定", "newapi_group_tag_update", `{"path":{"id":"1"},"body":{"name":"标签"},"request_id":"tag-update"}`, false},
		{"不接收只读模型字段", "newapi_model_create", `{"body":{"model_name":"local-model","bound_channels":[]},"request_id":"model-create"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var selected MCPTool
			for _, tool := range MCPTools() {
				if tool.Name == tc.tool {
					selected = tool
					break
				}
			}
			require.NotEmpty(t, selected.Name)
			_, _, _, err := PrepareMCPRequest(selected, []byte(tc.arguments))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMCPModelCatalogNamesUseModelSpecificValidation(t *testing.T) {
	for _, tool := range []string{"newapi_channel_models", "newapi_channel_models_enabled"} {
		for _, name := range []string{"SparkDesk-v4.0", "provider/task-model", "sk-private123456", "Bearer private"} {
			t.Run(tool+"/"+name, func(t *testing.T) {
				var data any = []string{name}
				if tool == "newapi_channel_models" {
					data = map[string]any{"items": []any{map[string]any{"id": name}}}
				}
				raw, err := common.Marshal(map[string]any{"success": true, "data": data})
				require.NoError(t, err)
				projected, _, err := ProjectMCPResponse(MCPTool{Name: tool, Category: "channels"}, raw, 200)
				if strings.HasPrefix(name, "sk-") || strings.HasPrefix(name, "Bearer ") {
					require.Error(t, err)
					assert.Empty(t, projected)
					return
				}
				require.NoError(t, err)
				assert.JSONEq(t, string(raw), string(projected))
			})
		}
	}
}

func TestMCPChannelTestParameters(t *testing.T) {
	var selected MCPTool
	for _, tool := range MCPTools() {
		if tool.Name == "newapi_channel_test" {
			selected = tool
			break
		}
	}
	require.NotEmpty(t, selected.Name)
	for _, tc := range []struct {
		name, query string
		valid       bool
	}{
		{"自动选择协议", `{}`, true},
		{"指定模型及流式", `{"model":"local-model","endpoint_type":"openai","stream":"true"}`, true},
		{"向量测试", `{"model":"local-embedding","endpoint_type":"embeddings","stream":"false"}`, true},
		{"长渠道模型名", `{"model":"` + strings.Repeat("m", 255) + `"}`, true},
		{"模型名超出上限", `{"model":"` + strings.Repeat("m", 256) + `"}`, false},
		{"不支持视频任务测试", `{"endpoint_type":"openai-video"}`, false},
		{"不接受任意上游地址", `{"base_url":"http://127.0.0.1"}`, false},
		{"不接受数字布尔值", `{"stream":true}`, false},
		{"不静默忽略拼写错误", `{"endpoint_typ":"openai"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := PrepareMCPRequest(selected, []byte(`{"path":{"id":"1"},"query":`+tc.query+`,"request_id":"test-channel"}`))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
