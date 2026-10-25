package model

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 分类删除事务需同时更新模板与删除分类，分别在支持的数据库方言中验证。
func TestDeletePromptCategoryDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open("file:prompt_category_delete_matrix?mode=memory&cache=shared")
			case "mysql":
				dsn := os.Getenv("TEST_MYSQL_DSN")
				if dsn == "" {
					t.Skip("TEST_MYSQL_DSN not configured")
				}
				driver = mysql.Open(dsn)
			case "postgres":
				dsn := os.Getenv("TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("TEST_POSTGRES_DSN not configured")
				}
				driver = postgres.Open(dsn)
			}
			db, err := gorm.Open(driver, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			previousDB, previousLogDB := DB, LOG_DB
			previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
			DB, LOG_DB = db, db
			common.SetDatabaseTypes(common.DatabaseType(dialect), common.DatabaseType(dialect))
			InitDBColumns()
			t.Cleanup(func() {
				DB, LOG_DB = previousDB, previousLogDB
				common.SetDatabaseTypes(previousMain, previousLog)
				InitDBColumns()
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(&PromptCategory{}, &PromptTemplate{}))

			category, err := SavePromptCategory(PromptCategory{Name: "待删除"})
			require.NoError(t, err)
			template := PromptTemplate{Title: "模板", CategoryID: category.ID, Version: 3}
			require.NoError(t, db.Create(&template).Error)
			other := PromptTemplate{Title: "其他模板", Version: 1}
			require.NoError(t, db.Create(&other).Error)

			moved, err := DeletePromptCategory(category.ID, category.Version)
			require.NoError(t, err)
			assert.Equal(t, int64(1), moved)
			var stored PromptTemplate
			require.NoError(t, db.First(&stored, template.ID).Error)
			assert.Zero(t, stored.CategoryID)
			assert.Equal(t, 3, stored.Version)
			_, err = DeletePromptCategory(category.ID, category.Version)
			require.ErrorIs(t, err, ErrPromptCategoryNotFound)
		})
	}
}
