package controller

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenGroupChannelAdminPersistenceAndPermissions(t *testing.T) {
	for _, role := range []int{common.RoleCommonUser, common.RoleAdminUser, common.RoleRootUser} {
		t.Run(fmt.Sprint(role), func(t *testing.T) {
			user := setupTokenAutoGroupsControllerTest(t)
			require.NoError(t, model.DB.Model(user).Update("role", role).Error)
			require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}, &model.ChannelGroupBinding{}, &model.Channel{}, &model.Ability{}))
			group := model.Group{Code: "test-code", Name: "测试显示名", Status: model.GroupStatusActive, Ratio: 1}
			require.NoError(t, model.DB.Create(&group).Error)
			channel := model.Channel{Name: "target", Key: "local-test", Status: common.ChannelStatusEnabled, GroupIds: []int{group.Id}, Models: "test-model"}
			require.NoError(t, channel.Insert())
			request := map[string]any{"name": "test-pin", "unlimited_quota": true, "expired_time": -1, "group_mode": "explicit", "group_ids": []int{group.Id}, "group_channel_ids": map[int]int{group.Id: channel.Id}}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			ctx.Set("role", role)
			AddToken(ctx)
			response := decodeAPIResponse(t, recorder)
			if role < common.RoleAdminUser {
				assert.False(t, response.Success, "普通用户不得设置分组限定渠道")
				return
			}
			require.True(t, response.Success, response.Message)
			var token model.Token
			require.NoError(t, model.DB.Where("name = ?", "test-pin").First(&token).Error)
			reloaded, err := model.GetTokenByIds(token.Id, user.Id)
			require.NoError(t, err)
			data, err := common.Marshal(buildMaskedTokenResponse(reloaded))
			require.NoError(t, err)
			var payload struct {
				GroupChannelIDs map[int]int `json:"group_channel_ids"`
			}
			require.NoError(t, common.Unmarshal(data, &payload))
			assert.Equal(t, map[int]int{group.Id: channel.Id}, payload.GroupChannelIDs)
			request["id"] = token.Id
			delete(request, "group_channel_ids")
			ctx, recorder = newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(ctx)
			require.True(t, decodeAPIResponse(t, recorder).Success)
			reloaded, err = model.GetTokenByIds(token.Id, user.Id)
			require.NoError(t, err)
			assert.Equal(t, map[int]int{group.Id: channel.Id}, reloaded.GroupChannelIds, "旧客户端省略字段保留限定")
			require.NoError(t, model.DB.Model(&group).Update("status", model.GroupStatusDisabled).Error)
			request["group_channel_ids"] = map[int]int{}
			ctx, recorder = newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(ctx)
			require.True(t, decodeAPIResponse(t, recorder).Success)
			reloaded, err = model.GetTokenByIds(token.Id, user.Id)
			require.NoError(t, err)
			assert.Empty(t, reloaded.GroupChannelIds, "已绑定分组停用后仍可用空对象清空限定")
		})
	}
}

func TestTokenGroupChannelRejectsInvalidConfigurationWithoutSaving(t *testing.T) {
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.Model(user).Update("role", common.RoleAdminUser).Error)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}, &model.ChannelGroupBinding{}, &model.Channel{}, &model.Ability{}))
	group := model.Group{Code: "test-code", Name: "测试显示名", Status: model.GroupStatusActive, Ratio: 1}
	other := model.Group{Code: "other-code", Name: "其他显示名", Status: model.GroupStatusActive, Ratio: 1}
	require.NoError(t, model.DB.Create(&group).Error)
	require.NoError(t, model.DB.Create(&other).Error)
	disabled := model.Channel{Name: "disabled", Key: "local-test", Status: common.ChannelStatusManuallyDisabled, GroupIds: []int{group.Id}, Models: "test-model"}
	wrongGroup := model.Channel{Name: "other", Key: "local-test", Status: common.ChannelStatusEnabled, GroupIds: []int{other.Id}, Models: "test-model"}
	require.NoError(t, disabled.Insert())
	require.NoError(t, wrongGroup.Insert())
	for _, test := range []struct {
		name     string
		mode     string
		channels map[int]any
	}{
		{"zero", "explicit", map[int]any{group.Id: 0}},
		{"negative", "explicit", map[int]any{group.Id: -1}},
		{"overflow", "explicit", map[int]any{group.Id: int64(2147483648)}},
		{"missing", "explicit", map[int]any{group.Id: 99999}},
		{"disabled", "explicit", map[int]any{group.Id: disabled.Id}},
		{"wrong group", "explicit", map[int]any{group.Id: wrongGroup.Id}},
		{"unselected group", "explicit", map[int]any{other.Id: wrongGroup.Id}},
		{"auto", "auto", map[int]any{other.Id: wrongGroup.Id}},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := map[string]any{"name": test.name, "unlimited_quota": true, "expired_time": -1, "group_mode": test.mode, "group_ids": []int{group.Id}, "group_channel_ids": test.channels}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			AddToken(ctx)
			assert.False(t, decodeAPIResponse(t, recorder).Success)
			var count int64
			require.NoError(t, model.DB.Model(&model.Token{}).Count(&count).Error)
			assert.Zero(t, count, "非法配置不得留下令牌")
		})
	}
}

func TestTokenGroupChannelDowngradedUserCannotClearRestrictionsByOmission(t *testing.T) {
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}))
	group := model.Group{Code: "test-code", Name: "测试显示名", Status: model.GroupStatusActive, Ratio: 1}
	require.NoError(t, model.DB.Create(&group).Error)
	token := model.Token{UserId: user.Id, Key: "restricted-token", Name: "restricted", GroupMode: model.TokenGroupModeExplicit, GroupIds: []int{group.Id}, GroupChannelIds: map[int]int{group.Id: 952}, UnlimitedQuota: true, ExpiredTime: -1}
	require.NoError(t, token.Insert())
	for _, mode := range []string{"auto", "inherit", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			request := map[string]any{"id": token.Id, "name": "changed", "unlimited_quota": true, "group_mode": mode}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(ctx)
			assert.Equal(t, http.StatusForbidden, recorder.Code)
			loaded, err := model.GetTokenByIds(token.Id, user.Id)
			require.NoError(t, err)
			assert.Equal(t, map[int]int{group.Id: 952}, loaded.GroupChannelIds)
		})
	}
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/?status_only=true", map[string]any{"id": token.Id, "status": common.TokenStatusDisabled}, user.Id)
	UpdateToken(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)
	loaded, err := model.GetTokenByIds(token.Id, user.Id)
	require.NoError(t, err)
	assert.Equal(t, common.TokenStatusDisabled, loaded.Status)
	assert.Equal(t, map[int]int{group.Id: 952}, loaded.GroupChannelIds)
}
