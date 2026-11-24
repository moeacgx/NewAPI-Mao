package model

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// RevenueCashTotal 是某个币种下实际外部收款的合计。不同币种从不相加。
type RevenueCashTotal struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

// RevenueDataPoint 表示一个时间桶内的收入数据。
// Cash 只含外部真实付款（按币种分组）；RedemptionQuota 是兑换核销的站内额度，不是现金。
// UnresolvedCashCount 是无法可靠确定币种或实付金额的外部订单数，不计入 Cash。
type RevenueDataPoint struct {
	Timestamp           int64              `json:"timestamp"`
	Cash                []RevenueCashTotal `json:"cash"`
	UnresolvedCashCount int                `json:"unresolved_cash_count"`
	RedemptionQuota     int64              `json:"redemption_quota"`
	RedemptionCount     int                `json:"redemption_count"`
}

// RevenueSummary 表示查询时间范围内的收入汇总。
type RevenueSummary struct {
	Cash                 []RevenueCashTotal `json:"cash"`
	UnresolvedCashCount  int                `json:"unresolved_cash_count"`
	TotalRedemptionQuota int64              `json:"total_redemption_quota"`
	TotalRedemptionCount int                `json:"total_redemption_count"`
}

// RevenueStatsResponse 表示收入统计接口的完整响应。
type RevenueStatsResponse struct {
	Summary    RevenueSummary     `json:"summary"`
	DataPoints []RevenueDataPoint `json:"data_points"`
}

// revenueTopUpRow 接收已完成订单；订阅订单的支付快照通过 subscription_orders 补充。
type revenueTopUpRow struct {
	CompleteTime     int64   `gorm:"column:complete_time"`
	PaymentProvider  string  `gorm:"column:payment_provider"`
	PaymentMethod    string  `gorm:"column:payment_method"`
	Money            float64 `gorm:"column:money"`
	ActualMoney      float64 `gorm:"column:actual_money"`
	PromoCodeId      int     `gorm:"column:promo_code_id"`
	ProviderAmount   string  `gorm:"column:provider_amount"`
	ProviderCurrency string  `gorm:"column:provider_currency"`
	SubOrderId       int     `gorm:"column:sub_order_id"`
	SubAmount        string  `gorm:"column:sub_provider_amount"`
	SubCurrency      string  `gorm:"column:sub_provider_currency"`
}

type revenueRedemptionRow struct {
	Bucket int64 `gorm:"column:bucket"`
	Quota  int64 `gorm:"column:quota"`
	Cnt    int   `gorm:"column:cnt"`
}

type revenueCashStatus int

const (
	revenueCashSkip revenueCashStatus = iota
	revenueCashResolved
	revenueCashUnresolved
)

// resolveRevenueCash 返回订单实际外部付款的币种与金额。
// 只采用可核验来源：
//  1. 支付快照 provider_amount + provider_currency（含订阅订单上的快照）。
//     单位由写入方决定：stripe/creem 充值快照为最小货币单位（分），订阅订单快照与其他渠道均为主单位。
//  2. 无快照的旧单仅在渠道本身固定币种时采用 actual_money：epay/bepusdt 为 CNY（回调校验强制 CNY），
//     stripe 为 USD（回调校验强制 USD）。
//
// 其余情况币种不可核验，返回 unresolved。paid_amount_cny 对 stripe 等渠道是按展示汇率折算的值，不是实付，故不采用。
func resolveRevenueCash(row revenueTopUpRow) (string, decimal.Decimal, revenueCashStatus) {
	provider := invoiceOrderPaymentProvider(row.PaymentProvider, row.PaymentMethod)
	if provider == PaymentProviderBalance {
		return "", decimal.Zero, revenueCashSkip
	}

	amountText := strings.TrimSpace(row.ProviderAmount)
	currency := strings.ToUpper(strings.TrimSpace(row.ProviderCurrency))
	minorUnit := provider == PaymentProviderStripe || provider == PaymentProviderCreem
	if row.SubOrderId > 0 {
		// 订阅订单快照由订阅流程写入，金额统一为主单位。
		amountText = strings.TrimSpace(row.SubAmount)
		currency = strings.ToUpper(strings.TrimSpace(row.SubCurrency))
		minorUnit = false
	}

	if amountText != "" || currency != "" {
		if amountText == "" || currency == "" {
			return "", decimal.Zero, revenueCashUnresolved
		}
		switch provider {
		case PaymentProviderStripe, PaymentProviderCreem, PaymentProviderEpay, PaymentProviderBepusdt, PaymentProviderOkpay, PaymentProviderWaffo:
		default:
			// 未知渠道的快照单位无法核验。
			return "", decimal.Zero, revenueCashUnresolved
		}
		amount, err := decimal.NewFromString(amountText)
		if err != nil || amount.IsNegative() {
			return "", decimal.Zero, revenueCashUnresolved
		}
		if minorUnit {
			amount = amount.Div(decimal.NewFromInt(100))
		}
		if amount.IsZero() {
			return "", decimal.Zero, revenueCashSkip
		}
		return currency, amount, revenueCashResolved
	}

	paid := invoiceOrderPaidAmount(row.Money, row.ActualMoney, row.PromoCodeId)
	if paid <= 0 {
		// 全额优惠或免费订单没有外部收款。
		return "", decimal.Zero, revenueCashSkip
	}
	switch provider {
	case PaymentProviderEpay, PaymentProviderBepusdt:
		return "CNY", decimal.NewFromFloat(paid), revenueCashResolved
	case PaymentProviderStripe:
		return "USD", decimal.NewFromFloat(paid), revenueCashResolved
	default:
		return "", decimal.Zero, revenueCashUnresolved
	}
}

type revenueCashAccumulator map[string]*struct {
	amount decimal.Decimal
	count  int
}

func (a revenueCashAccumulator) add(currency string, amount decimal.Decimal) {
	entry, ok := a[currency]
	if !ok {
		entry = &struct {
			amount decimal.Decimal
			count  int
		}{}
		a[currency] = entry
	}
	entry.amount = entry.amount.Add(amount)
	entry.count++
}

func (a revenueCashAccumulator) totals() []RevenueCashTotal {
	result := make([]RevenueCashTotal, 0, len(a))
	for currency, entry := range a {
		result = append(result, RevenueCashTotal{
			Currency: currency,
			Amount:   entry.amount.Round(2).InexactFloat64(),
			Count:    entry.count,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Currency < result[j].Currency })
	return result
}

// GetRevenueStats 聚合指定时间范围内的外部现金收款（按币种）与兑换码核销的站内额度。
// tzOffset 是客户端相对 UTC 的秒偏移量，用于把每日时间桶对齐到客户端本地零点。
func GetRevenueStats(startTime, endTime int64, granularity string, tzOffset int64) (*RevenueStatsResponse, error) {
	bucketSeconds := int64(86400)
	if granularity == "hour" {
		bucketSeconds = 3600
	}
	bucketOf := func(ts int64) int64 {
		return (ts+tzOffset)/bucketSeconds*bucketSeconds - tzOffset
	}

	var rows []revenueTopUpRow
	// 订阅订单完成后已经写入 top_ups，不再单独查询订阅表，避免重复统计；这里仅取其支付快照。
	if err := DB.Raw(
		"SELECT t.complete_time AS complete_time, COALESCE(t.payment_provider, '') AS payment_provider, "+
			"COALESCE(t.payment_method, '') AS payment_method, t.money AS money, t.actual_money AS actual_money, "+
			"t.promo_code_id AS promo_code_id, COALESCE(t.provider_amount, '') AS provider_amount, "+
			"COALESCE(t.provider_currency, '') AS provider_currency, COALESCE(so.id, 0) AS sub_order_id, "+
			"COALESCE(so.provider_amount, '') AS sub_provider_amount, COALESCE(so.provider_currency, '') AS sub_provider_currency "+
			"FROM top_ups t LEFT JOIN subscription_orders so ON so.trade_no = t.trade_no "+
			"WHERE t.status = 'success' AND t.complete_time >= ? AND t.complete_time <= ?",
		startTime, endTime,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}

	type bucketState struct {
		point *RevenueDataPoint
		cash  revenueCashAccumulator
	}
	buckets := make(map[int64]*bucketState)
	summaryCash := revenueCashAccumulator{}
	summary := RevenueSummary{}
	stateFor := func(bucket int64) *bucketState {
		state, ok := buckets[bucket]
		if !ok {
			state = &bucketState{point: &RevenueDataPoint{Timestamp: bucket}, cash: revenueCashAccumulator{}}
			buckets[bucket] = state
		}
		return state
	}

	for _, row := range rows {
		currency, amount, status := resolveRevenueCash(row)
		switch status {
		case revenueCashResolved:
			stateFor(bucketOf(row.CompleteTime)).cash.add(currency, amount)
			summaryCash.add(currency, amount)
		case revenueCashUnresolved:
			stateFor(bucketOf(row.CompleteTime)).point.UnresolvedCashCount++
			summary.UnresolvedCashCount++
		}
	}

	// 兑换码只按核销时间统计站内额度与次数；无独立收款记录，不计现金。
	var redemptionRows []revenueRedemptionRow
	if err := DB.Raw(
		"SELECT ((ru.created_time + ?) / ? * ? - ?) AS bucket, COALESCE(SUM(r.quota), 0) AS quota, COUNT(*) AS cnt "+
			"FROM redemption_usages ru "+
			"LEFT JOIN redemptions r ON r.id = ru.redemption_id "+
			"WHERE ru.created_time >= ? AND ru.created_time <= ? "+
			"GROUP BY bucket ORDER BY bucket",
		tzOffset, bucketSeconds, bucketSeconds, tzOffset, startTime, endTime,
	).Scan(&redemptionRows).Error; err != nil {
		return nil, err
	}
	for _, r := range redemptionRows {
		point := stateFor(r.Bucket).point
		point.RedemptionQuota += r.Quota
		point.RedemptionCount += r.Cnt
		summary.TotalRedemptionQuota += r.Quota
		summary.TotalRedemptionCount += r.Cnt
	}

	summary.Cash = summaryCash.totals()
	dataPoints := make([]RevenueDataPoint, 0, len(buckets))
	for _, state := range buckets {
		state.point.Cash = state.cash.totals()
		dataPoints = append(dataPoints, *state.point)
	}
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp < dataPoints[j].Timestamp
	})

	return &RevenueStatsResponse{Summary: summary, DataPoints: dataPoints}, nil
}
