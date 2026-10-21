package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseObserverDoesNotResetBillingFirstResponse(t *testing.T) {
	var observed []time.Time
	info := &RelayInfo{isFirstResponse: true, StartTime: time.Now().Add(-time.Second)}
	info.OnResponseData = func(at time.Time) { observed = append(observed, at) }
	info.SetFirstResponseTime()
	first := info.FirstResponseTime
	info.SetFirstResponseTime()
	require.Len(t, observed, 2)
	assert.Equal(t, first, info.FirstResponseTime)
	assert.True(t, info.HasSendResponse())
	assert.False(t, observed[1].Before(observed[0]))
}
