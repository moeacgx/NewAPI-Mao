package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTasksToDtoHidesUpstreamModelForUserView(t *testing.T) {
	tasks := []*model.Task{{
		TaskID:   "task-redirect",
		Platform: constant.TaskPlatformImage,
		Properties: model.Properties{
			OriginModelName:   "gpt-5.6-sol",
			UpstreamModelName: "gpt-5.6-sol-wm",
		},
	}}

	items := tasksToDto(tasks, false)
	require.Len(t, items, 1)

	props, ok := items[0].Properties.(model.Properties)
	require.True(t, ok)
	require.Equal(t, "gpt-5.6-sol", props.OriginModelName)
	require.Empty(t, props.UpstreamModelName)
}

func TestTasksToDtoPreservesAccountingRowsAndResultDiscarded(t *testing.T) {
	db := setupTaskGroupDisplayNameTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.Group{Code: "default", Name: "默认分组", Status: model.GroupStatusActive}).Error)
	require.NoError(t, db.Create(&model.User{Id: 96941, Username: "discarded-result-view"}).Error)
	previousRedis := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() { common.RedisEnabled = previousRedis })

	for _, admin := range []bool{false, true} {
		name := "普通用户"
		if admin {
			name = "管理员"
		}
		t.Run(name, func(t *testing.T) {
			tasks := []*model.Task{
				{TaskID: "discarded", UserId: 96941, Platform: "cloudflare-jev", Status: model.TaskStatusSuccess, Quota: 11,
					PrivateData: model.TaskPrivateData{ResultDiscarded: true, Key: "PRIVATE_CHANNEL_KEY", PluginData: []byte(`{"answers":"PRIVATE_ANSWERS"}`)}},
				{TaskID: "retained", UserId: 96941, Platform: "sora", Status: model.TaskStatusSuccess, Quota: 25},
			}
			items := tasksToDto(tasks, admin)
			require.Len(t, items, 2, "结果不保留不能隐藏账务行")
			payload, err := common.Marshal(items)
			require.NoError(t, err)
			var rows []map[string]any
			require.NoError(t, common.Unmarshal(payload, &rows))
			assert.Equal(t, true, rows[0]["result_discarded"])
			assert.Equal(t, float64(11), rows[0]["quota"])
			assert.NotContains(t, rows[1], "result_discarded")
			assert.NotContains(t, string(payload), "PRIVATE_CHANNEL_KEY")
			assert.NotContains(t, string(payload), "PRIVATE_ANSWERS")
		})
	}
}
