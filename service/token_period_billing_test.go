package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPeriodToken(t *testing.T, id int, key string, remain int, unlimited bool, period string, limit int64) *relaycommon.RelayInfo {
	t.Helper()
	truncate(t)
	seedUser(t, id, 1000000)
	seedToken(t, id, id, key, remain)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", id).Updates(map[string]any{
		"unlimited_quota":    unlimited,
		"quota_period":       period,
		"quota_period_limit": limit,
	}).Error)
	return &relaycommon.RelayInfo{UserId: id, TokenId: id, TokenKey: key, TokenUnlimited: unlimited, TokenQuotaPeriod: period}
}

func storedPeriodToken(t *testing.T, id int) model.Token {
	t.Helper()
	var token model.Token
	require.NoError(t, model.DB.First(&token, id).Error)
	return token
}

func TestPreConsumeStopsAtWeeklyLimitForUnlimitedTotalQuota(t *testing.T) {
	info := seedPeriodToken(t, 9101, "period-unlimited-key", 0, true, model.TokenQuotaPeriodWeekly, 100)

	require.NoError(t, PreConsumeTokenQuota(info, 60))
	require.Error(t, PreConsumeTokenQuota(info, 41), "unlimited total quota must not disable the weekly limit")
	stored := storedPeriodToken(t, 9101)
	assert.EqualValues(t, 60, stored.QuotaPeriodUsed, "a refused request leaves the ledger untouched")
	assert.Equal(t, 60, stored.UsedQuota)

	require.NoError(t, PreConsumeTokenQuota(info, 40))
	assert.EqualValues(t, 100, storedPeriodToken(t, 9101).QuotaPeriodUsed)
}

func TestPreConsumeWeekly500UsdWithTotal10000Usd(t *testing.T) {
	weekly := int64(500 * common.QuotaPerUnit)
	total := int(10000 * common.QuotaPerUnit)
	info := seedPeriodToken(t, 9102, "period-total-key", total, false, model.TokenQuotaPeriodWeekly, weekly)

	require.NoError(t, PreConsumeTokenQuota(info, int(weekly)))
	stored := storedPeriodToken(t, 9102)
	assert.Equal(t, weekly, stored.QuotaPeriodUsed)
	assert.Equal(t, total-int(weekly), stored.RemainQuota)

	require.Error(t, PreConsumeTokenQuota(info, 1), "the week is spent although total quota remains")
	assert.Equal(t, total-int(weekly), storedPeriodToken(t, 9102).RemainQuota)
}

func TestPreConsumeRollsBackPeriodWhenTotalQuotaIsInsufficient(t *testing.T) {
	info := seedPeriodToken(t, 9103, "period-rollback-key", 50, false, model.TokenQuotaPeriodDaily, 100)

	require.Error(t, PreConsumeTokenQuota(info, 60))
	stored := storedPeriodToken(t, 9103)
	assert.EqualValues(t, 0, stored.QuotaPeriodUsed, "period reservation is returned when the total quota refuses")
	assert.Equal(t, 50, stored.RemainQuota)
}

func TestPostConsumeAdjustsPeriodLedgerBothWays(t *testing.T) {
	info := seedPeriodToken(t, 9104, "period-settle-key", 1000, false, model.TokenQuotaPeriodDaily, 500)

	require.NoError(t, PostConsumeQuota(info, 120, 0, false))
	assert.EqualValues(t, 120, storedPeriodToken(t, 9104).QuotaPeriodUsed)
	require.NoError(t, PostConsumeQuota(info, -50, 120, false))
	stored := storedPeriodToken(t, 9104)
	assert.EqualValues(t, 70, stored.QuotaPeriodUsed)
	assert.Equal(t, 930, stored.RemainQuota)
}

func TestPreConsumeWithoutPeriodKeepsLegacyBehavior(t *testing.T) {
	info := seedPeriodToken(t, 9105, "period-none-key", 100, false, model.TokenQuotaPeriodNone, 0)

	require.NoError(t, PreConsumeTokenQuota(info, 100))
	assert.EqualValues(t, 0, storedPeriodToken(t, 9105).QuotaPeriodUsed)
	require.Error(t, PreConsumeTokenQuota(info, 1), "total quota still limits a token without a period")
}

func periodBillingSession(info *relaycommon.RelayInfo, preConsumed int) *BillingSession {
	return &BillingSession{
		relayInfo:        info,
		funding:          &WalletFunding{userId: info.UserId, consumed: preConsumed},
		preConsumedQuota: preConsumed,
		tokenConsumed:    preConsumed,
	}
}

func TestSettlementTopUpSharesAtomicLimitAndRecordsOverageInFull(t *testing.T) {
	info := seedPeriodToken(t, 9201, "period-settle-topup", 10000, false, model.TokenQuotaPeriodDaily, 100)
	require.NoError(t, PreConsumeTokenQuota(info, 80))

	session := periodBillingSession(info, 80)
	require.NoError(t, session.Settle(90), "a top-up that still fits is an ordinary charge")
	assert.EqualValues(t, 90, storedPeriodToken(t, 9201).QuotaPeriodUsed)

	over := periodBillingSession(info, 90)
	require.NoError(t, over.Settle(130), "usage that already happened is settled in full, never dropped")
	stored := storedPeriodToken(t, 9201)
	assert.EqualValues(t, 130, stored.QuotaPeriodUsed, "the overage is recorded, not clamped")
	assert.Equal(t, 10000-130, stored.RemainQuota, "total quota follows the same full amount")

	require.Error(t, PreConsumeTokenQuota(info, 1), "the next admission is refused once the window is over its limit")
	assert.EqualValues(t, 130, storedPeriodToken(t, 9201).QuotaPeriodUsed)
}

func TestSettlementRefundCreditsOnlyTheReservedWindow(t *testing.T) {
	info := seedPeriodToken(t, 9202, "period-settle-refund", 10000, false, model.TokenQuotaPeriodDaily, 500)
	require.NoError(t, PreConsumeTokenQuota(info, 50))
	require.NotEmpty(t, info.TokenPeriodStamp)

	// The window rolled over after the reservation; the new window already holds 20.
	stored := storedPeriodToken(t, 9202)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", stored.Id).Updates(map[string]any{
		"quota_period_window": stored.QuotaPeriodWindow + 86400,
		"quota_period_used":   20,
	}).Error)

	require.NoError(t, periodBillingSession(info, 50).Settle(0))
	stored = storedPeriodToken(t, 9202)
	assert.EqualValues(t, 20, stored.QuotaPeriodUsed, "the refund must not shrink the new window")
	assert.Equal(t, 10000, stored.RemainQuota, "the total quota is still returned in full")
}

func TestAsyncTaskAdjustmentHonorsPeriodLedger(t *testing.T) {
	ctx := context.Background()
	info := seedPeriodToken(t, 9203, "period-task-key", 10000, false, model.TokenQuotaPeriodWeekly, 100)
	require.NoError(t, PreConsumeTokenQuota(info, 80))

	task := &model.Task{TaskID: "task-period-topup", UserId: info.UserId, PrivateData: model.TaskPrivateData{
		TokenId: info.TokenId, TokenPeriodStamp: info.TokenPeriodStamp,
	}}
	taskAdjustTokenQuota(ctx, task, 40)
	assert.EqualValues(t, 120, storedPeriodToken(t, 9203).QuotaPeriodUsed, "an async top-up over the limit is recorded in full")
	require.Error(t, PreConsumeTokenQuota(info, 1))

	// A task submitted before period quotas existed carries no stamp and must not touch the ledger.
	legacy := &model.Task{TaskID: "task-legacy", UserId: info.UserId, PrivateData: model.TaskPrivateData{TokenId: info.TokenId}}
	taskAdjustTokenQuota(ctx, legacy, -15)
	assert.EqualValues(t, 120, storedPeriodToken(t, 9203).QuotaPeriodUsed, "a task without a stamp never moves the period ledger")

	// The window rolled over: the refund goes back to the original window only.
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", info.TokenId).Updates(map[string]any{
		"quota_period_window": time.Now().Unix() + 86400,
		"quota_period_used":   20,
	}).Error)
	taskAdjustTokenQuota(ctx, task, -80)
	stored := storedPeriodToken(t, 9203)
	assert.EqualValues(t, 20, stored.QuotaPeriodUsed, "an async refund must not shrink the new window")
	assert.Equal(t, 10000-40+15, stored.RemainQuota, "total quota still follows every adjustment")
}

func TestMidjourneyRefundCreditsOnlyTheReservedWindow(t *testing.T) {
	truncate(t)
	ctx := context.Background()
	const userID, tokenID, channelID = 9204, 9204, 9204
	seedUser(t, userID, 10000)
	seedToken(t, tokenID, userID, "sk-mj-period", 5000)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Updates(map[string]any{
		"quota_period": model.TokenQuotaPeriodDaily, "quota_period_limit": 1000,
	}).Error)
	seedChannel(t, channelID)

	info := &relaycommon.RelayInfo{
		UserId: userID, TokenId: tokenID, TokenKey: "sk-mj-period", UserQuota: 10000,
		TokenQuotaPeriod: model.TokenQuotaPeriodDaily,
		ChannelMeta:      &relaycommon.ChannelMeta{ChannelId: channelID},
	}
	task := &model.Midjourney{UserId: userID, Action: "IMAGINE", MjId: "mj-period", ChannelId: channelID, Progress: "0%"}
	prepared, err := PrepareMidjourneyTaskBilling(info, task, 300, true)
	require.NoError(t, err)
	require.NoError(t, task.Insert())
	_, err = SettleMidjourneyTaskBilling(info, task, prepared)
	require.NoError(t, err)
	persisted := getMidjourneyTask(t, task.Id)
	require.NotEmpty(t, persisted.TokenPeriodStamp, "the charged window is stored with the task")
	assert.EqualValues(t, 300, storedPeriodToken(t, tokenID).QuotaPeriodUsed)

	// Same window: the refund returns the period usage.
	assert.True(t, RefundMidjourneyQuota(ctx, task, "failed"))
	assert.EqualValues(t, 0, storedPeriodToken(t, tokenID).QuotaPeriodUsed)
	assert.Equal(t, 5000, getTokenRemainQuota(t, tokenID))

	// A new charge whose window then rolls over: the refund must not touch the new window.
	task2 := &model.Midjourney{UserId: userID, Action: "IMAGINE", MjId: "mj-period-2", ChannelId: channelID, Progress: "0%"}
	prepared, err = PrepareMidjourneyTaskBilling(info, task2, 200, true)
	require.NoError(t, err)
	require.NoError(t, task2.Insert())
	_, err = SettleMidjourneyTaskBilling(info, task2, prepared)
	require.NoError(t, err)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Updates(map[string]any{
		"quota_period_window": time.Now().Unix() + 86400, "quota_period_used": 20,
	}).Error)
	assert.True(t, RefundMidjourneyQuota(ctx, task2, "failed after rollover"))
	assert.EqualValues(t, 20, storedPeriodToken(t, tokenID).QuotaPeriodUsed, "the new window is left intact")
	assert.Equal(t, 5000, getTokenRemainQuota(t, tokenID), "the total quota is still returned")
}

func TestConcurrentReservationAndSettlementKeepLedgerConsistent(t *testing.T) {
	info := seedPeriodToken(t, 9205, "period-concurrent", 100000, false, model.TokenQuotaPeriodDaily, 100)
	const workers, each = 8, 20
	var wg sync.WaitGroup
	var admitted sync.Map
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			n := 0
			for j := 0; j < each; j++ {
				if PreConsumeTokenQuota(info, 1) == nil {
					n++
				}
			}
			admitted.Store(i, n)
		}(i)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for k := 0; k < 5; k++ {
			_ = PostConsumeQuota(info, 3, 0, false)
		}
	}()
	wg.Wait()
	total := 0
	admitted.Range(func(_, v any) bool { total += v.(int); return true })
	assert.LessOrEqual(t, total, 100, "reservations together never pass the limit")
	assert.EqualValues(t, total+15, storedPeriodToken(t, 9205).QuotaPeriodUsed, "every admitted request and every top-up is recorded exactly once")
	assert.Equal(t, 100000-total-15, storedPeriodToken(t, 9205).RemainQuota, "total quota matches the period ledger")
}
