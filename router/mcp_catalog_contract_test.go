package router

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPCatalogEveryBusinessToolHasRegisteredRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetApiRouter(engine)
	routes := make(map[string]bool)
	for _, route := range engine.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	names := make(map[string]bool)
	for _, tool := range service.MCPTools() {
		t.Run(tool.Name, func(t *testing.T) {
			require.False(t, names[tool.Name], "工具名称必须唯一，否则发现与实际调用可能错配")
			names[tool.Name] = true
			if tool.Name == "newapi_context" {
				return
			}
			assert.True(t, routes[tool.Method+" "+tool.Path], "工具引用未注册的业务接口：%s %s", tool.Method, tool.Path)
			assert.GreaterOrEqual(t, tool.MinRole, common.RoleAdminUser)
			if tool.Method != http.MethodGet {
				readOnlyPOST := false
				// 这些宿主 POST 仅查询或生成未保存草稿，绝不触发持久化业务。
				switch tool.Name {
				case "newapi_security_delete_preview", "newapi_my_token_key",
					"newapi_channel_keys_status", "newapi_groups_code_migration_preview", "newapi_groups_token_migration_preview",
					"newapi_prompt_archive_import", "newapi_prompt_store_import", "newapi_prompt_repository_import",
					"newapi_notification_preview", "newapi_oauth_discovery":
					readOnlyPOST = true
				}
				assert.True(t, tool.Write || readOnlyPOST, "非读取接口必须显式标记写操作")
			}
		})
	}
}

func TestMCPCatalogDoesNotExposeCredentialAndUnrestrictedOperations(t *testing.T) {
	for _, tool := range service.MCPTools() {
		assert.NotContains(t, tool.Path, "/api/mcp", "工具不能给自己签发新权限")
		assert.NotContains(t, tool.Path, "/proxy/", "不提供任意扩展代理")
		if tool.Path == "/api/option/" {
			assert.Equal(t, common.RoleRootUser, tool.MinRole, "系统设置仅允许 Root 独立授权")
		}
		assert.NotEqual(t, "/api/user/token", tool.Path, "不生成管理 PAT")
		assert.NotEqual(t, "/api/channel/:id/key", tool.Path, "不导出渠道密钥")
		if tool.Path == "/api/token/:id/key" {
			assert.Equal(t, "newapi_my_token_key", tool.Name)
			assert.Equal(t, common.RoleRootUser, tool.MinRole)
			assert.False(t, tool.Write, "敏感读取不能将密钥存入写回执")
		}
		assert.NotEqual(t, "/api/token/batch/keys", tool.Path, "不批量导出推理密钥")
		assert.NotEqual(t, "/api/user/self", tool.Path, "context 不应触发返佣结算")
	}
}

func TestMCPCatalogActiveGETOperationsRequireWriteAuthorization(t *testing.T) {
	for _, name := range []string{"newapi_channel_test", "newapi_channel_balance_refresh"} {
		t.Run(name, func(t *testing.T) {
			var selected *service.MCPTool
			for _, tool := range service.MCPTools() {
				if tool.Name == name {
					selected = &tool
					break
				}
			}
			require.NotNil(t, selected)
			assert.Equal(t, http.MethodGet, selected.Method)
			assert.True(t, selected.Write)
			assert.Equal(t, authz.ChannelOperate, selected.Permission)
			assert.Contains(t, selected.InputSchema["required"], "request_id")
		})
	}
}
