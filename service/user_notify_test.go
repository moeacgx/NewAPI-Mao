package service

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelTestNotificationKeepsRootDefaultAndPersonalSwitch(t *testing.T) {
	setupNotificationServiceTestDB(t)
	relayInfo, requests, sent := setupQuotaNotificationTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.User{}))
	enabled, disabled := true, false
	for index, test := range []struct {
		role        int
		switchValue *bool
		status      int
	}{
		{common.RoleRootUser, nil, common.UserStatusEnabled},
		{common.RoleAdminUser, nil, common.UserStatusEnabled},
		{common.RoleAdminUser, &enabled, common.UserStatusEnabled},
		{common.RoleRootUser, &disabled, common.UserStatusEnabled},
		{common.RoleAdminUser, &enabled, common.UserStatusDisabled},
	} {
		setting := relayInfo.UserSetting
		setting.ChannelNotifyEnabled = test.switchValue
		setting.Language = "ja"
		encoded, err := common.Marshal(setting)
		require.NoError(t, err)
		require.NoError(t, model.DB.Create(&model.User{
			Id: relayInfo.UserId + index, Username: fmt.Sprintf("notify-admin-%d", index),
			AffCode: fmt.Sprintf("notify-admin-%d", index),
			Role:    test.role, Status: test.status, Setting: string(encoded),
		}).Error)
	}

	NotifyRootUser(dto.NotifyTypeChannelTest, "通道测试完成", "所有通道测试已完成")
	assert.EqualValues(t, 2, sent.Load())
	// 个人语言不会翻译调用方传入的正文。
	requireQuotaNotification(t, requests, "通道测试完成")
	requireQuotaNotification(t, requests, "通道测试完成")
}
