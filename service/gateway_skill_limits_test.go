package service

import (
	"net/http"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewaySkillAdmissionProtectsCountAndContextBudget(t *testing.T) {
	limiter := &gatewaySkillLimiter{}
	first, apiErr := limiter.acquire(6, 2, 10)
	require.Nil(t, apiErr)
	_, apiErr = limiter.acquire(5, 2, 10)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.True(t, types.IsSkipRetryError(apiErr))
	second, apiErr := limiter.acquire(4, 2, 10)
	require.Nil(t, apiErr)
	_, apiErr = limiter.acquire(0, 2, 10)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	first.Release()
	first.Release()
	_, apiErr = limiter.acquire(11, 2, 10)
	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.StatusCode)
	third, apiErr := limiter.acquire(6, 2, 10)
	require.Nil(t, apiErr)
	second.Release()
	third.Release()
	var absent *GatewaySkillAdmission
	absent.Release()
	assert.Zero(t, limiter.active)
	assert.Zero(t, limiter.contextBytes)
}

func TestGatewaySkillAdmissionReleasesAtomically(t *testing.T) {
	limiter := &gatewaySkillLimiter{}
	admission, apiErr := limiter.acquire(10, 1, 10)
	require.Nil(t, apiErr)
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			admission.Release()
		}()
	}
	workers.Wait()
	next, apiErr := limiter.acquire(10, 1, 10)
	require.Nil(t, apiErr)
	next.Release()
	assert.Zero(t, limiter.active)
	assert.Zero(t, limiter.contextBytes)
}

func TestGatewaySkillAdmissionConfigurationCannotDisableProtection(t *testing.T) {
	t.Setenv("GATEWAY_SKILLS_MAX_CONCURRENT", "0")
	t.Setenv("GATEWAY_SKILLS_MAX_INFLIGHT_CONTEXT_BYTES", "-1")
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_TIMEOUT_SECONDS", "invalid")
	settings := gatewaySkillResourceLimits()
	assert.Positive(t, settings.maxConcurrent)
	assert.Positive(t, settings.maxContextBytes)
	assert.Positive(t, GatewaySkillDiscoveryTimeout())
	t.Setenv("GATEWAY_SKILLS_MAX_CONCURRENT", "3")
	t.Setenv("GATEWAY_SKILLS_MAX_INFLIGHT_CONTEXT_BYTES", "4096")
	t.Setenv("GATEWAY_SKILLS_DISCOVERY_TIMEOUT_SECONDS", "30")
	settings = gatewaySkillResourceLimits()
	assert.Equal(t, 3, settings.maxConcurrent)
	assert.EqualValues(t, 4096, settings.maxContextBytes)
	assert.EqualValues(t, 30, GatewaySkillDiscoveryTimeout().Seconds())
}
