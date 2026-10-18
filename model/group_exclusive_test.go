package model

import (
	"errors"
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTokenExclusiveGroupBinding(t *testing.T) {
	defaultGroup, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	single := &Token{GroupMode: TokenGroupModeExplicit, GroupIds: []int{exclusiveGroup.Id}}
	if err := ValidateTokenExclusiveGroupBinding(DB, single); err != nil {
		t.Fatalf("独立分组单独绑定应该成功: %v", err)
	}

	conflict := &Token{GroupMode: TokenGroupModeExplicit, GroupIds: []int{exclusiveGroup.Id, defaultGroup.Id}}
	if err := ValidateTokenExclusiveGroupBinding(DB, conflict); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("独立分组与普通分组应该冲突，实际: %v", err)
	}

	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", false).Error; err != nil {
		t.Fatalf("取消独立分组失败: %v", err)
	}
	if err := ValidateTokenExclusiveGroupBinding(DB, conflict); err != nil {
		t.Fatalf("普通多分组绑定应该保持可用: %v", err)
	}
}

func TestValidateTokenExclusiveGroupBindingCachedRefreshesAfterInvalidation(t *testing.T) {
	defaultGroup, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	InvalidateExclusiveGroupSnapshot()
	token := &Token{GroupMode: TokenGroupModeExplicit, GroupIds: []int{exclusiveGroup.Id, defaultGroup.Id}}
	if err := ValidateTokenExclusiveGroupBindingCached(token); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("快照应识别独立分组冲突，实际: %v", err)
	}

	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", false).Error; err != nil {
		t.Fatalf("取消独立分组失败: %v", err)
	}
	if err := ValidateTokenExclusiveGroupBindingCached(token); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("未失效的快照应避免重复查库，实际: %v", err)
	}
	InvalidateExclusiveGroupSnapshot()
	if err := ValidateTokenExclusiveGroupBindingCached(token); err != nil {
		t.Fatalf("快照失效后应读取最新独立属性: %v", err)
	}
}

func TestTokenInsertRejectsExclusiveGroupConflict(t *testing.T) {
	defaultGroup, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	token := &Token{
		UserId:         1,
		Key:            "exclusive-conflict-insert",
		Name:           "exclusive-conflict",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		GroupMode:      TokenGroupModeExplicit,
		GroupIds:       []int{exclusiveGroup.Id, defaultGroup.Id},
	}
	if err := token.Insert(); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("新建冲突令牌应该失败，实际: %v", err)
	}
	var count int64
	if err := DB.Model(&Token{}).Where("key = ?", token.Key).Count(&count).Error; err != nil {
		t.Fatalf("查询令牌失败: %v", err)
	}
	if count != 0 {
		t.Fatalf("冲突令牌不应写入数据库: %d", count)
	}
}

func TestTokenUpdateRejectsExclusiveGroupConflict(t *testing.T) {
	defaultGroup, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	token := &Token{
		UserId: 1, Key: "exclusive-update", Name: "exclusive-update",
		Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true,
		GroupMode: TokenGroupModeExplicit, GroupIds: []int{exclusiveGroup.Id},
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("创建独立单分组令牌失败: %v", err)
	}
	token.GroupIds = []int{exclusiveGroup.Id, defaultGroup.Id}
	token.GroupDetails = nil
	if err := token.Update(); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("更新令牌为冲突绑定应该失败，实际: %v", err)
	}
	reloaded, err := GetTokenById(token.Id)
	if err != nil {
		t.Fatalf("重读令牌失败: %v", err)
	}
	if len(reloaded.GroupIds) != 1 || reloaded.GroupIds[0] != exclusiveGroup.Id {
		t.Fatalf("更新失败后应保留原独立绑定: %#v", reloaded.GroupIds)
	}
}

func TestSaveGroupConfigPersistsExclusiveAndRejectsAutoMembership(t *testing.T) {
	defaultGroup, vipGroup := setupGroupBindingsTest(t)
	configs := []GroupConfig{
		{Id: defaultGroup.Id, Code: defaultGroup.Code, Name: defaultGroup.Name, Ratio: 1, Status: GroupStatusActive},
		{Id: vipGroup.Id, Code: vipGroup.Code, Name: vipGroup.Name, Ratio: 0.5, Exclusive: true, Status: GroupStatusActive},
	}
	if err := SaveGroupConfig(configs, nil); err != nil {
		t.Fatalf("保存独立分组失败: %v", err)
	}
	var saved Group
	if err := DB.First(&saved, "id = ?", vipGroup.Id).Error; err != nil {
		t.Fatalf("重读独立分组失败: %v", err)
	}
	if !saved.Exclusive || !saved.ToConfig(nil).Exclusive {
		t.Fatal("独立分组字段未持久化或未投影到 API 配置")
	}

	configs[1].AutoEnabled = true
	if err := SaveGroupConfig(configs, nil); err == nil {
		t.Fatal("独立分组不应允许加入自动分组")
	}
}

func TestSaveGroupConfigPreservesOmittedExclusiveAndAllowsExplicitFalse(t *testing.T) {
	defaultGroup, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	conflictToken := &Token{GroupMode: TokenGroupModeExplicit, GroupIds: []int{exclusiveGroup.Id, defaultGroup.Id}}
	InvalidateExclusiveGroupSnapshot()
	if err := ValidateTokenExclusiveGroupBindingCached(conflictToken); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("保存前应识别独立分组冲突，实际: %v", err)
	}
	configs := []GroupConfig{
		{Id: defaultGroup.Id, Code: defaultGroup.Code, Name: defaultGroup.Name, Ratio: 1, Status: GroupStatusActive, ExclusiveOmitted: true},
		{Id: exclusiveGroup.Id, Code: exclusiveGroup.Code, Name: exclusiveGroup.Name, Ratio: 1, Status: GroupStatusActive, ExclusiveOmitted: true},
	}
	if err := SaveGroupConfig(configs, nil); err != nil {
		t.Fatalf("保存旧客户端分组请求失败: %v", err)
	}
	var saved Group
	if err := DB.First(&saved, "id = ?", exclusiveGroup.Id).Error; err != nil {
		t.Fatalf("重读独立分组失败: %v", err)
	}
	if !saved.Exclusive {
		t.Fatal("旧客户端缺失 exclusive 时不应取消独立属性")
	}
	if err := ValidateTokenExclusiveGroupBindingCached(conflictToken); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("保留独立属性后快照仍应拦截冲突，实际: %v", err)
	}

	configs[1].ExclusiveOmitted = false
	configs[1].Exclusive = false
	if err := SaveGroupConfig(configs, nil); err != nil {
		t.Fatalf("明确取消独立属性失败: %v", err)
	}
	if err := DB.First(&saved, "id = ?", exclusiveGroup.Id).Error; err != nil {
		t.Fatalf("重读取消独立后的分组失败: %v", err)
	}
	if saved.Exclusive {
		t.Fatal("明确传入 false 时应取消独立属性")
	}
	if err := ValidateTokenExclusiveGroupBindingCached(conflictToken); err != nil {
		t.Fatalf("取消独立属性后快照应立即刷新: %v", err)
	}
}

func TestUpdateAutoGroupsRejectsExclusiveGroup(t *testing.T) {
	_, exclusiveGroup := setupGroupBindingsTest(t)
	if err := DB.Model(&Group{}).Where("id = ?", exclusiveGroup.Id).Update("exclusive", true).Error; err != nil {
		t.Fatalf("设置独立分组失败: %v", err)
	}
	value := `["` + exclusiveGroup.Code + `"]`
	if err := UpdateOption("AutoGroups", value); err == nil {
		t.Fatal("通用选项更新不应允许独立分组加入 AutoGroups")
	}
	if err := UpdateOptionsBulk(map[string]string{"AutoGroups": value}); err == nil {
		t.Fatal("批量选项更新不应允许独立分组加入 AutoGroups")
	}
}

func TestTokenGroupMigrationRejectsExclusiveTargetConflict(t *testing.T) {
	defaultGroup, sourceGroup := setupGroupBindingsTest(t)
	exclusiveTarget := &Group{Code: "hack-exclusive", Name: "Hack", Ratio: 1, Exclusive: true, Status: GroupStatusActive}
	if err := DB.Create(exclusiveTarget).Error; err != nil {
		t.Fatalf("创建独立目标分组失败: %v", err)
	}
	token := &Token{
		UserId: 1, Key: "exclusive-migration", Name: "exclusive-migration",
		Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true,
		GroupMode: TokenGroupModeExplicit, GroupIds: []int{sourceGroup.Id, defaultGroup.Id},
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("创建待迁移令牌失败: %v", err)
	}
	if _, err := PreviewTokenGroupMigration(sourceGroup.Id, exclusiveTarget.Id); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("预览应提前发现独立目标分组冲突，实际: %v", err)
	}
	if _, err := MigrateTokenGroup(sourceGroup.Id, exclusiveTarget.Id); !errors.Is(err, ErrTokenGroupBindingConflict) {
		t.Fatalf("迁移到独立分组后形成多组冲突应该失败，实际: %v", err)
	}
	reloaded, err := GetTokenById(token.Id)
	if err != nil {
		t.Fatalf("重读迁移失败后的令牌失败: %v", err)
	}
	if reloaded.Group != sourceGroup.Code+","+defaultGroup.Code {
		t.Fatalf("迁移失败应回滚原绑定，实际: %q", reloaded.Group)
	}
}

func TestTokenGroupMigrationPreservesHistoricalExclusiveConflict(t *testing.T) {
	for _, targetAlreadyBound := range []bool{false, true} {
		name := "replace"
		if targetAlreadyBound {
			name = "deduplicate"
		}
		t.Run(name, func(t *testing.T) {
			_, source := setupGroupBindingsTest(t)
			target := &Group{Code: "value", Name: "Value", Ratio: 1, Status: GroupStatusActive}
			hack := &Group{Code: "hack", Name: "Hack", Ratio: 1, Status: GroupStatusActive}
			require.NoError(t, DB.Create(target).Error)
			require.NoError(t, DB.Create(hack).Error)

			ids := []int{source.Id, hack.Id}
			limits := `{"vip":0.2,"hack":0.4}`
			if targetAlreadyBound {
				ids = []int{source.Id, target.Id, hack.Id}
				limits = `{"vip":0.2,"value":0.3,"hack":0.4}`
			}
			token := &Token{
				UserId: 1, Key: "historical-exclusive-" + name, Name: name,
				Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true,
				GroupMode: TokenGroupModeExplicit, GroupIds: ids, GroupRatioLimits: limits,
			}
			require.NoError(t, token.Insert())
			require.NoError(t, DB.Model(&Group{}).Where("id = ?", hack.Id).Update("exclusive", true).Error)

			preview, err := PreviewTokenGroupMigration(source.Id, target.Id)
			require.NoError(t, err)
			assert.Equal(t, 1, preview.MigratedTokens)
			if targetAlreadyBound {
				assert.Equal(t, 1, preview.DeduplicatedTokens)
			}
			result, err := MigrateTokenGroup(source.Id, target.Id)
			require.NoError(t, err)
			assert.Equal(t, preview.DeduplicatedTokens, result.DeduplicatedTokens)

			reloaded, err := GetTokenById(token.Id)
			require.NoError(t, err)
			assert.Equal(t, []int{target.Id, hack.Id}, reloaded.GroupIds)
			assert.Equal(t, "value,hack", reloaded.Group)
			assert.Equal(t, 0.4, reloaded.GetGroupRatioLimitsMap()[hack.Code])
			expectedTargetLimit := 0.2
			if targetAlreadyBound {
				expectedTargetLimit = 0.3
			}
			assert.Equal(t, expectedTargetLimit, reloaded.GetGroupRatioLimitsMap()[target.Code])
			assert.ErrorIs(t, ValidateTokenExclusiveGroupBinding(DB, reloaded), ErrTokenGroupBindingConflict)
		})
	}
}

func TestTokenGroupMigrationRejectsNewExclusiveConflictOnHistoricalConflict(t *testing.T) {
	defaultGroup, source := setupGroupBindingsTest(t)
	hack := &Group{Code: "hack", Name: "Hack", Ratio: 1, Status: GroupStatusActive}
	target := &Group{Code: "value-exclusive", Name: "Value", Ratio: 1, Status: GroupStatusActive, Exclusive: true}
	require.NoError(t, DB.Create(hack).Error)
	require.NoError(t, DB.Create(target).Error)
	token := &Token{
		UserId: 1, Key: "historical-exclusive-new-conflict", Name: "new-conflict",
		Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true,
		GroupMode: TokenGroupModeExplicit, GroupIds: []int{source.Id, defaultGroup.Id, hack.Id},
	}
	require.NoError(t, token.Insert())
	require.NoError(t, DB.Model(&Group{}).Where("id = ?", hack.Id).Update("exclusive", true).Error)

	_, err := PreviewTokenGroupMigration(source.Id, target.Id)
	require.ErrorIs(t, err, ErrTokenGroupBindingConflict)
	assert.Contains(t, err.Error(), fmt.Sprintf("令牌 %d", token.Id))
	_, err = MigrateTokenGroup(source.Id, target.Id)
	require.ErrorIs(t, err, ErrTokenGroupBindingConflict)

	reloaded, err := GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Equal(t, []int{source.Id, defaultGroup.Id, hack.Id}, reloaded.GroupIds)
}

func TestTokenGroupMigrationDeduplicatesExistingExclusiveTarget(t *testing.T) {
	defaultGroup, source := setupGroupBindingsTest(t)
	target := &Group{Code: "exclusive-target", Name: "独立目标", Ratio: 1, Status: GroupStatusActive}
	require.NoError(t, DB.Create(target).Error)
	token := &Token{
		UserId: 1, Key: "historical-exclusive-target", Name: "historical-exclusive-target",
		Status: common.TokenStatusEnabled, ExpiredTime: -1, UnlimitedQuota: true,
		GroupMode: TokenGroupModeExplicit, GroupIds: []int{source.Id, target.Id, defaultGroup.Id},
		GroupRatioLimits: `{"vip":0.2,"exclusive-target":0.3}`,
	}
	require.NoError(t, token.Insert())
	require.NoError(t, DB.Model(&Group{}).Where("id = ?", target.Id).Update("exclusive", true).Error)

	preview, err := PreviewTokenGroupMigration(source.Id, target.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, preview.DeduplicatedTokens)
	_, err = MigrateTokenGroup(source.Id, target.Id)
	require.NoError(t, err)

	reloaded, err := GetTokenById(token.Id)
	require.NoError(t, err)
	assert.Equal(t, []int{target.Id, defaultGroup.Id}, reloaded.GroupIds)
	assert.Equal(t, 0.3, reloaded.GetGroupRatioLimitsMap()[target.Code])
	assert.ErrorIs(t, ValidateTokenExclusiveGroupBinding(DB, reloaded), ErrTokenGroupBindingConflict)
}
