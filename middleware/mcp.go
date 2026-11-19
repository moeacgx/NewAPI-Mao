package middleware

import (
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const mcpIdentityContextKey = "mcp_verified_identity"

func MCPIdentity(c *gin.Context) *service.MCPIdentity {
	value, _ := c.Get(mcpIdentityContextKey)
	identity, _ := value.(*service.MCPIdentity)
	return identity
}

func MCPTransport() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		if len(c.Request.Header.Values("Origin")) > 1 {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || !isAllowedCredentialOrigin(c, origin) {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
		if c.Request.Method != http.MethodPost {
			c.Header("Allow", "POST")
			c.AbortWithStatus(http.StatusMethodNotAllowed)
			return
		}
		version := c.GetHeader("MCP-Protocol-Version")
		if len(c.Request.Header.Values("MCP-Protocol-Version")) > 1 {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if version == "" {
			version = "2025-03-26"
		}
		if !service.MCPProtocolSupported(version) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Unsupported MCP protocol version"})
			return
		}
		media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if err != nil || media != "application/json" {
			c.AbortWithStatus(http.StatusUnsupportedMediaType)
			return
		}
		jsonAccept, sseAccept := false, false
		for _, entry := range strings.Split(strings.Join(c.Request.Header.Values("Accept"), ","), ",") {
			kind, params, parseErr := mime.ParseMediaType(strings.TrimSpace(entry))
			if parseErr != nil {
				continue
			}
			if weight, exists := params["q"]; exists {
				quality, err := strconv.ParseFloat(weight, 64)
				if err != nil || !(quality > 0 && quality <= 1) {
					continue
				}
			}
			jsonAccept = jsonAccept || kind == "application/json"
			sseAccept = sseAccept || kind == "text/event-stream"
		}
		if !jsonAccept || !sseAccept {
			c.AbortWithStatus(http.StatusNotAcceptable)
			return
		}
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		identity, err := service.AuthenticateMCP(parts[1])
		if err != nil {
			c.Header("WWW-Authenticate", "Bearer")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		release, acquired := service.AcquireMCPLimit(identity.Key.ID)
		if !acquired {
			c.Header("Retry-After", "60")
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		defer release()
		if c.Request.ContentLength > service.MCPMaxUploadBytes {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MCPMaxUploadBytes)
		c.Set(mcpIdentityContextKey, identity)
		if err := model.DB.Model(&model.MCPKey{}).Where("id = ?", identity.Key.ID).Update("last_used_at", time.Now().Unix()).Error; err != nil {
			common.SysError("MCP last-use update failed")
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		c.Next()
	}
}
