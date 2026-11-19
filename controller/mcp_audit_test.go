package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMCPManualAuditIdentifiesOnlyInternalCredential(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/channel/", nil)
	c.Request.Header.Set("Authorization", "Bearer mcp_external-fixture")
	assert.Equal(t, "session", auditOperatorInfo(c)["auth_method"], "外部 Header 不能伪造内部 MCP 身份")
	c.Set("use_access_token", true)
	assert.Equal(t, "access_token", auditOperatorInfo(c)["auth_method"])
	c.Request = c.Request.WithContext(service.MCPInternalContext(c.Request.Context(), 42))
	assert.Equal(t, "mcp", auditOperatorInfo(c)["auth_method"], "手工审计与中间件审计使用相同身份来源")
}
