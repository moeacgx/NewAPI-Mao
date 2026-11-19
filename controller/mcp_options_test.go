package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPOptionsPagingAndCredentialProjection(t *testing.T) {
	previous := common.OptionMap
	common.OptionMap = map[string]string{
		"SiteName":           "本地站点",
		"SMTPPassword":       "never-return-this",
		"MaxTokenAutoGroups": "0",
		"ModelRatio":         `{"SparkDesk-v4.0":1.2}`,
	}
	t.Cleanup(func() { common.OptionMap = previous })
	for _, tc := range []struct {
		name, query string
		internal    bool
	}{
		{"分页读取", "?p=2&page_size=2", true},
		{"单键读取", "?key=ModelRatio", true},
		{"敏感键标记", "?key=SMTPPassword", true},
		{"HTTP头无法伪造分页", "?key=ModelRatio", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/option/"+tc.query, nil)
			request.Header.Set("X-MCP-Key-ID", "1")
			if tc.internal {
				request = request.WithContext(service.MCPInternalContext(request.Context(), 1))
			}
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = request
			GetOptions(c)
			require.Equal(t, 200, recorder.Code)
			if !tc.internal {
				var response struct {
					Data []struct {
						Key string `json:"key"`
					} `json:"data"`
				}
				require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
				require.Greater(t, len(response.Data), 1, "普通后台保留完整数组形状")
				return
			}
			assert.NotContains(t, recorder.Body.String(), "never-return-this")
			var response struct {
				Data struct {
					Items []service.MCPOptionView `json:"items"`
					Total int                     `json:"total"`
					Page  int                     `json:"page"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			switch tc.name {
			case "分页读取":
				assert.Equal(t, 5, response.Data.Total)
				assert.Equal(t, 2, response.Data.Page)
				require.Len(t, response.Data.Items, 2)
			case "单键读取":
				require.Len(t, response.Data.Items, 1)
				assert.Equal(t, `{"SparkDesk-v4.0":1.2}`, response.Data.Items[0].Value)
				assert.False(t, response.Data.Items[0].Redacted)
			case "敏感键标记":
				require.Len(t, response.Data.Items, 1)
				assert.True(t, response.Data.Items[0].Redacted)
				assert.Empty(t, response.Data.Items[0].Value)
			}
		})
	}
}
