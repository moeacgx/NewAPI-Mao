package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupContextForTokenGroupChannelsUsesCurrentGroupCode(t *testing.T) {
	setupDashboardAuthMiddlewareTest(t)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}, &model.Group{}, &model.GroupAlias{}, &model.TokenGroupBinding{}))
	require.NoError(t, model.DB.Create(&model.Group{Code: "default", Name: "默认显示名称", Status: model.GroupStatusActive, Ratio: 1}).Error)
	user := createMiddlewarePATUser(t, "group-channel-admin", "unused")
	require.NoError(t, model.DB.Model(user).Update("role", common.RoleAdminUser).Error)
	group := &model.Group{Code: "original-code", Name: "测试显示名称", Status: model.GroupStatusActive, Ratio: 1}
	require.NoError(t, model.DB.Create(group).Error)
	token := &model.Token{UserId: user.Id, Key: "group-channel-context", GroupMode: model.TokenGroupModeExplicit, GroupIds: []int{group.Id}, GroupChannelIds: map[int]int{group.Id: 951}}
	require.NoError(t, token.Insert())
	require.NoError(t, model.DB.Model(group).Update("code", "current-code").Error)
	loaded, err := model.GetTokenById(token.Id)
	require.NoError(t, err)
	ctx := newSelectedChannelContext()
	require.NoError(t, SetupContextForToken(ctx, loaded))
	assert.Equal(t, 951, service.RequestGroupChannelID(ctx, "current-code"))
	assert.Zero(t, service.RequestGroupChannelID(ctx, "original-code"))
}

func TestSetupContextForTokenGroupChannelsRejectsDowngradeAndChannelSuffix(t *testing.T) {
	for _, test := range []struct {
		name   string
		role   int
		parts  []string
		status int
	}{
		{name: "降权后拒绝使用", role: common.RoleCommonUser, status: http.StatusForbidden},
		{name: "管理员不能叠加密钥渠道后缀", role: common.RoleAdminUser, parts: []string{"key", "952"}, status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			setupDashboardAuthMiddlewareTest(t)
			user := createMiddlewarePATUser(t, "group-channel-rejected", "unused")
			require.NoError(t, model.DB.Model(user).Update("role", test.role).Error)
			token := &model.Token{UserId: user.Id, GroupChannelIds: map[int]int{1: 951}, GroupDetails: []model.GroupReference{{Id: 1, Code: "test-code", Name: "测试显示名称"}}}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			require.Error(t, SetupContextForToken(ctx, token, test.parts...))
			assert.Equal(t, test.status, recorder.Code)
			assert.True(t, ctx.IsAborted())
			assert.Zero(t, service.RequestGroupChannelID(ctx, "test-code"))
		})
	}
}

func TestDistributeGroupChannelRestrictionOverridesOldAffinity(t *testing.T) {
	oldChannel := setupUpstreamModelGuardMiddlewareTest(t)
	target := &model.Channel{Id: 991002, Name: "限定渠道", Key: "local-only-key", Type: constant.ChannelTypeOpenAI, Models: "client-model", Group: "guard-route", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(target).Error)
	require.NoError(t, target.AddAbilities(nil))
	model.InitChannelCache()
	settings := operation_setting.GetChannelAffinitySetting()
	original := *settings
	*settings = operation_setting.ChannelAffinitySetting{Enabled: true, DefaultTTLSeconds: 60, Rules: []operation_setting.ChannelAffinityRule{{Name: "token-group-channel-test", ModelRegex: []string{"^client-model$"}, KeySources: []operation_setting.ChannelAffinityKeySource{{Type: "request_header", Key: "X-Test-Affinity"}}, BindMultiKeyIndex: true, IncludeRuleName: true}}}
	t.Cleanup(func() { *settings = original })
	seed := newSelectedChannelContext()
	seed.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	seed.Request.Header.Set("X-Test-Affinity", t.Name())
	service.GetPreferredChannelAffinityBinding(seed, "client-model", "guard-route")
	common.SetContextKey(seed, constant.ContextKeyChannelId, oldChannel.Id)
	common.SetContextKey(seed, constant.ContextKeyChannelIsMultiKey, true)
	common.SetContextKey(seed, constant.ContextKeyChannelMultiKeyIndex, 0)
	service.RecordChannelAffinity(seed, oldChannel.Id)
	t.Cleanup(func() { service.ClearCurrentChannelAffinityCache(seed) })
	preferred, found := service.GetPreferredChannelByAffinity(seed, "client-model", "guard-route")
	require.True(t, found)
	require.Equal(t, oldChannel.Id, preferred)

	engine := gin.New()
	engine.POST("/v1/chat/completions", BodyStorageCleanup(), func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyUsingGroup, "guard-route")
		common.SetContextKey(c, constant.ContextKeyTokenGroup, "guard-route")
		common.SetContextKey(c, constant.ContextKeyTokenGroupChannels, map[string]int{"guard-route": target.Id})
	}, Distribute(), func(c *gin.Context) {
		c.String(http.StatusOK, strconv.Itoa(common.GetContextKeyInt(c, constant.ContextKeyChannelId)))
	})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"client-model"}`))
	request.Header.Set("Content-Type", gin.MIMEJSON)
	request.Header.Set("X-Test-Affinity", t.Name())
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	assert.Equal(t, strconv.Itoa(target.Id), response.Body.String())
}
