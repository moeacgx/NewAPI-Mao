package controller

import (
	"net/http"
	"strconv"
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
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Group{}, &model.GroupAlias{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
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
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
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
