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

// 不限分组授权使用手写子查询与布尔参数，需要在三种数据库上验证同一语义；
// MySQL/PostgreSQL 通过 TEST_MYSQL_DSN、TEST_POSTGRES_DSN 指向一次性空库，未配置时跳过。
func TestPromptAllGroupsAuthorizationDatabaseMatrix(t *testing.T) {
	for _, dialect := range []string{"sqlite", "mysql", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			var driver gorm.Dialector
			switch dialect {
			case "sqlite":
				driver = sqlite.Open("file:prompt_all_groups_matrix?mode=memory&cache=shared")
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
			tables := []interface{}{&Option{}, &Group{}, &GroupAlias{}, &AutoGroupMember{}, &ChannelGroupBinding{}, &TokenGroupBinding{}, &Channel{}, &Token{}, &User{}, &Ability{},
				&PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}, &PromptSkillRecord{}, &PromptSkillFile{}, &PromptSkillChunk{}}
			previousDB, previousLogDB := DB, LOG_DB
			previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
			DB, LOG_DB = db, db
			common.SetDatabaseTypes(common.DatabaseType(dialect), common.DatabaseType(dialect))
			InitDBColumns()
			ClearProgressiveSkillContentCache()
			t.Cleanup(func() {
				ClearProgressiveSkillContentCache()
				require.NoError(t, db.Migrator().DropTable(tables...))
				DB, LOG_DB = previousDB, previousLogDB
				common.SetDatabaseTypes(previousMain, previousLog)
				InitDBColumns()
				require.NoError(t, sqlDB.Close())
			})
			require.NoError(t, db.AutoMigrate(tables...))
			var version string
			query := "SELECT version()"
			if dialect == "sqlite" {
				query = "SELECT sqlite_version()"
			}
			require.NoError(t, db.Raw(query).Scan(&version).Error)
			t.Logf("database version: %s", version)

			first := &Group{Code: "matrix-default", Name: "矩阵默认组", Ratio: 1}
			second := &Group{Code: "matrix-vip", Name: "矩阵会员组", Ratio: 1}
			paused := &Group{Code: "matrix-paused", Name: "矩阵停用组", Ratio: 1}
			for _, group := range []*Group{first, second, paused} {
				require.NoError(t, db.Create(group).Error)
			}
			require.NoError(t, db.Model(paused).Update("status", GroupStatusDisabled).Error)
			token := &Token{UserId: 77, Key: "matrix-all-groups-key", Group: first.Code}
			require.NoError(t, db.Create(token).Error)

			open, err := SavePromptTemplate(PromptTemplate{Title: "通用技能", Description: "任意分组", Content: "通用正文", LoadMode: PromptLoadModeProgressive, Enabled: true, AllGroups: true, GroupIDs: []int{first.Id}})
			require.NoError(t, err)
			assert.True(t, open.AllGroups)
			assert.Empty(t, open.GroupIDs)
			restricted, err := SavePromptTemplate(PromptTemplate{Title: "默认组专用", Description: "仅默认组", Content: "专用正文", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{first.Id}})
			require.NoError(t, err)
			eager, err := SavePromptTemplate(PromptTemplate{Title: "通用全文", Content: "全文正文", Enabled: true, AllGroups: true})
			require.NoError(t, err)
			require.Error(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: open.ID, Enabled: true}}, nil))
			require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{
				{TemplateID: open.ID, Enabled: true}, {TemplateID: restricted.ID, Enabled: true, SortOrder: 1}, {TemplateID: eager.ID, Enabled: true, SortOrder: 2},
			}, []int{first.Id}))

			assert.Equal(t, []int{open.ID, restricted.ID}, activeSkillTemplateIDs(t, token.Id, first.Code))
			assert.Equal(t, []int{open.ID}, activeSkillTemplateIDs(t, token.Id, second.Code))
			assert.Empty(t, activeSkillTemplateIDs(t, token.Id, paused.Code))
			page, err := ReadActiveTokenSkillFile(token.Id, second.Code, open.ID, open.Skills[0].ID, open.Version, "", 0, 1024)
			require.NoError(t, err)
			assert.Equal(t, "通用正文", page.Content)
			_, err = ReadActiveTokenSkillFile(token.Id, second.Code, restricted.ID, restricted.Skills[0].ID, restricted.Version, "", 0, 1024)
			require.Error(t, err)
			_, err = ReadActiveTokenSkillFile(token.Id, paused.Code, open.ID, open.Skills[0].ID, open.Version, "", 0, 1024)
			require.Error(t, err)
			for _, group := range []*Group{second, paused} {
				loaded, loadErr := GetActiveTokenPrompts(token.Id, group.Code)
				require.NoError(t, loadErr)
				if group == second {
					require.Len(t, loaded, 1)
					assert.Equal(t, eager.ID, loaded[0].ID)
				} else {
					assert.Empty(t, loaded)
				}
			}

			require.NoError(t, db.Model(&PromptTemplate{}).Where("id = ?", open.ID).Update("all_groups", gorm.Expr("NULL")).Error)
			assert.Empty(t, activeSkillTemplateIDs(t, token.Id, second.Code), "存量 NULL 按白名单处理")
			assert.Equal(t, []int{restricted.ID}, activeSkillTemplateIDs(t, token.Id, first.Code), "不限分组已清空白名单，NULL 后不再对任何分组生效")

		})
	}
}
