package model

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRevenueStatsCurrencySplit(t *testing.T) {
	dbPath := "test_revenue_split.db"
	defer os.Remove(dbPath)

	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	require.NoError(t, err)

	err = DB.AutoMigrate(&User{}, &TopUp{}, &SubscriptionOrder{}, &Redemption{}, &RedemptionUsage{})
	require.NoError(t, err)

	now := time.Now().Unix()

	err = DB.Create(&User{Id: 10, Username: "testuser", Password: "x", Role: 1, Quota: 5000000}).Error
	require.NoError(t, err)

	// Balance payment: skip (no external cash)
	err = DB.Create(&TopUp{Id: 101, UserId: 10, Amount: 100, Money: 10, ActualMoney: 10, PaymentProvider: "balance", PaymentMethod: "balance", CompleteTime: now, Status: "success", TradeNo: "bal-101"}).Error
	require.NoError(t, err)

	// Stripe with snapshot (cents): 10 USD
	err = DB.Create(&TopUp{Id: 102, UserId: 10, Amount: 100, Money: 100, ActualMoney: 95, ProviderAmount: "1000", ProviderCurrency: "USD", PaymentProvider: "stripe", PaymentMethod: "stripe", CompleteTime: now + 100, Status: "success", TradeNo: "stripe-102"}).Error
	require.NoError(t, err)

	// Epay legacy (no snapshot, fixed CNY): 50 CNY
	err = DB.Create(&TopUp{Id: 103, UserId: 10, Amount: 50, Money: 50, ActualMoney: 50, PromoCodeId: 0, PaymentProvider: "epay", PaymentMethod: "wechat", CompleteTime: now + 200, Status: "success", TradeNo: "epay-103"}).Error
	require.NoError(t, err)

	// Bepusdt legacy: 72 CNY
	err = DB.Create(&TopUp{Id: 104, UserId: 10, Amount: 72, Money: 72, ActualMoney: 72, PaymentProvider: "bepusdt", PaymentMethod: "bepusdt", CompleteTime: now + 300, Status: "success", TradeNo: "bepusdt-104"}).Error
	require.NoError(t, err)

	// Okpay with snapshot: 6.5 USDT
	err = DB.Create(&TopUp{Id: 105, UserId: 10, Amount: 100, Money: 100, ActualMoney: 100, ProviderAmount: "6.50000000", ProviderCurrency: "USDT", PaymentProvider: "okpay", PaymentMethod: "okpay", CompleteTime: now + 400, Status: "success", TradeNo: "okpay-105"}).Error
	require.NoError(t, err)

	// Creem with snapshot (cents): 8 EUR
	err = DB.Create(&TopUp{Id: 106, UserId: 10, Amount: 80, Money: 80, ActualMoney: 80, ProviderAmount: "800", ProviderCurrency: "EUR", PaymentProvider: "creem", PaymentMethod: "creem", CompleteTime: now + 500, Status: "success", TradeNo: "creem-106"}).Error
	require.NoError(t, err)

	// Free/promo (actual_money=0): skip
	err = DB.Create(&TopUp{Id: 107, UserId: 10, Amount: 50, Money: 50, ActualMoney: 0, PromoCodeId: 1, PaymentProvider: "epay", PaymentMethod: "alipay", CompleteTime: now + 600, Status: "success", TradeNo: "free-107"}).Error
	require.NoError(t, err)

	// Unresolved: waffo without snapshot (not fixed currency)
	err = DB.Create(&TopUp{Id: 108, UserId: 10, Amount: 100, Money: 100, ActualMoney: 100, PaymentProvider: "waffo", PaymentMethod: "waffo", CompleteTime: now + 700, Status: "success", TradeNo: "waffo-108"}).Error
	require.NoError(t, err)

	// Subscription order with USD snapshot (main unit, not cents)
	err = DB.Create(&SubscriptionOrder{Id: 201, UserId: 10, PlanId: 1, Money: 20, ActualMoney: 20, ProviderAmount: "20.00", ProviderCurrency: "USD", PaymentProvider: "stripe", PaymentMethod: "stripe", CompleteTime: now + 800, Status: "success", TradeNo: "sub-201"}).Error
	require.NoError(t, err)

	err = DB.Create(&TopUp{Id: 201, UserId: 10, Amount: 0, Money: 20, ActualMoney: 20, PaymentProvider: "stripe", PaymentMethod: "stripe", CompleteTime: now + 800, Status: "success", TradeNo: "sub-201"}).Error
	require.NoError(t, err)

	// Redemption codes
	err = DB.Create(&Redemption{Id: 301, Name: "REDEEM100", Quota: 500000, UsedUserId: 10, CreatedTime: now - 100000, Status: 1}).Error
	require.NoError(t, err)

	err = DB.Create(&RedemptionUsage{Id: 3001, RedemptionId: 301, UserId: 10, CreatedTime: now + 1000}).Error
	require.NoError(t, err)

	// Query revenue stats
	stats, err := GetRevenueStats(now-10000, now+10000, "day", 0)
	require.NoError(t, err)
	require.NotNil(t, stats)

	summary := stats.Summary

	// Validate cash totals by currency
	cashByCurrency := make(map[string]RevenueCashTotal)
	for _, c := range summary.Cash {
		cashByCurrency[c.Currency] = c
	}

	// USD: stripe cents (10) + subscription (20) = 30 USD, 2 payments
	assert.Equal(t, 30.0, cashByCurrency["USD"].Amount, "USD amount should be 30")
	assert.Equal(t, 2, cashByCurrency["USD"].Count, "USD count should be 2")

	// CNY: epay (50) + bepusdt (72) = 122 CNY, 2 payments
	assert.Equal(t, 122.0, cashByCurrency["CNY"].Amount, "CNY amount should be 122")
	assert.Equal(t, 2, cashByCurrency["CNY"].Count, "CNY count should be 2")

	// USDT: okpay (6.5) = 6.5 USDT, 1 payment
	assert.Equal(t, 6.5, cashByCurrency["USDT"].Amount, "USDT amount should be 6.5")
	assert.Equal(t, 1, cashByCurrency["USDT"].Count, "USDT count should be 1")

	// EUR: creem cents (8) = 8 EUR, 1 payment
	assert.Equal(t, 8.0, cashByCurrency["EUR"].Amount, "EUR amount should be 8")
	assert.Equal(t, 1, cashByCurrency["EUR"].Count, "EUR count should be 1")

	// Unresolved: waffo (1)
	assert.Equal(t, 1, summary.UnresolvedCashCount, "Unresolved count should be 1")

	// Redemption: 500000 quota, 1 count
	assert.Equal(t, int64(500000), summary.TotalRedemptionQuota, "Redemption quota should be 500000")
	assert.Equal(t, 1, summary.TotalRedemptionCount, "Redemption count should be 1")

	// Ensure balance and free payments are skipped
	assert.NotContains(t, cashByCurrency, "balance", "Balance payment should be skipped")

	// Validate data points
	require.Greater(t, len(stats.DataPoints), 0, "Should have at least one data point")
	firstPoint := stats.DataPoints[0]
	pointCashByCurrency := make(map[string]RevenueCashTotal)
	for _, c := range firstPoint.Cash {
		pointCashByCurrency[c.Currency] = c
	}
	assert.Equal(t, 30.0, pointCashByCurrency["USD"].Amount, "Data point USD should match summary")
	assert.Equal(t, 1, firstPoint.UnresolvedCashCount, "Data point unresolved should match summary")
}
