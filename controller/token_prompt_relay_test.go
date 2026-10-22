package controller

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

func setupPromptRelayDB(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousPolicy := setting.GetSensitivePolicySnapshot()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	setting.ReplaceSensitivePolicySnapshot(setting.SensitivePolicySnapshot{})
	t.Cleanup(func() {
		model.DB = previousDB
		setting.ReplaceSensitivePolicySnapshot(previousPolicy)
		sqlDB, e := db.DB()
		require.NoError(t, e)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.Token{}, &model.Group{}, &model.GroupAlias{}, &model.PromptCategory{}, &model.PromptTemplate{}, &model.PromptTemplateGroup{}, &model.TokenPromptBinding{}))
	require.NoError(t, db.Create(&model.Group{Id: 71, Code: "allowed-code", Name: "高级模型组", Status: 1}).Error)
	require.NoError(t, db.Create(&model.Group{Id: 72, Code: "other-code", Name: "普通模型组", Status: 1}).Error)
	require.NoError(t, db.Create(&model.Token{Id: 31, UserId: 11, Key: "prompt-relay-test", Status: common.TokenStatusEnabled}).Error)
	require.NoError(t, db.Create(&model.PromptTemplate{ID: 41, Title: "写作技能", Content: "installed special instruction", Enabled: true, Version: 2}).Error)
	require.NoError(t, db.Create(&model.PromptTemplateGroup{TemplateID: 41, GroupID: 71}).Error)
	require.NoError(t, db.Create(&model.TokenPromptBinding{TokenID: 31, TemplateID: 41, Enabled: true}).Error)
}

func promptRelayContext(t *testing.T, path, body string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyTokenId, 31)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "prompt-test")
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "allowed-code")
	t.Cleanup(func() {
		if stored, ok := c.Get(common.KeyBodyStorage); ok {
			require.NoError(t, stored.(common.BodyStorage).Close())
		}
	})
	return c
}

func TestTokenPromptSelectedGroupRetryRemovesPreviousInstructions(t *testing.T) {
	setupPromptRelayDB(t)
	body := `{"model":"prompt-test","messages":[{"role":"user","content":"question"}]}`
	c := promptRelayContext(t, "/v1/chat/completions", body)
	for _, group := range []string{"allowed-code", "allowed-code", "other-code", "allowed-code"} {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, group)
		request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
		require.Nil(t, apiErr)
		encoded, err := common.Marshal(request)
		require.NoError(t, err)
		expected := 0
		if group == "allowed-code" {
			expected = 1
		}
		assert.Equal(t, expected, strings.Count(string(encoded), "installed special instruction"))
		storage, err := common.GetBodyStorage(c)
		require.NoError(t, err)
		stored, err := storage.Bytes()
		require.NoError(t, err)
		assert.Equal(t, expected, strings.Count(string(stored), "installed special instruction"))
		other := map[string]interface{}{}
		service.AppendTokenPromptLog(c, other)
		if expected == 0 {
			assert.NotContains(t, other, "prompt_templates")
		} else {
			audit, err := common.Marshal(other)
			require.NoError(t, err)
			assert.NotContains(t, string(audit), "installed special instruction")
			assert.Contains(t, string(audit), `"version":2`)
		}
	}
}

func TestTokenPromptSelectedContentPassesThroughSensitiveFilter(t *testing.T) {
	setupPromptRelayDB(t)
	withSelectedRouteSensitiveRules(t, setting.SensitiveRule{ID: "prompt-block", Name: "prompt-block", Enabled: true, Action: setting.SensitiveRuleActionBlock, Scope: setting.SensitiveRuleScopeRequest, TargetType: setting.SensitiveRuleTargetAll, Keywords: []string{"installed special instruction"}})
	body := `{"model":"prompt-test","messages":[{"role":"user","content":"question"}]}`
	c := promptRelayContext(t, "/v1/chat/completions", body)
	_, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeSensitiveWordsDetected, apiErr.GetErrorCode())
}

func TestTokenPromptReachesActualUpstreamForEveryProtocolAndPassthrough(t *testing.T) {
	setupPromptRelayDB(t)
	tests := []struct {
		name, path, body, field string
		format                  types.RelayFormat
		channelType             int
		handler                 func(*gin.Context, *relaycommon.RelayInfo) *types.NewAPIError
	}{
		{"chat", "/v1/chat/completions", `{"model":"prompt-test","messages":[{"role":"user","content":"question"}],"temperature":0,"vendor":{"flag":false}}`, "messages.0.content", types.RelayFormatOpenAI, constant.ChannelTypeOpenAI, relay.TextHelper},
		{"responses", "/v1/responses", `{"model":"prompt-test","input":"question","temperature":0,"vendor":{"flag":false}}`, "instructions", types.RelayFormatOpenAIResponses, constant.ChannelTypeOpenAI, relay.ResponsesHelper},
		{"claude", "/v1/messages", `{"model":"prompt-test","messages":[{"role":"user","content":"question"}],"max_tokens":32,"temperature":0,"vendor":{"flag":false}}`, "system.0.text", types.RelayFormatClaude, constant.ChannelTypeAnthropic, relay.ClaudeHelper},
		{"gemini", "/v1beta/models/prompt-test:generateContent", `{"contents":[{"role":"user","parts":[{"text":"question"}]}],"generationConfig":{"temperature":0},"vendor":{"flag":false}}`, "systemInstruction.parts.0.text", types.RelayFormatGemini, constant.ChannelTypeGemini, relay.GeminiHelper},
	}
	for _, tt := range tests {
		for _, passthrough := range []bool{false, true} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/passthrough=%t/stream=%t", tt.name, passthrough, stream), func(t *testing.T) {
					body, path := tt.body, tt.path
					if stream && tt.format != types.RelayFormatGemini {
						body = strings.TrimSuffix(body, "}") + `,"stream":true}`
					}
					if stream && tt.format == types.RelayFormatGemini {
						path = strings.Replace(path, ":generateContent", ":streamGenerateContent", 1) + "?alt=sse"
					}
					outbound := make(chan []byte, 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						payload, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						outbound <- payload
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusBadRequest)
						_, _ = io.WriteString(w, `{"error":{"message":"test rejection","type":"invalid_request_error"}}`)
					}))
					defer server.Close()
					c := promptRelayContext(t, path, body)
					common.SetContextKey(c, constant.ContextKeyChannelType, tt.channelType)
					common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, server.URL)
					common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
					common.SetContextKey(c, constant.ContextKeyChannelSetting, dto.ChannelSettings{PassThroughBodyEnabled: passthrough})
					request, apiErr := prepareSelectedRouteRequest(c, tt.format, []byte(body))
					require.Nil(t, apiErr)
					info, err := relaycommon.GenRelayInfo(c, tt.format, request, nil)
					require.NoError(t, err)
					apiErr = tt.handler(c, info)
					require.NotNil(t, apiErr)
					require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
					payload := <-outbound
					assert.Equal(t, "installed special instruction", gjson.GetBytes(payload, tt.field).String())
					if passthrough {
						assert.Equal(t, "false", gjson.GetBytes(payload, "vendor.flag").Raw)
					}
				})
			}
		}
	}
}

type promptBillingReservation struct{ reserved int }

func (b *promptBillingReservation) ReserveRequestQuota(n int) error { return b.Reserve(n) }

func (b *promptBillingReservation) Settle(int) error         { return nil }
func (b *promptBillingReservation) Refund(*gin.Context)      {}
func (b *promptBillingReservation) NeedsRefund() bool        { return false }
func (b *promptBillingReservation) GetPreConsumedQuota() int { return b.reserved }
func (b *promptBillingReservation) Reserve(n int) error {
	if n > b.reserved {
		b.reserved = n
	}
	return nil
}

func TestTokenPromptBillingIncludesLoadedInputAndRestoresAfterGroupChange(t *testing.T) {
	setupPromptRelayDB(t)
	oldCount := constant.CountToken
	oldRatios := ratio_setting.ModelRatio2JSONString()
	oldGroups := ratio_setting.GroupRatio2JSONString()
	t.Cleanup(func() {
		constant.CountToken = oldCount
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatios))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroups))
	})
	constant.CountToken = true
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"prompt-test":1}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"allowed-code":1,"other-code":1}`))
	body := `{"model":"prompt-test","messages":[{"role":"user","content":"question"}]}`
	c := promptRelayContext(t, "/v1/chat/completions", body)
	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
	require.Nil(t, apiErr)
	reservation := &promptBillingReservation{}
	info := &relaycommon.RelayInfo{Request: request, OriginModelName: "prompt-test", UsingGroup: "allowed-code", UserGroup: "allowed-code", RelayFormat: types.RelayFormatOpenAI, Billing: reservation}
	require.Nil(t, prepareTokenPromptBilling(c, info))
	loaded := info.GetEstimatePromptTokens()
	assert.Positive(t, loaded)
	assert.Positive(t, reservation.reserved)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "other-code")
	info.UsingGroup = "other-code"
	request, apiErr = prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
	require.Nil(t, apiErr)
	info.Request = request
	require.Nil(t, prepareTokenPromptBilling(c, info))
	assert.Less(t, info.GetEstimatePromptTokens(), loaded)
}

func TestTokenPromptTieredBillingKeepsExpressionAndRefreshesLoadedRequest(t *testing.T) {
	setupPromptRelayDB(t)
	oldCount := constant.CountToken
	constant.CountToken = false
	t.Cleanup(func() { constant.CountToken = oldCount })
	body := `{"model":"prompt-test","messages":[{"role":"user","content":"question"}]}`
	c := promptRelayContext(t, "/v1/chat/completions", body)
	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
	require.Nil(t, apiErr)
	const expr = `tier("frozen", p * 2)`
	info := &relaycommon.RelayInfo{
		Request: request, OriginModelName: "prompt-test", RelayFormat: types.RelayFormatOpenAI,
		Billing:               &promptBillingReservation{},
		PriceData:             hosttypes.PriceData{GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1}},
		TieredBillingSnapshot: &billingexpr.BillingSnapshot{BillingMode: "tiered_expr", ExprString: expr, QuotaPerUnit: 500000, GroupRatio: 1},
		BillingRequestInput:   &billingexpr.RequestInput{Body: []byte(body)},
	}
	require.Nil(t, prepareTokenPromptBilling(c, info))
	assert.Positive(t, info.GetEstimatePromptTokens())
	assert.Equal(t, info.GetEstimatePromptTokens(), info.TieredBillingSnapshot.EstimatedQuotaAfterGroup)
	assert.Equal(t, expr, info.TieredBillingSnapshot.ExprString)
	assert.Contains(t, string(info.BillingRequestInput.Body), "installed special instruction")
	assert.Equal(t, info.GetEstimatePromptTokens(), info.FinalPreConsumedQuota)
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "other-code")
	request, apiErr = prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(body))
	require.Nil(t, apiErr)
	info.Request = request
	require.Nil(t, prepareTokenPromptBilling(c, info))
	assert.NotContains(t, string(info.BillingRequestInput.Body), "installed special instruction")
}
