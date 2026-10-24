package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenGroupChannelStableIdentityAndGroupRemoval(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	token := &Token{UserId: 1, Key: "pin-key", GroupMode: TokenGroupModeExplicit, GroupIds: []int{first.Id, second.Id}, GroupChannelIds: map[int]int{first.Id: 951, second.Id: 952}}
	require.NoError(t, token.Insert())
	require.NoError(t, DB.Model(first).Updates(map[string]any{"name": "新显示名"}).Error)
	loaded, err := GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Equal(t, map[int]int{first.Id: 951, second.Id: 952}, loaded.GroupChannelIds)
	assert.Equal(t, "新显示名", loaded.GroupDetails[0].Name)
	loaded.GroupIds = []int{second.Id}
	require.NoError(t, loaded.Update())
	loaded, err = GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	assert.Equal(t, map[int]int{second.Id: 952}, loaded.GroupChannelIds)
	loaded.GroupMode = TokenGroupModeAuto
	require.NoError(t, loaded.Update())
	loaded, err = GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Empty(t, loaded.GroupChannelIds)
}

func TestTokenGroupChannelMigrationPreservesOtherGroupsAndRejectsPinnedSource(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	third := &Group{Code: "third", Name: "第三显示名", Status: GroupStatusActive, Ratio: 1}
	require.NoError(t, DB.Create(third).Error)
	token := &Token{UserId: 1, Key: "migration-pin", Status: common.TokenStatusEnabled, GroupMode: TokenGroupModeExplicit, GroupIds: []int{first.Id, third.Id}, GroupChannelIds: map[int]int{third.Id: 952}}
	require.NoError(t, token.Insert())
	_, err := MigrateTokenGroup(first.Id, second.Id)
	require.NoError(t, err)
	loaded, err := GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Equal(t, map[int]int{third.Id: 952}, loaded.GroupChannelIds)
	_, err = PreviewTokenGroupMigration(third.Id, first.Id)
	require.Error(t, err, "迁移不能静默解除源组限定")
	_, err = MigrateTokenGroup(third.Id, first.Id)
	require.Error(t, err)
	loaded, err = GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Equal(t, map[int]int{third.Id: 952}, loaded.GroupChannelIds)
}
