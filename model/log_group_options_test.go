package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogGroupNamesRespectEntityScope(t *testing.T) {
	db := openGroupIdentityTestDB(t)
	require.NoError(t, db.AutoMigrate(&Group{}, &GroupAlias{}, &Option{}))
	allowed := Group{Code: "42", Name: "当前可用名称", Status: GroupStatusActive}
	hidden := Group{Code: "private", Name: "隐藏组名称", Status: GroupStatusActive}
	require.NoError(t, db.Create(&allowed).Error)
	require.NoError(t, db.Create(&hidden).Error)
	require.NoError(t, db.Create(&[]GroupAlias{
		{Alias: "old-public", GroupId: allowed.Id},
		{Alias: "private", GroupId: allowed.Id},
		{Alias: "old-private", GroupId: hidden.Id},
		{Alias: "deleted", GroupId: 9999},
	}).Error)
	// 历史标签相同不代表这些标识归属于可用组。
	require.NoError(t, db.Create(&Option{Key: "UserUsableGroups", Value: `{"unrelated":"当前可用名称","old-private":"当前可用名称"}`}).Error)

	names, err := GetLogGroupNamesForIDs(nil)
	require.NoError(t, err)
	assert.Empty(t, names)
	names, err = GetLogGroupNamesForIDs([]int{allowed.Id})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"42": "当前可用名称", "old-public": "当前可用名称"}, names)

	require.NoError(t, db.Model(&allowed).Update("name", "改名后的名称").Error)
	names, err = GetLogGroupNamesForIDs([]int{allowed.Id, 9999})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"42": "改名后的名称", "old-public": "改名后的名称"}, names)
	require.NoError(t, db.Delete(&allowed).Error)
	names, err = GetLogGroupNamesForIDs([]int{allowed.Id})
	require.NoError(t, err)
	assert.Empty(t, names)
}
