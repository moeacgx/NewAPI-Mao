package model

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func mcpTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mcp.db")), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&MCPKey{}, &MCPCallLog{}, &MCPWriteRequest{}))
	old := DB
	DB = db
	t.Cleanup(func() { DB = old })
	return db
}

func TestMCPWriteRequestClaimAndReplay(t *testing.T) {
	mcpTestDB(t)
	var wg sync.WaitGroup
	claims := make(chan bool, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, claimed, err := ClaimMCPWriteRequest(1, "same-id", "same-hash")
			claims <- claimed
			errs <- err
		}()
	}
	wg.Wait()
	assert.NotEqual(t, <-claims, <-claims, "同一写请求只能有一个执行者")
	require.NoError(t, <-errs)
	require.NoError(t, <-errs)
	record, claimed, err := ClaimMCPWriteRequest(1, "same-id", "different-hash")
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, "same-hash", record.RequestHash)
	assert.Equal(t, "pending", record.State)
	require.NoError(t, CompleteMCPWriteRequest(record.ID, `{"isError":false}`, true))
	record, claimed, err = ClaimMCPWriteRequest(1, "same-id", "same-hash")
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, "done", record.State)
	assert.Equal(t, `{"isError":false}`, record.Result)
}

func TestMCPKeyOwnerIsolation(t *testing.T) {
	db := mcpTestDB(t)
	key := MCPKey{UserID: 7, Name: "本人", KeyHash: "hash", Tools: []string{"newapi_context"}}
	require.NoError(t, db.Create(&key).Error)
	assert.ErrorIs(t, RevokeMCPKey(8, key.ID), gorm.ErrRecordNotFound)
	items, total, err := ListMCPKeys(8, 0, 20)
	require.NoError(t, err)
	assert.Empty(t, items)
	assert.Zero(t, total)
	require.NoError(t, RevokeMCPKey(7, key.ID))
	require.NoError(t, RevokeMCPKey(7, key.ID), "重复撤销本人的密钥应保持成功")
	require.NoError(t, db.First(&key, key.ID).Error)
	assert.Positive(t, key.RevokedAt)
}

func TestMCPClaimDoesNotTrustAffectedRows(t *testing.T) {
	db := mcpTestDB(t)
	first, claimed, err := ClaimMCPWriteRequest(1, "mysql-found-rows", "original")
	require.NoError(t, err)
	require.True(t, claimed)
	require.NotEmpty(t, first.OwnerNonce)
	// 模拟 MySQL CLIENT_FOUND_ROWS：唯一冲突 no-op 仍报告一行匹配。
	require.NoError(t, db.Callback().Create().After("gorm:create").Register("mcp_simulate_found_rows", func(tx *gorm.DB) {
		if tx.Statement.Table == "mcp_write_requests" && tx.Error == nil {
			tx.RowsAffected = 1
		}
	}))
	second, claimed, err := ClaimMCPWriteRequest(1, "mysql-found-rows", "changed")
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Equal(t, first.OwnerNonce, second.OwnerNonce)
	assert.Equal(t, "original", second.RequestHash)
}
