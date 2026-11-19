package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"gorm.io/gorm/clause"
)

// MCPKey 与模型推理令牌、后台登录凭证完全独立，摘要永不对外序列化。
type MCPKey struct {
	ID         int      `json:"id" gorm:"primaryKey"`
	UserID     int      `json:"-" gorm:"index;not null"`
	Name       string   `json:"name" gorm:"size:100;not null"`
	KeyHash    string   `json:"-" gorm:"size:64;uniqueIndex;not null"`
	KeyPrefix  string   `json:"key_prefix" gorm:"size:20"`
	Tools      []string `json:"tools" gorm:"serializer:json;type:text"`
	CreatedAt  int64    `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt  int64    `json:"expires_at"`
	LastUsedAt int64    `json:"last_used_at"`
	RevokedAt  int64    `json:"revoked_at"`
}

// MCPCallLog 仅存元数据，禁止加入请求参数、凭据、错误原文或响应。
type MCPCallLog struct {
	ID         int64  `json:"id" gorm:"primaryKey"`
	UserID     int    `json:"-" gorm:"index;not null"`
	KeyID      int    `json:"key_id" gorm:"index;not null"`
	ToolName   string `json:"tool_name" gorm:"size:100"`
	RequestID  string `json:"request_id" gorm:"size:128"`
	Success    bool   `json:"success"`
	DurationMS int64  `json:"duration_ms"`
	CreatedAt  int64  `json:"created_at" gorm:"autoCreateTime"`
}

// MCPWriteRequest 的唯一约束跨实例仲裁执行权。pending 永不自动重试或过期回收，
// 进程崩溃或结果保存失败后必须人工核对业务状态，避免重复产生副作用。
type MCPWriteRequest struct {
	ID          int64  `gorm:"primaryKey"`
	KeyID       int    `gorm:"uniqueIndex:idx_mcp_write_identity,priority:1;not null"`
	RequestID   string `gorm:"size:128;uniqueIndex:idx_mcp_write_identity,priority:2;not null"`
	RequestHash string `gorm:"size:64;not null"`
	OwnerNonce  string `gorm:"size:64;not null"`
	State       string `gorm:"size:16;not null"`
	Result      string `gorm:"type:text"`
	Success     bool
	CreatedAt   int64 `gorm:"autoCreateTime"`
}

func ListMCPKeys(userID, offset, limit int) ([]MCPKey, int64, error) {
	items := make([]MCPKey, 0)
	var total int64
	query := DB.Model(&MCPKey{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func RevokeMCPKey(userID, keyID int) error {
	result := DB.Model(&MCPKey{}).Where("id = ? AND user_id = ? AND revoked_at = 0", keyID, userID).
		Update("revoked_at", time.Now().Unix())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var key MCPKey
		return DB.Select("id").Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error
	}
	return nil
}

func ListMCPLogs(userID, offset, limit int) ([]MCPCallLog, int64, error) {
	items := make([]MCPCallLog, 0)
	var total int64
	query := DB.Model(&MCPCallLog{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func ClaimMCPWriteRequest(keyID int, requestID, hash string) (*MCPWriteRequest, bool, error) {
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return nil, false, err
	}
	nonce := hex.EncodeToString(entropy[:])
	record := MCPWriteRequest{KeyID: keyID, RequestID: requestID, RequestHash: hash, OwnerNonce: nonce, State: "pending"}
	result := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&record)
	if result.Error != nil {
		return nil, false, result.Error
	}
	// MySQL clientFoundRows 可把冲突 no-op 更新计为一行，必须回读 nonce 判断所有权。
	var stored MCPWriteRequest
	err := DB.Where("key_id = ? AND request_id = ?", keyID, requestID).First(&stored).Error
	return &stored, err == nil && stored.OwnerNonce == nonce, err
}

func CompleteMCPWriteRequest(id int64, result string, success bool) error {
	// TEXT 在三种数据库均可用，控制上限以兼容 MySQL 的 TEXT 容量。
	if len(result) > 48*1024 {
		return errors.New("MCP result exceeds storage limit")
	}
	updated := DB.Model(&MCPWriteRequest{}).Where("id = ? AND state = ?", id, "pending").
		Updates(map[string]any{"state": "done", "result": result, "success": success})
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return errors.New("MCP request state changed")
	}
	return nil
}
