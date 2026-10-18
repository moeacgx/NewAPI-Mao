package model

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const UpstreamModelGuardNotificationEvent = "extension.upstream-model-guard.channel_disabled"

var ErrUpstreamModelGuardConfigConflict = errors.New("上游模型校验配置已更新，请刷新后重试")

type UpstreamModelGuardConfig struct {
	Id                     int    `json:"-" gorm:"primaryKey;autoIncrement:false"`
	ConfigVersion          int64  `json:"config_version" gorm:"not null"`
	Enabled                bool   `json:"enabled" gorm:"not null"`
	RulesJSON              string `json:"-" gorm:"type:text;not null"`
	ExcludedChannelIDsJSON string `json:"-" gorm:"type:text"`
	FailureThreshold       int    `json:"failure_threshold"`
	UpdatedAt              int64  `json:"updated_at" gorm:"bigint;not null"`
	UpdatedBy              int    `json:"updated_by"`
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
	DetectionSource            string   `json:"detection_source" gorm:"type:text"`
	RequestID                  string   `json:"request_id" gorm:"size:128"`
	Reason                     string   `json:"reason" gorm:"type:text;not null"`
	ConfigVersion              int64    `json:"config_version" gorm:"not null"`
	CreatedAt                  int64    `json:"created_at" gorm:"bigint;not null;index"`
	ConsecutiveMismatches      int      `json:"consecutive_mismatches"`
	FailureThreshold           int      `json:"failure_threshold"`
	ChannelDisabled            bool     `json:"channel_disabled"`
	ObservationKey             *string  `json:"-" gorm:"size:64;uniqueIndex:ux_upstream_guard_observation"`
}

// 所有启用规则的检测请求共用渠道计数，配置版本变化后从零累计。
type UpstreamModelGuardStreak struct {
	ChannelID             int   `gorm:"primaryKey;autoIncrement:false"`
	ConsecutiveMismatches int   `gorm:"not null"`
	ConfigVersion         int64 `gorm:"not null"`
}

// SQLite 旧表先增加普通可空列，避免 ADD COLUMN UNIQUE 失败；唯一索引交给 AutoMigrate。
func migrateSQLiteUpstreamModelGuardObservationKey() error {
	if DB == nil || DB.Dialector == nil || DB.Dialector.Name() != "sqlite" {
		return nil
	}
	migrator := DB.Migrator()
	if !migrator.HasTable(&UpstreamModelGuardRecord{}) || migrator.HasColumn(&UpstreamModelGuardRecord{}, "ObservationKey") {
		return nil
	}
	return DB.Exec("ALTER TABLE `upstream_model_guard_records` ADD COLUMN `observation_key` text").Error
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
		config = UpstreamModelGuardConfig{Id: 1, ConfigVersion: 1, RulesJSON: "[]", ExcludedChannelIDsJSON: "[]", FailureThreshold: 2}
		if err = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&config).Error; err == nil {
			err = db.First(&config, "id = ?", 1).Error
		}
	}
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 2
	}
	if strings.TrimSpace(config.ExcludedChannelIDsJSON) == "" {
		config.ExcludedChannelIDsJSON = "[]"
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
			"excluded_channel_ids_json": config.ExcludedChannelIDsJSON, "failure_threshold": config.FailureThreshold,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrUpstreamModelGuardConfigConflict
	}
	return nil
}

// ObserveUpstreamModelGuard 在同一事务中累计请求、记录异常及按阈值停用渠道。
// ObservationKey 由宿主对内部请求身份与渠道 ID 计算 SHA-256，不信任客户端请求 ID。
func ObserveUpstreamModelGuard(ctx context.Context, record *UpstreamModelGuardRecord, matched bool) (bool, error) {
	if record == nil || record.ChannelID <= 0 || record.ConfigVersion <= 0 {
		return false, errors.New("上游模型校验禁用记录无效")
	}
	if record.ObservationKey == nil || len(*record.ObservationKey) != 64 {
		return false, errors.New("上游模型校验请求身份无效")
	}
	if _, err := hex.DecodeString(*record.ObservationKey); err != nil {
		return false, errors.New("上游模型校验请求身份无效")
	}
	if record.DetectionSource == "" {
		record.DetectionSource = constant.UpstreamModelSourceResponseBody
	}
	if matched {
		var streak UpstreamModelGuardStreak
		result := DB.WithContext(ctx).Select("channel_id").
			Where("channel_id = ? AND config_version = ? AND consecutive_mismatches > ?", record.ChannelID, record.ConfigVersion, 0).
			Limit(1).Find(&streak)
		if result.Error != nil {
			return false, result.Error
		}
		if result.RowsAffected == 0 {
			// 没有当前版本的非零计数时，该只读查询就是匹配请求的线性化点。
			// 随后发生的不匹配保留自己的计数，正常请求无需锁住全局配置行。
			return false, nil
		}
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
		var excluded []int
		if strings.TrimSpace(config.ExcludedChannelIDsJSON) != "" {
			if err := common.UnmarshalJsonStr(config.ExcludedChannelIDsJSON, &excluded); err != nil {
				return err
			}
		}
		for _, channelID := range excluded {
			if channelID == record.ChannelID {
				return nil
			}
		}
		var channel Channel
		if err := lockForUpdate(tx).Select("id", "name", "status", "other_info").First(&channel, "id = ?", record.ChannelID).Error; err != nil {
			return err
		}
		if channel.Status != common.ChannelStatusEnabled {
			return nil
		}
		var existing int64
		if err := tx.Model(&UpstreamModelGuardRecord{}).Where("observation_key = ?", *record.ObservationKey).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return nil
		}
		if matched {
			return tx.Where("channel_id = ?", channel.Id).Delete(&UpstreamModelGuardStreak{}).Error
		}
		var streak UpstreamModelGuardStreak
		if err := tx.Where("channel_id = ?", channel.Id).Limit(1).Find(&streak).Error; err != nil {
			return err
		}
		if streak.ConfigVersion != config.ConfigVersion {
			streak.ConsecutiveMismatches = 0
		}
		streak.ChannelID = channel.Id
		streak.ConfigVersion = config.ConfigVersion
		streak.ConsecutiveMismatches++
		record.ConsecutiveMismatches = streak.ConsecutiveMismatches
		record.FailureThreshold = config.FailureThreshold
		if record.FailureThreshold <= 0 {
			record.FailureThreshold = 2
		}
		record.ChannelDisabled = record.ConsecutiveMismatches >= record.FailureThreshold
		now := time.Now().Unix()
		record.Id = 0
		record.ChannelName = channel.Name
		record.CreatedAt = now
		record.Reason = fmt.Sprintf("上游模型校验连续不匹配 %d/%d：请求模型 %q，预期 %s，实际 %q", record.ConsecutiveMismatches, record.FailureThreshold, record.RequestedModel, strings.Join(record.ExpectedUpstreamModels, ", "), record.ActualUpstreamModel)
		detectionSourceName := "响应正文"
		if record.DetectionSource == constant.UpstreamModelSourceCodexFasterModel {
			detectionSourceName = "Codex faster-model 响应头"
			record.Reason = fmt.Sprintf("上游模型校验连续不匹配 %d/%d：请求模型 %q，预期 %s，%s声明模型 %q", record.ConsecutiveMismatches, record.FailureThreshold, record.RequestedModel, strings.Join(record.ExpectedUpstreamModels, ", "), detectionSourceName, record.ActualUpstreamModel)
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
		if !record.ChannelDisabled {
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "channel_id"}}, DoUpdates: clause.AssignmentColumns([]string{"consecutive_mismatches", "config_version"})}).Create(&streak).Error
		}
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
			return errors.New("渠道状态已变化，请重试模型校验")
		}
		if err := tx.Model(&Ability{}).Where("channel_id = ?", channel.Id).Update("enabled", false).Error; err != nil {
			return err
		}
		if err := tx.Where("channel_id = ?", channel.Id).Delete(&UpstreamModelGuardStreak{}).Error; err != nil {
			return err
		}
		eventKey := "upstream-model-guard:" + strconv.Itoa(record.Id)
		payload := map[string]any{
			"channel_id": channel.Id, "channel_name": channel.Name,
			"group": record.GroupName, "requested_model": record.RequestedModel,
			"expected_upstream_models": strings.Join(record.ExpectedUpstreamModels, ", "),
			"actual_upstream_model":    record.ActualUpstreamModel, "request_id": record.RequestID,
			"detection_source": detectionSourceName,
			"reason":           record.Reason, "create_time": time.Unix(now, 0).Format(time.RFC3339),
			"comparison": upstreamModelGuardComparison(record), "module_id": "upstream-model-guard",
			"event_type": UpstreamModelGuardNotificationEvent, "event_key": eventKey,
			"consecutive_mismatches": record.ConsecutiveMismatches, "failure_threshold": record.FailureThreshold,
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
	if record.DetectionSource == constant.UpstreamModelSourceCodexFasterModel {
		parts[3].label = "头部声明模型"
		parts = append(parts, struct{ label, value string }{"检测来源", "Codex faster-model 响应头"})
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
	fmt.Fprintf(&result, "\n连续不匹配: %d/%d", record.ConsecutiveMismatches, record.FailureThreshold)
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
		if records[index].DetectionSource == "" {
			records[index].DetectionSource = constant.UpstreamModelSourceResponseBody
		}
		if records[index].ObservationKey == nil {
			records[index].ConsecutiveMismatches = 1
			records[index].FailureThreshold = 1
			records[index].ChannelDisabled = true
		}
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
