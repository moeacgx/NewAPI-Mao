package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDeleteRedemptionsByIDsArchivesSelectedCodes(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})
	codes := []Redemption{
		{Name: "batch-first", Key: "10000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled},
		{Name: "batch-second", Key: "10000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled},
	}
	require.NoError(t, DB.Create(&codes).Error)

	deleted, err := DeleteRedemptionsByIDs([]int{codes[0].Id, codes[1].Id, 999999})
	require.NoError(t, err)
	assert.Equal(t, []int{codes[0].Id, codes[1].Id}, deleted)

	var archived []Redemption
	require.NoError(t, DB.Unscoped().Where("id IN ?", []int{codes[0].Id, codes[1].Id}).Find(&archived).Error)
	require.Len(t, archived, 2)
	for _, code := range archived {
		assert.True(t, code.DeletedAt.Valid)
	}
}

func TestSearchRedemptionsFiltersAndPaginates(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	now := common.GetTimestamp()
	redemptions := []Redemption{
		{Id: 1, Name: "alpha-active", Key: "00000000000000000000000000000001", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: 0},
		{Id: 2, Name: "alpha-future", Key: "00000000000000000000000000000002", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now + 3600},
		{Id: 3, Name: "alpha-expired", Key: "00000000000000000000000000000003", Status: common.RedemptionCodeStatusEnabled, ExpiredTime: now - 10},
		{Id: 4, Name: "beta-disabled", Key: "00000000000000000000000000000004", Status: common.RedemptionCodeStatusDisabled, ExpiredTime: 0},
		{Id: 5, Name: "beta-used", Key: "00000000000000000000000000000005", Status: common.RedemptionCodeStatusUsed, ExpiredTime: 0},
	}
	require.NoError(t, DB.Create(&redemptions).Error)

	tests := []struct {
		name      string
		keyword   string
		status    string
		startIdx  int
		num       int
		wantTotal int64
		wantIds   []int
	}{
		{
			name:      "no filters returns all rows",
			num:       10,
			wantTotal: 5,
			wantIds:   []int{5, 4, 3, 2, 1},
		},
		{
			name:      "keyword filters by name prefix",
			keyword:   "alpha",
			num:       10,
			wantTotal: 3,
			wantIds:   []int{3, 2, 1},
		},
		{
			name:      "enabled status excludes expired rows",
			status:    "1",
			num:       10,
			wantTotal: 2,
			wantIds:   []int{2, 1},
		},
		{
			name:      "expired status returns enabled expired rows",
			status:    "expired",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{3},
		},
		{
			name:      "disabled status",
			status:    "2",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{4},
		},
		{
			name:      "used status",
			status:    "3",
			num:       10,
			wantTotal: 1,
			wantIds:   []int{5},
		},
		{
			name:      "pagination keeps unpaged total",
			startIdx:  1,
			num:       2,
			wantTotal: 5,
			wantIds:   []int{4, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, total, err := SearchRedemptions(tt.keyword, tt.status, tt.startIdx, tt.num)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			gotIds := make([]int, 0, len(rows))
			for _, row := range rows {
				gotIds = append(gotIds, row.Id)
			}
			assert.Equal(t, tt.wantIds, gotIds)
		})
	}
}

func setupRedeemFixture(t *testing.T, quota int) (userId int, key string) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Redemption{}, &RedemptionUsage{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionUsage{}).Error)
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&RedemptionUsage{}).Error)
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0, AffCode: "redeem-aff-1"}
	require.NoError(t, DB.Create(user).Error)

	key = "10000000000000000000000000000001"
	redemption := &Redemption{
		Name:        "redeem-test",
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       int64(quota),
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)
	return user.Id, key
}

func TestRedeemAllowsDifferentUsersUntilPerCodeLimit(t *testing.T) {
	firstUserId, key := setupRedeemFixture(t, 500)
	secondUser := &User{Username: "redeem-second-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0, AffCode: "redeem-aff-2"}
	require.NoError(t, DB.Create(secondUser).Error)
	require.NoError(t, DB.Model(&Redemption{}).Where("key = ?", key).Updates(map[string]interface{}{
		"max_redeem_count": 2,
	}).Error)

	quota, err := Redeem(key, firstUserId)
	require.NoError(t, err)
	assert.Equal(t, int64(500), quota)

	quota, err = Redeem(key, secondUser.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(500), quota)

	_, err = Redeem(key, secondUser.Id)
	require.Error(t, err, "the same user cannot redeem one code twice")
	_, err = Redeem(key, firstUserId)
	require.Error(t, err, "the code is unavailable after its final allowed redemption")

	var first, second User
	require.NoError(t, DB.First(&first, "id = ?", firstUserId).Error)
	require.NoError(t, DB.First(&second, "id = ?", secondUser.Id).Error)
	assert.Equal(t, int64(500), first.Quota)
	assert.Equal(t, int64(500), second.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, 2, redemption.MaxRedeemCount)
	assert.Equal(t, 2, redemption.RedeemedCount)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)

	var usageCount int64
	require.NoError(t, DB.Model(&RedemptionUsage{}).Where("redemption_id = ?", redemption.Id).Count(&usageCount).Error)
	assert.EqualValues(t, 2, usageCount)
}

func TestRedemptionInsertAndUpdatePersistRedeemLimit(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})

	redemption := &Redemption{Name: "limit-persistence", Key: "10000000000000000000000000000009", Status: common.RedemptionCodeStatusEnabled, Quota: 100, MaxRedeemCount: 7}
	require.NoError(t, redemption.Insert())
	var stored Redemption
	require.NoError(t, DB.First(&stored, "id = ?", redemption.Id).Error)
	assert.Equal(t, 7, stored.MaxRedeemCount)

	stored.MaxRedeemCount = 9
	require.NoError(t, stored.Update())
	require.NoError(t, DB.First(&stored, "id = ?", redemption.Id).Error)
	assert.Equal(t, 9, stored.MaxRedeemCount)
}

func TestRedeemAllowsOnlyOneConcurrentFinalSlot(t *testing.T) {
	firstUserId, key := setupRedeemFixture(t, 300)
	secondUser := &User{Username: "redeem-concurrent-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0, AffCode: "redeem-aff-3"}
	require.NoError(t, DB.Create(secondUser).Error)
	require.NoError(t, DB.Model(&Redemption{}).Where("key = ?", key).Updates(map[string]interface{}{
		"max_redeem_count": 2,
	}).Error)
	_, err := Redeem(key, firstUserId)
	require.NoError(t, err)

	const goroutines = 2
	users := []int{firstUserId, secondUser.Id}
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := range users {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, users[idx]); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "only one redemption may take the last slot")

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "key = ?", key).Error)
	assert.Equal(t, 2, redemption.RedeemedCount)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500)

	quota, err := Redeem(key, userId)
	require.NoError(t, err)
	assert.Equal(t, int64(500), quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, int64(500), user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, int64(500), user.Quota)
}

// Exactly one of several concurrent redeems of the same code may win, and
// quota must be credited exactly once.
func TestRedeemConcurrentSingleSuccess(t *testing.T) {
	userId, key := setupRedeemFixture(t, 300)

	const goroutines = 5
	successes := make([]bool, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if _, err := Redeem(key, userId); err == nil {
				successes[idx] = true
			}
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, ok := range successes {
		if ok {
			successCount++
		}
	}
	assert.Equal(t, 1, successCount, "exactly one concurrent redeem should succeed")

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.EqualValues(t, 300, user.Quota, "quota must be credited exactly once")
}
