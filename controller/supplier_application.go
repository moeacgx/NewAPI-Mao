package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SupplierApplicationAPI 的管理范围只由路由构造，不能由请求参数或用户角色自动切换。
type SupplierApplicationAPI struct{ Admin bool }

func (api SupplierApplicationAPI) authorized(c *gin.Context) bool {
	if c.GetInt("id") <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authentication required"})
		return false
	}
	if api.Admin && c.GetInt("role") < common.RoleAdminUser {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return false
	}
	return true
}

func writeSupplierApplicationError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "Supplier application service is unavailable"
	switch {
	case errors.Is(err, model.ErrSupplierApplicationInvalid):
		// 错误文本只含字段路径，不回显提交内容。
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, message = http.StatusNotFound, "Supplier application not found"
	case errors.Is(err, model.ErrSupplierApplicationDuplicate):
		status, message = http.StatusConflict, "The same supplier application was already submitted"
	case errors.Is(err, model.ErrSupplierApplicationLimit):
		status, message = http.StatusTooManyRequests, "Too many supplier applications in 24 hours"
	case errors.Is(err, model.ErrSupplierApplicationConflict):
		status, message = http.StatusConflict, "Supplier application changed, please refresh"
	default:
		// 数据库错误可能带有用户提交的联系方式，禁止写入通用日志或返回给客户端。
		common.SysError("supplier application storage operation failed")
	}
	c.AbortWithStatusJSON(status, gin.H{"success": false, "message": message})
}

func supplierApplicationQueryInt(c *gin.Context, key string, fallback, minimum, maximum int) (int, bool) {
	raw, exists := c.GetQuery(key)
	if !exists {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		writeSupplierApplicationError(c, model.ErrSupplierApplicationInvalid)
		return 0, false
	}
	return value, true
}

func supplierApplicationPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		writeSupplierApplicationError(c, model.ErrSupplierApplicationInvalid)
		return 0, false
	}
	return id, true
}

func decodeSupplierApplicationRequest(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "Supplier application request exceeds 64 KiB"})
			return false
		}
		writeSupplierApplicationError(c, model.ErrSupplierApplicationInvalid)
		return false
	}
	// 完整解码同时拒绝多段 JSON，避免尾随数据绕过总长度限制。
	if err := common.Unmarshal(data, destination); err != nil {
		writeSupplierApplicationError(c, model.ErrSupplierApplicationInvalid)
		return false
	}
	return true
}

func (api SupplierApplicationAPI) List(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	page, ok := supplierApplicationQueryInt(c, "p", 1, 1, 1000000)
	if !ok {
		return
	}
	size, ok := supplierApplicationQueryInt(c, "page_size", 20, 1, int(^uint(0)>>1))
	if !ok {
		return
	}
	if size > model.SupplierApplicationMaxPageSize {
		size = model.SupplierApplicationMaxPageSize
	}
	filter := model.SupplierApplicationFilter{Status: c.Query("status"), Keyword: strings.TrimSpace(c.Query("keyword"))}
	if filter.Status != "" && !model.ValidSupplierApplicationStatus(filter.Status) {
		writeSupplierApplicationError(c, model.ErrSupplierApplicationInvalid)
		return
	}
	pageInfo := &common.PageInfo{Page: page, PageSize: size}
	if !api.Admin {
		items, total, err := model.ListSupplierApplicationsForUser(c.GetInt("id"), filter, pageInfo)
		if err != nil {
			writeSupplierApplicationError(c, err)
			return
		}
		pageInfo.SetItems(items)
		pageInfo.SetTotal(int(total))
		common.ApiSuccess(c, pageInfo)
		return
	}
	filter.UserID, ok = supplierApplicationQueryInt(c, "user_id", 0, 0, int(^uint(0)>>1))
	if !ok {
		return
	}
	items, total, err := model.ListSupplierApplicationsForAdmin(c.GetInt("id"), filter, pageInfo)
	if err != nil {
		writeSupplierApplicationError(c, err)
		return
	}
	pageInfo.SetItems(items)
	pageInfo.SetTotal(int(total))
	common.ApiSuccess(c, pageInfo)
}

// Create 只在个人路由注册；管理路由没有创建入口。
func (api SupplierApplicationAPI) Create(c *gin.Context) {
	if !api.authorized(c) || api.Admin {
		return
	}
	var input model.SupplierApplicationInput
	if !decodeSupplierApplicationRequest(c, &input) {
		return
	}
	view, err := model.CreateSupplierApplication(c.GetInt("id"), input)
	if err != nil {
		writeSupplierApplicationError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

func (api SupplierApplicationAPI) Detail(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	id, ok := supplierApplicationPathID(c)
	if !ok {
		return
	}
	if api.Admin {
		view, err := model.GetSupplierApplicationForAdmin(id)
		if err != nil {
			writeSupplierApplicationError(c, err)
			return
		}
		common.ApiSuccess(c, view)
		return
	}
	view, err := model.GetSupplierApplicationForUser(id, c.GetInt("id"))
	if err != nil {
		writeSupplierApplicationError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

// Review 记录处理状态、内部备注、处理人和处理时间；仅管理路由注册。
func (api SupplierApplicationAPI) Review(c *gin.Context) {
	if !api.Admin || !api.authorized(c) {
		return
	}
	id, ok := supplierApplicationPathID(c)
	if !ok {
		return
	}
	var review model.SupplierApplicationReview
	if !decodeSupplierApplicationRequest(c, &review) {
		return
	}
	result, err := model.ReviewSupplierApplication(id, c.GetInt("id"), review)
	if err != nil {
		writeSupplierApplicationError(c, err)
		return
	}
	if result.Changed {
		// 审计只记录标识、状态和备注长度，不写入联系方式或备注正文。
		recordManageAudit(c, "supplier_application.review", map[string]interface{}{
			"id": id, "from": result.PreviousStatus, "to": result.View.Status,
			"note_length": len([]rune(result.View.AdminNote)), "applicant_id": result.View.UserId,
		})
	}
	common.ApiSuccess(c, result.View)
}
