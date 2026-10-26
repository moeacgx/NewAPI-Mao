package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/relay"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJevTaskHistoryPersistenceBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, key, status        string
		native, retain, wantTask bool
	}{
		{"官方同步", "typesafe", model.TaskStatusSuccess, true, false, false},
		{"Cloudflare 同步别名", "cloudflare-jev", model.TaskStatusSuccess, true, false, false},
		{"其他同步插件", "other", model.TaskStatusSuccess, true, false, true},
		{"异步仍需轮询", "cloudflare-jev", "", true, false, true},
		{"失败仍需退款认领", "cloudflare-jev", model.TaskStatusFailure, true, false, true},
		{"保留结果路由", "typesafe", model.TaskStatusSuccess, true, true, true},
		{"通用任务接口", "cloudflare-jev", model.TaskStatusSuccess, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setupCanvasControllerDB(t)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/systemone", nil)
			if tc.native {
				c.Set(pluginruntime.ContextKeyPinnedRoute, pluginruntime.PinnedRoute{
					Plugin: &pluginruntime.LoadedPlugin{Meta: pluginruntime.Meta{Key: tc.key}},
					Route:  pluginruntime.Route{RetainResult: common.GetPointer(tc.retain)},
				})
			}
			info := &relaycommon.RelayInfo{UserId: 1, OriginModelName: "Typesafe-jev", UsingGroup: "default",
				ChannelMeta:   &relaycommon.ChannelMeta{ChannelId: 1, UpstreamModelName: "typesafe/jev"},
				TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			response := &channel.TaskSubmitResponse{}
			if tc.status != "" {
				response.Immediate = &relaycommon.TaskInfo{Status: tc.status}
			}
			require.NoError(t, persistOfficialPluginTask(c, info, &relay.TaskSubmitResult{Platform: constant.TaskPlatform(tc.key), PluginResponse: response}))
			var count int64
			require.NoError(t, model.DB.Model(&model.Task{}).Count(&count).Error)
			if tc.wantTask {
				assert.EqualValues(t, 1, count)
			} else {
				assert.Zero(t, count)
			}
		})
	}
}
