package service

import (
	"testing"

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
