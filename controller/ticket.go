package controller

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TicketAPI 的管理范围只由路由构造，不能由请求参数或用户角色自动切换。
type TicketAPI struct{ Admin bool }

func (api TicketAPI) authorized(c *gin.Context) bool {
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

func writeTicketError(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "Ticket service is unavailable"
	switch {
	case errors.Is(err, model.ErrTicketInvalidInput):
		status, message = http.StatusBadRequest, "Invalid ticket input"
	case errors.Is(err, gorm.ErrRecordNotFound):
		status, message = http.StatusNotFound, "Ticket not found"
	case errors.Is(err, model.ErrTicketClosed):
		status, message = http.StatusConflict, "Reopen the ticket before replying"
	case errors.Is(err, model.ErrTicketConflict):
		status, message = http.StatusConflict, "Ticket changed, please refresh"
	default:
		// 数据库错误可能带有用户正文，禁止写入通用日志或返回给客户端。
		common.SysError("ticket storage operation failed")
	}
	c.AbortWithStatusJSON(status, gin.H{"success": false, "message": message})
}

func ticketQueryInt(c *gin.Context, key string, fallback, minimum, maximum int) (int, bool) {
	raw, exists := c.GetQuery(key)
	if !exists {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		writeTicketError(c, model.ErrTicketInvalidInput)
		return 0, false
	}
	return value, true
}

func ticketPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		writeTicketError(c, model.ErrTicketInvalidInput)
		return 0, false
	}
	return id, true
}

func decodeTicketRequest(c *gin.Context, destination any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "Ticket request exceeds 64 KiB"})
			return false
		}
		writeTicketError(c, model.ErrTicketInvalidInput)
		return false
	}
	// 完整解码同时拒绝多段 JSON，避免尾随数据绕过总长度限制。
	if err := common.Unmarshal(data, destination); err != nil {
		writeTicketError(c, model.ErrTicketInvalidInput)
		return false
	}
	return true
}

func (api TicketAPI) List(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	page, ok := ticketQueryInt(c, "p", 1, 1, 1000000)
	if !ok {
		return
	}
	size, ok := ticketQueryInt(c, "page_size", 20, 1, int(^uint(0)>>1))
	if !ok {
		return
	}
	if size > 100 {
		size = 100
	}
	filter := model.TicketFilter{Status: c.Query("status"), Keyword: strings.TrimSpace(c.Query("keyword"))}
	if (filter.Status != "" && !model.ValidTicketStatus(filter.Status)) || utf8.RuneCountInString(filter.Keyword) > model.TicketTitleMaxRunes || strings.ContainsRune(filter.Keyword, 0) {
		writeTicketError(c, model.ErrTicketInvalidInput)
		return
	}
	if api.Admin {
		filter.UserID, ok = ticketQueryInt(c, "user_id", 0, 0, int(^uint(0)>>1))
		if !ok {
			return
		}
	}
	pageInfo := &common.PageInfo{Page: page, PageSize: size}
	items, total, err := model.ListTickets(c.GetInt("id"), api.Admin, filter, pageInfo)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	pageInfo.SetItems(items)
	pageInfo.SetTotal(int(total))
	common.ApiSuccess(c, pageInfo)
}

func (api TicketAPI) Create(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	var input model.TicketCreateInput
	if !decodeTicketRequest(c, &input) {
		return
	}
	ticket, err := model.CreateTicket(c.GetInt("id"), input)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

func (api TicketAPI) Detail(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	id, ok := ticketPathID(c)
	if !ok {
		return
	}
	before, ok := ticketQueryInt(c, "before_message_id", 0, 0, int(^uint(0)>>1))
	if !ok {
		return
	}
	detail, err := model.GetTicketDetail(id, c.GetInt("id"), api.Admin, before)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func (api TicketAPI) Reply(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	id, ok := ticketPathID(c)
	if !ok {
		return
	}
	var input struct {
		Content string `json:"content"`
	}
	if !decodeTicketRequest(c, &input) {
		return
	}
	message, err := model.ReplyTicket(id, c.GetInt("id"), api.Admin, input.Content)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, message)
}

func (api TicketAPI) Status(c *gin.Context) {
	if !api.authorized(c) {
		return
	}
	id, ok := ticketPathID(c)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if !decodeTicketRequest(c, &input) {
		return
	}
	ticket, err := model.ChangeTicketStatus(id, c.GetInt("id"), api.Admin, input.Status)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

// AutoReply 返回管理员工单自动回复配置。配置读取也必须经过工单管理权限。
func (api TicketAPI) AutoReply(c *gin.Context) {
	if !api.Admin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}
	if !api.authorized(c) {
		return
	}
	config, err := model.GetTicketAutoReplyConfig()
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

// UpdateAutoReply 保存管理员工单自动回复配置。
func (api TicketAPI) UpdateAutoReply(c *gin.Context) {
	if !api.Admin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return
	}
	if !api.authorized(c) {
		return
	}
	var config model.TicketAutoReplyConfig
	if !decodeTicketRequest(c, &config) {
		return
	}
	updated, err := model.UpdateTicketAutoReplyConfig(config)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}
