package service

import (
	"math"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMCPKeyExpiryUpperBound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.MCPKey{}))
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; _ = sqlDB.Close() })
	user := &model.UserBase{Id: 9, Role: common.RoleRootUser}
	for _, expiresAt := range []int64{253402300800, 8640000000001, math.MaxInt64} {
		key, secret, err := CreateMCPKey(user, "越界时间", []string{"newapi_context"}, expiresAt)
		require.Error(t, err)
		assert.Nil(t, key)
		assert.Empty(t, secret)
	}
	var count int64
	require.NoError(t, db.Model(&model.MCPKey{}).Count(&count).Error)
	assert.Zero(t, count, "非法到期时间不得持久化")
	for _, expiresAt := range []int64{time.Now().Add(time.Hour).Unix(), 253402300799} {
		key, secret, err := CreateMCPKey(user, "有效时间", []string{"newapi_context"}, expiresAt)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)
		var stored model.MCPKey
		require.NoError(t, db.First(&stored, key.ID).Error)
		assert.Equal(t, expiresAt, stored.ExpiresAt)
	}
	names := make([]string, 0)
	for _, tool := range MCPTools() {
		names = append(names, tool.Name)
	}
	key, _, err := CreateMCPKey(user, "全部当前工具", names, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err, "完整后台授权不能被旧固定工具数量上限拦截")
	assert.Equal(t, names, key.Tools)
}

func TestMCPVisibleToolsQueriesOnlyGrantedPermissionsOncePerRequest(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.CasbinRule{}))
	previous := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previous; _ = sqlDB.Close() })
	require.NoError(t, db.Create(&model.CasbinRule{Ptype: "p", V0: authz.RoleSubject(authz.BuiltInRoleAdmin), V1: "channel", V2: "read", V3: "allow"}).Error)
	queries := 0
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("count_mcp_permission_queries", func(tx *gorm.DB) {
		if tx.Statement.Table == "casbin_rule" {
			queries++
		}
	}))
	user := &model.UserBase{Id: 9, Role: common.RoleAdminUser}
	key := &model.MCPKey{Tools: []string{"newapi_channel_get", "newapi_channel_models"}}
	tools := MCPVisibleTools(user, key)
	require.Len(t, tools, 2)
	assert.Equal(t, 1, queries, "同权限的两个已授权工具只查一次，未授权目录工具不查权限")
	// 下一次请求必须看到最新撤权，不能延用上一次的权限缓存。
	require.NoError(t, db.Create(&model.CasbinRule{Ptype: "p", V0: authz.UserSubject(user.Id), V1: "channel", V2: "read", V3: "deny"}).Error)
	queries = 0
	assert.Empty(t, MCPVisibleTools(user, key))
	assert.Equal(t, 1, queries, "拒绝结果也只在本次请求内复用")
	queries = 0
	assert.Empty(t, MCPVisibleTools(user, &model.MCPKey{Tools: []string{}}))
	assert.Zero(t, queries, "空授权集合不得查询目录权限")
}
