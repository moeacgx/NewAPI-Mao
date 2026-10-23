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

type promptTemplatePublic struct {
	ID           int                    `json:"id"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	Kind         string                 `json:"kind"`
	LoadMode     string                 `json:"load_mode"`
	CategoryID   int                    `json:"category_id"`
	CategoryName string                 `json:"category_name"`
	Skills       []promptSkillPublic    `json:"skills"`
	SkillCount   int                    `json:"skill_count"`
	Enabled      bool                   `json:"enabled"`
	SortOrder    int                    `json:"sort_order"`
	Version      int                    `json:"version"`
	GroupIDs     []int                  `json:"group_ids"`
	GroupDetails []model.GroupReference `json:"group_details"`
}

type promptSkillPublic struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type tokenPromptInstallationPublic struct {
	Template  promptTemplatePublic `json:"template"`
	Enabled   bool                 `json:"enabled"`
	SortOrder int                  `json:"sort_order"`
}

func publicPromptTemplate(template model.PromptTemplate) promptTemplatePublic {
	skills := make([]promptSkillPublic, 0, len(template.Skills))
	for _, skill := range template.Skills {
		skills = append(skills, promptSkillPublic{ID: skill.ID, Title: skill.Title, Description: skill.Description})
	}
	return promptTemplatePublic{
		ID: template.ID, Title: template.Title, Description: template.Description,
		Enabled: template.Enabled, SortOrder: template.SortOrder, Version: template.Version,
		GroupIDs: template.GroupIDs, GroupDetails: template.GroupDetails,
		Kind: template.Kind, LoadMode: template.LoadMode, CategoryID: template.CategoryID, CategoryName: template.CategoryName,
		Skills: skills, SkillCount: template.SkillCount,
	}
}

func GetPromptCategories(c *gin.Context) {
	categories, err := model.ListPromptCategories()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, categories)
}

func SavePromptCategory(c *gin.Context) {
	var input model.PromptCategory
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, 4096), &input); err != nil {
		common.ApiErrorMsg(c, "技能分类请求格式错误或超过大小限制")
		return
	}
	input.ID = 0
	if c.Request.Method == http.MethodPut {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			common.ApiErrorMsg(c, "技能分类 ID 无效")
			return
		}
		input.ID = id
	}
	saved, err := model.SavePromptCategory(input)
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

func GetPromptTemplates(c *gin.Context) {
	templates, err := model.ListPromptTemplateSummaries(true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	public := make([]promptTemplatePublic, 0, len(templates))
	for _, template := range templates {
		public = append(public, publicPromptTemplate(template))
	}
	common.ApiSuccess(c, public)
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
	// JSON 的 Unicode 转义与单技能镜像字段可放大正文，内容总量在模型层独立校验。
	if err := common.DecodeJson(http.MaxBytesReader(c.Writer, c.Request.Body, (model.MaxSkillPackageBytes+model.MaxSkillFileBytes)*6+16*1024*1024), &input); err != nil {
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
	installed, err := model.GetTokenPromptBindingSummaries(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	public := make([]tokenPromptInstallationPublic, 0, len(installed))
	for _, item := range installed {
		public = append(public, tokenPromptInstallationPublic{
			Template: publicPromptTemplate(item.Template), Enabled: item.Enabled, SortOrder: item.SortOrder,
		})
	}
	common.ApiSuccess(c, public)
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
	GetTokenPrompts(c)
}
