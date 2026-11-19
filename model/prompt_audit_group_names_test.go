package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPromptAuditGroupNamesTest(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupPromptAuditTestDB(t)
	require.NoError(t, db.AutoMigrate(&Group{}, &GroupAlias{}))
	require.NoError(t, db.Create(&[]Group{
		{Id: 11, Code: "default", Name: "普通用户", Status: GroupStatusActive},
		{Id: 22, Code: "vip", Name: "当前专业组", Status: GroupStatusActive},
		{Id: 44, Code: "reused", Name: "重新建立的组", Status: GroupStatusActive},
	}).Error)
	require.NoError(t, db.Create(&GroupAlias{GroupId: 22, Alias: "legacy-vip"}).Error)
	return db
}

func TestIssue403PromptAuditGroupDisplayProjection(t *testing.T) {
	db := setupPromptAuditGroupNamesTest(t)
	tests := []struct {
		name                 string
		id                   int
		code, snapshot, want string
	}{
		{"稳定ID优先于不一致code", 22, "default", "历史专业组", "当前专业组"},
		{"无ID历史alias", 0, "legacy-vip", "旧别名显示名", "当前专业组"},
		{"旧记录只在name保存alias", 0, "", "legacy-vip", "当前专业组"},
		{"已删除ID不重新认code", 33, "reused", "删除前名称", "删除前名称"},
		{"已删除ID缺名回退code", 33, "reused", "", "reused"},
		{"未知code保留历史名称", 0, "unknown", "历史名称", "历史名称"},
		{"未知code缺名回退", 0, "unknown", "", "unknown"},
		{"仅有已删除ID", 33, "", "", "33"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := promptAuditTestEvent(tt.name, time.Now().Add(time.Hour).Unix())
			event.GroupId, event.GroupCode, event.GroupName = tt.id, tt.code, tt.snapshot
			event.ChannelGroups = []PromptAuditEventChannelGroup{{Id: tt.id, Code: tt.code, Name: tt.snapshot}}
			event.TokenGroups = []PromptAuditEventTokenGroup{{Id: tt.id, Code: tt.code, Name: tt.snapshot}}
			require.NoError(t, CreatePromptAuditEvent(&event))
			var before PromptAuditEvent
			require.NoError(t, db.First(&before, event.Id).Error)
			detail, err := GetPromptAuditEvent(event.Id)
			require.NoError(t, err)
			listed, count, err := ListPromptAuditEvents(PromptAuditEventFilter{RequestId: tt.name}, 1, 20)
			require.NoError(t, err)
			require.EqualValues(t, 1, count)
			require.Len(t, listed, 1)
			for _, response := range []PromptAuditEvent{*detail, listed[0]} {
				assert.Equal(t, tt.snapshot, response.GroupName)
				assert.Equal(t, tt.code, response.GroupCode)
				assert.Equal(t, tt.id, response.GroupId)
				assert.Equal(t, tt.snapshot, response.ChannelGroups[0].Name)
				assert.Equal(t, tt.snapshot, response.TokenGroups[0].Name)
				encoded, err := common.Marshal(response)
				require.NoError(t, err)
				var payload map[string]any
				require.NoError(t, common.Unmarshal(encoded, &payload))
				assert.Equal(t, tt.want, payload["group_display_name"])
				for _, key := range []string{"channel_groups", "token_groups"} {
					groups, ok := payload[key].([]any)
					require.True(t, ok)
					require.Len(t, groups, 1)
					group, ok := groups[0].(map[string]any)
					require.True(t, ok)
					assert.Equal(t, tt.want, group["display_name"])
				}
			}
			var after PromptAuditEvent
			require.NoError(t, db.First(&after, event.Id).Error)
			assert.Equal(t, before, after, "读取展示名称不得更新原始事件证据")
		})
	}
}

func TestIssue403PromptAuditRenameDoesNotPersistDisplayFields(t *testing.T) {
	db := setupPromptAuditGroupNamesTest(t)
	event := promptAuditTestEvent("rename-snapshot", time.Now().Add(time.Hour).Unix())
	event.GroupId, event.GroupCode, event.GroupName = 22, "legacy-vip", "事发当时名称"
	event.ChannelGroups = []PromptAuditEventChannelGroup{{Id: 22, Code: "legacy-vip", Name: "事发当时名称"}}
	event.TokenGroups = []PromptAuditEventTokenGroup{{Id: 22, Code: "legacy-vip", Name: "事发当时名称"}}
	require.NoError(t, CreatePromptAuditEvent(&event))
	require.NoError(t, db.Model(&Group{}).Where("id = ?", 22).Update("name", "再次改名").Error)
	loaded, err := GetPromptAuditEvent(event.Id)
	require.NoError(t, err)
	data, err := common.Marshal(loaded)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"group_display_name":"再次改名"`)
	assert.Equal(t, "事发当时名称", loaded.GroupName)
	// 模拟下游复用读出的事件写入快照，展示值也不得混入证据。
	loaded.Id = 0
	loaded.RequestId = "copied-snapshot"
	require.NoError(t, CreatePromptAuditEvent(loaded))
	var stored PromptAuditEvent
	require.NoError(t, db.First(&stored, loaded.Id).Error)
	assert.Equal(t, event.ChannelGroupDetails, stored.ChannelGroupDetails)
	assert.Equal(t, event.TokenGroupDetails, stored.TokenGroupDetails)
	assert.NotContains(t, stored.ChannelGroupDetails, "display_name")
	assert.NotContains(t, stored.TokenGroupDetails, "display_name")
	assert.Equal(t, event.PromptCiphertext, stored.PromptCiphertext)
	assert.Equal(t, event.ContextSegments, stored.ContextSegments)
	assert.False(t, db.Migrator().HasColumn(&PromptAuditEvent{}, "group_display_name"))
}

func TestIssue403PromptAuditDisplayNamesAreBatched(t *testing.T) {
	db := setupPromptAuditGroupNamesTest(t)
	for i := 0; i < 25; i++ {
		event := promptAuditTestEvent("batch-names", time.Now().Add(time.Hour).Unix())
		event.GroupId, event.GroupCode = 22, "vip"
		event.ChannelGroups = []PromptAuditEventChannelGroup{{Code: "legacy-vip"}}
		event.TokenGroups = []PromptAuditEventTokenGroup{{Id: 22, Code: "vip"}}
		require.NoError(t, CreatePromptAuditEvent(&event))
	}
	queries := 0
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("issue403:group-query-count", func(tx *gorm.DB) {
		if tx.Statement.Table == "groups" || tx.Statement.Table == "group_aliases" {
			queries++
		}
	}))
	events, total, err := ListPromptAuditEvents(PromptAuditEventFilter{RequestId: "batch-names"}, 1, 100)
	require.NoError(t, err)
	require.EqualValues(t, 25, total)
	require.Len(t, events, 25)
	assert.Positive(t, queries)
	assert.LessOrEqual(t, queries, 3, "批量展示名称不得按事件或嵌套分组逐行查询")
}

func TestIssue403PromptAuditDeletedGroupDoesNotBindReusedCode(t *testing.T) {
	db := setupPromptAuditGroupNamesTest(t)
	event := promptAuditTestEvent("delete-and-recreate", time.Now().Add(time.Hour).Unix())
	event.GroupId, event.GroupCode, event.GroupName = 22, "vip", "删除前名称"
	event.ChannelGroups = []PromptAuditEventChannelGroup{{Id: 22, Code: "vip", Name: "删除前名称"}}
	event.TokenGroups = []PromptAuditEventTokenGroup{{Id: 22, Code: "vip", Name: "删除前名称"}}
	require.NoError(t, CreatePromptAuditEvent(&event))
	require.NoError(t, db.Delete(&Group{}, 22).Error)
	require.NoError(t, db.Create(&Group{Id: 55, Code: "vip", Name: "新建同code分组", Status: GroupStatusActive}).Error)
	loaded, err := GetPromptAuditEvent(event.Id)
	require.NoError(t, err)
	assert.Equal(t, 22, loaded.GroupId)
	assert.Equal(t, "删除前名称", loaded.GroupDisplayName)
	assert.Equal(t, "删除前名称", loaded.ChannelGroups[0].DisplayName)
	assert.Equal(t, "删除前名称", loaded.TokenGroups[0].DisplayName)
	assert.Equal(t, event.ChannelGroupDetails, loaded.ChannelGroupDetails)
	assert.Equal(t, event.TokenGroupDetails, loaded.TokenGroupDetails)
}

func TestIssue403PromptAuditNameLookupFailureKeepsEvidenceReadable(t *testing.T) {
	db := setupPromptAuditTestDB(t) // 旧版兼容fixture没有groups表，模拟名称来源不可用。
	event := promptAuditTestEvent("no-groups-table", time.Now().Add(time.Hour).Unix())
	event.GroupId, event.GroupCode, event.GroupName = 22, "vip", "事发名称"
	require.NoError(t, CreatePromptAuditEvent(&event))
	loaded, err := GetPromptAuditEvent(event.Id)
	require.NoError(t, err)
	data, err := common.Marshal(loaded)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"group_display_name":"事发名称"`)
	var stored PromptAuditEvent
	require.NoError(t, db.First(&stored, event.Id).Error)
	assert.Equal(t, event.GroupId, stored.GroupId)
	assert.Equal(t, event.GroupName, stored.GroupName)
}
