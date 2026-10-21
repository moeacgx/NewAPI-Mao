package controller

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/activerequests"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetSelfActiveRequests(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authentication required"})
		return
	}
	query := activerequests.Query{Page: 1, PageSize: 10, Search: strings.TrimSpace(c.Query("search")), Status: c.Query("status")}
	for _, parameter := range []struct {
		name  string
		value *int
	}{
		{"page", &query.Page}, {"page_size", &query.PageSize}, {"token_id", &query.TokenID},
	} {
		if raw, present := c.GetQuery(parameter.name); present {
			value, err := strconv.Atoi(raw)
			if err != nil || value <= 0 || parameter.name == "page_size" && value > 50 {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid active request query"})
				return
			}
			*parameter.value = value
		}
	}
	if utf8.RuneCountInString(query.Search) > 256 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Active request search is too long"})
		return
	}
	switch query.Status {
	case "", "waiting_response", "receiving", "processing", "retrying", "completed":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid active request status"})
		return
	}
	page, err := service.ListUserActiveRequests(c.Request.Context(), userID, query)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Active requests are temporarily unavailable"})
		return
	}
	common.ApiSuccess(c, page)
}
