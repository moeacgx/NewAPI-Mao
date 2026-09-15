package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCustomTaskPluginUploadIsIdempotentAndDeletionUsesVersionPin(t *testing.T) {
	previousDB := DB
	previousType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		DB = previousDB
		common.SetMainDatabaseType(previousType)
	})
	require.NoError(t, DB.AutoMigrate(&TaskPlugin{}, &Task{}))

	first := &TaskPlugin{Key: "custom", Version: "1.0.0", APIVersion: 1, Source: "source-1", SourceHash: "hash-1", SourceKind: "custom"}
	require.NoError(t, SaveTaskPlugin(first))
	assert.False(t, first.Active)
	assert.False(t, first.Enabled)
	require.NoError(t, ActivateTaskPlugin(first.Key, first.Version))
	duplicate := &TaskPlugin{Key: first.Key, Version: first.Version, APIVersion: 1, Source: first.Source, SourceHash: first.SourceHash, SourceKind: "custom"}
	require.NoError(t, SaveTaskPlugin(duplicate))
	assert.True(t, duplicate.Active)
	assert.True(t, duplicate.Enabled)

	second := &TaskPlugin{Key: "custom", Version: "2.0.0", APIVersion: 1, Source: "source-2", SourceHash: "hash-2", SourceKind: "custom"}
	require.NoError(t, SaveTaskPlugin(second))
	require.NoError(t, DB.Create(&Task{Platform: "custom", TaskID: "historical", Status: TaskStatusSuccess, PrivateData: TaskPrivateData{Execution: &TaskExecutionSnapshot{TaskPlugin: &TaskPluginSnapshot{Key: first.Key, Version: first.Version, SourceHash: first.SourceHash, SourceKind: "custom"}}}}).Error)
	_, err = DeleteTaskPluginVersion(second.Key, second.Version)
	require.NoError(t, err)
	_, err = GetTaskPluginVersion(second.Key, second.Version)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = DeleteTaskPluginVersion(first.Key, first.Version)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "历史任务")
}
