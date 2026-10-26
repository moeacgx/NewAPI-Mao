package model

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// LogQueryOptions 为旧日志调用提供可选筛选与请求取消，省略时保留原查询行为。
type LogQueryOptions struct {
	Skill   string
	Context context.Context
}

func logQueryContext(options []LogQueryOptions) (LogQueryOptions, context.CancelFunc, error) {
	var query LogQueryOptions
	if len(options) > 0 {
		query = options[0]
	}
	if query.Skill != "" && query.Skill != "used" {
		return query, nil, errors.New("invalid skill filter")
	}
	if query.Context == nil {
		query.Context = context.Background()
	}
	var cancel context.CancelFunc
	if query.Skill == "used" {
		query.Context, cancel = context.WithTimeout(query.Context, 15*time.Second)
	} else {
		query.Context, cancel = context.WithCancel(query.Context)
	}
	return query, cancel, nil
}

func logUsesSkill(raw string) bool {
	var other map[string]interface{}
	if common.UnmarshalJsonStr(raw, &other) != nil {
		return false
	}
	gateway, _ := other["gateway_skills"].(map[string]interface{})
	stage, _ := gateway["stage"].(string)
	if stage == "discovery" {
		return false
	}
	templates, _ := other["prompt_templates"].([]interface{})
	for _, value := range templates {
		entry, _ := value.(map[string]interface{})
		if _, valid := entry["id"].(float64); valid {
			return true
		}
	}
	if stage != "generation" {
		return false
	}
	files, _ := gateway["files"].([]interface{})
	for _, value := range files {
		entry, _ := value.(map[string]interface{})
		_, validID := entry["template_id"].(float64)
		skillID, _ := entry["skill_id"].(string)
		if validID && skillID != "" {
			return true
		}
	}
	return false
}

// visitSkillLogs 只用通用 SQL 缩小候选；严格语义由 Go 解析，兼容损坏的历史 TEXT。
func visitSkillLogs(tx *gorm.DB, visit func(*Log) error) error {
	conditions := make([]string, 0, 3)
	values := make([]interface{}, 0, 3)
	// Unicode 转义可能出现在字段名称内，不能因不存在原始键名而漏掉有效日志。
	for _, candidate := range []string{`%"prompt_templates"%`, `%"gateway_skills"%`, `%\u%`} {
		condition, pattern, err := buildLogLikeCondition("logs.other", candidate)
		if err != nil {
			return err
		}
		conditions = append(conditions, condition)
		values = append(values, pattern)
	}
	tx = tx.Model(&Log{}).Where("("+strings.Join(conditions, " OR ")+")", values...)
	rows, err := tx.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := tx.Statement.Context.Err(); err != nil {
			return err
		}
		entry := new(Log)
		if err := tx.ScanRows(rows, entry); err != nil {
			return err
		}
		if logUsesSkill(entry.Other) {
			if err := visit(entry); err != nil {
				return err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return tx.Statement.Context.Err()
}

func getSkillFilteredLogs(tx *gorm.DB, order string, startIdx, num int) ([]*Log, int64, error) {
	logs := make([]*Log, 0)
	var total int64
	err := visitSkillLogs(tx.Order(order), func(log *Log) error {
		if total >= int64(startIdx) && len(logs) < num {
			logs = append(logs, log)
		}
		total++
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// addLogStatValue 保留统计整数精度，溢出时明确失败而不是返回回绕数据。
func addLogStatValue(total, value int) (int, error) {
	if (value > 0 && total > math.MaxInt-value) || (value < 0 && total < math.MinInt-value) {
		return 0, errors.New("log statistics overflow")
	}
	return total + value, nil
}

func sumSkillFilteredLogs(quotaQuery, rateQuery *gorm.DB) (Stat, error) {
	var stat Stat
	err := visitSkillLogs(quotaQuery.Select("logs.other, logs.quota"), func(log *Log) error {
		var err error
		stat.Quota, err = addLogStatValue(stat.Quota, log.Quota)
		return err
	})
	if err != nil {
		return Stat{}, err
	}
	err = visitSkillLogs(rateQuery.Select("logs.other, logs.prompt_tokens, logs.completion_tokens"), func(log *Log) error {
		var err error
		stat.Rpm, err = addLogStatValue(stat.Rpm, 1)
		if err != nil {
			return err
		}
		stat.Tpm, err = addLogStatValue(stat.Tpm, log.PromptTokens)
		if err != nil {
			return err
		}
		stat.Tpm, err = addLogStatValue(stat.Tpm, log.CompletionTokens)
		return err
	})
	if err != nil {
		return Stat{}, err
	}
	return stat, nil
}
