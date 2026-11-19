package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 管理响应只投影模型已有分组，不扩大公开价格接口的可见范围。
func TestModelMetaGroupNamesResponses(t *testing.T) {
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
	originalRedis := common.RedisEnabled
	t.Cleanup(func() {
		model.DB, model.LOG_DB = originalDB, originalLogDB
		common.SetDatabaseTypes(originalMainType, originalLogType)
		common.RedisEnabled = originalRedis
		model.InvalidatePricingCache()
	})
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}, &model.Option{}))
	require.NoError(t, db.Create(&[]model.Group{
		{Id: 7, Code: "g_7", Name: "当前名称"},
		{Id: 8, Code: "private", Name: "不相关分组"},
	}).Error)
	require.NoError(t, db.Create(&model.GroupAlias{Alias: "legacy", GroupId: 7}).Error)
	require.NoError(t, db.Create(&model.GroupAlias{Alias: "__proto__", GroupId: 7}).Error)
	require.NoError(t, db.Create(&model.Channel{Id: 1, Name: "fixture", Type: 1, Status: 1}).Error)
	codes := []string{"g_7", "legacy", "deleted", "unknown", "__proto__"}
	for _, code := range codes {
		require.NoError(t, db.Create(&model.Ability{Group: code, Model: "fixture-model", ChannelId: 1, Enabled: true}).Error)
	}
	require.NoError(t, db.Create(&[]model.Model{
		{Id: 1, ModelName: "fixture-model", NameRule: model.NameRuleExact},
		{Id: 2, ModelName: "fixture-", NameRule: model.NameRulePrefix},
	}).Error)
	model.InvalidatePricingCache()

	router := gin.New()
	router.GET("/api/models/", GetAllModelsMeta)
	router.GET("/api/models/search", SearchModelsMeta)
	router.GET("/api/models/:id", GetModelMeta)
	type responseModel struct {
		ID     int               `json:"id"`
		Groups []string          `json:"enable_groups"`
		Names  map[string]string `json:"enable_group_names"`
	}
	for _, phase := range []struct{ name, current string }{
		{"初始名称", "当前名称"},
		{"改名后", "新的显示名称"},
		{"删除后", ""},
	} {
		t.Run(phase.name, func(t *testing.T) {
			if phase.current == "" {
				require.NoError(t, db.Delete(&model.Group{}, 7).Error)
			} else {
				require.NoError(t, db.Model(&model.Group{}).Where("id = ?", 7).Update("name", phase.current).Error)
			}
			for _, path := range []string{"/api/models/", "/api/models/search?keyword=fixture", "/api/models/1", "/api/models/2"} {
				t.Run(path, func(t *testing.T) {
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
					require.Equal(t, http.StatusOK, recorder.Code)
					var envelope struct {
						Success bool `json:"success"`
						Data    struct {
							responseModel
							Items []responseModel `json:"items"`
						} `json:"data"`
					}
					require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &envelope))
					require.True(t, envelope.Success, recorder.Body.String())
					items := envelope.Data.Items
					if envelope.Data.ID != 0 {
						items = []responseModel{envelope.Data.responseModel}
					}
					require.NotEmpty(t, items)
					for _, item := range items {
						assert.ElementsMatch(t, codes, item.Groups)
						expected := map[string]string{"deleted": "deleted", "unknown": "unknown"}
						for _, code := range []string{"g_7", "legacy", "__proto__"} {
							if phase.current == "" {
								expected[code] = code
							} else {
								expected[code] = phase.current
							}
						}
						assert.Equal(t, expected, item.Names)
						assert.NotContains(t, item.Names, "private")
					}
				})
			}
		})
	}
}
