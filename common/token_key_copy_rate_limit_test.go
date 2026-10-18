package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenKeyCopyRateLimitConfiguration(t *testing.T) {
	oldEnable, oldNum, oldDuration := TokenKeyReadRateLimitEnable, TokenKeyReadRateLimitNum, TokenKeyReadRateLimitDuration
	t.Cleanup(func() {
		TokenKeyReadRateLimitEnable, TokenKeyReadRateLimitNum, TokenKeyReadRateLimitDuration = oldEnable, oldNum, oldDuration
	})
	for _, test := range []struct {
		name, enable, count, duration string
		wantEnable                    bool
		wantCount                     int
		wantDuration                  int64
	}{
		{name: "默认", wantEnable: true, wantCount: 60, wantDuration: 60},
		{name: "自定义", enable: "false", count: "120", duration: "1200", wantCount: 120, wantDuration: 1200},
		{name: "非正数回退", count: "0", duration: "-1", wantEnable: true, wantCount: 60, wantDuration: 60},
		{name: "非法值回退", count: "bad", duration: "bad", wantEnable: true, wantCount: 60, wantDuration: 60},
		{name: "超出内存清理周期", count: "10", duration: "1201", wantEnable: true, wantCount: 10, wantDuration: 60},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TOKEN_KEY_READ_RATE_LIMIT_ENABLE", test.enable)
			t.Setenv("TOKEN_KEY_READ_RATE_LIMIT", test.count)
			t.Setenv("TOKEN_KEY_READ_RATE_LIMIT_DURATION", test.duration)
			initTokenKeyReadRateLimitConfig()
			assert.Equal(t, test.wantEnable, TokenKeyReadRateLimitEnable)
			assert.Equal(t, test.wantCount, TokenKeyReadRateLimitNum)
			assert.Equal(t, test.wantDuration, TokenKeyReadRateLimitDuration)
		})
	}
}
