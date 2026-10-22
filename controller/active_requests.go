package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/activerequests"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetSelfActiveRequests(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authentication required"})
		return
	}
	query, valid := parseActiveRequestQuery(c)
	if !valid {
		return
	}
	page, err := service.ListUserActiveRequests(c.Request.Context(), userID, query)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Active requests are temporarily unavailable"})
		return
	}
	common.ApiSuccess(c, page)
}

func GetAdminActiveRequests(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if c.GetInt("id") <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Authentication required"})
		return
	}
	if c.GetInt("role") < common.RoleAdminUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Administrator privileges required"})
		return
	}
	parameters := c.Request.URL.Query()
	ids, hasID := parameters["user_id"]
	names, hasUsername := parameters["username"]
	if hasID == hasUsername || hasID && len(ids) != 1 || hasUsername && len(names) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Exactly one user ID or username is required"})
		return
	}
	userID, username := 0, ""
	if hasID {
		var err error
		userID, err = strconv.Atoi(ids[0])
		if err != nil || userID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid user ID"})
			return
		}
	} else {
		username = strings.TrimSpace(names[0])
		if username == "" || utf8.RuneCountInString(username) > model.UserNameMaxLength {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid username"})
			return
		}
	}
	query, valid := parseActiveRequestQuery(c)
	if !valid {
		return
	}
	subject, err := model.GetUserIdentity(c.Request.Context(), userID, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "User not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Active requests are temporarily unavailable"})
		return
	}
	page, err := service.ListUserActiveRequests(c.Request.Context(), subject.ID, query)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Active requests are temporarily unavailable"})
		return
	}
	common.ApiSuccess(c, struct {
		activerequests.Page
		Subject *model.UserIdentity `json:"subject"`
	}{Page: page, Subject: subject})
}

func parseActiveRequestQuery(c *gin.Context) (activerequests.Query, bool) {
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
				return query, false
			}
			*parameter.value = value
		}
	}
	if utf8.RuneCountInString(query.Search) > 256 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Active request search is too long"})
		return query, false
	}
	switch query.Status {
	case "", "waiting_response", "receiving", "processing", "retrying", "completed":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid active request status"})
		return query, false
	}
	return query, true
}
