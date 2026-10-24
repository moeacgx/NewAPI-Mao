package service

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
)

type gatewaySkillLimits struct {
	maxConcurrent   int
	maxContextBytes int64
}

type gatewaySkillLimiter struct {
	mu           sync.Mutex
	active       int
	contextBytes int64
}

// GatewaySkillAdmission 持有整个客户端请求的技能额度，包含正式回答及跨组重试。
type GatewaySkillAdmission struct {
	limiter *gatewaySkillLimiter
	bytes   int64
	once    sync.Once
}

var gatewaySkillRequests gatewaySkillLimiter

func gatewaySkillPositiveEnv(name string, fallback, maximum int64) int64 {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 || value > maximum {
		common.SysError(fmt.Sprintf("%s 必须为 1 至 %d 的整数，使用默认值 %d", name, maximum, fallback))
		return fallback
	}
	return value
}

func gatewaySkillResourceLimits() gatewaySkillLimits {
	return gatewaySkillLimits{
		maxConcurrent:   int(gatewaySkillPositiveEnv("GATEWAY_SKILLS_MAX_CONCURRENT", 16, 65536)),
		maxContextBytes: gatewaySkillPositiveEnv("GATEWAY_SKILLS_MAX_INFLIGHT_CONTEXT_BYTES", 32*1024*1024, 1<<40),
	}
}

// GatewaySkillDiscoveryTimeout 仅约束私有读取阶段，不缩短正式回答的客户端期限。
func GatewaySkillDiscoveryTimeout() time.Duration {
	return time.Duration(gatewaySkillPositiveEnv("GATEWAY_SKILLS_DISCOVERY_TIMEOUT_SECONDS", 120, 3600)) * time.Second
}

func TryAcquireGatewaySkillRequest(contextBytes int64) (*GatewaySkillAdmission, *types.NewAPIError) {
	limits := gatewaySkillResourceLimits()
	return gatewaySkillRequests.acquire(contextBytes, limits.maxConcurrent, limits.maxContextBytes)
}

func (limiter *gatewaySkillLimiter) acquire(contextBytes int64, maxConcurrent int, maxBytes int64) (*GatewaySkillAdmission, *types.NewAPIError) {
	if contextBytes < 0 || contextBytes > maxBytes {
		return nil, types.NewErrorWithStatusCode(errors.New("请求正文超过云端技能在途上下文预算，请缩小请求或联系管理员调整预算"),
			types.ErrorCode("gateway_skills_context_too_large"), http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
	}
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	// 使用减法比较，避免多个大请求求和时溢出；检查与占用必须在同一锁内。
	if limiter.active >= maxConcurrent || contextBytes > maxBytes-limiter.contextBytes {
		return nil, types.NewErrorWithStatusCode(errors.New("云端技能请求繁忙，请稍后重试"),
			types.ErrorCode("gateway_skills_busy"), http.StatusTooManyRequests, types.ErrOptionWithSkipRetry())
	}
	limiter.active++
	limiter.contextBytes += contextBytes
	return &GatewaySkillAdmission{limiter: limiter, bytes: contextBytes}, nil
}

func (admission *GatewaySkillAdmission) Release() {
	if admission == nil {
		return
	}
	admission.once.Do(func() {
		admission.limiter.mu.Lock()
		defer admission.limiter.mu.Unlock()
		admission.limiter.active--
		admission.limiter.contextBytes -= admission.bytes
	})
}

// IsGatewaySkillResourceError 区分本地资源保护与渠道故障，避免重试、自动封禁或污染渠道质量。
func IsGatewaySkillResourceError(apiErr *types.NewAPIError) bool {
	if apiErr == nil {
		return false
	}
	switch apiErr.GetErrorCode() {
	case "gateway_skills_busy", "gateway_skills_context_too_large", "gateway_skills_timeout":
		return true
	default:
		return false
	}
}
