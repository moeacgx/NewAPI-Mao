package model

import (
	"context"

	"gorm.io/gorm"
)

// UserIdentity 仅包含查询对象的公开标识，避免读取和返回用户私密字段。
type UserIdentity struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// GetUserIdentity 按稳定 ID 或精确用户名解析未删除的用户。
func GetUserIdentity(ctx context.Context, userID int, username string) (*UserIdentity, error) {
	var identity UserIdentity
	query := DB.WithContext(ctx).Model(&User{}).Select("id", "username")
	if userID > 0 {
		query = query.Where("id = ?", userID)
	} else {
		query = query.Where("username = ?", username)
	}
	if err := query.Take(&identity).Error; err != nil {
		return nil, err
	}
	// 数据库排序规则可能忽略大小写；展示查询仍要求用户名精确匹配。
	if userID <= 0 && identity.Username != username {
		return nil, gorm.ErrRecordNotFound
	}
	return &identity, nil
}
