package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupTagsUserResponsePreservesCodeMapAndClipsBindings(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Group{}, &model.GroupAlias{}, &model.GroupTag{}, &model.GroupTagBinding{}, &model.User{}))
	first := model.Group{Code: "internal-a", Name: "当前显示名称", Ratio: 1, Status: model.GroupStatusActive}
	second := model.Group{Code: "internal-b", Name: "不可用分组", Ratio: 1, Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)
	user := model.User{Username: "tag-user", Password: "password", Group: first.Code}
	require.NoError(t, db.Create(&user).Error)
	_, err := model.SaveGroupTag(model.GroupTag{Name: "混合平台", Icons: []string{"OpenAI", "Claude.Color"}, GroupIDs: []int{first.Id, second.Id}})
	require.NoError(t, err)
	_, err = model.SaveGroupTag(model.GroupTag{Name: "隐藏标签", GroupIDs: []int{second.Id}})
	require.NoError(t, err)
	oldRatios := ratio_setting.GroupRatio2JSONString()
	oldUsable := setting.UserUsableGroups2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsable))
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"internal-a":1,"internal-b":1}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"internal-a":"用户可用"}`))
	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	var response struct {
		Success bool `json:"success"`
		Data    map[string]struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Tags []model.GroupTag `json:"group_tags"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, first.Id, response.Data[first.Code].Id)
	assert.Equal(t, first.Name, response.Data[first.Code].Name)
	assert.NotContains(t, response.Data, second.Code)
	require.Len(t, response.Tags, 1)
	assert.Equal(t, []int{first.Id}, response.Tags[0].GroupIDs)
}

func TestGroupTagControllerUsesURLIdentityForUpdates(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Group{}, &model.GroupTag{}, &model.GroupTagBinding{}))
	tag, err := model.SaveGroupTag(model.GroupTag{Name: "原名称"})
	require.NoError(t, err)
	ctx, recorder := newAuthenticatedContext(t, http.MethodPut, "/api/group/tags/1", map[string]interface{}{
		"id": 99999, "name": "新名称", "icons": []string{"OpenAI"}, "group_ids": []int{},
		"match_keywords": []string{"Gemini", "Pro"},
	}, 1)
	ctx.Params = gin.Params{{Key: "id", Value: stringInt(tag.Id)}}
	SaveGroupTag(ctx)
	var response struct {
		Success bool           `json:"success"`
		Data    model.GroupTag `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	assert.Equal(t, tag.Id, response.Data.Id)
	assert.Equal(t, "新名称", response.Data.Name)
	assert.Equal(t, []string{"Gemini", "Pro"}, response.Data.MatchKeywords)
}

func TestGroupTagControllerReturnsFinalMatchedGroupsOnCreateAndUpdate(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Group{}, &model.GroupTag{}, &model.GroupTagBinding{}))
	groups := []model.Group{
		{Code: "internal-one", Name: "Gemini 标准", Ratio: 1},
		{Code: "internal-two", Name: "Claude 专用", Ratio: 1},
	}
	require.NoError(t, db.Create(&groups).Error)
	var tagID int
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		keywords := []string{"gemini"}
		if method == http.MethodPut {
			keywords = append(keywords, "claude")
		}
		ctx, recorder := newAuthenticatedContext(t, method, "/api/group/tags", map[string]interface{}{
			"name": "混合标签", "match_keywords": keywords, "group_ids": []int{},
		}, 1)
		if method == http.MethodPut {
			ctx.Params = gin.Params{{Key: "id", Value: stringInt(tagID)}}
		}
		SaveGroupTag(ctx)
		var response struct {
			Success bool           `json:"success"`
			Data    model.GroupTag `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success, recorder.Body.String())
		tagID = response.Data.Id
		expected := []int{groups[0].Id}
		if method == http.MethodPut {
			expected = append(expected, groups[1].Id)
		}
		assert.Equal(t, expected, response.Data.GroupIDs)
		var committed []int
		require.NoError(t, db.Model(&model.GroupTagBinding{}).Where("tag_id = ?", tagID).Order("group_id").Pluck("group_id", &committed).Error)
		assert.Equal(t, committed, response.Data.GroupIDs)
	}
}
