package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTokenPeriodForTest(t *testing.T, id int, period string, limit int64) {
	t.Helper()
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", id).Updates(map[string]interface{}{
		"quota_period":       period,
		"quota_period_limit": limit,
	}).Error)
}

func TestTokenQuotaPeriodWindowBoundaries(t *testing.T) {
	utc := time.UTC
	tests := []struct {
		name        string
		period      string
		now         time.Time
		wantStart   time.Time
		wantResetAt time.Time
		wantOK      bool
	}{
		{"daily mid day", TokenQuotaPeriodDaily, time.Date(2026, 11, 25, 10, 30, 0, 0, utc), time.Date(2026, 11, 25, 0, 0, 0, 0, utc), time.Date(2026, 11, 26, 0, 0, 0, 0, utc), true},
		{"weekly wednesday starts monday", TokenQuotaPeriodWeekly, time.Date(2026, 11, 25, 10, 30, 0, 0, utc), time.Date(2026, 11, 23, 0, 0, 0, 0, utc), time.Date(2026, 11, 30, 0, 0, 0, 0, utc), true},
		{"weekly sunday still previous monday", TokenQuotaPeriodWeekly, time.Date(2026, 11, 29, 23, 59, 59, 0, utc), time.Date(2026, 11, 23, 0, 0, 0, 0, utc), time.Date(2026, 11, 30, 0, 0, 0, 0, utc), true},
		{"weekly monday starts new window", TokenQuotaPeriodWeekly, time.Date(2026, 11, 30, 0, 0, 0, 0, utc), time.Date(2026, 11, 30, 0, 0, 0, 0, utc), time.Date(2026, 12, 7, 0, 0, 0, 0, utc), true},
		{"none has no window", TokenQuotaPeriodNone, time.Date(2026, 11, 25, 0, 0, 0, 0, utc), time.Time{}, time.Time{}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start, resetAt, ok := TokenQuotaPeriodWindow(test.period, test.now)
			assert.Equal(t, test.wantOK, ok)
			if test.wantOK {
				assert.Equal(t, test.wantStart.Unix(), start)
				assert.Equal(t, test.wantResetAt.Unix(), resetAt)
			}
		})
	}
}

func TestNormalizeTokenQuotaPeriod(t *testing.T) {
	for input, want := range map[string]string{"": "none", " None ": "none", "DAILY": "daily", "weekly": "weekly"} {
		got, ok := NormalizeTokenQuotaPeriod(input)
		assert.True(t, ok, input)
		assert.Equal(t, want, got, input)
	}
	_, ok := NormalizeTokenQuotaPeriod("monthly")
	assert.False(t, ok)
}

func TestTryReserveTokenPeriodQuotaStopsAtLimitAndRefundsNeverGoNegative(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodWeekly, 100)

	reserved, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 60)
	require.NoError(t, err)
	assert.True(t, reserved)
	reserved, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 41)
	require.NoError(t, err)
	assert.False(t, reserved, "a request that would pass the weekly limit must be refused")
	assert.EqualValues(t, 60, getTokenFromDB(t, token.Id).QuotaPeriodUsed, "a refused request must not change usage")

	reserved, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 40)
	require.NoError(t, err)
	assert.True(t, reserved, "usage may reach exactly the limit")

	require.NoError(t, AdjustTokenPeriodUsage(token.Id, TokenQuotaPeriodWeekly, -30))
	assert.EqualValues(t, 70, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
	require.NoError(t, AdjustTokenPeriodUsage(token.Id, TokenQuotaPeriodWeekly, -500))
	assert.EqualValues(t, 0, getTokenFromDB(t, token.Id).QuotaPeriodUsed, "refund is floored at zero")
}

func TestTokenPeriodQuotaRestartsInNewWindow(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 100)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]interface{}{
		"quota_period_used":   100,
		"quota_period_window": 1,
	}).Error)

	reserved, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 60)
	require.NoError(t, err)
	assert.True(t, reserved, "an expired window must not keep the old usage")
	window, _, _ := TokenQuotaPeriodWindow(TokenQuotaPeriodDaily, time.Now())
	stored := getTokenFromDB(t, token.Id)
	assert.EqualValues(t, 60, stored.QuotaPeriodUsed)
	assert.Equal(t, window, stored.QuotaPeriodWindow)

	stored.QuotaPeriodWindow = 1
	stored.QuotaPeriodUsed = 100
	stored.FillQuotaPeriodState(time.Now())
	assert.EqualValues(t, 0, stored.QuotaPeriodUsed, "read view shows zero for a stale window")
	assert.Greater(t, stored.QuotaPeriodResetAt, time.Now().Unix())
}

func TestTokenPeriodQuotaAppliesToUnlimitedTotalQuota(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 0)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("unlimited_quota", true).Error)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 500)

	reserved, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 500)
	require.NoError(t, err)
	assert.True(t, reserved)
	reserved, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 1)
	require.NoError(t, err)
	assert.False(t, reserved, "unlimited total quota must not disable the period limit")
}

func TestTokenWithoutPeriodIsNotLimitedAndStaleSnapshotIsTolerated(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)

	reserved, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodNone, 999999)
	require.NoError(t, err)
	assert.True(t, reserved)
	// The caller's snapshot says daily, but the stored token has no period any more.
	reserved, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 999999)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.EqualValues(t, 0, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
}

func TestAdjustTokenPeriodUsageWithUnknownPeriodTouchesOnlyActivePeriod(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodWeekly, 100)

	require.NoError(t, AdjustTokenPeriodUsage(token.Id, "", 40))
	assert.EqualValues(t, 40, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
	require.NoError(t, AdjustTokenPeriodUsage(token.Id, "", -15))
	assert.EqualValues(t, 25, getTokenFromDB(t, token.Id).QuotaPeriodUsed)

	plain := createReserveTestToken(t, 1000)
	require.NoError(t, AdjustTokenPeriodUsage(plain.Id, "", 40))
	assert.EqualValues(t, 0, getTokenFromDB(t, plain.Id).QuotaPeriodUsed, "tokens without a period keep no ledger")
}
