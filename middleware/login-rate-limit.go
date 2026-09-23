package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const redisPasswordLoginAdmissionScript = `
local account_count = tonumber(redis.call('GET', KEYS[1]) or '0')
if account_count >= tonumber(ARGV[1]) then
  local ttl = redis.call('TTL', KEYS[1])
  if ttl < 1 then redis.call('EXPIRE', KEYS[1], ARGV[8]); ttl = tonumber(ARGV[8]) end
  return {1, ttl}
end
local ip_count = tonumber(redis.call('GET', KEYS[2]) or '0')
if ip_count >= tonumber(ARGV[2]) then
  local ttl = redis.call('TTL', KEYS[2])
  if ttl < 1 then redis.call('EXPIRE', KEYS[2], ARGV[8]); ttl = tonumber(ARGV[8]) end
  return {1, ttl}
end
redis.call('ZREMRANGEBYSCORE', KEYS[4], '-inf', ARGV[3])
if redis.call('ZCARD', KEYS[4]) >= tonumber(ARGV[4]) then return {2, 1} end
if not redis.call('SET', KEYS[3], ARGV[5], 'NX', 'EX', ARGV[6]) then return {2, 1} end
redis.call('ZADD', KEYS[4], ARGV[7], ARGV[5])
local inflight_ttl = redis.call('TTL', KEYS[4])
if inflight_ttl < tonumber(ARGV[6]) then redis.call('EXPIRE', KEYS[4], ARGV[6]) end
return {0, 0}
`

const redisPasswordLoginRecordFailureScript = `
local account_count = redis.call('INCR', KEYS[1])
local account_ttl = redis.call('TTL', KEYS[1])
if account_count == 1 or account_ttl < 0 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local ip_count = redis.call('INCR', KEYS[2])
local ip_ttl = redis.call('TTL', KEYS[2])
if ip_count == 1 or ip_ttl < 0 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
return {account_count, ip_count}
`

const redisPasswordLoginReleaseScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]) end
redis.call('ZREM', KEYS[2], ARGV[1])
return 1
`

type passwordLoginCounter struct {
	count     int
	expiresAt time.Time
}

type passwordLoginLease struct {
	token     string
	expiresAt time.Time
}

var passwordLoginNow = time.Now

var passwordLoginMemory = struct {
	sync.Mutex
	operations    uint64
	counters      map[string]passwordLoginCounter
	accountLeases map[string]passwordLoginLease
	ipLeases      map[string]map[string]time.Time
}{
	counters:      make(map[string]passwordLoginCounter),
	accountLeases: make(map[string]passwordLoginLease),
	ipLeases:      make(map[string]map[string]time.Time),
}

// PasswordLoginAttempt owns one account lease and one slot in the client IP lease set.
type PasswordLoginAttempt struct {
	accountFailureKey string
	ipFailureKey      string
	accountLeaseKey   string
	ipLeaseKey        string
	token             string
	redis             bool
	active            bool
}

func passwordLoginKeys(clientIP, username string) (string, string, string, string) {
	normalizedUsername := strings.TrimSpace(username)
	identity := "account:\x00" + normalizedUsername
	if normalizedUsername == "" {
		identity = "anonymous:\x00"
	}
	digest := common.HmacSha256(identity, common.SessionSecret)
	base := redisRateLimitNamespace + ":login:"
	accountScope := clientIP + ":" + digest
	return base + "failure:account:" + accountScope,
		base + "failure:ip:" + clientIP,
		base + "inflight:account:" + accountScope,
		base + "inflight:ip:" + clientIP
}

// BeginPasswordLoginAttempt checks failure limits and atomically acquires login leases.
func BeginPasswordLoginAttempt(c *gin.Context, username string) (*PasswordLoginAttempt, bool) {
	if !common.LoginFailureRateLimitEnable {
		return &PasswordLoginAttempt{}, true
	}
	accountFailureKey, ipFailureKey, accountLeaseKey, ipLeaseKey := passwordLoginKeys(c.ClientIP(), username)
	now := passwordLoginNow()
	attempt := &PasswordLoginAttempt{
		accountFailureKey: accountFailureKey,
		ipFailureKey:      ipFailureKey,
		accountLeaseKey:   accountLeaseKey,
		ipLeaseKey:        ipLeaseKey,
		token:             uuid.NewString(),
		redis:             common.RedisEnabled,
		active:            true,
	}

	if common.RedisEnabled {
		if common.RDB == nil {
			logger.LogError(c.Request.Context(), "password login admission failed: Redis client is not initialized")
			writePasswordLoginProtectionError(c, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE", i18n.MsgAuthLoginUnavailable, 1)
			return nil, false
		}
		values, err := common.RDB.Eval(
			c.Request.Context(), redisPasswordLoginAdmissionScript,
			[]string{accountFailureKey, ipFailureKey, accountLeaseKey, ipLeaseKey},
			common.LoginFailureRateLimitNum, common.LoginFailureIPRateLimitNum,
			now.Unix(), common.LoginInflightIPLimit, attempt.token,
			common.LoginInflightLeaseDuration,
			now.Add(time.Duration(common.LoginInflightLeaseDuration)*time.Second).Unix(),
			common.LoginFailureRateLimitDuration,
		).Slice()
		if err != nil || len(values) != 2 {
			logger.LogError(c.Request.Context(), fmt.Sprintf("password login admission failed: %v", err))
			writePasswordLoginProtectionError(c, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE", i18n.MsgAuthLoginUnavailable, 1)
			return nil, false
		}
		decision, decisionErr := redisReplyInteger(values[0])
		retryAfter, retryErr := redisReplyInteger(values[1])
		if decisionErr != nil || retryErr != nil {
			writePasswordLoginProtectionError(c, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE", i18n.MsgAuthLoginUnavailable, 1)
			return nil, false
		}
		switch decision {
		case 0:
			return attempt, true
		case 1:
			writePasswordLoginProtectionError(c, http.StatusTooManyRequests, "AUTH_LOGIN_RATE_LIMITED", i18n.MsgAuthLoginRateLimited, retryAfter)
		default:
			writePasswordLoginProtectionError(c, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY", i18n.MsgAuthLoginBusy, 1)
		}
		return nil, false
	}

	passwordLoginMemory.Lock()
	defer passwordLoginMemory.Unlock()
	cleanupPasswordLoginMemory(now)
	for _, limit := range []struct {
		key string
		max int
	}{{accountFailureKey, common.LoginFailureRateLimitNum}, {ipFailureKey, common.LoginFailureIPRateLimitNum}} {
		if counter, ok := passwordLoginMemory.counters[limit.key]; ok {
			if !now.Before(counter.expiresAt) {
				delete(passwordLoginMemory.counters, limit.key)
			} else if counter.count >= limit.max {
				writePasswordLoginProtectionError(c, http.StatusTooManyRequests, "AUTH_LOGIN_RATE_LIMITED", i18n.MsgAuthLoginRateLimited, retryAfterSeconds(now, counter.expiresAt))
				return nil, false
			}
		}
	}
	if lease, ok := passwordLoginMemory.accountLeases[accountLeaseKey]; ok {
		if now.Before(lease.expiresAt) {
			writePasswordLoginProtectionError(c, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY", i18n.MsgAuthLoginBusy, 1)
			return nil, false
		}
		delete(passwordLoginMemory.accountLeases, accountLeaseKey)
	}
	ipLeases := passwordLoginMemory.ipLeases[ipLeaseKey]
	for token, expiresAt := range ipLeases {
		if !now.Before(expiresAt) {
			delete(ipLeases, token)
		}
	}
	if len(ipLeases) >= common.LoginInflightIPLimit {
		writePasswordLoginProtectionError(c, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY", i18n.MsgAuthLoginBusy, 1)
		return nil, false
	}
	expiresAt := now.Add(time.Duration(common.LoginInflightLeaseDuration) * time.Second)
	passwordLoginMemory.accountLeases[accountLeaseKey] = passwordLoginLease{token: attempt.token, expiresAt: expiresAt}
	if ipLeases == nil {
		ipLeases = make(map[string]time.Time)
		passwordLoginMemory.ipLeases[ipLeaseKey] = ipLeases
	}
	ipLeases[attempt.token] = expiresAt
	return attempt, true
}

// RecordFailure records one failed credential attempt in both failure scopes.
func (attempt *PasswordLoginAttempt) RecordFailure(c *gin.Context) bool {
	if attempt == nil || !attempt.active || !common.LoginFailureRateLimitEnable {
		return true
	}
	if attempt.redis {
		if common.RDB == nil {
			logger.LogError(c.Request.Context(), "password login failure recording failed: Redis client is not initialized")
			writePasswordLoginProtectionError(c, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE", i18n.MsgAuthLoginUnavailable, 1)
			return false
		}
		_, err := common.RDB.Eval(
			c.Request.Context(), redisPasswordLoginRecordFailureScript,
			[]string{attempt.accountFailureKey, attempt.ipFailureKey},
			common.LoginFailureRateLimitDuration,
		).Result()
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("password login failure recording failed: %v", err))
			writePasswordLoginProtectionError(c, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE", i18n.MsgAuthLoginUnavailable, 1)
			return false
		}
		return true
	}

	now := passwordLoginNow()
	passwordLoginMemory.Lock()
	defer passwordLoginMemory.Unlock()
	cleanupPasswordLoginMemory(now)
	for _, key := range []string{attempt.accountFailureKey, attempt.ipFailureKey} {
		counter, ok := passwordLoginMemory.counters[key]
		if !ok || !now.Before(counter.expiresAt) {
			counter = passwordLoginCounter{}
			counter.expiresAt = now.Add(time.Duration(common.LoginFailureRateLimitDuration) * time.Second)
		}
		counter.count++
		passwordLoginMemory.counters[key] = counter
	}
	return true
}

// Release gives up only the leases owned by this attempt token.
func (attempt *PasswordLoginAttempt) Release() {
	if attempt == nil || !attempt.active {
		return
	}
	attempt.active = false
	if attempt.redis {
		if common.RDB == nil {
			common.SysError("password login lease release failed: Redis client is not initialized")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := common.RDB.Eval(ctx, redisPasswordLoginReleaseScript,
			[]string{attempt.accountLeaseKey, attempt.ipLeaseKey}, attempt.token).Result(); err != nil {
			common.SysError(fmt.Sprintf("password login lease release failed: %v", err))
		}
		return
	}

	passwordLoginMemory.Lock()
	defer passwordLoginMemory.Unlock()
	if lease, ok := passwordLoginMemory.accountLeases[attempt.accountLeaseKey]; ok && lease.token == attempt.token {
		delete(passwordLoginMemory.accountLeases, attempt.accountLeaseKey)
	}
	if ipLeases := passwordLoginMemory.ipLeases[attempt.ipLeaseKey]; ipLeases != nil {
		delete(ipLeases, attempt.token)
		if len(ipLeases) == 0 {
			delete(passwordLoginMemory.ipLeases, attempt.ipLeaseKey)
		}
	}
}

func cleanupPasswordLoginMemory(now time.Time) {
	passwordLoginMemory.operations++
	if passwordLoginMemory.operations%256 != 0 {
		return
	}
	for key, counter := range passwordLoginMemory.counters {
		if !now.Before(counter.expiresAt) {
			delete(passwordLoginMemory.counters, key)
		}
	}
	for key, lease := range passwordLoginMemory.accountLeases {
		if !now.Before(lease.expiresAt) {
			delete(passwordLoginMemory.accountLeases, key)
		}
	}
	for key, leases := range passwordLoginMemory.ipLeases {
		for token, expiresAt := range leases {
			if !now.Before(expiresAt) {
				delete(leases, token)
			}
		}
		if len(leases) == 0 {
			delete(passwordLoginMemory.ipLeases, key)
		}
	}
}

func retryAfterSeconds(now, expiresAt time.Time) int64 {
	seconds := int64((expiresAt.Sub(now) + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func writePasswordLoginProtectionError(c *gin.Context, status int, code, messageKey string, retryAfter int64) {
	c.Header("Retry-After", strconv.FormatInt(retryAfter, 10))
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"code":    code,
		"message": common.TranslateMessage(c, messageKey, map[string]any{"Seconds": retryAfter}),
	})
}
