package model

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteDefaultConnectionWriteIsolation(t *testing.T) {
	previousPath := common.SQLitePath
	_, query, _ := strings.Cut(previousPath, "?")
	common.SQLitePath = filepath.Join(t.TempDir(), "connection.db") + "?" + query
	t.Cleanup(func() { common.SQLitePath = previousPath })
	t.Setenv("SQLITE_CONNECTION_TEST_DSN", "local")
	db, kind, err := chooseDB("SQLITE_CONNECTION_TEST_DSN", false)
	require.NoError(t, err)
	assert.Equal(t, common.DatabaseTypeSQLite, kind)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(2)
	require.NoError(t, db.Exec("CREATE TABLE concurrency_probe (id INTEGER PRIMARY KEY, value INTEGER NOT NULL)").Error)
	require.NoError(t, db.Exec("INSERT INTO concurrency_probe VALUES (1, 5000000000)").Error)
	ctx := context.Background()
	reader, err := sqlDB.Conn(ctx)
	require.NoError(t, err)
	defer reader.Close()
	var mode string
	var timeout int
	require.NoError(t, reader.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode))
	require.NoError(t, reader.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeout))
	assert.Equal(t, "wal", mode)
	assert.Equal(t, 30000, timeout)

	tx, err := sqlDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	var quota int64
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT value FROM concurrency_probe WHERE id = 1").Scan(&quota))
	assert.Equal(t, int64(5000000000), quota)
	// 第二个物理连接也必须获得同样的初始化 PRAGMA。
	require.NoError(t, tx.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&timeout))
	assert.Equal(t, 30000, timeout)
	// 将竞争连接的等待缩短，确定性检查 BEGIN 已持有写锁，避免等待默认 30 秒。
	_, err = reader.ExecContext(ctx, "PRAGMA busy_timeout=1")
	require.NoError(t, err)
	_, err = reader.ExecContext(ctx, "UPDATE concurrency_probe SET value = 1 WHERE id = 1")
	assert.ErrorContains(t, err, "database is locked")
	_, err = tx.ExecContext(ctx, "UPDATE concurrency_probe SET value = 5000000001 WHERE id = 1")
	require.NoError(t, err)
	require.NoError(t, reader.QueryRowContext(ctx, "SELECT value FROM concurrency_probe WHERE id = 1").Scan(&quota))
	assert.Equal(t, int64(5000000000), quota, "WAL 读者应读取上一个已提交快照")
	require.NoError(t, tx.Commit())
	require.NoError(t, reader.QueryRowContext(ctx, "SELECT value FROM concurrency_probe WHERE id = 1").Scan(&quota))
	assert.Equal(t, int64(5000000001), quota)
}
