package model

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	SupplierApplicationStatusPending     = "pending"
	SupplierApplicationStatusContacted   = "contacted"
	SupplierApplicationStatusCooperating = "cooperating"
	SupplierApplicationStatusRejected    = "rejected"

	SupplierApplicationSiteNameMaxRunes   = 100
	SupplierApplicationSiteURLMaxRunes    = 500
	SupplierApplicationContactMaxRunes    = 200
	SupplierApplicationRemarkMaxRunes     = 1000
	SupplierApplicationAdminNoteMaxRunes  = 2000
	SupplierApplicationKeywordMaxRunes    = 100
	SupplierQuoteGroupNameMaxRunes        = 64
	SupplierQuoteModelMaxRunes            = 100
	SupplierQuoteDescriptionMaxRunes      = 500
	SupplierApplicationMaxGroups          = 20
	SupplierQuoteMaxModels                = 20
	SupplierQuoteRatioMin                 = 0.0001
	SupplierQuoteRatioMax                 = 1000.0
	supplierQuoteRatioScale               = 10000.0
	SupplierApplicationDailyLimit         = 20
	supplierApplicationDailyWindowSeconds = 24 * 60 * 60
	SupplierApplicationMaxPageSize        = 100
	supplierApplicationMaxPage            = 1000000
)

var (
	ErrSupplierApplicationInvalid   = errors.New("invalid supplier application input")
	ErrSupplierApplicationDuplicate = errors.New("supplier application already submitted")
	ErrSupplierApplicationLimit     = errors.New("supplier application daily limit reached")
	ErrSupplierApplicationConflict  = errors.New("supplier application changed, please refresh")
)

// 联系方式与平台枚举由本功能确定；倍率保留供应商填写的原始口径，不与本站计费倍率换算。
var (
	supplierContactTypes = map[string]bool{"wechat": true, "qq": true, "telegram": true, "email": true, "phone": true, "other": true}
	supplierPlatforms    = map[string]bool{"openai": true, "anthropic": true, "gemini": true, "grok": true, "deepseek": true, "other": true}
	// 只拦截明显的接入凭据形态；这是尽力而为的提示，不是密钥扫描器。
	supplierCredentialPattern = regexp.MustCompile(`(?i)(\bsk-[a-z0-9_-]{16,}|\bbearer\s+[a-z0-9._-]{16,}|(?:password|passwd|secret|api[_-]?key|access[_-]?token|authorization|密码|密钥)\s*[=:：]\s*\S)`)
)

// SupplierApplication 是一次不可修改的提交；管理员只更新处理状态和内部备注。
type SupplierApplication struct {
	Id          int    `json:"id" gorm:"primaryKey"`
	UserId      int    `json:"user_id" gorm:"not null;index:idx_supplier_app_user_created,priority:1;uniqueIndex:idx_supplier_app_user_fingerprint,priority:1"`
	SiteName    string `json:"site_name" gorm:"type:varchar(100);not null"`
	SiteUrl     string `json:"site_url" gorm:"type:varchar(500);not null"`
	ContactType string `json:"contact_type" gorm:"type:varchar(16);not null"`
	Contact     string `json:"contact" gorm:"type:varchar(200);not null"`
	Remark      string `json:"remark" gorm:"type:text;not null"`
	Status      string `json:"status" gorm:"type:varchar(16);not null;index"`
	AdminNote   string `json:"admin_note" gorm:"type:text;not null"`
	HandledBy   int    `json:"handled_by" gorm:"not null"`
	HandledAt   int64  `json:"handled_at" gorm:"not null"`
	Version     int64  `json:"version" gorm:"not null"`
	Fingerprint string `json:"-" gorm:"type:varchar(64);not null;uniqueIndex:idx_supplier_app_user_fingerprint,priority:2"`
	CreatedAt   int64  `json:"created_at" gorm:"not null;index;index:idx_supplier_app_user_created,priority:2"`
	UpdatedAt   int64  `json:"updated_at" gorm:"not null"`
}

type SupplierApplicationQuote struct {
	Id            int     `json:"id" gorm:"primaryKey"`
	ApplicationId int     `json:"application_id" gorm:"not null;index"`
	SortOrder     int     `json:"sort_order" gorm:"not null"`
	GroupName     string  `json:"group_name" gorm:"type:varchar(64);not null"`
	Platform      string  `json:"platform" gorm:"type:varchar(32);not null"`
	Ratio         float64 `json:"ratio" gorm:"not null"`
	ModelsJSON    string  `json:"-" gorm:"column:models;type:text;not null"`
	Description   string  `json:"description" gorm:"type:text;not null"`
}

// SupplierQuoteInput 的倍率使用指针以区分“未填写”和显式 0。
type SupplierQuoteInput struct {
	GroupName   string   `json:"group_name"`
	Platform    string   `json:"platform"`
	Ratio       *float64 `json:"ratio"`
	Models      []string `json:"models"`
	Description string   `json:"description"`
}

// SupplierApplicationInput 不含归属、状态、备注等字段，客户端无法指定它们。
type SupplierApplicationInput struct {
	SiteName    string               `json:"site_name"`
	SiteUrl     string               `json:"site_url"`
	ContactType string               `json:"contact_type"`
	Contact     string               `json:"contact"`
	Remark      string               `json:"remark"`
	Groups      []SupplierQuoteInput `json:"groups"`
}

type SupplierApplicationReview struct {
	Status    string `json:"status"`
	AdminNote string `json:"admin_note"`
	Version   int64  `json:"version"`
}

type SupplierApplicationFilter struct {
	Status  string
	Keyword string
	UserID  int
}

type SupplierQuoteView struct {
	Id          int      `json:"id"`
	GroupName   string   `json:"group_name"`
	Platform    string   `json:"platform"`
	Ratio       float64  `json:"ratio"`
	Models      []string `json:"models"`
	Description string   `json:"description"`
}

// SupplierApplicationUserView 是申请人可见的唯一形态：不含内部备注、处理人和处理时间。
type SupplierApplicationUserView struct {
	Id          int                 `json:"id"`
	SiteName    string              `json:"site_name"`
	SiteUrl     string              `json:"site_url"`
	ContactType string              `json:"contact_type"`
	Contact     string              `json:"contact"`
	Remark      string              `json:"remark"`
	Status      string              `json:"status"`
	CreatedAt   int64               `json:"created_at"`
	Groups      []SupplierQuoteView `json:"groups"`
}

// SupplierApplicationAdminView 面向有权限的管理员，包含完整资料和处理信息。
type SupplierApplicationAdminView struct {
	Id            int                 `json:"id"`
	UserId        int                 `json:"user_id"`
	Username      string              `json:"username"`
	SiteName      string              `json:"site_name"`
	SiteUrl       string              `json:"site_url"`
	ContactType   string              `json:"contact_type"`
	Contact       string              `json:"contact"`
	Remark        string              `json:"remark"`
	Status        string              `json:"status"`
	AdminNote     string              `json:"admin_note"`
	HandledBy     int                 `json:"handled_by"`
	HandledByName string              `json:"handled_by_name"`
	HandledAt     int64               `json:"handled_at"`
	Version       int64               `json:"version"`
	CreatedAt     int64               `json:"created_at"`
	UpdatedAt     int64               `json:"updated_at"`
	Groups        []SupplierQuoteView `json:"groups"`
}

func ValidSupplierApplicationStatus(status string) bool {
	switch status {
	case SupplierApplicationStatusPending, SupplierApplicationStatusContacted, SupplierApplicationStatusCooperating, SupplierApplicationStatusRejected:
		return true
	default:
		return false
	}
}

func supplierInvalid(field string) error {
	return fmt.Errorf("%w: %s", ErrSupplierApplicationInvalid, field)
}

// validSupplierText 校验纯文本：有效 UTF-8、无零字符、长度受限，且拒绝明显的接入凭据。
// 单行字段额外拒绝换行和其他控制字符。
func validSupplierText(value string, required bool, limit int, multiline bool) bool {
	if value == "" {
		return !required
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > limit || supplierCredentialPattern.MatchString(value) {
		return false
	}
	for _, r := range value {
		if r == '\n' || r == '\r' || r == '\t' {
			if !multiline {
				return false
			}
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// supplierSiteURLValid 只校验形态，不会请求该地址。拒绝内嵌账号密码的地址。
func supplierSiteURLValid(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return parsed.Hostname() != "" && parsed.User == nil && !strings.ContainsAny(raw, " \t\r\n")
}

// supplierRatioValid 要求有限正数，范围 [0.0001, 1000]，至多 4 位小数。
func supplierRatioValid(ratio float64) bool {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < SupplierQuoteRatioMin || ratio > SupplierQuoteRatioMax {
		return false
	}
	scaled := ratio * supplierQuoteRatioScale
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

// normalizeSupplierApplicationInput 去除首尾空白并完整校验；返回的错误只带字段路径，不回显用户文本。
func normalizeSupplierApplicationInput(input SupplierApplicationInput) (SupplierApplicationInput, error) {
	input.SiteName, input.SiteUrl = strings.TrimSpace(input.SiteName), strings.TrimSpace(input.SiteUrl)
	input.Contact, input.Remark = strings.TrimSpace(input.Contact), strings.TrimSpace(input.Remark)
	input.ContactType = strings.TrimSpace(input.ContactType)
	if !validSupplierText(input.SiteName, true, SupplierApplicationSiteNameMaxRunes, false) {
		return input, supplierInvalid("site_name")
	}
	if !validSupplierText(input.SiteUrl, false, SupplierApplicationSiteURLMaxRunes, false) || (input.SiteUrl != "" && !supplierSiteURLValid(input.SiteUrl)) {
		return input, supplierInvalid("site_url")
	}
	if !supplierContactTypes[input.ContactType] {
		return input, supplierInvalid("contact_type")
	}
	if !validSupplierText(input.Contact, true, SupplierApplicationContactMaxRunes, false) {
		return input, supplierInvalid("contact")
	}
	if !validSupplierText(input.Remark, false, SupplierApplicationRemarkMaxRunes, true) {
		return input, supplierInvalid("remark")
	}
	if len(input.Groups) < 1 || len(input.Groups) > SupplierApplicationMaxGroups {
		return input, supplierInvalid("groups")
	}
	groups := make([]SupplierQuoteInput, len(input.Groups))
	for index, group := range input.Groups {
		path := "groups[" + strconv.Itoa(index) + "]."
		group.GroupName, group.Platform = strings.TrimSpace(group.GroupName), strings.TrimSpace(group.Platform)
		group.Description = strings.TrimSpace(group.Description)
		if !validSupplierText(group.GroupName, true, SupplierQuoteGroupNameMaxRunes, false) {
			return input, supplierInvalid(path + "group_name")
		}
		if !supplierPlatforms[group.Platform] {
			return input, supplierInvalid(path + "platform")
		}
		if group.Ratio == nil || !supplierRatioValid(*group.Ratio) {
			return input, supplierInvalid(path + "ratio")
		}
		if !validSupplierText(group.Description, false, SupplierQuoteDescriptionMaxRunes, true) {
			return input, supplierInvalid(path + "description")
		}
		if len(group.Models) > SupplierQuoteMaxModels {
			return input, supplierInvalid(path + "models")
		}
		seen := make(map[string]bool, len(group.Models))
		models := make([]string, len(group.Models))
		for modelIndex, name := range group.Models {
			name = strings.TrimSpace(name)
			if !validSupplierText(name, true, SupplierQuoteModelMaxRunes, false) || seen[name] {
				return input, supplierInvalid(path + "models")
			}
			seen[name] = true
			models[modelIndex] = name
		}
		group.Models = models
		groups[index] = group
	}
	input.Groups = groups
	return input, nil
}

// 同一用户的相同内容只能保存一次；指纹基于规范化后的内容，网络重试和并发双击都会命中它。
func supplierApplicationFingerprint(input SupplierApplicationInput) (string, error) {
	data, err := common.Marshal(input)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func CreateSupplierApplication(userID int, input SupplierApplicationInput) (*SupplierApplicationUserView, error) {
	if userID <= 0 {
		return nil, ErrSupplierApplicationInvalid
	}
	input, err := normalizeSupplierApplicationInput(input)
	if err != nil {
		return nil, err
	}
	fingerprint, err := supplierApplicationFingerprint(input)
	if err != nil {
		return nil, err
	}
	now := nowUnix()
	application := &SupplierApplication{
		UserId: userID, SiteName: input.SiteName, SiteUrl: input.SiteUrl, ContactType: input.ContactType,
		Contact: input.Contact, Remark: input.Remark, Status: SupplierApplicationStatusPending,
		Version: 1, Fingerprint: fingerprint, CreatedAt: now, UpdatedAt: now,
	}
	quotes := make([]SupplierApplicationQuote, len(input.Groups))
	for index, group := range input.Groups {
		modelsJSON, marshalErr := common.Marshal(group.Models)
		if marshalErr != nil {
			return nil, marshalErr
		}
		quotes[index] = SupplierApplicationQuote{
			SortOrder: index, GroupName: group.GroupName, Platform: group.Platform,
			Ratio: *group.Ratio, ModelsJSON: string(modelsJSON), Description: group.Description,
		}
	}
	// 并发相同指纹提交时，SQLite 可能在无 busy_timeout 的测试环境中立即返回
	// SQLITE_BUSY；使用指数退避最多重试一次，给先到的事务时间提交。
	const maxBusyRetries = 4
	for attempt := 0; attempt <= maxBusyRetries; attempt++ {
		if attempt > 0 {
			application.Id = 0
			for i := range quotes {
				quotes[i].Id = 0
				quotes[i].ApplicationId = 0
			}
			time.Sleep(time.Duration(5*(1<<attempt)) * time.Millisecond)
		}
		err = DB.Transaction(func(tx *gorm.DB) error {
			var recent, duplicate int64
			if err := tx.Model(&SupplierApplication{}).Where("user_id = ? AND created_at > ?", userID, now-supplierApplicationDailyWindowSeconds).Count(&recent).Error; err != nil {
				return err
			}
			if recent >= SupplierApplicationDailyLimit {
				return ErrSupplierApplicationLimit
			}
			if err := tx.Model(&SupplierApplication{}).Where("user_id = ? AND fingerprint = ?", userID, fingerprint).Count(&duplicate).Error; err != nil {
				return err
			}
			if duplicate > 0 {
				return ErrSupplierApplicationDuplicate
			}
			if err := tx.Create(application).Error; err != nil {
				return err
			}
			for index := range quotes {
				quotes[index].ApplicationId = application.Id
			}
			return tx.Create(&quotes).Error
		})
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "SQLITE_BUSY") && !strings.Contains(err.Error(), "database is locked") {
			break
		}
	}
	if err != nil {
		// 并发提交相同内容时，唯一索引冲突或 SQLite 锁定会让事务回滚；事务外复查后统一报告为重复。
		// SQLITE_BUSY (database is locked) 也可能是并发重复提交的信号。
		if !errors.Is(err, ErrSupplierApplicationLimit) {
			var existing int64
			if checkErr := DB.Model(&SupplierApplication{}).Where("user_id = ? AND fingerprint = ?", userID, fingerprint).Count(&existing).Error; checkErr == nil && existing > 0 {
				return nil, ErrSupplierApplicationDuplicate
			}
		}
		return nil, err
	}
	return buildSupplierUserView(application, quotes), nil
}

func buildSupplierQuoteViews(quotes []SupplierApplicationQuote) []SupplierQuoteView {
	views := make([]SupplierQuoteView, len(quotes))
	for index, quote := range quotes {
		models := make([]string, 0)
		if quote.ModelsJSON != "" {
			// 历史或损坏的标签列表不应让整页不可读；解析失败时按无模型显示。
			if err := common.UnmarshalJsonStr(quote.ModelsJSON, &models); err != nil {
				models = make([]string, 0)
			}
		}
		views[index] = SupplierQuoteView{Id: quote.Id, GroupName: quote.GroupName, Platform: quote.Platform, Ratio: quote.Ratio, Models: models, Description: quote.Description}
	}
	return views
}

func buildSupplierUserView(application *SupplierApplication, quotes []SupplierApplicationQuote) *SupplierApplicationUserView {
	return &SupplierApplicationUserView{
		Id: application.Id, SiteName: application.SiteName, SiteUrl: application.SiteUrl,
		ContactType: application.ContactType, Contact: application.Contact, Remark: application.Remark,
		Status: application.Status, CreatedAt: application.CreatedAt, Groups: buildSupplierQuoteViews(quotes),
	}
}

// loadSupplierQuotes 一次查询取回一页申请的全部报价，按申请分组并保持提交顺序。
func loadSupplierQuotes(applicationIDs []int) (map[int][]SupplierApplicationQuote, error) {
	grouped := make(map[int][]SupplierApplicationQuote, len(applicationIDs))
	if len(applicationIDs) == 0 {
		return grouped, nil
	}
	var quotes []SupplierApplicationQuote
	if err := DB.Where("application_id IN ?", applicationIDs).Order("application_id ASC, sort_order ASC, id ASC").Find(&quotes).Error; err != nil {
		return nil, err
	}
	for _, quote := range quotes {
		grouped[quote.ApplicationId] = append(grouped[quote.ApplicationId], quote)
	}
	return grouped, nil
}

func supplierApplicationAccessQuery(db *gorm.DB, userID int, admin bool) *gorm.DB {
	query := db.Model(&SupplierApplication{})
	if !admin {
		query = query.Where("user_id = ?", userID)
	}
	return query
}

func listSupplierApplications(userID int, admin bool, filter SupplierApplicationFilter, page *common.PageInfo) ([]SupplierApplication, int64, error) {
	if userID <= 0 || page == nil || page.Page < 1 || page.Page > supplierApplicationMaxPage || page.PageSize < 1 || page.PageSize > SupplierApplicationMaxPageSize || filter.UserID < 0 {
		return nil, 0, ErrSupplierApplicationInvalid
	}
	query := supplierApplicationAccessQuery(DB, userID, admin)
	if admin && filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Status != "" {
		if !ValidSupplierApplicationStatus(filter.Status) {
			return nil, 0, ErrSupplierApplicationInvalid
		}
		query = query.Where("status = ?", filter.Status)
	}
	keyword := strings.TrimSpace(filter.Keyword)
	if admin && keyword != "" {
		if utf8.RuneCountInString(keyword) > SupplierApplicationKeywordMaxRunes || !utf8.ValidString(keyword) || strings.ContainsRune(keyword, 0) {
			return nil, 0, ErrSupplierApplicationInvalid
		}
		// 使用显式转义符，在三种数据库中均按字面量搜索百分号和下划线。
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(strings.ToLower(keyword)) + "%"
		match := "LOWER(site_name) LIKE ? ESCAPE '!' OR LOWER(site_url) LIKE ? ESCAPE '!' OR LOWER(contact) LIKE ? ESCAPE '!' OR id IN (SELECT application_id FROM supplier_application_quotes WHERE LOWER(group_name) LIKE ? ESCAPE '!')"
		args := []any{pattern, pattern, pattern, pattern}
		if numeric, err := strconv.Atoi(keyword); err == nil && numeric > 0 {
			match += " OR id = ? OR user_id = ?"
			args = append(args, numeric, numeric)
		}
		query = query.Where(match, args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]SupplierApplication, 0)
	err := query.Order("created_at DESC, id DESC").Offset(page.GetStartIdx()).Limit(page.PageSize).Find(&items).Error
	return items, total, err
}

func ListSupplierApplicationsForUser(userID int, filter SupplierApplicationFilter, page *common.PageInfo) ([]SupplierApplicationUserView, int64, error) {
	// 申请人只能按状态筛选自己的记录，搜索词和 user_id 不生效。
	filter.Keyword, filter.UserID = "", 0
	items, total, err := listSupplierApplications(userID, false, filter, page)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int, len(items))
	for index, item := range items {
		ids[index] = item.Id
	}
	quotes, err := loadSupplierQuotes(ids)
	if err != nil {
		return nil, 0, err
	}
	views := make([]SupplierApplicationUserView, len(items))
	for index := range items {
		views[index] = *buildSupplierUserView(&items[index], quotes[items[index].Id])
	}
	return views, total, nil
}

func GetSupplierApplicationForUser(applicationID, userID int) (*SupplierApplicationUserView, error) {
	if applicationID <= 0 || userID <= 0 {
		return nil, ErrSupplierApplicationInvalid
	}
	var application SupplierApplication
	if err := supplierApplicationAccessQuery(DB, userID, false).First(&application, applicationID).Error; err != nil {
		return nil, err
	}
	quotes, err := loadSupplierQuotes([]int{application.Id})
	if err != nil {
		return nil, err
	}
	return buildSupplierUserView(&application, quotes[application.Id]), nil
}

// buildSupplierAdminViews 批量解析申请人与处理人的用户名，避免逐行查询。
func buildSupplierAdminViews(applications []SupplierApplication) ([]SupplierApplicationAdminView, error) {
	ids := make([]int, len(applications))
	userIDs := make([]int, 0, len(applications)*2)
	for index, application := range applications {
		ids[index] = application.Id
		userIDs = append(userIDs, application.UserId)
		if application.HandledBy > 0 {
			userIDs = append(userIDs, application.HandledBy)
		}
	}
	quotes, err := loadSupplierQuotes(ids)
	if err != nil {
		return nil, err
	}
	usernames := make(map[int]string, len(userIDs))
	if len(userIDs) > 0 {
		var users []struct {
			Id       int
			Username string
		}
		if err := DB.Model(&User{}).Unscoped().Select("id, username").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, err
		}
		for _, user := range users {
			usernames[user.Id] = user.Username
		}
	}
	views := make([]SupplierApplicationAdminView, len(applications))
	for index, application := range applications {
		views[index] = SupplierApplicationAdminView{
			Id: application.Id, UserId: application.UserId, Username: usernames[application.UserId],
			SiteName: application.SiteName, SiteUrl: application.SiteUrl, ContactType: application.ContactType,
			Contact: application.Contact, Remark: application.Remark, Status: application.Status,
			AdminNote: application.AdminNote, HandledBy: application.HandledBy, HandledByName: usernames[application.HandledBy],
			HandledAt: application.HandledAt, Version: application.Version, CreatedAt: application.CreatedAt,
			UpdatedAt: application.UpdatedAt, Groups: buildSupplierQuoteViews(quotes[application.Id]),
		}
	}
	return views, nil
}

func ListSupplierApplicationsForAdmin(adminID int, filter SupplierApplicationFilter, page *common.PageInfo) ([]SupplierApplicationAdminView, int64, error) {
	items, total, err := listSupplierApplications(adminID, true, filter, page)
	if err != nil {
		return nil, 0, err
	}
	views, err := buildSupplierAdminViews(items)
	return views, total, err
}

func GetSupplierApplicationForAdmin(applicationID int) (*SupplierApplicationAdminView, error) {
	if applicationID <= 0 {
		return nil, ErrSupplierApplicationInvalid
	}
	var application SupplierApplication
	if err := DB.First(&application, applicationID).Error; err != nil {
		return nil, err
	}
	views, err := buildSupplierAdminViews([]SupplierApplication{application})
	if err != nil {
		return nil, err
	}
	return &views[0], nil
}

// SupplierApplicationReviewResult 返回更新结果和变更前的状态，供审计日志使用。
type SupplierApplicationReviewResult struct {
	View           *SupplierApplicationAdminView
	PreviousStatus string
	Changed        bool
}

// ReviewSupplierApplication 仅更新处理状态、内部备注、处理人和处理时间；
// 不创建渠道、不修改分组或计费配置。版本条件防止多人同时处理时互相覆盖。
func ReviewSupplierApplication(applicationID, adminID int, review SupplierApplicationReview) (*SupplierApplicationReviewResult, error) {
	review.AdminNote = strings.TrimSpace(review.AdminNote)
	if applicationID <= 0 || adminID <= 0 || review.Version <= 0 || !ValidSupplierApplicationStatus(review.Status) {
		return nil, ErrSupplierApplicationInvalid
	}
	if !validSupplierText(review.AdminNote, false, SupplierApplicationAdminNoteMaxRunes, true) {
		return nil, supplierInvalid("admin_note")
	}
	result := &SupplierApplicationReviewResult{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var application SupplierApplication
		if err := lockForUpdate(tx.Model(&SupplierApplication{})).First(&application, applicationID).Error; err != nil {
			return err
		}
		if application.Version != review.Version {
			return ErrSupplierApplicationConflict
		}
		result.PreviousStatus = application.Status
		if application.Status == review.Status && application.AdminNote == review.AdminNote {
			return nil
		}
		now := nowUnix()
		update := tx.Model(&SupplierApplication{}).Where("id = ? AND version = ?", application.Id, application.Version).Updates(map[string]any{
			"status": review.Status, "admin_note": review.AdminNote, "handled_by": adminID,
			"handled_at": now, "version": application.Version + 1, "updated_at": now,
		})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrSupplierApplicationConflict
		}
		result.Changed = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	result.View, err = GetSupplierApplicationForAdmin(applicationID)
	return result, err
}
