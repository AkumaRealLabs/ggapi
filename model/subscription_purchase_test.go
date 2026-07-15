package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSubscriptionPurchaseTest(t *testing.T, maxPurchases int) (*User, *SubscriptionPlan) {
	t.Helper()
	truncateTables(t)

	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 10
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
	})

	user := &User{
		Id:       9101,
		Username: "test-purchaser",
		Quota:    100,
		Group:    "test-basic",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(user).Error)

	plan := &SubscriptionPlan{
		Id:                 9201,
		Title:              "test-membership",
		PriceAmount:        2,
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		MembershipOnly:     true,
		UpgradeGroup:       "test-member",
		MaxPurchasePerUser: maxPurchases,
	}
	require.NoError(t, DB.Create(plan).Error)
	return user, plan
}

func seedSubscriptionPurchaseBlocker(t *testing.T, blocker string, user *User, plan *SubscriptionPlan) {
	t.Helper()
	now := GetDBTimestamp()

	switch blocker {
	case "disabled-plan":
		require.NoError(t, DB.Model(plan).Update("enabled", false).Error)
	case "active-membership":
		require.NoError(t, DB.Create(&UserSubscription{
			UserId:         user.Id,
			PlanId:         plan.Id + 1,
			StartTime:      now - 60,
			EndTime:        now + 3600,
			Status:         SubscriptionStatusActive,
			MembershipOnly: true,
		}).Error)
	case "full-membership-queue":
		for i := 0; i < MaxScheduledMembershipsPerUser; i++ {
			startTime := now + int64((i+1)*3600)
			require.NoError(t, DB.Create(&UserSubscription{
				UserId:         user.Id,
				PlanId:         plan.Id + 100 + i,
				StartTime:      startTime,
				EndTime:        startTime + 3600,
				Status:         SubscriptionStatusScheduled,
				MembershipOnly: true,
			}).Error)
		}
	case "pending-order":
		require.NoError(t, DB.Create(&SubscriptionOrder{
			UserId:          user.Id,
			PlanId:          plan.Id,
			TradeNo:         "test-pending-order",
			PaymentMethod:   PaymentMethodStripe,
			PaymentProvider: PaymentProviderStripe,
			Status:          common.TopUpStatusPending,
			CreateTime:      now,
		}).Error)
	case "disabled-unset-token-group":
		require.NoError(t, DB.Create(&Token{
			UserId:      user.Id,
			Key:         "test-disabled-unset-token-group",
			Name:        "test-disabled-unset-token-group",
			Status:      common.TokenStatusDisabled,
			CreatedTime: now,
			Group:       "",
		}).Error)
	case "unset-token-group":
		require.NoError(t, DB.Create(&Token{
			UserId:      user.Id,
			Key:         "test-unset-token-group",
			Name:        "test-unset-token-group",
			Status:      common.TokenStatusEnabled,
			CreatedTime: now,
			Group:       "",
		}).Error)
	case "purchase-limit":
		require.NoError(t, DB.Create(&UserSubscription{
			UserId:         user.Id,
			PlanId:         plan.Id,
			StartTime:      now - 7200,
			EndTime:        now - 3600,
			Status:         "expired",
			MembershipOnly: true,
		}).Error)
	default:
		require.FailNow(t, "unknown purchase blocker", blocker)
	}
}

func countSubscriptionPurchaseRows(t *testing.T, model any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(model).Count(&count).Error)
	return count
}

func TestInsertPendingSubscriptionOrderValidatesBeforeCreatingOrder(t *testing.T) {
	tests := []struct {
		name         string
		blocker      string
		maxPurchases int
		expectedErr  error
	}{
		{
			name:        "disabled plan",
			blocker:     "disabled-plan",
			expectedErr: ErrSubscriptionPlanDisabled,
		},
		{
			name:        "full membership queue",
			blocker:     "full-membership-queue",
			expectedErr: ErrMembershipQueueFull,
		},
		{
			name:        "same plan pending order",
			blocker:     "pending-order",
			expectedErr: ErrMembershipPendingOrderExists,
		},
		{
			name:        "unset token group",
			blocker:     "unset-token-group",
			expectedErr: ErrMembershipTokenGroupUnset,
		},
		{
			name:         "lifetime purchase limit",
			blocker:      "purchase-limit",
			maxPurchases: 1,
			expectedErr:  ErrSubscriptionPurchaseLimit,
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, plan := setupSubscriptionPurchaseTest(t, test.maxPurchases)
			seedSubscriptionPurchaseBlocker(t, test.blocker, user, plan)
			ordersBefore := countSubscriptionPurchaseRows(t, &SubscriptionOrder{})

			_, err := InsertPendingSubscriptionOrder(&SubscriptionOrder{
				UserId:          user.Id,
				PlanId:          plan.Id,
				TradeNo:         fmt.Sprintf("test-new-order-%d", index),
				PaymentMethod:   PaymentMethodStripe,
				PaymentProvider: PaymentProviderStripe,
				Status:          common.TopUpStatusPending,
			})

			require.ErrorIs(t, err, test.expectedErr)
			assert.Equal(t, ordersBefore, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
		})
	}
}

func TestBalancePurchaseValidationDoesNotChargeOrCreateRows(t *testing.T) {
	tests := []struct {
		name         string
		blocker      string
		maxPurchases int
		expectedErr  error
	}{
		{
			name:        "disabled plan",
			blocker:     "disabled-plan",
			expectedErr: ErrSubscriptionPlanDisabled,
		},
		{
			name:        "full membership queue",
			blocker:     "full-membership-queue",
			expectedErr: ErrMembershipQueueFull,
		},
		{
			name:        "same plan pending order",
			blocker:     "pending-order",
			expectedErr: ErrMembershipPendingOrderExists,
		},
		{
			name:        "unset token group",
			blocker:     "unset-token-group",
			expectedErr: ErrMembershipTokenGroupUnset,
		},
		{
			name:         "lifetime purchase limit",
			blocker:      "purchase-limit",
			maxPurchases: 1,
			expectedErr:  ErrSubscriptionPurchaseLimit,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user, plan := setupSubscriptionPurchaseTest(t, test.maxPurchases)
			seedSubscriptionPurchaseBlocker(t, test.blocker, user, plan)
			ordersBefore := countSubscriptionPurchaseRows(t, &SubscriptionOrder{})
			subscriptionsBefore := countSubscriptionPurchaseRows(t, &UserSubscription{})

			_, err := PurchaseSubscriptionWithBalance(user.Id, plan.Id)

			require.ErrorIs(t, err, test.expectedErr)
			assert.Equal(t, ordersBefore, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
			assert.Equal(t, subscriptionsBefore, countSubscriptionPurchaseRows(t, &UserSubscription{}))

			var storedUser User
			require.NoError(t, DB.First(&storedUser, user.Id).Error)
			assert.Equal(t, 100, storedUser.Quota)
			assert.Equal(t, "test-basic", storedUser.Group)
		})
	}
}

func TestActiveMembershipAllowsSelfServiceRenewal(t *testing.T) {
	t.Run("external checkout", func(t *testing.T) {
		user, plan := setupSubscriptionPurchaseTest(t, 0)
		seedSubscriptionPurchaseBlocker(t, "active-membership", user, plan)

		_, err := InsertPendingSubscriptionOrder(&SubscriptionOrder{
			UserId:          user.Id,
			PlanId:          plan.Id,
			TradeNo:         "test-active-membership-renewal",
			PaymentMethod:   PaymentMethodStripe,
			PaymentProvider: PaymentProviderStripe,
			Status:          common.TopUpStatusPending,
		})

		require.NoError(t, err)
		assert.EqualValues(t, 1, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
	})

	t.Run("balance purchase", func(t *testing.T) {
		user, plan := setupSubscriptionPurchaseTest(t, 0)
		seedSubscriptionPurchaseBlocker(t, "active-membership", user, plan)

		status, err := PurchaseSubscriptionWithBalance(user.Id, plan.Id)
		require.NoError(t, err)
		assert.Equal(t, SubscriptionStatusScheduled, status)

		var renewed UserSubscription
		require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", user.Id, plan.Id).First(&renewed).Error)
		assert.Equal(t, SubscriptionStatusScheduled, renewed.Status)
		assert.Empty(t, renewed.PrevUserGroup)
		var storedUser User
		require.NoError(t, DB.First(&storedUser, user.Id).Error)
		assert.Equal(t, 80, storedUser.Quota)
		assert.Equal(t, "test-basic", storedUser.Group)
	})
}

func TestExpiredMembershipAllowsBalancePurchase(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&UserSubscription{
		UserId:         user.Id,
		PlanId:         plan.Id + 1,
		StartTime:      now - 7200,
		EndTime:        now - 3600,
		Status:         "active",
		MembershipOnly: true,
	}).Error)

	deletedToken := &Token{
		UserId:      user.Id,
		Key:         "test-deleted-unset-token",
		Name:        "test-deleted-unset-token",
		Status:      common.TokenStatusEnabled,
		CreatedTime: now,
		Group:       "",
	}
	require.NoError(t, DB.Create(deletedToken).Error)
	require.NoError(t, DB.Delete(deletedToken).Error)

	status, err := PurchaseSubscriptionWithBalance(user.Id, plan.Id)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusActive, status)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 80, storedUser.Quota)
	assert.Equal(t, "test-member", storedUser.Group)
	assert.EqualValues(t, 1, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
	assert.EqualValues(t, 2, countSubscriptionPurchaseRows(t, &UserSubscription{}))
}

func TestExpiredPendingOrderAllowsAnotherCheckout(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-expired-pending-order",
		PaymentMethod:   PaymentMethodCreem,
		PaymentProvider: PaymentProviderCreem,
		Status:          common.TopUpStatusPending,
		CreateTime:      now,
	}).Error)
	require.NoError(t, ExpireSubscriptionOrder("test-expired-pending-order", PaymentProviderCreem))

	_, err := InsertPendingSubscriptionOrder(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-retry-order",
		PaymentMethod:   PaymentMethodCreem,
		PaymentProvider: PaymentProviderCreem,
		Status:          common.TopUpStatusPending,
	})

	require.NoError(t, err)
	assert.EqualValues(t, 2, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
}

func TestMembershipPendingOrderBlocksDifferentMembershipPlan(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	otherPlan := &SubscriptionPlan{
		Id:                 plan.Id + 1,
		Title:              "test-other-membership",
		PriceAmount:        3,
		DurationUnit:       SubscriptionDurationMonth,
		DurationValue:      1,
		Enabled:            true,
		MembershipOnly:     true,
		UpgradeGroup:       "test-other-member",
		MaxPurchasePerUser: 0,
	}
	require.NoError(t, DB.Create(otherPlan).Error)
	seedSubscriptionPurchaseBlocker(t, "pending-order", user, plan)

	_, err := InsertPendingSubscriptionOrder(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          otherPlan.Id,
		TradeNo:         "test-cross-plan-order",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	})

	require.ErrorIs(t, err, ErrMembershipPendingOrderExists)
	assert.EqualValues(t, 1, countSubscriptionPurchaseRows(t, &SubscriptionOrder{}))
	pendingCount, err := CountPendingMembershipOrdersByUser(user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 1, pendingCount)
}

func TestDisabledUnsetGroupTokenAllowsMembershipPurchase(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	seedSubscriptionPurchaseBlocker(t, "disabled-unset-token-group", user, plan)

	status, err := PurchaseSubscriptionWithBalance(user.Id, plan.Id)
	require.NoError(t, err)
	assert.Equal(t, SubscriptionStatusActive, status)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, 80, storedUser.Quota)
	assert.Equal(t, "test-member", storedUser.Group)
}

func TestCalcSubscriptionBalanceQuotaUsesSafeCeiling(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = oldQuotaPerUnit
	})

	common.QuotaPerUnit = 10
	quota, err := calcSubscriptionBalanceQuota(1.01)
	require.NoError(t, err)
	assert.Equal(t, 11, quota)

	common.QuotaPerUnit = 2
	quota, err = calcSubscriptionBalanceQuota(float64(common.MaxQuota))
	require.ErrorIs(t, err, ErrSubscriptionPlanPriceRange)
	assert.Zero(t, quota)
}

func TestStalePendingMembershipOrderDoesNotBlockCheckout(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-stale-pending-order",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      now - PendingSubscriptionOrderTTLSeconds - 10,
	}).Error)

	_, err := InsertPendingSubscriptionOrder(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-after-stale-pending",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	})
	require.NoError(t, err)

	var stale SubscriptionOrder
	require.NoError(t, DB.Where("trade_no = ?", "test-stale-pending-order").First(&stale).Error)
	assert.Equal(t, common.TopUpStatusExpired, stale.Status)

	count, err := CountPendingMembershipOrdersByUser(user.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
}

func TestEnsureMembershipTokenGroupAllowed(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	require.NoError(t, EnsureMembershipTokenGroupAllowed(user.Id, "", true))

	_, err := AdminBindSubscription(user.Id, plan.Id, "")
	require.NoError(t, err)

	require.ErrorIs(t, EnsureMembershipTokenGroupAllowed(user.Id, "", true), ErrMembershipTokenGroupUnset)
	require.NoError(t, EnsureMembershipTokenGroupAllowed(user.Id, "default", true))
	require.NoError(t, EnsureMembershipTokenGroupAllowed(user.Id, "", false))

	// Pending membership checkout must also block empty-group keys: paid
	// fulfillment skips the purchase-time token guard.
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ?", user.Id).
		Update("status", SubscriptionStatusCancelled).Error)
	require.NoError(t, DB.Create(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-pending-token-guard",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      GetDBTimestamp(),
	}).Error)
	require.ErrorIs(t, EnsureMembershipTokenGroupAllowed(user.Id, "", true), ErrMembershipTokenGroupUnset)
}

func TestExpireStalePendingDoesNotClobberCompletedOrders(t *testing.T) {
	user, plan := setupSubscriptionPurchaseTest(t, 0)
	now := GetDBTimestamp()
	require.NoError(t, DB.Create(&SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          plan.Id,
		TradeNo:         "test-race-pending-to-success",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
		CreateTime:      now - PendingSubscriptionOrderTTLSeconds - 10,
	}).Error)

	var orderIDs []int
	require.NoError(t, DB.Model(&SubscriptionOrder{}).
		Where("trade_no = ?", "test-race-pending-to-success").
		Pluck("id", &orderIDs).Error)
	require.Len(t, orderIDs, 1)

	// Simulate concurrent fulfillment after the stale select but before update.
	require.NoError(t, DB.Model(&SubscriptionOrder{}).Where("id = ?", orderIDs[0]).
		Updates(map[string]interface{}{
			"status":        common.TopUpStatusSuccess,
			"complete_time": now,
		}).Error)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		return expireStalePendingSubscriptionOrdersTx(tx, user.Id, plan, now)
	}))

	var order SubscriptionOrder
	require.NoError(t, DB.Where("trade_no = ?", "test-race-pending-to-success").First(&order).Error)
	assert.Equal(t, common.TopUpStatusSuccess, order.Status)
}
