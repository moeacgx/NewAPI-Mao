package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUserGroupNamesDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Cleanup(model.InitDBColumns)
	db := setupManageUserTestDB(t)
	model.InitDBColumns()
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.AffiliateRecord{}, &model.AffiliateBalance{}))
	require.NoError(t, db.Create(&model.Group{Id: 31, Code: "current-code", Name: "当前名称", Status: model.GroupStatusActive}).Error)
	require.NoError(t, db.Create(&model.Group{Id: 32, Code: "other-code", Name: "其他组名称", Status: model.GroupStatusActive}).Error)
	require.NoError(t, db.Create(&model.GroupAlias{GroupId: 31, Alias: "legacy-code"}).Error)
	return db
}

func userGroupResponse(t *testing.T, handler gin.HandlerFunc, target string, id, role int) map[string]interface{} {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, target, nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	c.Set("id", id)
	c.Set("role", role)
	handler(c)
	require.Equal(t, http.StatusOK, rec.Code)
	var response map[string]interface{}
	require.NoError(t, common.Unmarshal(rec.Body.Bytes(), &response))
	return response
}

// 直接写入历史关联，避免模型写入钩子提前规范化待验证的 code/alias。
func TestUserGroupNamesSelfAndAdminContract(t *testing.T) {
	for _, tc := range []struct {
		name, code, want string
		groupID          int
	}{
		{"稳定ID优先于其他组code", "other-code", "当前名称", 31},
		{"历史alias", "legacy-code", "当前名称", 0},
		{"当前code", "current-code", "当前名称", 0},
		{"删除ID不可串到同名code", "other-code", "other-code", 999},
		{"未知标识", "unknown", "unknown", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupUserGroupNamesDB(t)
			user := model.User{Id: 7, Username: "fixture-user", Password: "fixture-secret", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: tc.code, GroupId: tc.groupID, Remark: "admin-only-remark", AffCode: "fixture-aff"}
			require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error)
			self := userGroupResponse(t, GetSelf, "/api/user/self", user.Id, common.RoleCommonUser)
			require.Equal(t, true, self["success"])
			data, ok := self["data"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, tc.want, data["group_name"])
			assert.Equal(t, tc.code, data["group"])
			for _, key := range []string{"password", "access_token", "remark", "group_names", "group_id"} {
				assert.NotContains(t, data, key)
			}
			admin := userGroupResponse(t, GetUser, "/api/user/7", user.Id, common.RoleAdminUser)
			require.Equal(t, true, admin["success"])
			adminData := admin["data"].(map[string]interface{})
			assert.Equal(t, tc.want, adminData["group_name"])
			assert.Equal(t, tc.code, adminData["group"])
			assert.Equal(t, "admin-only-remark", adminData["remark"])
			assert.Empty(t, adminData["password"])
			assert.NotContains(t, adminData, "access_token")
		})
	}
}

func TestUserGroupNamesRenameAndMissingName(t *testing.T) {
	db := setupUserGroupNamesDB(t)
	user := model.User{Id: 7, Username: "fixture-user", Group: "legacy-code", GroupId: 31, AffCode: "fixture-aff", Role: common.RoleCommonUser}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error)
	for _, name := range []string{"重命名", ""} {
		require.NoError(t, db.Model(&model.Group{}).Where("id = ?", 31).Update("name", name).Error)
		response := userGroupResponse(t, GetSelf, "/api/user/self", 7, common.RoleCommonUser)
		data := response["data"].(map[string]interface{})
		expected := name
		if expected == "" {
			expected = "legacy-code"
		}
		assert.Equal(t, expected, data["group_name"])
		assert.Equal(t, "legacy-code", data["group"])
	}
}

func TestUserGroupNamesAdminRoleGateUnchanged(t *testing.T) {
	db := setupUserGroupNamesDB(t)
	user := model.User{Id: 7, Username: "admin-target", Group: "current-code", GroupId: 31, AffCode: "fixture-aff", Role: common.RoleAdminUser}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error)
	denied := userGroupResponse(t, GetUser, "/api/user/7", 7, common.RoleAdminUser)
	assert.Equal(t, false, denied["success"])
	assert.NotContains(t, denied, "data")
	root := userGroupResponse(t, GetUser, "/api/user/7", 7, common.RoleRootUser)
	require.Equal(t, true, root["success"])
	assert.Equal(t, "当前名称", root["data"].(map[string]interface{})["group_name"])
}

func TestUserGroupNamesLegacyAliasFollowsRenameAndDeletion(t *testing.T) {
	db := setupUserGroupNamesDB(t)
	user := model.User{Id: 7, Username: "legacy-user", Group: "legacy-code", AffCode: "legacy-aff", Role: common.RoleCommonUser}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&user).Error)
	require.NoError(t, db.Model(&model.Group{}).Where("id = ?", 31).Update("name", "历史组的新名称").Error)
	response := userGroupResponse(t, GetSelf, "/api/user/self", 7, common.RoleCommonUser)
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "历史组的新名称", data["group_name"])
	assert.Equal(t, "legacy-code", data["group"])
	require.NoError(t, db.Delete(&model.Group{}, 31).Error)
	response = userGroupResponse(t, GetSelf, "/api/user/self", 7, common.RoleCommonUser)
	data = response["data"].(map[string]interface{})
	assert.Equal(t, "legacy-code", data["group_name"])
	assert.Equal(t, "legacy-code", data["group"])
}

func TestUserGroupNamesListAndSearchKeepBusinessIdentifier(t *testing.T) {
	db := setupUserGroupNamesDB(t)
	users := []model.User{
		{Id: 7, Username: "by-id", Group: "other-code", GroupId: 31, AffCode: "fixture-aff-7"},
		{Id: 8, Username: "by-alias", Group: "legacy-code", AffCode: "fixture-aff-8"},
	}
	require.NoError(t, db.Session(&gorm.Session{SkipHooks: true}).Create(&users).Error)
	for _, route := range []struct {
		handler gin.HandlerFunc
		url     string
		count   int
	}{
		{GetAllUsers, "/api/user/?p=1&page_size=20", 2},
		{SearchUsers, "/api/user/search?group=legacy-code&p=1&page_size=20", 1},
	} {
		response := userGroupResponse(t, route.handler, route.url, 1, common.RoleRootUser)
		require.Equal(t, true, response["success"])
		items := response["data"].(map[string]interface{})["items"].([]interface{})
		require.Len(t, items, route.count)
		for _, item := range items {
			data := item.(map[string]interface{})
			assert.Equal(t, "当前名称", data["group_name"])
			assert.Contains(t, []string{"other-code", "legacy-code"}, data["group"])
		}
	}
}
