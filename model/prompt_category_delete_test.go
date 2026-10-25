package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletePromptCategoryMovesTemplatesToUncategorized(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	develop, err := SavePromptCategory(PromptCategory{Name: "开发"})
	require.NoError(t, err)
	writing, err := SavePromptCategory(PromptCategory{Name: "写作"})
	require.NoError(t, err)
	moved, err := SavePromptTemplate(PromptTemplate{Title: "代码审查", Description: "审查", Content: "审查正文", LoadMode: PromptLoadModeProgressive, CategoryID: develop.ID, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	kept, err := SavePromptTemplate(PromptTemplate{Title: "文章润色", Description: "润色", Content: "润色正文", LoadMode: PromptLoadModeProgressive, CategoryID: writing.ID, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: moved.ID, Enabled: true}}, []int{group.Id}))

	_, err = DeletePromptCategory(develop.ID, develop.Version+1)
	require.ErrorIs(t, err, ErrPromptVersionConflict)
	count, err := DeletePromptCategory(develop.ID, develop.Version)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	_, err = DeletePromptCategory(develop.ID, develop.Version)
	require.ErrorIs(t, err, ErrPromptCategoryNotFound)

	categories, err := ListPromptCategories()
	require.NoError(t, err)
	require.Len(t, categories, 1)
	assert.Equal(t, writing.ID, categories[0].ID)
	items, err := ListPromptTemplates(false)
	require.NoError(t, err)
	byID := make(map[int]PromptTemplate, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	assert.Zero(t, byID[moved.ID].CategoryID)
	assert.Empty(t, byID[moved.ID].CategoryName)
	assert.Equal(t, moved.Version, byID[moved.ID].Version, "分类只做目录组织，删除分类不使已安装技能的版本失效")
	assert.Equal(t, writing.ID, byID[kept.ID].CategoryID)
	page, err := ReadActiveTokenSkillFile(token.Id, group.Code, moved.ID, moved.Skills[0].ID, moved.Version, "", 0, 1024)
	require.NoError(t, err)
	assert.Equal(t, "审查正文", page.Content)

	_, err = SavePromptTemplate(*moved)
	require.ErrorContains(t, err, "分类")
}
