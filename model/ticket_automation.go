package model

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// TicketAutoReplyOptionKey 是管理员工单自动回复配置在 Option 表中的稳定键。
const TicketAutoReplyOptionKey = "TicketAutoReplyConfig"

// TicketAutoReplyConfig 控制用户创建工单时是否追加一条系统管理员回复。
// 自动回复只在工单创建事务内写入一次，后续不会因为配置变化重新发送。
type TicketAutoReplyConfig struct {
	Enabled bool   `json:"enabled"`
	Content string `json:"content"`
}

var errTicketAutoReplyConfigInvalid = errors.New("invalid ticket auto-reply configuration")

func defaultTicketAutoReplyConfig() TicketAutoReplyConfig {
	return TicketAutoReplyConfig{}
}

func normalizeTicketAutoReplyConfig(config TicketAutoReplyConfig) (TicketAutoReplyConfig, error) {
	config.Content = strings.TrimSpace(config.Content)
	if config.Content != "" && (!utf8.ValidString(config.Content) || strings.ContainsRune(config.Content, 0) || utf8.RuneCountInString(config.Content) > TicketContentMaxRunes) {
		return TicketAutoReplyConfig{}, errTicketAutoReplyConfigInvalid
	}
	if config.Enabled && config.Content == "" {
		return TicketAutoReplyConfig{}, errTicketAutoReplyConfigInvalid
	}
	return config, nil
}

func decodeTicketAutoReplyConfig(value string) TicketAutoReplyConfig {
	config := defaultTicketAutoReplyConfig()
	if common.UnmarshalJsonStr(value, &config) != nil {
		// 配置损坏时关闭自动回复，避免阻塞用户创建工单或意外发送未知内容。
		return defaultTicketAutoReplyConfig()
	}
	config, err := normalizeTicketAutoReplyConfig(config)
	if err != nil {
		return defaultTicketAutoReplyConfig()
	}
	return config
}

// GetTicketAutoReplyConfig 返回当前配置。未迁移 Option 表或未配置时保持关闭，
// 使旧版本数据库能够继续创建工单。
func GetTicketAutoReplyConfig() (TicketAutoReplyConfig, error) {
	return getTicketAutoReplyConfigTx(DB)
}

func getTicketAutoReplyConfigTx(tx *gorm.DB) (TicketAutoReplyConfig, error) {
	config := defaultTicketAutoReplyConfig()
	if tx == nil || !tx.Migrator().HasTable(&Option{}) {
		return config, nil
	}
	var option Option
	// 使用结构条件让 GORM 按当前方言引用保留字 key，避免测试或嵌入式数据库尚未初始化列名缓存。
	err := tx.Where(&Option{Key: TicketAutoReplyOptionKey}).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return config, nil
	}
	if err != nil {
		return config, err
	}
	return decodeTicketAutoReplyConfig(option.Value), nil
}

// UpdateTicketAutoReplyConfig 原子保存自动回复配置。配置只允许管理员路由调用。
func UpdateTicketAutoReplyConfig(config TicketAutoReplyConfig) (TicketAutoReplyConfig, error) {
	normalized, err := normalizeTicketAutoReplyConfig(config)
	if err != nil {
		return TicketAutoReplyConfig{}, ErrTicketInvalidInput
	}
	value, err := common.Marshal(normalized)
	if err != nil {
		return TicketAutoReplyConfig{}, err
	}
	if err := UpdateOption(TicketAutoReplyOptionKey, string(value)); err != nil {
		return TicketAutoReplyConfig{}, err
	}
	return normalized, nil
}
