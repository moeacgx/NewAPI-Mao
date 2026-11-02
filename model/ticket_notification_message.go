package model

import (
	"errors"

	"gorm.io/gorm"
)

const TicketNotificationMessageTTL = 3 * 24 * 60 * 60

// TicketNotificationMessage 独立保留原消息位置，不受通知中心五条历史清理影响。
// 不保存凭据，正文只供后台编辑，不通过公共工单接口暴露。
type TicketNotificationMessage struct {
	Id            int    `gorm:"primaryKey"`
	DeliveryId    int    `gorm:"not null;uniqueIndex"`
	TaskId        int    `gorm:"not null;default:0;index"`
	TicketId      int    `gorm:"not null;index"`
	UserMessageId int    `gorm:"not null"`
	BotId         int    `gorm:"not null"`
	ChatId        string `gorm:"type:varchar(128);not null"`
	MessageId     int    `gorm:"not null"`
	Content       string `gorm:"type:text;not null"`
	Status        string `gorm:"type:varchar(16);not null;index"`
	AttemptCount  int    `gorm:"not null"`
	NextAttemptAt int64  `gorm:"not null;index"`
	LastError     string `gorm:"type:text"`
	CreatedAt     int64  `gorm:"not null;index"`
}

// CompleteTicketNotificationDelivery 将发送回执和原投递成功状态一起提交。
// 外部 Telegram 与本地数据库无法组成事务；回执落库失败时调用方留失败记录，禁止重新发消息。
func CompleteTicketNotificationDelivery(message *TicketNotificationMessage) error {
	err := DB.Transaction(func(tx *gorm.DB) error {
		// 与规则停用/删除串行提交，HTTP 回包不能重新写入刚被清理的关联。
		if err := LockNotificationSequenceTx(tx); err != nil {
			return err
		}
		var delivery NotificationDelivery
		err := tx.Select("task_id", "event_id").Where("id = ? AND status = ?", message.DeliveryId, NotificationDeliverySending).Take(&delivery).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notificationDeliveryStateError(message.DeliveryId)
		}
		if err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&NotificationTask{}).
			Where("id = ? AND enabled = ? AND active_after_event_id < ? AND event_type = ?", delivery.TaskId, true, delivery.EventId, NotificationEventTypeTicketUserMessage).
			Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			message.TaskId = delivery.TaskId
			if err := tx.Create(message).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&NotificationDelivery{}).Where("id = ? AND status = ?", message.DeliveryId, NotificationDeliverySending).
			Updates(map[string]any{"status": NotificationDeliverySuccess, "sent_at": message.CreatedAt, "updated_at": message.CreatedAt, "last_error": ""})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return notificationDeliveryStateError(message.DeliveryId)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// ClearTicketNotificationTaskMessagesTx 按稳定规则归属清理，不能依赖已被裁剪的投递历史。
// 通知表可先于工单关联表启用；尚未迁移时没有需要清理的工单回执。
func ClearTicketNotificationTaskMessagesTx(tx *gorm.DB, taskID int) error {
	if tx == nil || taskID <= 0 {
		return errors.New("invalid ticket notification task cleanup")
	}
	if !tx.Migrator().HasTable(&TicketNotificationMessage{}) {
		return nil
	}
	return tx.Where("task_id = ?", taskID).Delete(&TicketNotificationMessage{}).Error
}

// TicketNotificationEditQuery 按消息顺序判定回复，排除创建时作者为 0 的自动确认。
// 在发送后持续核对，避免人工回复与原消息投递交错时遗漏。
func TicketNotificationEditQuery(now int64) *gorm.DB {
	replies := DB.Model(&TicketMessage{}).Select("1").
		Where("ticket_messages.ticket_id = ticket_notification_messages.ticket_id AND ticket_messages.id > ticket_notification_messages.user_message_id").
		Where("author_role = ? AND author_id > ?", "admin", 0)
	tickets := DB.Model(&Ticket{}).Select("1").Where("tickets.id = ticket_notification_messages.ticket_id")
	return DB.Model(&TicketNotificationMessage{}).
		Where("status IN ? AND next_attempt_at <= ? AND created_at > ?", []string{NotificationDeliveryPending, NotificationDeliveryRetrying}, now, now-TicketNotificationMessageTTL).
		Where("EXISTS (?) AND EXISTS (?)", replies, tickets)
}

func HasTicketNotificationEditWork(now int64) (bool, error) {
	if DB == nil || !DB.Migrator().HasTable(&TicketNotificationMessage{}) {
		return false, nil
	}
	var ids []int
	if err := TicketNotificationEditQuery(now).Limit(1).Pluck("id", &ids).Error; err != nil {
		return false, err
	}
	if len(ids) > 0 {
		return true, nil
	}
	err := DB.Model(&TicketNotificationMessage{}).Where("created_at <= ?", now-TicketNotificationMessageTTL).Limit(1).Pluck("id", &ids).Error
	return len(ids) > 0, err
}
