package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"gorm.io/gorm"
)

// AccessTokenFingerprint 标识 PAT 代际，不向审计暴露凭据；兼容 CHAR 尾部填充。
func AccessTokenFingerprint(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// RevokeUserAccessToken 锁定实际撤销的代际，只清除 PAT，不修改浏览器会话。
func RevokeUserAccessToken(userID int) (string, error) {
	if userID <= 0 {
		return "", gorm.ErrRecordNotFound
	}
	var fingerprint string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).First(&user, userID).Error; err != nil {
			return err
		}
		fingerprint = AccessTokenFingerprint(user.GetAccessToken())
		if user.AccessToken == nil {
			return nil
		}
		return tx.Model(&User{}).Where("id = ?", userID).Update("access_token", nil).Error
	})
	if err != nil {
		return "", err
	}
	return fingerprint, nil
}
