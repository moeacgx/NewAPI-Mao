package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func activeSkillTemplateIDs(t *testing.T, tokenID int, groupCode string) []int {
	t.Helper()
	catalog, err := GetActiveTokenSkills(tokenID, groupCode)
	require.NoError(t, err)
	ids := make([]int, 0, len(catalog))
	for _, skill := range catalog {
		ids = append(ids, skill.TemplateID)
	}
	return ids
}

func TestPromptTemplateAllGroupsAuthorizesAnyActiveGroup(t *testing.T) {
	first, second, token := setupProgressiveSkillTest(t)
	paused := &Group{Code: "paused-code", Name: "暂停的分组", Ratio: 1}
	require.NoError(t, DB.Create(paused).Error)
	// Status 带 default:1，零值创建会被写成启用，需显式更新为停用。
	require.NoError(t, DB.Model(paused).Update("status", GroupStatusDisabled).Error)

	_, err := SavePromptTemplate(PromptTemplate{Title: "缺少白名单", Description: "用途", Content: "正文", LoadMode: PromptLoadModeProgressive, Enabled: true})
	require.Error(t, err, "未选择不限分组时上架仍需要白名单")

	open, err := SavePromptTemplate(PromptTemplate{Title: "通用技能", Description: "任意分组可用", Content: "通用正文", LoadMode: PromptLoadModeProgressive, Enabled: true, AllGroups: true, GroupIDs: []int{first.Id}})
	require.NoError(t, err)
	assert.True(t, open.AllGroups)
	assert.Empty(t, open.GroupIDs, "不限分组时清空白名单，避免两种授权语义并存")
	var bindingCount int64
	require.NoError(t, DB.Model(&PromptTemplateGroup{}).Where("template_id = ?", open.ID).Count(&bindingCount).Error)
	assert.Zero(t, bindingCount)
	restricted, err := SavePromptTemplate(PromptTemplate{Title: "默认组专用", Description: "仅默认组", Content: "专用正文", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{first.Id}})
	require.NoError(t, err)

	require.Error(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: open.ID, Enabled: true}}, nil), "令牌没有任何可用分组时不能安装")
	require.Error(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: open.ID, Enabled: true}, {TemplateID: restricted.ID, Enabled: true}}, []int{second.Id}), "白名单模板仍要求分组交集")
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: open.ID, Enabled: true}, {TemplateID: restricted.ID, Enabled: true, SortOrder: 1}}, []int{first.Id}))

	assert.Equal(t, []int{open.ID, restricted.ID}, activeSkillTemplateIDs(t, token.Id, first.Code))
	assert.Equal(t, []int{open.ID}, activeSkillTemplateIDs(t, token.Id, second.Code), "白名单外的分组只装载不限分组的技能")
	assert.Empty(t, activeSkillTemplateIDs(t, token.Id, paused.Code), "停用分组不因不限分组而生效")

	page, err := ReadActiveTokenSkillFile(token.Id, second.Code, open.ID, open.Skills[0].ID, open.Version, "", 0, 1024)
	require.NoError(t, err)
	assert.Equal(t, "通用正文", page.Content)
	_, err = ReadActiveTokenSkillFile(token.Id, paused.Code, open.ID, open.Skills[0].ID, open.Version, "", 0, 1024)
	require.Error(t, err)
	_, err = ReadActiveTokenSkillFile(token.Id, second.Code, restricted.ID, restricted.Skills[0].ID, restricted.Version, "", 0, 1024)
	require.Error(t, err, "白名单模板不能在其他分组读取正文")

	eager, err := SavePromptTemplate(PromptTemplate{Title: "通用全文", Content: "全文正文", Enabled: true, AllGroups: true})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: open.ID, Enabled: true}, {TemplateID: eager.ID, Enabled: true, SortOrder: 1}}, []int{second.Id}))
	loaded, err := GetActiveTokenPrompts(token.Id, second.Code)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, eager.ID, loaded[0].ID)
	loaded, err = GetActiveTokenPrompts(token.Id, paused.Code)
	require.NoError(t, err)
	assert.Empty(t, loaded)

	// 取消不限分组并改回白名单后，已安装令牌在白名单外分组立即失去该技能。
	open.AllGroups, open.GroupIDs = false, []int{first.Id}
	open, err = SavePromptTemplate(*open)
	require.NoError(t, err)
	assert.False(t, open.AllGroups)
	assert.Equal(t, []int{first.Id}, open.GroupIDs)
	assert.Empty(t, activeSkillTemplateIDs(t, token.Id, second.Code))
	assert.Equal(t, []int{open.ID}, activeSkillTemplateIDs(t, token.Id, first.Code))
}

func TestPromptTemplateLegacyNullAllGroupsKeepsWhitelist(t *testing.T) {
	first, second, token := setupProgressiveSkillTest(t)
	template, err := SavePromptTemplate(PromptTemplate{Title: "存量技能", Description: "旧数据", Content: "正文", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{first.Id}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: template.ID, Enabled: true}}, []int{first.Id}))
	// 迁移新增的列对存量行为 NULL，必须按白名单处理，不能被当作不限分组。
	require.NoError(t, DB.Model(&PromptTemplate{}).Where("id = ?", template.ID).Update("all_groups", gorm.Expr("NULL")).Error)
	items, err := ListPromptTemplates(false)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.False(t, items[0].AllGroups)
	assert.Equal(t, []int{template.ID}, activeSkillTemplateIDs(t, token.Id, first.Code))
	assert.Empty(t, activeSkillTemplateIDs(t, token.Id, second.Code))
}
