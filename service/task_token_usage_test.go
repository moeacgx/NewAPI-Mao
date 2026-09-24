package service

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	channelmetrics "github.com/QuantumNous/new-api/pkg/channel_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	kitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskTokenUsageAcceptsIndependentFiniteIntegerFacts(t *testing.T) {
	usage := TaskTokenUsage(map[string]any{
		"input_tokens":  486.0,
		"output_tokens": 0.0,
		"totalTokens":   9999.0,
	})
	require.NotNil(t, usage)
	assert.Equal(t, 486, usage.PromptTokens)
	assert.True(t, usage.HasPromptTokens)
	assert.Equal(t, 0, usage.CompletionTokens)
	assert.True(t, usage.HasCompletionTokens)
}

func TestTaskTokenUsageIgnoresInvalidFieldWithoutDroppingOtherField(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
	}{
		{"负数", -1.0}, {"小数", 1.5}, {"字符串", "486"}, {"整数类型", 486},
		{"布尔值", false}, {"空值", nil}, {"非数", math.NaN()},
		{"正无穷", math.Inf(1)}, {"负无穷", math.Inf(-1)},
		{"超出日志整数上限", float64(math.MaxInt32) + 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usage := TaskTokenUsage(map[string]any{"input_tokens": tt.value, "output_tokens": 70.0})
			require.NotNil(t, usage)
			assert.False(t, usage.HasPromptTokens)
			assert.Equal(t, 70, usage.CompletionTokens)
			assert.True(t, usage.HasCompletionTokens)
			assert.Nil(t, TaskTokenUsage(map[string]any{"output_tokens": tt.value}))
		})
	}
	usage := TaskTokenUsage(map[string]any{"input_tokens": float64(math.MaxInt32)})
	require.NotNil(t, usage)
	assert.Equal(t, math.MaxInt32, usage.PromptTokens)
	assert.True(t, usage.HasPromptTokens)
	assert.False(t, usage.HasCompletionTokens)
}

func TestTaskTokenUsageDoesNotInferLegacyBillingUnits(t *testing.T) {
	assert.Nil(t, TaskTokenUsage(nil))
	assert.Nil(t, TaskTokenUsage(map[string]any{
		"upstreamUnits": 3.5, "completionTokens": 50.0, "totalTokens": 100.0,
		"estimated_input_tokens": 32000.0,
	}))
}

func TestTaskTokenUsageRejectsCombinedLogOverflow(t *testing.T) {
	for _, tt := range []struct {
		name     string
		input    float64
		output   float64
		accepted bool
	}{
		{"两项均为上限", float64(math.MaxInt32), float64(math.MaxInt32), false},
		{"总和超出上限一项", float64(math.MaxInt32), 1, false},
		{"总和恰为上限", float64(math.MaxInt32) - 1, 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usage := TaskTokenUsage(map[string]any{"input_tokens": tt.input, "output_tokens": tt.output})
			if !tt.accepted {
				assert.Nil(t, usage)
				return
			}
			require.NotNil(t, usage)
			assert.Equal(t, math.MaxInt32-1, usage.PromptTokens)
			assert.Equal(t, 1, usage.CompletionTokens)
			assert.True(t, usage.HasPromptTokens)
			assert.True(t, usage.HasCompletionTokens)
		})
	}
}

func TestTaskTokenUsageFeedsChannelMetricsWithFixedQuota(t *testing.T) {
	collector, _ := installChannelMetricTestRuntime(t)
	c := newChannelMetricTestContext(t, "task-token-metrics")
	info := newChannelMetricTestRelayInfo()
	info.ChannelMeta.ChannelId = 97002
	usage := TaskTokenUsage(map[string]any{"input_tokens": 486.0, "output_tokens": 70.0})
	require.NotNil(t, usage)
	BeginChannelMetricRequest(c)
	BindChannelMetricRelayInfo(c, info)
	BeginChannelMetricAttempt(c, info, 97002, "cloudflare-jev", 1)
	AttachChannelMetricUsageAfterSettlement(c, ChannelMetricUsage{
		InputTokensTotal: int64(usage.PromptTokens), OutputTokens: int64(usage.CompletionTokens),
	}, 500, nil)
	FinishChannelMetricAttempt(c, info, nil, false, "")
	c.Writer.WriteHeader(http.StatusOK)
	FinishChannelMetricRequest(c, info, nil)

	batch := drainChannelMetricTestBatch(t, collector)
	attempts := channelMetricTestBuckets(batch, channelmetrics.ScopeChannelAttempt, channelmetrics.OutcomeSuccess)
	require.Len(t, attempts, 1)
	assert.EqualValues(t, 486, attempts[0].Counters.InputTokensTotal)
	assert.EqualValues(t, 70, attempts[0].Counters.OutputTokens)
	assert.EqualValues(t, 500, attempts[0].Counters.ChargedQuota)
}

func TestLogTaskConsumptionRecordsActualTokens(t *testing.T) {
	truncate(t)
	user := model.User{Id: 97001, Username: "task-token-usage", Quota: 10000}
	channel := model.Channel{Id: 97001, Name: "task-token-usage"}
	require.NoError(t, model.DB.Create(&user).Error)
	require.NoError(t, model.DB.Create(&channel).Error)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", nil)
	c.Set(common.RequestIdKey, "task-token-usage")
	info := &relaycommon.RelayInfo{
		StartTime: time.Now().Add(-3 * time.Second),
		UserId:    user.Id, OriginModelName: "Typesafe-jev", UsingGroup: "default",
		ChannelMeta:   &relaycommon.ChannelMeta{ChannelId: channel.Id},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{},
		PriceData:     types.PriceData{Quota: 500, ModelPrice: 0.001},
	}
	LogTaskConsumption(c, info, &kitdto.Usage{PromptTokens: 486, CompletionTokens: 70, HasPromptTokens: true, HasCompletionTokens: true})
	var log model.Log
	require.NoError(t, model.LOG_DB.Where("request_id = ?", "task-token-usage").First(&log).Error)
	assert.Equal(t, 486, log.PromptTokens)
	assert.Equal(t, 70, log.CompletionTokens)
	assert.Equal(t, 500, log.Quota)
	assert.Equal(t, 3, log.UseTime)
	var other map[string]any
	require.NoError(t, common.UnmarshalJsonStr(log.Other, &other))
	assert.Equal(t, map[string]any{"input_tokens": 486.0, "output_tokens": 70.0}, other["task_token_usage"])
	require.NoError(t, model.DB.First(&user, user.Id).Error)
	assert.EqualValues(t, 500, user.UsedQuota)
	assert.EqualValues(t, 1, user.RequestCount)
	require.NoError(t, model.DB.First(&channel, channel.Id).Error)
	assert.EqualValues(t, 500, channel.UsedQuota)

	for _, tt := range []struct {
		name  string
		usage *kitdto.Usage
		want  map[string]any
	}{
		{"只有显式零输出", TaskTokenUsage(map[string]any{"output_tokens": 0.0}), map[string]any{"output_tokens": 0.0}},
		{"缺失用量", nil, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requestID := "task-token-usage-" + tt.name
			c.Set(common.RequestIdKey, requestID)
			LogTaskConsumption(c, info, tt.usage)
			var row model.Log
			require.NoError(t, model.LOG_DB.Where("request_id = ?", requestID).First(&row).Error)
			assert.Zero(t, row.PromptTokens)
			assert.Zero(t, row.CompletionTokens)
			assert.Equal(t, 500, row.Quota)
			var metadata map[string]any
			require.NoError(t, common.UnmarshalJsonStr(row.Other, &metadata))
			if tt.want == nil {
				assert.NotContains(t, metadata, "task_token_usage")
			} else {
				assert.Equal(t, tt.want, metadata["task_token_usage"])
			}
		})
	}
}
