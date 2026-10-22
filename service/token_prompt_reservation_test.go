package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPromptReservationRejectsInsufficientWalletWithoutDebt(t *testing.T) {
	oldDB, oldRedis := model.DB, common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:prompt-reservation-wallet?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = oldDB
		common.RedisEnabled = oldRedis
		sqlDB, e := db.DB()
		require.NoError(t, e)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}))
	require.NoError(t, db.Create(&model.User{Id: 63, Username: "prompt-budget", Quota: 10}).Error)
	info := &relaycommon.RelayInfo{UserId: 63, TokenQuotaExempt: true}
	session := &BillingSession{relayInfo: info, funding: &WalletFunding{userId: 63, consumed: 20}, preConsumedQuota: 20}
	require.ErrorIs(t, session.ReserveRequestQuota(40), ErrInsufficientWalletQuota)
	var user model.User
	require.NoError(t, db.First(&user, 63).Error)
	assert.Equal(t, int64(10), user.Quota)
	assert.Equal(t, 20, session.GetPreConsumedQuota())
	require.NoError(t, session.ReserveRequestQuota(25))
	require.NoError(t, db.First(&user, 63).Error)
	assert.Equal(t, int64(5), user.Quota)
	assert.Equal(t, 25, session.GetPreConsumedQuota())
}

func TestPromptReservationRefundsWalletWhenLimitedTokenRejectsIncrease(t *testing.T) {
	oldDB, oldRedis := model.DB, common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:prompt-reservation-token?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, common.RedisEnabled = db, false
	t.Cleanup(func() {
		model.DB, common.RedisEnabled = oldDB, oldRedis
		sqlDB, e := db.DB()
		require.NoError(t, e)
		require.NoError(t, sqlDB.Close())
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
	require.NoError(t, db.Create(&model.User{Id: 64, Username: "prompt-token-budget", Quota: 100}).Error)
	require.NoError(t, db.Create(&model.Token{Id: 65, UserId: 64, Key: "prompttokenbudget", RemainQuota: 2}).Error)
	info := &relaycommon.RelayInfo{UserId: 64, TokenId: 65, TokenKey: "prompttokenbudget"}
	session := &BillingSession{relayInfo: info, funding: &WalletFunding{userId: 64, consumed: 20}, preConsumedQuota: 20, tokenConsumed: 20}
	require.Error(t, session.ReserveRequestQuota(30))
	var user model.User
	require.NoError(t, db.First(&user, 64).Error)
	var token model.Token
	require.NoError(t, db.First(&token, 65).Error)
	assert.Equal(t, int64(100), user.Quota)
	assert.Equal(t, 2, token.RemainQuota)
	assert.Equal(t, 20, session.GetPreConsumedQuota())
}
