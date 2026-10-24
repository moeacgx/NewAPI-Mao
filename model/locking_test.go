package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

// lockForUpdate must emit FOR UPDATE on databases that support it and skip
// it on SQLite, where the syntax does not exist.
//
// The dummy dialector is used because SQLite drivers strip locking clauses
// from the generated SQL, which would mask what the helper itself does.
func TestLockForUpdateEmitsRowLock(t *testing.T) {
	dummyDB, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	require.NoError(t, err)
	buildSQL := func() string {
		var rows []Redemption
		return lockForUpdate(dummyDB).Where("id = ?", 1).Find(&rows).Statement.SQL.String()
	}

	t.Cleanup(func() {
		common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	})

	common.SetDatabaseTypes(common.DatabaseTypeMySQL, common.DatabaseTypeSQLite)
	assert.Contains(t, buildSQL(), "FOR UPDATE")

	common.SetDatabaseTypes(common.DatabaseTypePostgreSQL, common.DatabaseTypeSQLite)
	assert.Contains(t, buildSQL(), "FOR UPDATE")

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	assert.NotContains(t, buildSQL(), "FOR UPDATE")
}

func TestLockForShareUsesCompatibleDatabaseSyntax(t *testing.T) {
	mainType, logType := common.MainDatabaseType(), common.LogDatabaseType()
	t.Cleanup(func() { common.SetDatabaseTypes(mainType, logType) })
	for _, test := range []struct {
		name      string
		dbType    common.DatabaseType
		dialector gorm.Dialector
		want      string
	}{
		{"MySQL 5.7", common.DatabaseTypeMySQL, mysql.New(mysql.Config{SkipInitializeWithVersion: true, DontSupportForShareClause: true}), "LOCK IN SHARE MODE"},
		{"MySQL 8", common.DatabaseTypeMySQL, mysql.New(mysql.Config{SkipInitializeWithVersion: true}), "FOR SHARE"},
		{"PostgreSQL 9.6", common.DatabaseTypePostgreSQL, postgres.New(postgres.Config{DSN: "host=localhost dbname=unused"}), "FOR SHARE"},
		{"SQLite", common.DatabaseTypeSQLite, tests.DummyDialector{}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			common.SetDatabaseTypes(test.dbType, common.DatabaseTypeSQLite)
			database, err := gorm.Open(test.dialector, &gorm.Config{DryRun: true, DisableAutomaticPing: true})
			require.NoError(t, err)
			if database.Dialector.Name() != "dummy" {
				if sqlDB, sqlErr := database.DB(); sqlErr == nil {
					t.Cleanup(func() { _ = sqlDB.Close() })
				}
			}
			var token Token
			sql := lockForShare(database).Select("id").First(&token, 1).Statement.SQL.String()
			if test.want == "" {
				assert.NotContains(t, sql, "FOR SHARE")
				assert.NotContains(t, sql, "LOCK IN SHARE MODE")
			} else {
				assert.Contains(t, sql, test.want)
			}
			assert.NotContains(t, sql, "FOR UPDATE", "只读事务不应串行化并发读取")
		})
	}
}
