package model

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpstreamModelGuardToleranceDefaultsToTwoAndRecordsBothRequests(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 0
	require.NoError(t, SaveUpstreamModelGuardConfig(context.Background(), config.ConfigVersion, config))
	loaded, err := LoadUpstreamModelGuardConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, loaded.FailureThreshold)
	assert.Equal(t, "[]", loaded.ExcludedChannelIDsJSON)
	for index := 1; index <= 2; index++ {
		record := upstreamModelGuardTestRecord(channel, config)
		record.RequestID = fmt.Sprintf("request-%d", index)
		changed, err := ObserveUpstreamModelGuard(context.Background(), record, false)
		require.NoError(t, err)
		assert.Equal(t, index == 2, changed)
		assert.Equal(t, index, record.ConsecutiveMismatches)
		assert.Equal(t, 2, record.FailureThreshold)
		assert.Equal(t, index == 2, record.ChannelDisabled)
		var deliveryCount int64
		require.NoError(t, DB.Model(&NotificationDelivery{}).Count(&deliveryCount).Error)
		assert.EqualValues(t, index-1, deliveryCount)
	}
	records, total, err := ListUpstreamModelGuardRecords(context.Background(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, records, 2)
	assert.True(t, records[0].ChannelDisabled)
	assert.False(t, records[1].ChannelDisabled)
	var streakCount int64
	require.NoError(t, DB.Model(&UpstreamModelGuardStreak{}).Count(&streakCount).Error)
	assert.Zero(t, streakCount)
	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	assert.Equal(t, common.ChannelStatusManuallyDisabled, stored.Status)
}

func TestUpstreamModelGuardToleranceResetsAcrossModelsAndSharesChannelCount(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 2
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	first := upstreamModelGuardTestRecord(channel, config)
	changed, err := ObserveUpstreamModelGuard(t.Context(), first, false)
	require.NoError(t, err)
	assert.False(t, changed)
	// 同一请求的后续匹配帧不能撤销已计入的不匹配。
	changed, err = ObserveUpstreamModelGuard(t.Context(), first, true)
	require.NoError(t, err)
	assert.False(t, changed)
	var streak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	matched := upstreamModelGuardTestRecord(channel, config)
	matched.GroupID, matched.Group, matched.RequestedModel = 2, "other", "other-model"
	changed, err = ObserveUpstreamModelGuard(t.Context(), matched, true)
	require.NoError(t, err)
	assert.False(t, changed)
	var count int64
	require.NoError(t, DB.Model(&UpstreamModelGuardStreak{}).Count(&count).Error)
	assert.Zero(t, count)
	_, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total, "匹配请求不保存异常记录")
	for index := 1; index <= 2; index++ {
		record := upstreamModelGuardTestRecord(channel, config)
		if index == 2 {
			record.GroupID, record.Group, record.RequestedModel = 2, "other", "other-model"
		}
		changed, err = ObserveUpstreamModelGuard(t.Context(), record, false)
		require.NoError(t, err)
		assert.Equal(t, index == 2, changed)
		assert.Equal(t, index, record.ConsecutiveMismatches)
	}
	require.True(t, UpdateChannelStatus(channel.Id, "", common.ChannelStatusEnabled, ""))
	record := upstreamModelGuardTestRecord(channel, config)
	changed, err = ObserveUpstreamModelGuard(t.Context(), record, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, 1, record.ConsecutiveMismatches, "人工恢复后的新请求从零累计")
}

func TestUpstreamModelGuardToleranceDeduplicatesConcurrentRequests(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 3
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	first := upstreamModelGuardTestRecord(channel, config)
	duplicate := *first
	duplicate.RequestedModel, duplicate.GroupID = "different-model", 2
	second := upstreamModelGuardTestRecord(channel, config)
	start := make(chan struct{})
	results := make(chan error, 3)
	var wg sync.WaitGroup
	for _, record := range []*UpstreamModelGuardRecord{first, &duplicate, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := ObserveUpstreamModelGuard(context.Background(), record, false)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	var streak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 2, streak.ConsecutiveMismatches)
	records, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	assert.Equal(t, 2, records[0].ConsecutiveMismatches)
	assert.Equal(t, 1, records[1].ConsecutiveMismatches)
	third := upstreamModelGuardTestRecord(channel, config)
	changed, err := ObserveUpstreamModelGuard(t.Context(), third, false)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, 3, third.ConsecutiveMismatches)
	var deliveries int64
	require.NoError(t, DB.Model(&NotificationDelivery{}).Count(&deliveries).Error)
	assert.EqualValues(t, 1, deliveries)
}

func TestUpstreamModelGuardToleranceWhitelistAndVersionInvalidateOldCount(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 2
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	_, err := ObserveUpstreamModelGuard(t.Context(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	stale := upstreamModelGuardTestRecord(channel, config)
	config.ExcludedChannelIDsJSON = fmt.Sprintf("[%d]", channel.Id)
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	for _, matched := range []bool{false, true} {
		changed, err := ObserveUpstreamModelGuard(t.Context(), upstreamModelGuardTestRecord(channel, config), matched)
		require.NoError(t, err)
		assert.False(t, changed)
	}
	var streak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches, "白名单请求不增加也不清除已有计数")
	config.ExcludedChannelIDsJSON = "[]"
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	changed, err := ObserveUpstreamModelGuard(t.Context(), stale, false)
	require.NoError(t, err)
	assert.False(t, changed)
	fresh := upstreamModelGuardTestRecord(channel, config)
	changed, err = ObserveUpstreamModelGuard(t.Context(), fresh, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, 1, fresh.ConsecutiveMismatches, "配置版本变化后旧计数失效")
	_, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
}

func TestUpstreamModelGuardToleranceNotificationFailureRollsBackThresholdObservation(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 2
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	_, err := ObserveUpstreamModelGuard(t.Context(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("guard-threshold-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "notification_deliveries" {
			tx.AddError(errors.New("notification unavailable"))
		}
	}))
	second := upstreamModelGuardTestRecord(channel, config)
	changed, err := ObserveUpstreamModelGuard(t.Context(), second, false)
	require.ErrorContains(t, err, "notification unavailable")
	assert.False(t, changed)
	var streak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	_, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	available, err := IsChannelEnabledAfterUpstreamModelGuard(t.Context(), channel.Id)
	require.NoError(t, err)
	assert.True(t, available)
	require.NoError(t, DB.Callback().Create().Remove("guard-threshold-failure"))
	changed, err = ObserveUpstreamModelGuard(t.Context(), second, false)
	require.NoError(t, err)
	assert.True(t, changed, "事务回滚后同一请求仍可重试")
}

func TestUpstreamModelGuardToleranceHistoricalRecordsRemainDisabled(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	for range 2 {
		record := upstreamModelGuardTestRecord(channel, config)
		record.ObservationKey = nil
		record.ExpectedUpstreamModelsJSON = `["old-provider"]`
		require.NoError(t, DB.Create(record).Error)
	}
	require.NoError(t, DB.AutoMigrate(&UpstreamModelGuardRecord{}))
	records, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	for _, record := range records {
		assert.Equal(t, 1, record.ConsecutiveMismatches)
		assert.Equal(t, 1, record.FailureThreshold)
		assert.True(t, record.ChannelDisabled)
		assert.Nil(t, record.ObservationKey)
	}
}

func TestUpstreamModelGuardToleranceMaximumThresholdAndChannelIsolation(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 100
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	require.NoError(t, DB.Create(&UpstreamModelGuardStreak{ChannelID: channel.Id, ConsecutiveMismatches: 99, ConfigVersion: config.ConfigVersion}).Error)
	other := &Channel{Name: "独立渠道", Key: "other-key", Status: common.ChannelStatusEnabled, Models: "client-model", Group: "default"}
	require.NoError(t, DB.Create(other).Error)
	require.NoError(t, other.AddAbilities(nil))
	otherRecord := upstreamModelGuardTestRecord(other, config)
	changed, err := ObserveUpstreamModelGuard(t.Context(), otherRecord, false)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, 1, otherRecord.ConsecutiveMismatches)
	last := upstreamModelGuardTestRecord(channel, config)
	changed, err = ObserveUpstreamModelGuard(t.Context(), last, false)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, 100, last.ConsecutiveMismatches)
	assert.Equal(t, 100, last.FailureThreshold)
	var otherStreak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&otherStreak, "channel_id = ?", other.Id).Error)
	assert.Equal(t, 1, otherStreak.ConsecutiveMismatches)
	var event NotificationEvent
	require.NoError(t, DB.First(&event).Error)
	var payload map[string]any
	require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
	assert.Contains(t, payload["comparison"], "连续不匹配: 100/100")
	assert.Equal(t, float64(100), payload["consecutive_mismatches"])
	assert.Equal(t, float64(100), payload["failure_threshold"])
}

func TestUpstreamModelGuardMatchedWithoutStreakDoesNotWaitForWriter(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	for _, state := range []struct {
		name    string
		count   int
		version int64
	}{
		{name: "没有计数"},
		{name: "当前计数为零", count: 0, version: config.ConfigVersion},
		{name: "旧版本非零计数", count: 1, version: config.ConfigVersion - 1},
	} {
		t.Run(state.name, func(t *testing.T) {
			require.NoError(t, DB.Where("channel_id = ?", channel.Id).Delete(&UpstreamModelGuardStreak{}).Error)
			if state.version > 0 {
				require.NoError(t, DB.Create(&UpstreamModelGuardStreak{ChannelID: channel.Id, ConsecutiveMismatches: state.count, ConfigVersion: state.version}).Error)
			}
			writer := DB.Begin()
			require.NoError(t, writer.Error)
			defer writer.Rollback()
			// 已建立的立即事务持有 SQLite 写锁，WAL 读连接仍应能完成匹配快路径。
			require.NoError(t, writer.Model(&UpstreamModelGuardConfig{}).Where("id = ?", 1).Update("updated_by", 99).Error)
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			changed, err := ObserveUpstreamModelGuard(ctx, upstreamModelGuardTestRecord(channel, config), true)
			require.NoError(t, err, "无当前非零计数的匹配请求不应进入写事务")
			assert.False(t, changed)
		})
	}
}

func TestUpstreamModelGuardMatchedOldConfigDoesNotClearNewCount(t *testing.T) {
	channel, config := setupUpstreamModelGuardModelTest(t)
	config.FailureThreshold = 2
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	oldMatched := upstreamModelGuardTestRecord(channel, config)
	require.NoError(t, SaveUpstreamModelGuardConfig(t.Context(), config.ConfigVersion, config))
	fresh := upstreamModelGuardTestRecord(channel, config)
	_, err := ObserveUpstreamModelGuard(t.Context(), fresh, false)
	require.NoError(t, err)
	_, err = ObserveUpstreamModelGuard(t.Context(), oldMatched, true)
	require.NoError(t, err)
	var streak UpstreamModelGuardStreak
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	assert.Equal(t, config.ConfigVersion, streak.ConfigVersion)
	// 当前版本同一 HTTP 的匹配仍不得撤销已经落库的不匹配。
	_, err = ObserveUpstreamModelGuard(t.Context(), fresh, true)
	require.NoError(t, err)
	require.NoError(t, DB.First(&streak, "channel_id = ?", channel.Id).Error)
	assert.Equal(t, 1, streak.ConsecutiveMismatches)
	changed, err := ObserveUpstreamModelGuard(t.Context(), upstreamModelGuardTestRecord(channel, config), false)
	require.NoError(t, err)
	assert.True(t, changed)
}
