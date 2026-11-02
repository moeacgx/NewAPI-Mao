package service

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// dispatchTicketNotificationEdits 运行于现有系统任务租约内；编辑可幂等重试，
// 不复用发送通知的未知结果终止策略，也不阻塞或回滚已经提交的人工回复。
func dispatchTicketNotificationEdits(ctx context.Context) error {
	if !model.DB.Migrator().HasTable(&model.TicketNotificationMessage{}) {
		return nil
	}
	now := time.Now().Unix()
	var expired []int
	if err := model.DB.Model(&model.TicketNotificationMessage{}).
		Where("created_at <= ?", now-model.TicketNotificationMessageTTL).
		Order("id ASC").Limit(notificationDispatchBatchSize).Pluck("id", &expired).Error; err != nil {
		return err
	}
	if len(expired) > 0 {
		if err := model.DB.Where("id IN ?", expired).Delete(&model.TicketNotificationMessage{}).Error; err != nil {
			return err
		}
	}
	var messages []model.TicketNotificationMessage
	if err := model.TicketNotificationEditQuery(now).Order("id ASC").Limit(notificationDispatchBatchSize).Find(&messages).Error; err != nil {
		return err
	}
	var passErr error
	for _, message := range messages {
		if err := ctx.Err(); err != nil {
			return errors.Join(passErr, err)
		}
		if message.AttemptCount >= notificationMaxAttempts {
			passErr = errors.Join(passErr, model.DB.Model(&model.TicketNotificationMessage{}).
				Where("id = ? AND attempt_count = ?", message.Id, message.AttemptCount).
				Updates(map[string]any{"status": model.NotificationDeliveryDead, "last_error": "ticket message edit retry limit reached after worker interruption"}).Error)
			continue
		}
		// CAS 同时保护手工诊断调用；中断后到期再编辑相同内容，不会新增消息。
		claimed := model.DB.Model(&model.TicketNotificationMessage{}).
			Where("id = ? AND attempt_count = ? AND next_attempt_at = ?", message.Id, message.AttemptCount, message.NextAttemptAt).
			Updates(map[string]any{"status": model.NotificationDeliveryRetrying, "attempt_count": message.AttemptCount + 1, "next_attempt_at": now + int64(notificationDeliveryLease/time.Second)})
		if claimed.Error != nil {
			passErr = errors.Join(passErr, claimed.Error)
			continue
		}
		if claimed.RowsAffected != 1 {
			continue
		}
		message.AttemptCount++
		var bot model.NotificationBot
		err := model.DB.First(&bot, message.BotId).Error
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && (!bot.Enabled || bot.Type != model.NotificationEndpointTypeTelegram)) {
			passErr = errors.Join(passErr, model.DB.Delete(&message).Error)
			continue
		}
		var token string
		if err == nil {
			token, err = model.NotificationBotToken(bot.Id)
		}
		if err != nil {
			err = &telegramDeliveryError{err: errors.New("notification bot unavailable for ticket message edit"), retryable: true}
		} else {
			_, err = telegramMessageContext(ctx, token, message.ChatId, model.NotificationTicketRepliedPrefix+message.Content, message.MessageId)
		}
		if err == nil {
			// 群内的勾选标记保留，后台成功后立即释放跟踪状态和原文快照。
			passErr = errors.Join(passErr, model.DB.Delete(&message).Error)
			continue
		}
		updates := map[string]any{"status": model.NotificationDeliveryDead, "last_error": err.Error()}
		var deliveryErr *telegramDeliveryError
		if errors.As(err, &deliveryErr) && deliveryErr.retryable && message.AttemptCount < notificationMaxAttempts {
			delay := notificationRetryDelay(message.AttemptCount)
			if deliveryErr.retryAfter > 0 {
				retrySeconds := deliveryErr.retryAfter
				if retrySeconds > 3600 {
					retrySeconds = 3600
				}
				delay = time.Duration(retrySeconds) * time.Second
			}
			updates["status"] = model.NotificationDeliveryRetrying
			updates["next_attempt_at"] = time.Now().Add(delay).Unix()
		}
		passErr = errors.Join(passErr, model.DB.Model(&model.TicketNotificationMessage{}).
			Where("id = ? AND attempt_count = ?", message.Id, message.AttemptCount).Updates(updates).Error)
	}
	return passErr
}
