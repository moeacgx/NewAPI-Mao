package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenQueryRateLimitConfiguration(t *testing.T) {
	oldEnable, oldNum, oldDuration := TokenQueryRateLimitEnable, TokenQueryRateLimitNum, TokenQueryRateLimitDuration
	t.Cleanup(func() {
		TokenQueryRateLimitEnable, TokenQueryRateLimitNum, TokenQueryRateLimitDuration = oldEnable, oldNum, oldDuration
	})

	for _, test := range []struct {
		name, enable, count, duration string
		wantEnable                    bool
		wantCount                     int
		wantDuration                  int64
	}{
		{name: "默认", wantEnable: true, wantCount: 20, wantDuration: 1200},
		{name: "自定义", enable: "false", count: "30", duration: "60", wantCount: 30, wantDuration: 60},
		{name: "非正数回退", count: "0", duration: "-1", wantEnable: true, wantCount: 20, wantDuration: 1200},
		{name: "非法值回退", count: "bad", duration: "bad", wantEnable: true, wantCount: 20, wantDuration: 1200},
		{name: "超出内存清理周期", count: "10", duration: "1201", wantEnable: true, wantCount: 10, wantDuration: 1200},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TOKEN_QUERY_RATE_LIMIT_ENABLE", test.enable)
			t.Setenv("TOKEN_QUERY_RATE_LIMIT", test.count)
			t.Setenv("TOKEN_QUERY_RATE_LIMIT_DURATION", test.duration)
			initTokenQueryRateLimitConfig()
			assert.Equal(t, test.wantEnable, TokenQueryRateLimitEnable)
			assert.Equal(t, test.wantCount, TokenQueryRateLimitNum)
			assert.Equal(t, test.wantDuration, TokenQueryRateLimitDuration)
		})
	}
}
