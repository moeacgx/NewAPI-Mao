package model

import (
	"sort"
	"time"

	"gorm.io/gorm"
)

const TicketBatchMaxSize = 100

// TicketVersionRef 将用户确认时的版本带入批量写入，防止覆盖稍后发生的回复。
type TicketVersionRef struct {
	Id      int   `json:"id"`
	Version int64 `json:"version"`
}

// TicketActionLog 与工单事务一起提交，软删除后仍保留操作者与变更证据。
type TicketActionLog struct {
	Id         int64  `json:"id" gorm:"primaryKey"`
	TicketId   int    `json:"ticket_id" gorm:"not null;index"`
	ActorId    int    `json:"actor_id" gorm:"not null"`
	Action     string `json:"action" gorm:"type:varchar(32);not null"`
	FromStatus string `json:"from_status" gorm:"type:varchar(16);not null"`
	ToStatus   string `json:"to_status" gorm:"type:varchar(16);not null"`
	Version    int64  `json:"version" gorm:"not null"`
	CreatedAt  int64  `json:"created_at" gorm:"not null;index"`
}

func ChangeTicketsStatus(actorID int, refs []TicketVersionRef, status string) (int, error) {
	if !ValidTicketStatus(status) {
		return 0, ErrTicketInvalidInput
	}
	return mutateTicketBatch(actorID, refs, status, false)
}

func DeleteTickets(actorID int, refs []TicketVersionRef) (int, error) {
	return mutateTicketBatch(actorID, refs, "", true)
}

func mutateTicketBatch(actorID int, refs []TicketVersionRef, status string, deleting bool) (int, error) {
	if actorID <= 0 || len(refs) < 1 || len(refs) > TicketBatchMaxSize {
		return 0, ErrTicketInvalidInput
	}
	versions := make(map[int]int64, len(refs))
	ids := make([]int, 0, len(refs))
	for _, ref := range refs {
		if ref.Id <= 0 || ref.Version <= 0 {
			return 0, ErrTicketInvalidInput
		}
		if _, duplicate := versions[ref.Id]; duplicate {
			return 0, ErrTicketInvalidInput
		}
		versions[ref.Id] = ref.Version
		ids = append(ids, ref.Id)
	}
	sort.Ints(ids)
	changed := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var tickets []Ticket
		if err := lockForUpdate(tx).Where("id IN ?", ids).Order("id ASC").Find(&tickets).Error; err != nil {
			return err
		}
		if len(tickets) != len(ids) {
			return gorm.ErrRecordNotFound
		}
		for _, ticket := range tickets {
			if ticket.Version != versions[ticket.Id] {
				return ErrTicketConflict
			}
		}
		for _, ticket := range tickets {
			before := ticket.Status
			action := "batch_status"
			if deleting {
				now := nowUnix()
				result := tx.Model(&Ticket{}).Where("id = ? AND version = ?", ticket.Id, ticket.Version).Updates(map[string]any{
					"deleted_at": time.Unix(now, 0), "version": ticket.Version + 1, "updated_at": now,
				})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return ErrTicketConflict
				}
				ticket.Version++
				ticket.UpdatedAt = now
				action = "delete"
			} else {
				if ticket.Status == status {
					continue
				}
				if err := updateTicketStateTx(tx, &ticket, status); err != nil {
					return err
				}
				if err := enqueueTicketNotificationTx(tx, &ticket, NotificationEventTypeTicketStatusChanged, actorID, "admin"); err != nil {
					return err
				}
			}
			if err := tx.Create(&TicketActionLog{TicketId: ticket.Id, ActorId: actorID, Action: action, FromStatus: before, ToStatus: ticket.Status, Version: ticket.Version, CreatedAt: ticket.UpdatedAt}).Error; err != nil {
				return err
			}
			changed++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return changed, nil
}
