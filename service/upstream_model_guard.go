package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
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
	ConfigVersion int64                    `json:"config_version"`
	Enabled       bool                     `json:"enabled"`
	Rules         []UpstreamModelGuardRule `json:"rules"`
	UpdatedAt     int64                    `json:"updated_at"`
	UpdatedBy     int                      `json:"updated_by"`
}

type UpstreamModelGuardConfigUpdate struct {
	ExpectedVersion int64                    `json:"expected_version"`
	Enabled         bool                     `json:"enabled"`
	Rules           []UpstreamModelGuardRule `json:"rules"`
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
	stored := &model.UpstreamModelGuardConfig{Enabled: update.Enabled, RulesJSON: string(data), UpdatedBy: userID}
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
func BindUpstreamModelGuard(info *relaycommon.RelayInfo) {
	if info == nil || !IsUpstreamModelGuardModuleEnabled() {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	config, err := GetUpstreamModelGuardConfig(ctx)
	cancel()
	if err != nil {
		common.SysError(fmt.Sprintf("upstream model guard config unavailable: %v", err))
		return
	}
	if !config.Enabled {
		return
	}
	requestedModel := strings.TrimSpace(info.OriginModelName)
	var mu sync.Mutex
	checked := make(map[string]bool)
	info.OnUpstreamResponseModel = func(current *relaycommon.RelayInfo, actual string) {
		if current == nil || current.ChannelMeta == nil || !IsUpstreamModelGuardModuleEnabled() {
			return
		}
		actual = strings.TrimSpace(actual)
		if actual == "" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
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
			for _, expected := range rule.UpstreamModels {
				if actual == expected {
					return
				}
			}
			key := fmt.Sprintf("%d:%d:%s", current.ChannelId, groupID, rule.Model)
			if checked[key] {
				return
			}
			record := &model.UpstreamModelGuardRecord{
				ChannelID: current.ChannelId, GroupID: groupID, Group: strings.TrimSpace(current.UsingGroup),
				RequestedModel: rule.Model, ExpectedUpstreamModels: append([]string(nil), rule.UpstreamModels...),
				ActualUpstreamModel: actual, RequestID: current.RequestId, ConfigVersion: config.ConfigVersion,
			}
			if len(record.RequestID) > 128 {
				record.RequestID = strings.ToValidUTF8(record.RequestID[:128], "")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := model.DisableChannelForUpstreamModelGuard(ctx, record)
			cancel()
			if err != nil {
				common.SysError(fmt.Sprintf("upstream model guard disable failed: channel_id=%d request_id=%s error=%v", current.ChannelId, record.RequestID, err))
				return
			}
			checked[key] = true
			return
		}
	}
}
