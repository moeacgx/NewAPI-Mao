package model

import (
	"context"
	"errors"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const TicketAutoArchiveOptionKey = "TicketAutoArchiveConfig"

type TicketAutoArchiveConfig struct {
	Enabled      bool     `json:"enabled"`
	InactiveDays int      `json:"inactive_days"`
	Statuses     []string `json:"statuses"`
}

func normalizeTicketAutoArchiveConfig(config TicketAutoArchiveConfig) (TicketAutoArchiveConfig, error) {
	if config.InactiveDays < 1 || config.InactiveDays > 3650 || len(config.Statuses) == 0 || len(config.Statuses) > 4 {
		return TicketAutoArchiveConfig{}, ErrTicketInvalidInput
	}
	seen := make(map[string]bool, len(config.Statuses))
	statuses := make([]string, 0, len(config.Statuses))
	for _, status := range config.Statuses {
		if !ValidTicketStatus(status) || status == TicketStatusArchived || seen[status] {
			return TicketAutoArchiveConfig{}, ErrTicketInvalidInput
		}
		seen[status] = true
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	config.Statuses = statuses
	return config, nil
}

func GetTicketAutoArchiveConfig() (TicketAutoArchiveConfig, error) {
	return getTicketAutoArchiveConfigTx(DB)
}

func getTicketAutoArchiveConfigTx(tx *gorm.DB) (TicketAutoArchiveConfig, error) {
	fallback := TicketAutoArchiveConfig{InactiveDays: 30, Statuses: []string{TicketStatusResolved, TicketStatusClosed}}
	var option Option
	err := tx.Where(&Option{Key: TicketAutoArchiveOptionKey}).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fallback, nil
	}
	if err != nil {
		return fallback, err
	}
	var config TicketAutoArchiveConfig
	if err = common.UnmarshalJsonStr(option.Value, &config); err != nil {
		return fallback, ErrTicketInvalidInput
	}
	return normalizeTicketAutoArchiveConfig(config)
}

func UpdateTicketAutoArchiveConfig(config TicketAutoArchiveConfig) (TicketAutoArchiveConfig, error) {
	normalized, err := normalizeTicketAutoArchiveConfig(config)
	if err != nil {
		return TicketAutoArchiveConfig{}, err
	}
	data, err := common.Marshal(normalized)
	if err != nil {
		return TicketAutoArchiveConfig{}, err
	}
	if err = UpdateOption(TicketAutoArchiveOptionKey, string(data)); err != nil {
		return TicketAutoArchiveConfig{}, err
	}
	return normalized, nil
}

// ArchiveInactiveTickets 扫描快照只用于找候选；每笔重新锁定并检查配置和活动时间。
// 已归档与软删除记录不会再次命中；人工回复/状态变化会延后归档。
func ArchiveInactiveTickets(ctx context.Context, now int64, limit int) (int, error) {
	if now <= 0 || limit < 1 || limit > TicketBatchMaxSize {
		return 0, ErrTicketInvalidInput
	}
	db := DB.WithContext(ctx)
	config, err := getTicketAutoArchiveConfigTx(db)
	if err != nil || !config.Enabled {
		return 0, err
	}
	cutoff := now - int64(config.InactiveDays)*86400
	var ids []int
	if err = db.Model(&Ticket{}).Where("status IN ? AND updated_at <= ?", config.Statuses, cutoff).Order("id ASC").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	archived := 0
	for _, id := range ids {
		changed := false
		err = db.Transaction(func(tx *gorm.DB) error {
			// 配置行也锁定，保存关闭开关与本笔归档有清晰的先后顺序。
			current, err := getTicketAutoArchiveConfigTx(lockForUpdate(tx))
			if err != nil || !current.Enabled {
				return err
			}
			var ticket Ticket
			if err = lockForUpdate(tx).First(&ticket, id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			} else if err != nil {
				return err
			}
			eligible := false
			for _, status := range current.Statuses {
				if ticket.Status == status {
					eligible = true
					break
				}
			}
			if !eligible || ticket.UpdatedAt > now-int64(current.InactiveDays)*86400 {
				return nil
			}
			before := ticket.Status
			if err = updateTicketStateTx(tx, &ticket, TicketStatusArchived); err != nil {
				return err
			}
			if err = tx.Create(&TicketActionLog{TicketId: ticket.Id, Action: "auto_archive", FromStatus: before, ToStatus: ticket.Status, Version: ticket.Version, CreatedAt: ticket.UpdatedAt}).Error; err != nil {
				return err
			}
			changed = true
			return nil
		})
		if err != nil {
			return archived, err
		}
		if changed {
			archived++
		}
	}
	return archived, nil
}
