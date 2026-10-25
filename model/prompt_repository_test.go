package model

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func repositorySourceFixture(t *testing.T, template PromptTemplate) *PromptRepositorySource {
	t.Helper()
	require.NoError(t, NormalizePromptTemplateContent(&template))
	hash, err := PromptTemplateContentHash(&template)
	require.NoError(t, err)
	return &PromptRepositorySource{URL: "https://github.com/acme/skills/tree/main/review", Ref: "main", Path: "review",
		Commit: strings.Repeat("a", 40), ImportedAt: 1792843200, ContentHash: hash}
}

func TestPromptRepositorySourceSurvivesSaveAndLegacyEdits(t *testing.T) {
	group, _, _ := setupProgressiveSkillTest(t)
	draft := PromptTemplate{Title: "审查", Description: "检查代码", Content: "原始正文", LoadMode: PromptLoadModeProgressive,
		Skills: []PromptSkill{{Path: "review/SKILL.md", Resources: []PromptSkillResource{{Path: "references/a.md", Content: "引用"}}}}}
	draft.RepositorySource = repositorySourceFixture(t, draft)
	draft.Enabled, draft.GroupIDs = true, []int{group.Id}
	saved, err := SavePromptTemplate(draft)
	require.NoError(t, err)
	require.NotNil(t, saved.RepositorySource)
	assert.Equal(t, "https://github.com/acme/skills/tree/main/review", saved.RepositorySource.URL)
	assert.False(t, saved.RepositorySource.LocallyModified, "刚导入的副本与来源一致")

	// 单技能的标题和说明属于站点设置，更新时会保留，修改它们不算来源内容被改动。
	saved.Title, saved.Description = "代码审查", "站内改写的说明"
	saved, err = SavePromptTemplate(*saved)
	require.NoError(t, err)
	require.NotNil(t, saved.RepositorySource)
	assert.False(t, saved.RepositorySource.LocallyModified)

	// 旧客户端不认识来源字段；编辑正文后仍保留来源，并标记为站内修改。
	var legacy PromptTemplate
	require.NoError(t, common.UnmarshalJsonStr(`{"title":"代码审查","description":"检查代码","content":"管理员修订正文","load_mode":"progressive"}`, &legacy))
	legacy.ID, legacy.Version, legacy.GroupIDs, legacy.Enabled, legacy.Skills = saved.ID, saved.Version, saved.GroupIDs, saved.Enabled, saved.Skills
	_, err = SavePromptTemplate(legacy)
	require.NoError(t, err)
	templates, err := ListPromptTemplates(false)
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.NotNil(t, templates[0].RepositorySource)
	assert.Equal(t, strings.Repeat("a", 40), templates[0].RepositorySource.Commit)
	assert.True(t, templates[0].RepositorySource.LocallyModified)
	encoded, err := common.Marshal(templates[0])
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"locally_modified":true`)
	assert.NotContains(t, string(encoded), "repository_source_json")

	// 用户目录摘要不读取来源，避免把管理员信息带到公共接口。
	summaries, err := ListPromptTemplateSummaries(true)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Nil(t, summaries[0].RepositorySource)
}

func TestPromptRepositorySourceRejectsUnsafeRecords(t *testing.T) {
	setupProgressiveSkillTest(t)
	base := PromptTemplate{Title: "审查", Content: "正文", LoadMode: PromptLoadModeProgressive}
	valid := repositorySourceFixture(t, base)
	for name, mutate := range map[string]func(*PromptRepositorySource){
		"非 HTTPS": func(source *PromptRepositorySource) { source.URL = "http://github.com/acme/skills" },
		"其他主机":    func(source *PromptRepositorySource) { source.URL = "https://example.com/acme/skills" },
		"认证信息":    func(source *PromptRepositorySource) { source.URL = "https://user:secret@github.com/acme/skills" },
		"查询参数":    func(source *PromptRepositorySource) { source.URL = "https://github.com/acme/skills?token=secret" },
		"端口":      func(source *PromptRepositorySource) { source.URL = "https://github.com:443/acme/skills" },
		"引用越界":    func(source *PromptRepositorySource) { source.Ref = "../main" },
		"目录越界":    func(source *PromptRepositorySource) { source.Path = "../secret" },
		"提交不完整":   func(source *PromptRepositorySource) { source.Commit = "abc" },
		"指纹无效":    func(source *PromptRepositorySource) { source.ContentHash = "zz" },
		"缺少导入时间":  func(source *PromptRepositorySource) { source.ImportedAt = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			source := *valid
			mutate(&source)
			input := base
			input.RepositorySource = &source
			_, err := SavePromptTemplate(input)
			require.Error(t, err)
		})
	}
	var count int64
	require.NoError(t, DB.Model(&PromptTemplate{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestPromptDescriptionsAcceptLongTextUpToStorageBoundary(t *testing.T) {
	group, _, _ := setupProgressiveSkillTest(t)
	long := strings.Repeat("长用途说明，", 600)
	require.Greater(t, utf8.RuneCountInString(long), 512)
	tooLong := strings.Repeat("a", MaxPromptDescriptionBytes+1)
	for _, test := range []struct {
		name  string
		build func(description string) PromptTemplate
		read  func(*PromptTemplate) string
	}{
		{"全文单技能", func(description string) PromptTemplate {
			return PromptTemplate{Title: "全文", Description: description, Content: "正文", Enabled: true, GroupIDs: []int{group.Id}}
		}, func(template *PromptTemplate) string { return template.Description }},
		{"全文合集", func(description string) PromptTemplate {
			return PromptTemplate{Title: "合集", Kind: "bundle", Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{
				{Title: "一", Description: description, Content: "正文一"}, {Title: "二", Content: "正文二"}}}
		}, func(template *PromptTemplate) string { return template.Skills[0].Description }},
		{"渐进式合集", func(description string) PromptTemplate {
			return PromptTemplate{Title: "渐进", Kind: "bundle", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{
				{Title: "一", Description: description, Path: "one/SKILL.md", Content: "正文一"}, {Title: "二", Description: "短说明", Path: "two/SKILL.md", Content: "正文二"}}}
		}, func(template *PromptTemplate) string { return template.Skills[0].Description }},
	} {
		t.Run(test.name, func(t *testing.T) {
			saved, err := SavePromptTemplate(test.build(long))
			require.NoError(t, err)
			assert.Equal(t, long, test.read(saved))
			_, err = SavePromptTemplate(test.build(tooLong))
			require.ErrorContains(t, err, "16 KiB")
		})
	}
}

func TestDeletePromptTemplateUninstallsTokensAndRemovesFiles(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	// 先创建保留模板，使待删除模板持有最大主键，覆盖 SQLite 复用主键的场景。
	other, err := SavePromptTemplate(PromptTemplate{Title: "保留的提示词", Content: "保留正文", Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "研发技能包", Kind: "bundle", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{
		{ID: "fixed-review", Title: "审查", Description: "审查代码", Path: "review/SKILL.md", Content: "旧审查正文", Resources: []PromptSkillResource{{Path: "references/a.md", Content: "引用"}}},
		{Title: "测试", Description: "编写测试", Path: "test/SKILL.md", Content: "测试正文"},
	}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}, {TemplateID: other.ID, Enabled: true, SortOrder: 1}}, []int{group.Id}))
	page, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, "fixed-review", pack.Version, "", 0, 1024)
	require.NoError(t, err)
	assert.Equal(t, "旧审查正文", page.Content)

	_, err = DeletePromptTemplate(pack.ID, pack.Version+1)
	require.ErrorIs(t, err, ErrPromptVersionConflict)
	catalog, err := GetActiveTokenSkills(token.Id, group.Code)
	require.NoError(t, err)
	assert.Len(t, catalog, 2, "版本冲突不能删除任何数据")

	uninstalled, err := DeletePromptTemplate(pack.ID, pack.Version)
	require.NoError(t, err)
	assert.Equal(t, 1, uninstalled)
	for _, table := range []interface{}{&PromptTemplateGroup{}, &TokenPromptBinding{}, &PromptSkillRecord{}, &PromptSkillFile{}, &PromptSkillChunk{}} {
		var count int64
		require.NoError(t, DB.Model(table).Where("template_id = ?", pack.ID).Count(&count).Error)
		assert.Zero(t, count, "%T 仍残留已删除模板的数据", table)
	}
	catalog, err = GetActiveTokenSkills(token.Id, group.Code)
	require.NoError(t, err)
	assert.Empty(t, catalog)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, "fixed-review", pack.Version, "", 0, 1024)
	require.Error(t, err)
	installed, err := GetTokenPromptBindings(token.UserId, token.Id)
	require.NoError(t, err)
	require.Len(t, installed, 1)
	assert.Equal(t, other.ID, installed[0].Template.ID)
	_, err = DeletePromptTemplate(pack.ID, pack.Version)
	require.ErrorIs(t, err, ErrPromptTemplateNotFound)

	// 新模板即使拿到相同主键、版本和技能 ID，也只能读到自己的正文。
	recreated, err := SavePromptTemplate(PromptTemplate{Title: "新技能包", Kind: "bundle", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{
		{ID: "fixed-review", Title: "审查", Description: "审查代码", Path: "review/SKILL.md", Content: "新审查正文"},
		{Title: "测试", Description: "编写测试", Path: "test/SKILL.md", Content: "测试正文"},
	}})
	require.NoError(t, err)
	require.Equal(t, pack.ID, recreated.ID, "前置条件：SQLite 复用了已删除模板的主键")
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: recreated.ID, Enabled: true}}, []int{group.Id}))
	page, err = ReadActiveTokenSkillFile(token.Id, group.Code, recreated.ID, "fixed-review", recreated.Version, "", 0, 1024)
	require.NoError(t, err)
	assert.Equal(t, "新审查正文", page.Content)
}
