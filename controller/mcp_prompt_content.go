package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// getMCPPromptContent 仅从已有 AdminAuth 路由的 MCP 私有上下文分支调用。
// 外部请求头不参与分流；正文授权仍由工具白名单及各自响应投影控制。
func getMCPPromptContent(c *gin.Context) {
	query := c.Request.URL.Query()
	if ids, exists := query["id"]; exists {
		id, err := strconv.ParseInt(ids[0], 10, 32)
		if err != nil || id <= 0 || len(ids) != 1 || query.Has("p") || query.Has("page_size") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "提示词 ID 或查询参数无效"})
			return
		}
		item, err := model.GetAdminPromptTemplate(int(id))
		if errors.Is(err, model.ErrPromptTemplateNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "提示词不存在"})
			return
		}
		if err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, item)
		return
	}
	rawPage, rawSize := "1", "5"
	if query.Has("p") {
		rawPage = query.Get("p")
	}
	if query.Has("page_size") {
		rawSize = query.Get("page_size")
	}
	page, pageErr := strconv.Atoi(rawPage)
	pageSize, sizeErr := strconv.Atoi(rawSize)
	if pageErr != nil || sizeErr != nil || page < 1 || page > 10000 || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "提示词分页参数无效"})
		return
	}
	items, total, err := model.ListMCPPromptContentPage(page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}
