package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPChannelModelsCSVPreservedInFull(t *testing.T) {
	models := strings.TrimSuffix(strings.Repeat("provider/model-2026:latest,", 40), ",")
	require.Greater(t, len(models), 256)
	for _, name := range []string{"newapi_channel_get", "newapi_channels_list", "newapi_channels_search"} {
		t.Run(name, func(t *testing.T) {
			data := any(map[string]any{"id": 7, "models": models, "key": "private-key"})
			if name != "newapi_channel_get" {
				data = map[string]any{"items": []any{data}, "total": 1}
			}
			raw, err := common.Marshal(map[string]any{"success": true, "data": data})
			require.NoError(t, err)
			projected, success, err := ProjectMCPResponse(MCPTool{Name: name, Category: "channels"}, raw, 200)
			require.NoError(t, err)
			require.True(t, success)
			var response struct {
				Data struct {
					Models string `json:"models"`
					Items  []struct {
						Models string `json:"models"`
					} `json:"items"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(projected, &response))
			actual := response.Data.Models
			if len(response.Data.Items) > 0 {
				actual = response.Data.Items[0].Models
			}
			assert.Equal(t, models, actual, "不能截断或替换会用于后续更新的模型 CSV")
			assert.NotContains(t, string(projected), "private-key")
		})
	}
}

func TestMCPChannelModelsCSVUnsafeFailsAsWhole(t *testing.T) {
	for _, models := range []any{
		strings.Repeat("m", 8193), "model\nAuthorization: private", "model,sk-private123456",
		"https://user:private@upstream.example/model", `{"key":"private"}`, []string{"model"}, nil,
	} {
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"id": 7, "models": models}})
		require.NoError(t, err)
		projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get", Category: "channels"}, raw, 200)
		require.Error(t, err)
		assert.Empty(t, projected, "不返回可被误用于完整回写的残缺对象")
	}
}

func TestMCPChannelModelsHistoricalLongNameRemainsReadable(t *testing.T) {
	raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"models": strings.Repeat("m", 512)}})
	require.NoError(t, err)
	projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get"}, raw, 200)
	require.NoError(t, err)
	require.True(t, success)
	assert.JSONEq(t, string(raw), string(projected), "历史名称可以完整诊断，新写入仍须遵守宿主 255 字节限制")
}

func TestMCPChannelTestPreservesSafeDiagnostics(t *testing.T) {
	for _, success := range []bool{true, false} {
		raw, err := common.Marshal(map[string]any{
			"success": success, "time": 1.25, "error_code": "channel:no_available_key",
			"upstream_response_model_name": "provider/model-2026", "message": "Bearer private-credential",
			"data": map[string]any{"key": "private"}, "request": "private",
		})
		require.NoError(t, err)
		projected, ok, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_test", Category: "channels"}, raw, 200)
		require.NoError(t, err)
		assert.Equal(t, success, ok)
		var result map[string]any
		require.NoError(t, common.Unmarshal(projected, &result))
		assert.Equal(t, 1.25, result["time"])
		assert.Equal(t, "provider/model-2026", result["upstream_response_model_name"])
		if !success {
			assert.Equal(t, "channel:no_available_key", result["error_code"])
			assert.Equal(t, "Business request failed", result["message"])
		}
		assert.NotContains(t, string(projected), "private")
	}
	// 同样形状的失败正文不能扩大其它工具的错误信息出口。
	projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get"}, []byte(`{"success":false,"time":1.25,"upstream_response_model_name":"private","error_code":"bad_response"}`), 200)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":false,"status":200,"message":"Business request failed"}`, string(projected))
}

func TestMCPChannelTestRejectsUntrustedDiagnosticValues(t *testing.T) {
	for _, code := range []any{"private_credential", "Bearer private", "https://private.example", 123, map[string]any{"key": "private"}} {
		raw, err := common.Marshal(map[string]any{
			"success": false, "error_code": code, "time": map[string]any{"secret": "private"},
			"upstream_response_model_name": "Bearer private", "message": "private",
		})
		require.NoError(t, err)
		projected, ok, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_test"}, raw, 200)
		require.NoError(t, err)
		assert.False(t, ok)
		assert.NotContains(t, string(projected), "private")
		assert.NotContains(t, string(projected), `"error_code"`, "只保留宿主固定错误枚举")
		assert.NotContains(t, string(projected), `"time"`, "时间必须为数值，不能成为原文出口")
	}
}

func TestMCPMissingModelsAndModelBindingsRemainUsable(t *testing.T) {
	for _, tc := range []struct{ tool, raw, expected string }{
		{"newapi_models_missing", `{"success":true,"data":["provider/model:latest","task-model","中文模型"]}`, `{"success":true,"data":["provider/model:latest","task-model","中文模型"]}`},
		{"newapi_model_get", `{"success":true,"data":{"id":1,"bound_channels":[{"name":"绑定渠道","type":1,"status":2,"key":"private"}],"matched_models":["provider/model-a","provider/model-b"],"matched_count":2,"quota_types":[0,1]}}`, `{"success":true,"data":{"id":1,"bound_channels":[{"name":"绑定渠道","type":1}],"matched_models":["provider/model-a","provider/model-b"],"matched_count":2,"quota_types":[0,1]}}`},
		{"newapi_models_list", `{"success":true,"data":{"items":[{"id":1,"matched_models":["model"],"matched_count":1,"quota_types":[0]}]}}`, `{"success":true,"data":{"items":[{"id":1,"matched_models":["model"],"matched_count":1,"quota_types":[0]}]}}`},
		{"newapi_models_search", `{"success":true,"data":[{"id":1,"bound_channels":[{"name":"绑定渠道","type":2}]}]}`, `{"success":true,"data":[{"id":1,"bound_channels":[{"name":"绑定渠道","type":2}]}]}`},
	} {
		t.Run(tc.tool, func(t *testing.T) {
			projected, success, err := ProjectMCPResponse(MCPTool{Name: tc.tool, Category: "models"}, []byte(tc.raw), 200)
			require.NoError(t, err)
			require.True(t, success)
			assert.JSONEq(t, tc.expected, string(projected))
		})
	}
}

func TestMCPModelNameContainersDoNotExposeArbitraryPayloads(t *testing.T) {
	for _, tc := range []struct{ tool, data string }{
		{"newapi_models_missing", `["Bearer private"]`},
		{"newapi_models_missing", `[{"name":"private"}]`},
		{"newapi_model_get", `{"matched_models":[{"name":"private"}]}`},
		{"newapi_model_get", `{"quota_types":[{"key":"private"}]}`},
		{"newapi_model_get", `{"matched_count":"private"}`},
	} {
		projected, _, err := ProjectMCPResponse(MCPTool{Name: tc.tool, Category: "models"}, []byte(`{"success":true,"data":`+tc.data+`}`), 200)
		require.Error(t, err)
		assert.Empty(t, projected)
	}
	projected, _, err := ProjectMCPResponse(MCPTool{Name: "newapi_users_list"}, []byte(`{"success":true,"data":["private"],"bound_channels":[{"name":"private"}],"matched_models":["private"]}`), 200)
	require.NoError(t, err)
	assert.JSONEq(t, `{"success":true,"data":[null]}`, string(projected), "模型关联出口不能扩散到其它工具")
}

func TestMCPModelMetadataNamesDoNotConfuseTaskWithCredential(t *testing.T) {
	var createTool MCPTool
	for _, candidate := range MCPTools() {
		if candidate.Name == "newapi_model_create" {
			createTool = candidate
		}
	}
	require.NotEmpty(t, createTool.Name)
	for _, tc := range []struct {
		name  any
		valid bool
	}{
		{"task-model", true},
		{"provider/task-model", true},
		{"sk-private123456", false},
		{"Bearer private-credential", false},
		{nil, false},
		{[]string{"task-model"}, false},
		{123, false},
	} {
		input, err := common.Marshal(map[string]any{"body": map[string]any{"model_name": tc.name}, "request_id": "model-name"})
		require.NoError(t, err)
		_, _, _, inputErr := PrepareMCPRequest(createTool, input)
		raw, err := common.Marshal(map[string]any{"success": true, "data": map[string]any{"model_name": tc.name}})
		require.NoError(t, err)
		projected, _, outputErr := ProjectMCPResponse(MCPTool{Name: "newapi_model_get"}, raw, 200)
		if tc.valid {
			require.NoError(t, inputErr)
			require.NoError(t, outputErr)
			assert.JSONEq(t, string(raw), string(projected), "模型名在创建和查询时必须使用相同语义")
		} else {
			require.Error(t, inputErr)
			require.Error(t, outputErr)
			assert.Empty(t, projected)
		}
	}
}
