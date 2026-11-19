package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func mcpManagementUser(c *gin.Context) *model.UserBase {
	user, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil || user.Status != common.UserStatusEnabled || user.Role < common.RoleAdminUser {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "MCP management requires an active administrator"})
		return nil
	}
	return user.ToBaseUser()
}

func mcpPagination(c *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("p", "1"))
	if err != nil || page < 1 || page > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid page"})
		return 0, 0, false
	}
	size, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || size < 1 || size > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid page_size"})
		return 0, 0, false
	}
	return (page - 1) * size, size, true
}

func GetMCPTools(c *gin.Context) {
	user := mcpManagementUser(c)
	if user == nil {
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": service.MCPVisibleTools(user, nil)})
}

func GetMCPKeys(c *gin.Context) {
	user := mcpManagementUser(c)
	if user == nil {
		return
	}
	offset, limit, ok := mcpPagination(c)
	if !ok {
		return
	}
	items, total, err := model.ListMCPKeys(user.Id, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Cannot load MCP keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": items, "total": total}})
}

func CreateMCPKey(c *gin.Context) {
	user := mcpManagementUser(c)
	if user == nil {
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, service.MCPMaxBodyBytes))
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "Request body too large"})
		return
	}
	var request struct {
		Name      string   `json:"name"`
		Tools     []string `json:"tools"`
		ExpiresAt int64    `json:"expires_at"`
	}
	if err := common.Unmarshal(raw, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid JSON request"})
		return
	}
	key, secret, err := service.CreateMCPKey(user, request.Name, request.Tools, request.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"key": key, "secret": secret}})
}

func RevokeMCPKey(c *gin.Context) {
	user := mcpManagementUser(c)
	if user == nil {
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid key ID"})
		return
	}
	err = model.RevokeMCPKey(user.Id, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "MCP key not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Cannot revoke MCP key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": nil})
}

func GetMCPLogs(c *gin.Context) {
	user := mcpManagementUser(c)
	if user == nil {
		return
	}
	offset, limit, ok := mcpPagination(c)
	if !ok {
		return
	}
	items, total, err := model.ListMCPLogs(user.Id, offset, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Cannot load MCP logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": items, "total": total}})
}
