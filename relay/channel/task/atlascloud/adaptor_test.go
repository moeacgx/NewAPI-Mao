package atlascloud

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAtlasCloudDurationValidatedBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		fields string
		want   float64
		valid  bool
	}{
		{"duration", `"duration":8`, 8, true},
		{"seconds", `"seconds":"6"`, 6, true},
		{"metadata override", `"duration":4,"metadata":{"duration":8}`, 8, true},
		{"metadata string", `"metadata":{"duration":"6"}`, 6, true},
		{"maximum", `"duration":3600`, 3600, true},
		{"zero", `"duration":0`, 0, true},
		{"metadata zero preserves duration", `"duration":8,"metadata":{"duration":0}`, 8, true},
		{"duration zero preserves seconds", `"seconds":"6","duration":0`, 6, true},
		{"negative", `"duration":-1`, 0, false},
		{"oversized", `"duration":3601`, 0, false},
		{"huge", `"duration":18446744073686646784`, 0, false},
		{"fraction", `"duration":1.5`, 0, false},
		{"seconds invalid", `"seconds":"abc"`, 0, false},
		{"seconds oversized", `"seconds":"3601"`, 0, false},
		{"metadata oversized", `"metadata":{"duration":3601}`, 0, false},
		{"metadata invalid", `"metadata":{"duration":"NaN"}`, 0, false},
		{"metadata fraction", `"metadata":{"duration":2.5}`, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{"model":"atlas-video","prompt":"test",`+tc.fields+`}`))
			c.Request.Header.Set("Content-Type", "application/json")
			t.Cleanup(func() {
				if storage, ok := c.Get(common.KeyBodyStorage); ok {
					_ = storage.(common.BodyStorage).Close()
				}
			})
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}, ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeAtlasCloud, UpstreamModelName: "mapped-video"}}
			adaptor := &TaskAdaptor{}
			taskErr := adaptor.ValidateRequestAndSetAction(c, info)
			if !tc.valid {
				require.NotNil(t, taskErr)
				assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
				return
			}
			require.Nil(t, taskErr)
			assert.Equal(t, tc.want, adaptor.EstimateBilling(c, info)["seconds"])
			body, err := adaptor.BuildRequestBody(c, info)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.DecodeJson(body, &payload))
			assert.Equal(t, "mapped-video", payload["model"])
			if tc.want > 0 {
				assert.Equal(t, tc.want, payload["duration"])
			} else {
				assert.NotContains(t, payload, "duration")
			}
		})
	}
}

func TestAtlasCloudRetainsPublicIDAndNativePolling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	writer := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(writer)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", nil)
	c.Set("task_request", relaycommon.TaskSubmitReq{Duration: 8, Size: "1280x720"})
	info := &relaycommon.RelayInfo{OriginModelName: "local-alias", TaskRelayInfo: &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"}}
	a := &TaskAdaptor{}
	id, data, taskErr := a.DoResponse(c, &http.Response{Body: io.NopCloser(strings.NewReader(`{"data":{"id":"private-upstream-id"}}`))}, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "private-upstream-id", id)
	assert.Contains(t, string(data), "private-upstream-id")
	assert.NotContains(t, writer.Body.String(), "private-upstream-id")
	assert.Contains(t, writer.Body.String(), "task_public")
	assert.Contains(t, writer.Body.String(), "local-alias")
	for _, tc := range []struct{ body, status string }{
		{`{"data":{"status":"completed","outputs":["https://media.invalid/video.mp4"]}}`, model.TaskStatusSuccess},
		{`{"data":{"status":"completed"}}`, model.TaskStatusFailure},
		{`{"data":{"status":"failed"}}`, model.TaskStatusFailure},
	} {
		result, err := a.ParseTaskResult([]byte(tc.body))
		require.NoError(t, err)
		assert.Equal(t, tc.status, result.Status)
	}
}

func TestAtlasCloudFetchUsesNativePredictionEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/model/prediction/upstream-task", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer test-only", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{"data":{"status":"processing"}}`)
	}))
	defer server.Close()
	service.InitHttpClient()
	a := &TaskAdaptor{}
	resp, err := a.FetchTask(server.URL, "test-only", map[string]any{"task_id": "upstream-task"}, "")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	result, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusInProgress, result.Status)
}
