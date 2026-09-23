package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginFailureRateLimitConfiguration(t *testing.T) {
	previousEnabled := LoginFailureRateLimitEnable
	previousAccountLimit := LoginFailureRateLimitNum
	previousIPLimit := LoginFailureIPRateLimitNum
	previousDuration := LoginFailureRateLimitDuration
	previousInflightLimit := LoginInflightIPLimit
	previousLeaseDuration := LoginInflightLeaseDuration
	t.Cleanup(func() {
		LoginFailureRateLimitEnable = previousEnabled
		LoginFailureRateLimitNum = previousAccountLimit
		LoginFailureIPRateLimitNum = previousIPLimit
		LoginFailureRateLimitDuration = previousDuration
		LoginInflightIPLimit = previousInflightLimit
		LoginInflightLeaseDuration = previousLeaseDuration
	})

	for _, test := range []struct {
		name                                     string
		enabled, account, ip, duration, inflight string
		lease                                    string
		wantEnabled                              bool
		wantAccount, wantIP, wantDuration        int
		wantInflight, wantLease                  int
	}{
		{name: "defaults", wantEnabled: true, wantAccount: 5, wantIP: 30, wantDuration: 60, wantInflight: 8, wantLease: 10},
		{name: "custom", enabled: "false", account: "9", ip: "90", duration: "120", inflight: "4", lease: "20", wantAccount: 9, wantIP: 90, wantDuration: 120, wantInflight: 4, wantLease: 20},
		{name: "invalid values", enabled: "bad", account: "0", ip: "-1", duration: "1201", inflight: "0", lease: "61", wantEnabled: true, wantAccount: 5, wantIP: 30, wantDuration: 60, wantInflight: 8, wantLease: 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LOGIN_FAILURE_RATE_LIMIT_ENABLE", test.enabled)
			t.Setenv("LOGIN_FAILURE_RATE_LIMIT", test.account)
			t.Setenv("LOGIN_FAILURE_IP_RATE_LIMIT", test.ip)
			t.Setenv("LOGIN_FAILURE_RATE_LIMIT_DURATION", test.duration)
			t.Setenv("LOGIN_INFLIGHT_IP_LIMIT", test.inflight)
			t.Setenv("LOGIN_INFLIGHT_LEASE_DURATION", test.lease)

			initLoginFailureRateLimitConfig()

			assert.Equal(t, test.wantEnabled, LoginFailureRateLimitEnable)
			assert.Equal(t, test.wantAccount, LoginFailureRateLimitNum)
			assert.Equal(t, test.wantIP, LoginFailureIPRateLimitNum)
			assert.Equal(t, int64(test.wantDuration), LoginFailureRateLimitDuration)
			assert.Equal(t, test.wantInflight, LoginInflightIPLimit)
			assert.Equal(t, int64(test.wantLease), LoginInflightLeaseDuration)
		})
	}
}
