package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPModelCatalogPaginationPreservesDashboardContract(t *testing.T) {
	require.Greater(t, len(openAIModels), 100, "真实内置目录应覆盖超过一页的情形")
	for _, tc := range []struct {
		name, query string
		internal    bool
		start, size int
	}{
		{"普通后台保持完整数组", "?p=2&page_size=2", false, 0, len(openAIModels)},
		{"MCP默认分页", "", true, 0, 20},
		{"MCP第二页", "?p=2&page_size=2", true, 2, 2},
		{"MCP末页", fmt.Sprintf("?p=%d&page_size=100", (len(openAIModels)-1)/100+1), true, (len(openAIModels) - 1) / 100 * 100, len(openAIModels) - (len(openAIModels)-1)/100*100},
		{"MCP超出末页", "?p=10000&page_size=100", true, len(openAIModels), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/channel/models"+tc.query, nil)
			// 外部请求头不能切换业务响应契约。
			request.Header.Set("X-MCP-Key-ID", "1")
			if tc.internal {
				request = request.WithContext(service.MCPInternalContext(request.Context(), 1))
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = request
			ChannelListModels(c)
			require.Equal(t, http.StatusOK, recorder.Code)
			if !tc.internal {
				var response struct {
					Success bool               `json:"success"`
					Data    []dto.OpenAIModels `json:"data"`
				}
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
				assert.True(t, response.Success)
				assert.Equal(t, openAIModels, response.Data)
				return
			}
			var response struct {
				Success bool `json:"success"`
				Data    struct {
					Items    []dto.OpenAIModels `json:"items"`
					Total    int                `json:"total"`
					Page     int                `json:"page"`
					PageSize int                `json:"page_size"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			assert.True(t, response.Success)
			assert.Equal(t, len(openAIModels), response.Data.Total)
			require.Len(t, response.Data.Items, tc.size)
			assert.Equal(t, openAIModels[tc.start:tc.start+tc.size], response.Data.Items)
			assert.LessOrEqual(t, recorder.Body.Len(), service.MCPMaxResponseBytes)
			assert.Greater(t, response.Data.Page, 0)
			assert.Greater(t, response.Data.PageSize, 0)
		})
	}
}

func TestMCPModelCatalogAllPagesPreserveModelIDsAfterProjection(t *testing.T) {
	var actual []string
	for page := 1; len(actual) < len(openAIModels); page++ {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/channel/models?p=%d&page_size=100", page), nil)
		request = request.WithContext(service.MCPInternalContext(request.Context(), 1))
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = request
		ChannelListModels(c)
		projected, success, err := service.ProjectMCPResponse(service.MCPTool{Name: "newapi_channel_models", Category: "channels"}, recorder.Body.Bytes(), recorder.Code)
		require.NoError(t, err)
		require.True(t, success)
		require.LessOrEqual(t, recorder.Body.Len(), service.MCPMaxResponseBytes)
		var response struct {
			Data struct {
				Items []struct {
					ID string `json:"id"`
				} `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, common.Unmarshal(projected, &response))
		require.NotEmpty(t, response.Data.Items, "到达总数前不能产生空页")
		for _, item := range response.Data.Items {
			actual = append(actual, item.ID)
		}
	}
	expected := make([]string, 0, len(openAIModels))
	for _, item := range openAIModels {
		expected = append(expected, item.Id)
	}
	assert.Equal(t, expected, actual, "真实内置模型名称必须逐个保留，包括 SparkDesk 等包含 sk- 片段的合法名称")
}
