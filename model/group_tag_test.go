package model

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupTagPersistsLogosAndStableBindings(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	tag, err := SaveGroupTag(GroupTag{
		Name: " 国产模型 ", Description: "多个供应商", SortOrder: 4,
		Icons:    []string{"DeepSeek.Color", "https://cdn.example.com/kimi.svg", "DeepSeek.Color"},
		GroupIDs: []int{second.Id, first.Id, second.Id},
	})
	require.NoError(t, err)
	assert.Equal(t, "国产模型", tag.Name)
	assert.Equal(t, []int{first.Id, second.Id}, tag.GroupIDs)
	assert.Equal(t, []string{"DeepSeek.Color", "https://cdn.example.com/kimi.svg"}, tag.Icons)
	require.NoError(t, DB.Create(&GroupAlias{Alias: "historical", GroupId: second.Id}).Error)
	require.NoError(t, DB.Model(second).Updates(map[string]interface{}{"name": "改名后的显示名称", "code": "new-code"}).Error)
	alias, err := GetGroupByCodeOrAlias("historical")
	require.NoError(t, err)
	assert.Equal(t, second.Id, alias.Id)
	assert.Equal(t, "改名后的显示名称", alias.Name)
	stored, err := GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Equal(t, tag.GroupIDs, stored[0].GroupIDs)
	assert.Equal(t, tag.Icons, stored[0].Icons)

	// 旧管理端仅保存分组属性，不会清空独立保存的标签绑定。
	config := first.ToConfig(nil)
	config.Name = "默认分组新名称"
	require.NoError(t, SaveGroupConfig([]GroupConfig{config}, nil))
	stored, err = GetGroupTags(nil)
	require.NoError(t, err)
	assert.Equal(t, tag.GroupIDs, stored[0].GroupIDs)
}

func TestGroupTagProjectionDoesNotExposeUnavailableGroups(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	shared, err := SaveGroupTag(GroupTag{Name: "混合平台", GroupIDs: []int{first.Id, second.Id}})
	require.NoError(t, err)
	_, err = SaveGroupTag(GroupTag{Name: "隐藏平台", GroupIDs: []int{second.Id}})
	require.NoError(t, err)
	_, err = SaveGroupTag(GroupTag{Name: "空标签"})
	require.NoError(t, err)
	tags, err := GetGroupTags([]int{first.Id})
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, shared.Id, tags[0].Id)
	assert.Equal(t, []int{first.Id}, tags[0].GroupIDs)
	empty, err := GetGroupTags([]int{})
	require.NoError(t, err)
	assert.Empty(t, empty)
	assert.NotNil(t, empty)
}

func TestGroupTagInvalidUpdatePreservesStoredValues(t *testing.T) {
	first, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	tag, err := SaveGroupTag(GroupTag{Name: "原标签", Icons: []string{"OpenAI"}, GroupIDs: []int{first.Id}})
	require.NoError(t, err)
	tests := []struct {
		name   string
		mutate func(*GroupTag)
	}{
		{"空名称", func(tag *GroupTag) { tag.Name = " " }},
		{"超长名称", func(tag *GroupTag) { tag.Name = strings.Repeat("字", 65) }},
		{"超长说明", func(tag *GroupTag) { tag.Description = strings.Repeat("字", 513) }},
		{"排序越界", func(tag *GroupTag) { tag.SortOrder = 1000001 }},
		{"不存在的分组", func(tag *GroupTag) { tag.GroupIDs = []int{99999} }},
		{"虚拟分组", func(tag *GroupTag) { tag.GroupIDs = []int{0} }},
		{"脚本地址", func(tag *GroupTag) { tag.Icons = []string{"javascript:alert(1)"} }},
		{"内联图片", func(tag *GroupTag) { tag.Icons = []string{"data:image/svg+xml,<svg/>"} }},
		{"凭据地址", func(tag *GroupTag) { tag.Icons = []string{"https://user:secret@example.com/a.png"} }},
		{"缺少主机", func(tag *GroupTag) { tag.Icons = []string{"https://"} }},
		{"过多图标", func(tag *GroupTag) { tag.Icons = []string{"A", "B", "C", "D", "E", "F", "G"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			updated := *tag
			updated.Name = "修改"
			test.mutate(&updated)
			_, err := SaveGroupTag(updated)
			require.Error(t, err)
			stored, err := GetGroupTags(nil)
			require.NoError(t, err)
			require.Len(t, stored, 1)
			assert.Equal(t, "原标签", stored[0].Name)
			assert.Equal(t, tag.Icons, stored[0].Icons)
			assert.Equal(t, tag.GroupIDs, stored[0].GroupIDs)
		})
	}
}

func TestGroupTagClearAndDeleteDoNotMutateGroups(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	tag, err := SaveGroupTag(GroupTag{Name: "分类", SortOrder: 9, Icons: []string{"OpenAI"}, GroupIDs: []int{first.Id, second.Id}})
	require.NoError(t, err)
	tag.Icons, tag.GroupIDs, tag.SortOrder, tag.Description = []string{}, []int{}, 0, ""
	_, err = SaveGroupTag(*tag)
	require.NoError(t, err)
	stored, err := GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, stored, 1)
	assert.Empty(t, stored[0].GroupIDs)
	assert.Empty(t, stored[0].Icons)
	assert.Zero(t, stored[0].SortOrder)
	require.NoError(t, DeleteGroupTag(tag.Id))
	var groups []Group
	require.NoError(t, DB.Order("id").Find(&groups).Error)
	require.Len(t, groups, 2)
	assert.Equal(t, first.Code, groups[0].Code)
	assert.Equal(t, second.Ratio, groups[1].Ratio)
}

func TestGroupTagGroupDeletionCleansOnlyDeletedGroupBindings(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	tag, err := SaveGroupTag(GroupTag{Name: "分类", GroupIDs: []int{first.Id, second.Id}})
	require.NoError(t, err)
	require.NoError(t, SaveGroupConfig(nil, []int{second.Id}))
	var bindings []GroupTagBinding
	require.NoError(t, DB.Find(&bindings).Error)
	assert.Equal(t, []GroupTagBinding{{TagID: tag.Id, GroupID: first.Id}}, bindings)
	tags, err := GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, []int{first.Id}, tags[0].GroupIDs)
}

func TestGroupTagMatchKeywordsPersistAndAutoBindNewOrRenamedGroups(t *testing.T) {
	_, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&GroupTag{}, &GroupTagBinding{}))
	tag, err := SaveGroupTag(GroupTag{
		Name: "平台标签", MatchKeywords: []string{" GEMINI ", "pro", "gemini"},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"GEMINI", "pro"}, tag.MatchKeywords)
	var stored GroupTag
	require.NoError(t, DB.First(&stored, tag.Id).Error)
	assert.Contains(t, stored.MatchKeywordsJSON, "GEMINI")
	tags, err := GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, []string{"GEMINI", "pro"}, tags[0].MatchKeywords)

	// 新增分组在同一分组保存事务中自动追加稳定 ID 关联。
	require.NoError(t, SaveGroupConfig([]GroupConfig{{
		Code: "gemini-new", Name: "Gemini Pro", Ratio: 1, Status: GroupStatusActive,
	}}, nil))
	var created Group
	require.NoError(t, DB.Where("name = ?", "Gemini Pro").First(&created).Error)

	// 改名后命中另一个关键词也会自动追加，原有绑定保持不变。
	config := second.ToConfig(nil)
	config.Name = "Pro 国内模型"
	require.NoError(t, SaveGroupConfig([]GroupConfig{config}, nil))

	var bindings []GroupTagBinding
	require.NoError(t, DB.Where("tag_id = ?", tag.Id).Order("group_id").Find(&bindings).Error)
	assert.ElementsMatch(t, []GroupTagBinding{
		{TagID: tag.Id, GroupID: created.Id},
		{TagID: tag.Id, GroupID: second.Id},
	}, bindings)

	// 移除规则不会删除既有持久绑定；后续新分组不再按旧词自动加入。
	updatedTags, err := GetGroupTags(nil)
	require.NoError(t, err)
	require.Len(t, updatedTags, 1)
	tag = &updatedTags[0]
	tag.MatchKeywords = []string{"gemini"}
	_, err = SaveGroupTag(*tag)
	require.NoError(t, err)
	var afterRemoval []GroupTagBinding
	require.NoError(t, DB.Where("tag_id = ?", tag.Id).Find(&afterRemoval).Error)
	assert.Len(t, afterRemoval, 2)
	require.NoError(t, SaveGroupConfig([]GroupConfig{{
		Code: "pro-new", Name: "Pro New", Ratio: 1, Status: GroupStatusActive,
	}}, nil))
	var proNew Group
	require.NoError(t, DB.Where("name = ?", "Pro New").First(&proNew).Error)
	var count int64
	require.NoError(t, DB.Model(&GroupTagBinding{}).Where("tag_id = ? AND group_id = ?", tag.Id, proNew.Id).Count(&count).Error)
	assert.Zero(t, count)
}

func TestGroupTagMatchingSeesConcurrentCommittedRuleAndDeletion(t *testing.T) {
	for _, scenario := range []string{"新增规则与新增分组", "删除标签与分组改名", "删除分组与新增规则"} {
		t.Run(scenario, func(t *testing.T) {
			setupGroupBindingsTest(t)
			originalDB := DB
			dsn := filepath.Join(t.TempDir(), "group-tags.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
			db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			require.NoError(t, err)
			otherDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
			require.NoError(t, err)
			DB = db
			t.Cleanup(func() {
				DB = originalDB
				for _, connection := range []*gorm.DB{db, otherDB} {
					sqlDB, err := connection.DB()
					require.NoError(t, err)
					require.NoError(t, sqlDB.Close())
				}
			})
			require.NoError(t, db.AutoMigrate(
				&Option{}, &Group{}, &GroupAlias{}, &AutoGroupMember{},
				&ChannelGroupBinding{}, &TokenGroupBinding{}, &Channel{}, &Token{},
				&User{}, &Ability{}, &GroupTag{}, &GroupTagBinding{},
			))
			group := Group{Code: "internal-model", Name: "Gemini 现有组", Ratio: 1, Status: GroupStatusActive}
			require.NoError(t, db.Create(&group).Error)
			tag, err := normalizeGroupTag(GroupTag{Name: "规则标签", MatchKeywords: []string{"gemini"}})
			require.NoError(t, err)
			if scenario == "删除标签与分组改名" {
				require.NoError(t, db.Create(&tag).Error)
			}

			// 第一个独立连接持有门闩；第二个公共写入口已到达门闩时再提交业务变化。
			// 不取得进程互斥，避免用同进程互斥掩盖数据库锁缺失。
			tx := otherDB.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			require.NoError(t, lockGroupTagWrites(tx))
			secondStarted := make(chan struct{}, 1)
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test_group_tag_start", func(query *gorm.DB) {
				if query.Statement.Table == "options" {
					select {
					case secondStarted <- struct{}{}:
					default:
					}
				}
			}))
			secondDone := make(chan error, 1)
			go func() {
				if scenario == "删除分组与新增规则" {
					_, err := SaveGroupTag(GroupTag{Name: "新增规则", MatchKeywords: []string{"gemini"}})
					secondDone <- err
					return
				}
				config := GroupConfig{Code: "new-temp", Name: "Gemini 新组", Ratio: 1, Status: GroupStatusActive}
				if scenario == "删除标签与分组改名" {
					config = group.ToConfig(nil)
					config.Name = "Gemini 改名组"
				}
				secondDone <- SaveGroupConfig([]GroupConfig{config}, nil)
			}()
			<-secondStarted
			switch scenario {
			case "新增规则与新增分组":
				require.NoError(t, tx.Create(&tag).Error)
			case "删除标签与分组改名":
				require.NoError(t, tx.Delete(&tag).Error)
			case "删除分组与新增规则":
				require.NoError(t, tx.Delete(&group).Error)
			}
			require.NoError(t, tx.Commit().Error)
			require.NoError(t, <-secondDone)
			var bindings []GroupTagBinding
			require.NoError(t, db.Find(&bindings).Error)
			if scenario == "新增规则与新增分组" {
				var created Group
				require.NoError(t, db.Where("name = ?", "Gemini 新组").First(&created).Error)
				assert.Equal(t, []GroupTagBinding{{TagID: tag.Id, GroupID: created.Id}}, bindings)
			} else {
				assert.Empty(t, bindings, "并发删除后不能产生孤儿关联")
			}
		})
	}
}
