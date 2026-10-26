package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClosedCombinedInvoiceReleasesOrdersAndPreservesHistory(t *testing.T) {
	db := setupInvoiceOrderTestDB(t)
	setupInvoicePendingNotificationFixture(t, db)
	createInvoiceOrderTestUser(t, db, 2201, 2_000_000)
	topUp := recentInvoiceTopUp(2201, "TOP-CLOSE", 70)
	require.NoError(t, db.Create(&topUp).Error)
	subscription := SubscriptionOrder{
		UserId: 2201, TradeNo: "SUB-CLOSE", Money: 10, ActualMoney: 10,
		PaymentMethod: PaymentMethodStripe, PaymentProvider: PaymentProviderStripe,
		Status: common.TopUpStatusSuccess, CreateTime: topUp.CreateTime, CompleteTime: topUp.CompleteTime,
	}
	require.NoError(t, db.Create(&subscription).Error)
	mirror := recentInvoiceTopUp(2201, subscription.TradeNo, 10)
	require.NoError(t, db.Create(&mirror).Error)
	refs := []InvoiceOrderReference{
		{SourceType: InvoiceSourceTopUp, SourceId: topUp.TradeNo},
		{SourceType: InvoiceSourceSubscription, SourceId: subscription.TradeNo},
	}
	record, err := CreateCombinedInvoiceWithBalance(2201, refs, validCombinedInvoiceRequest())
	require.NoError(t, err)
	require.NoError(t, UpdateInvoiceRecord(record.Id, "", InvoiceStatusClosed, "关闭并释放"))

	orders, err := GetRecentInvoiceOrders(2201)
	require.NoError(t, err)
	require.Len(t, orders, 2)
	for _, order := range orders {
		assert.True(t, order.InvoiceEligible)
		assert.False(t, order.Invoiced)
		assert.Empty(t, order.InvoiceStatus)
	}
	require.NoError(t, db.First(&mirror, mirror.Id).Error)
	assert.False(t, mirror.InvoiceRequired)
	assert.Empty(t, mirror.InvoiceTitle)
	assert.Empty(t, mirror.InvoiceStatus)

	replacement, err := CreateCombinedInvoiceExternalPayment(2201, refs, validCombinedInvoiceRequest(), InvoiceExternalPaymentOptions{
		PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay", ProviderMerchantId: "test-merchant",
	})
	require.NoError(t, err)
	require.Len(t, replacement.Orders, 2)
	for _, status := range []string{InvoiceStatusPending, InvoiceStatusIssued} {
		require.ErrorContains(t, UpdateInvoiceRecord(record.Id, "", status, "重开"), "关闭")
	}
	require.NoError(t, UpdateInvoiceRecord(record.Id, "", InvoiceStatusClosed, "补充备注"))
	require.NoError(t, db.First(&topUp, topUp.Id).Error)
	assert.True(t, topUp.InvoiceRequired)
	assert.Equal(t, InvoiceStatusPaymentPending, topUp.InvoiceStatus)
	_, err = PreviewInvoiceOrders(2201, refs)
	require.ErrorContains(t, err, "已经申请过发票")

	records, total, err := GetUserInvoiceRecords(2201, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, records, 2)
	assert.Equal(t, record.Orders, records[1].Orders)
	assert.Equal(t, InvoiceStatusClosed, records[1].Status)
	assert.Equal(t, InvoicePaymentStatusSuccess, records[1].PaymentStatus)
	assert.Equal(t, record.FeeAmount, records[1].FeeAmount)
	var user User
	require.NoError(t, db.First(&user, 2201).Error)
	assert.EqualValues(t, 1_000_000, user.Quota, "关闭不自动退款，新外部申请不扣余额")
}

func TestHistoricalClosedInvoiceOrdersBecomeAvailable(t *testing.T) {
	for _, source := range []string{InvoiceSourceTopUp, InvoiceSourceSubscription, InvoiceSourceCombined} {
		for _, deleted := range []bool{false, true} {
			name := source
			if deleted {
				name += "_deleted"
			}
			t.Run(name, func(t *testing.T) {
				db := setupInvoiceOrderTestDB(t)
				createInvoiceOrderTestUser(t, db, 2202, 2_000_000)
				topUp := recentInvoiceTopUp(2202, "OLD-CLOSED", 70)
				topUp.InvoiceRequired = true
				topUp.InvoiceStatus = InvoiceStatusClosed
				topUp.InvoiceTitle = "旧发票抬头"
				require.NoError(t, db.Create(&topUp).Error)
				ref := InvoiceOrderReference{SourceType: InvoiceSourceTopUp, SourceId: topUp.TradeNo}
				if source == InvoiceSourceSubscription {
					order := SubscriptionOrder{
						UserId: 2202, TradeNo: topUp.TradeNo, Money: 70, ActualMoney: 70,
						Status: common.TopUpStatusSuccess, PaymentMethod: "alipay",
						CreateTime: topUp.CreateTime, CompleteTime: topUp.CompleteTime,
						InvoiceRequired: true, InvoiceStatus: InvoiceStatusClosed,
					}
					require.NoError(t, db.Create(&order).Error)
					ref.SourceType = InvoiceSourceSubscription
				}
				record := InvoiceRecord{UserId: 2202, SourceType: source, SourceId: topUp.TradeNo, Status: InvoiceStatusClosed}
				require.NoError(t, db.Create(&record).Error)
				if source == InvoiceSourceCombined {
					require.NoError(t, db.Create(&InvoiceOrderLink{InvoiceId: record.Id, UserId: 2202, SourceType: ref.SourceType, SourceId: ref.SourceId, BaseAmount: 70}).Error)
				}
				if deleted {
					require.NoError(t, db.Delete(&record).Error)
				}
				// 先直接预览，确保不依赖用户先打开订单列表才能修复历史占用。
				_, err := PreviewInvoiceOrders(2202, []InvoiceOrderReference{ref})
				require.NoError(t, err)
				orders, err := GetRecentInvoiceOrders(2202)
				require.NoError(t, err)
				require.Len(t, orders, 1)
				assert.True(t, orders[0].InvoiceEligible)
				replacement, err := CreateCombinedInvoiceExternalPayment(2202, []InvoiceOrderReference{ref}, validCombinedInvoiceRequest(), InvoiceExternalPaymentOptions{PaymentProvider: PaymentProviderEpay, PaymentMethod: "alipay"})
				require.NoError(t, err)
				assert.Len(t, replacement.Orders, 1)
				require.NoError(t, db.Unscoped().First(&record, record.Id).Error)
				assert.Equal(t, deleted, record.DeletedAt.Valid)
			})
		}
	}
}

func TestUnclosedDeletedInvoiceStillPreventsReuse(t *testing.T) {
	for _, status := range []string{InvoiceStatusPending, InvoiceStatusIssued, InvoiceStatusPaymentPending} {
		t.Run(status, func(t *testing.T) {
			db := setupInvoiceOrderTestDB(t)
			topUp := recentInvoiceTopUp(2203, "DELETED-ACTIVE", 70)
			require.NoError(t, db.Create(&topUp).Error)
			record := InvoiceRecord{UserId: 2203, SourceType: InvoiceSourceTopUp, SourceId: topUp.TradeNo, Status: status}
			require.NoError(t, db.Create(&record).Error)
			require.NoError(t, db.Delete(&record).Error)
			_, err := PreviewInvoiceOrders(2203, []InvoiceOrderReference{{SourceType: InvoiceSourceTopUp, SourceId: topUp.TradeNo}})
			require.ErrorContains(t, err, "已经申请过发票")
		})
	}
}

func TestCloseDirectInvoiceReleasesSourceAndSubscriptionMirror(t *testing.T) {
	for _, source := range []string{InvoiceSourceTopUp, InvoiceSourceSubscription} {
		t.Run(source, func(t *testing.T) {
			db := setupInvoiceOrderTestDB(t)
			topUp := recentInvoiceTopUp(2204, "DIRECT-CLOSE", 70)
			topUp.InvoiceRequired = true
			topUp.InvoiceStatus = InvoiceStatusIssued
			topUp.InvoiceTitle = "原抬头"
			require.NoError(t, db.Create(&topUp).Error)
			if source == InvoiceSourceSubscription {
				require.NoError(t, db.Create(&SubscriptionOrder{
					UserId: 2204, TradeNo: topUp.TradeNo, Money: 70, ActualMoney: 70,
					PaymentMethod: "alipay", Status: common.TopUpStatusSuccess,
					CreateTime: topUp.CreateTime, CompleteTime: topUp.CompleteTime,
					InvoiceRequired: true, InvoiceStatus: InvoiceStatusIssued,
				}).Error)
			}
			record := InvoiceRecord{UserId: 2204, SourceType: source, SourceId: topUp.TradeNo, Status: InvoiceStatusIssued, PaymentStatus: InvoicePaymentStatusSuccess}
			require.NoError(t, db.Create(&record).Error)
			require.NoError(t, UpdateInvoiceRecord(record.Id, "", InvoiceStatusClosed, "作废后重新申请"))
			require.NoError(t, db.First(&topUp, topUp.Id).Error)
			assert.False(t, topUp.InvoiceRequired)
			assert.Empty(t, topUp.InvoiceStatus)
			assert.Empty(t, topUp.InvoiceTitle)
			_, err := PreviewInvoiceOrders(2204, []InvoiceOrderReference{{SourceType: source, SourceId: topUp.TradeNo}})
			require.NoError(t, err)
		})
	}
}
