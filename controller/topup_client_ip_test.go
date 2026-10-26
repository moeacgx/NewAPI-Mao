package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEpayOrderLogPreservesClientIPFromTrustedRequest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		trustedProxies string
		peer           string
		forwardedFor   string
		wantRequestIP  string
		manual         bool
		recordIP       bool
		forceRecordIP  bool
	}{
		{
			name: "可信代理取转发链首个非可信地址", trustedProxies: "192.0.2.0/24",
			peer: "192.0.2.10:12345", forwardedFor: "198.51.100.77, 203.0.113.42",
			wantRequestIP: "203.0.113.42", recordIP: true,
		},
		{
			name: "非可信来源忽略伪造转发头和CF头", trustedProxies: "192.0.2.0/24",
			peer: "198.51.100.30:12345", forwardedFor: "203.0.113.42",
			wantRequestIP: "198.51.100.30", recordIP: true,
		},
		{
			name: "严格直连忽略转发头", trustedProxies: "none",
			peer: "192.0.2.10:12345", forwardedFor: "203.0.113.42",
			wantRequestIP: "192.0.2.10", recordIP: true,
		},
		{
			name: "管理员补单保留下单地址", trustedProxies: "192.0.2.0/24",
			peer: "192.0.2.10:12345", forwardedFor: "203.0.113.42",
			wantRequestIP: "203.0.113.42", recordIP: true, manual: true,
		},
		{
			name: "用户关闭记录时订单地址不进入日志", trustedProxies: "192.0.2.0/24",
			peer: "192.0.2.10:12345", forwardedFor: "203.0.113.42",
			wantRequestIP: "203.0.113.42",
		},
		{
			name: "全局强制记录沿用消费日志开关语义", trustedProxies: "192.0.2.0/24",
			peer: "192.0.2.10:12345", forwardedFor: "203.0.113.42",
			wantRequestIP: "203.0.113.42", forceRecordIP: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initModelListColumnNames(t)
			db := setupSubscriptionPaymentControllerTestDB(t)
			withConfirmedPaymentCompliance(t)
			oldPrice, oldMinTopUp := operation_setting.Price, operation_setting.MinTopUp
			oldPayAddress, oldEpayID, oldEpayKey := operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey
			oldPayMethods := operation_setting.PayMethods
			oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
			oldDiscounts := operation_setting.GetPaymentSetting().AmountDiscount
			oldForceRecordIP, oldLogConsume, oldExport := common.ForceRecordLogIpEnabled, common.LogConsumeEnabled, common.DataExportEnabled
			t.Cleanup(func() {
				operation_setting.Price, operation_setting.MinTopUp = oldPrice, oldMinTopUp
				operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey = oldPayAddress, oldEpayID, oldEpayKey
				operation_setting.PayMethods = oldPayMethods
				operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
				operation_setting.GetPaymentSetting().AmountDiscount = oldDiscounts
				common.ForceRecordLogIpEnabled, common.LogConsumeEnabled, common.DataExportEnabled = oldForceRecordIP, oldLogConsume, oldExport
			})
			operation_setting.Price, operation_setting.MinTopUp = 1, 1
			// Purchase 只生成支付链接，不发送请求；使用保留域名避免接触真实支付网关。
			operation_setting.PayAddress, operation_setting.EpayId, operation_setting.EpayKey = "https://pay.example.invalid", "test-merchant", "test-signing-key"
			operation_setting.PayMethods = []map[string]string{{"type": "alipay", "name": "支付宝"}}
			operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
			operation_setting.GetPaymentSetting().AmountDiscount = map[int]float64{}
			common.ForceRecordLogIpEnabled, common.LogConsumeEnabled, common.DataExportEnabled = tc.forceRecordIP, true, false
			user := &model.User{
				Id: 945, Username: "topup-client-ip", Status: common.UserStatusEnabled, Group: "default", Quota: 7,
				Setting: common.MapToJsonStr(map[string]interface{}{"record_ip_log": tc.recordIP}),
			}
			require.NoError(t, db.Create(user).Error)

			t.Setenv("TRUSTED_PROXIES", tc.trustedProxies)
			router := gin.New()
			require.NoError(t, middleware.ConfigureTrustedProxies(router))
			router.Use(func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set("username", user.Username)
				c.Next()
			})
			router.POST("/api/user/epay/pay", RequestEpay)
			router.POST("/api/user/epay/notify", EpayNotify)
			router.POST("/api/user/topup/complete", AdminCompleteTopUp)
			// 本地调用实际日志入口以比较来源，不执行模型转发或扣费。
			router.POST("/consume-log", func(c *gin.Context) {
				model.RecordConsumeLog(c, user.Id, model.RecordConsumeLogParams{ModelName: "client-ip-proof"})
				c.Status(http.StatusNoContent)
			})

			payload, err := common.Marshal(EpayRequest{Amount: 2, PaymentMethod: "alipay"})
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodPost, "/api/user/epay/pay", bytes.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("X-Forwarded-For", tc.forwardedFor)
			request.Header.Set("CF-Connecting-IP", "198.51.100.222")
			request.RemoteAddr = tc.peer
			created := httptest.NewRecorder()
			router.ServeHTTP(created, request)
			require.Equal(t, http.StatusOK, created.Code)
			var response struct {
				Message string            `json:"message"`
				URL     string            `json:"url"`
				Data    map[string]string `json:"data"`
			}
			require.NoError(t, common.Unmarshal(created.Body.Bytes(), &response))
			require.Equal(t, "success", response.Message, created.Body.String())
			require.Equal(t, "https://pay.example.invalid/submit.php", response.URL)
			require.NotEmpty(t, response.Data["money"])

			var order model.TopUp
			require.NoError(t, db.Where("user_id = ?", user.Id).First(&order).Error)
			assert.Equal(t, tc.wantRequestIP, order.RequestIP)
			require.Equal(t, common.TopUpStatusPending, order.Status)

			consumeRequest := httptest.NewRequest(http.MethodPost, "/consume-log", nil)
			consumeRequest.RemoteAddr = tc.peer
			consumeRequest.Header = request.Header.Clone()
			consumed := httptest.NewRecorder()
			router.ServeHTTP(consumed, consumeRequest)
			require.Equal(t, http.StatusNoContent, consumed.Code)
			var consumeLog model.Log
			require.NoError(t, db.Where("user_id = ? AND type = ?", user.Id, model.LogTypeConsume).First(&consumeLog).Error)

			for attempt := 0; attempt < 2; attempt++ {
				var completion *http.Request
				if tc.manual {
					body, err := common.Marshal(AdminCompleteTopupRequest{TradeNo: order.TradeNo})
					require.NoError(t, err)
					completion = httptest.NewRequest(http.MethodPost, "/api/user/topup/complete", bytes.NewReader(body))
					completion.Header.Set("Content-Type", "application/json")
				} else {
					params := epay.GenerateParams(map[string]string{
						"pid": operation_setting.EpayId, "out_trade_no": order.TradeNo,
						"trade_no": "test-provider-order", "type": "alipay", "money": response.Data["money"],
						"trade_status": epay.StatusTradeSuccess,
					}, operation_setting.EpayKey)
					form := url.Values{}
					for key, value := range params {
						form.Set(key, value)
					}
					completion = httptest.NewRequest(http.MethodPost, "/api/user/epay/notify", strings.NewReader(form.Encode()))
					completion.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				// 支付服务与补单请求来自不同地址，均不得代替原始下单地址。
				completion.RemoteAddr = "198.51.100.90:23456"
				completion.Header.Set("X-Forwarded-For", "198.51.100.91")
				completion.Header.Set("CF-Connecting-IP", "198.51.100.92")
				completed := httptest.NewRecorder()
				router.ServeHTTP(completed, completion)
				require.Equal(t, http.StatusOK, completed.Code)
				if tc.manual {
					assert.Contains(t, completed.Body.String(), `"success":true`)
				} else {
					assert.Equal(t, "success", completed.Body.String())
				}
			}

			var savedOrder model.TopUp
			require.NoError(t, db.First(&savedOrder, order.Id).Error)
			assert.Equal(t, common.TopUpStatusSuccess, savedOrder.Status)
			assert.Equal(t, tc.wantRequestIP, savedOrder.RequestIP)
			var topupLogs []model.Log
			require.NoError(t, db.Where("user_id = ? AND type = ?", user.Id, model.LogTypeTopup).Find(&topupLogs).Error)
			require.Len(t, topupLogs, 1)
			wantLogIP := ""
			if tc.recordIP || tc.forceRecordIP {
				wantLogIP = tc.wantRequestIP
			}
			assert.Equal(t, wantLogIP, consumeLog.Ip)
			assert.Equal(t, consumeLog.Ip, topupLogs[0].Ip)
			var savedUser model.User
			require.NoError(t, db.First(&savedUser, user.Id).Error)
			assert.Equal(t, int64(7)+order.CreditedQuota, savedUser.Quota)
		})
	}
}
