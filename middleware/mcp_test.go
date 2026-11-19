package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPAuditAuthMethodUsesOnlyPrivateIdentity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		keyID       int
		accessToken bool
		expected    string
	}{
		{"MCP内部身份", 7, false, "mcp"},
		{"MCP身份优先", 7, true, "mcp"},
		{"普通会话", 0, false, "session"},
		{"后台令牌", 0, true, "access_token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/api/vendors/", nil)
			// HTTP 头不能伪装成进程内 MCP 身份。
			c.Request.Header.Set("X-MCP-Key-ID", "7")
			c.Set("use_access_token", tc.accessToken)
			if tc.keyID != 0 {
				c.Request = c.Request.WithContext(service.MCPInternalContext(c.Request.Context(), tc.keyID))
			}
			require.NotNil(t, c.Request)
			assert.Equal(t, tc.expected, auditAuthMethod(c))
			assert.Empty(t, c.GetString("session_id"))
		})
	}
}
