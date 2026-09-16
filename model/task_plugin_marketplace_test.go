package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTaskPluginMarketplaceSourcesDefaultsClearAndValidation(t *testing.T) {
	old := DB
	oldType := common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.OptionMapRWMutex.Lock()
	oldOptions := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = oldOptions
		common.OptionMapRWMutex.Unlock()
	})
	t.Cleanup(func() { DB = old; common.SetMainDatabaseType(oldType) })
	require.NoError(t, db.AutoMigrate(&Option{}))
	got, err := GetTaskPluginMarketplaceSources()
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.NoError(t, SaveTaskPluginMarketplaceSources([]setting.TaskPluginMarketplaceSource{}))
	got, err = GetTaskPluginMarketplaceSources()
	require.NoError(t, err)
	require.Empty(t, got)
	assert.NotNil(t, got)
	// 模拟另一个实例直接更新数据库，本地 OptionMap 仍保留旧的 []。
	remoteSources := []setting.TaskPluginMarketplaceSource{{Name: "Other node", IndexURL: "https://other.example/index.json"}}
	payload, err := common.Marshal(remoteSources)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Option{}).Where(&Option{Key: setting.TaskPluginMarketplaceSourcesKey}).Update("value", string(payload)).Error)
	got, err = GetTaskPluginMarketplaceSources()
	require.NoError(t, err)
	assert.Equal(t, remoteSources, got)
	require.Error(t, SaveTaskPluginMarketplaceSources([]setting.TaskPluginMarketplaceSource{{Name: "x", IndexURL: "http://bad"}}))
	for _, rawURL := range []string{"https://user:secret@plugins.example/index.json", "https://plugins.example/index.json?token=secret", "https://plugins.example/index.json#fragment", "https://plugins.example/index.json?", "https://plugins.example/index.json#", "https://plugins.example\\evil/index.json"} {
		assert.Error(t, SaveTaskPluginMarketplaceSources([]setting.TaskPluginMarketplaceSource{{Name: "bad", IndexURL: rawURL}}))
	}
	got, err = GetTaskPluginMarketplaceSources()
	require.NoError(t, err)
	assert.Equal(t, remoteSources, got)
}
