package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Token quota periods. The period ledger limits what a token may spend inside
// the current natural day or week of the server time zone and restarts when the
// window changes. It is independent of the lifetime quota (remain_quota and
// unlimited_quota), which never resets.
const (
	TokenQuotaPeriodNone   = "none"
	TokenQuotaPeriodDaily  = "daily"
	TokenQuotaPeriodWeekly = "weekly"
)

// NormalizeTokenQuotaPeriod trims and lower-cases a client supplied period.
// An empty value means none; ok is false for any other unknown value.
func NormalizeTokenQuotaPeriod(period string) (string, bool) {
	switch normalized := strings.ToLower(strings.TrimSpace(period)); normalized {
	case "", TokenQuotaPeriodNone:
		return TokenQuotaPeriodNone, true
	case TokenQuotaPeriodDaily, TokenQuotaPeriodWeekly:
		return normalized, true
	}
	return "", false
}

// TokenQuotaPeriodWindow returns the unix start of the natural day (daily) or
// of the week starting on Monday (weekly) that contains now, plus the unix time
// of the next reset. ok is false for none or an unknown period.
func TokenQuotaPeriodWindow(period string, now time.Time) (start int64, resetAt int64, ok bool) {
	year, month, day := now.Date()
	dayStart := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	switch period {
	case TokenQuotaPeriodDaily:
		return dayStart.Unix(), dayStart.AddDate(0, 0, 1).Unix(), true
	case TokenQuotaPeriodWeekly:
		weekStart := dayStart.AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
		return weekStart.Unix(), weekStart.AddDate(0, 0, 7).Unix(), true
	}
	return 0, 0, false
}

// FillQuotaPeriodState derives the read-only usage view returned to clients:
// usage counts as zero once the stored window is no longer the current one, and
// reset_at is the next window boundary. It never writes to the database.
func (token *Token) FillQuotaPeriodState(now time.Time) {
	start, resetAt, ok := TokenQuotaPeriodWindow(token.QuotaPeriod, now)
	if !ok {
		token.QuotaPeriodUsed, token.QuotaPeriodResetAt = 0, 0
		return
	}
	token.QuotaPeriodResetAt = resetAt
	if token.QuotaPeriodWindow != start {
		token.QuotaPeriodUsed = 0
	}
}

// TryReserveTokenPeriodQuota atomically checks and adds quota to the token's
// current period window in one conditional UPDATE, so concurrent requests can
// never pass the check together. A stale window restarts from zero inside the
// same statement. It returns false only when the period limit would be exceeded.
func TryReserveTokenPeriodQuota(id int, period string, quota int) (bool, error) {
	if quota < 0 {
		return false, errors.New("quota 不能为负数！")
	}
	window, _, ok := TokenQuotaPeriodWindow(period, time.Now())
	if quota == 0 || !ok {
		return true, nil
	}
	activeScope := "id = ? AND quota_period = ? AND quota_period_limit > 0"
	result := DB.Model(&Token{}).Where(activeScope, id, period).
		Where("(CASE WHEN quota_period_window = ? THEN quota_period_used ELSE 0 END) + ? <= quota_period_limit", window, quota).
		Updates(map[string]interface{}{
			"quota_period_used":   gorm.Expr("CASE WHEN quota_period_window = ? THEN quota_period_used + ? ELSE ? END", window, quota, quota),
			"quota_period_window": window,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}
	// No row changed: either the limit would be exceeded, or the caller's
	// snapshot is stale and the period was removed or changed meanwhile.
	var active int64
	if err := DB.Model(&Token{}).Where(activeScope, id, period).Count(&active).Error; err != nil {
		return false, err
	}
	return active == 0, nil
}

// AdjustTokenPeriodUsage applies a settlement difference (delta > 0 spend,
// delta < 0 refund) to the period ledger. Spend restarts a stale window; refund
// only reduces the current window and never goes below zero. period may be empty
// when the caller does not know the token's period (async task settlement).
// The map keys are written in alphabetical order, so quota_period_used is always
// evaluated against the previous quota_period_window, also on MySQL.
func AdjustTokenPeriodUsage(id int, period string, delta int) error {
	return AdjustTokenPeriodUsageWithTx(DB, id, period, delta)
}

// AdjustTokenPeriodUsageWithTx is AdjustTokenPeriodUsage inside the caller's transaction.
func AdjustTokenPeriodUsageWithTx(tx *gorm.DB, id int, period string, delta int) error {
	if delta == 0 {
		return nil
	}
	periods := []string{TokenQuotaPeriodDaily, TokenQuotaPeriodWeekly}
	if period != "" {
		periods = []string{period}
	}
	now := time.Now()
	for _, candidate := range periods {
		window, _, ok := TokenQuotaPeriodWindow(candidate, now)
		if !ok {
			continue
		}
		query := tx.Model(&Token{}).Where("id = ? AND quota_period = ?", id, candidate)
		var err error
		if delta > 0 {
			err = query.Updates(map[string]interface{}{
				"quota_period_used":   gorm.Expr("CASE WHEN quota_period_window = ? THEN quota_period_used + ? ELSE ? END", window, delta, delta),
				"quota_period_window": window,
			}).Error
		} else {
			err = query.Where("quota_period_window = ?", window).Update("quota_period_used",
				gorm.Expr("CASE WHEN quota_period_used >= ? THEN quota_period_used - ? ELSE 0 END", -delta, -delta)).Error
		}
		if err != nil {
			return err
		}
	}
	return nil
}
