package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pluginPollSecret = "REVIEW_FAKE_SECRET signed_url=https://example.invalid?access=FAKE Authorization=REVIEW_SECRET"

type failingPluginPollReader struct{}

func (failingPluginPollReader) Read([]byte) (int, error) { return 0, errors.New(pluginPollSecret) }
func (failingPluginPollReader) Close() error             { return nil }

type pluginBoundaryPollAdaptor struct{ stage string }

func (*pluginBoundaryPollAdaptor) Init(*relaycommon.RelayInfo) {}
func (*pluginBoundaryPollAdaptor) AdjustBillingOnComplete(*model.Task, *relaycommon.TaskInfo) int {
	return 0
}
func (a *pluginBoundaryPollAdaptor) FetchTask(string, string, map[string]any, string) (*http.Response, error) {
	if a.stage == "fetch_failed" {
		return nil, errors.New(pluginPollSecret)
	}
	var body io.ReadCloser = io.NopCloser(strings.NewReader(`{}`))
	if a.stage == "read_failed" {
		body = failingPluginPollReader{}
	}
	return &http.Response{StatusCode: 429, Body: body, Header: make(http.Header)}, nil
}
func (a *pluginBoundaryPollAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	if a.stage == "parse_failed" {
		return nil, errors.New(pluginPollSecret)
	}
	return &relaycommon.TaskInfo{Status: pluginPollSecret, Reason: pluginPollSecret}, nil
}

func TestTaskPluginPollingErrorBoundariesRedactPayload(t *testing.T) {
	for _, stage := range []string{"fetch_failed", "read_failed", "parse_failed", "unknown_status"} {
		t.Run(stage, func(t *testing.T) {
			truncate(t)
			seedChannel(t, 8801)
			var ch model.Channel
			require.NoError(t, model.DB.First(&ch, 8801).Error)
			ch.Type = constant.ChannelTypeTaskPlugin
			ch.BaseURL = common.GetPointer("https://example.invalid")
			require.NoError(t, model.DB.Save(&ch).Error)
			task := &model.Task{TaskID: "task_public_boundary", Platform: "review", ChannelId: ch.Id, Status: model.TaskStatusInProgress, Progress: "10%", PrivateData: model.TaskPrivateData{UpstreamTaskID: "private-provider-id", Execution: &model.TaskExecutionSnapshot{TaskPlugin: &model.TaskPluginSnapshot{Key: "review", Version: "1.2.3"}}}}
			require.NoError(t, task.Insert())
			oldFactory, oldPinned := GetTaskAdaptorFunc, GetPinnedTaskAdaptorFunc
			oldCache, oldDebug := common.MemoryCacheEnabled, common.DebugEnabled
			common.MemoryCacheEnabled = false
			common.DebugEnabled = true
			GetTaskAdaptorFunc = func(constant.TaskPlatform) TaskPollingAdaptor { return nil }
			GetPinnedTaskAdaptorFunc = func(*model.Task) (TaskPollingAdaptor, error) { return &pluginBoundaryPollAdaptor{stage: stage}, nil }
			var logs bytes.Buffer
			common.LogWriterMu.Lock()
			oldWriter, oldError := gin.DefaultWriter, gin.DefaultErrorWriter
			gin.DefaultWriter = &logs
			gin.DefaultErrorWriter = &logs
			common.LogWriterMu.Unlock()
			t.Cleanup(func() {
				GetTaskAdaptorFunc = oldFactory
				GetPinnedTaskAdaptorFunc = oldPinned
				common.MemoryCacheEnabled = oldCache
				common.DebugEnabled = oldDebug
				common.LogWriterMu.Lock()
				gin.DefaultWriter = oldWriter
				gin.DefaultErrorWriter = oldError
				common.LogWriterMu.Unlock()
			})
			tasks := map[string]*model.Task{task.GetUpstreamTaskID(): task}
			err := updateVideoSingleTask(context.Background(), nil, &ch, task.GetUpstreamTaskID(), tasks)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "stage="+stage)
			require.NoError(t, updateVideoTasks(context.Background(), "review", ch.Id, []string{task.GetUpstreamTaskID()}, tasks))
			assert.Contains(t, logs.String(), "stage="+stage)
			assert.Contains(t, logs.String(), "task_public_boundary")
			assert.Contains(t, logs.String(), "version=\"1.2.3\"")
			for _, marker := range []string{"REVIEW_FAKE_SECRET", "signed_url", "access=FAKE", "Authorization", "private-provider-id"} {
				assert.NotContains(t, err.Error(), marker)
				assert.NotContains(t, logs.String(), marker)
			}
		})
	}
}
