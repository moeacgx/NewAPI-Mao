package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMCPChannelGroupReferencesSurviveProjection(t *testing.T) {
	raw := []byte(`{"success":true,"data":{"id":7,"group":"billing-code","group_ids":[2],"group_details":[{"id":2,"code":"billing-code","name":"显示分组"}]}}`)
	projected, success, err := ProjectMCPResponse(MCPTool{Name: "newapi_channel_get", Category: "channels"}, raw, 200)
	require.NoError(t, err)
	require.True(t, success)
	assert.JSONEq(t, string(raw), string(projected))
}

func TestMCPModelGroupNamesRespectRenameAliasAndDeletedGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}))
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; _ = sqlDB.Close() })
	group := model.Group{Code: "billing-code", Name: "显示分组"}
	require.NoError(t, db.Create(&group).Error)
	require.NoError(t, db.Create(&model.GroupAlias{Alias: "historical-code", GroupId: group.Id}).Error)
	raw := []byte(`{"success":true,"data":{"id":7,"enable_groups":["billing-code","historical-code","deleted-code"]}}`)
	tool := MCPTool{Name: "newapi_model_get", Category: "models"}
	projected, _, err := ProjectMCPResponse(tool, raw, 200)
	require.NoError(t, err)
	assert.Contains(t, string(projected), `"enable_group_names":["显示分组","显示分组","deleted-code"]`)
	assert.Contains(t, string(projected), `"enable_groups":["billing-code","historical-code","deleted-code"]`)
	require.NoError(t, db.Model(&group).Update("name", "改名后").Error)
	projected, _, err = ProjectMCPResponse(tool, raw, 200)
	require.NoError(t, err)
	assert.Contains(t, string(projected), `"enable_group_names":["改名后","改名后","deleted-code"]`)
	for _, name := range []string{"newapi_conversation_config", "newapi_model_guard_config"} {
		payload := `{"success":true,"data":{"group_codes":["billing-code","historical-code","deleted-code"]}}`
		if name == "newapi_model_guard_config" {
			payload = `{"success":true,"data":{"rules":[{"group_codes":["billing-code","historical-code","deleted-code"],"model":"test-model","upstream_models":["test-upstream"]}]}}`
		}
		result, _, err := ProjectMCPResponse(MCPTool{Name: name, TypedArguments: true}, []byte(payload), 200)
		require.NoError(t, err)
		assert.Contains(t, string(result), `"group_names":["改名后","改名后","deleted-code"]`)
		assert.Contains(t, string(result), `"group_codes":["billing-code","historical-code","deleted-code"]`)
	}
}
