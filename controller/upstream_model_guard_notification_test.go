package controller

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/extension"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpstreamModelGuardNotificationDefaultCanBeSavedWithExistingBot(t *testing.T) {
	setupNotificationControllerTestDB(t)
	previous := extension.DefaultManager
	previousVersion := common.Version
	common.Version = "v1.0.0-rc.10.1.10.326"
	t.Cleanup(func() { common.Version = previousVersion })
	t.Setenv("EXTENSIONS_ROOT", t.TempDir())
	require.NoError(t, extension.Init())
	t.Cleanup(func() { extension.DefaultManager = previous })
	require.NoError(t, os.CopyFS(filepath.Join(extension.DefaultManager.RootDir(), "upstream-model-guard"), os.DirFS("../extensions/upstream-model-guard")))
	require.NoError(t, extension.DefaultManager.Scan())
	_, err := extension.DefaultManager.SetEnabled("upstream-model-guard", true)
	require.NoError(t, err)
	registered, err := extension.DefaultManager.ResolveNotificationEvent("upstream-model-guard", "channel_disabled", common.RoleRootUser)
	require.NoError(t, err)
	assert.Equal(t, model.UpstreamModelGuardNotificationEvent, registered.EventType)
	definition, ok := findNotificationEventDefinition(model.UpstreamModelGuardNotificationEvent)
	require.True(t, ok)
	assert.True(t, definition.Available)
	assert.Contains(t, definition.Variables, "channel_id")
	assert.Contains(t, definition.Variables, "actual_upstream_model")
	assert.Contains(t, definition.Variables, "comparison")
	assert.Contains(t, definition.Variables, "consecutive_mismatches")
	assert.Contains(t, definition.Variables, "failure_threshold")
	assert.Contains(t, definition.DefaultTemplate, "{{channel_name}}")
	assert.Contains(t, definition.DefaultTemplate, "{{channel_id}}")
	assert.Contains(t, definition.DefaultTemplate, "{{comparison}}")
	bot := &model.NotificationBot{Name: "existing bot", Token: "local-test-only", Enabled: true}
	require.NoError(t, model.CreateNotificationBot(bot))
	payload := map[string]any{
		"name": "model mismatch", "event_type": definition.Value, "bot_id": bot.Id,
		"template": definition.DefaultTemplate, "enabled": true,
		"targets": []map[string]any{{"chat_id": "-10001", "enabled": true}},
	}
	body, err := common.Marshal(payload)
	require.NoError(t, err)
	response := invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(body), 0)
	require.Contains(t, response.Body.String(), `"success":true`)
	var task model.NotificationTask
	require.NoError(t, model.DB.First(&task).Error)
	assert.Equal(t, bot.Id, task.BotId)
	assert.Equal(t, definition.DefaultTemplate, task.Template)
	payload["template"] = "{{bot_token}}"
	body, err = common.Marshal(payload)
	require.NoError(t, err)
	response = invokeNotificationHandler(CreateNotificationTask, common.RoleRootUser, http.MethodPost, "/api/notification/tasks", string(body), 0)
	assert.Contains(t, response.Body.String(), `"success":false`)
	assert.Contains(t, response.Body.String(), "unknown notification template variable: bot_token")
	var count int64
	require.NoError(t, model.DB.Model(&model.NotificationTask{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
