package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func RevokeAccessToken(c *gin.Context) {
	ref, err := model.RevokeUserAccessToken(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if ref != "" {
		recordUserSecurityAudit(c, c.GetInt("id"), "access_token.revoke", map[string]interface{}{"token_ref": ref})
	}
	common.ApiSuccess(c, nil)
}
