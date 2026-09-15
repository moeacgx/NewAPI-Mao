package jsplugin

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// Legacy 将官方纯解析接口接到本地任务链；每个任务单独创建，不能跨任务复用。
type Legacy struct {
	*TaskAdaptor
	task     *model.Task
	response *http.Response
}

func NewLegacy(plugin *pluginruntime.LoadedPlugin, task *model.Task) *Legacy {
	if task != nil {
		copyTask := *task
		copyTask.Data = task.PrivateData.PluginData
		task = &copyTask
	}
	return &Legacy{TaskAdaptor: New(plugin), task: task}
}

func (a *Legacy) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	return "", nil, service.TaskErrorWrapperLocal(fmt.Errorf("插件必须在持久化后返回响应"), "plugin_persistence_required", 500)
}

func (a *Legacy) FetchTask(baseURL, key string, body map[string]any, proxy string) (response *http.Response, returnedErr error) {
	defer func() {
		if returnedErr != nil {
			returnedErr = a.pollingError("fetch_failed")
		}
	}()
	if a.task == nil {
		return nil, fmt.Errorf("插件轮询缺少版本固定任务")
	}
	if a.task.PrivateData.PluginImmediate != nil {
		a.response = &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}
		return a.response, nil
	}
	var err error
	if a.FetchMode() == "batch" {
		a.response, err = a.TaskAdaptor.FetchBatchTasks(baseURL, key, []*model.Task{a.task}, proxy)
	} else {
		a.response, err = a.TaskAdaptor.FetchTask(baseURL, key, a.task, proxy)
	}
	return a.response, err
}

func (a *Legacy) ParseTaskResult(body []byte) (resultInfo *relaycommon.TaskInfo, returnedErr error) {
	defer func() {
		if returnedErr != nil {
			returnedErr = a.pollingError("parse_failed")
			return
		}
		if resultInfo == nil {
			returnedErr = a.pollingError("empty_result")
			return
		}
		switch model.TaskStatus(resultInfo.Status) {
		case model.TaskStatusSubmitted, model.TaskStatusQueued, model.TaskStatusInProgress, model.TaskStatusSuccess, model.TaskStatusFailure:
			resultInfo.Reason = ""
			progress, err := strconv.Atoi(strings.TrimSuffix(resultInfo.Progress, "%"))
			if err != nil || progress < 0 || progress > 100 {
				resultInfo.Progress = ""
			} else {
				resultInfo.Progress = strconv.Itoa(progress) + "%"
			}
		default:
			resultInfo = nil
			returnedErr = a.pollingError("unknown_status")
		}
	}()
	if a.task == nil {
		return nil, fmt.Errorf("插件解析缺少版本固定任务")
	}
	if a.task.PrivateData.PluginImmediate != nil {
		result := *a.task.PrivateData.PluginImmediate
		return &result, nil
	}
	if a.FetchMode() == "batch" {
		results, err := a.ParseBatchResult([]*model.Task{a.task}, a.response, body)
		if err != nil {
			return nil, err
		}
		result := results[a.task.GetUpstreamTaskID()]
		if result == nil {
			return nil, fmt.Errorf("插件批查询缺少目标任务")
		}
		return &result.TaskInfo, nil
	}
	return a.TaskAdaptor.ParseTaskResult(a.task, a.response, body)
}

// pollingError 不包装原始异常，避免供应商 URL、响应和凭据进入日志或调用者。
func (a *Legacy) pollingError(stage string) error {
	taskID := ""
	if a.task != nil {
		taskID = a.task.TaskID
	}
	status := 0
	if a.response != nil {
		status = a.response.StatusCode
	}
	return fmt.Errorf("task_plugin_poll task=%q plugin=%q version=%q stage=%s http_status=%d", taskID, a.plugin.Meta.Key, a.plugin.Meta.Version, stage, status)
}
