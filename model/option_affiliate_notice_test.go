package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAffiliatePromotionNoticeOptionsValidatedBeforePersistence(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Option{}))
	affiliateSetting := setting.GetAffiliateSetting()
	previous := *affiliateSetting
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)
	common.OptionMapRWMutex.Unlock()
	keys := []string{
		setting.AffiliatePromotionNoticeEnabledKey,
		setting.AffiliatePromotionNoticeTitleKey,
		setting.AffiliatePromotionNoticeTextKey,
		"affiliate_setting.first_level_ratio",
	}
	t.Cleanup(func() {
		*setting.GetAffiliateSetting() = previous
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		_ = DB.Where("key IN ?", keys).Delete(&Option{}).Error
	})

	require.NoError(t, UpdateOptionsBulk(map[string]string{
		setting.AffiliatePromotionNoticeEnabledKey: "true",
		setting.AffiliatePromotionNoticeTitleKey:   strings.Repeat("标", setting.AffiliatePromotionNoticeTitleMaxRunes),
		setting.AffiliatePromotionNoticeTextKey:    strings.Repeat("文", setting.AffiliatePromotionNoticeTextMaxRunes),
		"affiliate_setting.first_level_ratio":      "7",
	}))
	assert.True(t, setting.GetAffiliateSetting().PromotionNoticeEnabled)
	assert.Equal(t, 7, setting.GetAffiliateSetting().FirstLevelRatio)

	// Empty text is a valid value: the pages fall back to their built-in copy.
	require.NoError(t, UpdateOptionsBulk(map[string]string{
		setting.AffiliatePromotionNoticeTitleKey: "",
		setting.AffiliatePromotionNoticeTextKey:  "",
	}))
	assert.Empty(t, setting.GetAffiliateSetting().PromotionNoticeTitle)

	invalid := []map[string]string{
		{setting.AffiliatePromotionNoticeEnabledKey: "yes"},
		{setting.AffiliatePromotionNoticeEnabledKey: ""},
		{setting.AffiliatePromotionNoticeTitleKey: strings.Repeat("标", setting.AffiliatePromotionNoticeTitleMaxRunes+1)},
		{setting.AffiliatePromotionNoticeTextKey: strings.Repeat("文", setting.AffiliatePromotionNoticeTextMaxRunes+1)},
		{setting.AffiliatePromotionNoticeTitleKey: "bad\xff"},
	}
	for _, values := range invalid {
		require.Error(t, UpdateOptionsBulk(values), "%v", values)
	}

	// One invalid member must roll the whole mixed save back: the valid ratio
	// and switch values are neither stored nor published.
	require.Error(t, UpdateOptionsBulk(map[string]string{
		"affiliate_setting.first_level_ratio":      "9",
		setting.AffiliatePromotionNoticeEnabledKey: "false",
		setting.AffiliatePromotionNoticeTextKey:    strings.Repeat("文", setting.AffiliatePromotionNoticeTextMaxRunes+1),
	}))
	var ratio Option
	require.NoError(t, DB.Where("key = ?", "affiliate_setting.first_level_ratio").First(&ratio).Error)
	assert.Equal(t, "7", ratio.Value)
	var enabled Option
	require.NoError(t, DB.Where("key = ?", setting.AffiliatePromotionNoticeEnabledKey).First(&enabled).Error)
	assert.Equal(t, "true", enabled.Value)
	assert.Equal(t, 7, setting.GetAffiliateSetting().FirstLevelRatio)
	assert.True(t, setting.GetAffiliateSetting().PromotionNoticeEnabled)
}
