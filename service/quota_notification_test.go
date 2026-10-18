package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func TestQuotaReminderDueOnlyOnThresholdCrossing(t *testing.T) {
	oldEnabled := common.QuotaRemindEnabled
	oldThreshold := common.QuotaRemindThreshold
	t.Cleanup(func() {
		common.QuotaRemindEnabled = oldEnabled
		common.QuotaRemindThreshold = oldThreshold
	})
	common.QuotaRemindEnabled = true
	common.QuotaRemindThreshold = 100

	setting := dto.UserSetting{}
	require.True(t, quotaReminderDue(setting, 120, 90))
	require.False(t, quotaReminderDue(setting, 90, 80))
	require.False(t, quotaReminderDue(setting, 80, 70))
	require.True(t, quotaReminderDue(setting, 150, 99))
}

func TestQuotaReminderDueHonorsSiteAndUserSwitches(t *testing.T) {
	oldEnabled := common.QuotaRemindEnabled
	oldThreshold := common.QuotaRemindThreshold
	t.Cleanup(func() {
		common.QuotaRemindEnabled = oldEnabled
		common.QuotaRemindThreshold = oldThreshold
	})
	common.QuotaRemindThreshold = 100
	setting := dto.UserSetting{}
	setting.QuotaWarningEnabled = quotaBoolPtr(false)
	require.False(t, quotaReminderDue(setting, 120, 90))
	setting.QuotaWarningEnabled = quotaBoolPtr(true)
	common.QuotaRemindEnabled = false
	require.False(t, quotaReminderDue(setting, 120, 90))
}

func quotaBoolPtr(value bool) *bool { return &value }
