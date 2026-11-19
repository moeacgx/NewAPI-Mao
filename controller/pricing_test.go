package controller

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPricingCatalogMatchesAnonymousModelSquare(t *testing.T) {
	setupCanvasControllerDB(t)
	oldGroups := setting.UserUsableGroups2JSONString()
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"公开分组"}`))
	model.InvalidatePricingCache()
	t.Cleanup(func() {
		model.InvalidatePricingCache()
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldGroups))
	})
	require.NoError(t, model.DB.AutoMigrate(&model.Model{}, &model.Vendor{}, &model.Channel{}, &model.Ability{}))
	channel := model.Channel{Name: "catalog-local", Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(&channel).Error)
	for _, ability := range []model.Ability{
		{Model: "public-square-model", Group: "default", Enabled: true, ChannelId: channel.Id},
		{Model: "private-square-model", Group: "private", Enabled: true, ChannelId: channel.Id},
		{Model: "all-square-model", Group: "all", Enabled: true, ChannelId: channel.Id},
	} {
		require.NoError(t, model.DB.Create(&ability).Error)
	}
	engine := gin.New()
	engine.GET("/api/pricing", GetPricing)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/pricing", nil))
	require.Equal(t, http.StatusOK, response.Code)
	var payload struct {
		Data []model.Pricing `json:"data"`
	}
	require.NoError(t, common.Unmarshal(response.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 2)
	assert.ElementsMatch(t, []string{"public-square-model", "all-square-model"}, []string{payload.Data[0].ModelName, payload.Data[1].ModelName})
	for _, name := range []string{"public-square-model", "all-square-model", "private-square-model"} {
		listed, err := model.IsModelInCatalog(name)
		require.NoError(t, err)
		assert.Equal(t, name != "private-square-model", listed, name)
	}
}

func TestBuildPricingGroupNamesUsesDisplayNameAndFallback(t *testing.T) {
	groupRatio := map[string]float64{
		"vip":      0.5,
		"fallback": 1,
	}
	activeGroupNames := map[string]string{
		"vip":        "尊贵用户",
		"legacy-vip": "尊贵用户",
		"ignored":    "不可用分组",
	}

	got := buildPricingGroupNames(groupRatio, activeGroupNames)
	want := map[string]string{
		"vip":        "尊贵用户",
		"legacy-vip": "尊贵用户",
		"fallback":   "fallback",
		"ignored":    "不可用分组",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("模型广场分组显示名称错误：got %#v, want %#v", got, want)
	}
}
