package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureTokenAutoGroupsTest(t *testing.T, maxCount string, autoGroups string) {
	t.Helper()
	originalMax := setting.GetMaxTokenAutoGroups()
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups(maxCount))
	require.NoError(t, setting.UpdateAutoGroupsByJsonString(autoGroups))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(stringInt(originalMax)))
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})
}

func stringInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func setupTokenAutoGroupsControllerTest(t *testing.T) *model.User {
	t.Helper()
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	user := &model.User{
		Id:       101,
		Username: "token-auto-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func baseAutoTokenRequest(name string) map[string]any {
	return map[string]any{
		"name":              name,
		"expired_time":      -1,
		"remain_quota":      0,
		"unlimited_quota":   true,
		"group":             "auto",
		"cross_group_retry": true,
	}
}

func newTokenAutoGroupsAuthenticatedContext(t *testing.T, method string, target string, body any, userID int) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	ctx, recorder := newAuthenticatedContext(t, method, target, body, userID)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	return ctx, recorder
}

func TestAddTokenEmptyAutoGroupsInheritGlobalAuto(t *testing.T) {
	tests := []struct {
		name         string
		includeField bool
		value        any
	}{
		{name: "omitted"},
		{name: "null", includeField: true, value: nil},
		{name: "empty array", includeField: true, value: []string{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
			user := setupTokenAutoGroupsControllerTest(t)
			request := baseAutoTokenRequest("create-" + test.name)
			if test.includeField {
				request["auto_groups"] = test.value
			}

			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			AddToken(ctx)

			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)
			var token model.Token
			require.NoError(t, model.DB.Where("name = ?", request["name"]).First(&token).Error)
			assert.Empty(t, token.AutoGroups)
			assert.True(t, token.CrossGroupRetry)
			payload, err := common.Marshal(buildMaskedTokenResponse(&token))
			require.NoError(t, err)
			var responseData map[string]any
			require.NoError(t, common.Unmarshal(payload, &responseData))
			assert.Nil(t, responseData["auto_groups"])
		})
	}
}

func TestAddTokenPersistsOrderedAutoGroupsSnapshot(t *testing.T) {
	configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
	user := setupTokenAutoGroupsControllerTest(t)
	request := baseAutoTokenRequest("ordered-snapshot")
	request["auto_groups"] = []string{"vip", "default"}

	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
	AddToken(ctx)
	require.True(t, decodeAPIResponse(t, recorder).Success)

	var token model.Token
	require.NoError(t, model.DB.Where("name = ?", "ordered-snapshot").First(&token).Error)
	assert.JSONEq(t, `["vip","default"]`, token.AutoGroups)

	getCtx, getRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/"+stringInt(token.Id), nil, user.Id)
	getCtx.Params = append(getCtx.Params, gin.Param{Key: "id", Value: stringInt(token.Id)})
	GetToken(getCtx)
	getResponse := decodeAPIResponse(t, getRecorder)
	require.True(t, getResponse.Success)
	var data struct {
		AutoGroups []string `json:"auto_groups"`
	}
	require.NoError(t, common.Unmarshal(getResponse.Data, &data))
	assert.Equal(t, []string{"vip", "default"}, data.AutoGroups)
}

func TestUpdateTokenAutoGroupsTriStateAndNonAutoCleanup(t *testing.T) {
	tests := []struct {
		name               string
		includeField       bool
		value              any
		group              string
		expectedAutoGroups string
		expectedRetry      bool
	}{
		{name: "omitted preserves", group: "auto", expectedAutoGroups: `["vip","default"]`, expectedRetry: true},
		{name: "null inherits", includeField: true, value: nil, group: "auto", expectedRetry: true},
		{name: "empty inherits", includeField: true, value: []string{}, group: "auto", expectedRetry: true},
		{name: "non auto clears and disables retry", includeField: true, value: []string{"vip"}, group: "default"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, "5", `["default","vip"]`)
			user := setupTokenAutoGroupsControllerTest(t)
			token := seedToken(t, model.DB, user.Id, "update-auto", "update-auto-key")
			token.Group = "auto"
			token.CrossGroupRetry = true
			require.NoError(t, token.SetAutoGroups([]string{"vip", "default"}))
			require.NoError(t, model.DB.Save(token).Error)

			request := baseAutoTokenRequest("updated-auto")
			request["id"] = token.Id
			request["status"] = common.TokenStatusEnabled
			request["group"] = test.group
			if test.includeField {
				request["auto_groups"] = test.value
			}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(ctx)
			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)

			var updated model.Token
			require.NoError(t, model.DB.First(&updated, token.Id).Error)
			if test.expectedAutoGroups == "" {
				assert.Empty(t, updated.AutoGroups)
			} else {
				assert.JSONEq(t, test.expectedAutoGroups, updated.AutoGroups)
			}
			assert.Equal(t, test.expectedRetry, updated.CrossGroupRetry)
		})
	}
}

func TestAddTokenRejectsInvalidAutoGroups(t *testing.T) {
	tests := []struct {
		name     string
		maxCount string
		groups   []string
	}{
		{name: "over limit", maxCount: "1", groups: []string{"default", "vip"}},
		{name: "duplicate", maxCount: "5", groups: []string{"default", "default"}},
		{name: "auto pseudo group", maxCount: "5", groups: []string{"auto"}},
		{name: "unavailable", maxCount: "5", groups: []string{"missing"}},
		{name: "unlimited duplicate", maxCount: "0", groups: []string{"default", "default"}},
		{name: "unlimited auto pseudo group", maxCount: "0", groups: []string{"auto"}},
		{name: "unlimited unavailable", maxCount: "0", groups: []string{"missing"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, test.maxCount, `["default","vip"]`)
			user := setupTokenAutoGroupsControllerTest(t)
			request := baseAutoTokenRequest("invalid-" + test.name)
			request["auto_groups"] = test.groups

			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			AddToken(ctx)

			response := decodeAPIResponse(t, recorder)
			assert.False(t, response.Success)
			var count int64
			require.NoError(t, model.DB.Model(&model.Token{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestAddAndUpdateTokenRejectExplicitGroupsOverLimit(t *testing.T) {
	require.NoError(t, i18n.Init())
	configureTokenAutoGroupsTest(t, "2", `[]`)
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}))
	for _, group := range []model.Group{
		{Code: "default", Name: "标准显示名", Status: model.GroupStatusActive, Ratio: 1},
		{Code: "vip", Name: "高级显示名", Status: model.GroupStatusActive, Ratio: 1},
		{Code: "svip", Name: "旗舰显示名", Status: model.GroupStatusActive, Ratio: 1},
	} {
		require.NoError(t, model.DB.Create(&group).Error)
	}

	overCSV := map[string]any{
		"name":            "too-many-groups",
		"expired_time":    -1,
		"remain_quota":    0,
		"unlimited_quota": true,
		"group":           "default,vip,svip",
		"group_mode":      model.TokenGroupModeExplicit,
	}
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", overCSV, user.Id)
	ctx.Request.Header.Set("Accept-Language", "zh-CN")
	AddToken(ctx)
	response := decodeAPIResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, "令牌最多可选 2 个分组", response.Message)
	assert.NotContains(t, response.Message, "无效的令牌")
	var count int64
	require.NoError(t, model.DB.Model(&model.Token{}).Count(&count).Error)
	assert.Zero(t, count)

	var stored []model.Group
	require.NoError(t, model.DB.Order("id asc").Find(&stored).Error)
	require.Len(t, stored, 3)
	overIDs := map[string]any{
		"name":            "too-many-ids",
		"expired_time":    -1,
		"remain_quota":    0,
		"unlimited_quota": true,
		"group":           "default",
		"group_ids":       []int{stored[0].Id, stored[1].Id, stored[2].Id},
		"group_mode":      model.TokenGroupModeExplicit,
	}
	idCtx, idRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", overIDs, user.Id)
	idCtx.Request.Header.Set("Accept-Language", "en")
	AddToken(idCtx)
	idResponse := decodeAPIResponse(t, idRecorder)
	assert.False(t, idResponse.Success)
	assert.Equal(t, "A token can select at most 2 groups", idResponse.Message)
	assert.NotContains(t, idResponse.Message, "Invalid token")

	allowed := map[string]any{
		"name":            "at-limit",
		"expired_time":    -1,
		"remain_quota":    0,
		"unlimited_quota": true,
		"group":           "vip,default",
		"group_mode":      model.TokenGroupModeExplicit,
	}
	okCtx, okRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", allowed, user.Id)
	AddToken(okCtx)
	okResponse := decodeAPIResponse(t, okRecorder)
	require.True(t, okResponse.Success, okResponse.Message)
	var created model.Token
	require.NoError(t, model.DB.Where("name = ?", "at-limit").First(&created).Error)
	assert.Equal(t, "vip,default", created.Group)

	updateRequest := map[string]any{
		"id":              created.Id,
		"name":            "at-limit",
		"status":          common.TokenStatusEnabled,
		"expired_time":    -1,
		"remain_quota":    0,
		"unlimited_quota": true,
		"group":           "default,vip,svip",
		"group_mode":      model.TokenGroupModeExplicit,
	}
	updateCtx, updateRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", updateRequest, user.Id)
	updateCtx.Request.Header.Set("Accept-Language", "zh-CN")
	UpdateToken(updateCtx)
	updateResponse := decodeAPIResponse(t, updateRecorder)
	assert.False(t, updateResponse.Success)
	assert.Equal(t, "令牌最多可选 2 个分组", updateResponse.Message)
	var unchanged model.Token
	require.NoError(t, model.DB.First(&unchanged, created.Id).Error)
	assert.Equal(t, "vip,default", unchanged.Group)
}

func TestUpdateTokenStatusOnlyKeepsHistoricalOverLimitGroups(t *testing.T) {
	configureTokenAutoGroupsTest(t, "1", `[]`)
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.TokenGroupBinding{}))
	for _, group := range []model.Group{
		{Code: "default", Name: "标准显示名", Status: model.GroupStatusActive, Ratio: 1},
		{Code: "vip", Name: "高级显示名", Status: model.GroupStatusActive, Ratio: 1},
	} {
		require.NoError(t, model.DB.Create(&group).Error)
	}
	token := seedToken(t, model.DB, user.Id, "historical", "historical-key")
	token.Group = "default,vip"
	token.GroupMode = model.TokenGroupModeExplicit
	require.NoError(t, model.DB.Save(token).Error)

	request := map[string]any{
		"id":     token.Id,
		"name":   token.Name,
		"status": common.TokenStatusDisabled,
	}
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/?status_only=true", request, user.Id)
	UpdateToken(ctx)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var updated model.Token
	require.NoError(t, model.DB.First(&updated, token.Id).Error)
	assert.Equal(t, common.TokenStatusDisabled, updated.Status)
	assert.Equal(t, "default,vip", updated.Group)
}

func TestTokenGroupLimitUsesResolvedGroupIdentity(t *testing.T) {
	require.NoError(t, i18n.Init())
	configureTokenAutoGroupsTest(t, "2", `[]`)
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}))
	for _, code := range []string{"default", "vip", "svip"} {
		group := model.Group{Code: code, Name: "显示名称-" + code, Status: model.GroupStatusActive, Ratio: 1}
		require.NoError(t, model.DB.Create(&group).Error)
		if code == "vip" {
			require.NoError(t, model.DB.Create(&model.GroupAlias{Alias: "legacy-vip", GroupId: group.Id}).Error)
		}
	}

	for _, test := range []struct {
		name    string
		groups  string
		allowed bool
	}{
		{name: "trailing-empty-over-limit", groups: "default,vip,svip,"},
		{name: "duplicate-over-limit", groups: "default,vip,svip,default"},
		{name: "alias-same-identity", groups: "default,vip,legacy-vip", allowed: true},
		{name: "duplicate-at-limit", groups: "default,vip,default,", allowed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := map[string]any{
				"name": test.name, "expired_time": -1, "unlimited_quota": true,
				"group": test.groups, "group_mode": model.TokenGroupModeExplicit,
			}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			ctx.Request.Header.Set("Accept-Language", "zh-CN")
			AddToken(ctx)
			response := decodeAPIResponse(t, recorder)
			require.Equal(t, test.allowed, response.Success, response.Message)
			var count int64
			require.NoError(t, model.DB.Model(&model.Token{}).Where("name = ?", test.name).Count(&count).Error)
			if test.allowed {
				assert.EqualValues(t, 1, count)
			} else {
				assert.Zero(t, count)
				assert.Equal(t, "令牌最多可选 2 个分组", response.Message)
			}

			token := &model.Token{UserId: user.Id, Name: "edit-" + test.name, Key: "key-" + test.name,
				Group: "default", GroupMode: model.TokenGroupModeExplicit, UnlimitedQuota: true, ExpiredTime: -1}
			require.NoError(t, token.Insert())
			request["id"] = token.Id
			request["name"] = token.Name
			updateCtx, updateRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			updateCtx.Request.Header.Set("Accept-Language", "zh-CN")
			UpdateToken(updateCtx)
			updateResponse := decodeAPIResponse(t, updateRecorder)
			require.Equal(t, test.allowed, updateResponse.Success, updateResponse.Message)
			var saved model.Token
			require.NoError(t, model.DB.First(&saved, token.Id).Error)
			if test.allowed {
				assert.Equal(t, "default,vip", saved.Group)
			} else {
				assert.Equal(t, "令牌最多可选 2 个分组", updateResponse.Message)
				assert.Equal(t, "default", saved.Group)
			}
		})
	}
}

func TestGetTokenAutoGroupsReturnsFullFilteredGlobalOrderAndLimit(t *testing.T) {
	for _, test := range []struct {
		name       string
		configured int
		expected   int
	}{
		{name: "default unlimited", configured: setting.DefaultMaxTokenAutoGroups, expected: 0},
		{name: "explicit positive limit", configured: 1, expected: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, stringInt(test.configured), `["vip","missing","default"]`)
			user := setupTokenAutoGroupsControllerTest(t)

			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/auto-groups", nil, user.Id)
			GetTokenAutoGroups(ctx)

			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)
			var data struct {
				Groups   []string `json:"groups"`
				MaxCount int      `json:"max_count"`
			}
			require.NoError(t, common.Unmarshal(response.Data, &data))
			assert.Equal(t, []string{"vip", "default"}, data.Groups)
			assert.Equal(t, test.expected, data.MaxCount)
		})
	}
}

func TestAddAndUpdateTokenUnlimitedPreservesMoreThanFiveGroups(t *testing.T) {
	for _, mode := range []string{"codes", "ids", "auto"} {
		t.Run(mode, func(t *testing.T) {
			configureTokenAutoGroupsTest(t, "0", `[]`)
			require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"标准显示名","vip":"高级显示名","svip":"旗舰显示名","team":"团队显示名","pro":"专业显示名","enterprise":"企业显示名"}`))
			require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1,"team":1,"pro":1,"enterprise":1}`))
			user := setupTokenAutoGroupsControllerTest(t)
			require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.TokenGroupBinding{}))
			codes := []string{"default", "vip", "svip", "team", "pro", "enterprise"}
			ids := make([]int, 0, len(codes))
			for _, code := range codes {
				group := model.Group{Code: code, Name: "显示名称-" + code, Status: model.GroupStatusActive, Ratio: 1}
				require.NoError(t, model.DB.Create(&group).Error)
				ids = append(ids, group.Id)
			}
			request := map[string]any{
				"name": "unlimited-" + mode, "expired_time": -1, "unlimited_quota": true,
				"group": strings.Join(codes, ","), "group_mode": model.TokenGroupModeExplicit,
			}
			if mode == "ids" {
				request["group"] = "default"
				request["group_ids"] = ids
			}
			if mode == "auto" {
				request["group"] = "auto"
				request["group_mode"] = model.TokenGroupModeAuto
				request["auto_groups"] = codes
				request["cross_group_retry"] = true
			}
			ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPost, "/api/token/", request, user.Id)
			AddToken(ctx)
			response := decodeAPIResponse(t, recorder)
			require.True(t, response.Success, response.Message)
			var created model.Token
			require.NoError(t, model.DB.Where("name = ?", request["name"]).First(&created).Error)
			if mode == "auto" {
				groups, err := created.GetAutoGroups()
				require.NoError(t, err)
				assert.Equal(t, codes, groups)
			} else {
				assert.Equal(t, strings.Join(codes, ","), created.Group)
			}

			reversedCodes := []string{"enterprise", "pro", "team", "svip", "vip", "default"}
			request["id"] = created.Id
			request["status"] = common.TokenStatusEnabled
			request["group"] = strings.Join(reversedCodes, ",")
			if mode == "ids" {
				request["group"] = "default"
				request["group_ids"] = []int{ids[5], ids[4], ids[3], ids[2], ids[1], ids[0]}
			}
			if mode == "auto" {
				request["group"] = "auto"
				request["auto_groups"] = reversedCodes
			}
			updateCtx, updateRecorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodPut, "/api/token/", request, user.Id)
			UpdateToken(updateCtx)
			updateResponse := decodeAPIResponse(t, updateRecorder)
			require.True(t, updateResponse.Success, updateResponse.Message)
			var updated model.Token
			require.NoError(t, model.DB.First(&updated, created.Id).Error)
			if mode == "auto" {
				groups, err := updated.GetAutoGroups()
				require.NoError(t, err)
				assert.Equal(t, reversedCodes, groups)
			} else {
				assert.Equal(t, strings.Join(reversedCodes, ","), updated.Group)
			}
		})
	}
}
