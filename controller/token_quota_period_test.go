package controller

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quotaUnitsFromUSD(usd int64) int64 {
	return usd * int64(common.QuotaPerUnit)
}

func setupTokenPeriodTest(t *testing.T) *model.User {
	t.Helper()
	configureTokenAutoGroupsTest(t, "0", `["default","vip"]`)
	user := setupTokenAutoGroupsControllerTest(t)
	require.NoError(t, model.DB.AutoMigrate(&model.Group{}, &model.TokenGroupBinding{}))
	return user
}

func submitPeriodToken(t *testing.T, method string, user *model.User, fields map[string]any) tokenAPIResponse {
	t.Helper()
	request := baseAutoTokenRequest("period-token")
	for key, value := range fields {
		request[key] = value
	}
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, method, "/api/token/", request, user.Id)
	if method == http.MethodPost {
		AddToken(ctx)
	} else {
		UpdateToken(ctx)
	}
	return decodeAPIResponse(t, recorder)
}

func createPeriodToken(t *testing.T, user *model.User, fields map[string]any) model.Token {
	t.Helper()
	response := submitPeriodToken(t, http.MethodPost, user, fields)
	require.True(t, response.Success, response.Message)
	var stored model.Token
	require.NoError(t, model.DB.Order("id desc").First(&stored).Error)
	return stored
}

// readPeriodToken reads the token through the same handler the edit form uses to prefill.
func readPeriodToken(t *testing.T, user *model.User, id int) map[string]any {
	t.Helper()
	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/"+strconv.Itoa(id), nil, user.Id)
	ctx.Params = gin.Params{{Key: "id", Value: strconv.Itoa(id)}}
	GetToken(ctx)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var body map[string]any
	require.NoError(t, common.Unmarshal(response.Data, &body))
	return body
}

func TestAddTokenWeekly500UsdWithTotal10000UsdSavesAndReadsBack(t *testing.T) {
	user := setupTokenPeriodTest(t)
	weekly := quotaUnitsFromUSD(500)
	total := quotaUnitsFromUSD(10000)
	require.EqualValues(t, 5_000_000_000, total, "10000 USD exceeds the int32 per-request boundary")

	stored := createPeriodToken(t, user, map[string]any{
		"unlimited_quota": false, "remain_quota": total,
		"quota_period": "weekly", "quota_period_limit": weekly,
	})
	assert.Equal(t, "weekly", stored.QuotaPeriod)
	assert.Equal(t, weekly, stored.QuotaPeriodLimit)
	assert.EqualValues(t, total, stored.RemainQuota)
	assert.False(t, stored.UnlimitedQuota)

	body := readPeriodToken(t, user, stored.Id)
	assert.Equal(t, "weekly", body["quota_period"])
	assert.EqualValues(t, weekly, body["quota_period_limit"])
	assert.EqualValues(t, total, body["remain_quota"])
	assert.EqualValues(t, 0, body["quota_period_used"])
	assert.Greater(t, body["quota_period_reset_at"], float64(0))
}

func TestUpdateTokenSwitchesDailyAndWeeklyAndReopensWithSameValues(t *testing.T) {
	user := setupTokenPeriodTest(t)
	limit := quotaUnitsFromUSD(500)
	stored := createPeriodToken(t, user, map[string]any{
		"unlimited_quota": true, "quota_period": "daily", "quota_period_limit": limit,
	})
	assert.Equal(t, "daily", readPeriodToken(t, user, stored.Id)["quota_period"])

	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", stored.Id).Updates(map[string]any{
		"quota_period_used": 123, "quota_period_window": 42,
	}).Error)

	for _, period := range []string{"weekly", "daily"} {
		response := submitPeriodToken(t, http.MethodPut, user, map[string]any{
			"id": stored.Id, "status": common.TokenStatusEnabled,
			"unlimited_quota": true, "quota_period": period, "quota_period_limit": limit,
		})
		require.True(t, response.Success, response.Message)
		body := readPeriodToken(t, user, stored.Id)
		assert.Equal(t, period, body["quota_period"])
		assert.EqualValues(t, limit, body["quota_period_limit"])
		assert.Equal(t, true, body["unlimited_quota"])
		assert.EqualValues(t, 0, body["quota_period_used"], "changing the period starts a fresh ledger")
	}
}

func TestUnlimitedTotalKeepsPeriodAndPeriodListShowsBoth(t *testing.T) {
	user := setupTokenPeriodTest(t)
	weekly := quotaUnitsFromUSD(500)
	createPeriodToken(t, user, map[string]any{
		"unlimited_quota": true, "quota_period": "weekly", "quota_period_limit": weekly,
	})

	ctx, recorder := newTokenAutoGroupsAuthenticatedContext(t, http.MethodGet, "/api/token/?p=1&page_size=10", nil, user.Id)
	GetAllTokens(ctx)
	response := decodeAPIResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var page struct {
		Items []map[string]any `json:"items"`
	}
	require.NoError(t, common.Unmarshal(response.Data, &page))
	require.Len(t, page.Items, 1)
	assert.Equal(t, true, page.Items[0]["unlimited_quota"])
	assert.Equal(t, "weekly", page.Items[0]["quota_period"])
	assert.EqualValues(t, weekly, page.Items[0]["quota_period_limit"])

	// Turning the total quota to a finite value must not touch the period.
	id := int(page.Items[0]["id"].(float64))
	response = submitPeriodToken(t, http.MethodPut, user, map[string]any{
		"id": id, "status": common.TokenStatusEnabled, "unlimited_quota": false,
		"remain_quota": quotaUnitsFromUSD(10000), "quota_period": "weekly", "quota_period_limit": weekly,
	})
	require.True(t, response.Success, response.Message)
	body := readPeriodToken(t, user, id)
	assert.Equal(t, "weekly", body["quota_period"])
	assert.EqualValues(t, weekly, body["quota_period_limit"])
	assert.EqualValues(t, quotaUnitsFromUSD(10000), body["remain_quota"])
}

func TestUpdateTokenOmittingPeriodKeepsStoredPeriodAndLegacyTokenHasNone(t *testing.T) {
	user := setupTokenPeriodTest(t)
	limit := quotaUnitsFromUSD(500)
	stored := createPeriodToken(t, user, map[string]any{
		"unlimited_quota": true, "quota_period": "weekly", "quota_period_limit": limit,
	})
	// An older client never sends the period fields.
	response := submitPeriodToken(t, http.MethodPut, user, map[string]any{"id": stored.Id, "status": common.TokenStatusEnabled, "unlimited_quota": true})
	require.True(t, response.Success, response.Message)
	body := readPeriodToken(t, user, stored.Id)
	assert.Equal(t, "weekly", body["quota_period"])
	assert.EqualValues(t, limit, body["quota_period_limit"])

	legacy := createPeriodToken(t, user, map[string]any{"unlimited_quota": true})
	assert.Equal(t, "none", legacy.QuotaPeriod)
	assert.EqualValues(t, 0, legacy.QuotaPeriodLimit)

	// Choosing no period clears the limit.
	response = submitPeriodToken(t, http.MethodPut, user, map[string]any{"id": stored.Id, "status": common.TokenStatusEnabled, "unlimited_quota": true, "quota_period": "none", "quota_period_limit": limit})
	require.True(t, response.Success, response.Message)
	body = readPeriodToken(t, user, stored.Id)
	assert.Equal(t, "none", body["quota_period"])
	assert.EqualValues(t, 0, body["quota_period_limit"])
}

func TestTokenQuotaRequestBoundariesAreValidatedByBackend(t *testing.T) {
	user := setupTokenPeriodTest(t)
	maxQuota := common.MaxTokenQuota()
	require.Greater(t, maxQuota, int64(common.MaxQuota), "the total quota bound must not be the int32 charge boundary")

	tests := []struct {
		name    string
		fields  map[string]any
		allowed bool
	}{
		{"total at max", map[string]any{"unlimited_quota": false, "remain_quota": maxQuota}, true},
		{"total over max", map[string]any{"unlimited_quota": false, "remain_quota": maxQuota + 1}, false},
		{"total negative", map[string]any{"unlimited_quota": false, "remain_quota": -1}, false},
		{"period at max", map[string]any{"quota_period": "daily", "quota_period_limit": maxQuota}, true},
		{"period over max", map[string]any{"quota_period": "daily", "quota_period_limit": maxQuota + 1}, false},
		{"period zero with daily", map[string]any{"quota_period": "daily", "quota_period_limit": 0}, false},
		{"period negative", map[string]any{"quota_period": "weekly", "quota_period_limit": -5}, false},
		{"unknown period", map[string]any{"quota_period": "monthly", "quota_period_limit": 100}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var before int64
			require.NoError(t, model.DB.Model(&model.Token{}).Count(&before).Error)
			response := submitPeriodToken(t, http.MethodPost, user, test.fields)
			assert.Equal(t, test.allowed, response.Success, response.Message)
			var after int64
			require.NoError(t, model.DB.Model(&model.Token{}).Count(&after).Error)
			if test.allowed {
				assert.Equal(t, before+1, after)
			} else {
				assert.Equal(t, before, after, "a rejected request must not persist or truncate anything")
			}
		})
	}
}

func TestTokenRequestCannotWriteServerSideLedger(t *testing.T) {
	user := setupTokenPeriodTest(t)
	stored := createPeriodToken(t, user, map[string]any{
		"unlimited_quota": true, "quota_period": "daily", "quota_period_limit": 1000,
		"quota_period_used": 999, "quota_period_window": 7,
	})
	assert.EqualValues(t, 0, stored.QuotaPeriodUsed)
	assert.EqualValues(t, 0, stored.QuotaPeriodWindow)
}
