package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	TicketStatusOpen       = "open"
	TicketStatusProcessing = "processing"
	TicketStatusResolved   = "resolved"
	TicketStatusClosed     = "closed"
	TicketStatusArchived   = "archived"
	TicketTitleMaxRunes    = 120
	TicketContentMaxRunes  = 10000
	TicketMessagePageSize  = 50
)

var (
	ErrTicketInvalidInput = errors.New("invalid ticket input")
	ErrTicketClosed       = errors.New("ticket is closed")
	ErrTicketConflict     = errors.New("ticket changed, please refresh")
)

type Ticket struct {
	Id        int            `json:"id" gorm:"primaryKey"`
	UserId    int            `json:"user_id" gorm:"not null;index:idx_ticket_user_updated,priority:1"`
	Title     string         `json:"title" gorm:"type:varchar(120);not null"`
	Category  string         `json:"category" gorm:"type:varchar(16);not null"`
	Priority  string         `json:"priority" gorm:"type:varchar(16);not null"`
	Status    string         `json:"status" gorm:"type:varchar(16);not null;index"`
	Version   int64          `json:"version" gorm:"not null"`
	CreatedAt int64          `json:"created_at" gorm:"not null"`
	UpdatedAt int64          `json:"updated_at" gorm:"not null;index;index:idx_ticket_user_updated,priority:2"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type TicketMessage struct {
	Id         int    `json:"id" gorm:"primaryKey;index:idx_ticket_message_history,priority:2"`
	TicketId   int    `json:"ticket_id" gorm:"not null;index:idx_ticket_message_history,priority:1"`
	AuthorId   int    `json:"author_id" gorm:"not null"`
	AuthorRole string `json:"author_role" gorm:"type:varchar(16);not null"`
	Content    string `json:"content" gorm:"type:text;not null"`
	CreatedAt  int64  `json:"created_at" gorm:"not null"`
}

type TicketCreateInput struct {
	Title    string `json:"title"`
	Category string `json:"category"`
	Priority string `json:"priority"`
	Content  string `json:"content"`
}

type TicketFilter struct {
	Status  string
	Keyword string
	UserID  int
}

type TicketDetail struct {
	Ticket   Ticket          `json:"ticket"`
	Messages []TicketMessage `json:"messages"`
	HasMore  bool            `json:"has_more"`
}

// 工单正文保持纯文本；拒绝 PostgreSQL 不支持的零字符。
func validTicketText(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(value) <= limit
}

func ValidTicketStatus(status string) bool {
	switch status {
	case TicketStatusOpen, TicketStatusProcessing, TicketStatusResolved, TicketStatusClosed, TicketStatusArchived:
		return true
	default:
		return false
	}
}

func CreateTicket(userID int, input TicketCreateInput) (*Ticket, error) {
	input.Title, input.Content = strings.TrimSpace(input.Title), strings.TrimSpace(input.Content)
	if userID <= 0 || !validTicketText(input.Title, TicketTitleMaxRunes) || !validTicketText(input.Content, TicketContentMaxRunes) {
		return nil, ErrTicketInvalidInput
	}
	switch input.Category {
	case "general", "billing", "technical", "account":
	default:
		return nil, ErrTicketInvalidInput
	}
	switch input.Priority {
	case "low", "normal", "high":
	default:
		return nil, ErrTicketInvalidInput
	}
	now := nowUnix()
	ticket := &Ticket{UserId: userID, Title: input.Title, Category: input.Category, Priority: input.Priority, Status: TicketStatusOpen, Version: 1, CreatedAt: now, UpdatedAt: now}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		message := &TicketMessage{TicketId: ticket.Id, AuthorId: userID, AuthorRole: "user", Content: input.Content, CreatedAt: now}
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		autoReply, err := getTicketAutoReplyConfigTx(tx)
		if err != nil {
			return err
		}
		if autoReply.Enabled {
			// 自动回复和首条用户消息处于同一事务，创建成功时最多写入这一条。
			autoReplyMessage := &TicketMessage{
				TicketId: ticket.Id, AuthorId: 0, AuthorRole: "admin",
				Content: autoReply.Content, CreatedAt: now,
			}
			if err := tx.Create(autoReplyMessage).Error; err != nil {
				return err
			}
		}
		return enqueueTicketNotificationTx(tx, ticket, NotificationEventTypeTicketCreated, userID, "user")
	})
	return ticket, err
}

// 个人查询不根据用户角色扩大权限；管理范围必须由独立受保护接口明确传入。
func ticketAccessQuery(db *gorm.DB, userID int, admin bool) *gorm.DB {
	query := db.Model(&Ticket{})
	if !admin {
		query = query.Where("user_id = ?", userID)
	}
	return query
}

func ListTickets(userID int, admin bool, filter TicketFilter, page *common.PageInfo) ([]Ticket, int64, error) {
	if userID <= 0 || page == nil || page.Page < 1 || page.Page > 1000000 || page.PageSize < 1 || page.PageSize > 100 || filter.UserID < 0 {
		return nil, 0, ErrTicketInvalidInput
	}
	query := ticketAccessQuery(DB, userID, admin)
	if admin && filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		if !ValidTicketStatus(filter.Status) {
			return nil, 0, ErrTicketInvalidInput
		}
		query = query.Where("status = ?", filter.Status)
	}
	keyword := strings.TrimSpace(filter.Keyword)
	if keyword != "" {
		if !validTicketText(keyword, TicketTitleMaxRunes) {
			return nil, 0, ErrTicketInvalidInput
		}
		// 使用显式转义符，在三种数据库中均按字面量搜索百分号和下划线。
		keyword = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(keyword))
		query = query.Where("LOWER(title) LIKE ? ESCAPE '!'", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]Ticket, 0)
	err := query.Order("updated_at DESC, id DESC").Offset(page.GetStartIdx()).Limit(page.PageSize).Find(&items).Error
	return items, total, err
}

func GetTicketDetail(ticketID, userID int, admin bool, beforeMessageID int) (*TicketDetail, error) {
	if ticketID <= 0 || userID <= 0 || beforeMessageID < 0 {
		return nil, ErrTicketInvalidInput
	}
	detail := &TicketDetail{Messages: make([]TicketMessage, 0)}
	if err := ticketAccessQuery(DB, userID, admin).First(&detail.Ticket, ticketID).Error; err != nil {
		return nil, err
	}
	query := DB.Where("ticket_id = ?", ticketID)
	if beforeMessageID > 0 {
		query = query.Where("id < ?", beforeMessageID)
	}
	if err := query.Order("id DESC").Limit(TicketMessagePageSize + 1).Find(&detail.Messages).Error; err != nil {
		return nil, err
	}
	detail.HasMore = len(detail.Messages) > TicketMessagePageSize
	if detail.HasMore {
		detail.Messages = detail.Messages[:TicketMessagePageSize]
	}
	for left, right := 0, len(detail.Messages)-1; left < right; left, right = left+1, right-1 {
		detail.Messages[left], detail.Messages[right] = detail.Messages[right], detail.Messages[left]
	}
	return detail, nil
}

func ReplyTicket(ticketID, userID int, admin bool, content string) (*TicketMessage, error) {
	content = strings.TrimSpace(content)
	if ticketID <= 0 || userID <= 0 || !validTicketText(content, TicketContentMaxRunes) {
		return nil, ErrTicketInvalidInput
	}
	message := &TicketMessage{TicketId: ticketID, AuthorId: userID, AuthorRole: "user", Content: content, CreatedAt: nowUnix()}
	status, eventType := TicketStatusOpen, NotificationEventTypeTicketUserReplied
	if admin {
		message.AuthorRole = "admin"
		status, eventType = TicketStatusProcessing, NotificationEventTypeTicketAdminReplied
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var ticket Ticket
		if err := lockForUpdate(ticketAccessQuery(tx, userID, admin)).First(&ticket, ticketID).Error; err != nil {
			return err
		}
		if ticket.Status == TicketStatusClosed || ticket.Status == TicketStatusArchived {
			return ErrTicketClosed
		}
		if err := updateTicketStateTx(tx, &ticket, status); err != nil {
			return err
		}
		if err := tx.Create(message).Error; err != nil {
			return err
		}
		return enqueueTicketNotificationTx(tx, &ticket, eventType, userID, message.AuthorRole)
	})
	return message, err
}

func ChangeTicketStatus(ticketID, userID int, admin bool, status string) (*Ticket, error) {
	if ticketID <= 0 || userID <= 0 || !ValidTicketStatus(status) || (!admin && status != TicketStatusOpen && status != TicketStatusClosed) {
		return nil, ErrTicketInvalidInput
	}
	actorRole := "user"
	if admin {
		actorRole = "admin"
	}
	ticket := &Ticket{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(ticketAccessQuery(tx, userID, admin)).First(ticket, ticketID).Error; err != nil {
			return err
		}
		if ticket.Status == status {
			return nil
		}
		if err := updateTicketStateTx(tx, ticket, status); err != nil {
			return err
		}
		return enqueueTicketNotificationTx(tx, ticket, NotificationEventTypeTicketStatusChanged, userID, actorRole)
	})
	return ticket, err
}

// 版本条件防止无行锁数据库上把并发更新覆盖为旧状态。
func updateTicketStateTx(tx *gorm.DB, ticket *Ticket, status string) error {
	now := nowUnix()
	result := tx.Model(&Ticket{}).Where("id = ? AND version = ?", ticket.Id, ticket.Version).Updates(map[string]any{
		"status": status, "version": ticket.Version + 1, "updated_at": now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrTicketConflict
	}
	ticket.Status, ticket.UpdatedAt = status, now
	ticket.Version++
	return nil
}

func enqueueTicketNotificationTx(tx *gorm.DB, ticket *Ticket, eventType string, actorID int, actorRole string) error {
	return EnqueueNotificationEventTx(tx, eventType, fmt.Sprintf("ticket:%d:version:%d", ticket.Id, ticket.Version), map[string]any{
		"ticket_id": ticket.Id, "title": ticket.Title, "user_id": ticket.UserId,
		"category": ticket.Category, "priority": ticket.Priority, "status": ticket.Status,
		"actor_id": actorID, "actor_role": actorRole, "create_time": ticket.UpdatedAt,
	})
}
