package model

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMCPChannelBatchTagSingleConnection(t *testing.T) {
	// 单连接可稳定暴露事务内调用全局 DB 导致的自等待；上下文只用于限制失败等待。
	for _, rollback := range []bool{false, true} {
		name := "提交同步标签"
		if rollback {
			name = "能力更新失败全部回滚"
		}
		t.Run(name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, err := db.DB()
			require.NoError(t, err)
			sqlDB.SetMaxOpenConns(1)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			previousDB := DB
			DB = db.WithContext(ctx)
			t.Cleanup(func() {
				DB = previousDB
				cancel()
				_ = sqlDB.Close()
			})
			require.NoError(t, DB.AutoMigrate(&Channel{}, &Ability{}, &Group{}))
			group := Group{Code: "legacy-tag-group", Name: "标签测试分组", Status: 1, Ratio: 1}
			require.NoError(t, DB.Create(&group).Error)
			oldTag := "原标签"
			channels := []Channel{
				{Name: "选择一", Key: "local-one", Models: "model-a,model-b", Group: group.Code, Status: common.ChannelStatusEnabled, Tag: &oldTag},
				{Name: "选择二", Key: "local-two", Models: "model-c", Group: group.Code, Status: common.ChannelStatusManuallyDisabled, Tag: &oldTag},
				{Name: "未选择", Key: "local-three", Models: "model-d", Group: group.Code, Status: common.ChannelStatusEnabled, Tag: &oldTag},
			}
			for i := range channels {
				require.NoError(t, DB.Create(&channels[i]).Error)
				require.NoError(t, channels[i].UpdateAbilities(DB))
			}
			var before []Ability
			require.NoError(t, DB.Order("channel_id, model").Find(&before).Error)
			expectedFailure := errors.New("测试能力表写入失败")
			if rollback {
				require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("mcp_tag_test_fail", func(tx *gorm.DB) {
					if tx.Statement.Table == "abilities" {
						tx.AddError(expectedFailure)
					}
				}))
			}
			newTag := "更新标签"
			err = BatchSetChannelTag([]int{channels[0].Id, channels[1].Id}, &newTag)
			if rollback {
				require.ErrorIs(t, err, expectedFailure)
			} else {
				require.NoError(t, err)
			}
			var saved []Channel
			require.NoError(t, DB.Order("id").Find(&saved).Error)
			require.Len(t, saved, 3)
			for i := range saved {
				want := oldTag
				if !rollback && i < 2 {
					want = newTag
				}
				require.NotNil(t, saved[i].Tag)
				assert.Equal(t, want, *saved[i].Tag)
				assert.Equal(t, channels[i].Status, saved[i].Status)
			}
			var after []Ability
			require.NoError(t, DB.Order("channel_id, model").Find(&after).Error)
			require.Len(t, after, len(before))
			if rollback {
				assert.Equal(t, before, after)
			} else {
				for i := range before {
					if before[i].ChannelId != channels[2].Id {
						before[i].Tag = &newTag
					}
				}
				assert.Equal(t, before, after, "能力表须保留分组、模型和启用状态，只同步所选渠道的新标签")
			}
		})
	}
}
