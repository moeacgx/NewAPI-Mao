package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanvasDefaultGroupForUsableGroups(t *testing.T) {
	usableGroups := map[string]map[string]interface{}{"vip": {}}
	assert.Equal(t, "vip", canvasDefaultGroupForUsableGroups(usableGroups, " vip "))
	assert.Empty(t, canvasDefaultGroupForUsableGroups(usableGroups, "default"))

	legacyGroups := map[string]map[string]interface{}{
		"legacy-vip": {"code": "vip"},
	}
	assert.Equal(t, "legacy-vip", canvasDefaultGroupForUsableGroups(legacyGroups, "vip"))
}

func TestGetUserGroupsCanvasModalityDefaultsStayWithinUsableGroups(t *testing.T) {
	db := setupTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.User{}))
	allowed := model.Group{Code: "vip-code", Name: "会员显示名", Ratio: 1, Status: model.GroupStatusActive}
	hidden := model.Group{Code: "hidden-code", Name: "隐藏分组", Ratio: 1, Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&allowed).Error)
	require.NoError(t, db.Create(&hidden).Error)
	user := model.User{Username: "canvas-modality-user", Password: "test-password", Group: allowed.Code}
	require.NoError(t, db.Create(&user).Error)

	oldRatios, oldUsable := ratio_setting.GroupRatio2JSONString(), setting.UserUsableGroups2JSONString()
	previousSettings := *model_setting.GetGlobalSettings()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUsable))
		*model_setting.GetGlobalSettings() = previousSettings
	})
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"vip-code":1,"hidden-code":2}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"vip-code":"会员可用"}`))
	settings := model_setting.GetGlobalSettings()
	settings.CanvasDefaultGroup = "vip-code"
	settings.CanvasTextDefaultGroup = " vip-code "
	settings.CanvasImageDefaultGroup = "hidden-code"
	settings.CanvasAudioDefaultGroup = "   "
	settings.CanvasVideoDefaultGroup = "missing"

	ctx, recorder := newAuthenticatedContext(t, http.MethodGet, "/api/user/self/groups", nil, user.Id)
	GetUserGroups(ctx)
	var response struct {
		Success bool `json:"success"`
		Data    map[string]struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
		Canvas      string `json:"canvas_default_group"`
		CanvasText  string `json:"canvas_text_default_group"`
		CanvasImage string `json:"canvas_image_default_group"`
		CanvasAudio string `json:"canvas_audio_default_group"`
		CanvasVideo string `json:"canvas_video_default_group"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Contains(t, response.Data, "vip-code")
	assert.Equal(t, "vip-code", response.Data["vip-code"].Code)
	assert.Equal(t, "会员显示名", response.Data["vip-code"].Name)
	assert.NotContains(t, response.Data, "hidden-code")
	assert.Equal(t, "vip-code", response.Canvas)
	assert.Equal(t, "vip-code", response.CanvasText)
	assert.Empty(t, response.CanvasImage)
	assert.Empty(t, response.CanvasAudio)
	assert.Empty(t, response.CanvasVideo)
}
