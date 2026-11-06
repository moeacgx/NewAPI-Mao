package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenAuthOverLimitGroupsUsesExplicitError(t *testing.T) {
	require.NoError(t, i18n.Init())
	setupDashboardAuthMiddlewareTest(t)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))

	originalMax := setting.GetMaxTokenAutoGroups()
	originalUsable := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("1"))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default display","vip":"VIP display"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(strconv.Itoa(originalMax)))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})

	user := createMiddlewarePATUser(t, "over-limit-owner", "over-limit-pat")
	token := &model.Token{
		UserId:         user.Id,
		Name:           "historical",
		Key:            "historicaloverlimit",
		Status:         common.TokenStatusEnabled,
		ExpiredTime:    -1,
		UnlimitedQuota: true,
		Group:          "default,vip",
		GroupMode:      model.TokenGroupModeExplicit,
	}
	require.NoError(t, model.DB.Create(token).Error)

	engine := gin.New()
	engine.GET("/v1/models", TokenAuth(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	overLimit := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	overLimit.Header.Set("Authorization", "Bearer sk-historicaloverlimit")
	overLimit.Header.Set("Accept-Language", "zh-CN")
	overLimitResponse := httptest.NewRecorder()
	engine.ServeHTTP(overLimitResponse, overLimit)
	require.Equal(t, http.StatusForbidden, overLimitResponse.Code, overLimitResponse.Body.String())
	assert.Contains(t, overLimitResponse.Body.String(), "令牌最多可选 1 个分组")
	assert.NotContains(t, overLimitResponse.Body.String(), "无效的令牌")

	require.NoError(t, setting.UpdateMaxTokenAutoGroups("2"))
	recovered := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	recovered.Header.Set("Authorization", "Bearer sk-historicaloverlimit")
	recovered.Header.Set("Accept-Language", "zh-CN")
	recoveredResponse := httptest.NewRecorder()
	engine.ServeHTTP(recoveredResponse, recovered)
	require.Equal(t, http.StatusNoContent, recoveredResponse.Code, recoveredResponse.Body.String())
	assert.NotContains(t, recoveredResponse.Body.String(), "无效的令牌")
}

func TestTokenAuthUnlimitedPreservesGroupAuthorization(t *testing.T) {
	setupDashboardAuthMiddlewareTest(t)
	model.InitDBColumns()
	require.NoError(t, model.DB.AutoMigrate(&model.Token{}))
	originalMax := setting.GetMaxTokenAutoGroups()
	originalUsable := setting.UserUsableGroups2JSONString()
	originalRatios := ratio_setting.GroupRatio2JSONString()
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("0"))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"标准显示名","vip":"高级显示名","svip":"旗舰显示名","team":"团队显示名","pro":"专业显示名","enterprise":"企业显示名"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":1,"svip":1,"team":1,"pro":1,"enterprise":1}`))
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(strconv.Itoa(originalMax)))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsable))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalRatios))
	})
	user := createMiddlewarePATUser(t, "unlimited-owner", "unlimited-pat")
	token := &model.Token{
		UserId: user.Id, Name: "unlimited", Key: "unlimitedgroups", Status: common.TokenStatusEnabled,
		ExpiredTime: -1, UnlimitedQuota: true, Group: "default,vip,svip,team,pro,enterprise", GroupMode: model.TokenGroupModeExplicit,
	}
	require.NoError(t, model.DB.Create(token).Error)
	engine := gin.New()
	engine.GET("/v1/models", TokenAuth(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for _, test := range []struct {
		name   string
		revoke bool
		status int
	}{
		{name: "six authorized groups", status: http.StatusNoContent},
		{name: "revoked group", revoke: true, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.revoke {
				require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"标准显示名"}`))
			}
			request := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
			request.Header.Set("Authorization", "Bearer sk-unlimitedgroups")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code, response.Body.String())
			if test.revoke {
				assert.Contains(t, response.Body.String(), "无权访问")
			}
		})
	}
}
