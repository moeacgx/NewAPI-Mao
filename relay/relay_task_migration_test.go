package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/atlascloud"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationKeepsNativeDispatch(t *testing.T) {
	for _, platform := range []constant.TaskPlatform{"suno", "17", "24", "35", "41", "45", "50", "51", "52", "54", "55", "1", "61"} {
		require.NotNil(t, GetTaskAdaptor(platform), "platform %s", platform)
	}
	assert.IsType(t, &atlascloud.TaskAdaptor{}, GetTaskAdaptor("61"))
	assert.Nil(t, GetTaskAdaptor("sora"))
	assert.Nil(t, GetTaskAdaptor("unknown"))
}

func TestOriginTaskMigrationAliasesAndMultipleGroups(t *testing.T) {
	for _, mode := range []string{model.TokenGroupModeExplicit, model.TokenGroupModeAuto, model.TokenGroupModeInherit, ""} {
		t.Run(mode, func(t *testing.T) {
			c, info, group, _, task := setupOriginTaskRouteTest(t)
			require.NoError(t, model.DB.Create(&model.GroupAlias{Alias: "old-group", GroupId: group.Id}).Error)
			require.NoError(t, model.DB.Model(task).Update("group", "old-group").Error)
			common.SetContextKey(c, constant.ContextKeyTokenGroupMode, mode)
			common.SetContextKey(c, constant.ContextKeyUserGroup, "old-group")
			if mode == model.TokenGroupModeExplicit {
				common.SetContextKey(c, constant.ContextKeyTokenGroup, "other")
				common.SetContextKey(c, constant.ContextKeyTokenGroups, []string{"other", "old-group"})
				common.SetContextKey(c, constant.ContextKeyTokenGroupIds, []int{group.Id + 1, group.Id})
			}
			if mode == model.TokenGroupModeAuto {
				common.SetContextKey(c, constant.ContextKeyUserGroup, group.Code)
				common.SetContextKey(c, constant.ContextKeyTokenAutoGroups, []string{"missing", group.Code})
			}
			require.Nil(t, ResolveOriginTask(c, info))
			assert.Equal(t, group.Code, info.UsingGroup)
			assert.Equal(t, group.Code, common.GetContextKeyString(c, constant.ContextKeySelectedChannelGroup))
		})
	}
}

func TestOriginTaskMigrationExplicitAliasFallbackWithoutIDs(t *testing.T) {
	c, info, group, _, task := setupOriginTaskRouteTest(t)
	require.NoError(t, model.DB.Create(&model.GroupAlias{Alias: "old-explicit", GroupId: group.Id}).Error)
	require.NoError(t, model.DB.Model(task).Update("group", "old-explicit").Error)
	common.SetContextKey(c, constant.ContextKeyTokenGroupMode, "")
	common.SetContextKey(c, constant.ContextKeyTokenGroup, "other")
	common.SetContextKey(c, constant.ContextKeyTokenGroups, []string{"other", "old-explicit"})
	require.Nil(t, ResolveOriginTask(c, info))
	assert.Equal(t, group.Code, info.UsingGroup)
	common.SetContextKey(c, constant.ContextKeyTokenGroups, []string{"other"})
	info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	taskErr := ResolveOriginTask(c, info)
	require.NotNil(t, taskErr)
	assert.Equal(t, "task_group_forbidden", taskErr.Code)
}

// 已有预扣会话仅用于隔离本测试的价格计算；资金结算由 service 回归测试覆盖。
type migrationExistingBilling struct{ relaycommon.BillingSettler }

func TestRemixInheritedRatiosSurvivePriceRebuildAndRetry(t *testing.T) {
	for _, snapshot := range []bool{false, true} {
		t.Run(strconv.FormatBool(snapshot), func(t *testing.T) {
			c, info, _, _, task := setupOriginTaskRouteTest(t)
			if snapshot {
				task.PrivateData.BillingContext = &model.TaskBillingContext{OtherRatios: map[string]float64{"seconds": 8, "size": 2}}
			} else {
				task.SetData(map[string]any{"seconds": "8", "size": "1792x1024"})
			}
			require.NoError(t, model.DB.Save(task).Error)
			require.Nil(t, ResolveOriginTask(c, info))
			oldPrices := ratio_setting.ModelPrice2JSONString()
			t.Cleanup(func() { require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(oldPrices)) })
			require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"sora-remix":0.01}`))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v1/videos/task_origin/remix", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"id":"upstream-remix","status":"queued"}`)
			}))
			defer server.Close()
			service.InitHttpClient()
			common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeSora)
			common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
			info.Billing = &migrationExistingBilling{}
			inherited := info.PriceData.OtherRatios()
			for attempt := 0; attempt < 2; attempt++ {
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos/task_origin/remix", strings.NewReader(`{"prompt":"continue"}`))
				c.Request.Header.Set("Content-Type", "application/json")
				result, taskErr := RelayTaskSubmit(c, info)
				require.Nil(t, taskErr)
				require.NotNil(t, result)
				assert.Equal(t, inherited, info.PriceData.OtherRatios())
				assert.Equal(t, common.QuotaFromFloat(0.01*common.QuotaPerUnit*inherited["seconds"]*inherited["size"]), result.Quota)
				assert.Equal(t, "upstream-remix", result.UpstreamTaskID)
			}
			if storage, exists := c.Get(common.KeyBodyStorage); exists {
				_ = storage.(common.BodyStorage).Close()
			}
		})
	}
}
