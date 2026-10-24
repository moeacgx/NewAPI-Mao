package service

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const repositoryUpdateTree = `{"tree":[{"path":"skills/review/SKILL.md","mode":"100644","type":"blob","size":40},{"path":"skills/review/references/old.md","mode":"100644","type":"blob","size":10},{"path":"skills/test/SKILL.md","mode":"100644","type":"blob","size":40}]}`

func setupRepositoryUpdateTest(t *testing.T) (*model.Group, *model.Token) {
	t.Helper()
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldMainType, oldLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	model.InitDBColumns()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "repository-update.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.Token{}, &model.TokenGroupBinding{}, &model.PromptCategory{}, &model.PromptTemplate{},
		&model.PromptTemplateGroup{}, &model.TokenPromptBinding{}, &model.PromptSkillRecord{}, &model.PromptSkillFile{}, &model.PromptSkillChunk{}))
	model.DB, model.LOG_DB = db, db
	model.ClearProgressiveSkillContentCache()
	t.Cleanup(func() {
		model.ClearProgressiveSkillContentCache()
		model.DB, model.LOG_DB = oldDB, oldLogDB
		common.SetDatabaseTypes(oldMainType, oldLogType)
		model.InitDBColumns()
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	group := &model.Group{Code: "repo-update", Name: "仓库技能分组", Ratio: 1, Status: model.GroupStatusActive}
	require.NoError(t, db.Create(group).Error)
	token := &model.Token{UserId: 7, Key: "repository-update-key", Group: group.Code}
	require.NoError(t, db.Create(token).Error)
	return group, token
}

// importRepositoryTemplate 模拟管理员导入后配置分类、白名单、上架和排序，再安装到令牌。
func importRepositoryTemplate(t *testing.T, group *model.Group, token *model.Token, raw map[string]string) *model.PromptTemplate {
	t.Helper()
	var requests []string
	draft, err := importRepositorySkills(context.Background(), "https://github.com/acme/skills", repositoryFixtureClient(t, repositoryUpdateTree, raw, &requests))
	require.NoError(t, err)
	category, err := model.SavePromptCategory(model.PromptCategory{Name: "研发"})
	require.NoError(t, err)
	draft.Title, draft.Description = "站内技能包", "站内整理的说明"
	draft.CategoryID, draft.Enabled, draft.SortOrder, draft.GroupIDs = category.ID, true, 9, []int{group.Id}
	saved, err := model.SavePromptTemplate(*draft)
	require.NoError(t, err)
	require.NotNil(t, saved.RepositorySource)
	require.False(t, saved.RepositorySource.LocallyModified)
	require.NoError(t, model.ReplaceTokenPromptBindings(token.UserId, token.Id, []model.TokenPromptBindingInput{{TemplateID: saved.ID, Enabled: true}}, []int{group.Id}))
	return saved
}

func repositoryUpdateRaw(review string) map[string]string {
	return map[string]string{
		"skills/review/SKILL.md":          "---\nname: review\ndescription: 审查代码时使用\n---\n" + review,
		"skills/review/references/old.md": "旧引用",
		"skills/test/SKILL.md":            "---\nname: test\ndescription: 编写测试时使用\n---\n测试正文",
	}
}

func TestRepositoryUpdateReplacesFilesAndKeepsSiteSettings(t *testing.T) {
	group, token := setupRepositoryUpdateTest(t)
	saved := importRepositoryTemplate(t, group, token, repositoryUpdateRaw("旧审查正文"))
	reviewID := saved.Skills[0].ID
	require.Equal(t, "skills/review/SKILL.md", saved.Skills[0].Path)

	var requests []string
	unchanged, updated, err := updatePromptTemplateFromRepository(context.Background(), saved.ID, saved.Version, false, repositoryFixtureClient(t, repositoryUpdateTree, repositoryUpdateRaw("旧审查正文"), &requests))
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, saved.Version, unchanged.Version, "上游没有变化时不写库")

	// 上游修改审查正文、删除引用文件并新增一个技能。
	tree := `{"tree":[{"path":"skills/review/SKILL.md","mode":"100644","type":"blob","size":40},{"path":"skills/test/SKILL.md","mode":"100644","type":"blob","size":40},{"path":"skills/zeta/SKILL.md","mode":"100644","type":"blob","size":40}]}`
	raw := repositoryUpdateRaw("新审查正文")
	delete(raw, "skills/review/references/old.md")
	raw["skills/zeta/SKILL.md"] = "---\nname: zeta\ndescription: 新增技能\n---\n新增正文"
	requests = nil
	result, updated, err := updatePromptTemplateFromRepository(context.Background(), saved.ID, saved.Version, false, repositoryFixtureClient(t, tree, raw, &requests))
	require.NoError(t, err)
	require.True(t, updated)
	assert.Equal(t, saved.ID, result.ID)
	assert.Equal(t, saved.Version+1, result.Version)
	assert.Equal(t, "站内技能包", result.Title)
	assert.Equal(t, "站内整理的说明", result.Description)
	assert.Equal(t, saved.CategoryID, result.CategoryID)
	assert.True(t, result.Enabled)
	assert.Equal(t, 9, result.SortOrder)
	assert.Equal(t, []int{group.Id}, result.GroupIDs)
	require.Len(t, result.Skills, 3)
	assert.Equal(t, reviewID, result.Skills[0].ID, "相同主文件路径沿用技能 ID")
	assert.Equal(t, "新审查正文", result.Skills[0].Content)
	assert.Empty(t, result.Skills[0].Resources, "上游删除的引用文件同步删除")
	assert.Equal(t, "zeta", result.Skills[2].Title)
	require.NotNil(t, result.RepositorySource)
	assert.False(t, result.RepositorySource.LocallyModified)
	assert.Equal(t, "https://github.com/acme/skills", result.RepositorySource.URL)
	assert.NotEqual(t, saved.RepositorySource.ContentHash, result.RepositorySource.ContentHash)

	installed, err := model.GetTokenPromptBindings(token.UserId, token.Id)
	require.NoError(t, err)
	require.Len(t, installed, 1, "令牌安装关系保留")
	page, err := model.ReadActiveTokenSkillFile(token.Id, group.Code, result.ID, reviewID, result.Version, "", 0, 1024)
	require.NoError(t, err)
	assert.Equal(t, "新审查正文", page.Content)
	_, err = model.ReadActiveTokenSkillFile(token.Id, group.Code, result.ID, reviewID, result.Version, "references/old.md", 0, 1024)
	require.Error(t, err)
}

func TestRepositoryUpdateRequiresConfirmationBeforeOverwritingLocalEdits(t *testing.T) {
	group, token := setupRepositoryUpdateTest(t)
	saved := importRepositoryTemplate(t, group, token, repositoryUpdateRaw("旧审查正文"))
	saved.Skills[0].Content = "站内修订正文"
	edited, err := model.SavePromptTemplate(*saved)
	require.NoError(t, err)
	require.True(t, edited.RepositorySource.LocallyModified)

	client := &http.Client{Transport: skillRepositoryTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("未确认覆盖前不能访问 GitHub")
		return nil, nil
	})}
	_, _, err = updatePromptTemplateFromRepository(context.Background(), edited.ID, edited.Version, false, client)
	require.ErrorIs(t, err, ErrRepositoryLocalChanges)
	_, _, err = updatePromptTemplateFromRepository(context.Background(), edited.ID, edited.Version-1, true, client)
	require.ErrorIs(t, err, model.ErrPromptVersionConflict)

	var requests []string
	result, updated, err := updatePromptTemplateFromRepository(context.Background(), edited.ID, edited.Version, true, repositoryFixtureClient(t, repositoryUpdateTree, repositoryUpdateRaw("旧审查正文"), &requests))
	require.NoError(t, err)
	assert.True(t, updated, "确认覆盖后即使上游未变也恢复为来源内容")
	assert.Equal(t, "旧审查正文", result.Skills[0].Content)
	assert.False(t, result.RepositorySource.LocallyModified)
}

func TestRepositoryUpdateFailureKeepsCurrentCopy(t *testing.T) {
	group, token := setupRepositoryUpdateTest(t)
	saved := importRepositoryTemplate(t, group, token, repositoryUpdateRaw("旧审查正文"))
	client := &http.Client{Transport: skillRepositoryTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})}
	_, _, err := updatePromptTemplateFromRepository(context.Background(), saved.ID, saved.Version, false, client)
	require.ErrorContains(t, err, "站内技能保持不变")
	current, err := model.GetAdminPromptTemplate(saved.ID)
	require.NoError(t, err)
	assert.Equal(t, saved.Version, current.Version)
	assert.Equal(t, "旧审查正文", current.Skills[0].Content)

	manual, err := model.SavePromptTemplate(model.PromptTemplate{Title: "手写技能", Content: "正文"})
	require.NoError(t, err)
	_, _, err = updatePromptTemplateFromRepository(context.Background(), manual.ID, manual.Version, false, client)
	require.ErrorContains(t, err, "没有记录 GitHub 来源")
}
