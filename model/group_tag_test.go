package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
