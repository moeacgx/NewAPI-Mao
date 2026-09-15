package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginSourceTaskAuthorizationUsesPersistedGroups(t *testing.T) {
	for _, failure := range []string{"", "owner", "group", "model", "channel", "version"} {
		t.Run(failure, func(t *testing.T) {
			c, info, group, ch, task := setupOriginTaskRouteTest(t)
			c.Set("id", task.UserId)
			ch.Type = constant.ChannelTypeTaskPlugin
			ch.Setting = common.GetPointer(`{"task_plugin_key":"sora"}`)
			require.NoError(t, model.DB.Save(ch).Error)
			task.Platform = "sora"
			task.PrivateData.Execution = &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: "sora", SourceHash: "hash"}}
			require.NoError(t, model.DB.Save(task).Error)
			info.OriginModelName = "sora-remix"
			c.Set("task_plugin_snapshot", &model.TaskPluginSnapshot{Key: "sora", SourceHash: "hash"})
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/task/plugins/sora", strings.NewReader(`{"model":"sora-remix","originTaskIds":["task_origin"]}`))
			c.Request.Header.Set("Content-Type", "application/json")
			switch failure {
			case "owner":
				c.Set("id", task.UserId+1)
			case "group":
				require.NoError(t, model.DB.Model(group).Update("status", model.GroupStatusDisabled).Error)
			case "model":
				c.Set("token_model_limit_enabled", true)
				c.Set("token_model_limit", map[string]bool{"other": true})
			case "channel":
				require.NoError(t, model.DB.Model(&model.Ability{}).Where("channel_id = ?", ch.Id).Update("enabled", false).Error)
			case "version":
				c.Set("task_plugin_snapshot", &model.TaskPluginSnapshot{Key: "sora", SourceHash: "new-hash"})
			}
			err := ResolvePluginOriginTasks(c, info)
			if failure != "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, group.Code, info.UsingGroup)
			require.Len(t, info.OriginTasks, 1)
			assert.Equal(t, task.GetUpstreamTaskID(), info.OriginTaskID)
			assert.IsType(t, &model.Channel{}, info.LockedChannel)
			assert.NotNil(t, GetTaskAdaptor("61"))
			assert.Nil(t, GetTaskAdaptor("62"))
			info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
		})
	}
}
