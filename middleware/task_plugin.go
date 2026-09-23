package middleware

import (
	"mime"
	"net/http"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// TaskPluginRequest 只标记显式插件入口，模型、令牌分组和渠道选择仍由本地中间件处理。
func TaskPluginRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.Param("plugin_key")
		if !jsplugin.ValidPluginKey(key) {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
			c.AbortWithStatus(http.StatusUnsupportedMediaType)
			return
		}
		loaded, pin, err := service.ActiveTaskPlugin(key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "官方任务插件未启用或不可用"})
			return
		}
		var body map[string]any
		if err := common.UnmarshalBodyReusable(c, &body); err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		modelName, _ := body["model"].(string)
		if strings.TrimSpace(modelName) == "" {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.Set("official_task_plugin", loaded)
		c.Set("task_plugin_snapshot", pin)
		c.Set("platform", key)
		c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
		c.Next()
	}
}

// PrepareTaskPluginRoute 复用上游原生解码合同，并保留本地活动版本与源码快照门禁。
// 当前仅开放 JSON submit；查询和动态路由仍使用现有通用任务读取入口。
func PrepareTaskPluginRoute() gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(jsplugin.ContextKeyPinnedRoute)
		pinned, ok := value.(jsplugin.PinnedRoute)
		if !exists || !ok || pinned.Plugin == nil {
			abortTaskPluginRouteErrorDetail(c, http.StatusInternalServerError, "")
			return
		}
		loaded, pin, err := service.ActiveTaskPlugin(pinned.Plugin.Meta.Key)
		if err != nil || loaded != pinned.Plugin {
			// 失效版本也不能通过错误渲染 Hook 执行。
			c.Set(jsplugin.ContextKeyPinnedRoute, jsplugin.PinnedRoute{})
			abortTaskPluginRouteErrorDetail(c, http.StatusServiceUnavailable, "")
			return
		}
		if pinned.Route.Type != jsplugin.RouteTypeSubmit {
			abortTaskPluginRouteErrorDetail(c, http.StatusNotImplemented, "")
			return
		}
		requestContext := jsplugin.RouteRequestContext{
			Path: c.Request.URL.Path, Method: c.Request.Method,
			Params: make(map[string]string, len(c.Params)), Query: c.Request.URL.Query(),
		}
		for _, param := range c.Params {
			requestContext.Params[param.Key] = param.Value
		}
		c.Set(jsplugin.ContextKeyRouteRequest, requestContext)
		canonical := ""
		for _, contentType := range c.Request.Header.Values("Content-Type") {
			mediaType, params, parseErr := mime.ParseMediaType(contentType)
			if parseErr != nil || (mediaType != "application/json" && !strings.HasSuffix(mediaType, "+json")) {
				abortTaskPluginRouteErrorDetail(c, http.StatusUnsupportedMediaType, "")
				return
			}
			current := mime.FormatMediaType(strings.ToLower(mediaType), params)
			if canonical != "" && current != canonical {
				abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
				return
			}
			canonical = current
		}
		if canonical == "" {
			abortTaskPluginRouteErrorDetail(c, http.StatusUnsupportedMediaType, "")
			return
		}
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		raw, err := storage.Bytes()
		if err != nil || !utf8.Valid(raw) {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		var body map[string]any
		if err := common.Unmarshal(raw, &body); err != nil || body == nil {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		requestContext.Body = map[string]any{"kind": string(jsplugin.BodyJSON), "value": body}
		requestContext.RequestBody = body
		c.Set(jsplugin.ContextKeyRouteRequest, requestContext)
		claimedModel, _ := body["model"].(string)
		if len(pinned.Route.Models) > 0 && !slices.Contains(pinned.Route.Models, claimedModel) {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		resolvedValue, err := loaded.Engine.CallMember(c.Request.Context(), "native", pinned.Route.Decode, requestContext.JSValue())
		if err != nil {
			// 插件异常可能包含供应商凭据或源码，不传播异常原文。
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		resolved, ok := resolvedValue.(map[string]any)
		if !ok || resolved["kind"] != string(jsplugin.RouteTypeSubmit) {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		if _, forbidden := resolved["renderer"]; forbidden {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		// 尚未迁移原生源任务 intent 合同，不能静默丢弃后继续提交。
		if _, present := resolved["originTaskIds"]; present {
			abortTaskPluginRouteErrorDetail(c, http.StatusNotImplemented, "")
			return
		}
		modelName, valid := resolved["model"].(string)
		if !valid || strings.TrimSpace(modelName) == "" || !slices.Contains(loaded.Meta.Models, modelName) || (len(pinned.Route.Models) > 0 && !slices.Contains(pinned.Route.Models, modelName)) {
			abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
			return
		}
		action := pinned.Route.Action
		if value, present := resolved["action"]; present {
			resolvedAction, valid := value.(string)
			if !valid {
				abortTaskPluginRouteErrorDetail(c, http.StatusBadRequest, "")
				return
			}
			action = jsplugin.ResolveRouteAction(pinned.Route, resolvedAction)
		}
		if replacement, present := resolved["requestBody"]; present {
			requestContext.RequestBody = replacement
		}
		c.Set(jsplugin.ContextKeyRouteRequest, requestContext)
		c.Set(jsplugin.ContextKeyPinnedPlugin, jsplugin.PinnedPlugin{Generation: pinned.Generation, Plugin: loaded})
		c.Set("official_task_plugin", loaded)
		c.Set("task_plugin_snapshot", pin)
		c.Set("task_request", requestContext.RequestBody)
		c.Set("resolved_task_model", modelName)
		c.Set("expected_task_plugin_key", loaded.Meta.Key)
		c.Set("task_plugin_key", loaded.Meta.Key)
		c.Set("platform", loaded.Meta.Key)
		c.Set("task_action", action)
		c.Set("relay_mode", relayconstant.RelayModeVideoSubmit)
		c.Next()
	}
}

func RespondTaskPluginError(c *gin.Context, taskErr *dto.TaskError) bool {
	if taskErr == nil {
		return false
	}
	pinnedValue, exists := c.Get(jsplugin.ContextKeyPinnedRoute)
	pinned, ok := pinnedValue.(jsplugin.PinnedRoute)
	if !exists || !ok || pinned.Plugin == nil {
		return false
	}
	sanitized := sanitizedTaskPluginError(taskErr.StatusCode, "")
	requestID := c.GetString(common.RequestIdKey)
	hasRenderer, err := pinned.Plugin.Engine.HasCallablePath(c.Request.Context(), "native", "error")
	requestValue, exists := c.Get(jsplugin.ContextKeyRouteRequest)
	requestContext, ok := requestValue.(jsplugin.RouteRequestContext)
	if err == nil && hasRenderer && exists && ok {
		body, callErr := pinned.Plugin.Engine.CallMember(c.Request.Context(), "native", "error", requestContext.JSValue(), map[string]any{
			"code":       sanitized.Code,
			"message":    sanitized.Message,
			"httpStatus": sanitized.HTTPStatus,
			"retryable":  sanitized.Retryable,
			"requestId":  requestID,
		})
		if callErr == nil {
			c.JSON(sanitized.HTTPStatus, body)
			return true
		}
	}
	message := sanitized.Message
	if requestID != "" {
		message = common.MessageWithRequestId(sanitized.Message, requestID)
	}
	c.JSON(sanitized.HTTPStatus, &dto.TaskError{
		Code:       sanitized.Code,
		Message:    message,
		StatusCode: sanitized.HTTPStatus,
	})
	return true
}

func abortTaskPluginRouteError(c *gin.Context, status int) {
	abortTaskPluginRouteErrorDetail(c, status, "")
}

func abortTaskPluginRouteErrorDetail(c *gin.Context, status int, detail string) {
	taskErr := sanitizedTaskPluginError(status, detail)
	c.Abort()
	if RespondTaskPluginError(c, &dto.TaskError{Code: taskErr.Code, Message: detail, StatusCode: taskErr.HTTPStatus}) {
		return
	}
	message := taskErr.Message
	if requestID := c.GetString(common.RequestIdKey); requestID != "" {
		message = common.MessageWithRequestId(taskErr.Message, requestID)
	}
	c.JSON(taskErr.HTTPStatus, &dto.TaskError{
		Code:       taskErr.Code,
		Message:    message,
		StatusCode: taskErr.HTTPStatus,
	})
}

func sanitizedTaskPluginError(status int, detail string) dto.TaskPluginError {
	var taskErr dto.TaskPluginError
	switch status {
	case http.StatusBadRequest:
		taskErr = dto.TaskPluginError{Code: "invalid_request", Message: "Invalid request", HTTPStatus: status}
	case http.StatusUnauthorized:
		taskErr = dto.TaskPluginError{Code: "authentication_error", Message: "Authentication failed", HTTPStatus: status}
	case http.StatusForbidden:
		taskErr = dto.TaskPluginError{Code: "permission_denied", Message: "Access denied", HTTPStatus: status}
	case http.StatusNotFound:
		taskErr = dto.TaskPluginError{Code: "task_not_found", Message: "Task not found", HTTPStatus: status}
	case http.StatusConflict:
		taskErr = dto.TaskPluginError{Code: "request_conflict", Message: "Request conflict", HTTPStatus: status}
	case http.StatusTooManyRequests:
		taskErr = dto.TaskPluginError{Code: "rate_limit_exceeded", Message: "Too many requests", HTTPStatus: status, Retryable: true}
	default:
		if status < 400 || status > 599 {
			status = http.StatusInternalServerError
		}
		if status < 500 {
			taskErr = dto.TaskPluginError{Code: "invalid_request", Message: "Invalid request", HTTPStatus: status}
		} else {
			taskErr = dto.TaskPluginError{Code: "server_error", Message: "Task request failed", HTTPStatus: status, Retryable: status >= 500}
		}
	}
	if detail != "" && taskErr.HTTPStatus < 500 {
		taskErr.Message = detail
	}
	return taskErr
}
