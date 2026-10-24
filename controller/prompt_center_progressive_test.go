package controller

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressivePromptAPIKeepsAdminFilesAndPublicMetadataSeparate(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}, &model.PromptSkillRecord{}, &model.PromptSkillFile{}, &model.PromptSkillChunk{}))
	group := model.Group{Code: "internal-skill-group", Name: "技能专属模型组", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	body := "private-main-body-" + strings.Repeat("中文技能说明。", 20000)
	input := model.PromptTemplate{Title: "渐进式技能", Description: "适用于代码审查", Content: body, LoadMode: model.PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []model.PromptSkill{{Path: "skills/review/SKILL.md", Resources: []model.PromptSkillResource{{Path: "references/private.txt", Content: "private-resource-body"}}}}}
	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/prompt-center/admin/", input, 52)
	SavePromptTemplate(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success, recorder.Body.String()[:min(512, recorder.Body.Len())])
	var saved struct {
		Data model.PromptTemplate `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &saved))
	require.Len(t, saved.Data.Skills, 1)
	assert.Equal(t, body, saved.Data.Content)
	assert.Equal(t, "private-resource-body", saved.Data.Skills[0].Resources[0].Content)
	assert.NotEmpty(t, saved.Data.Skills[0].ID)
	token := seedToken(t, db, 52, "progressive-token", "private-progressive-key")
	require.NoError(t, model.ReplaceTokenPromptBindings(52, token.Id, []model.TokenPromptBindingInput{{TemplateID: saved.Data.ID, Enabled: true}}, []int{group.Id}))
	for _, installed := range []bool{false, true} {
		ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/", nil, 52)
		if installed {
			ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(token.Id)}}
			GetTokenPrompts(ctx)
		} else {
			GetPromptTemplates(ctx)
		}
		require.True(t, decodeAPIResponse(t, recorder).Success)
		assert.Contains(t, recorder.Body.String(), `"load_mode":"progressive"`)
		assert.Contains(t, recorder.Body.String(), saved.Data.Skills[0].ID)
		assert.Contains(t, recorder.Body.String(), "适用于代码审查")
		assert.Contains(t, recorder.Body.String(), "技能专属模型组")
		assert.NotContains(t, recorder.Body.String(), `"content"`)
		assert.NotContains(t, recorder.Body.String(), `"resources"`)
		assert.NotContains(t, recorder.Body.String(), `"path"`)
		assert.NotContains(t, recorder.Body.String(), "private-")
	}
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/prompt-center/admin/", nil, 52)
	GetAdminPromptTemplates(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	assert.Contains(t, recorder.Body.String(), "private-main-body-")
	assert.Contains(t, recorder.Body.String(), "private-resource-body")
	saved.Data.Content = "更新主正文"
	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/prompt-center/admin/1", saved.Data, 52)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(saved.Data.ID)}}
	SavePromptTemplate(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	var updated struct {
		Data model.PromptTemplate `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &updated))
	assert.Equal(t, saved.Data.Skills[0].ID, updated.Data.Skills[0].ID)
	assert.Equal(t, saved.Data.Skills[0].Resources, updated.Data.Skills[0].Resources)
	assert.Equal(t, "更新主正文", updated.Data.Skills[0].Content)
	ctx, recorder = newAuthenticatedContext(t, http.MethodPut, "/api/prompt-center/admin/1", saved.Data, 52)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(saved.Data.ID)}}
	SavePromptTemplate(ctx)
	assert.Equal(t, http.StatusConflict, recorder.Code)
}
