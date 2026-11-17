package controller

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionListsResolveCurrentGroupNamesWithoutChangingCodes(t *testing.T) {
	db := setupTaskGroupDisplayNameTestDB(t)
	withConfirmedPaymentCompliance(t)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}))
	group := model.Group{Code: "stable-code", Name: "当前名称", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Create(&model.Group{Code: "unnamed-code", Name: "", Status: model.GroupStatusActive}).Error)
	require.NoError(t, db.Create(&model.GroupAlias{Alias: "legacy-code", GroupId: group.Id}).Error)
	plans := []model.SubscriptionPlan{
		{Title: "current", Enabled: true, UpgradeGroup: "stable-code", DowngradeGroup: "legacy-code"},
		{Title: "alias", Enabled: true, UpgradeGroup: "legacy-code"},
		{Title: "deleted", Enabled: true, UpgradeGroup: "deleted-code"},
		{Title: "empty", Enabled: true},
		{Title: "disabled", Enabled: false, UpgradeGroup: "stable-code"},
		{Title: "unnamed", Enabled: true, UpgradeGroup: "unnamed-code"},
	}
	require.NoError(t, db.Create(&plans).Error)
	// GORM 的 default:true 会覆盖 Create 的 false 零值，明确初始化禁用状态。
	require.NoError(t, db.Model(&plans[4]).Update("enabled", false).Error)
	for _, name := range []string{"当前名称", "改名后的名称"} {
		require.NoError(t, db.Model(&group).Update("name", name).Error)
		for _, tc := range []struct {
			name    string
			handler gin.HandlerFunc
			count   int
		}{{"user", GetSubscriptionPlans, 5}, {"admin", AdminListSubscriptionPlans, 6}} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				w := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(w)
				tc.handler(ctx)
				var response struct {
					Success bool `json:"success"`
					Data    []struct {
						Plan map[string]interface{} `json:"plan"`
					} `json:"data"`
				}
				require.NoError(t, common.Unmarshal(w.Body.Bytes(), &response))
				require.True(t, response.Success)
				require.Len(t, response.Data, tc.count)
				for _, row := range response.Data {
					code, _ := row.Plan["upgrade_group"].(string)
					expected := code
					if code == "stable-code" || code == "legacy-code" {
						expected = name
					}
					assert.Equal(t, expected, row.Plan["upgrade_group_name"])
					if row.Plan["title"] == "current" {
						assert.Equal(t, "stable-code", code)
						assert.Equal(t, "legacy-code", row.Plan["downgrade_group"])
						assert.Equal(t, name, row.Plan["downgrade_group_name"])
					}
				}
			})
		}
	}
	var stored model.SubscriptionPlan
	require.NoError(t, db.First(&stored, plans[1].Id).Error)
	assert.Equal(t, "legacy-code", stored.UpgradeGroup)

	// 删除实体但留下历史套餐和 alias，名称不得归给其他分组。
	require.NoError(t, db.Delete(&group).Error)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	AdminListSubscriptionPlans(ctx)
	var deleted struct {
		Success bool `json:"success"`
		Data    []struct {
			Plan struct {
				UpgradeGroup     string `json:"upgrade_group"`
				UpgradeGroupName string `json:"upgrade_group_name"`
			} `json:"plan"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &deleted))
	require.True(t, deleted.Success)
	require.Len(t, deleted.Data, 6)
	for _, row := range deleted.Data {
		assert.Equal(t, row.Plan.UpgradeGroup, row.Plan.UpgradeGroupName)
	}
}

func TestSubscriptionPlanWritesKeepCodesAndRejectDisplayNames(t *testing.T) {
	db := setupTaskGroupDisplayNameTestDB(t)
	withConfirmedPaymentCompliance(t)
	require.NoError(t, db.AutoMigrate(&model.SubscriptionPlan{}))
	oldRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"stable-code":1}`))
	t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldRatios)) })
	group := model.Group{Code: "stable-code", Name: "当前分组名称", Status: model.GroupStatusActive}
	require.NoError(t, db.Create(&group).Error)
	for _, tc := range []struct {
		name, code string
		success    bool
	}{
		{"code", "stable-code", true},
		{"display-name", "当前分组名称", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, recorder := newSubscriptionPaymentContext(t, map[string]interface{}{
				"plan": map[string]interface{}{
					"title": "测试套餐", "price_amount": 10, "duration_value": 1,
					"upgrade_group": tc.code, "downgrade_group": tc.code,
					"upgrade_group_name": "任意展示文字",
				},
			}, 1)
			AdminCreateSubscriptionPlan(ctx)
			var response struct {
				Success bool                   `json:"success"`
				Data    model.SubscriptionPlan `json:"data"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
			require.Equal(t, tc.success, response.Success)
			if !tc.success {
				return
			}
			assert.Equal(t, "stable-code", response.Data.UpgradeGroup)
			ctx, recorder = newSubscriptionPaymentContext(t, AdminUpsertSubscriptionPlanRequest{Plan: response.Data}, 1)
			ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(response.Data.Id)}}
			AdminUpdateSubscriptionPlan(ctx)
			var updated struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &updated))
			require.True(t, updated.Success)
			var stored model.SubscriptionPlan
			require.NoError(t, db.First(&stored, response.Data.Id).Error)
			assert.Equal(t, "stable-code", stored.UpgradeGroup)
			assert.Equal(t, "stable-code", stored.DowngradeGroup)
		})
	}
}
