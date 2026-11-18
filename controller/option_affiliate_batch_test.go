package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

type optionBatchResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func putAffiliateBatch(t *testing.T, body string) optionBatchResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPut, "/api/option/affiliate-batch", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	UpdateAffiliateOptionsBatch(c)
	var response optionBatchResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

func TestUpdateAffiliateOptionsBatchIsAtomicAndValidated(t *testing.T) {
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousRedis := common.RedisEnabled
	previousType := common.MainDatabaseType()
	previousSetting := *setting.GetAffiliateSetting()
	db, err := gorm.Open(sqlite.Open("file:affiliate_option_batch?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.User{}, &model.Log{}))
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.InitDBColumns()
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		*setting.GetAffiliateSetting() = previousSetting
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedis
		common.SetMainDatabaseType(previousType)
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
	})

	ok := putAffiliateBatch(t, `{"options":[
		{"key":"affiliate_setting.first_level_ratio","value":8},
		{"key":"affiliate_setting.promotion_notice_enabled","value":true},
		{"key":"affiliate_setting.promotion_notice_title","value":"Batch title"},
		{"key":"affiliate_setting.promotion_notice_text","value":"Batch body"}]}`)
	require.True(t, ok.Success, ok.Message)
	current := setting.GetAffiliateSetting()
	assert.Equal(t, 8, current.FirstLevelRatio)
	assert.True(t, current.PromotionNoticeEnabled)
	assert.Equal(t, "Batch title", current.PromotionNoticeTitle)
	var stored []model.Option
	require.NoError(t, db.Where("key LIKE ?", "affiliate_setting.%").Find(&stored).Error)
	assert.Len(t, stored, 4)

	rejected := []string{
		`{"options":[]}`,
		`{"options":[{"key":"SystemName","value":"x"}]}`,
		`{"options":[{"key":"affiliate_setting.first_level_ratio","value":1},{"key":"affiliate_setting.first_level_ratio","value":2}]}`,
		`{"options":[{"key":"affiliate_setting.promotion_notice_enabled","value":"maybe"}]}`,
		`{"options":[{"key":"affiliate_setting.first_level_ratio","value":3},{"key":"affiliate_setting.promotion_notice_title","value":"` + strings.Repeat("a", 101) + `"}]}`,
		`{"options":[{"key":"affiliate_setting.first_level_ratio","value":{"nested":1}}]}`,
	}
	for _, body := range rejected {
		assert.False(t, putAffiliateBatch(t, body).Success, body)
	}
	// Nothing from the rejected batches may have been persisted or published.
	assert.Equal(t, 8, setting.GetAffiliateSetting().FirstLevelRatio)
	assert.Equal(t, "Batch title", setting.GetAffiliateSetting().PromotionNoticeTitle)
	var ratio model.Option
	require.NoError(t, db.Where("key = ?", "affiliate_setting.first_level_ratio").First(&ratio).Error)
	assert.Equal(t, "8", ratio.Value)
}
