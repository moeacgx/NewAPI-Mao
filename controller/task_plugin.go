package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func ListTaskPlugins(c *gin.Context) {
	rows, err := model.ListTaskPlugins()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	seen := make(map[string]bool)
	for _, row := range rows {
		if seen[row.Key] {
			continue
		}
		seen[row.Key] = true
		selected, err := model.GetTaskPluginVersion(row.Key, "")
		if err != nil {
			common.ApiError(c, err)
			return
		}
		item, err := taskPluginManagementView(selected, false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		items = append(items, item)
	}
	common.ApiSuccess(c, items)
}

func GetTaskPlugin(c *gin.Context) {
	row, err := model.GetTaskPluginVersion(c.Param("key"), c.Query("version"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	item, err := taskPluginManagementView(row, c.GetInt("role") >= common.RoleRootUser)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

// taskPluginManagementView 是两套管理模板共享的元数据契约，不让前端解释源码。
func taskPluginManagementView(row *model.TaskPlugin, includeSource bool) (gin.H, error) {
	loaded, err := service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, APIVersion: row.APIVersion, SourceHash: row.SourceHash, SourceKind: "builtin"})
	if err != nil {
		return nil, err
	}
	channels, inFlight, err := model.GetTaskPluginUsage(row.Key)
	if err != nil {
		return nil, err
	}
	item := gin.H{"key": row.Key, "version": row.Version, "api_version": row.APIVersion, "source_hash": row.SourceHash, "enabled": row.Enabled, "active": row.Active, "source_kind": "builtin", "meta": loaded.Meta, "channel_count": len(channels), "in_flight_count": inFlight}
	if includeSource {
		item["source"] = row.Source
	}
	return item, nil
}

func GetTaskPluginVersions(c *gin.Context) {
	rows, err := model.ListTaskPluginVersions(c.Param("key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for i := range rows {
		rows[i].Source = ""
	}
	common.ApiSuccess(c, rows)
}

func GetTaskPluginRuntime(c *gin.Context) {
	enabled, err := service.TaskPluginsEnabled()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"enabled": enabled, "channel_type": constant.ChannelTypeTaskPlugin, "builtin_only": true, "billing": "per_call", "resource_access": "authenticated_inline_only", "production_ready": false})
}

func SetTaskPluginRuntime(c *gin.Context) {
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.Enabled == nil {
		common.ApiErrorMsg(c, "必须提供 enabled 布尔值")
		return
	}
	if err := model.UpdateOption(setting.TaskPluginEnabledKey, strconv.FormatBool(*request.Enabled)); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "task_plugin.runtime", map[string]interface{}{"enabled": *request.Enabled})
	GetTaskPluginRuntime(c)
}

func SetTaskPluginStatus(c *gin.Context) {
	var request struct {
		Enabled *bool `json:"enabled"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.Enabled == nil {
		common.ApiErrorMsg(c, "必须提供 enabled 布尔值")
		return
	}
	if err := model.SetTaskPluginEnabled(c.Param("key"), *request.Enabled); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "task_plugin.status", map[string]interface{}{"key": c.Param("key"), "enabled": *request.Enabled})
	common.ApiSuccess(c, nil)
}

func ActivateTaskPlugin(c *gin.Context) {
	var request struct {
		Version string `json:"version"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil || request.Version == "" {
		common.ApiErrorMsg(c, "必须提供 version")
		return
	}
	row, err := model.GetTaskPluginVersion(c.Param("key"), request.Version)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	_, err = service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, SourceHash: row.SourceHash, SourceKind: "builtin", APIVersion: row.APIVersion})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err = model.ActivateTaskPlugin(row.Key, row.Version); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "task_plugin.activate", map[string]interface{}{"key": row.Key, "version": row.Version, "source_hash": row.SourceHash})
	common.ApiSuccess(c, nil)
}

func GetTaskPluginOptions(c *gin.Context) {
	items := make([]gin.H, 0)
	for _, meta := range jsplugin.DefaultRegistry.Snapshot().Factory {
		items = append(items, gin.H{"key": meta.Key, "name": meta.Name, "version": meta.Version, "models": meta.Models, "channel_type": constant.ChannelTypeTaskPlugin})
	}
	common.ApiSuccess(c, items)
}

// UnsupportedTaskPluginOperation 明确拒绝未验收的源码上传、市场、删除和在线试运行。
func UnsupportedTaskPluginOperation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "message": fmt.Sprintf("本阶段仅支持内置插件管理，尚未开放 %s", c.Request.Method)})
}
