package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// releaseClosedInvoiceSourcesTx 保留关闭申请的明细，并释放来源订单和唯一占用。
// 非空快照同时标记释放已完成，重复保存旧申请不能再次清理新申请的来源状态。
func releaseClosedInvoiceSourcesTx(tx *gorm.DB, record *InvoiceRecord) error {
	if record.Status != InvoiceStatusClosed || record.ClosedOrdersSnapshot != "" {
		return nil
	}
	links := make([]InvoiceOrderLink, 0)
	if record.SourceType == InvoiceSourceCombined {
		if err := tx.Where("invoice_id = ?", record.Id).Order("id asc").Find(&links).Error; err != nil {
			return err
		}
	}
	snapshot, err := common.Marshal(links)
	if err != nil {
		return err
	}
	sources := links
	if record.SourceType != InvoiceSourceCombined {
		sources = []InvoiceOrderLink{{SourceType: record.SourceType, SourceId: record.SourceId}}
	}
	updates := map[string]interface{}{
		"invoice_required":    false,
		"invoice_type":        "",
		"invoice_kind":        "",
		"invoice_title":       "",
		"invoice_tax_no":      "",
		"invoice_email":       "",
		"invoice_phone":       "",
		"invoice_remark":      "",
		"invoice_base_amount": 0,
		"invoice_fee_amount":  0,
		"invoice_status":      "",
	}
	for _, source := range sources {
		switch source.SourceType {
		case InvoiceSourceTopUp:
			if err := tx.Model(&TopUp{}).
				Where("user_id = ? AND trade_no = ? AND invoice_status = ?", record.UserId, source.SourceId, InvoiceStatusClosed).
				Updates(updates).Error; err != nil {
				return err
			}
		case InvoiceSourceSubscription:
			if err := tx.Model(&SubscriptionOrder{}).
				Where("user_id = ? AND trade_no = ? AND invoice_status = ?", record.UserId, source.SourceId, InvoiceStatusClosed).
				Updates(updates).Error; err != nil {
				return err
			}
			// 直接申请与合并申请都需同时释放订阅的充值镜像。
			if err := tx.Model(&TopUp{}).
				Where("user_id = ? AND trade_no = ? AND invoice_status = ?", record.UserId, source.SourceId, InvoiceStatusClosed).
				Updates(updates).Error; err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown invoice source type %s", source.SourceType)
		}
	}
	if err := tx.Where("invoice_id = ?", record.Id).Delete(&InvoiceOrderLink{}).Error; err != nil {
		return err
	}
	record.ClosedOrdersSnapshot = string(snapshot)
	return tx.Unscoped().Model(&InvoiceRecord{}).Where("id = ?", record.Id).
		Update("closed_orders_snapshot", record.ClosedOrdersSnapshot).Error
}

// releaseHistoricalClosedInvoiceSourcesTx 按用户惰性修复旧版关闭申请，包含软删除记录。
func releaseHistoricalClosedInvoiceSourcesTx(tx *gorm.DB, userId int) error {
	var records []InvoiceRecord
	if err := lockForUpdate(tx.Unscoped()).
		Where("user_id = ? AND status = ?", userId, InvoiceStatusClosed).
		Where("closed_orders_snapshot IS NULL OR closed_orders_snapshot = ?", "").
		Order("id asc").Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		if err := releaseClosedInvoiceSourcesTx(tx, &records[i]); err != nil {
			return err
		}
	}
	return nil
}
