package relayconvert

import (
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeResponsesUsageSeparatesNativeAndConvertedSnapshots(t *testing.T) {
	var source dto.Usage
	require.NoError(t, kitutil.Unmarshal([]byte(`{"input_tokens":9,"output_tokens":3,"input_tokens_details":{"cached_tokens":4,"image_tokens":2}}`), &source))
	native := NormalizeResponsesUsage(&source)
	assert.Nil(t, native.BillingUsage)
	converted := UsageFromResponsesUsage(&source)
	require.NotNil(t, converted.BillingUsage)
	assert.Equal(t, dto.BillingUsageSourceOAIResponses, converted.BillingUsage.Source)
	assert.Equal(t, native.PromptTokens, converted.PromptTokens)
	assert.Equal(t, native.PromptTokensDetails, converted.PromptTokensDetails)
	assert.Nil(t, source.BillingUsage)

	source.BillingUsage = converted.BillingUsage
	preserved := NormalizeResponsesUsage(&source)
	require.NotNil(t, preserved.BillingUsage)
	assert.Equal(t, source.BillingUsage, preserved.BillingUsage)
	assert.NotSame(t, source.BillingUsage, preserved.BillingUsage)
	require.NotNil(t, preserved.InputTokensDetails)
	preserved.InputTokensDetails.ImageTokens = 99
	assert.Equal(t, 2, source.InputTokensDetails.ImageTokens)
	require.NotNil(t, preserved.BillingUsage.OpenAIUsage)
	preserved.BillingUsage.OpenAIUsage.InputTokensDetails.CachedTokens = 99
	assert.Equal(t, 4, source.BillingUsage.OpenAIUsage.InputTokensDetails.CachedTokens)
}

func TestNormalizeResponsesUsagePreservesExplicitZeroPresence(t *testing.T) {
	var source dto.Usage
	require.NoError(t, kitutil.Unmarshal([]byte(`{"input_tokens":0,"output_tokens":0,"input_tokens_details":{"cached_tokens":0,"cache_write_tokens":0},"cache_write_tokens":8}`), &source))
	usage := NormalizeResponsesUsage(&source)
	assert.True(t, usage.HasInputTokens)
	assert.True(t, usage.HasOutputTokens)
	assert.True(t, usage.HasPromptTokens)
	assert.True(t, usage.HasCompletionTokens)
	assert.False(t, usage.HasTotalTokens)
	assert.True(t, usage.PromptTokensDetails.HasCachedTokens)
	cacheWrite, present := usage.GetCacheCreationTokensWithPresence()
	assert.True(t, present)
	assert.Zero(t, cacheWrite, "嵌套显式零不能被顶层缓存创建别名覆盖")
	assert.Zero(t, usage.PromptTokens)
	assert.Zero(t, usage.CompletionTokens)

	missing := NormalizeResponsesUsage(nil)
	assert.False(t, missing.HasInputTokens)
	assert.False(t, missing.HasOutputTokens)
	assert.Nil(t, missing.InputTokensDetails)
	assert.Nil(t, missing.BillingUsage)
}
