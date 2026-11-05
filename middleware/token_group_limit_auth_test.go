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
