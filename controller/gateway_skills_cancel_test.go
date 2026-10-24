package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewaySkillsParentCancellationTakesPrecedenceOverPrivateDeadline(t *testing.T) {
	for _, privateDeadlineFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("private-deadline-first=%t", privateDeadlineFirst), func(t *testing.T) {
			t.Setenv("GATEWAY_SKILLS_DISCOVERY_TIMEOUT_SECONDS", "1")
			relayEngine, user, token, _ := setupGatewaySkillsRelay(t, "http://gateway-skills-cancel.test", false, 10000)
			common.RetryTimes = 1
			previousErrorLog := constant.ErrorLogEnabled
			constant.ErrorLogEnabled = true
			t.Cleanup(func() { constant.ErrorLogEnabled = previousErrorLog })
			client := service.GetHttpClient()
			previousTransport := client.Transport
			t.Cleanup(func() { client.Transport = previousTransport })

			for _, fullRelay := range []bool{false, true} {
				clientContext, cancelClient := context.WithCancel(context.Background())
				defer cancelClient()
				calls := 0
				client.Transport = gatewaySkillContextRoundTripper(func(request *http.Request) (*http.Response, error) {
					calls++
					_, err := io.Copy(io.Discard, request.Body)
					assert.NoError(t, err)
					assert.NoError(t, request.Body.Close())
					if !privateDeadlineFirst {
						cancelClient()
					}
					<-request.Context().Done()
					privateErr := request.Context().Err()
					if privateDeadlineFirst {
						assert.ErrorIs(t, privateErr, context.DeadlineExceeded)
						// 固定先后顺序，复现子期限触发后、HTTP 返回前父请求取消。
						cancelClient()
					}
					return nil, privateErr
				})
				engine := relayEngine
				if !fullRelay {
					engine = gin.New()
					engine.Use(middleware.BodyStorageCleanup(), middleware.TokenAuth(), middleware.PromptAudit(), middleware.Distribute())
					engine.POST("/v1/chat/completions", func(c *gin.Context) {
						request, err := helper.GetAndValidateRequest(c, types.RelayFormatOpenAI)
						require.NoError(t, err)
						info, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAI, request, nil)
						require.NoError(t, err)
						storage, err := common.GetBodyStorage(c)
						require.NoError(t, err)
						var admission *service.GatewaySkillAdmission
						defer func() { admission.Release() }()
						info, apiErr := discoverGatewaySkills(c, info, types.RelayFormatOpenAI, &admission, storage.Size())
						require.NotNil(t, apiErr)
						assert.ErrorIs(t, apiErr, context.Canceled)
						assert.NotErrorIs(t, apiErr, context.DeadlineExceeded)
						assert.True(t, types.IsSkipRetryError(apiErr))
						assert.Same(t, clientContext, c.Request.Context())
						assert.Equal(t, "request_context_canceled", requestContextErrorReason(c, apiErr))
						assert.False(t, shouldRetry(c, apiErr, 3))
						assert.False(t, service.ShouldDisableChannel(apiErr))
						processChannelError(c, *types.NewChannelError(1, constant.ChannelTypeOpenAI, "测试渠道", false, "", true), apiErr)
						if info.Billing != nil {
							info.Billing.Refund(c)
						}
						writeRelayErrorResponse(c, nil, types.RelayFormatOpenAI, apiErr, "cancelled-skill")
					})
				}
				request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(fmt.Sprintf(`{"model":%q,"messages":[{"role":"user","content":"review code"}]}`, canvasRouteBillingModel))).WithContext(clientContext)
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", "Bearer sk-"+token.Key)
				recorder := httptest.NewRecorder()
				engine.ServeHTTP(recorder, request)
				assert.Equal(t, 1, calls, "父取消后不得继续私有轮次或正式生成，fullRelay=%t", fullRelay)
				assert.Empty(t, recorder.Body.String(), "父取消不再写入技能 504 或渠道错误响应")
				require.Eventually(t, func() bool {
					var storedUser model.User
					var storedToken model.Token
					return model.DB.First(&storedUser, user.Id).Error == nil && model.DB.First(&storedToken, token.Id).Error == nil &&
						storedUser.Quota == 10000 && storedToken.RemainQuota == 10000
				}, 3*time.Second, 10*time.Millisecond)
			}
			var count int64
			require.NoError(t, model.LOG_DB.Model(&model.Log{}).Where("token_id = ? AND type IN ?", token.Id, []int{model.LogTypeError, model.LogTypeConsume}).Count(&count).Error)
			assert.Zero(t, count, "未结算父取消不能产生消费或渠道错误日志")
			var channel model.Channel
			require.NoError(t, model.DB.Where("name = ?", "canvas-route-upstream").First(&channel).Error)
			assert.Equal(t, common.ChannelStatusEnabled, channel.Status)
		})
	}
}
