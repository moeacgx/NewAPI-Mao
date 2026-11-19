package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPCompletionRatioMetaManagement(t *testing.T) {
	previous := common.OptionMap
	common.OptionMap = map[string]string{
		"ModelRatio":          `{"gpt-4o":1,"local-meta-model":1}`,
		"AudioRatio":          `{"local-audio-model":1}`,
		"CompletionRatioMeta": `{"stale-secret":{"ratio":999,"locked":false}}`,
	}
	t.Cleanup(func() { common.OptionMap = previous })
	var tool service.MCPTool
	for _, item := range service.MCPTools() {
		if item.Name == "newapi_settings_list" {
			tool = item
		}
	}
	require.NotEmpty(t, tool.Name)
	for _, tc := range []struct {
		name, query string
		total       int
	}{
		{"精确查询", "?key=CompletionRatioMeta", 1},
		{"搜索查询", "?keyword=COMPLETION", 1},
		{"排序分页", "?p=2&page_size=1", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/option/"+tc.query, nil)
			req = req.WithContext(service.MCPInternalContext(req.Context(), 1))
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = req
			GetOptions(c)
			require.Equal(t, http.StatusOK, recorder.Code)
			projected, success, err := service.ProjectMCPResponse(tool, recorder.Body.Bytes(), recorder.Code)
			require.NoError(t, err)
			require.True(t, success)
			var result struct {
				Data struct {
					Items []service.MCPOptionView `json:"items"`
					Total int                     `json:"total"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(projected, &result))
			require.Len(t, result.Data.Items, 1)
			assert.Equal(t, tc.total, result.Data.Total)
			assert.Equal(t, "CompletionRatioMeta", result.Data.Items[0].Key)
			assert.False(t, result.Data.Items[0].Redacted)
			var meta map[string]ratio_setting.CompletionRatioInfo
			require.NoError(t, common.UnmarshalJsonStr(result.Data.Items[0].Value, &meta))
			require.Len(t, meta, 3)
			for _, name := range []string{"gpt-4o", "local-meta-model", "local-audio-model"} {
				assert.Equal(t, ratio_setting.GetCompletionRatioInfo(name), meta[name])
			}
			assert.NotContains(t, string(projected), "stale-secret")
		})
	}
	// 派生项不能作为持久化配置写回；此路径应在任何 DB 操作前拒绝。
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/option/", strings.NewReader(`{"key":"CompletionRatioMeta","value":"{}"}`))
	UpdateOption(c)
	var failure struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &failure))
	assert.False(t, failure.Success)
	assert.Contains(t, failure.Message, "只读")
}
