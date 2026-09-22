package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogUserIDFilterAppliesToRowsAndStatistics(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	now := time.Now().Unix()
	logs := []*Log{
		{UserId: 101, CreatedAt: now, Type: LogTypeConsume, Username: "renamed-user", Quota: 7, PromptTokens: 2, CompletionTokens: 3},
		{UserId: 202, CreatedAt: now, Type: LogTypeConsume, Username: "other-user", Quota: 11, PromptTokens: 5, CompletionTokens: 8},
		{UserId: 101, CreatedAt: now, Type: LogTypeManage, Username: "renamed-user", Quota: 99},
	}
	for _, log := range logs {
		require.NoError(t, db.Create(log).Error)
	}

	items, total, err := GetAllLogsByUserID(
		LogTypeUnknown, now-10, now+10, "", "", 101, "", 0, 20, 0, "", "", "",
	)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	assert.ElementsMatch(t, []int{101, 101}, []int{items[0].UserId, items[1].UserId})

	stat, err := SumUsedQuotaByUserID(
		LogTypeUnknown, now-10, now+10, "", "", 101, "", 0, "",
	)
	require.NoError(t, err)
	assert.Equal(t, 7, stat.Quota)
	assert.Equal(t, 1, stat.Rpm)
	assert.Equal(t, 5, stat.Tpm)
}
