package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const selectedRouteOriginalBody = `{"model":"gpt-test","messages":[{"role":"user","content":"route secret"}]}`

func withSelectedRouteSensitiveRules(t *testing.T, rules ...setting.SensitiveRule) {
	t.Helper()
	previous := setting.GetSensitivePolicySnapshot()
	setting.ReplaceSensitivePolicySnapshot(setting.SensitivePolicySnapshot{
		CheckEnabled:         true,
		CheckOnPromptEnabled: true,
		Rules:                rules,
		RulesConfigured:      true,
	})
	t.Cleanup(func() { setting.ReplaceSensitivePolicySnapshot(previous) })
}

func newSelectedRouteRelayContext(t *testing.T) *gin.Context {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}))
	require.NoError(t, db.Create(&[]model.Group{
		{Id: 1, Code: "alpha", Name: "alpha", Status: model.GroupStatusActive},
		{Id: 2, Code: "beta", Name: "beta", Status: model.GroupStatusActive},
	}).Error)

	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = previousDB })
	previousGinMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousGinMode) })
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(selectedRouteOriginalBody))
	c.Request.Header.Set("Content-Type", "application/json")
	t.Cleanup(func() {
		middleware.ReleaseChannelConcurrencyForContext(c)
		middleware.ReleaseGroupUserConcurrencyForContext(c)
		if value, ok := c.Get(common.KeyBodyStorage); ok {
			if storage, ok := value.(common.BodyStorage); ok && storage != nil {
				require.NoError(t, storage.Close())
			}
		}
	})
	return c
}

func installSelectedRoute(t *testing.T, c *gin.Context, channel *model.Channel, group string) {
	t.Helper()
	common.SetContextKey(c, constant.ContextKeySelectedChannelGroup, group)
	require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "gpt-test"))
	setSelectedSecurityAuditRoute(c, channel, group)
}

func selectedRouteChannel(t *testing.T, id int, tag string, groups ...string) *model.Channel {
	t.Helper()
	channel := &model.Channel{
		Id: id, Type: constant.ChannelTypeOpenAI, Key: "sk-test", Name: tag,
		Status: common.ChannelStatusEnabled, Tag: common.GetPointer(tag),
	}
	for _, code := range groups {
		group, err := model.GetGroupByCodeOrAlias(code)
		require.NoError(t, err)
		channel.GroupDetails = append(channel.GroupDetails, model.GroupReference{Id: group.Id, Code: group.Code, Name: group.Name})
	}
	return channel
}

func serializedPreparedRequest(t *testing.T, request dto.Request) string {
	t.Helper()
	body, err := common.Marshal(request)
	require.NoError(t, err)
	return string(body)
}

func TestPrepareSelectedRouteRequestAppliesFirstAttemptChannelTagRule(t *testing.T) {
	withSelectedRouteSensitiveRules(t, setting.SensitiveRule{
		ID: "first-tag", Name: "first-tag", Enabled: true,
		Action: setting.SensitiveRuleActionBlock, Scope: setting.SensitiveRuleScopeRequest,
		TargetType: setting.SensitiveRuleTargetChannelTags, ChannelTags: []string{"protected"},
		Keywords: []string{"route secret"},
	})
	c := newSelectedRouteRelayContext(t)
	channel := selectedRouteChannel(t, 11, "protected", "alpha")
	installSelectedRoute(t, c, channel, "alpha")

	selected, ok := common.GetContextKeyType[*model.Channel](c, constant.ContextKeySelectedChannel)
	require.True(t, ok)
	require.Same(t, channel, selected)
	require.Equal(t, "protected", selected.GetTag())
	selectedByRelay, channelErr := getChannel(c, &relaycommon.RelayInfo{UsingGroup: "alpha"}, nil)
	require.Nil(t, channelErr)
	require.Same(t, channel, selectedByRelay)

	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, request)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodeSensitiveWordsDetected, apiErr.GetErrorCode())
}

func TestPrepareSelectedRouteRequestBlocksRetryInFinalSelectedGroup(t *testing.T) {
	withSelectedRouteSensitiveRules(t, setting.SensitiveRule{
		ID: "beta-only", Name: "beta-only", Enabled: true,
		Action: setting.SensitiveRuleActionBlock, Scope: setting.SensitiveRuleScopeRequest,
		TargetType: setting.SensitiveRuleTargetGroups, GroupCodes: []string{"beta"},
		Keywords: []string{"route secret"},
	})
	c := newSelectedRouteRelayContext(t)
	first := selectedRouteChannel(t, 21, "shared", "alpha", "beta")
	installSelectedRoute(t, c, first, "alpha")

	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, apiErr)
	require.Contains(t, serializedPreparedRequest(t, request), "route secret")

	retry := selectedRouteChannel(t, 22, "retry", "beta")
	installSelectedRoute(t, c, retry, "beta")
	request, apiErr = prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, request)
	require.NotNil(t, apiErr)
	require.Equal(t, types.ErrorCodeSensitiveWordsDetected, apiErr.GetErrorCode())
}

func TestPrepareSelectedRouteRequestRetryMaskReachesSerializedPayload(t *testing.T) {
	withSelectedRouteSensitiveRules(t, setting.SensitiveRule{
		ID: "retry-mask", Name: "retry-mask", Enabled: true,
		Action: setting.SensitiveRuleActionMask, Scope: setting.SensitiveRuleScopeRequest,
		TargetType: setting.SensitiveRuleTargetChannels, ChannelIds: []int{32},
		Keywords: []string{"route secret"}, Replacement: "[RETRY-MASK]",
	})
	c := newSelectedRouteRelayContext(t)
	installSelectedRoute(t, c, selectedRouteChannel(t, 31, "first", "alpha"), "alpha")

	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, apiErr)
	require.Contains(t, serializedPreparedRequest(t, request), "route secret")

	installSelectedRoute(t, c, selectedRouteChannel(t, 32, "retry", "beta"), "beta")
	request, apiErr = prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, apiErr)
	serialized := serializedPreparedRequest(t, request)
	require.Contains(t, serialized, "[RETRY-MASK]")
	require.NotContains(t, serialized, "route secret")
	storage, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	stored, err := storage.Bytes()
	require.NoError(t, err)
	require.Contains(t, string(stored), "[RETRY-MASK]")
}

func TestPrepareSelectedRouteRequestRetryDoesNotInheritFirstChannelMask(t *testing.T) {
	withSelectedRouteSensitiveRules(t, setting.SensitiveRule{
		ID: "first-mask", Name: "first-mask", Enabled: true,
		Action: setting.SensitiveRuleActionMask, Scope: setting.SensitiveRuleScopeRequest,
		TargetType: setting.SensitiveRuleTargetChannels, ChannelIds: []int{41},
		Keywords: []string{"route secret"}, Replacement: "[FIRST-MASK]",
	})
	c := newSelectedRouteRelayContext(t)
	installSelectedRoute(t, c, selectedRouteChannel(t, 41, "first", "alpha"), "alpha")

	request, apiErr := prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, apiErr)
	require.Contains(t, serializedPreparedRequest(t, request), "[FIRST-MASK]")

	installSelectedRoute(t, c, selectedRouteChannel(t, 42, "retry", "beta"), "beta")
	request, apiErr = prepareSelectedRouteRequest(c, types.RelayFormatOpenAI, []byte(selectedRouteOriginalBody))
	require.Nil(t, apiErr)
	serialized := serializedPreparedRequest(t, request)
	require.Contains(t, serialized, "route secret")
	require.NotContains(t, serialized, "[FIRST-MASK]")
}
