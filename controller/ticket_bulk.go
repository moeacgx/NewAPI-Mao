package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (api TicketAPI) authorizedAdmin(c *gin.Context) bool {
	if !api.Admin {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "Permission denied"})
		return false
	}
	return api.authorized(c)
}

func (api TicketAPI) BatchStatus(c *gin.Context) {
	if !api.authorizedAdmin(c) {
		return
	}
	var input struct {
		Tickets []model.TicketVersionRef `json:"tickets"`
		Status  string                   `json:"status"`
	}
	if !decodeTicketRequest(c, &input) {
		return
	}
	count, err := model.ChangeTicketsStatus(c.GetInt("id"), input.Tickets, input.Status)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"updated_count": count})
}

func (api TicketAPI) BatchDelete(c *gin.Context) {
	if !api.authorizedAdmin(c) {
		return
	}
	var input struct {
		Tickets []model.TicketVersionRef `json:"tickets"`
	}
	if !decodeTicketRequest(c, &input) {
		return
	}
	count, err := model.DeleteTickets(c.GetInt("id"), input.Tickets)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted_count": count})
}

func (api TicketAPI) AutoArchive(c *gin.Context) {
	if !api.authorizedAdmin(c) {
		return
	}
	config, err := model.GetTicketAutoArchiveConfig()
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, config)
}

func (api TicketAPI) UpdateAutoArchive(c *gin.Context) {
	if !api.authorizedAdmin(c) {
		return
	}
	var config model.TicketAutoArchiveConfig
	if !decodeTicketRequest(c, &config) {
		return
	}
	updated, err := model.UpdateTicketAutoArchiveConfig(config)
	if err != nil {
		writeTicketError(c, err)
		return
	}
	common.ApiSuccess(c, updated)
}
