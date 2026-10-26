package model

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSkillFilterMatchesActualSkillUsage(t *testing.T) {
	cases := []struct {
		name, other string
		used        bool
	}{
		{"始终装载", `{"prompt_templates":[{"id":1}]}`, true},
		{"混合模板条目", `{"prompt_templates":[null,{},"x",{"id":1}]}`, true},
		{"正式读取", `{"gateway_skills":{"stage":"generation","files":[{"template_id":1,"skill_id":"go"}]}}`, true},
		{"Unicode转义键", `{"\u0070rompt_templates":[{"id":2}]}`, true},
		{"发现优先", `{"prompt_templates":[{"id":1}],"gateway_skills":{"stage":"discovery","files":[{"template_id":1,"skill_id":"go"}]}}`, false},
		{"未读取", `{"gateway_skills":{"stage":"generation","catalogue_count":3,"files":[]}}`, false},
		{"模板ID字符串", `{"prompt_templates":[{"id":"1"}]}`, false},
		{"模板非数组", `{"prompt_templates":{"id":1}}`, false},
		{"空数组项", `{"prompt_templates":[{},null]}`, false},
		{"文件缺少技能", `{"gateway_skills":{"stage":"generation","files":[{"template_id":1,"skill_id":""}]}}`, false},
		{"不同文件不可拼接", `{"gateway_skills":{"stage":"generation","files":[{"template_id":1},{"skill_id":"go"}]}}`, false},
		{"文件ID类型错误", `{"gateway_skills":{"stage":"generation","files":[{"template_id":"1","skill_id":"go"}]}}`, false},
		{"文件非数组", `{"gateway_skills":{"stage":"generation","files":{"template_id":1,"skill_id":"go"}}}`, false},
		{"无正式阶段", `{"gateway_skills":{"files":[{"template_id":1,"skill_id":"go"}]}}`, false},
		{"嵌套键不匹配", `{"admin_info":{"prompt_templates":[{"id":1}]}}`, false},
		{"无技能", `{}`, false},
		{"坏JSON", `{"prompt_templates":[{"id":1}]`, false},
		{"错误根类型", `[{"prompt_templates":[{"id":1}]}]`, false},
		{"空值", `null`, false},
	}
	db := setupOperationAuditLogTestDB(t)
	for i, item := range cases {
		t.Run(item.name, func(t *testing.T) { assert.Equal(t, item.used, logUsesSkill(item.other)) })
		require.NoError(t, db.Create(&Log{Id: i + 1, UserId: 1, Type: LogTypeConsume, Other: item.other, CreatedAt: 100}).Error)
	}
	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "", LogQueryOptions{Skill: "used"})
	require.NoError(t, err)
	assert.Equal(t, int64(4), total)
	ids := make([]int, len(logs))
	for i, log := range logs {
		ids[i] = log.Id
	}
	assert.Equal(t, []int{4, 3, 2, 1}, ids)
}

func TestLogSkillFilterPaginationUserIsolationAndStatistics(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	now := time.Now().Unix()
	used := `{"prompt_templates":[{"id":1}],"admin_info":{"secret":"hidden"}}`
	logs := []Log{
		{Id: 1, UserId: 101, CreatedAt: now - 120, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: used, Quota: 11, PromptTokens: 100},
		{Id: 2, UserId: 101, CreatedAt: now - 20, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: `{}`, Quota: 50, PromptTokens: 50},
		{Id: 3, UserId: 101, CreatedAt: now - 20, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: used, Quota: 7, PromptTokens: 2, CompletionTokens: 3},
		{Id: 4, UserId: 101, CreatedAt: now - 10, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: used, Quota: 13, PromptTokens: 5, CompletionTokens: 7},
		{Id: 5, UserId: 202, CreatedAt: now - 5, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: used, Quota: 1000, PromptTokens: 1000},
		{Id: 6, UserId: 101, CreatedAt: now - 4, Type: LogTypeError, TokenName: "selected", ModelName: "model", Other: used, Quota: 999, PromptTokens: 999},
		{Id: 7, UserId: 101, CreatedAt: now - 3, Type: LogTypeConsume, TokenName: "selected", ModelName: "model", Other: `{"gateway_skills":{"stage":"discovery"},"prompt_templates":[{"id":1}]}`, Quota: 888, PromptTokens: 888},
		{Id: 8, UserId: 101, CreatedAt: now - 2, Type: LogTypeConsume, TokenName: "other", ModelName: "model", Other: used, Quota: 777},
	}
	require.NoError(t, db.Create(&logs).Error)
	query := LogQueryOptions{Skill: "used"}
	page, total, err := GetAllLogsByUserID(LogTypeConsume, now-200, now, "model", "", 101, "selected", 1, 1, 0, "", "", "", query)
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, 3, page[0].Id)
	self, total, err := GetUserLogs(101, LogTypeConsume, now-200, now, "model", "selected", 2, 1, "", "", "", query)
	require.NoError(t, err)
	require.Len(t, self, 1)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, now-120, self[0].CreatedAt)
	assert.Equal(t, 101, self[0].UserId)
	assert.NotContains(t, self[0].Other, "admin_info")
	stat, err := SumUsedQuotaByUserID(LogTypeUnknown, now-200, now, "model", "", 101, "selected", 0, "", query)
	require.NoError(t, err)
	assert.Equal(t, Stat{Quota: 31, Rpm: 2, Tpm: 17}, stat)
	// 费用按所选历史区间，速率仍按原有最近 60 秒口径。
	stat, err = SumUsedQuotaByUserID(LogTypeUnknown, now-200, now-100, "model", "", 101, "selected", 0, "", query)
	require.NoError(t, err)
	assert.Equal(t, Stat{Quota: 11, Rpm: 2, Tpm: 17}, stat)
	_, total, err = GetAllLogsByUserID(LogTypeConsume, now-200, now, "model", "", 101, "selected", 10, 1, 0, "", "", "", query)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	_, unfiltered, err := GetAllLogsByUserID(LogTypeConsume, now-200, now, "model", "", 101, "selected", 0, 20, 0, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(5), unfiltered)
}

func TestLogSkillFilterCancellationReturnsNoPartialResults(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	require.NoError(t, db.Create(&Log{Type: LogTypeConsume, UserId: 1, Other: `{"prompt_templates":[{"id":1}]}`, Quota: 7}).Error)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	query := LogQueryOptions{Skill: "used", Context: ctx}
	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "", query)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, logs)
	assert.Zero(t, total)
	logs, total, err = GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "", query)
	require.Error(t, err)
	assert.Nil(t, logs)
	assert.Zero(t, total)
	stat, err := SumUsedQuota(LogTypeUnknown, 0, 0, "", "", "", 0, "", query)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, Stat{}, stat)
}

func TestLogSkillStatisticsOverflowReturnsError(t *testing.T) {
	db := setupOperationAuditLogTestDB(t)
	require.NoError(t, db.Create(&[]Log{
		{Type: LogTypeConsume, Quota: math.MaxInt, Other: `{"prompt_templates":[{"id":1}]}`},
		{Type: LogTypeConsume, Quota: 1, Other: `{"prompt_templates":[{"id":1}]}`},
	}).Error)
	stat, err := SumUsedQuota(LogTypeUnknown, 0, 0, "", "", "", 0, "", LogQueryOptions{Skill: "used"})
	require.ErrorContains(t, err, "overflow")
	assert.Equal(t, Stat{}, stat)
}
