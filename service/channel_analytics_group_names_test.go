package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	channelmetrics "github.com/QuantumNous/new-api/pkg/channel_metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelAnalyticsCurrentGroupNamesKeepIdentifiersAndFilterScope(t *testing.T) {
	for _, scenario := range []struct {
		name string
		want string
	}{
		{name: "当前实体覆盖历史标识", want: "当前企业分组"},
		{name: "改名立即刷新", want: "改名后的企业分组"},
		{name: "删除实体回退原标识", want: "old-enterprise"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := setupChannelAnalyticsTestDB(t)
			require.NoError(t, db.AutoMigrate(&model.Group{}, &model.GroupAlias{}))
			require.NoError(t, db.Create(&[]model.Group{
				{Id: 81, Code: "enterprise", Name: "当前企业分组"},
				{Id: 82, Code: "unused-private", Name: "不应出现在筛选中的分组", UserSelectable: false},
			}).Error)
			require.NoError(t, db.Create(&model.GroupAlias{Alias: "old-enterprise", GroupId: 81}).Error)
			require.NoError(t, db.Create(&model.Channel{Id: 71, Name: "验证渠道", Type: 1, Key: "fixture", Group: "old-enterprise,unknown"}).Error)
			bucketTs := time.Now().Unix() / 300 * 300
			bucket := channelAnalyticsTestBucket(bucketTs, "group-name-alias", string(channelmetrics.ScopeChannelAttempt), string(channelmetrics.OutcomeSuccess), 2, 2)
			bucket.ChannelPresent, bucket.ChannelId = true, 71
			bucket.Group, bucket.GroupHash = "old-enterprise", channelmetrics.SHA256String("old-enterprise")
			require.NoError(t, db.Create(&bucket).Error)
			query := channelAnalyticsTestQuery(bucketTs)
			query.Groups = []string{"old-enterprise"}
			if scenario.want == "改名后的企业分组" {
				require.NoError(t, db.Model(&model.Group{}).Where("id = ?", 81).Update("name", scenario.want).Error)
			}
			if scenario.want == "old-enterprise" {
				require.NoError(t, db.Delete(&model.Group{}, 81).Error)
			}
			channels, err := GetChannelAnalyticsChannels(query)
			require.NoError(t, err)
			require.Len(t, channels.Items, 1)
			assert.Equal(t, "old-enterprise,unknown", channels.Items[0].Group)
			// 校验追加的公开 JSON 字段，旧接口字段保留原义。
			encoded, err := common.Marshal(channels.Items[0])
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, common.Unmarshal(encoded, &payload))
			assert.Equal(t, scenario.want+", unknown", payload["group_name"])
			filters, err := GetChannelAnalyticsFilters()
			require.NoError(t, err)
			assert.Equal(t, []dto.ChannelAnalyticsFilterGroup{{Code: "old-enterprise", Name: scenario.want}}, filters.Groups)
			stability, err := GetChannelAnalyticsStability(dto.ChannelAnalyticsStabilityQuery{
				ChannelAnalyticsQuery: query, Dimension: "group", WindowSeconds: []int64{300},
			})
			require.NoError(t, err)
			require.Len(t, stability.Items, 1)
			assert.Equal(t, "old-enterprise", stability.Items[0].Group)
			assert.Equal(t, scenario.want, stability.Items[0].GroupName)
			assert.Equal(t, []string{"old-enterprise"}, query.Groups)
		})
	}
}

func TestChannelAnalyticsGroupNameLookupFailurePreservesMetrics(t *testing.T) {
	// 此夹具有渠道和指标表但没有分组表，模拟名称查询失败。
	db := setupChannelAnalyticsTestDB(t)
	require.NoError(t, db.Create(&model.Channel{Id: 71, Name: "验证渠道", Type: 1, Key: "fixture", Group: "vip,unknown"}).Error)
	bucketTs := time.Now().Unix() / 300 * 300
	bucket := channelAnalyticsTestBucket(bucketTs, "lookup-failure", string(channelmetrics.ScopeChannelAttempt), string(channelmetrics.OutcomeSuccess), 2, 2)
	bucket.ChannelPresent, bucket.ChannelId = true, 71
	require.NoError(t, db.Create(&bucket).Error)
	response, err := GetChannelAnalyticsChannels(channelAnalyticsTestQuery(bucketTs))
	require.NoError(t, err)
	require.Len(t, response.Items, 1)
	assert.Equal(t, int64(2), response.Items[0].ChannelAttemptCount)
	encoded, err := common.Marshal(response.Items[0])
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(encoded, &payload))
	assert.Equal(t, "VIP 用户, unknown", payload["group_name"])
}

func TestChannelAnalyticsGroupNamesDoNotHideMetricQueryFailure(t *testing.T) {
	db := setupChannelAnalyticsTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	_, err = GetChannelAnalyticsChannels(channelAnalyticsTestQuery(time.Now().Unix() / 300 * 300))
	require.Error(t, err)
}
