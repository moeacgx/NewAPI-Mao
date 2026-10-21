package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSub2APIBalanceTest(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.Channel{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = originalDB
		require.NoError(t, sqlDB.Close())
	})
}

func newSub2APITestChannel(t *testing.T, baseURL string) *model.Channel {
	t.Helper()
	channel := &model.Channel{Type: constant.ChannelTypeSub2API, BaseURL: &baseURL, Key: "sk-test", Name: "sub2api-test"}
	require.NoError(t, model.DB.Create(channel).Error)
	return channel
}

func TestUpdateChannelSub2APIBalancePrefersRemaining(t *testing.T) {
	setupSub2APIBalanceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/usage", r.URL.Path)
		assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balance":100,"remaining":986.17,"isValid":true}`))
	}))
	defer server.Close()

	channel := newSub2APITestChannel(t, server.URL)
	balance, err := updateChannelSub2APIBalance(channel)
	require.NoError(t, err)
	assert.InDelta(t, 986.17, balance, 1e-9)
	assert.InDelta(t, 986.17, channel.Balance, 1e-9)
}

func TestUpdateChannelSub2APIBalanceWalletFallback(t *testing.T) {
	setupSub2APIBalanceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balance":42.5,"isValid":true}`))
	}))
	defer server.Close()

	channel := newSub2APITestChannel(t, server.URL)
	balance, err := updateChannelSub2APIBalance(channel)
	require.NoError(t, err)
	assert.InDelta(t, 42.5, balance, 1e-9)
}

func TestUpdateChannelSub2APIBalanceRejectsInvalidKey(t *testing.T) {
	setupSub2APIBalanceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"isValid":false}`))
	}))
	defer server.Close()

	channel := newSub2APITestChannel(t, server.URL)
	_, err := updateChannelSub2APIBalance(channel)
	require.EqualError(t, err, "Sub2API key is invalid (isValid=false)")
}

func TestUpdateChannelSub2APIBalanceRejectsUnlimitedSubscription(t *testing.T) {
	setupSub2APIBalanceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"remaining":-1,"isValid":true,"subscription":{"daily_limit_usd":null,"weekly_limit_usd":null,"monthly_limit_usd":null}}`))
	}))
	defer server.Close()

	channel := newSub2APITestChannel(t, server.URL)
	_, err := updateChannelSub2APIBalance(channel)
	require.EqualError(t, err, "Sub2API /v1/usage reports an unlimited subscription without a finite balance")
}

func TestUpdateStandardChannelBalanceRoutesSub2API(t *testing.T) {
	setupSub2APIBalanceTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"balance":77.7,"isValid":true}`))
	}))
	defer server.Close()

	channel := newSub2APITestChannel(t, server.URL)
	balance, err := updateChannelBalance(channel)
	require.NoError(t, err)
	assert.InDelta(t, 77.7, balance, 1e-9)
}

func TestSub2APIBalanceUnlimitedSentinelAndFiniteValues(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		want   float64
		reject bool
	}{
		{"零限额的无限标记", `{"remaining":-1,"subscription":{"daily_limit_usd":0,"weekly_limit_usd":0,"monthly_limit_usd":0}}`, 12, true},
		{"负限额的无限标记", `{"remaining":-1,"subscription":{"daily_limit_usd":-2}}`, 12, true},
		{"无限订阅有限余额", `{"remaining":8,"subscription":{"daily_limit_usd":0}}`, 8, false},
		{"无限订阅钱包回退", `{"balance":5,"subscription":{"daily_limit_usd":0}}`, 5, false},
		{"普通钱包负余额", `{"balance":-1}`, -1, false},
		{"有限订阅负余额", `{"remaining":-1,"subscription":{"daily_limit_usd":10}}`, -1, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setupSub2APIBalanceTest(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tt.body)) }))
			defer server.Close()
			channel := newSub2APITestChannel(t, server.URL)
			channel.UpdateBalance(12)
			balance, err := updateChannelBalance(channel)
			if tt.reject {
				require.ErrorContains(t, err, "unlimited subscription")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, balance)
			}
			var stored model.Channel
			require.NoError(t, model.DB.First(&stored, channel.Id).Error)
			assert.Equal(t, tt.want, stored.Balance)
		})
	}
}
