package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var passwordLoginProtectionTestSequence atomic.Uint64

func configurePasswordLoginProtection(t *testing.T, useRedis bool, accountLimit, ipLimit int) *miniredis.Miniredis {
	t.Helper()
	previousRedis, previousClient := common.RedisEnabled, common.RDB
	previousEnabled := common.LoginFailureRateLimitEnable
	previousAccountLimit := common.LoginFailureRateLimitNum
	previousIPLimit := common.LoginFailureIPRateLimitNum
	previousDuration := common.LoginFailureRateLimitDuration
	previousInflightLimit := common.LoginInflightIPLimit
	previousLeaseDuration := common.LoginInflightLeaseDuration
	previousSecret := common.SessionSecret
	previousNow := passwordLoginNow
	common.RedisEnabled = useRedis
	common.LoginFailureRateLimitEnable = true
	common.LoginFailureRateLimitNum = accountLimit
	common.LoginFailureIPRateLimitNum = ipLimit
	common.LoginFailureRateLimitDuration = 60
	common.LoginInflightIPLimit = 8
	common.LoginInflightLeaseDuration = 10
	common.SessionSecret = "password-login-rate-limit-test-secret"
	passwordLoginNow = time.Now
	var redisServer *miniredis.Miniredis
	if useRedis {
		redisServer, _ = useRateLimitMiniRedis(t)
	}
	t.Cleanup(func() {
		common.RedisEnabled, common.RDB = previousRedis, previousClient
		common.LoginFailureRateLimitEnable = previousEnabled
		common.LoginFailureRateLimitNum = previousAccountLimit
		common.LoginFailureIPRateLimitNum = previousIPLimit
		common.LoginFailureRateLimitDuration = previousDuration
		common.LoginInflightIPLimit = previousInflightLimit
		common.LoginInflightLeaseDuration = previousLeaseDuration
		common.SessionSecret = previousSecret
		passwordLoginNow = previousNow
	})
	return redisServer
}

func newPasswordLoginContext(t *testing.T, username, clientIP string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/login", nil)
	c.Request.RemoteAddr = clientIP + ":12345"
	return c, recorder
}

func beginPasswordLoginForTest(t *testing.T, username, clientIP string) (*PasswordLoginAttempt, *httptest.ResponseRecorder, bool) {
	t.Helper()
	c, recorder := newPasswordLoginContext(t, username, clientIP)
	attempt, admitted := BeginPasswordLoginAttempt(c, username)
	return attempt, recorder, admitted
}

func assertLoginProtectionResponse(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	require.Equal(t, status, recorder.Code)
	var response struct {
		Code string `json:"code"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, code, response.Code)
}

func TestPasswordLoginFailureLimitsCountOnlyFailuresAndIsolateAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			configurePasswordLoginProtection(t, useRedis, 2, 10)
			clientIP := fmt.Sprintf("192.0.2.%d", 10+passwordLoginProtectionTestSequence.Add(1)%100)

			for range 4 {
				attempt, recorder, admitted := beginPasswordLoginForTest(t, " Alice@Example.com ", clientIP)
				require.True(t, admitted, recorder.Body.String())
				attempt.Release()
			}

			for _, username := range []string{"alice@example.com", " alice@example.com "} {
				attempt, recorder, admitted := beginPasswordLoginForTest(t, username, clientIP)
				require.True(t, admitted, recorder.Body.String())
				c, _ := newPasswordLoginContext(t, username, clientIP)
				require.True(t, attempt.RecordFailure(c))
				attempt.Release()
			}

			attempt, limited, admitted := beginPasswordLoginForTest(t, "alice@example.com", clientIP)
			assert.Nil(t, attempt)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, limited, http.StatusTooManyRequests, "AUTH_LOGIN_RATE_LIMITED")
			assert.NotEmpty(t, limited.Result().Header.Get("Retry-After"))

			caseDistinctAccount, caseDistinctRecorder, admitted := beginPasswordLoginForTest(t, "Alice@example.com", clientIP)
			require.True(t, admitted, caseDistinctRecorder.Body.String())
			caseDistinctAccount.Release()
			otherAccount, otherRecorder, admitted := beginPasswordLoginForTest(t, "bob@example.com", clientIP)
			require.True(t, admitted, otherRecorder.Body.String())
			otherAccount.Release()
			otherIP, otherIPRecorder, admitted := beginPasswordLoginForTest(t, "alice@example.com", "198.51.100.23")
			require.True(t, admitted, otherIPRecorder.Body.String())
			otherIP.Release()
		})
	}
}

func TestPasswordLoginIPFailureLimitStopsRandomAccountKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			configurePasswordLoginProtection(t, useRedis, 10, 2)
			clientIP := fmt.Sprintf("198.51.100.%d", 30+passwordLoginProtectionTestSequence.Add(1)%100)
			for index := range 2 {
				username := fmt.Sprintf("unknown-%d@example.com", index)
				attempt, recorder, admitted := beginPasswordLoginForTest(t, username, clientIP)
				require.True(t, admitted, recorder.Body.String())
				c, _ := newPasswordLoginContext(t, username, clientIP)
				require.True(t, attempt.RecordFailure(c))
				attempt.Release()
			}

			attempt, limited, admitted := beginPasswordLoginForTest(t, "fresh-account@example.com", clientIP)
			assert.Nil(t, attempt)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, limited, http.StatusTooManyRequests, "AUTH_LOGIN_RATE_LIMITED")
		})
	}
}

func TestPasswordLoginAnonymousIdentityDoesNotCollideWithLegalUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			configurePasswordLoginProtection(t, useRedis, 1, 10)
			clientIP := fmt.Sprintf("198.51.100.%d", 80+passwordLoginProtectionTestSequence.Add(1)%100)

			anonymousAttempt, recorder, admitted := beginPasswordLoginForTest(t, "", clientIP)
			require.True(t, admitted, recorder.Body.String())
			c, _ := newPasswordLoginContext(t, "", clientIP)
			require.True(t, anonymousAttempt.RecordFailure(c))
			anonymousAttempt.Release()

			legalAccount, legalRecorder, admitted := beginPasswordLoginForTest(t, "__anonymous__", clientIP)
			require.True(t, admitted, legalRecorder.Body.String())
			legalAccount.Release()
		})
	}
}

func TestPasswordLoginInflightLimitAndTokenSafeRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			redisServer := configurePasswordLoginProtection(t, useRedis, 20, 100)
			clientIP := fmt.Sprintf("203.0.113.%d", 40+passwordLoginProtectionTestSequence.Add(1)%100)
			attempts := make([]*PasswordLoginAttempt, 0, 8)
			for index := range 8 {
				attempt, recorder, admitted := beginPasswordLoginForTest(t, fmt.Sprintf("user-%d", index), clientIP)
				require.True(t, admitted, recorder.Body.String())
				attempts = append(attempts, attempt)
			}

			ninth, busy, admitted := beginPasswordLoginForTest(t, "user-8", clientIP)
			assert.Nil(t, ninth)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, busy, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY")
			assert.Equal(t, "1", busy.Result().Header.Get("Retry-After"))

			attempts[0].Release()
			replacement, replacementRecorder, admitted := beginPasswordLoginForTest(t, "user-8", clientIP)
			require.True(t, admitted, replacementRecorder.Body.String())
			replacement.Release()
			for _, attempt := range attempts[1:] {
				attempt.Release()
			}

			oldAttempt, oldRecorder, admitted := beginPasswordLoginForTest(t, "same-account", clientIP)
			require.True(t, admitted, oldRecorder.Body.String())
			concurrentAttempt, concurrentBusy, admitted := beginPasswordLoginForTest(t, "same-account", clientIP)
			assert.Nil(t, concurrentAttempt)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, concurrentBusy, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY")
			baseNow := passwordLoginNow()
			passwordLoginNow = func() time.Time { return baseNow.Add(11 * time.Second) }
			if useRedis {
				redisServer.FastForward(11 * time.Second)
			}
			newAttempt, newRecorder, admitted := beginPasswordLoginForTest(t, "same-account", clientIP)
			require.True(t, admitted, newRecorder.Body.String())

			oldAttempt.Release()
			blockedByNew, stillBusy, admitted := beginPasswordLoginForTest(t, "same-account", clientIP)
			assert.Nil(t, blockedByNew)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, stillBusy, http.StatusTooManyRequests, "AUTH_LOGIN_BUSY")
			newAttempt.Release()
		})
	}
}

func TestPasswordLoginFailureRecordingIsConcurrentAndPrivate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, useRedis := range []bool{false, true} {
		t.Run(fmt.Sprintf("redis=%t", useRedis), func(t *testing.T) {
			const failures = 8
			configurePasswordLoginProtection(t, useRedis, failures, failures)
			clientIP := fmt.Sprintf("192.0.2.%d", 60+passwordLoginProtectionTestSequence.Add(1)%100)
			attempts := make([]*PasswordLoginAttempt, 0, failures)
			for index := range failures {
				username := fmt.Sprintf("Private-Account-%d@Example.com", index)
				attempt, recorder, admitted := beginPasswordLoginForTest(t, username, clientIP)
				require.True(t, admitted, recorder.Body.String())
				attempts = append(attempts, attempt)
			}

			var waitGroup sync.WaitGroup
			failuresRecorded := make(chan bool, failures)
			for index, attempt := range attempts {
				waitGroup.Add(1)
				go func(index int, attempt *PasswordLoginAttempt) {
					defer waitGroup.Done()
					c, _ := newPasswordLoginContext(t, fmt.Sprintf("Private-Account-%d@Example.com", index), clientIP)
					failuresRecorded <- attempt.RecordFailure(c)
					attempt.Release()
				}(index, attempt)
			}
			waitGroup.Wait()
			close(failuresRecorded)
			for recorded := range failuresRecorded {
				assert.True(t, recorded)
			}

			next, limited, admitted := beginPasswordLoginForTest(t, "new-random-account", clientIP)
			assert.Nil(t, next)
			assert.False(t, admitted)
			assertLoginProtectionResponse(t, limited, http.StatusTooManyRequests, "AUTH_LOGIN_RATE_LIMITED")

			if useRedis {
				keys, err := common.RDB.Keys(context.Background(), redisRateLimitNamespace+":login:*").Result()
				require.NoError(t, err)
				require.NotEmpty(t, keys)
				for _, key := range keys {
					assert.NotContains(t, strings.ToLower(key), "private-account")
					assert.NotContains(t, strings.ToLower(key), "example.com")
				}
			}
		})
	}
}

func TestPasswordLoginRedisAdmissionFailureIsFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePasswordLoginProtection(t, true, 5, 30)
	require.NoError(t, common.RDB.Close())

	attempt, recorder, admitted := beginPasswordLoginForTest(t, "user@example.com", "192.0.2.240")
	assert.Nil(t, attempt)
	assert.False(t, admitted)
	assertLoginProtectionResponse(t, recorder, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE")
}

func TestPasswordLoginNilRedisClientIsFailClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePasswordLoginProtection(t, false, 5, 30)
	common.RedisEnabled = true
	common.RDB = nil

	attempt, recorder, admitted := beginPasswordLoginForTest(t, "user@example.com", "192.0.2.241")
	assert.Nil(t, attempt)
	assert.False(t, admitted)
	assertLoginProtectionResponse(t, recorder, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE")

	fakeAttempt := &PasswordLoginAttempt{redis: true, active: true}
	c, failureRecorder := newPasswordLoginContext(t, "user@example.com", "192.0.2.241")
	assert.False(t, fakeAttempt.RecordFailure(c))
	assertLoginProtectionResponse(t, failureRecorder, http.StatusServiceUnavailable, "AUTH_LOGIN_UNAVAILABLE")
	assert.NotPanics(t, fakeAttempt.Release)
}

func TestPasswordLoginMemoryCleanupAtOperationBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePasswordLoginProtection(t, false, 1, 30)
	now := time.Now()
	passwordLoginNow = func() time.Time { return now }
	clientIP := "192.0.2.242"
	accountFailureKey, _, _, _ := passwordLoginKeys(clientIP, "expired@example.com")
	unrelatedKey := "unrelated-expired-counter"

	passwordLoginMemory.Lock()
	passwordLoginMemory.operations = 255
	passwordLoginMemory.counters[accountFailureKey] = passwordLoginCounter{count: 1, expiresAt: now.Add(-time.Second)}
	passwordLoginMemory.counters[unrelatedKey] = passwordLoginCounter{count: 1, expiresAt: now.Add(-time.Second)}
	passwordLoginMemory.Unlock()

	attempt, recorder, admitted := beginPasswordLoginForTest(t, "expired@example.com", clientIP)
	require.True(t, admitted, recorder.Body.String())
	attempt.Release()

	passwordLoginMemory.Lock()
	_, unrelatedExists := passwordLoginMemory.counters[unrelatedKey]
	passwordLoginMemory.Unlock()
	assert.False(t, unrelatedExists, "the 256th operation should sweep unrelated expired entries")
}

func TestPasswordLoginMemoryFailureRecordResetsExpiredWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configurePasswordLoginProtection(t, false, 2, 10)
	baseNow := time.Now()
	passwordLoginNow = func() time.Time { return baseNow }
	clientIP := "192.0.2.243"

	attempt, recorder, admitted := beginPasswordLoginForTest(t, "window@example.com", clientIP)
	require.True(t, admitted, recorder.Body.String())
	c, _ := newPasswordLoginContext(t, "window@example.com", clientIP)
	require.True(t, attempt.RecordFailure(c))

	passwordLoginNow = func() time.Time { return baseNow.Add(61 * time.Second) }
	require.True(t, attempt.RecordFailure(c))
	attempt.Release()

	next, nextRecorder, admitted := beginPasswordLoginForTest(t, "window@example.com", clientIP)
	require.True(t, admitted, nextRecorder.Body.String(), "expired failures must not carry into the new window")
	next.Release()
}
