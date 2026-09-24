package jsplugin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/logger"
)

// PerformanceFailureInput 只包含性能分类所需的路由身份，不携带请求或供应商正文。
type PerformanceFailureInput struct {
	Model, UpstreamModel, RequestPath, Method string
}

type PerformanceFailure struct {
	Stage      string
	HTTPStatus int
	ErrorCode  string
}

// ShouldRecordPerformanceFailure 仅接受严格布尔 false 作为排除指令。
// 排队和执行共用100ms预算；任何钩子故障都沿用宿主原有统计规则。
func (p *LoadedPlugin) ShouldRecordPerformanceFailure(ctx context.Context, input PerformanceFailureInput, failure PerformanceFailure) bool {
	if p == nil || p.Engine == nil || !p.PerformanceFailureFilter {
		return true
	}
	boundedCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	// 错误码是上游可控字段。只接受有界标识符，避免把异常正文当作错误码传入。
	code := failure.ErrorCode
	if len(code) > 128 || strings.IndexFunc(code, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' || r == ':')
	}) >= 0 {
		code = "invalid_error_code"
	}
	value, err := p.Engine.CallBoolean(boundedCtx, "shouldRecordPerformanceFailure", map[string]any{
		"pluginKey": p.Meta.Key, "pluginVersion": p.Meta.Version,
		"model": input.Model, "upstreamModel": input.UpstreamModel,
		"requestPath": input.RequestPath, "method": input.Method,
	}, map[string]any{
		"stage": failure.Stage, "httpStatus": failure.HTTPStatus, "errorCode": code,
	})
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("task_plugin event=performance_filter_fallback plugin=%q version=%q reason=hook_failed", p.Meta.Key, p.Meta.Version))
		return true
	}
	if value == nil {
		return true
	}
	record, valid := value.(bool)
	if !valid {
		logger.LogWarn(ctx, fmt.Sprintf("task_plugin event=performance_filter_fallback plugin=%q version=%q reason=invalid_result", p.Meta.Key, p.Meta.Version))
		return true
	}
	return record
}
