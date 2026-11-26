package model

import (
	"sync"
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

	stamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 60)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.False(t, stamp.IsZero())
	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 41)
	require.NoError(t, err)
	assert.False(t, allowed, "a request that would pass the weekly limit must be refused")
	assert.EqualValues(t, 60, getTokenFromDB(t, token.Id).QuotaPeriodUsed, "a refused request must not change usage")

	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 40)
	require.NoError(t, err)
	assert.True(t, allowed, "usage may reach exactly the limit")

	applied, err := RefundTokenPeriodQuota(nil, token.Id, stamp, 30)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.EqualValues(t, 70, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
	_, err = RefundTokenPeriodQuota(nil, token.Id, stamp, 500)
	require.NoError(t, err)
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

	_, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 60)
	require.NoError(t, err)
	assert.True(t, allowed, "an expired window must not keep the old usage")
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

	_, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 500)
	require.NoError(t, err)
	assert.True(t, allowed)
	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 1)
	require.NoError(t, err)
	assert.False(t, allowed, "unlimited total quota must not disable the period limit")
}

func TestTokenWithoutPeriodIsNotLimitedAndStaleSnapshotIsTolerated(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)

	stamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodNone, 999999)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.True(t, stamp.IsZero(), "no period ledger means nothing recorded and nothing to refund")
	// The caller's snapshot says daily, but the stored token has no period any more.
	stamp, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 999999)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.True(t, stamp.IsZero())
	assert.EqualValues(t, 0, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
}

func TestStaleSnapshotReservesAgainstLivePeriod(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodWeekly, 100)

	// The request authenticated while the token was daily; it is weekly now.
	stamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 70)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, TokenQuotaPeriodWeekly, stamp.Period)
	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 31)
	require.NoError(t, err)
	assert.False(t, allowed, "the live weekly limit still applies to stale snapshots")
	assert.EqualValues(t, 70, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
}

func TestChargeTokenPeriodQuotaRecordsOverageInFullAndBlocksNextReservation(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 100)

	reserved, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 80)
	require.NoError(t, err)
	require.True(t, allowed)

	stamp, over, err := ChargeTokenPeriodQuota(nil, token.Id, TokenQuotaPeriodDaily, 10)
	require.NoError(t, err)
	assert.False(t, over, "a top-up that fits is an ordinary charge")
	assert.Equal(t, reserved, stamp)
	assert.EqualValues(t, 90, getTokenFromDB(t, token.Id).QuotaPeriodUsed)

	// Consumption that already happened is never dropped or clamped.
	_, over, err = ChargeTokenPeriodQuota(nil, token.Id, "", 30)
	require.NoError(t, err)
	assert.True(t, over)
	assert.EqualValues(t, 120, getTokenFromDB(t, token.Id).QuotaPeriodUsed)

	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 1)
	require.NoError(t, err)
	assert.False(t, allowed, "after an overage nothing more is admitted in this window")

	plain := createReserveTestToken(t, 1000)
	stamp, over, err = ChargeTokenPeriodQuota(nil, plain.Id, "", 40)
	require.NoError(t, err)
	assert.True(t, stamp.IsZero())
	assert.False(t, over)
	assert.EqualValues(t, 0, getTokenFromDB(t, plain.Id).QuotaPeriodUsed, "tokens without a period keep no ledger")
}

func TestRefundTokenPeriodQuotaOnlyCreditsTheOriginalWindow(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 100)

	stamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 50)
	require.NoError(t, err)
	require.True(t, allowed)

	// The window rolled over and the new window already holds 20.
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Updates(map[string]interface{}{
		"quota_period_window": stamp.Window + 86400,
		"quota_period_used":   20,
	}).Error)
	applied, err := RefundTokenPeriodQuota(nil, token.Id, stamp, 50)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 20, getTokenFromDB(t, token.Id).QuotaPeriodUsed, "a refund must not shrink the new window")

	// The period kind changed after the spend.
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodWeekly, 100)
	applied, err = RefundTokenPeriodQuota(nil, token.Id, TokenPeriodStamp{Period: TokenQuotaPeriodDaily, Window: stamp.Window + 86400}, 10)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 20, getTokenFromDB(t, token.Id).QuotaPeriodUsed)

	applied, err = RefundTokenPeriodQuota(nil, token.Id, TokenPeriodStamp{}, 10)
	require.NoError(t, err)
	assert.False(t, applied, "a zero stamp refunds nothing")
}

func TestTokenPeriodStampRoundTrip(t *testing.T) {
	stamp := TokenPeriodStamp{Period: TokenQuotaPeriodWeekly, Window: 1790000000}
	assert.Equal(t, stamp, ParseTokenPeriodStamp(stamp.String()))
	for _, raw := range []string{"", "weekly", "weekly:x", "monthly:5", "daily:0", "none:5"} {
		assert.True(t, ParseTokenPeriodStamp(raw).IsZero(), raw)
	}
	assert.Equal(t, "", TokenPeriodStamp{}.String())
}

func updateTokenPeriodConfigForTest(t *testing.T, snapshot Token, period string, limit int64) {
	t.Helper()
	snapshot.QuotaPeriod = period
	snapshot.QuotaPeriodLimit = limit
	snapshot.ApplyQuotaPeriodConfig = true
	require.NoError(t, snapshot.Update())
}

func TestTokenUpdatePeriodConfigNeverWritesBackStaleLedger(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 100)
	snapshot := getTokenFromDB(t, token.Id) // loaded before any spend: used=0, window=0

	stamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 40)
	require.NoError(t, err)
	require.True(t, allowed)

	// Same period, new limit, while a request spent 40 after the snapshot was loaded.
	updateTokenPeriodConfigForTest(t, snapshot, TokenQuotaPeriodDaily, 500)
	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, TokenQuotaPeriodDaily, stored.QuotaPeriod)
	assert.EqualValues(t, 500, stored.QuotaPeriodLimit)
	assert.EqualValues(t, 40, stored.QuotaPeriodUsed, "the stale snapshot must not wipe concurrent usage")
	assert.Equal(t, stamp.Window, stored.QuotaPeriodWindow)

	// A snapshot with an old window and large usage must not revive them either.
	snapshot.QuotaPeriodUsed, snapshot.QuotaPeriodWindow = 99999, 7
	updateTokenPeriodConfigForTest(t, snapshot, TokenQuotaPeriodDaily, 500)
	stored = getTokenFromDB(t, token.Id)
	assert.EqualValues(t, 40, stored.QuotaPeriodUsed)
	assert.Equal(t, stamp.Window, stored.QuotaPeriodWindow)
}

func TestTokenUpdateChangingPeriodResetsLedgerAtomically(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 100)
	snapshot := getTokenFromDB(t, token.Id)

	oldStamp, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 90)
	require.NoError(t, err)
	require.True(t, allowed)

	// The snapshot still says daily/0 and would not see a change by itself; the
	// reset must follow the period stored in the database, not the snapshot.
	updateTokenPeriodConfigForTest(t, snapshot, TokenQuotaPeriodWeekly, 500)
	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, TokenQuotaPeriodWeekly, stored.QuotaPeriod)
	assert.EqualValues(t, 500, stored.QuotaPeriodLimit)
	assert.EqualValues(t, 0, stored.QuotaPeriodUsed)
	assert.EqualValues(t, 0, stored.QuotaPeriodWindow)

	// A refund of the old daily spend must not credit the weekly ledger.
	_, allowed, err = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 70)
	require.NoError(t, err)
	require.True(t, allowed)
	applied, err := RefundTokenPeriodQuota(nil, token.Id, oldStamp, 90)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.EqualValues(t, 70, getTokenFromDB(t, token.Id).QuotaPeriodUsed)

	// Switching to none clears limit and ledger; coming back starts from zero.
	updateTokenPeriodConfigForTest(t, getTokenFromDB(t, token.Id), TokenQuotaPeriodNone, 500)
	stored = getTokenFromDB(t, token.Id)
	assert.Equal(t, TokenQuotaPeriodNone, stored.QuotaPeriod)
	assert.EqualValues(t, 0, stored.QuotaPeriodLimit)
	assert.EqualValues(t, 0, stored.QuotaPeriodUsed)
	updateTokenPeriodConfigForTest(t, getTokenFromDB(t, token.Id), TokenQuotaPeriodDaily, 500)
	assert.EqualValues(t, 0, getTokenFromDB(t, token.Id).QuotaPeriodUsed)
}

func TestTokenUpdateWithoutPeriodConfigLeavesPeriodAndLedgerAlone(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 1000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodWeekly, 100)
	snapshot := getTokenFromDB(t, token.Id)
	_, allowed, err := TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodWeekly, 25)
	require.NoError(t, err)
	require.True(t, allowed)

	snapshot.Name = "renamed"
	snapshot.QuotaPeriod = TokenQuotaPeriodDaily // must be ignored without the apply flag
	require.NoError(t, snapshot.Update())
	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, "renamed", stored.Name)
	assert.Equal(t, TokenQuotaPeriodWeekly, stored.QuotaPeriod)
	assert.EqualValues(t, 100, stored.QuotaPeriodLimit)
	assert.EqualValues(t, 25, stored.QuotaPeriodUsed)
}

func TestTokenUpdatePeriodConfigKeepsConcurrentSpendInSameWindow(t *testing.T) {
	truncateTables(t)
	token := createReserveTestToken(t, 100000)
	setTokenPeriodForTest(t, token.Id, TokenQuotaPeriodDaily, 1000000)
	snapshot := getTokenFromDB(t, token.Id)

	const workers, perWorker = 8, 25
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWorker; j++ {
				_, _, _ = TryReserveTokenPeriodQuota(token.Id, TokenQuotaPeriodDaily, 1)
			}
		}()
	}
	for i := 0; i < 5; i++ {
		updateTokenPeriodConfigForTest(t, snapshot, TokenQuotaPeriodDaily, 1000000+int64(i))
	}
	wg.Wait()
	assert.EqualValues(t, workers*perWorker, getTokenFromDB(t, token.Id).QuotaPeriodUsed,
		"same-window updates racing with spends must not lose any usage")
}
