package controller

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLogGroupOptionsPreserveResponseAndScope(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Group{}, &model.GroupAlias{}, &model.AutoGroupMember{}, &model.GroupTag{}, &model.GroupTagBinding{}, &model.User{}))
	allowed := model.Group{Code: "public", Name: "当前显示名", Ratio: 1, Status: model.GroupStatusActive}
	hidden := model.Group{Code: "private", Name: "隐藏显示名", Ratio: 1, Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&allowed).Error)
	require.NoError(t, db.Create(&hidden).Error)
	require.NoError(t, db.Create(&[]model.GroupAlias{
		{Alias: "old-public", GroupId: allowed.Id}, {Alias: "private", GroupId: allowed.Id}, {Alias: "old-private", GroupId: hidden.Id},
	}).Error)
	require.NoError(t, db.Create(&model.Option{Key: "UserUsableGroups", Value: `{"fake-public":"当前显示名"}`}).Error)
	user := model.User{Username: "log-options-user", Password: "test-password", Group: allowed.Code}
	require.NoError(t, db.Create(&user).Error)
	oldRatios, oldUsable := ratio_setting.GroupRatio2JSONString(), setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsable))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"public":1,"private":2}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"public":"用户可用"}`))
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	var response struct {
		Success bool `json:"success"`
		Data    map[string]struct {
			Id   int    `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
		Names  map[string]string `json:"group_names"`
		Tags   []model.GroupTag  `json:"group_tags"`
		Canvas string            `json:"canvas_default_group"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Len(t, response.Data, 1)
	assert.Equal(t, allowed.Id, response.Data[allowed.Code].Id)
	assert.Equal(t, allowed.Code, response.Data[allowed.Code].Code)
	assert.Equal(t, allowed.Name, response.Data[allowed.Code].Name)
	assert.Equal(t, map[string]string{"public": allowed.Name, "old-public": allowed.Name}, response.Names)
	assert.NotNil(t, response.Tags)

	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/group/details", nil, user.Id)
	GetGroupDetails(ctx)
	var adminResponse struct {
		Success   bool                   `json:"success"`
		Data      []model.Group          `json:"data"`
		Names     map[string]string      `json:"group_names"`
		AutoGroup map[string]interface{} `json:"auto_group"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &adminResponse))
	require.True(t, adminResponse.Success)
	assert.Len(t, adminResponse.Data, 2)
	assert.Equal(t, hidden.Name, adminResponse.Names["private"])
	assert.Equal(t, hidden.Name, adminResponse.Names["old-private"])
	assert.Equal(t, allowed.Name, adminResponse.Names["fake-public"])
	assert.NotNil(t, adminResponse.AutoGroup)
}

func TestLogGroupOptionsMappingFailureDoesNotBreakOriginalData(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.AutoGroupMember{}, &model.GroupAlias{}, &model.User{}, &model.GroupTag{}, &model.GroupTagBinding{}))
	group := model.Group{Code: "legacy", Name: "原有名称", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	user := model.User{Username: "mapping-failure-user", Group: group.Code}
	require.NoError(t, db.Create(&user).Error)
	oldRatios, oldUsable := ratio_setting.GroupRatio2JSONString(), setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsable))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"legacy":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"legacy":"用户可用"}`))
	// 仅让名称映射读取失败，原有分组列表仍可正常读取。
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("test:fail-group-name-map", func(tx *gorm.DB) {
		if tx.Statement.Table == "group_aliases" {
			tx.AddError(errors.New("模拟映射不可用"))
		}
	}))
	t.Cleanup(func() { db.Callback().Query().Remove("test:fail-group-name-map") })
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/group/details", nil, 1)
	GetGroupDetails(ctx)
	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	assert.NotEmpty(t, response["data"])
	assert.NotContains(t, response, "group_names")

	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, true, response["success"])
	assert.NotEmpty(t, response["data"])
	assert.NotContains(t, response, "group_names")
	// 映射恢复后同一个接口重新返回该字段，前端重试不需要改变原选组请求。
	require.NoError(t, db.Callback().Query().Remove("test:fail-group-name-map"))
	ctx, recorder = newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, map[string]interface{}{"legacy": group.Name}, response["group_names"])
}
