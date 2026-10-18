package service

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupQuotaNotificationTest(t *testing.T) (*relaycommon.RelayInfo, <-chan string, *atomic.Int32) {
	t.Helper()
	oldEnabled, oldThreshold := common.QuotaRemindEnabled, common.QuotaRemindThreshold
	oldRedis, oldLimit := common.RedisEnabled, constant.NotifyLimitCount
	oldDuration := constant.NotificationLimitDurationMinute
	common.QuotaRemindEnabled, common.QuotaRemindThreshold = true, 100
	common.RedisEnabled, constant.NotifyLimitCount = false, 1
	constant.NotificationLimitDurationMinute = 60
	// 每个 fixture 独立清理额度提醒计数，不依赖上一用例的频控状态。
	const userID = 98101
	clearLimit := func() {
		notifyLimitStore.Range(func(key, _ interface{}) bool {
			if strings.HasPrefix(key.(string), "9810") {
				notifyLimitStore.Delete(key)
			}
			return true
		})
	}
	clearLimit()
	t.Cleanup(func() {
		clearLimit()
		common.QuotaRemindEnabled, common.QuotaRemindThreshold = oldEnabled, oldThreshold
		common.RedisEnabled, constant.NotifyLimitCount = oldRedis, oldLimit
		constant.NotificationLimitDurationMinute = oldDuration
	})
	requests := make(chan string, 4)
	var sent atomic.Int32
	server := useTelegramTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Title string `json:"title"`
		}
		if err := common.DecodeJson(r.Body, &payload); err != nil {
			t.Errorf("通知请求无效: %v", err)
		}
		sent.Add(1)
		requests <- payload.Title
		w.WriteHeader(http.StatusOK)
	}))
	relayInfo := &relaycommon.RelayInfo{
		UserId: userID, UserQuota: 300,
		UserSetting: dto.UserSetting{NotifyType: dto.NotifyTypeGotify, GotifyUrl: server.URL, GotifyToken: "test-token"},
	}
	return relayInfo, requests, &sent
}

func requireQuotaNotification(t *testing.T, requests <-chan string, title string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case got := <-requests:
		assert.Equal(t, title, got)
	case <-ctx.Done():
		t.Fatal("未收到额度提醒")
	}
}

func TestWalletQuotaReminderRecoversAfterStaleConcurrentSnapshots(t *testing.T) {
	setupNotificationServiceTestDB(t)
	relayInfo, requests, sent := setupQuotaNotificationTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.User{}))
	user := &model.User{Id: relayInfo.UserId, Username: "quota-reminder", Quota: 300}
	require.NoError(t, model.DB.Create(user).Error)

	// 两个请求都读到 300，各自消费 120 时单独估算仍有 180。
	// 合并扣减可在检查之后才落库，等价于通知看不到本批待写增量。
	checkAndSendQuotaNotify(relayInfo, 120, 0)
	checkAndSendQuotaNotify(relayInfo, 120, 0)
	assert.Zero(t, sent.Load())
	require.NoError(t, model.DB.Model(user).Update("quota", 60).Error)
	current, err := model.GetUserQuota(user.Id, true)
	require.NoError(t, err)
	relayInfo.UserQuota = current
	checkAndSendQuotaNotify(relayInfo, 10, 0)
	requireQuotaNotification(t, requests, "您的额度即将用尽")

	// 持续低余额仍是候选，但实际发送沿用 NotifyUser 的频控。
	notifyUserQuotaWarning(relayInfo, 40, "您的额度即将用尽")
	assert.EqualValues(t, 1, sent.Load())
	assert.False(t, quotaReminderDue(relayInfo.UserSetting, 150), "充值高于阈值后停止候选")
	assert.True(t, quotaReminderDue(relayInfo.UserSetting, 90), "后续再次低于阈值仍可提醒")
}

func TestQuotaReminderSwitchesAndThreshold(t *testing.T) {
	relayInfo, _, sent := setupQuotaNotificationTest(t)
	relayInfo.UserQuota = 50
	common.QuotaRemindEnabled = false
	checkAndSendQuotaNotify(relayInfo, 10, 0)
	common.QuotaRemindEnabled = true
	disabled := false
	relayInfo.UserSetting.QuotaWarningEnabled = &disabled
	checkAndSendQuotaNotify(relayInfo, 10, 0)
	assert.Zero(t, sent.Load())
	assert.False(t, quotaReminderDue(dto.UserSetting{}, 100), "等于阈值不提醒")
	assert.True(t, quotaReminderDue(dto.UserSetting{}, 99))
	assert.False(t, quotaReminderDue(dto.UserSetting{QuotaWarningThreshold: 20}, 30))
	assert.True(t, quotaReminderDue(dto.UserSetting{QuotaWarningThreshold: 20}, 19))
}

func TestSubscriptionQuotaReminderWhileAlreadyBelowThreshold(t *testing.T) {
	relayInfo, requests, sent := setupQuotaNotificationTest(t)
	relayInfo.SubscriptionId = 5
	relayInfo.SubscriptionAmountTotal = 1000
	relayInfo.SubscriptionAmountUsedAfterPreConsume = 950
	relayInfo.SubscriptionPreConsumed = 10
	relayInfo.SubscriptionPostDelta = 5

	// 本请求开始前剩余 60，结算后为 45，不能因已错过跨线而永久静默。
	checkAndSendSubscriptionQuotaNotify(relayInfo)
	requireQuotaNotification(t, requests, "您的订阅额度即将用尽")
	notifyUserQuotaWarning(relayInfo, 40, "您的订阅额度即将用尽")
	assert.EqualValues(t, 1, sent.Load())
}

func TestSubscriptionQuotaReminderHonorsSwitchesAndUnlimitedPlans(t *testing.T) {
	relayInfo, _, sent := setupQuotaNotificationTest(t)
	relayInfo.SubscriptionId = 5
	relayInfo.SubscriptionAmountTotal = 1000
	relayInfo.SubscriptionAmountUsedAfterPreConsume = 950
	common.QuotaRemindEnabled = false
	checkAndSendSubscriptionQuotaNotify(relayInfo)
	common.QuotaRemindEnabled = true
	disabled := false
	relayInfo.UserSetting.QuotaWarningEnabled = &disabled
	checkAndSendSubscriptionQuotaNotify(relayInfo)
	relayInfo.UserSetting.QuotaWarningEnabled = nil
	relayInfo.SubscriptionAmountTotal = 0
	checkAndSendSubscriptionQuotaNotify(relayInfo)
	assert.Zero(t, sent.Load())
}
