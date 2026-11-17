package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGroupValidationBindingNames(t *testing.T) {
	for _, entity := range []string{"channel", "token"} {
		t.Run(entity, func(t *testing.T) {
			_, group := setupGroupBindingsTest(t)
			const alias = "legacy-premium"
			require.NoError(t, DB.Create(&GroupAlias{Alias: alias, GroupId: group.Id}).Error)
			require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("status", GroupStatusDisabled).Error)

			for _, name := range []string{"当前贵宾组", "改名后的贵宾组", ""} {
				require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("name", name).Error)
				for _, input := range []struct {
					label string
					ids   []int
					code  string
				}{
					{label: "id", ids: []int{group.Id}},
					{label: "code", code: group.Code},
					{label: "alias", code: alias},
				} {
					t.Run(name+"/"+input.label, func(t *testing.T) {
						var err error
						if entity == "channel" {
							err = PrepareChannelGroupBindings(DB, &Channel{GroupIds: input.ids, Group: input.code})
						} else {
							err = PrepareTokenGroupBindings(DB, &Token{GroupIds: input.ids, Group: input.code})
						}
						wantName := name
						if wantName == "" {
							wantName = input.code
						}
						want := fmt.Sprintf("分组 %q 已禁用", wantName)
						if name == "" && input.ids != nil {
							want = fmt.Sprintf("分组 ID %d 不存在或已禁用", group.Id)
						}
						require.EqualError(t, err, want)
					})
				}
			}
		})
	}
}

func TestGroupValidationBindingIdentityAndUnknownFallback(t *testing.T) {
	_, group := setupGroupBindingsTest(t)
	const alias = "legacy-premium"
	require.NoError(t, DB.Create(&GroupAlias{Alias: alias, GroupId: group.Id}).Error)
	require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("name", "当前贵宾组").Error)

	channel := &Channel{Group: alias + "," + group.Code}
	require.NoError(t, PrepareChannelGroupBindings(DB, channel))
	assert.Equal(t, group.Code, channel.Group)
	assert.Equal(t, []int{group.Id}, channel.GroupIds)
	require.Len(t, channel.GroupDetails, 1)
	assert.Equal(t, "当前贵宾组", channel.GroupDetails[0].Name)

	token := &Token{Group: alias + "," + group.Code}
	require.NoError(t, PrepareTokenGroupBindings(DB, token))
	assert.Equal(t, group.Code, token.Group)
	assert.Equal(t, []int{group.Id}, token.GroupIds)
	assert.Equal(t, TokenGroupModeExplicit, token.GroupMode)
	require.Len(t, token.GroupDetails, 1)
	assert.Equal(t, "当前贵宾组", token.GroupDetails[0].Name)

	for _, entity := range []string{"channel", "token"} {
		for _, code := range []string{"unknown-group", "all"} {
			t.Run(entity+"/"+code, func(t *testing.T) {
				var err error
				if entity == "channel" {
					err = PrepareChannelGroupBindings(DB, &Channel{Group: code})
				} else {
					err = PrepareTokenGroupBindings(DB, &Token{Group: code})
				}
				require.Error(t, err)
				assert.Contains(t, err.Error(), code)
				if code == "unknown-group" {
					assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
				} else {
					assert.Contains(t, err.Error(), "保留值")
				}
			})
		}
	}
	require.EqualError(t, PrepareChannelGroupBindings(DB, &Channel{GroupIds: []int{999}}), "分组 ID 999 不存在")
	require.EqualError(t, PrepareTokenGroupBindings(DB, &Token{GroupIds: []int{999}}), "分组 ID 999 不存在")
}

func TestGroupValidationConfigParameterNames(t *testing.T) {
	for _, input := range []struct {
		label       string
		ratio       float64
		concurrency int
		exclusive   bool
		auto        bool
		want        string
	}{
		{label: "negative_ratio", ratio: -1, want: "分组 %s 的倍率不能小于 0"},
		{label: "negative_concurrency", ratio: 1, concurrency: -1, want: "分组 %s 的单用户并发上限不能小于 0"},
		{label: "exclusive_auto", ratio: 1, exclusive: true, auto: true, want: "独立分组 %s 不能加入自动分组"},
	} {
		t.Run(input.label, func(t *testing.T) {
			_, group := setupGroupBindingsTest(t)
			for _, name := range []string{"  当前贵宾组  ", ""} {
				t.Run(name, func(t *testing.T) {
					config := GroupConfig{
						Id: group.Id, Code: group.Code, Name: name, Ratio: input.ratio,
						SingleUserConcurrencyLimit: input.concurrency, Status: GroupStatusActive,
						Exclusive: input.exclusive, AutoEnabled: input.auto,
					}
					wantName := "当前贵宾组"
					if name == "" {
						wantName = group.Code
					}
					result, err := SaveGroupConfigWithOptionsAndResult([]GroupConfig{config}, nil, nil)
					require.EqualError(t, err, fmt.Sprintf(input.want, wantName))
					assert.Nil(t, result)
					var stored Group
					require.NoError(t, DB.First(&stored, group.Id).Error)
					assert.Equal(t, group.Code, stored.Code)
					assert.Equal(t, group.Name, stored.Name)
					assert.Equal(t, group.Ratio, stored.Ratio)
					assert.Equal(t, group.SingleUserConcurrencyLimit, stored.SingleUserConcurrencyLimit)
					assert.Equal(t, group.Exclusive, stored.Exclusive)
				})
			}
		})
	}
}

func TestGroupValidationAutoOptionNames(t *testing.T) {
	_, group := setupGroupBindingsTest(t)
	const alias = "legacy-premium"
	require.NoError(t, DB.Create(&GroupAlias{Alias: alias, GroupId: group.Id}).Error)
	require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("exclusive", true).Error)
	require.NoError(t, DB.Create(&Option{Key: "AutoGroups", Value: `[]`}).Error)

	for _, name := range []string{"当前独立组", "改名后的独立组", ""} {
		require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("name", name).Error)
		for _, identifier := range []string{group.Code, alias} {
			t.Run(name+"/"+identifier, func(t *testing.T) {
				value, err := common.Marshal([]string{identifier})
				require.NoError(t, err)
				wantName := name
				if wantName == "" {
					wantName = group.Code
				}
				require.EqualError(t, UpdateOption("AutoGroups", string(value)), fmt.Sprintf("独立分组 %s 不能加入自动分组", wantName))
				var stored Option
				require.NoError(t, DB.First(&stored, Option{Key: "AutoGroups"}).Error)
				assert.Equal(t, `[]`, stored.Value)
			})
		}
	}
	value, err := common.Marshal([]string{"unknown-group"})
	require.NoError(t, err)
	err = UpdateOption("AutoGroups", string(value))
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	assert.Contains(t, err.Error(), "unknown-group")
}

func TestGroupValidationMigrationRatioNames(t *testing.T) {
	for _, hasBindingLimit := range []bool{false, true} {
		t.Run(fmt.Sprintf("binding_limit_%t", hasBindingLimit), func(t *testing.T) {
			target, source := setupGroupBindingsTest(t)
			const alias = "legacy-premium"
			require.NoError(t, DB.Create(&GroupAlias{Alias: alias, GroupId: source.Id}).Error)
			token := &Token{UserId: 1, Key: "validation-migration", Name: "倍率镜像回归",
				GroupMode: TokenGroupModeExplicit, GroupIds: []int{source.Id}, UnlimitedQuota: true}
			require.NoError(t, token.Insert())
			rawLimits, err := common.Marshal(map[string]float64{alias: 2})
			require.NoError(t, err)
			require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).
				Updates(&Token{Group: alias, GroupRatioLimits: string(rawLimits)}).Error)
			if hasBindingLimit {
				require.NoError(t, DB.Model(&TokenGroupBinding{}).Where("token_id = ?", token.Id).Update("ratio_limit", 1).Error)
			}
			for _, name := range []string{"当前贵宾组", "改名后的贵宾组", ""} {
				require.NoError(t, DB.Model(&Group{}).Where("id = ?", source.Id).Update("name", name).Error)
				wantName := name
				if wantName == "" {
					wantName = source.Code
				}
				want := fmt.Sprintf("令牌 %d 分组 %s 的稳定倍率保护与旧镜像不一致，请先修复数据后再迁移", token.Id, wantName)
				_, err = PreviewTokenGroupMigration(source.Id, target.Id)
				require.EqualError(t, err, want)
				_, err = MigrateTokenGroup(source.Id, target.Id)
				require.EqualError(t, err, want)
				var stored Token
				require.NoError(t, DB.First(&stored, token.Id).Error)
				assert.Equal(t, alias, stored.Group)
				assert.Equal(t, string(rawLimits), stored.GroupRatioLimits)
				var bindings []TokenGroupBinding
				require.NoError(t, DB.Where("token_id = ?", token.Id).Find(&bindings).Error)
				require.Len(t, bindings, 1)
				assert.Equal(t, source.Id, bindings[0].GroupId)
				if hasBindingLimit {
					require.NotNil(t, bindings[0].RatioLimit)
					assert.Equal(t, float64(1), *bindings[0].RatioLimit)
				} else {
					assert.Nil(t, bindings[0].RatioLimit)
				}
			}
		})
	}
}
