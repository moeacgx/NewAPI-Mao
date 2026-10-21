package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetPromptTemplates(c *gin.Context) {
	templates, err := model.ListPromptTemplates(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, templates)
}

func GetAdminPromptTemplates(c *gin.Context) {
	templates, err := model.ListPromptTemplates(false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, templates)
}

func SavePromptTemplate(c *gin.Context) {
	var input model.PromptTemplate
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024), &input); err != nil {
		common.ApiErrorMsg(c, "提示词请求格式错误或超过大小限制")
		return
	}
	input.ID = 0
	if c.Request.Method == http.MethodPut {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "提示词 ID 无效")
			return
		}
		input.ID = id
	}
	saved, err := model.SavePromptTemplate(input)
	if errors.Is(err, model.ErrPromptVersionConflict) {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, saved)
}

func GetTokenPrompts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "令牌 ID 无效")
		return
	}
	installed, err := model.GetTokenPromptBindings(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, installed)
}

func UpdateTokenPrompts(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "令牌 ID 无效")
		return
	}
	var input struct {
		Bindings *[]model.TokenPromptBindingInput `json:"bindings"`
	}
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024), &input); err != nil || input.Bindings == nil {
		common.ApiErrorMsg(c, "请提交完整的提示词安装列表")
		return
	}
	token, err := model.GetTokenForPromptInstallation(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userGroup, err := getTokenRequestUserGroup(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	groups := []string{userGroup}
	if token.Group == "auto" || token.GroupMode == model.TokenGroupModeAuto {
		groups = service.GetUserAutoGroup(userGroup)
		if token.AutoGroups != "" {
			autoGroups, parseErr := token.GetAutoGroups()
			if parseErr != nil {
				common.ApiError(c, parseErr)
				return
			}
			groups = service.FilterUserTokenAutoGroups(userGroup, autoGroups)
		}
	} else if len(token.GroupDetails) > 0 {
		groups = make([]string, 0, len(token.GroupDetails))
		for _, group := range token.GroupDetails {
			groups = append(groups, group.Code)
		}
	} else if token.Group != "" {
		groups, err = service.ParseTokenGroupList(token.Group)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	allowedIDs := make([]int, 0, len(groups))
	for _, code := range groups {
		group, lookupErr := model.GetGroupByCodeOrAlias(code)
		if lookupErr != nil || group.Status != model.GroupStatusActive || !service.IsUserSelectableGroup(userGroup, group.Code) {
			continue
		}
		allowedIDs = append(allowedIDs, group.Id)
	}
	if err := model.ReplaceTokenPromptBindings(c.GetInt("id"), id, *input.Bindings, allowedIDs); err != nil {
		common.ApiError(c, err)
		return
	}
	installed, err := model.GetTokenPromptBindings(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, installed)
}
