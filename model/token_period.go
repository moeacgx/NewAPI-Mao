package model

import (
	"errors"
	"fmt"
	"strconv"
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

// TokenPeriodStamp identifies the exact ledger window a spend was recorded in.
// A refund may only be applied to that same (period, window): once the window
// has rolled over, or the period kind was changed, the original spend is no
// longer part of the ledger and crediting it back would shrink new usage.
type TokenPeriodStamp struct {
	Period string
	Window int64
}

// IsZero reports whether the stamp carries no ledger identity.
func (s TokenPeriodStamp) IsZero() bool {
	return (s.Period != TokenQuotaPeriodDaily && s.Period != TokenQuotaPeriodWeekly) || s.Window <= 0
}

// String is the durable form kept on tasks and relay info ("" for no stamp).
func (s TokenPeriodStamp) String() string {
	if s.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s:%d", s.Period, s.Window)
}

// ParseTokenPeriodStamp reads String's form; anything else is the zero stamp.
func ParseTokenPeriodStamp(raw string) TokenPeriodStamp {
	period, window, found := strings.Cut(raw, ":")
	if !found {
		return TokenPeriodStamp{}
	}
	value, err := strconv.ParseInt(window, 10, 64)
	if err != nil {
		return TokenPeriodStamp{}
	}
	stamp := TokenPeriodStamp{Period: period, Window: value}
	if stamp.IsZero() {
		return TokenPeriodStamp{}
	}
	return stamp
}

// applyTokenQuotaPeriodConfig writes the period configuration and, when the
// period kind changes, resets the ledger, all inside the caller's transaction.
// The reset statement compares against the period stored in the database now,
// never against a loaded snapshot, and runs first: the following statement then
// overwrites quota_period (assignments inside one UPDATE are not order-safe
// across databases). Keeping the same period only updates the limit and leaves
// used/window to concurrent spenders. The row stays locked until the
// transaction commits, so no spender can land between the two statements.
func applyTokenQuotaPeriodConfig(tx *gorm.DB, id int, period string, limit int64) error {
	normalized, ok := NormalizeTokenQuotaPeriod(period)
	if !ok {
		return errors.New("invalid token quota period")
	}
	if normalized == TokenQuotaPeriodNone {
		limit = 0
	}
	if err := tx.Model(&Token{}).Where("id = ? AND quota_period <> ?", id, normalized).
		Updates(map[string]interface{}{"quota_period_used": 0, "quota_period_window": 0}).Error; err != nil {
		return err
	}
	return tx.Model(&Token{}).Where("id = ?", id).
		Updates(map[string]interface{}{"quota_period": normalized, "quota_period_limit": limit}).Error
}

// periodSpendColumns adds quota to the window's usage or restarts a stale window.
// The map keys are written in alphabetical order, so quota_period_used is
// always evaluated against the previous quota_period_window, also on MySQL.
func periodSpendColumns(window int64, quota int) map[string]interface{} {
	return map[string]interface{}{
		"quota_period_used":   gorm.Expr("CASE WHEN quota_period_window = ? THEN quota_period_used + ? ELSE ? END", window, quota, quota),
		"quota_period_window": window,
	}
}

const activePeriodScope = "id = ? AND quota_period = ? AND quota_period_limit > 0"

// spendTokenPeriod runs the one atomic statement behind both admission and
// settlement top-ups. With enforce it only succeeds while usage stays within the
// limit; without enforce the spend is recorded unconditionally (the usage has
// already happened) and over reports whether it pushed the window past the limit.
// applied is false when the token has no active ledger for period, or when
// enforce refused the spend.
func spendTokenPeriod(tx *gorm.DB, id int, period string, quota int, enforce bool) (stamp TokenPeriodStamp, applied bool, over bool, err error) {
	window, _, valid := TokenQuotaPeriodWindow(period, time.Now())
	if !valid {
		return TokenPeriodStamp{}, false, false, nil
	}
	stamp = TokenPeriodStamp{Period: period, Window: window}
	withinLimit := tx.Model(&Token{}).Where(activePeriodScope, id, period).
		Where("(CASE WHEN quota_period_window = ? THEN quota_period_used ELSE 0 END) + ? <= quota_period_limit", window, quota).
		Updates(periodSpendColumns(window, quota))
	if withinLimit.Error != nil {
		return TokenPeriodStamp{}, false, false, withinLimit.Error
	}
	if withinLimit.RowsAffected == 1 {
		return stamp, true, false, nil
	}
	if enforce {
		return TokenPeriodStamp{}, false, false, nil
	}
	forced := tx.Model(&Token{}).Where(activePeriodScope, id, period).Updates(periodSpendColumns(window, quota))
	if forced.Error != nil {
		return TokenPeriodStamp{}, false, false, forced.Error
	}
	if forced.RowsAffected == 1 {
		return stamp, true, true, nil
	}
	return TokenPeriodStamp{}, false, false, nil
}

// currentTokenPeriod reads the period configured right now, so a request that
// authenticated with an older snapshot still lands in the live ledger.
func currentTokenPeriod(tx *gorm.DB, id int) (string, error) {
	var token Token
	err := tx.Select("quota_period", "quota_period_limit").Where("id = ?", id).Limit(1).Find(&token).Error
	if err != nil || token.QuotaPeriodLimit <= 0 {
		return "", err
	}
	period, _ := NormalizeTokenQuotaPeriod(token.QuotaPeriod)
	if period == TokenQuotaPeriodNone {
		return "", nil
	}
	return period, nil
}

// TryReserveTokenPeriodQuota is the admission check: it atomically verifies and
// adds quota to the token's current window in one conditional UPDATE, so
// concurrent requests can never pass the check together. allowed is false only
// when the limit would be exceeded; stamp identifies the window to refund
// against and is zero when the token has no active period ledger (nothing was
// recorded). period is the caller's snapshot and is re-read once if it is stale.
func TryReserveTokenPeriodQuota(id int, period string, quota int) (stamp TokenPeriodStamp, allowed bool, err error) {
	if quota < 0 {
		return TokenPeriodStamp{}, false, errors.New("quota 不能为负数！")
	}
	if quota == 0 {
		return TokenPeriodStamp{}, true, nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		if period == "" || period == TokenQuotaPeriodNone {
			return TokenPeriodStamp{}, true, nil
		}
		var applied bool
		stamp, applied, _, err = spendTokenPeriod(DB, id, period, quota, true)
		if err != nil {
			return TokenPeriodStamp{}, false, err
		}
		if applied {
			return stamp, true, nil
		}
		live, readErr := currentTokenPeriod(DB, id)
		if readErr != nil {
			return TokenPeriodStamp{}, false, readErr
		}
		if live == period {
			return TokenPeriodStamp{}, false, nil
		}
		period = live
	}
	return TokenPeriodStamp{}, false, nil
}

// ChargeTokenPeriodQuota records a settlement top-up (usage that has already
// been consumed) with the same atomic statement as admission. A top-up that
// fits is recorded as usual; one that does not fit is still recorded in full
// (never dropped or clamped) and reported through over, so callers can surface
// it while the next admission is refused. period may be empty when the caller
// does not know it (async tasks): the token's live period is used.
func ChargeTokenPeriodQuota(tx *gorm.DB, id int, period string, quota int) (stamp TokenPeriodStamp, over bool, err error) {
	if tx == nil {
		tx = DB
	}
	if quota <= 0 {
		return TokenPeriodStamp{}, false, nil
	}
	if period == "" || period == TokenQuotaPeriodNone {
		if period, err = currentTokenPeriod(tx, id); err != nil || period == "" {
			return TokenPeriodStamp{}, false, err
		}
	}
	stamp, applied, over, err := spendTokenPeriod(tx, id, period, quota, false)
	if err != nil || !applied {
		return TokenPeriodStamp{}, false, err
	}
	return stamp, over, nil
}

// RefundTokenPeriodQuota credits quota back to the ledger window a spend was
// recorded in. A zero stamp, a rolled-over window or a changed period applies
// nothing (applied=false): that usage is no longer in the current ledger.
// Usage never goes below zero.
func RefundTokenPeriodQuota(tx *gorm.DB, id int, stamp TokenPeriodStamp, quota int) (applied bool, err error) {
	if tx == nil {
		tx = DB
	}
	if stamp.IsZero() || quota <= 0 {
		return false, nil
	}
	result := tx.Model(&Token{}).
		Where("id = ? AND quota_period = ? AND quota_period_window = ?", id, stamp.Period, stamp.Window).
		Update("quota_period_used", gorm.Expr("CASE WHEN quota_period_used >= ? THEN quota_period_used - ? ELSE 0 END", quota, quota))
	return result.RowsAffected == 1, result.Error
}
