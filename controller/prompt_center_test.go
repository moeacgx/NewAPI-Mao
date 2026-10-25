package controller

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenPromptAPIEnforcesOwnershipAndReturnsBadgeCounts(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Group{}, &model.GroupAlias{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
	group := model.Group{Code: "prompt-code", Name: "提示词专用分组", Ratio: 1, Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	user := model.User{Username: "prompt-owner", Group: group.Code}
	require.NoError(t, db.Create(&user).Error)
	oldRatios := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios)) })
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"prompt-code":1}`))
	token := seedToken(t, db, user.Id, "prompt-token", "private-prompt-token-key")
	require.NoError(t, db.Model(token).Update("group", group.Code).Error)
	prompt, err := model.SavePromptTemplate(model.PromptTemplate{Title: "编码", Content: "审查代码", Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/", nil, user.Id)
	GetPromptTemplates(ctx)
	assert.True(t, decodeAPIResponse(t, recorder).Success)
	assert.NotContains(t, recorder.Body.String(), prompt.Content)
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/admin/", nil, user.Id)
	GetAdminPromptTemplates(ctx)
	assert.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), prompt.Content)
	request := map[string]interface{}{"bindings": []model.TokenPromptBindingInput{{TemplateID: prompt.ID, Enabled: true}}}
	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/token/1/prompts", request, user.Id+1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	UpdateTokenPrompts(ctx)
	assert.False(t, decodeAPIResponse(t, recorder).Success)
	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/token/1/prompts", request, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	ctx.Set("group", group.Code)
	UpdateTokenPrompts(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success, recorder.Body.String())
	assert.NotContains(t, recorder.Body.String(), prompt.Content)
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/token/1/prompts", nil, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	GetTokenPrompts(ctx)
	assert.True(t, decodeAPIResponse(t, recorder).Success)
	assert.NotContains(t, recorder.Body.String(), prompt.Content)
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/token/?size=10", nil, user.Id)
	GetAllTokens(ctx)
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Items []model.Token `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data.Items, 1)
	assert.Equal(t, 1, response.Data.Items[0].PromptCount)
	assert.Equal(t, 1, response.Data.Items[0].EnabledPromptCount)
	assert.NotContains(t, recorder.Body.String(), token.Key)
}

func TestPromptTemplateAPIUsesURLIDAndReportsVersionConflict(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
	group := model.Group{Code: "prompt-admin", Name: "管理员展示名称"}
	require.NoError(t, db.Create(&group).Error)
	prompt, err := model.SavePromptTemplate(model.PromptTemplate{Title: "旧标题", Content: "正文", Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	request := map[string]interface{}{"id": 999, "title": "新标题", "content": "新正文", "enabled": false, "version": prompt.Version, "group_ids": []int{group.Id}}
	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/prompt-center/admin/1", request, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(prompt.ID)}}
	SavePromptTemplate(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/prompt-center/admin/1", request, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(prompt.ID)}}
	SavePromptTemplate(ctx)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.False(t, decodeAPIResponse(t, recorder).Success)
}

func TestPromptBundlePublicAPINeverExposesInternalContent(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PromptCategory{}, &model.Group{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
	group := model.Group{Code: "bundle-group", Name: "技能可用模型组"}
	require.NoError(t, db.Create(&group).Error)
	category, err := model.SavePromptCategory(model.PromptCategory{Name: "编程开发"})
	require.NoError(t, err)
	bundle, err := model.SavePromptTemplate(model.PromptTemplate{Title: "研发合集", Kind: "bundle", CategoryID: category.ID, Skills: []model.PromptSkill{{Title: "审查", Content: "private-review-body"}, {Title: "测试", Content: "private-test-body"}}, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	token := seedToken(t, db, 52, "bundle-token", "private-bundle-key")
	require.NoError(t, model.ReplaceTokenPromptBindings(52, token.Id, []model.TokenPromptBindingInput{{TemplateID: bundle.ID, Enabled: true}}, []int{group.Id}))
	for _, installed := range []bool{false, true} {
		ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/", nil, 52)
		if installed {
			ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
			GetTokenPrompts(ctx)
		} else {
			GetPromptTemplates(ctx)
		}
		require.True(t, decodeAPIResponse(t, recorder).Success)
		assert.Contains(t, recorder.Body.String(), `"kind":"bundle"`)
		assert.Contains(t, recorder.Body.String(), `"skill_count":2`)
		assert.Contains(t, recorder.Body.String(), "编程开发")
		assert.Contains(t, recorder.Body.String(), "审查")
		assert.NotContains(t, recorder.Body.String(), `"content"`)
		assert.NotContains(t, recorder.Body.String(), "private-")
		assert.NotContains(t, recorder.Body.String(), "skills_json")
	}
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/admin/", nil, 52)
	GetAdminPromptTemplates(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), "private-review-body")
	assert.NotContains(t, recorder.Body.String(), "skills_json")
}

func TestPromptCategoryAPIUsesVersionAndURLIdentity(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.PromptCategory{}))
	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/prompt-center/admin/categories/", map[string]interface{}{"name": "写作", "id": 999}, 1)
	SavePromptCategory(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	var response struct {
		Data model.PromptCategory `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.NotEqual(t, 999, response.Data.ID)
	input := map[string]interface{}{"name": "专业写作", "version": response.Data.Version, "id": 999}
	for _, conflict := range []bool{false, true} {
		ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/prompt-center/admin/categories/1", input, 1)
		ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(response.Data.ID)}}
		SavePromptCategory(ctx)
		if conflict {
			assert.Equal(t, http.StatusConflict, recorder.Code)
		} else {
			require.True(t, decodeAPIResponse(t, recorder).Success)
		}
	}
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/categories/", nil, 1)
	GetPromptCategories(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), "专业写作")
	assert.NotContains(t, recorder.Body.String(), "name_key")
}

func TestPromptTemplateDeleteAndRepositoryUpdateAPIContracts(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}, &model.PromptSkillRecord{}, &model.PromptSkillFile{}, &model.PromptSkillChunk{}))
	group := model.Group{Code: "repo-api-group", Name: "仓库技能展示名", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	source := &model.PromptRepositorySource{URL: "https://github.com/acme/skills", Ref: "main", Commit: strings.Repeat("a", 40), ImportedAt: 1792843200, ContentHash: strings.Repeat("b", 64)}
	skill, err := model.SavePromptTemplate(model.PromptTemplate{Title: "仓库技能", Description: "用途", Content: "正文", LoadMode: model.PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, RepositorySource: source})
	require.NoError(t, err)
	require.True(t, skill.RepositorySource.LocallyModified, "指纹与正文不一致时视为站内已修改")
	token := seedToken(t, db, 61, "repo-api-token", "private-repo-api-key")
	require.NoError(t, model.ReplaceTokenPromptBindings(61, token.Id, []model.TokenPromptBindingInput{{TemplateID: skill.ID, Enabled: true}}, []int{group.Id}))

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/", nil, 61)
	GetPromptTemplates(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.NotContains(t, recorder.Body.String(), "repository_source")
	assert.NotContains(t, recorder.Body.String(), "github.com")
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/admin/", nil, 1)
	GetAdminPromptTemplates(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), `"repository_source":{"url":"https://github.com/acme/skills"`)

	// 站内修改未确认时在访问 GitHub 前返回专用冲突码；版本不一致返回普通冲突。
	ctx, recorder = newAuthenticatedContext(t, http.MethodPost, "/api/prompt-center/admin/1/update-repository", map[string]any{"version": skill.Version}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(skill.ID)}}
	UpdatePromptTemplateRepository(ctx)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"code":"repository_local_changes"`)
	ctx, recorder = newAuthenticatedContext(t, http.MethodPost, "/api/prompt-center/admin/1/update-repository", map[string]any{"version": skill.Version + 1, "overwrite_local": true}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(skill.ID)}}
	UpdatePromptTemplateRepository(ctx)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "repository_local_changes")

	for _, query := range []string{"", "?version=0", "?version=abc"} {
		ctx, recorder = newAuthenticatedContext(t, http.MethodDelete, "/api/prompt-center/admin/1"+query, nil, 1)
		ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(skill.ID)}}
		DeletePromptTemplate(ctx)
		assert.False(t, decodeAPIResponse(t, recorder).Success, query)
	}
	ctx, recorder = newAuthenticatedContext(t, http.MethodDelete, "/api/prompt-center/admin/1?version="+strconv.Itoa(skill.Version+1), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(skill.ID)}}
	DeletePromptTemplate(ctx)
	assert.Equal(t, http.StatusConflict, recorder.Code)
	ctx, recorder = newAuthenticatedContext(t, http.MethodDelete, "/api/prompt-center/admin/1?version="+strconv.Itoa(skill.Version), nil, 1)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(skill.ID)}}
	DeletePromptTemplate(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success, recorder.Body.String())
	assert.Contains(t, recorder.Body.String(), `"uninstalled_tokens":1`)

	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/token/1/prompts", nil, 61)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
	GetTokenPrompts(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.NotContains(t, recorder.Body.String(), "仓库技能")
}

func TestPromptTemplatesAPIExposeAllGroups(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
	_, err := model.SavePromptTemplate(model.PromptTemplate{Title: "通用提示词", Content: "正文", Enabled: true, AllGroups: true})
	require.NoError(t, err)

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/", nil, 1)
	GetPromptTemplates(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), `"all_groups":true`)
	assert.Contains(t, recorder.Body.String(), `"group_ids":[]`)
}
