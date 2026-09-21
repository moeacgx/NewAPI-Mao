package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

const UpstreamModelGuardModuleID = "upstream-model-guard"

var upstreamModelGuardModuleState struct {
	sync.RWMutex
	enabled func() bool
}

// RegisterUpstreamModelGuardModuleEnabledCheck 由扩展宿主注入状态查询，避免服务与扩展循环依赖。
func RegisterUpstreamModelGuardModuleEnabledCheck(check func() bool) {
	upstreamModelGuardModuleState.Lock()
	upstreamModelGuardModuleState.enabled = check
	upstreamModelGuardModuleState.Unlock()
}

type UpstreamModelGuardRule struct {
	Enabled        bool     `json:"enabled"`
	GroupCodes     []string `json:"group_codes"`
	Model          string   `json:"model"`
	UpstreamModels []string `json:"upstream_models"`
	groupIDs       []int
}

type UpstreamModelGuardConfig struct {
	ConfigVersion      int64                       `json:"config_version"`
	Enabled            bool                        `json:"enabled"`
	Rules              []UpstreamModelGuardRule    `json:"rules"`
	UpdatedAt          int64                       `json:"updated_at"`
	UpdatedBy          int                         `json:"updated_by"`
	ExcludedChannelIDs []int                       `json:"excluded_channel_ids"`
	ExcludedChannels   []UpstreamModelGuardChannel `json:"excluded_channels"`
	FailureThreshold   int                         `json:"failure_threshold"`
}

// UpstreamModelGuardChannel 仅用于白名单选择，不能包含渠道凭据或地址。
type UpstreamModelGuardChannel struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

func ListUpstreamModelGuardChannels(ctx context.Context, keyword string, page, pageSize int) ([]UpstreamModelGuardChannel, int64, error) {
	query := model.DB.WithContext(ctx).Model(&model.Channel{})
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		if id, err := strconv.Atoi(keyword); err == nil && id > 0 {
			query = query.Where("id = ? OR name LIKE ?", id, "%"+keyword+"%")
		} else {
			query = query.Where("name LIKE ?", "%"+keyword+"%")
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	channels := make([]UpstreamModelGuardChannel, 0)
	err := query.Select("id", "name", "status").Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&channels).Error
	return channels, total, err
}

type UpstreamModelGuardConfigUpdate struct {
	ExpectedVersion    int64                    `json:"expected_version"`
	Enabled            bool                     `json:"enabled"`
	Rules              []UpstreamModelGuardRule `json:"rules"`
	ExcludedChannelIDs *[]int                   `json:"excluded_channel_ids"`
	FailureThreshold   *int                     `json:"failure_threshold"`
}

func IsUpstreamModelGuardModuleEnabled() bool {
	upstreamModelGuardModuleState.RLock()
	check := upstreamModelGuardModuleState.enabled
	upstreamModelGuardModuleState.RUnlock()
	return check != nil && check()
}

func GetUpstreamModelGuardConfig(ctx context.Context) (*UpstreamModelGuardConfig, error) {
	stored, err := model.LoadUpstreamModelGuardConfig(ctx)
	if err != nil {
		return nil, err
	}
	var rules []model.UpstreamModelGuardStoredRule
	if err := common.UnmarshalJsonStr(stored.RulesJSON, &rules); err != nil {
		return nil, err
	}
	var groups []model.Group
	if len(rules) > 0 {
		if err := model.DB.WithContext(ctx).Select("id", "code").Find(&groups).Error; err != nil {
			return nil, err
		}
	}
	codes := make(map[int]string, len(groups))
	for _, group := range groups {
		codes[group.Id] = group.Code
	}
	config := &UpstreamModelGuardConfig{
		ConfigVersion: stored.ConfigVersion, Enabled: stored.Enabled,
		Rules: make([]UpstreamModelGuardRule, 0, len(rules)), UpdatedAt: stored.UpdatedAt, UpdatedBy: stored.UpdatedBy,
		ExcludedChannelIDs: []int{}, ExcludedChannels: []UpstreamModelGuardChannel{}, FailureThreshold: stored.FailureThreshold,
	}
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 2
	}
	if strings.TrimSpace(stored.ExcludedChannelIDsJSON) != "" {
		if err := common.UnmarshalJsonStr(stored.ExcludedChannelIDsJSON, &config.ExcludedChannelIDs); err != nil {
			return nil, err
		}
	}
	if len(config.ExcludedChannelIDs) > 0 {
		if err := model.DB.WithContext(ctx).Model(&model.Channel{}).Select("id", "name", "status").Where("id IN ?", config.ExcludedChannelIDs).Order("id ASC").Find(&config.ExcludedChannels).Error; err != nil {
			return nil, err
		}
	}
	for _, rule := range rules {
		groupCodes := make([]string, 0, len(rule.GroupIDs))
		groupIDs := append([]int(nil), rule.GroupIDs...)
		for index, id := range rule.GroupIDs {
			code, ok := codes[id]
			if !ok && index < len(rule.GroupCodes) {
				code = rule.GroupCodes[index]
				groupIDs[index] = 0
			}
			groupCodes = append(groupCodes, code)
		}
		config.Rules = append(config.Rules, UpstreamModelGuardRule{
			Enabled: rule.Enabled, GroupCodes: groupCodes, Model: rule.Model,
			UpstreamModels: rule.UpstreamModels, groupIDs: groupIDs,
		})
	}
	return config, nil
}

func SaveUpstreamModelGuardConfig(ctx context.Context, update UpstreamModelGuardConfigUpdate, userID int) (*UpstreamModelGuardConfig, error) {
	if update.ExpectedVersion <= 0 {
		return nil, errors.New("expected_version 必须大于 0")
	}
	if len(update.Rules) > 100 {
		return nil, errors.New("校验规则最多 100 条")
	}
	previous, err := GetUpstreamModelGuardConfig(ctx)
	if err != nil {
		return nil, err
	}
	threshold := previous.FailureThreshold
	if update.FailureThreshold != nil {
		threshold = *update.FailureThreshold
	}
	if threshold < 1 || threshold > 100 {
		return nil, errors.New("连续不匹配次数必须为 1 至 100 的整数")
	}
	excluded := previous.ExcludedChannelIDs
	if update.ExcludedChannelIDs != nil {
		if len(*update.ExcludedChannelIDs) > 1000 {
			return nil, errors.New("渠道白名单最多 1000 条")
		}
		excluded = make([]int, 0, len(*update.ExcludedChannelIDs))
		seenIDs := make(map[int]bool)
		for _, id := range *update.ExcludedChannelIDs {
			if id <= 0 {
				return nil, errors.New("白名单渠道 ID 必须为正整数")
			}
			if !seenIDs[id] {
				excluded = append(excluded, id)
				seenIDs[id] = true
			}
		}
		if len(excluded) > 0 {
			var channels []UpstreamModelGuardChannel
			if err := model.DB.WithContext(ctx).Model(&model.Channel{}).Select("id").Where("id IN ?", excluded).Find(&channels).Error; err != nil {
				return nil, err
			}
			known := make(map[int]bool)
			for _, channel := range channels {
				known[channel.ID] = true
			}
			// 保留已选但被删除的历史 ID，禁止添加未知 ID。
			for _, id := range previous.ExcludedChannelIDs {
				known[id] = true
			}
			for _, id := range excluded {
				if !known[id] {
					return nil, fmt.Errorf("白名单渠道 #%d 不存在", id)
				}
			}
		}
	}
	excludedJSON, err := common.Marshal(excluded)
	if err != nil {
		return nil, err
	}
	var groups []model.Group
	if err := model.DB.WithContext(ctx).Select("id", "code", "name", "status").Find(&groups).Error; err != nil {
		return nil, err
	}
	groupMap := make(map[string]model.Group, len(groups))
	for _, group := range groups {
		groupMap[group.Code] = group
	}
	storedRules := make([]model.UpstreamModelGuardStoredRule, 0, len(update.Rules))
	seen := make(map[string]bool)
	for index, rule := range update.Rules {
		modelName, err := normalizeUpstreamModelGuardName(rule.Model, 255)
		if err != nil {
			return nil, fmt.Errorf("规则 %d 请求模型：%w", index+1, err)
		}
		if len(rule.GroupCodes) == 0 || len(rule.GroupCodes) > 100 || len(rule.UpstreamModels) == 0 || len(rule.UpstreamModels) > 50 {
			return nil, fmt.Errorf("规则 %d 需要 1 至 100 个分组及 1 至 50 个上游模型", index+1)
		}
		stored := model.UpstreamModelGuardStoredRule{Enabled: rule.Enabled, Model: modelName, GroupIDs: []int{}, GroupCodes: []string{}, UpstreamModels: []string{}}
		seenGroups := make(map[int]bool)
		for _, raw := range rule.GroupCodes {
			code := strings.TrimSpace(raw)
			group, ok := groupMap[code]
			groupName := strings.TrimSpace(group.Name)
			if groupName == "" {
				groupName = code
			}
			if !ok || group.Status != model.GroupStatusActive {
				return nil, fmt.Errorf("规则 %d 分组 %q 不存在或已停用", index+1, groupName)
			}
			if seenGroups[group.Id] {
				continue
			}
			seenGroups[group.Id] = true
			key := fmt.Sprintf("%d\x00%s", group.Id, modelName)
			if rule.Enabled && seen[key] {
				return nil, fmt.Errorf("分组 %q 的模型 %q 存在重复启用规则", groupName, modelName)
			}
			if rule.Enabled {
				seen[key] = true
			}
			stored.GroupIDs = append(stored.GroupIDs, group.Id)
			stored.GroupCodes = append(stored.GroupCodes, code)
		}
		seenModels := make(map[string]bool)
		for _, raw := range rule.UpstreamModels {
			upstream, err := normalizeUpstreamModelGuardName(raw, 255)
			if err != nil {
				return nil, fmt.Errorf("规则 %d 上游模型：%w", index+1, err)
			}
			if !seenModels[upstream] {
				seenModels[upstream] = true
				stored.UpstreamModels = append(stored.UpstreamModels, upstream)
			}
		}
		storedRules = append(storedRules, stored)
	}
	data, err := common.Marshal(storedRules)
	if err != nil {
		return nil, err
	}
	stored := &model.UpstreamModelGuardConfig{Enabled: update.Enabled, RulesJSON: string(data), UpdatedBy: userID, FailureThreshold: threshold, ExcludedChannelIDsJSON: string(excludedJSON)}
	if err := model.SaveUpstreamModelGuardConfig(ctx, update.ExpectedVersion, stored); err != nil {
		return nil, err
	}
	return GetUpstreamModelGuardConfig(ctx)
}

func normalizeUpstreamModelGuardName(raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > limit || !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("名称不能为空、包含控制字符或超过 %d 字节", limit)
	}
	return value, nil
}

// BindUpstreamModelGuard 在响应解析时同步持久化，不依赖消费日志或后台扫描。
func BindUpstreamModelGuard(info *relaycommon.RelayInfo) func() {
	finish := func() {}
	if info == nil || !IsUpstreamModelGuardModuleEnabled() {
		return finish
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	config, err := GetUpstreamModelGuardConfig(ctx)
	cancel()
	if err != nil {
		common.SysError(fmt.Sprintf("upstream model guard config unavailable: %v", err))
		return finish
	}
	if !config.Enabled {
		return finish
	}
	requestedModel := strings.TrimSpace(info.OriginModelName)
	var mu sync.Mutex
	requestID := strings.TrimSpace(info.RequestId)
	if requestID == "" {
		requestID = common.GetUUID()
	}
	excluded := make(map[int]bool, len(config.ExcludedChannelIDs))
	for _, id := range config.ExcludedChannelIDs {
		excluded[id] = true
	}
	checked := make(map[int]bool)
	observed := make(map[int]*model.UpstreamModelGuardRecord)
	mismatched := make(map[int]bool)
	info.OnUpstreamModelEvidence = func(current *relaycommon.RelayInfo, evidence relaycommon.UpstreamModelEvidence) {
		if current == nil || current.ChannelMeta == nil || !IsUpstreamModelGuardModuleEnabled() {
			return
		}
		actual := strings.TrimSpace(evidence.Model)
		if actual == "" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		if excluded[current.ChannelId] || checked[current.ChannelId] {
			return
		}
		for _, rule := range config.Rules {
			if !rule.Enabled || requestedModel != rule.Model {
				continue
			}
			groupID := 0
			for index, group := range rule.GroupCodes {
				if group == strings.TrimSpace(current.UsingGroup) && index < len(rule.groupIDs) {
					groupID = rule.groupIDs[index]
					break
				}
			}
			if groupID == 0 {
				continue
			}
			matched := false
			for _, expected := range rule.UpstreamModels {
				if actual == expected {
					matched = true
					break
				}
			}
			if matched && mismatched[current.ChannelId] {
				return
			}
			// 允许的缓冲模型不等于正文完整匹配，不能仅凭头部清零连续异常。
			if matched && evidence.Source == constant.UpstreamModelSourceCodexFasterModel {
				return
			}
			observationKey := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%d", requestID, current.ChannelId))))
			record := &model.UpstreamModelGuardRecord{
				ChannelID: current.ChannelId, GroupID: groupID, Group: strings.TrimSpace(current.UsingGroup),
				RequestedModel: rule.Model, ExpectedUpstreamModels: append([]string(nil), rule.UpstreamModels...),
				ActualUpstreamModel: actual, RequestID: current.RequestId, ConfigVersion: config.ConfigVersion,
				DetectionSource: evidence.Source,
				ObservationKey:  &observationKey,
			}
			if len(record.RequestID) > 128 {
				record.RequestID = strings.ToValidUTF8(record.RequestID[:128], "")
			}
			observed[current.ChannelId] = record
			if matched {
				return
			}
			mismatched[current.ChannelId] = true
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := model.ObserveUpstreamModelGuard(ctx, record, false)
			cancel()
			if err != nil {
				common.SysError(fmt.Sprintf("upstream model guard disable failed: channel_id=%d request_id=%s error=%v", current.ChannelId, record.RequestID, err))
				return
			}
			checked[current.ChannelId] = true
			return
		}
	}
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if !IsUpstreamModelGuardModuleEnabled() {
			return
		}
		for channelID, record := range observed {
			if checked[channelID] {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := model.ObserveUpstreamModelGuard(ctx, record, !mismatched[channelID])
			cancel()
			if err != nil {
				common.SysError(fmt.Sprintf("upstream model guard observation failed: channel_id=%d error=%v", channelID, err))
				continue
			}
			checked[channelID] = true
		}
	}
}
