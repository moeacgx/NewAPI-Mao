package controller

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func ResolveOfficialPluginOrigins() gin.HandlerFunc {
	return func(c *gin.Context) {
		info, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
		if err == nil {
			err = relay.ResolvePluginOriginTasks(c, info)
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if ch, ok := info.LockedChannel.(*model.Channel); ok {
			if setupErr := middleware.SetupContextForSelectedChannel(c, ch, info.OriginModelName); setupErr != nil {
				c.AbortWithStatus(http.StatusServiceUnavailable)
				return
			}
			setSelectedSecurityAuditRoute(c, ch, info.UsingGroup)
		}
		c.Set(resolvedOriginTaskRelayInfoContextKey, info)
		c.Next()
	}
}

func GetOfficialPluginTask(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	task, found, err := model.GetByTaskId(c.GetInt("id"), c.Param("task_id"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !found || relay.AuthorizePluginTaskAccess(c, task, c.Param("plugin_key")) != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if _, err := relay.GetPinnedTaskAdaptor(task); err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	// 公开查询只返回宿主状态和本地资源地址，不暴露供应商响应及签名 URL。
	result := gin.H{"task_id": task.TaskID, "platform": task.Platform, "status": task.Status, "progress": task.Progress, "created_at": task.CreatedAt, "finished_at": task.FinishTime, "model": task.Properties.OriginModelName}
	if task.Status == model.TaskStatusFailure {
		result["error"] = "上游任务失败"
	}
	if task.Status == model.TaskStatusSuccess {
		result["artifacts_url"] = "/v1/task/plugins/" + string(task.Platform) + "/" + task.TaskID + "/artifacts"
	}
	common.ApiSuccess(c, result)
}

// GetOfficialPluginArtifacts 仅开放已存数据中的小型内联资源，不访问远程 URL。
func GetOfficialPluginArtifacts(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	task, found, err := model.GetByTaskId(c.GetInt("id"), c.Param("task_id"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	if !found || relay.AuthorizePluginTaskAccess(c, task, c.Param("plugin_key")) != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	if task.Status != model.TaskStatusSuccess {
		c.AbortWithStatus(http.StatusConflict)
		return
	}
	if _, err := relay.GetPinnedTaskAdaptor(task); err != nil {
		c.AbortWithStatus(http.StatusServiceUnavailable)
		return
	}
	artifactKey := c.Param("artifact_key")
	if !strings.HasPrefix(task.PrivateData.PluginResultURL, "data:") {
		c.AbortWithStatusJSON(http.StatusNotImplemented, gin.H{"error": "remote artifacts are not enabled"})
		return
	}
	if artifactKey == "" {
		common.ApiSuccess(c, []gin.H{{"key": "result", "type": "media"}})
		return
	}
	if artifactKey != "result" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	resourceURL := task.PrivateData.PluginResultURL
	header, encoded, ok := strings.Cut(strings.TrimPrefix(resourceURL, "data:"), ",")
	mimeType, isBase64 := strings.CutSuffix(header, ";base64")
	allowed := map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true, "video/mp4": true, "video/webm": true, "audio/mpeg": true, "audio/wav": true, "audio/ogg": true}
	if !ok || !isBase64 || len(encoded) > 12<<20 || !allowed[mimeType] {
		c.AbortWithStatus(http.StatusUnsupportedMediaType)
		return
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(data) > 8<<20 {
		c.AbortWithStatus(http.StatusRequestEntityTooLarge)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "attachment")
	c.Data(http.StatusOK, mimeType, data)
}

// persistOfficialPluginTask 在结算和返回公开 ID 前持久化版本与原计费来源。
func persistOfficialPluginTask(c *gin.Context, info *relaycommon.RelayInfo, result *relay.TaskSubmitResult) error {
	task := model.InitTask(result.Platform, info)
	task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
	task.PrivateData.Execution = service.TaskExecutionSnapshotFromContext(c)
	task.PrivateData.PluginState = result.PluginResponse.PluginState
	task.PrivateData.PluginData = result.TaskData
	task.PrivateData.PluginImmediate = result.PluginResponse.Immediate
	task.PrivateData.Key = info.ApiKey
	task.PrivateData.BillingSource = info.BillingSource
	task.PrivateData.SubscriptionId = info.SubscriptionId
	task.PrivateData.TokenId = info.TokenId
	task.PrivateData.NodeName = common.NodeName
	task.PrivateData.BillingContext = &model.TaskBillingContext{ModelPrice: info.PriceData.ModelPrice, GroupRatio: info.PriceData.GroupRatioInfo.GroupRatio, OriginModelName: info.OriginModelName, PerCallBilling: true}
	task.Quota = result.Quota
	task.Data = []byte(`{}`)
	task.Action = info.Action
	if err := task.Insert(); err != nil {
		return err
	}
	// 即时终态也交给本地轮询 CAS 与退款认领处理，不能在这里重复退款。
	return nil
}
