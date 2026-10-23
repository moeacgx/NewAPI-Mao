package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptContentValidationExplainsFailure(t *testing.T) {
	for _, tc := range []struct {
		name, content, want string
	}{
		{"空正文", " \n\t", "提示词正文不能为空"},
		{"无效编码", string([]byte{0xff}), "提示词正文必须为有效 UTF-8 文本"},
		{"中文按字节超限", strings.Repeat("字", 10923), "提示词正文为 32769 字节，超过 32768 字节上限"},
		{"恰好到上限", strings.Repeat("字", 10922) + "ab", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := NormalizePromptTemplateContent(&PromptTemplate{Title: "示例", Content: tc.content})
			if tc.want == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.want)
		})
	}
	for _, tc := range []struct {
		name, content, want string
	}{
		{"空子技能", " ", "合集内技能「审查」正文不能为空"},
		{"无效子技能编码", string([]byte{0xff}), "合集内技能「审查」正文必须为有效 UTF-8 文本"},
		{"子技能超限", strings.Repeat("x", MaxPromptContentBytes+1), "合集内技能「审查」正文为 32769 字节，超过 32768 字节上限"},
		{"拼接总量超限", strings.Repeat("x", MaxPromptContentBytes), fmt.Sprintf("技能合集拼接正文为 %d 字节，超过 32768 字节上限（包含技能标题和分隔符），请减少技能或拆分合集", len("## 技能：审查\n")+MaxPromptContentBytes+len("\n\n## 技能：测试\n正文"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := NormalizePromptTemplateContent(&PromptTemplate{Title: "示例", Kind: "bundle", Skills: []PromptSkill{{Title: "审查", Content: tc.content}, {Title: "测试", Content: "正文"}}})
			require.EqualError(t, err, tc.want)
		})
	}
}

func TestPromptBundleCanonicalContentAndCategoryRename(t *testing.T) {
	group, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	category, err := SavePromptCategory(PromptCategory{Name: "开发"})
	require.NoError(t, err)
	bundle, err := SavePromptTemplate(PromptTemplate{Title: "开发合集", Kind: "bundle", CategoryID: category.ID, Content: "不得使用的客户端正文", Skills: []PromptSkill{{Title: "审查", Content: "查找缺陷"}, {Title: "测试", Content: "补充边界"}}, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	assert.Equal(t, "## 技能：审查\n查找缺陷\n\n## 技能：测试\n补充边界", bundle.Content)
	assert.Equal(t, 2, bundle.SkillCount)
	assert.Equal(t, "开发", bundle.CategoryName)
	expectedContent := bundle.Content
	token := Token{UserId: 91, Key: "bundle-key"}
	require.NoError(t, DB.Create(&token).Error)
	require.NoError(t, ReplaceTokenPromptBindings(91, token.Id, []TokenPromptBindingInput{{TemplateID: bundle.ID, Enabled: true}}, []int{group.Id}))
	// 即使派生字段被直接修改，装载也必须由内部技能重新生成正文。
	require.NoError(t, DB.Model(bundle).Update("content", "篡改的派生字段").Error)
	loaded, err := GetActiveTokenPrompts(token.Id, group.Code)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, expectedContent, loaded[0].Content)
	old := *category
	category.Name = "工程"
	_, err = SavePromptCategory(*category)
	require.NoError(t, err)
	_, err = SavePromptCategory(old)
	assert.ErrorIs(t, err, ErrPromptVersionConflict)
	installed, err := GetTokenPromptBindings(91, token.Id)
	require.NoError(t, err)
	assert.Equal(t, "工程", installed[0].Template.CategoryName)
	require.NoError(t, HydrateTokenPromptCounts([]*Token{&token}))
	assert.Equal(t, 1, token.PromptCount)
	assert.Equal(t, 1, token.EnabledPromptCount)
	require.NoError(t, DB.Model(bundle).Update("kind", "unknown").Error)
	_, err = GetActiveTokenPrompts(token.Id, group.Code)
	require.Error(t, err)
}

func TestPromptBundleValidationAndLegacyCompatibility(t *testing.T) {
	group, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	for _, input := range []PromptTemplate{
		{Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: "正文"}}},
		{Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: "正文"}, {Title: " 一 ", Content: "正文"}}},
		{Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: " "}, {Title: "二", Content: "正文"}}},
		{Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: strings.Repeat("x", MaxPromptContentBytes)}, {Title: "二", Content: "正文"}}},
		{Kind: "bundle", Skills: make([]PromptSkill, 17)},
		{Kind: "unknown", Content: "正文"},
		{Content: "正文", CategoryID: 99999},
		{Content: "正文", CategoryID: -1},
	} {
		input.Title = "校验"
		_, err := SavePromptTemplate(input)
		require.Error(t, err)
	}
	legacy := PromptTemplate{Title: "历史技能", Content: "保留原始正文", Enabled: true}
	require.NoError(t, DB.Create(&legacy).Error)
	require.NoError(t, DB.Create(&PromptTemplateGroup{TemplateID: legacy.ID, GroupID: group.Id}).Error)
	items, err := ListPromptTemplates(true)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "prompt", items[0].Kind)
	assert.Equal(t, 1, items[0].SkillCount)
	assert.NotNil(t, items[0].Skills)
	assert.Empty(t, items[0].Skills)
	assert.Equal(t, legacy.Content, items[0].Content)
	for _, name := range []string{"", " ", strings.Repeat("字", 41)} {
		_, err = SavePromptCategory(PromptCategory{Name: name})
		require.Error(t, err)
	}
	_, err = SavePromptCategory(PromptCategory{Name: "开发"})
	require.NoError(t, err)
	_, err = SavePromptCategory(PromptCategory{Name: " 开发 "})
	require.Error(t, err)
}

func TestPromptBundleBudgetUsesCanonicalContentOnInstallAndUpdate(t *testing.T) {
	group, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	token := Token{UserId: 92, Key: "bundle-budget-key"}
	require.NoError(t, DB.Create(&token).Error)
	bindings := make([]TokenPromptBindingInput, 0, 3)
	var bundle *PromptTemplate
	for i := 0; i < 3; i++ {
		var err error
		bundle, err = SavePromptTemplate(PromptTemplate{Title: "有大小限制的合集", Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: strings.Repeat("a", 10000)}, {Title: "二", Content: strings.Repeat("b", 10000)}}, Enabled: true, GroupIDs: []int{group.Id}})
		require.NoError(t, err)
		bindings = append(bindings, TokenPromptBindingInput{TemplateID: bundle.ID, Enabled: true})
	}
	require.NoError(t, ReplaceTokenPromptBindings(92, token.Id, bindings, []int{group.Id}))
	bundle.Skills[0].Content = strings.Repeat("c", 20000)
	_, err := SavePromptTemplate(*bundle)
	require.Error(t, err)
	// 不能通过缩短派生正文绕过安装与运行时的合集明细大小约束。
	bundle.Skills = []PromptSkill{{Title: "一", Content: strings.Repeat("x", 20000)}, {Title: "二", Content: strings.Repeat("y", 10000)}}
	require.NoError(t, NormalizePromptTemplateContent(bundle))
	require.NoError(t, DB.Model(&PromptTemplate{}).Where("id = ?", bundle.ID).Updates(map[string]interface{}{"skills_json": bundle.SkillsJSON, "content": "短正文"}).Error)
	require.Error(t, ReplaceTokenPromptBindings(92, token.Id, bindings, []int{group.Id}))
	_, err = GetActiveTokenPrompts(token.Id, group.Code)
	require.Error(t, err)
}
