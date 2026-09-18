package model

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpstreamModelGuardMigratesLegacySQLiteRows(t *testing.T) {
	previousDB, previousDBType := DB, common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy-guard.db")), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousDBType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	// 直接构造 0.1.0 的真实字段，避免用新版模型先建表掩盖 ADD UNIQUE 失败。
	require.NoError(t, db.Exec("CREATE TABLE `upstream_model_guard_records` (`id` integer PRIMARY KEY AUTOINCREMENT, `channel_id` integer NOT NULL, `channel_name` text NOT NULL, `group_id` integer NOT NULL, `group` text NOT NULL, `requested_model` text NOT NULL, `expected_upstream_models_json` text NOT NULL, `actual_upstream_model` text NOT NULL, `request_id` text, `reason` text NOT NULL, `config_version` bigint NOT NULL, `created_at` bigint NOT NULL)").Error)
	require.NoError(t, db.Exec("CREATE TABLE `upstream_model_guard_configs` (`id` integer PRIMARY KEY, `config_version` bigint NOT NULL, `enabled` numeric NOT NULL, `rules_json` text NOT NULL, `updated_at` bigint NOT NULL, `updated_by` integer)").Error)
	require.NoError(t, db.Exec("INSERT INTO upstream_model_guard_configs (id, config_version, enabled, rules_json, updated_at, updated_by) VALUES (?, ?, ?, ?, ?, ?)", 1, 7, true, "[]", 123, 42).Error)
	for _, id := range []int{1, 2} {
		require.NoError(t, db.Exec("INSERT INTO upstream_model_guard_records (id, channel_id, channel_name, group_id, `group`, requested_model, expected_upstream_models_json, actual_upstream_model, request_id, reason, config_version, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", id, 10, "旧渠道", 3, "legacy", "client-model", `["expected"]`, "actual", "old-request", "旧禁用原因", 7, 123).Error)
	}
	require.NoError(t, migrateSQLiteUpstreamModelGuardObservationKey())
	require.NoError(t, db.AutoMigrate(&UpstreamModelGuardConfig{}, &UpstreamModelGuardRecord{}, &UpstreamModelGuardStreak{}, &Group{}))
	require.NoError(t, migrateSQLiteUpstreamModelGuardObservationKey(), "重启时重复迁移应安全跳过")
	config, err := LoadUpstreamModelGuardConfig(t.Context())
	require.NoError(t, err)
	assert.EqualValues(t, 7, config.ConfigVersion)
	assert.True(t, config.Enabled)
	assert.Equal(t, 2, config.FailureThreshold)
	assert.Equal(t, "[]", config.ExcludedChannelIDsJSON)
	records, total, err := ListUpstreamModelGuardRecords(t.Context(), 1, 50)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	for _, record := range records {
		assert.Equal(t, "旧渠道", record.ChannelName)
		assert.Equal(t, "旧禁用原因", record.Reason)
		assert.Equal(t, 1, record.ConsecutiveMismatches)
		assert.Equal(t, 1, record.FailureThreshold)
		assert.True(t, record.ChannelDisabled)
		assert.Nil(t, record.ObservationKey)
	}
	key := strings.Repeat("a", 64)
	newRecord := records[0]
	newRecord.Id = 0
	newRecord.ObservationKey = &key
	require.NoError(t, db.Create(&newRecord).Error)
	newRecord.Id = 0
	require.Error(t, db.Create(&newRecord).Error, "迁移后必须拒绝重复 HTTP 观察键")
}
