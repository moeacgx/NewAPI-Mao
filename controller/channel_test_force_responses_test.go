package controller

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelProbeForceResponsesUsesConfiguredUpstreamProtocol(t *testing.T) {
	group, channel := setupChannelGroupDisplayControllerTestDB(t)
	savedModelRatio := ratio_setting.ModelRatio2JSONString()
	savedGroupRatio := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(savedModelRatio))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(savedGroupRatio))
	})
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"gpt-4o-mini":1}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"group_2":1}`))
	user := &model.User{Username: "force-responses-probe", Password: "test-password", Group: group.Code, Quota: 1000000, Status: common.UserStatusEnabled}
	require.NoError(t, model.DB.Create(user).Error)
	for _, tt := range []struct {
		name      string
		force     bool
		endpoint  constant.EndpointType
		wantPath  string
		wantField string
	}{
		{"关闭保留 Chat", false, constant.EndpointTypeOpenAI, "/v1/chat/completions", "messages"},
		{"默认探测", true, "", "/v1/responses", "input"},
		{"显式 Chat", true, constant.EndpointTypeOpenAI, "/v1/responses", "input"},
		{"显式 Claude", true, constant.EndpointTypeAnthropic, "/v1/responses", "input"},
		{"显式 Gemini", true, constant.EndpointTypeGemini, "/v1/responses", "input"},
		{"向量不转换", true, constant.EndpointTypeEmbeddings, "/v1/embeddings", "input"},
		{"Compact 不转换", true, constant.EndpointTypeOpenAIResponseCompact, "/v1/responses/compact", "input"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var outbound map[string]any
			var path string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				assert.NoError(t, common.DecodeJson(r.Body, &outbound))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = io.WriteString(w, `{"error":{"type":"invalid_request_error","message":"probe captured"}}`)
			}))
			t.Cleanup(server.Close)
			channel.BaseURL = common.GetPointer(server.URL)
			channel.SetSetting(dto.ChannelSettings{ForceResponses: tt.force})
			result := testChannel(context.Background(), channel, user.Id, "gpt-4o-mini", string(tt.endpoint), false)
			require.NotNil(t, result.newAPIError, "local error: %v", result.localErr)
			require.Equal(t, http.StatusBadRequest, result.newAPIError.StatusCode, "local error: %v", result.localErr)
			assert.Equal(t, tt.wantPath, path, "local error: %v", result.localErr)
			assert.Contains(t, outbound, tt.wantField)
		})
	}
}
