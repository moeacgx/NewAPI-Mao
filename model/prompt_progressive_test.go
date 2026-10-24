package model

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProgressiveSkillTest(t *testing.T) (*Group, *Group, *Token) {
	t.Helper()
	ClearProgressiveSkillContentCache()
	t.Cleanup(ClearProgressiveSkillContentCache)
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}, &PromptSkillRecord{}, &PromptSkillFile{}, &PromptSkillChunk{}))
	token := &Token{UserId: 321, Key: "progressive-test-key", Group: first.Code}
	require.NoError(t, DB.Create(token).Error)
	return first, second, token
}

func TestProgressiveSkillsPreserveLargeFilesAndStableIDsAcrossEdits(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	body := strings.Repeat("中文段落。", 9000)
	resource := strings.Repeat("资料引用。", 6000)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "研发技能包", Kind: "bundle", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{
		{Title: "审查", Description: "审查代码时使用", Path: "skills/review/SKILL.md", Content: body, Resources: []PromptSkillResource{{Path: "references/规范.md", Content: resource}}},
		{Title: "审查", Description: "另一独立技能", Path: "skills/review-two/SKILL.md", Content: "第二个技能的正文"},
	}})
	require.NoError(t, err)
	require.Len(t, pack.Skills, 2)
	firstID, secondID := pack.Skills[0].ID, pack.Skills[1].ID
	assert.NotEqual(t, firstID, secondID)
	assert.Empty(t, pack.Content, "合集不再保存全量拼接正文")
	var stored PromptTemplate
	require.NoError(t, DB.First(&stored, pack.ID).Error)
	assert.Empty(t, stored.Content)
	assert.Empty(t, stored.SkillsJSON)

	// 已启用的普通提示词恰好用满旧预算，渐进式文件不占这份预算。
	bindings := []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}
	for i := 0; i < 2; i++ {
		prompt, saveErr := SavePromptTemplate(PromptTemplate{Title: "普通提示词", Content: strings.Repeat("x", MaxPromptContentBytes), Enabled: true, GroupIDs: []int{group.Id}})
		require.NoError(t, saveErr)
		bindings = append(bindings, TokenPromptBindingInput{TemplateID: prompt.ID, Enabled: true})
	}
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, bindings, []int{group.Id}))
	eager, err := GetActiveTokenPrompts(token.Id, group.Code)
	require.NoError(t, err)
	assert.Len(t, eager, 2)
	catalog, err := GetActiveTokenSkills(token.Id, group.Code)
	require.NoError(t, err)
	require.Len(t, catalog, 2)
	assert.Equal(t, "审查代码时使用", catalog[0].Description)
	assert.Equal(t, firstID, catalog[0].SkillID)

	var received strings.Builder
	for offset := 0; offset < len(body); {
		page, readErr := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, firstID, pack.Version, "", offset, MaxSkillReadBytes-1)
		require.NoError(t, readErr)
		require.True(t, utf8.ValidString(page.Content))
		assert.Equal(t, offset, page.Offset)
		assert.Equal(t, len(body), page.TotalBytes)
		assert.Equal(t, []string{"references/规范.md"}, page.Resources)
		require.Greater(t, page.NextOffset, offset)
		received.WriteString(page.Content)
		offset = page.NextOffset
		assert.Equal(t, offset == len(body), page.EOF)
	}
	assert.Equal(t, body, received.String())
	page, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, firstID, pack.Version, "references/规范.md", 0, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Equal(t, resource[:page.NextOffset], page.Content)
	assert.Empty(t, page.Resources)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, secondID, pack.Version, "references/规范.md", 0, 100)
	require.Error(t, err, "同一安装包的另一技能也不能读取它的资源")
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, firstID, pack.Version, "", 1, 100)
	require.ErrorContains(t, err, "UTF-8")

	items, err := ListPromptTemplates(false)
	require.NoError(t, err)
	assert.Equal(t, body, items[0].Skills[0].Content)
	assert.Equal(t, resource, items[0].Skills[0].Resources[0].Content)
	pack.Skills[0], pack.Skills[1] = pack.Skills[1], pack.Skills[0]
	pack.Skills[1].Content += "修订"
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	assert.Equal(t, secondID, pack.Skills[0].ID)
	assert.Equal(t, firstID, pack.Skills[1].ID)
	assert.Equal(t, body+"修订", pack.Skills[1].Content)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, firstID, pack.Version-1, "", 0, 100)
	require.ErrorContains(t, err, "版本")
	_, err = SavePromptTemplate(PromptTemplate{Title: "冒用包", LoadMode: PromptLoadModeProgressive, Content: "正文", Skills: []PromptSkill{{ID: firstID}}})
	require.ErrorContains(t, err, "其他技能包")
	other, err := SavePromptTemplate(PromptTemplate{Title: "另一技能包", Description: "独立资料", LoadMode: PromptLoadModeProgressive, Content: "原正文", Skills: []PromptSkill{{Resources: []PromptSkillResource{{Path: "references/original.txt", Content: "原引用"}}}}})
	require.NoError(t, err)
	other.Title, other.Content, other.Skills[0].ID = "不能提交的修改", "替换正文", firstID
	_, err = SavePromptTemplate(*other)
	require.ErrorContains(t, err, "其他技能包")
	items, err = ListPromptTemplates(false)
	require.NoError(t, err)
	var unchanged *PromptTemplate
	for _, item := range items {
		if item.ID == other.ID {
			unchanged = &item
		}
	}
	require.NotNil(t, unchanged)
	assert.Equal(t, "另一技能包", unchanged.Title)
	assert.Equal(t, other.Version, unchanged.Version)
	assert.Equal(t, "原正文", unchanged.Content)
	assert.Equal(t, []PromptSkillResource{{Path: "references/original.txt", Content: "原引用"}}, unchanged.Skills[0].Resources)
}

func TestProgressiveSingleSkillAuthorizationAndResourcePreservation(t *testing.T) {
	group, otherGroup, token := setupProgressiveSkillTest(t)
	require.NoError(t, DB.Create(&GroupAlias{Alias: "old-progressive-group", GroupId: group.Id}).Error)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "单技能", Description: "按需检查", LoadMode: PromptLoadModeProgressive, Content: "主正文", Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{{Path: "kit/SKILL.md", Resources: []PromptSkillResource{{Path: "references/说明.txt", Content: "引用内容"}}}}})
	require.NoError(t, err)
	require.Len(t, pack.Skills, 1)
	assert.Equal(t, "kit/SKILL.md", pack.Skills[0].Path)
	assert.Equal(t, "引用内容", pack.Skills[0].Resources[0].Content)
	bindings := []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, bindings, []int{group.Id}))
	id := pack.Skills[0].ID
	page, err := ReadActiveTokenSkillFile(token.Id, "old-progressive-group", pack.ID, id, pack.Version, "SKILL.md", 0, 100)
	require.NoError(t, err)
	assert.Equal(t, "主正文", page.Content)
	for _, code := range []string{otherGroup.Code, "auto", "不存在"} {
		catalog, lookupErr := GetActiveTokenSkills(token.Id, code)
		require.NoError(t, lookupErr)
		assert.Empty(t, catalog)
		_, readErr := ReadActiveTokenSkillFile(token.Id, code, pack.ID, id, pack.Version, "", 0, 100)
		require.Error(t, readErr)
	}
	for _, path := range []string{"../secret", "/secret", "references\\secret", "C:/secret"} {
		_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, id, pack.Version, path, 0, 100)
		require.Error(t, err)
	}
	pack.Content = "更新正文"
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	assert.Equal(t, id, pack.Skills[0].ID)
	assert.Equal(t, "引用内容", pack.Skills[0].Resources[0].Content)
	assert.Equal(t, "更新正文", pack.Skills[0].Content)
	pack.Skills[0].Resources = nil
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, id, pack.Version, "references/说明.txt", 0, 100)
	require.ErrorContains(t, err, "不存在", "保存移除资料后，新版本不能读取旧文件分块")
	page, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, id, pack.Version, "", 0, 100)
	require.NoError(t, err)
	assert.Empty(t, page.Resources)
	for _, operation := range []string{"停用绑定", "卸载", "下架", "移出白名单", "分组停用", "令牌停用", "令牌过期", "令牌删除"} {
		t.Run(operation, func(t *testing.T) {
			require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]interface{}{"status": common.TokenStatusEnabled, "expired_time": -1, "deleted_at": nil}).Error)
			require.NoError(t, DB.Model(&PromptTemplate{}).Where("id = ?", pack.ID).Update("enabled", true).Error)
			require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("status", GroupStatusActive).Error)
			require.NoError(t, DB.Where("template_id = ?", pack.ID).Delete(&PromptTemplateGroup{}).Error)
			require.NoError(t, DB.Create(&PromptTemplateGroup{TemplateID: pack.ID, GroupID: group.Id}).Error)
			require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, bindings, []int{group.Id}))
			switch operation {
			case "停用绑定":
				require.NoError(t, DB.Model(&TokenPromptBinding{}).Where("token_id = ?", token.Id).Update("enabled", false).Error)
			case "卸载":
				require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{}, []int{group.Id}))
			case "下架":
				require.NoError(t, DB.Model(&PromptTemplate{}).Where("id = ?", pack.ID).Update("enabled", false).Error)
			case "移出白名单":
				require.NoError(t, DB.Where("template_id = ?", pack.ID).Delete(&PromptTemplateGroup{}).Error)
			case "分组停用":
				require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("status", GroupStatusDisabled).Error)
			case "令牌停用":
				require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("status", common.TokenStatusDisabled).Error)
			case "令牌过期":
				require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("expired_time", common.GetTimestamp()-1).Error)
			case "令牌删除":
				require.NoError(t, DB.Delete(&Token{}, token.Id).Error)
			}
			_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, id, pack.Version, "", 0, 100)
			require.Error(t, err)
			catalog, lookupErr := GetActiveTokenSkills(token.Id, group.Code)
			require.NoError(t, lookupErr)
			assert.Empty(t, catalog)
		})
	}
}

func TestProgressiveSkillValidationSeparatesFileAndPackageBudgets(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*PromptTemplate)
		want string
	}{
		{"发布缺少说明", func(p *PromptTemplate) { p.Enabled = true; p.Description = "" }, "用途说明"},
		{"单文件上限", func(p *PromptTemplate) { p.Content = strings.Repeat("a", MaxSkillFileBytes+1) }, "8 MiB"},
		{"资源越界", func(p *PromptTemplate) {
			p.Skills = []PromptSkill{{Resources: []PromptSkillResource{{Path: "../secret"}}}}
		}, "路径"},
		{"资源覆盖主文件", func(p *PromptTemplate) {
			p.Skills = []PromptSkill{{Resources: []PromptSkillResource{{Path: "skill.md"}}}}
		}, "路径"},
		{"资源不能是根目录", func(p *PromptTemplate) {
			p.Skills = []PromptSkill{{Resources: []PromptSkillResource{{Path: "."}}}}
		}, "路径"},
		{"资源无效编码", func(p *PromptTemplate) {
			p.Skills = []PromptSkill{{Resources: []PromptSkillResource{{Path: "ref.txt", Content: string([]byte{0xff})}}}}
		}, "UTF-8"},
		{"身份重复", func(p *PromptTemplate) {
			p.Kind = "bundle"
			p.Skills = []PromptSkill{{ID: "duplicate", Title: "一", Content: "正文"}, {ID: "duplicate", Title: "二", Content: "正文"}}
		}, "身份"},
		{"无效模式", func(p *PromptTemplate) { p.LoadMode = "invalid" }, "模式"},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := PromptTemplate{Title: "技能", Description: "用途", Content: "正文", LoadMode: PromptLoadModeProgressive}
			test.edit(&p)
			require.ErrorContains(t, NormalizePromptTemplateContent(&p), test.want)
		})
	}
	large := PromptTemplate{Title: "按需技能", Content: strings.Repeat("a", MaxPromptContentBytes*3), LoadMode: PromptLoadModeProgressive}
	require.NoError(t, NormalizePromptTemplateContent(&large))
	large.LoadMode = PromptLoadModeEager
	require.ErrorContains(t, NormalizePromptTemplateContent(&large), "32768")
	shared := strings.Repeat("a", MaxSkillFileBytes)
	large = PromptTemplate{Title: "超大包", Content: "正文", LoadMode: PromptLoadModeProgressive, Skills: []PromptSkill{{}}}
	for i := 0; i < 8; i++ {
		large.Skills[0].Resources = append(large.Skills[0].Resources, PromptSkillResource{Path: fmt.Sprintf("references/%d.txt", i), Content: shared})
	}
	require.ErrorContains(t, NormalizePromptTemplateContent(&large), "64 MiB")
	large.Skills[0].Resources = nil
	for i := 0; i < MaxSkillResourceFiles; i++ {
		large.Skills[0].Resources = append(large.Skills[0].Resources, PromptSkillResource{Path: fmt.Sprintf("references/%d.txt", i)})
	}
	require.ErrorContains(t, NormalizePromptTemplateContent(&large), "4096")
}

func TestProgressiveCatalogDoesNotReadBodiesAndModeChangePurgesOwnedFiles(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "目录独立", Description: "按需查阅", Content: "私有正文", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{{Resources: []PromptSkillResource{{Path: "references/data.txt", Content: "引用正文"}}}}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}, []int{group.Id}))
	// 正文损坏也不应阻断用户浏览目录；目录与文件读取是独立契约。
	require.NoError(t, DB.Model(&PromptSkillChunk{}).Where("template_id = ?", pack.ID).Update("content", "").Error)
	_, err = ListPromptTemplates(false)
	require.ErrorContains(t, err, "分块不完整")
	summaries, err := ListPromptTemplateSummaries(true)
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, pack.Skills[0].ID, summaries[0].Skills[0].ID)
	assert.Equal(t, "按需查阅", summaries[0].Skills[0].Description)
	assert.Empty(t, summaries[0].Content)
	assert.Empty(t, summaries[0].Skills[0].Content)
	assert.Empty(t, summaries[0].Skills[0].Resources)
	installed, err := GetTokenPromptBindingSummaries(token.UserId, token.Id)
	require.NoError(t, err)
	require.Len(t, installed, 1)
	assert.Empty(t, installed[0].Template.Skills[0].Content)
	pack.LoadMode, pack.Content = PromptLoadModeEager, "普通全文提示词"
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	for _, table := range []interface{}{&PromptSkillRecord{}, &PromptSkillFile{}, &PromptSkillChunk{}} {
		var count int64
		require.NoError(t, DB.Model(table).Where("template_id = ?", pack.ID).Count(&count).Error)
		assert.Zero(t, count, "改回全文模式必须清理引用文件，不能留下可读取的孤儿资料")
	}
	catalog, err := GetActiveTokenSkills(token.Id, group.Code)
	require.NoError(t, err)
	assert.Empty(t, catalog)
	eager, err := GetActiveTokenPrompts(token.Id, group.Code)
	require.NoError(t, err)
	require.Len(t, eager, 1)
	assert.Equal(t, "普通全文提示词", eager[0].Content)
}

func TestProgressiveFileReadRejectsMissingChunks(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "完整性验证", Description: "按需读取", Content: strings.Repeat("a", 100*1024), LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}, []int{group.Id}))
	// 页起点在丢失块中、终点落在后续块时，必须返回错误，不能错误拼页或发生切片越界。
	require.NoError(t, DB.Where("template_id = ? AND chunk_index = ?", pack.ID, 0).Delete(&PromptSkillChunk{}).Error)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 40*1024, MaxSkillReadBytes)
	require.ErrorContains(t, err, "分块不完整")
	_, err = ListPromptTemplates(false)
	require.ErrorContains(t, err, "分块不完整")
}

func TestProgressiveBundleModeChangeDropsResourceStorage(t *testing.T) {
	group, _, _ := setupProgressiveSkillTest(t)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "切换合集模式", LoadMode: PromptLoadModeProgressive, Kind: "bundle", GroupIDs: []int{group.Id}, Skills: []PromptSkill{
		{Title: "审查", Content: "先检查代码。", Resources: []PromptSkillResource{{Path: "references/large.txt", Content: strings.Repeat("x", 100*1024)}}},
		{Title: "写作", Content: "使用简单语言。"},
	}})
	require.NoError(t, err)
	pack.LoadMode = PromptLoadModeEager
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	assert.Contains(t, pack.Content, "先检查代码。")
	assert.Contains(t, pack.Content, "使用简单语言。")
	for _, skill := range pack.Skills {
		assert.Empty(t, skill.ID)
		assert.Empty(t, skill.Path)
		assert.Empty(t, skill.Resources)
	}
	var count int64
	require.NoError(t, DB.Model(&PromptSkillFile{}).Where("template_id = ?", pack.ID).Count(&count).Error)
	assert.Zero(t, count)
	var stored PromptTemplate
	require.NoError(t, DB.First(&stored, pack.ID).Error)
	assert.NotContains(t, stored.SkillsJSON, "references/")
	assert.Less(t, len(stored.SkillsJSON), 1000)
}

func TestPromptTemplateRejectsValuesUnsupportedByTextColumns(t *testing.T) {
	for _, test := range []struct {
		name  string
		input PromptTemplate
		want  string
	}{
		{"元数据中的 NUL", PromptTemplate{Title: "技能", Description: "不合法\x00说明", Content: "正文", LoadMode: PromptLoadModeProgressive}, "说明必须为有效文本"},
		{"合集子项中的 NUL", PromptTemplate{Title: "合集", Kind: "bundle", LoadMode: PromptLoadModeProgressive, Skills: []PromptSkill{{Title: "一", Description: "不合法\x00", Content: "一"}, {Title: "二", Content: "二"}}}, "说明必须为有效文本"},
		{"旧正文中的 NUL", PromptTemplate{Title: "旧提示词", Content: "不合法\x00正文"}, "UTF-8"},
		{"旧合集 JSON 转义膨胀", PromptTemplate{Title: "旧合集", Kind: "bundle", Skills: []PromptSkill{{Title: "一", Content: strings.Repeat("\x01", 16000)}, {Title: "二", Content: "另一个技能"}}}, "TEXT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorContains(t, NormalizePromptTemplateContent(&test.input), test.want)
		})
	}
}
