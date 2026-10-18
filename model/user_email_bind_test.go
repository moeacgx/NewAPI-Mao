package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindEmailPreservesConcurrentUserChanges(t *testing.T) {
	setupUserUpdateTestState(t)
	staleUser := createUserBindTestUser(t)
	// 模拟读取用户后，管理员及计费路径已提交了更新。
	require.NoError(t, DB.Model(&User{}).Where("id = ?", staleUser.Id).Updates(map[string]any{
		"username": "renamed-user", "display_name": "最新资料",
		"role": common.RoleAdminUser, "status": common.UserStatusDisabled,
		"group": "vip", "group_id": 321, "auth_version": 8,
		"github_id": "new-github-id", "password": "new-password-hash",
		"quota": 900, "used_quota": 100, "request_count": 4,
		"access_token": "rotated-test-token",
	}).Error)
	var expected User
	require.NoError(t, DB.First(&expected, staleUser.Id).Error)
	expected.Email = "account+bind@example.com"

	require.NoError(t, BindEmailToUser(&staleUser, " Account+Bind@Example.COM "))
	var stored User
	require.NoError(t, DB.First(&stored, staleUser.Id).Error)
	assert.Equal(t, expected, stored, "邮箱绑定只允许更新邮箱")
	assert.Equal(t, stored, staleUser, "调用方和缓存必须使用最新用户快照")
	var count int64
	require.NoError(t, DB.Unscoped().Model(&User{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestBindEmailRejectsMissingOrDeletedIdentity(t *testing.T) {
	for _, state := range []string{"零ID", "负ID", "不存在", "已删除"} {
		t.Run(state, func(t *testing.T) {
			setupUserUpdateTestState(t)
			existing := createUserBindTestUser(t)
			user := User{Id: existing.Id + 100}
			switch state {
			case "零ID":
				user.Id = 0
			case "负ID":
				user.Id = -1
			case "已删除":
				user = existing
				require.NoError(t, DB.Delete(&existing).Error)
			}
			require.Error(t, BindEmailToUser(&user, "missing@example.com"))
			var count int64
			require.NoError(t, DB.Unscoped().Model(&User{}).Count(&count).Error)
			assert.EqualValues(t, 1, count)
			var stored User
			require.NoError(t, DB.Unscoped().First(&stored, existing.Id).Error)
			assert.Empty(t, stored.Email)
		})
	}
}

func TestBindEmailRejectsOccupiedAddressWithoutChangingEitherAccount(t *testing.T) {
	setupUserUpdateTestState(t)
	user := createUserBindTestUser(t)
	user.Email = "original@example.com"
	require.NoError(t, DB.Model(&user).Update("email", user.Email).Error)
	owner := User{Username: "email-owner", Email: "Taken@Example.com", AffCode: "email-owner"}
	require.NoError(t, DB.Create(&owner).Error)
	require.ErrorIs(t, BindEmailToUser(&user, " TAKEN@example.COM "), ErrEmailAlreadyTaken)
	var stored User
	require.NoError(t, DB.First(&stored, user.Id).Error)
	assert.Equal(t, "original@example.com", stored.Email)
	require.NoError(t, DB.First(&owner, owner.Id).Error)
	assert.Equal(t, "Taken@Example.com", owner.Email)

	// 同一账号重复绑定自身地址仍然是原位更新。
	require.NoError(t, BindEmailToUser(&user, " ORIGINAL@EXAMPLE.COM "))
	assert.Equal(t, "original@example.com", user.Email)
}
