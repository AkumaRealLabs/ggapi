package model

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

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
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
		DB.Exec("DELETE FROM users")
		DB.Exec("DELETE FROM logs")
	})

	user := &User{Username: "redeem-user", Password: "password", Status: common.UserStatusEnabled, Quota: 0}
	require.NoError(t, DB.Create(user).Error)

	key = "10000000000000000000000000000001"
	redemption := &Redemption{
		Name:        "redeem-test",
		Key:         key,
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       quota,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)
	return user.Id, key
}

func setupAffiliateRedemptionTest(t *testing.T) {
	t.Helper()
	setupAffiliateTest(t)
	require.NoError(t, DB.AutoMigrate(&Redemption{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&Redemption{}).Error)
	})
}

func createAffiliateRedemption(t *testing.T, quota int) *Redemption {
	t.Helper()
	redemption := &Redemption{
		Name:        "affiliate-redemption",
		Key:         "20000000000000000000000000000001",
		Status:      common.RedemptionCodeStatusEnabled,
		Quota:       quota,
		CreatedTime: common.GetTimestamp(),
	}
	require.NoError(t, DB.Create(redemption).Error)
	return redemption
}

func TestRedeemCreditsQuotaExactlyOnce(t *testing.T) {
	userId, key := setupRedeemFixture(t, 500)

	quota, err := Redeem(key, userId)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)

	var user User
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)

	var redemption Redemption
	require.NoError(t, DB.First(&redemption, "name = ?", "redeem-test").Error)
	assert.Equal(t, common.RedemptionCodeStatusUsed, redemption.Status)
	assert.Equal(t, userId, redemption.UsedUserId)

	// Redeeming the same code again must fail and must not credit quota.
	_, err = Redeem(key, userId)
	require.Error(t, err)
	require.NoError(t, DB.First(&user, "id = ?", userId).Error)
	assert.Equal(t, 500, user.Quota)
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
	assert.Equal(t, 300, user.Quota, "quota must be credited exactly once")
}

func TestRedeemCreatesAffiliateCommissionExactlyOnce(t *testing.T) {
	setupAffiliateRedemptionTest(t)
	createAffiliateTestUser(t, 1, "inviter", "redeem-inviter", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "redeem-invitee", 1, nil)
	redemption := createAffiliateRedemption(t, 500)

	quota, err := Redeem(redemption.Key, 2)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)
	_, err = Redeem(redemption.Key, 2)
	require.Error(t, err)

	var inviter, invitee User
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	assert.Equal(t, 500, invitee.Quota)
	assert.Equal(t, 50, inviter.AffQuota)
	assert.Equal(t, 50, inviter.AffHistoryQuota)

	var commissions []AffiliateCommission
	require.NoError(t, DB.Find(&commissions).Error)
	require.Len(t, commissions, 1)
	commission := commissions[0]
	assert.Equal(t, AffiliateCommissionSourceRedemption, commission.SourceType)
	assert.Equal(t, redemption.Id, commission.SourceId)
	assert.Equal(t, fmt.Sprintf("RED-%d", redemption.Id), commission.SourceOrderNo)
	assert.NotContains(t, commission.SourceOrderNo, redemption.Key)
	assert.Empty(t, commission.PaymentProvider)
	assert.Equal(t, 500, commission.BaseQuota)
	assert.Equal(t, 10.0, commission.CommissionRate)
	assert.Equal(t, 50, commission.CommissionQuota)

	items, total, err := ListAffiliateCommissions(
		&common.PageInfo{Page: 1, PageSize: 20},
		AffiliateCommissionFilters{SourceType: AffiliateCommissionSourceRedemption},
	)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, redemption.Id, items[0].SourceId)
}

func TestAffiliateCommissionFailureRollsBackRedemption(t *testing.T) {
	setupAffiliateRedemptionTest(t)
	createAffiliateTestUser(t, 1, "inviter", "redeem-rollback", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "redeem-invitee", 1, nil)
	redemption := createAffiliateRedemption(t, 500)
	require.NoError(t, DB.Create(&AffiliateCommission{
		InviterId:       1,
		InviteeId:       2,
		SourceType:      AffiliateCommissionSourceRedemption,
		SourceId:        redemption.Id,
		SourceOrderNo:   fmt.Sprintf("RED-%d", redemption.Id),
		BaseQuota:       redemption.Quota,
		CommissionRate:  10,
		CommissionQuota: 0,
	}).Error)

	_, err := Redeem(redemption.Key, 2)
	require.Error(t, err)

	var gotRedemption Redemption
	var inviter, invitee User
	require.NoError(t, DB.First(&gotRedemption, redemption.Id).Error)
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, gotRedemption.Status)
	assert.Zero(t, gotRedemption.UsedUserId)
	assert.Zero(t, gotRedemption.RedeemedTime)
	assert.Zero(t, invitee.Quota)
	assert.Zero(t, inviter.AffQuota)
	assert.Zero(t, inviter.AffHistoryQuota)
}

func TestRedemptionSettlementRejectsConcurrentAffiliateRebind(t *testing.T) {
	setupAffiliateRedemptionTest(t)
	createAffiliateTestUser(t, 1, "old-inviter", "old-inviter", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "rebind-invitee", 1, nil)
	createAffiliateTestUser(t, 3, "new-inviter", "new-inviter", 0, nil)
	redemption := createAffiliateRedemption(t, 500)

	var inviteeSnapshot User
	require.NoError(t, DB.Select("id", "inviter_id").First(&inviteeSnapshot, 2).Error)
	assert.Equal(t, 1, inviteeSnapshot.InviterId)

	// SQLite serializes writers, so stage the production interleaving directly:
	// a concurrent rebind commits after the snapshot and before settlement locks.
	rebindResult := make(chan error, 1)
	go func() {
		rebindResult <- SetAffiliateUserConfig(2, "rebind-invitee", nil, 3)
	}()
	require.NoError(t, <-rebindResult)

	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Redemption{}).Where("id = ?", redemption.Id).Updates(map[string]any{
			"redeemed_time": common.GetTimestamp(),
			"status":        common.RedemptionCodeStatusUsed,
			"used_user_id":  2,
		}).Error; err != nil {
			return err
		}
		if _, _, err := SettleAffiliateCommissionTx(
			tx,
			2,
			AffiliateCommissionSourceRedemption,
			redemption.Id,
			fmt.Sprintf("RED-%d", redemption.Id),
			"",
			redemption.Quota,
			inviteeSnapshot.InviterId,
		); err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", 2).
			Update("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
	})
	require.ErrorIs(t, err, ErrAffiliateRelationChanged)

	var gotRedemption Redemption
	var oldInviter, newInviter, invitee User
	require.NoError(t, DB.First(&gotRedemption, redemption.Id).Error)
	require.NoError(t, DB.First(&oldInviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	require.NoError(t, DB.First(&newInviter, 3).Error)
	assert.Equal(t, common.RedemptionCodeStatusEnabled, gotRedemption.Status)
	assert.Zero(t, gotRedemption.UsedUserId)
	assert.Zero(t, invitee.Quota)
	assert.Equal(t, 3, invitee.InviterId)
	assert.Zero(t, oldInviter.AffQuota)
	assert.Zero(t, newInviter.AffQuota)
	var commissionCount int64
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&commissionCount).Error)
	assert.Zero(t, commissionCount)

	quota, err := Redeem(redemption.Key, 2)
	require.NoError(t, err)
	assert.Equal(t, 500, quota)
	require.NoError(t, DB.First(&oldInviter, 1).Error)
	require.NoError(t, DB.First(&newInviter, 3).Error)
	assert.Zero(t, oldInviter.AffQuota)
	assert.Equal(t, 50, newInviter.AffQuota)
	var commission AffiliateCommission
	require.NoError(t, DB.First(&commission).Error)
	assert.Equal(t, 3, commission.InviterId)
	assert.Equal(t, AffiliateCommissionSourceRedemption, commission.SourceType)
}

func TestRedeemSkipsIneligibleAffiliateCommission(t *testing.T) {
	zeroRate := 0.0
	tests := []struct {
		name                string
		inviterId           int
		inviterStatus       int
		customRate          *float64
		inviterAffQuota     int
		inviterAffHistory   int
		complianceConfirmed bool
	}{
		{
			name:                "no inviter",
			inviterStatus:       common.UserStatusEnabled,
			complianceConfirmed: true,
		},
		{
			name:                "disabled inviter",
			inviterId:           1,
			inviterStatus:       common.UserStatusDisabled,
			complianceConfirmed: true,
		},
		{
			name:                "zero commission rate",
			inviterId:           1,
			inviterStatus:       common.UserStatusEnabled,
			customRate:          &zeroRate,
			complianceConfirmed: true,
		},
		{
			name:                "affiliate balance overflow",
			inviterId:           1,
			inviterStatus:       common.UserStatusEnabled,
			inviterAffQuota:     math.MaxInt32 - 10,
			inviterAffHistory:   math.MaxInt32 - 10,
			complianceConfirmed: true,
		},
		{
			name:          "unconfirmed compliance",
			inviterId:     1,
			inviterStatus: common.UserStatusEnabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupAffiliateRedemptionTest(t)
			if tt.inviterId > 0 {
				createAffiliateTestUser(t, tt.inviterId, "inviter", "skip-inviter", 0, tt.customRate)
				require.NoError(t, DB.Model(&User{}).Where("id = ?", tt.inviterId).Updates(map[string]any{
					"status":      tt.inviterStatus,
					"aff_quota":   tt.inviterAffQuota,
					"aff_history": tt.inviterAffHistory,
				}).Error)
			}
			createAffiliateTestUser(t, 2, "invitee", "skip-invitee", tt.inviterId, nil)
			operation_setting.GetPaymentSetting().ComplianceConfirmed = tt.complianceConfirmed
			redemption := createAffiliateRedemption(t, 500)

			quota, err := Redeem(redemption.Key, 2)
			require.NoError(t, err)
			assert.Equal(t, 500, quota)

			var invitee User
			require.NoError(t, DB.First(&invitee, 2).Error)
			assert.Equal(t, 500, invitee.Quota)
			var commissionCount int64
			require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&commissionCount).Error)
			assert.Zero(t, commissionCount)
			if tt.inviterId > 0 {
				var inviter User
				require.NoError(t, DB.First(&inviter, tt.inviterId).Error)
				assert.Equal(t, tt.inviterAffQuota, inviter.AffQuota)
				assert.Equal(t, tt.inviterAffHistory, inviter.AffHistoryQuota)
			}
		})
	}
}
