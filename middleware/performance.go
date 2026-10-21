package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// SystemPerformanceCheck 检查系统性能中间件
func SystemPerformanceCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅检查 Relay 接口 (/v1, /v1beta 等)
		// 这里简单判断路径前缀，可以根据实际路由调整
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/v1/messages") {
			if err := checkSystemPerformance(); err != nil {
				clientError, statusCode := performanceClientClaudeError(err)
				c.JSON(statusCode, gin.H{"error": clientError})
				c.Abort()
				return
			}
		} else {
			if err := checkSystemPerformance(); err != nil {
				clientError, statusCode := performanceClientOpenAIError(err)
				c.JSON(statusCode, gin.H{"error": clientError})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func performanceClientOpenAIError(apiErr *types.NewAPIError) (types.OpenAIError, int) {
	clientErr := apiErr.ToOpenAIError()
	message := clientErr.Message
	clientStatus := apiErr.StatusCode
	if types.IsUpstreamReturnedError(apiErr) {
		message, clientStatus, _ = common.ReplaceClientErrorCandidates(apiErr.StatusCode, apiErr.Error(), clientErr.Message)
	}
	clientErr.Message = message
	return clientErr, clientStatus
}

func performanceClientClaudeError(apiErr *types.NewAPIError) (types.ClaudeError, int) {
	clientErr := apiErr.ToClaudeError()
	message := clientErr.Message
	clientStatus := apiErr.StatusCode
	if types.IsUpstreamReturnedError(apiErr) {
		message, clientStatus, _ = common.ReplaceClientErrorCandidates(apiErr.StatusCode, apiErr.Error(), clientErr.Message)
	}
	clientErr.Message = message
	return clientErr, clientStatus
}

// checkSystemPerformance 检查系统性能是否超过阈值
func checkSystemPerformance() *types.NewAPIError {
	// 过载标记在采样侧完成：CPU / 内存按容器 cgroup 配额计算，并要求连续多次
	// 超阈值；判定时再按当前阈值复核，并忽略过期采样。
	overload := common.CheckSystemOverload(common.GetSystemStatus(), common.GetPerformanceMonitorConfig(), time.Now())
	if overload == nil {
		return nil
	}
	return types.NewErrorWithStatusCode(overload, types.ErrorCode(overload.Code), http.StatusServiceUnavailable)
}
