package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPModelVendorZeroIsScopedToMetadataBody(t *testing.T) {
	for _, name := range []string{"newapi_model_create", "newapi_model_update"} {
		t.Run(name, func(t *testing.T) {
			var tool MCPTool
			for _, candidate := range MCPTools() {
				if candidate.Name == name {
					tool = candidate
				}
			}
			require.NotEmpty(t, tool.Name)
			for _, vendorID := range []int{0, 7} {
				body := map[string]any{"model_name": "vendor/model", "description": "", "icon": "", "tags": "", "vendor_id": vendorID, "endpoints": "", "status": 1, "sync_official": 0, "name_rule": 0}
				if name == "newapi_model_update" {
					body["id"] = 12
				}
				raw, err := common.Marshal(map[string]any{"body": body, "request_id": "model-vendor"})
				require.NoError(t, err)
				_, _, preparedBody, err := PrepareMCPRequest(tool, raw)
				require.NoError(t, err, "模型元数据的 vendor_id=0 表示未绑定")
				var prepared map[string]any
				require.NoError(t, common.Unmarshal(preparedBody, &prepared))
				assert.EqualValues(t, vendorID, prepared["vendor_id"])
			}
		})
	}
	for _, tool := range MCPTools() {
		switch tool.Name {
		case "newapi_model_update":
			_, _, _, err := PrepareMCPRequest(tool, []byte(`{"body":{"id":0,"model_name":"model","description":"","icon":"","tags":"","vendor_id":0,"endpoints":"","status":1,"sync_official":0,"name_rule":0},"request_id":"zero-model"}`))
			require.Error(t, err, "模型本身的 ID 仍不能为零")
		case "newapi_channel_create":
			_, _, _, err := PrepareMCPRequest(tool, []byte(`{"body":{"mode":"single","channel":{"name":"渠道","type":1,"key":"test-key","models":"model","group":"default","vendor_id":0}},"request_id":"zero-channel-vendor"}`))
			require.Error(t, err, "不能把模型元数据例外扩大到渠道嵌套对象")
		}
	}
}

func TestMCPMetadataInputsMustBeSafelyProjectableBeforeExecution(t *testing.T) {
	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"newapi_model_create", map[string]any{"model_name": "model", "endpoints": `{"openai":{"path":"https://private.example/v1","method":"POST"}}`}},
		{"newapi_model_create", map[string]any{"model_name": "model", "endpoints": `{"openai":{"path":"/v1/chat","method":"POST","key":"hidden"}}`}},
		{"newapi_model_create", map[string]any{"model_name": "model", "description": strings.Repeat("中", 2800)}},
		{"newapi_vendor_create", map[string]any{"name": "供应商", "description": "api_key=private-credential"}},
		{"newapi_vendor_update", map[string]any{"id": 7, "name": "供应商", "description": "", "icon": "https://cdn.example/icon?token=private", "status": 1}},
		{"newapi_group_tag_create", map[string]any{"name": "标签", "icons": []string{"data:image/svg+xml;private"}}},
		{"newapi_group_tag_update", map[string]any{"name": "标签", "description": "", "icons": []string{}, "sort_order": 0, "group_ids": []int{1}, "match_keywords": []string{"Bearer private-credential"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tool MCPTool
			for _, candidate := range MCPTools() {
				if candidate.Name == tc.name {
					tool = candidate
				}
			}
			require.NotEmpty(t, tool.Name)
			input := map[string]any{"body": tc.body, "request_id": "unsafe-metadata"}
			if tc.name == "newapi_group_tag_update" {
				input["path"] = map[string]string{"id": "7"}
			}
			raw, err := common.Marshal(input)
			require.NoError(t, err)
			arguments, _, _, err := PrepareMCPRequest(tool, raw)
			require.Error(t, err, "不安全元数据应在路由执行和去重占用之前拒绝")
			assert.Nil(t, arguments)
		})
	}
}

func TestMCPChannelModelWritesRemainReadable(t *testing.T) {
	boundaryNames := make([]string, 0, 33)
	for i := 0; i < 31; i++ {
		prefix := fmt.Sprintf("model-%02d-", i)
		boundaryNames = append(boundaryNames, prefix+strings.Repeat("m", 255-len(prefix)))
	}
	boundaryNames = append(boundaryNames, strings.Repeat("n", 127), strings.Repeat("p", 128))
	boundaryCSV := strings.Join(boundaryNames, ",")
	require.Len(t, boundaryCSV, 8192)
	for _, name := range []string{"newapi_channel_create", "newapi_channel_update"} {
		var tool MCPTool
		for _, candidate := range MCPTools() {
			if candidate.Name == name {
				tool = candidate
			}
		}
		require.NotEmpty(t, tool.Name)
		for _, tc := range []struct {
			models string
			valid  bool
		}{
			{"provider/model:latest,task-model,中文模型", true},
			{boundaryCSV, true},
			{strings.Repeat("中", 85), true},
			{strings.Repeat("m", 256), false},
			{strings.Repeat("中", 86), false},
			{strings.Repeat("中", 2800), false},
			{"model,sk-private123456", false},
			{"model\nAuthorization: private", false},
		} {
			t.Run(name, func(t *testing.T) {
				body := map[string]any{"id": 7, "models": tc.models}
				if name == "newapi_channel_create" {
					body = map[string]any{"mode": "single", "channel": map[string]any{"name": "渠道", "type": 1, "key": "test-key", "models": tc.models, "group": "default"}}
				}
				raw, err := common.Marshal(map[string]any{"body": body, "request_id": "channel-models"})
				require.NoError(t, err)
				_, _, preparedBody, err := PrepareMCPRequest(tool, raw)
				if !tc.valid {
					require.Error(t, err, "创建和更新不能写入随后无法安全读取的模型列表")
					return
				}
				require.NoError(t, err)
				response, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"id": 7, "models": tc.models}})
				require.NoError(t, err)
				projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get", Category: "channels"}, response, 200)
				require.NoError(t, err)
				require.True(t, success)
				assert.Contains(t, string(preparedBody), tc.models)
				assert.JSONEq(t, string(response), string(projected), "允许写入的列表必须能完整读回")
			})
		}
	}
}

func TestMCPChannelModelWritesRejectEmptyCSVEntries(t *testing.T) {
	for _, name := range []string{"newapi_channel_create", "newapi_channel_update"} {
		var tool MCPTool
		for _, candidate := range MCPTools() {
			if candidate.Name == name {
				tool = candidate
			}
		}
		require.NotEmpty(t, tool.Name)
		for _, tc := range []struct{ label, models string }{
			{"整体空值", ""},
			{"整体空白", "   "},
			{"仅分隔符", ","},
			{"连续分隔符", "gpt-4o-mini,,sdk-fail"},
			{"开头分隔符", ",gpt-4o-mini"},
			{"末尾分隔符", "gpt-4o-mini,"},
			{"中间空白项", "gpt-4o-mini, ,sdk-fail"},
			{"中间Unicode空白项", "gpt-4o-mini,\u3000,sdk-fail"},
			{"开头空白项", " ,gpt-4o-mini"},
			{"末尾空白项", "gpt-4o-mini, "},
		} {
			t.Run(name+"/"+tc.label, func(t *testing.T) {
				body := map[string]any{"id": 7, "models": tc.models}
				if name == "newapi_channel_create" {
					body = map[string]any{"mode": "single", "channel": map[string]any{"name": "渠道", "type": 1, "key": "test-key", "models": tc.models, "group": "default"}}
				}
				raw, err := common.Marshal(map[string]any{"body": body, "request_id": "empty-channel-models"})
				require.NoError(t, err)
				arguments, _, _, err := PrepareMCPRequest(tool, raw)
				require.Error(t, err, "必须在真实路由执行前拒绝空模型，不能新增空 ability")
				assert.Nil(t, arguments)
			})
		}
		if name == "newapi_channel_update" {
			_, _, body, err := PrepareMCPRequest(tool, []byte(`{"body":{"id":7,"name":"保持原模型"},"request_id":"omit-channel-models"}`))
			require.NoError(t, err, "部分更新省略 models 不等于清空模型")
			assert.JSONEq(t, `{"id":7,"name":"保持原模型"}`, string(body))
		}
	}
	for _, models := range []string{"", "gpt-4o-mini,,sdk-fail", ",gpt-4o-mini,", "gpt-4o-mini, ,sdk-fail"} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"id": 7, "models": models}})
		require.NoError(t, err)
		projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get"}, raw, 200)
		require.NoError(t, err, "只拒绝新写入，历史配置仍应完整可读")
		require.True(t, success)
		assert.JSONEq(t, string(raw), string(projected))
	}
}

func TestMCPArgumentBoundaries(t *testing.T) {
	var tool MCPTool
	for _, candidate := range MCPTools() {
		if candidate.Name == "newapi_channel_status_set" {
			tool = candidate
		}
	}
	require.NotEmpty(t, tool.Name)
	for _, tc := range []struct {
		name, input string
		valid       bool
	}{
		{"正确状态", `{"path":{"id":"42"},"body":{"status":1},"request_id":"op-1"}`, true},
		{"枚举越界", `{"path":{"id":"42"},"body":{"status":9},"request_id":"op-1"}`, false},
		{"穿越", `{"path":{"id":"../42"},"body":{"status":1},"request_id":"op-1"}`, false},
		{"编码穿越", `{"path":{"id":"%2f42"},"body":{"status":1},"request_id":"op-1"}`, false},
		{"ID类型", `{"path":{"id":42},"body":{"status":1},"request_id":"op-1"}`, false},
		{"ID非数字", `{"path":{"id":"name"},"body":{"status":1},"request_id":"op-1"}`, false},
		{"分页过大", `{"path":{"id":"42"},"query":{"page_size":"101"},"body":{"status":1},"request_id":"op-1"}`, false},
		{"旧分页别名", `{"path":{"id":"42"},"query":{"ps":"-1"},"body":{"status":1},"request_id":"op-1"}`, false},
		{"额外参数", `{"path":{"id":"42"},"body":{"status":1,"key":"hidden"},"request_id":"op-1"}`, false},
		{"缺少幂等ID", `{"path":{"id":"42"},"body":{"status":1}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := PrepareMCPRequest(tool, []byte(tc.input))
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestMCPCreateChannelSchemaPreventsNilAndBatch(t *testing.T) {
	var tool MCPTool
	for _, candidate := range MCPTools() {
		if candidate.Name == "newapi_channel_create" {
			tool = candidate
		}
	}
	require.NotEmpty(t, tool.Name)
	for _, body := range []string{`{}`, `{"mode":"single","channel":null}`, `{"mode":"batch","channel":{"name":"test","type":1,"key":"test-secret","models":"model-a","group":"default"}}`, `{"mode":"single","channel":{"name":"test"}}`} {
		_, _, _, err := PrepareMCPRequest(tool, []byte(`{"body":`+body+`,"request_id":"create-1"}`))
		require.Error(t, err)
	}
	_, _, _, err := PrepareMCPRequest(tool, []byte(`{"body":{"mode":"single","channel":{"name":"test","type":1,"key":"test-secret","models":"model-a","group":"default"}},"request_id":"create-1"}`))
	require.NoError(t, err)
}

func TestMCPResponseProjection(t *testing.T) {
	input := []byte(`{"success":true,"message":"secret message","data":{"id":1,"name":"渠道","key":"short","key_preview":"x","header_override":{"Authorization":"secret"},"other":"{\"key\":\"nested\"}","content":"secret","properties":{"data":"secret"},"email":"private","payout_account":"private","source":"script","items":[{"id":2,"secret":"hidden","status":1}]}}`)
	output, success, err := ProjectMCPResponse(MCPTool{Category: "channels"}, input, 200)
	require.NoError(t, err)
	assert.True(t, success)
	assert.Contains(t, string(output), `"id":1`)
	assert.Contains(t, string(output), `"id":2`)
	for _, sensitive := range []string{"secret", "short", "private", "nested", "script", "key_preview", "header_override", "properties"} {
		assert.NotContains(t, string(output), sensitive)
	}
}
