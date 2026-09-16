package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelListExplicitTokenGroups(t *testing.T) {
	originalDB, originalLogDB := model.DB, model.LOG_DB
	// Cleanup 按后进先出执行，先注册以便在公共夹具关闭测试库后恢复原连接。
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
	})
	setupRelayRouterTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Channel{}, &model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}, &model.Log{}))
	usableGroups := setting.UserUsableGroups2JSONString()
	groupRatios := ratio_setting.GroupRatio2JSONString()
	modelRatios := ratio_setting.ModelRatio2JSONString()
	maxGroups := setting.GetMaxTokenAutoGroups()
	selfUse := operation_setting.SelfUseModeEnabled
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(usableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(groupRatios))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(modelRatios))
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprint(maxGroups)))
		operation_setting.SelfUseModeEnabled = selfUse
	})
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP","outside":"Outside","auto":"Auto"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"outside":1}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"list-default":1,"list-vip":1,"list-shared":1,"list-disabled":1,"list-outside":1}`))
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("5"))
	operation_setting.SelfUseModeEnabled = false

	groups := []model.Group{
		{Code: "default", Name: "Default", Status: model.GroupStatusActive},
		{Code: "vip", Name: "VIP", Status: model.GroupStatusActive},
		{Code: "outside", Name: "Outside", Status: model.GroupStatusActive},
	}
	require.NoError(t, model.DB.Create(&groups).Error)
	user := model.User{Username: "model-list-groups-user", Status: common.UserStatusEnabled, Group: "default", Quota: 100}
	require.NoError(t, model.DB.Create(&user).Error)
	channel := model.Channel{Type: constant.ChannelTypeOpenAI, Status: common.ChannelStatusEnabled, Name: "model-list-fixture"}
	require.NoError(t, model.DB.Create(&channel).Error)
	require.NoError(t, model.DB.Create(&[]model.Ability{
		{Group: "default", Model: "list-default", ChannelId: channel.Id, Enabled: true},
		{Group: "default", Model: "list-shared", ChannelId: channel.Id, Enabled: true},
		{Group: "vip", Model: "list-vip", ChannelId: channel.Id, Enabled: true},
		{Group: "vip", Model: "list-shared", ChannelId: channel.Id, Enabled: true},
		{Group: "vip", Model: "list-unpriced", ChannelId: channel.Id, Enabled: true},
		{Group: "vip", Model: "list-disabled", ChannelId: channel.Id, Enabled: false},
		{Group: "outside", Model: "list-outside", ChannelId: channel.Id, Enabled: true},
	}).Error)
	engine := gin.New()
	SetRelayRouter(engine)

	tests := []struct {
		name       string
		path       string
		header     string
		groupIDs   []int
		legacy     string
		mode       string
		autoGroups string
		limit      string
		expected   []string
	}{
		{name: "显式多分组模型并集", path: "/v1/models", groupIDs: []int{groups[1].Id, groups[0].Id}, expected: []string{"list-vip", "list-shared", "list-default"}},
		{name: "历史逗号分组", path: "/v1/models", legacy: "vip,default", expected: []string{"list-vip", "list-shared", "list-default"}},
		{name: "多分组模型白名单", path: "/v1/models", groupIDs: []int{groups[1].Id, groups[0].Id}, limit: "list-vip,list-default,list-outside,list-disabled,list-unpriced,list-missing", expected: []string{"list-vip", "list-default"}},
		{name: "Anthropic多分组", path: "/v1/models", header: "x-api-key", groupIDs: []int{groups[1].Id, groups[0].Id}, expected: []string{"list-vip", "list-shared", "list-default"}},
		{name: "Gemini多分组", path: "/v1beta/models", header: "x-goog-api-key", groupIDs: []int{groups[1].Id, groups[0].Id}, expected: []string{"list-vip", "list-shared", "list-default"}},
		{name: "Gemini兼容OpenAI多分组", path: "/v1beta/openai/models", groupIDs: []int{groups[1].Id, groups[0].Id}, expected: []string{"list-vip", "list-shared", "list-default"}},
		{name: "单分组保持隔离", path: "/v1/models", groupIDs: []int{groups[1].Id}, expected: []string{"list-vip", "list-shared"}},
		{name: "继承用户分组", path: "/v1/models", mode: model.TokenGroupModeInherit, expected: []string{"list-default", "list-shared"}},
		{name: "auto指定分组", path: "/v1/models", mode: model.TokenGroupModeAuto, autoGroups: `["vip"]`, expected: []string{"list-vip", "list-shared"}},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := model.Token{UserId: user.Id, Key: fmt.Sprintf("modelslist%d", index), Status: common.TokenStatusEnabled,
				ExpiredTime: -1, UnlimitedQuota: true, GroupMode: test.mode, GroupIds: test.groupIDs,
				Group: test.legacy, AutoGroups: test.autoGroups, ModelLimitsEnabled: test.limit != "", ModelLimits: test.limit}
			if test.legacy != "" {
				// 保留未迁移旧令牌，验证读取时的分组绑定还原。
				require.NoError(t, model.DB.Create(&token).Error)
			} else {
				require.NoError(t, token.Insert())
			}
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			switch test.header {
			case "x-api-key":
				request.Header.Set(test.header, token.Key)
				request.Header.Set("anthropic-version", "2023-06-01")
			case "x-goog-api-key":
				request.Header.Set(test.header, token.Key)
			default:
				request.Header.Set("Authorization", "Bearer sk-"+token.Key)
			}
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var payload struct {
				Data []struct {
					ID string `json:"id"`
				} `json:"data"`
				Models []struct {
					Name string `json:"name"`
				} `json:"models"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
			ids := make([]string, 0, len(payload.Data)+len(payload.Models))
			for _, entry := range payload.Data {
				ids = append(ids, entry.ID)
			}
			for _, entry := range payload.Models {
				ids = append(ids, entry.Name)
			}
			assert.ElementsMatch(t, test.expected, ids, "响应：%s", recorder.Body.String())
		})
	}

	t.Run("多分组撤权仍拒绝", func(t *testing.T) {
		token := model.Token{UserId: user.Id, Key: "modelsrevoked", Status: common.TokenStatusEnabled,
			ExpiredTime: -1, UnlimitedQuota: true, GroupIds: []int{groups[1].Id, groups[0].Id}}
		require.NoError(t, token.Insert())
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default"}`))
		request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		request.Header.Set("Authorization", "Bearer sk-"+token.Key)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusForbidden, recorder.Code)
		assert.Contains(t, recorder.Body.String(), "VIP")
	})
}
