package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const UpstreamModelGuardNotificationEvent = "extension.upstream-model-guard.channel_disabled"

var ErrUpstreamModelGuardConfigConflict = errors.New("上游模型校验配置已更新，请刷新后重试")

type UpstreamModelGuardConfig struct {
	Id            int    `json:"-" gorm:"primaryKey;autoIncrement:false"`
	ConfigVersion int64  `json:"config_version" gorm:"not null"`
	Enabled       bool   `json:"enabled" gorm:"not null"`
	RulesJSON     string `json:"-" gorm:"type:text;not null"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;not null"`
	UpdatedBy     int    `json:"updated_by"`
}

// GroupIDs 保存稳定身份，GroupCodes 仅供已删除分组的历史配置展示。
type UpstreamModelGuardStoredRule struct {
	Enabled        bool     `json:"enabled"`
	GroupIDs       []int    `json:"group_ids"`
	GroupCodes     []string `json:"group_codes"`
	Model          string   `json:"model"`
	UpstreamModels []string `json:"upstream_models"`
}

type UpstreamModelGuardRecord struct {
	Id                         int      `json:"id" gorm:"primaryKey"`
	ChannelID                  int      `json:"channel_id" gorm:"not null;index"`
	ChannelName                string   `json:"channel_name" gorm:"type:text;not null"`
	GroupID                    int      `json:"group_id" gorm:"not null;index"`
	Group                      string   `json:"group" gorm:"size:64;not null"`
	GroupName                  string   `json:"group_name" gorm:"-"`
	RequestedModel             string   `json:"requested_model" gorm:"type:text;not null"`
	ExpectedUpstreamModelsJSON string   `json:"-" gorm:"type:text;not null"`
	ExpectedUpstreamModels     []string `json:"expected_upstream_models" gorm:"-"`
	ActualUpstreamModel        string   `json:"actual_upstream_model" gorm:"type:text;not null"`
	RequestID                  string   `json:"request_id" gorm:"size:128"`
	Reason                     string   `json:"reason" gorm:"type:text;not null"`
	ConfigVersion              int64    `json:"config_version" gorm:"not null"`
	CreatedAt                  int64    `json:"created_at" gorm:"bigint;not null;index"`
}

func EnsureUpstreamModelGuardConfig() error {
	_, err := LoadUpstreamModelGuardConfig(context.Background())
	return err
}

func LoadUpstreamModelGuardConfig(ctx context.Context) (*UpstreamModelGuardConfig, error) {
	var config UpstreamModelGuardConfig
	db := DB.WithContext(ctx)
	err := db.First(&config, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		config = UpstreamModelGuardConfig{Id: 1, ConfigVersion: 1, RulesJSON: "[]"}
		if err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&config).Error; err == nil {
			err = db.First(&config, "id = ?", 1).Error
		}
	}
	return &config, err
}

func SaveUpstreamModelGuardConfig(ctx context.Context, expectedVersion int64, config *UpstreamModelGuardConfig) error {
	if config == nil || expectedVersion <= 0 {
		return errors.New("上游模型校验配置版本无效")
	}
	if _, err := LoadUpstreamModelGuardConfig(ctx); err != nil {
		return err
	}
	config.Id = 1
	config.ConfigVersion = expectedVersion + 1
	config.UpdatedAt = time.Now().Unix()
	result := DB.WithContext(ctx).Model(&UpstreamModelGuardConfig{}).
		Where("id = ? AND config_version = ?", 1, expectedVersion).
		Updates(map[string]any{
			"config_version": config.ConfigVersion, "enabled": config.Enabled,
			"rules_json": config.RulesJSON, "updated_at": config.UpdatedAt, "updated_by": config.UpdatedBy,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrUpstreamModelGuardConfigConflict
	}
	return nil
}

// DisableChannelForUpstreamModelGuard 原子停用整条渠道，保留所有密钥和独立配置。
func DisableChannelForUpstreamModelGuard(ctx context.Context, record *UpstreamModelGuardRecord) (bool, error) {
	if record == nil || record.ChannelID <= 0 || record.ConfigVersion <= 0 {
		return false, errors.New("上游模型校验禁用记录无效")
	}
	changed := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var config UpstreamModelGuardConfig
		if err := lockForUpdate(tx).First(&config, "id = ?", 1).Error; err != nil {
			return err
		}
		if !config.Enabled || config.ConfigVersion != record.ConfigVersion {
			return nil
		}
		var channel Channel
		if err := lockForUpdate(tx).Select("id", "name", "status", "other_info").First(&channel, "id = ?", record.ChannelID).Error; err != nil {
			return err
		}
		if channel.Status != common.ChannelStatusEnabled {
			return nil
		}
		now := time.Now().Unix()
		record.Id = 0
		record.ChannelName = channel.Name
		record.CreatedAt = now
		record.Reason = fmt.Sprintf("上游模型校验失败：请求模型 %q，预期 %s，实际 %q", record.RequestedModel, strings.Join(record.ExpectedUpstreamModels, ", "), record.ActualUpstreamModel)
		other := make(map[string]any)
		if strings.TrimSpace(channel.OtherInfo) != "" {
			if err := common.UnmarshalJsonStr(channel.OtherInfo, &other); err != nil {
				return err
			}
			if other == nil {
				other = make(map[string]any)
			}
		}
		other["status_reason"] = record.Reason
		other["status_time"] = now
		other["upstream_model_guard"] = true
		otherJSON, err := common.Marshal(other)
		if err != nil {
			return err
		}
		result := tx.Model(&Channel{}).Where("id = ? AND status = ?", channel.Id, common.ChannelStatusEnabled).
			Updates(map[string]any{"status": common.ChannelStatusManuallyDisabled, "other_info": string(otherJSON)})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return nil
		}
		if err := tx.Model(&Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", false).Error; err != nil {
			return err
		}
		expectedJSON, err := common.Marshal(record.ExpectedUpstreamModels)
		if err != nil {
			return err
		}
		record.ExpectedUpstreamModelsJSON = string(expectedJSON)
		// 使用稳定身份解析名称，避免同 code 重建分组后把新名称归到旧记录。
		var group Group
		if err := tx.Select("name").Where("id = ?", record.GroupID).Limit(1).Find(&group).Error; err != nil {
			return err
		}
		record.GroupName = strings.TrimSpace(group.Name)
		if record.GroupName == "" {
			record.GroupName = record.Group
		}
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		eventKey := "upstream-model-guard:" + strconv.Itoa(record.Id)
		payload := map[string]any{
			"channel_id": channel.Id, "channel_name": channel.Name,
			"group": record.GroupName, "requested_model": record.RequestedModel,
			"expected_upstream_models": strings.Join(record.ExpectedUpstreamModels, ", "),
			"actual_upstream_model":    record.ActualUpstreamModel, "request_id": record.RequestID,
			"reason": record.Reason, "create_time": time.Unix(now, 0).Format(time.RFC3339),
			"comparison": upstreamModelGuardComparison(record), "module_id": "upstream-model-guard",
			"event_type": UpstreamModelGuardNotificationEvent, "event_key": eventKey,
		}
		if _, err := EnqueueNotificationEventTxWithResult(tx, UpstreamModelGuardNotificationEvent, eventKey, payload); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	if changed {
		CacheUpdateChannelStatus(record.ChannelID, common.ChannelStatusManuallyDisabled)
		InvalidatePricingCache()
	}
	return changed, nil
}

// 每个比较维度独立限长，避免允许列表过长挤掉实际模型。
func upstreamModelGuardComparison(record *UpstreamModelGuardRecord) string {
	parts := []struct{ label, value string }{
		{"分组", record.GroupName}, {"请求模型", record.RequestedModel},
		{"允许上游模型", strings.Join(record.ExpectedUpstreamModels, ", ")}, {"实际上游模型", record.ActualUpstreamModel},
	}
	var result strings.Builder
	for index, part := range parts {
		value := part.value
		characters := 0
		for offset := range value {
			if characters == 227 {
				value = value[:offset] + "..."
				break
			}
			characters++
		}
		if index > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(part.label)
		result.WriteString(": ")
		result.WriteString(value)
	}
	return result.String()
}

func ListUpstreamModelGuardRecords(ctx context.Context, page, pageSize int) ([]UpstreamModelGuardRecord, int64, error) {
	if page < 1 || page > 1000000 || pageSize < 1 || pageSize > 100 {
		return nil, 0, errors.New("分页参数无效")
	}
	query := DB.WithContext(ctx).Model(&UpstreamModelGuardRecord{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	records := make([]UpstreamModelGuardRecord, 0)
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	groupIDs := make([]int, 0, len(records))
	for _, record := range records {
		groupIDs = append(groupIDs, record.GroupID)
	}
	groupNames := make(map[int]string, len(groupIDs))
	if len(groupIDs) > 0 {
		var groups []Group
		if err := DB.WithContext(ctx).Select("id", "name").Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
			return nil, 0, err
		}
		for _, group := range groups {
			groupNames[group.Id] = strings.TrimSpace(group.Name)
		}
	}
	for index := range records {
		records[index].GroupName = groupNames[records[index].GroupID]
		if records[index].GroupName == "" {
			records[index].GroupName = records[index].Group
		}
		if err := common.UnmarshalJsonStr(records[index].ExpectedUpstreamModelsJSON, &records[index].ExpectedUpstreamModels); err != nil {
			return nil, 0, err
		}
	}
	return records, total, nil
}

// IsChannelEnabledAfterUpstreamModelGuard 绕过节点内存缓存，以数据库状态作为最终选渠门禁。
func IsChannelEnabledAfterUpstreamModelGuard(ctx context.Context, channelID int) (bool, error) {
	if channelID <= 0 {
		return false, nil
	}
	var channel Channel
	err := DB.WithContext(ctx).Select("status").First(&channel, "id = ?", channelID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil && channel.Status == common.ChannelStatusEnabled, err
}
