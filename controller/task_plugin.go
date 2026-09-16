package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

func isHex64(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

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
	kind := row.SourceKind
	if kind == "" {
		kind = "builtin"
	}
	loaded, err := service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, APIVersion: row.APIVersion, SourceHash: row.SourceHash, SourceKind: kind})
	if err != nil {
		return nil, err
	}
	channels, inFlight, err := model.GetTaskPluginUsage(row.Key)
	if err != nil {
		return nil, err
	}
	item := gin.H{"key": row.Key, "version": row.Version, "api_version": row.APIVersion, "source_hash": row.SourceHash, "enabled": row.Enabled, "active": row.Active, "source_kind": kind, "meta": loaded.Meta, "channel_count": len(channels), "in_flight_count": inFlight}
	if row.Remark != "" {
		item["remark"] = row.Remark
	}
	if row.Marketplace != "" {
		var provenance any
		if err := common.UnmarshalJsonStr(row.Marketplace, &provenance); err == nil {
			item["marketplace"] = provenance
		}
	}
	if includeSource {
		item["source"] = row.Source
	}
	return item, nil
}

func GetTaskPluginMarketplaceSources(c *gin.Context) {
	sources, err := model.GetTaskPluginMarketplaceSources()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, sources)
}

func SetTaskPluginMarketplaceSources(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)
	var sources []setting.TaskPluginMarketplaceSource
	if err := common.DecodeJson(c.Request.Body, &sources); err != nil || sources == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "marketplace_sources_invalid", "message": "invalid marketplace sources"})
		return
	}
	if err := model.SaveTaskPluginMarketplaceSources(sources); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "marketplace_sources_invalid", "message": "invalid marketplace sources"})
		return
	}
	recordManageAudit(c, "task_plugin.marketplace_sources", map[string]interface{}{"count": len(sources)})
	common.ApiSuccess(c, sources)
}

func GetTaskPluginVersions(c *gin.Context) {
	rows, err := model.ListTaskPluginVersions(c.Param("key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for i := range rows {
		item := gin.H{"id": rows[i].Id, "key": rows[i].Key, "version": rows[i].Version,
			"api_version": rows[i].APIVersion, "source_hash": rows[i].SourceHash,
			"source_kind": rows[i].SourceKind, "enabled": rows[i].Enabled,
			"active": rows[i].Active, "created_at": rows[i].CreatedAt}
		if rows[i].Remark != "" {
			item["remark"] = rows[i].Remark
		}
		if rows[i].Marketplace != "" {
			var provenance any
			if err := common.UnmarshalJsonStr(rows[i].Marketplace, &provenance); err == nil {
				item["marketplace"] = provenance
			}
		}
		items = append(items, item)
	}
	common.ApiSuccess(c, items)
}

func GetTaskPluginRuntime(c *gin.Context) {
	enabled, err := service.TaskPluginsEnabled()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"enabled": enabled, "channel_type": constant.ChannelTypeTaskPlugin, "builtin_only": false, "billing": "per_call", "resource_access": "authenticated_inline_only", "production_ready": false})
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
	kind := row.SourceKind
	if kind == "" {
		kind = "builtin"
	}
	_, err = service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: row.Key, Version: row.Version, SourceHash: row.SourceHash, SourceKind: kind, APIVersion: row.APIVersion})
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
	rows, err := model.ListTaskPlugins()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0)
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
		kind := selected.SourceKind
		if kind == "" {
			kind = "builtin"
		}
		loaded, err := service.LoadPinnedTaskPlugin(&model.TaskPluginSnapshot{Key: selected.Key, Version: selected.Version, APIVersion: selected.APIVersion, SourceHash: selected.SourceHash, SourceKind: kind})
		if err != nil {
			common.ApiError(c, err)
			return
		}
		meta := loaded.Meta
		items = append(items, gin.H{"key": meta.Key, "name": meta.Name, "version": meta.Version, "models": meta.Models, "channel_type": constant.ChannelTypeTaskPlugin})
	}
	common.ApiSuccess(c, items)
}

// UploadTaskPlugin 编译并校验自定义源码后保存为禁用、未激活版本。
func UploadTaskPlugin(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)
	var request struct {
		Source          string `json:"source"`
		SourceSha256    string `json:"sourceSha256"`
		ExpectedKey     string `json:"expectedKey"`
		ExpectedVersion string `json:"expectedVersion"`
		Remark          string `json:"remark"`
		Marketplace     *struct {
			Name     string `json:"name"`
			IndexURL string `json:"index_url"`
			Path     string `json:"path"`
		} `json:"marketplace"`
	}
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "code": "body_too_large", "message": "plugin upload body exceeds 8 MiB"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "invalid_body", "message": "invalid plugin upload body"})
		return
	}
	if strings.TrimSpace(request.Source) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "source_required", "message": "source is required"})
		return
	}
	sourceBytes := []byte(request.Source)
	if len(sourceBytes) > 1<<20 {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"success": false, "code": "source_too_large", "message": "source exceeds 1 MiB"})
		return
	}
	if request.Marketplace == nil && request.SourceSha256 != "" {
		if !isHex64(request.SourceSha256) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "hash_invalid", "message": "sourceSha256 must be 64 hex characters"})
			return
		}
		h := sha256.Sum256(sourceBytes)
		if !strings.EqualFold(hex.EncodeToString(h[:]), request.SourceSha256) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "hash_mismatch", "message": "sourceSha256 does not match source"})
			return
		}
	}
	if request.Marketplace != nil {
		bad := func(code string) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": code, "message": code})
		}
		if request.SourceSha256 == "" || request.ExpectedKey == "" || request.ExpectedVersion == "" {
			bad("marketplace_provenance_required")
			return
		}
		if !isHex64(request.SourceSha256) || !utf8.Valid(sourceBytes) {
			bad("marketplace_hash_invalid")
			return
		}
		h := sha256.Sum256(sourceBytes)
		if !strings.EqualFold(hex.EncodeToString(h[:]), request.SourceSha256) {
			bad("marketplace_hash_mismatch")
			return
		}
		sources, err := model.GetTaskPluginMarketplaceSources()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		found := false
		for _, s := range sources {
			if s.Name == request.Marketplace.Name && s.IndexURL == request.Marketplace.IndexURL {
				found = true
				break
			}
		}
		if !found {
			bad("marketplace_source_invalid")
			return
		}
		u, err := url.Parse(request.Marketplace.Path)
		if err != nil || request.Marketplace.Path == "" || u.IsAbs() || u.Host != "" || u.RawQuery != "" || u.Fragment != "" || strings.HasPrefix(request.Marketplace.Path, "/") || strings.ContainsAny(request.Marketplace.Path, " \t\r\n") || strings.Contains(request.Marketplace.Path, "\\") {
			bad("marketplace_path_invalid")
			return
		}
		for _, seg := range strings.Split(request.Marketplace.Path, "/") {
			dec, e := url.PathUnescape(seg)
			badChar := false
			for _, r := range dec {
				if r < 32 || r == 127 || unicode.IsSpace(r) || strings.ContainsRune(`/\%?#`, r) {
					badChar = true
					break
				}
			}
			if e != nil || seg == "" || dec == "" || dec == ".." || dec == "." || badChar {
				bad("marketplace_path_invalid")
				return
			}
		}
	}
	if utf8.RuneCountInString(request.Remark) > 1024 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "remark_too_long", "message": "remark is too long"})
		return
	}
	loaded, err := jsplugin.CompilePlugin(request.Source, jsplugin.Options{Log: func(string) {}})
	if err != nil {
		// 编译器错误可能包含源码片段或内部堆栈，不返回给客户端。
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "source_compile_invalid", "message": "plugin source failed validation"})
		return
	}
	if request.Marketplace != nil && (loaded.Meta.Key != request.ExpectedKey || loaded.Meta.Version != request.ExpectedVersion) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "code": "marketplace_meta_mismatch", "message": "plugin metadata does not match expected provenance"})
		return
	}
	hash := sha256.Sum256([]byte(request.Source))
	row := &model.TaskPlugin{Key: loaded.Meta.Key, Version: loaded.Meta.Version, APIVersion: loaded.Meta.APIVersion, Source: request.Source, SourceHash: hex.EncodeToString(hash[:]), SourceKind: "custom", Enabled: false, Active: false, Remark: request.Remark}
	if request.Marketplace != nil {
		payload, _ := common.Marshal(request.Marketplace)
		row.Marketplace = string(payload)
	}
	if err := model.SaveTaskPlugin(row); err != nil {
		if strings.Contains(err.Error(), "different source") {
			c.JSON(http.StatusConflict, gin.H{"success": false, "code": "plugin_version_conflict", "message": "plugin version already exists with different source"})
			return
		}
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "task_plugin.upload", map[string]interface{}{"key": row.Key, "version": row.Version, "source_hash": row.SourceHash})
	item, err := taskPluginManagementView(row, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, item)
}

func DeleteTaskPluginVersion(c *gin.Context) {
	result, err := model.DeleteTaskPluginVersion(c.Param("key"), c.Param("version"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "task_plugin.delete", map[string]interface{}{"key": c.Param("key"), "version": c.Param("version")})
	common.ApiSuccess(c, result)
}

// UnsupportedTaskPluginOperation 明确拒绝尚未验收的市场、远程资源和在线试运行。
func UnsupportedTaskPluginOperation(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"success": false, "message": fmt.Sprintf("该任务插件能力尚未开放：%s %s", c.Request.Method, c.Request.URL.Path)})
}
