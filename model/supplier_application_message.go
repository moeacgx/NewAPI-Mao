package model

import (
	"errors"
	"math"
	"strings"

	"gorm.io/gorm"
)

const (
	SupplierMessageTypeMessage     = "message"
	SupplierMessageTypeNegotiation = "negotiation"
	SupplierMessageTypeOffer       = "offer"
	SupplierMessageTypeTestQuota   = "test_quota"

	SupplierMessageAuthorApplicant = "applicant"
	SupplierMessageAuthorAdmin     = "admin"

	SupplierTestQuotaStatusPending = "pending"
	SupplierTestQuotaStatusIssued  = "issued"

	SupplierMessageBodyMaxRunes   = 2000
	SupplierMessageNoteMaxRunes   = 1000
	SupplierTestQuotaUnitMaxRunes = 32
	SupplierTestQuotaQuantityMin  = 0.0001
	SupplierTestQuotaQuantityMax  = 1000000000.0
)

// SupplierApplicationMessage 是申请下的一条不可修改对话记录。
// 议价、优惠和测试额度都只保存文本，不增加余额、不生成兑换码、不创建渠道，也不修改分组或计费。
type SupplierApplicationMessage struct {
	Id            int     `json:"id" gorm:"primaryKey"`
	ApplicationId int     `json:"application_id" gorm:"not null;index:idx_supplier_app_message_order,priority:1"`
	AuthorId      int     `json:"author_id" gorm:"not null"`
	AuthorRole    string  `json:"author_role" gorm:"type:varchar(16);not null"`
	MessageType   string  `json:"type" gorm:"column:message_type;type:varchar(32);not null"`
	Body          string  `json:"body" gorm:"type:text;not null"`
	GroupName     string  `json:"group_name" gorm:"type:varchar(64);not null"`
	Ratio         float64 `json:"ratio" gorm:"not null"`
	Quantity      float64 `json:"quantity" gorm:"not null"`
	Unit          string  `json:"unit" gorm:"type:varchar(32);not null"`
	QuotaStatus   string  `json:"quota_status" gorm:"type:varchar(16);not null"`
	CreatedAt     int64   `json:"created_at" gorm:"not null;index:idx_supplier_app_message_order,priority:2"`
}

// SupplierApplicationMessageInput 只接收对话内容。归属、时间和申请身份由服务端决定。
type SupplierApplicationMessageInput struct {
	Type        string   `json:"type"`
	Body        string   `json:"body"`
	GroupName   string   `json:"group_name"`
	Ratio       *float64 `json:"ratio"`
	Quantity    *float64 `json:"quantity"`
	Unit        string   `json:"unit"`
	QuotaStatus string   `json:"quota_status"`
}

// SupplierApplicationMessageView 对申请人和有读权限的管理员使用同一形态，不含内部备注。
type SupplierApplicationMessageView struct {
	Id          int      `json:"id"`
	AuthorId    int      `json:"author_id"`
	AuthorRole  string   `json:"author_role"`
	AuthorName  string   `json:"author_name"`
	Type        string   `json:"type"`
	Body        string   `json:"body"`
	GroupName   string   `json:"group_name,omitempty"`
	Ratio       *float64 `json:"ratio,omitempty"`
	Quantity    *float64 `json:"quantity,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	QuotaStatus string   `json:"quota_status,omitempty"`
	CreatedAt   int64    `json:"created_at"`
}

// SupplierApplicationMessageResult 只把审计需要的标识返回给控制器，不包含联系方式。
type SupplierApplicationMessageResult struct {
	View        *SupplierApplicationMessageView
	ApplicantID int
}

type normalizedSupplierMessage struct {
	Type        string
	Body        string
	GroupName   string
	Ratio       *float64
	Quantity    *float64
	Unit        string
	QuotaStatus string
}

func supplierQuantityValid(quantity float64) bool {
	if math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity < SupplierTestQuotaQuantityMin || quantity > SupplierTestQuotaQuantityMax {
		return false
	}
	scaled := quantity * supplierQuoteRatioScale
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

// normalizeSupplierMessage 按类型保留该类型的字段，并拒绝明显的接入凭据。错误只含字段路径。
func normalizeSupplierMessage(input SupplierApplicationMessageInput) (normalizedSupplierMessage, error) {
	normalized := normalizedSupplierMessage{Type: strings.TrimSpace(input.Type)}
	body := strings.TrimSpace(input.Body)
	groupName := strings.TrimSpace(input.GroupName)
	unit := strings.TrimSpace(input.Unit)
	status := strings.TrimSpace(input.QuotaStatus)
	switch normalized.Type {
	case SupplierMessageTypeMessage:
		if input.Ratio != nil || input.Quantity != nil || groupName != "" || unit != "" || status != "" {
			return normalized, supplierInvalid("type")
		}
		if !validSupplierText(body, true, SupplierMessageBodyMaxRunes, true) {
			return normalized, supplierInvalid("body")
		}
		normalized.Body = body
	case SupplierMessageTypeNegotiation:
		if input.Quantity != nil || unit != "" || status != "" {
			return normalized, supplierInvalid("type")
		}
		if !validSupplierText(groupName, true, SupplierQuoteGroupNameMaxRunes, false) {
			return normalized, supplierInvalid("group_name")
		}
		if input.Ratio == nil || !supplierRatioValid(*input.Ratio) {
			return normalized, supplierInvalid("ratio")
		}
		if !validSupplierText(body, true, SupplierMessageNoteMaxRunes, true) {
			return normalized, supplierInvalid("body")
		}
		normalized.GroupName = groupName
		normalized.Ratio = input.Ratio
		normalized.Body = body
	case SupplierMessageTypeOffer:
		if input.Ratio != nil || input.Quantity != nil || groupName != "" || unit != "" || status != "" {
			return normalized, supplierInvalid("type")
		}
		if !validSupplierText(body, true, SupplierMessageBodyMaxRunes, true) {
			return normalized, supplierInvalid("body")
		}
		normalized.Body = body
	case SupplierMessageTypeTestQuota:
		if input.Ratio != nil || groupName != "" {
			return normalized, supplierInvalid("type")
		}
		if input.Quantity == nil || !supplierQuantityValid(*input.Quantity) {
			return normalized, supplierInvalid("quantity")
		}
		if !validSupplierText(unit, true, SupplierTestQuotaUnitMaxRunes, false) {
			return normalized, supplierInvalid("unit")
		}
		if status != SupplierTestQuotaStatusPending && status != SupplierTestQuotaStatusIssued {
			return normalized, supplierInvalid("quota_status")
		}
		if !validSupplierText(body, true, SupplierMessageNoteMaxRunes, true) {
			return normalized, supplierInvalid("body")
		}
		normalized.Quantity = input.Quantity
		normalized.Unit = unit
		normalized.QuotaStatus = status
		normalized.Body = body
	default:
		return normalized, supplierInvalid("type")
	}
	return normalized, nil
}

func supplierMessageView(row SupplierApplicationMessage, authorName string) SupplierApplicationMessageView {
	view := SupplierApplicationMessageView{
		Id: row.Id, AuthorId: row.AuthorId, AuthorRole: row.AuthorRole, AuthorName: authorName,
		Type: row.MessageType, Body: row.Body, CreatedAt: row.CreatedAt,
	}
	switch row.MessageType {
	case SupplierMessageTypeNegotiation:
		view.GroupName = row.GroupName
		ratio := row.Ratio
		view.Ratio = &ratio
	case SupplierMessageTypeTestQuota:
		quantity := row.Quantity
		view.Quantity = &quantity
		view.Unit = row.Unit
		view.QuotaStatus = row.QuotaStatus
	}
	return view
}

func loadSupplierMessageViews(applicationIDs []int) (map[int][]SupplierApplicationMessageView, error) {
	grouped := make(map[int][]SupplierApplicationMessageView, len(applicationIDs))
	for _, id := range applicationIDs {
		grouped[id] = []SupplierApplicationMessageView{}
	}
	if len(applicationIDs) == 0 {
		return grouped, nil
	}
	var rows []SupplierApplicationMessage
	if err := DB.Where("application_id IN ?", applicationIDs).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	names := map[int]string{}
	if len(rows) > 0 {
		authorIDs := make([]int, 0, len(rows))
		seen := map[int]bool{}
		for _, row := range rows {
			if row.AuthorId > 0 && !seen[row.AuthorId] {
				seen[row.AuthorId] = true
				authorIDs = append(authorIDs, row.AuthorId)
			}
		}
		var users []struct {
			Id       int
			Username string
		}
		if err := DB.Model(&User{}).Unscoped().Select("id, username").Where("id IN ?", authorIDs).Find(&users).Error; err != nil {
			return nil, err
		}
		for _, user := range users {
			names[user.Id] = user.Username
		}
	}
	for _, row := range rows {
		grouped[row.ApplicationId] = append(grouped[row.ApplicationId], supplierMessageView(row, names[row.AuthorId]))
	}
	return grouped, nil
}

func attachUserSupplierMessages(views []SupplierApplicationUserView) error {
	ids := make([]int, len(views))
	for index := range views {
		ids[index] = views[index].Id
	}
	grouped, err := loadSupplierMessageViews(ids)
	if err != nil {
		return err
	}
	for index := range views {
		views[index].Messages = grouped[views[index].Id]
	}
	return nil
}

func attachAdminSupplierMessages(views []SupplierApplicationAdminView) error {
	ids := make([]int, len(views))
	for index := range views {
		ids[index] = views[index].Id
	}
	grouped, err := loadSupplierMessageViews(ids)
	if err != nil {
		return err
	}
	for index := range views {
		views[index].Messages = grouped[views[index].Id]
	}
	return nil
}

// CreateSupplierApplicationMessage 只追加一条记录。调用方不能借此修改申请、用户余额或站点配置。
func CreateSupplierApplicationMessage(applicationID, authorID int, admin bool, input SupplierApplicationMessageInput) (*SupplierApplicationMessageResult, error) {
	if applicationID <= 0 || authorID <= 0 {
		return nil, ErrSupplierApplicationInvalid
	}
	var application SupplierApplication
	query := DB
	if !admin {
		query = query.Where("user_id = ?", authorID)
	}
	if err := query.First(&application, applicationID).Error; err != nil {
		return nil, err
	}
	normalized, err := normalizeSupplierMessage(input)
	if err != nil {
		return nil, err
	}
	role := SupplierMessageAuthorApplicant
	if admin {
		role = SupplierMessageAuthorAdmin
	}
	row := SupplierApplicationMessage{
		ApplicationId: application.Id, AuthorId: authorID, AuthorRole: role, MessageType: normalized.Type,
		Body: normalized.Body, GroupName: normalized.GroupName, Unit: normalized.Unit, QuotaStatus: normalized.QuotaStatus,
		CreatedAt: nowUnix(),
	}
	if normalized.Ratio != nil {
		row.Ratio = *normalized.Ratio
	}
	if normalized.Quantity != nil {
		row.Quantity = *normalized.Quantity
	}
	if err := DB.Create(&row).Error; err != nil {
		return nil, err
	}
	var author User
	if err := DB.Unscoped().Select("id, username").First(&author, authorID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	view := supplierMessageView(row, author.Username)
	return &SupplierApplicationMessageResult{View: &view, ApplicantID: application.UserId}, nil
}
