package model

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUpstreamModelGuardModelTest(t *testing.T) (*Channel, *UpstreamModelGuardConfig) {
	t.Helper()
	originalDB, originalDBType := DB, common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "guard.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	sqlDB, err := DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		DB = originalDB
		common.SetMainDatabaseType(originalDBType)
	})
	require.NoError(t, DB.AutoMigrate(&NotificationBot{}, &NotificationTask{}, &NotificationTarget{}, &NotificationEventReceipt{}, &NotificationEvent{}, &NotificationDelivery{}))
	require.NoError(t, DB.AutoMigrate(&Group{}, &Channel{}, &Ability{}, &UpstreamModelGuardConfig{}, &UpstreamModelGuardRecord{}, &UpstreamModelGuardStreak{}))
	originalMemory := common.MemoryCacheEnabled
	originalGroups, originalChannels, originalAdvanced := group2model2channels, channelsIDM, channel2advancedCustomConfig
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		common.MemoryCacheEnabled = originalMemory
		channelSyncLock.Lock()
		group2model2channels, channelsIDM, channel2advancedCustomConfig = originalGroups, originalChannels, originalAdvanced
		channelSyncLock.Unlock()
	})
	channel := &Channel{
		Name: "渠道一", Key: "first-key\nsecond-key", Models: "client-model,other-model", Group: "default,other",
		Status: common.ChannelStatusEnabled, UsedQuota: 321, OtherInfo: `{"retained":"value"}`,
		ChannelInfo: ChannelInfo{IsMultiKey: true, MultiKeySize: 2, MultiKeyMode: constant.MultiKeyModePolling, MultiKeyPollingIndex: 1},
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))
	config, err := LoadUpstreamModelGuardConfig(context.Background())
	require.NoError(t, err)
	config.Enabled = true
	config.FailureThreshold = 1
	require.NoError(t, SaveUpstreamModelGuardConfig(context.Background(), config.ConfigVersion, config))
	bot := &NotificationBot{Name: "existing-bot", Token: "no-network", Enabled: true}
	require.NoError(t, DB.Create(bot).Error)
	task := &NotificationTask{Name: "guard", EventType: UpstreamModelGuardNotificationEvent, BotId: bot.Id, Template: "{{mention}}", Enabled: true}
	require.NoError(t, CreateNotificationTask(task))
	require.NoError(t, CreateNotificationTarget(&NotificationTarget{TaskId: task.Id, ChatId: "-10001", Enabled: true}))
	return channel, config
}

func upstreamModelGuardTestRecord(channel *Channel, config *UpstreamModelGuardConfig) *UpstreamModelGuardRecord {
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(common.GetUUID())))
	return &UpstreamModelGuardRecord{
		ChannelID: channel.Id, GroupID: 1, Group: "default", RequestedModel: "client-model",
		ExpectedUpstreamModels: []string{"provider-model", "provider-model-v2"}, ActualUpstreamModel: "wrong-model",
		RequestID: "guard-request", ConfigVersion: config.ConfigVersion, ObservationKey: &key,
	}
}

func TestUpstreamModelGuardDisablesWholeChannelAndEnqueuesOnce(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	originalInfo := channel.ChannelInfo
	var wg sync.WaitGroup
	results := make(chan bool, 2)
	errors := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			changed, err := ObserveUpstreamModelGuard(context.Background(), upstreamModelGuardTestRecord(channel, config), false)
			results <- changed
			errors <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	changedCount := 0
	for changed := range results {
		if changed {
			changedCount++
		}
	}
	for err := range errors {
		require.NoError(t, err)
	}
	assert.Equal(t, 1, changedCount)
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
	assert.Equal(t, channel.Key, stored.Key)
	assert.Equal(t, channel.Models, stored.Models)
	assert.Equal(t, channel.Group, stored.Group)
	assert.Equal(t, channel.UsedQuota, stored.UsedQuota)
	assert.Equal(t, originalInfo, stored.ChannelInfo)
	assert.Equal(t, "value", stored.GetOtherInfo()["retained"])
	var enabled int64
	require.NoError(t, DB.Model(&Ability{}).Where("channel_id = ? AND enabled = ?", channel.Id, true).Count(&enabled).Error)
	assert.Zero(t, enabled)
	records, total, err := ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, records, 1)
	assert.Equal(t, []string{"provider-model", "provider-model-v2"}, records[0].ExpectedUpstreamModels)
	var events []NotificationEvent
	require.NoError(t, DB.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, UpstreamModelGuardNotificationEvent, events[0].EventType)
	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(events[0].Payload, &payload))
	assert.Equal(t, float64(channel.Id), payload["channel_id"])
	assert.Equal(t, channel.Name, payload["channel_name"])
	assert.Equal(t, "wrong-model", payload["actual_upstream_model"])
	var deliveries int64
	require.NoError(t, DB.Model(&NotificationDelivery{}).Count(&deliveries).Error)
	assert.EqualValues(t, 1, deliveries)
}

func TestUpstreamModelGuardUsesGroupNameWithoutChangingIdentity(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	group := Group{Code: "Codex-Pro", Name: "专业分组 <高级>", Status: GroupStatusActive}
	require.NoError(t, DB.Create(&group).Error)
	record := upstreamModelGuardTestRecord(channel, config)
	record.GroupID, record.Group = group.Id, group.Code
	changed, err := ObserveUpstreamModelGuard(context.Background(), record, false)
	require.NoError(t, err)
	require.True(t, changed)
	var event NotificationEvent
	require.NoError(t, DB.First(&event).Error)
	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
	assert.Equal(t, group.Name, payload["group"])
	assert.Contains(t, payload["comparison"], "分组: 专业分组 <高级>")
	assert.NotContains(t, payload["comparison"], group.Code)

	for _, state := range []struct{ name, expected string }{
		{name: "专业分组 <高级>", expected: "专业分组 <高级>"},
		{name: "新的显示名称", expected: "新的显示名称"},
		{name: "", expected: group.Code},
	} {
		require.NoError(t, DB.Model(&Group{}).Where("id = ?", group.Id).Update("name", state.name).Error)
		records, _, err := ListUpstreamModelGuardRecords(context.Background(), 1, 50)
		require.NoError(t, err)
		require.Len(t, records, 1)
		data, err := common.Marshal(records[0])
		require.NoError(t, err)
		var item map[string]any
		require.NoError(t, common.Unmarshal(data, &item))
		assert.Equal(t, state.expected, item["group_name"])
		assert.Equal(t, group.Code, item["group"])
		assert.EqualValues(t, group.Id, item["group_id"])
	}
	require.NoError(t, DB.Delete(&group).Error)
	require.NoError(t, DB.Create(&Group{Id: group.Id + 100, Code: group.Code, Name: "重新创建的其他分组"}).Error)
	records, _, err := ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	data, err := common.Marshal(records[0])
	require.NoError(t, err)
	var item map[string]any
	require.NoError(t, common.Unmarshal(data, &item))
	assert.Equal(t, group.Code, item["group_name"], "不能将新建分组的名称套用到旧记录")
	var unchanged NotificationEvent
	require.NoError(t, DB.First(&unchanged, event.Id).Error)
	assert.Equal(t, event.Payload, unchanged.Payload, "改名不重写历史通知负载")
}

func TestUpstreamModelGuardRollsBackWhenNotificationFails(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	common.MemoryCacheEnabled = true
	InitChannelCache()
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("guard_notification_failure", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "NotificationDelivery" {
			tx.AddError(errors.New("delivery storage failure"))
		}
	}))
	changed, err := ObserveUpstreamModelGuard(context.Background(), upstreamModelGuardTestRecord(channel, config), false)
	require.ErrorContains(t, err, "delivery storage failure")
	assert.False(t, changed)
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusEnabled, stored.Status)
	assert.Equal(t, channel.OtherInfo, stored.OtherInfo)
	cached, err := CacheGetChannel(channel.Id)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, cached.Status)
	for _, table := range []any{&UpstreamModelGuardRecord{}, &NotificationEvent{}, &NotificationDelivery{}} {
		var count int64
		require.NoError(t, DB.Model(table).Count(&count).Error)
		assert.Zero(t, count)
	}
}

func TestUpstreamModelGuardRejectsStaleConfigurationAndAllowsExplicitRecovery(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	stale := upstreamModelGuardTestRecord(channel, config)
	require.NoError(t, SaveUpstreamModelGuardConfig(context.Background(), config.ConfigVersion, config))
	changed, err := ObserveUpstreamModelGuard(context.Background(), stale, false)
	require.NoError(t, err)
	assert.False(t, changed)
	changed, err = ObserveUpstreamModelGuard(context.Background(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	assert.True(t, changed)
	available, err := IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.False(t, available)
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("status", common.ChannelStatusEnabled).Error)
	available, err = IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.True(t, available)
	changed, err = ObserveUpstreamModelGuard(context.Background(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	assert.True(t, changed)
	_, total, err := ListUpstreamModelGuardRecords(context.Background(), 1, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
}

func TestUpstreamModelGuardNotificationSummaryPreservesEveryDimension(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	record := upstreamModelGuardTestRecord(channel, config)
	record.ExpectedUpstreamModels = []string{strings.Repeat("期", 1024)}
	record.ActualUpstreamModel = "实际模型-" + strings.Repeat("实", 5000)
	changed, err := ObserveUpstreamModelGuard(context.Background(), record, false)
	require.NoError(t, err)
	assert.True(t, changed)
	var event NotificationEvent
	require.NoError(t, DB.First(&event).Error)
	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
	comparison, ok := payload["comparison"].(string)
	require.True(t, ok)
	assert.LessOrEqual(t, utf8.RuneCountInString(comparison), 1024)
	assert.Contains(t, comparison, "分组: default")
	assert.Contains(t, comparison, "请求模型: client-model")
	assert.Contains(t, comparison, "允许上游模型: 期期")
	assert.Contains(t, comparison, "实际上游模型: 实际模型-")
	assert.Equal(t, "upstream-model-guard", payload["module_id"])
	assert.Equal(t, event.EventType, payload["event_type"])
	assert.Equal(t, event.EventKey, payload["event_key"])
	records, _, err := ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, record.ActualUpstreamModel, records[0].ActualUpstreamModel)
}

func TestUpstreamModelGuardFinalCheckRejectsStaleEnabledCache(t *testing.T) {
	channel, _ := setupUpstreamModelGuardModelTest(t)
	common.MemoryCacheEnabled = true
	InitChannelCache()
	require.NoError(t, DB.Model(&Channel{}).Where("id = ?", channel.Id).Update("status", common.ChannelStatusManuallyDisabled).Error)
	cached, err := CacheGetChannel(channel.Id)
	require.NoError(t, err)
	assert.Equal(t, common.ChannelStatusEnabled, cached.Status)
	available, err := IsChannelEnabledAfterUpstreamModelGuard(context.Background(), channel.Id)
	require.NoError(t, err)
	assert.False(t, available)
}
