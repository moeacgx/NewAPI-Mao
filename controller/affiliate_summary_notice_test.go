package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type affiliateSummaryNoticeResponse struct {
	Success bool `json:"success"`
	Data    struct {
		PromotionNotice struct {
			Enabled bool   `json:"enabled"`
			Title   string `json:"title"`
			Text    string `json:"text"`
		} `json:"promotion_notice"`
	} `json:"data"`
}

func requestAffiliateSummaryNotice(t *testing.T, userId int) affiliateSummaryNoticeResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/affiliate/summary", nil)
	c.Set("id", userId)

	GetAffiliateSummary(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response affiliateSummaryNoticeResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	return response
}

func TestAffiliateSummaryExposesPromotionNoticeSettings(t *testing.T) {
	previousDB := model.DB
	previousRedis := common.RedisEnabled
	previousType := common.MainDatabaseType()
	previousSetting := *setting.GetAffiliateSetting()
	db, err := gorm.Open(sqlite.Open("file:affiliate_summary_notice?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.AffiliateBalance{}, &model.AffiliateRecord{}))
	model.DB = db
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	t.Cleanup(func() {
		*setting.GetAffiliateSetting() = previousSetting
		model.DB = previousDB
		common.RedisEnabled = previousRedis
		common.SetMainDatabaseType(previousType)
	})

	user := &model.User{
		Username: "affiliate-notice-user", Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AffCode: "abcd",
	}
	require.NoError(t, db.Create(user).Error)

	affiliateSetting := setting.GetAffiliateSetting()
	affiliateSetting.PromotionNoticeEnabled = false
	affiliateSetting.PromotionNoticeTitle = ""
	affiliateSetting.PromotionNoticeText = ""
	defaults := requestAffiliateSummaryNotice(t, user.Id)
	assert.False(t, defaults.Data.PromotionNotice.Enabled)
	assert.Empty(t, defaults.Data.PromotionNotice.Title)
	assert.Empty(t, defaults.Data.PromotionNotice.Text)

	affiliateSetting.PromotionNoticeEnabled = true
	affiliateSetting.PromotionNoticeTitle = "Custom <b>title</b>"
	affiliateSetting.PromotionNoticeText = "Line one\nLine <i>two</i>"
	configured := requestAffiliateSummaryNotice(t, user.Id)
	assert.True(t, configured.Data.PromotionNotice.Enabled)
	assert.Equal(t, "Custom <b>title</b>", configured.Data.PromotionNotice.Title)
	assert.Equal(t, "Line one\nLine <i>two</i>", configured.Data.PromotionNotice.Text)
}
