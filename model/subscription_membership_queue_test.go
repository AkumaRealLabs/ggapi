package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedMembershipQueueUser(t *testing.T, id int) *User {
	t.Helper()
	user := &User{
		Id:       id,
		Username: fmt.Sprintf("test-membership-user-%d", id),
		Group:    "test-base",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

func seedMembershipQueuePlan(t *testing.T, id int, group string, durationSeconds int64) *SubscriptionPlan {
	t.Helper()
	plan := &SubscriptionPlan{
		Id:             id,
		Title:          fmt.Sprintf("test-membership-%d", id),
		PriceAmount:    1,
		DurationUnit:   SubscriptionDurationCustom,
		CustomSeconds:  durationSeconds,
		Enabled:        true,
		MembershipOnly: true,
		UpgradeGroup:   group,
	}
	require.NoError(t, DB.Create(plan).Error)
	return plan
}

func getMembershipQueueSub(t *testing.T, userId int, planId int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("user_id = ? AND plan_id = ?", userId, planId).First(&sub).Error)
	return sub
}

func assertSingleActiveMembership(t *testing.T, userId int) {
	t.Helper()
	var count int64
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND membership_only = ?",
			userId, SubscriptionStatusActive, true).
		Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestPaidMembershipFulfillmentQueuesBehindAdminGrant(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9301)
	paidPlan := seedMembershipQueuePlan(t, 9311, "test-paid-member", 3600)
	adminPlan := seedMembershipQueuePlan(t, 9312, "test-admin-member", 7200)

	order := &SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          paidPlan.Id,
		TradeNo:         "test-paid-membership-queue",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	_, err := InsertPendingSubscriptionOrder(order)
	require.NoError(t, err)
	message, err := AdminBindSubscription(user.Id, adminPlan.Id, "")
	require.NoError(t, err)
	assert.Equal(t, "Subscription activated.", message)
	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, "", PaymentProviderStripe, PaymentMethodStripe))

	active := getMembershipQueueSub(t, user.Id, adminPlan.Id)
	scheduled := getMembershipQueueSub(t, user.Id, paidPlan.Id)
	assert.Equal(t, SubscriptionStatusActive, active.Status)
	assert.Equal(t, SubscriptionStatusScheduled, scheduled.Status)
	assert.Equal(t, active.EndTime, scheduled.StartTime)
	assert.Equal(t, paidPlan.CustomSeconds, scheduled.EndTime-scheduled.StartTime)
	assert.Empty(t, scheduled.PrevUserGroup)
	assertSingleActiveMembership(t, user.Id)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, user.Id).Error)
	assert.Equal(t, adminPlan.UpgradeGroup, storedUser.Group)
}

func TestMembershipWorkerActivatesQueueAndRestoresBaseGroup(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9302)
	firstPlan := seedMembershipQueuePlan(t, 9321, "test-member-a", 3600)
	secondPlan := seedMembershipQueuePlan(t, 9322, "test-member-b", 5400)

	_, err := AdminBindSubscription(user.Id, firstPlan.Id, "")
	require.NoError(t, err)
	message, err := AdminBindSubscription(user.Id, secondPlan.Id, "")
	require.NoError(t, err)
	assert.Equal(t, "Membership queued.", message)

	first := getMembershipQueueSub(t, user.Id, firstPlan.Id)
	second := getMembershipQueueSub(t, user.Id, secondPlan.Id)
	require.Equal(t, SubscriptionStatusActive, first.Status)
	require.Equal(t, SubscriptionStatusScheduled, second.Status)
	duration := second.EndTime - second.StartTime
	now := GetDBTimestamp()
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", first.Id).
		Update("end_time", now-1).Error)
	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", second.Id).
		Updates(map[string]interface{}{
			"start_time": now - 1,
			"end_time":   now - 1 + duration,
		}).Error)

	// Membership expiry is intentionally not handled by the generic expire pass.
	expired, err := ExpireDueSubscriptions(10)
	require.NoError(t, err)
	assert.Zero(t, expired)
	// Expire + activate happen in one per-user transaction.
	// Return value is users processed (including expire-only).
	processed, err := ActivateDueMembershipSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed)

	first = getMembershipQueueSub(t, user.Id, firstPlan.Id)
	second = getMembershipQueueSub(t, user.Id, secondPlan.Id)
	assert.Equal(t, SubscriptionStatusExpired, first.Status)
	assert.Equal(t, SubscriptionStatusActive, second.Status)
	assert.Equal(t, duration, second.EndTime-second.StartTime)
	assert.Equal(t, "test-base", second.PrevUserGroup)
	assertSingleActiveMembership(t, user.Id)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, secondPlan.UpgradeGroup, user.Group)

	processed, err = ActivateDueMembershipSubscriptions(10)
	require.NoError(t, err)
	assert.Zero(t, processed)
	assertSingleActiveMembership(t, user.Id)

	require.NoError(t, DB.Model(&UserSubscription{}).Where("id = ?", second.Id).
		Update("end_time", GetDBTimestamp()-1).Error)
	processed, err = ActivateDueMembershipSubscriptions(10)
	require.NoError(t, err)
	assert.Equal(t, 1, processed) // expire-only still processes the user
	second = getMembershipQueueSub(t, user.Id, secondPlan.Id)
	assert.Equal(t, SubscriptionStatusExpired, second.Status)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, "test-base", user.Group)
}

func TestAdminInvalidatingActiveMembershipActivatesNextImmediately(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9303)
	firstPlan := seedMembershipQueuePlan(t, 9331, "test-member-first", 3600)
	secondPlan := seedMembershipQueuePlan(t, 9332, "test-member-next", 7200)

	_, err := AdminBindSubscription(user.Id, firstPlan.Id, "")
	require.NoError(t, err)
	_, err = AdminBindSubscription(user.Id, secondPlan.Id, "")
	require.NoError(t, err)
	first := getMembershipQueueSub(t, user.Id, firstPlan.Id)
	secondBefore := getMembershipQueueSub(t, user.Id, secondPlan.Id)
	duration := secondBefore.EndTime - secondBefore.StartTime

	resultGroup, err := AdminInvalidateUserSubscription(first.Id)
	require.NoError(t, err)
	assert.Equal(t, secondPlan.UpgradeGroup, resultGroup)

	first = getMembershipQueueSub(t, user.Id, firstPlan.Id)
	second := getMembershipQueueSub(t, user.Id, secondPlan.Id)
	assert.Equal(t, SubscriptionStatusCancelled, first.Status)
	assert.Equal(t, SubscriptionStatusActive, second.Status)
	assert.Equal(t, duration, second.EndTime-second.StartTime)
	assert.Equal(t, "test-base", second.PrevUserGroup)
	assertSingleActiveMembership(t, user.Id)
	require.NoError(t, DB.First(&user, user.Id).Error)
	assert.Equal(t, secondPlan.UpgradeGroup, user.Group)
}

func TestRemovingScheduledMembershipReflowsLaterQueue(t *testing.T) {
	for _, test := range []struct {
		name   string
		remove func(int) error
	}{
		{
			name: "cancel",
			remove: func(id int) error {
				_, err := AdminInvalidateUserSubscription(id)
				return err
			},
		},
		{
			name: "delete",
			remove: func(id int) error {
				_, err := AdminDeleteUserSubscription(id)
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			truncateTables(t)
			user := seedMembershipQueueUser(t, 9400)
			activePlan := seedMembershipQueuePlan(t, 9411, "test-member-active", 3600)
			removedPlan := seedMembershipQueuePlan(t, 9412, "test-member-removed", 1800)
			lastPlan := seedMembershipQueuePlan(t, 9413, "test-member-last", 2700)

			_, err := AdminBindSubscription(user.Id, activePlan.Id, "")
			require.NoError(t, err)
			_, err = AdminBindSubscription(user.Id, removedPlan.Id, "")
			require.NoError(t, err)
			_, err = AdminBindSubscription(user.Id, lastPlan.Id, "")
			require.NoError(t, err)
			active := getMembershipQueueSub(t, user.Id, activePlan.Id)
			removed := getMembershipQueueSub(t, user.Id, removedPlan.Id)
			last := getMembershipQueueSub(t, user.Id, lastPlan.Id)
			duration := last.EndTime - last.StartTime

			require.NoError(t, test.remove(removed.Id))

			last = getMembershipQueueSub(t, user.Id, lastPlan.Id)
			assert.Equal(t, SubscriptionStatusScheduled, last.Status)
			assert.Equal(t, active.EndTime, last.StartTime)
			assert.Equal(t, active.EndTime+duration, last.EndTime)
			assertSingleActiveMembership(t, user.Id)
			require.NoError(t, DB.First(&user, user.Id).Error)
			assert.Equal(t, activePlan.UpgradeGroup, user.Group)
		})
	}
}

func TestScheduledMembershipBlocksPlanTypeChange(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9305)
	activePlan := seedMembershipQueuePlan(t, 9351, "test-member-live", 3600)
	scheduledPlan := seedMembershipQueuePlan(t, 9352, "test-member-scheduled", 3600)

	_, err := AdminBindSubscription(user.Id, activePlan.Id, "")
	require.NoError(t, err)
	_, err = AdminBindSubscription(user.Id, scheduledPlan.Id, "")
	require.NoError(t, err)
	scheduled := getMembershipQueueSub(t, user.Id, scheduledPlan.Id)
	require.Equal(t, SubscriptionStatusScheduled, scheduled.Status)

	membershipOnly := false
	err = AdminUpdateSubscriptionPlanFields(scheduledPlan.Id, map[string]interface{}{
		"title": "test-converted-plan",
	}, &membershipOnly)
	require.ErrorIs(t, err, ErrSubscriptionPlanTypeChangeBlocked)
}

func TestAdminMembershipGrantRespectsScheduledQueueLimit(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9306)
	activePlan := seedMembershipQueuePlan(t, 9360, "test-member-active", 3600)
	_, err := AdminBindSubscription(user.Id, activePlan.Id, "")
	require.NoError(t, err)

	for i := 0; i < MaxScheduledMembershipsPerUser; i++ {
		plan := seedMembershipQueuePlan(t, 9361+i, fmt.Sprintf("test-member-queued-%d", i), 3600)
		_, err = AdminBindSubscription(user.Id, plan.Id, "")
		require.NoError(t, err)
	}
	overflowPlan := seedMembershipQueuePlan(t, 9370, "test-member-overflow", 3600)
	_, err = AdminBindSubscription(user.Id, overflowPlan.Id, "")
	require.ErrorIs(t, err, ErrMembershipQueueFull)

	var scheduledCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND membership_only = ?",
			user.Id, SubscriptionStatusScheduled, true).
		Count(&scheduledCount).Error)
	assert.EqualValues(t, MaxScheduledMembershipsPerUser, scheduledCount)
	assertSingleActiveMembership(t, user.Id)
}

func TestPendingMembershipReservesScheduledQueueSlot(t *testing.T) {
	truncateTables(t)
	user := seedMembershipQueueUser(t, 9307)
	activePlan := seedMembershipQueuePlan(t, 9380, "test-member-active", 3600)
	_, err := AdminBindSubscription(user.Id, activePlan.Id, "")
	require.NoError(t, err)
	for i := 0; i < MaxScheduledMembershipsPerUser-1; i++ {
		plan := seedMembershipQueuePlan(t, 9381+i, fmt.Sprintf("test-member-queued-%d", i), 3600)
		_, err = AdminBindSubscription(user.Id, plan.Id, "")
		require.NoError(t, err)
	}

	paidPlan := seedMembershipQueuePlan(t, 9385, "test-member-paid", 3600)
	order := &SubscriptionOrder{
		UserId:          user.Id,
		PlanId:          paidPlan.Id,
		TradeNo:         "test-membership-reserved-slot",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	_, err = InsertPendingSubscriptionOrder(order)
	require.NoError(t, err)

	adminPlan := seedMembershipQueuePlan(t, 9386, "test-member-admin-overflow", 3600)
	_, err = AdminBindSubscription(user.Id, adminPlan.Id, "")
	require.ErrorIs(t, err, ErrMembershipQueueFull)
	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, "", PaymentProviderStripe, PaymentMethodStripe))

	var scheduledCount int64
	require.NoError(t, DB.Model(&UserSubscription{}).
		Where("user_id = ? AND status = ? AND membership_only = ?",
			user.Id, SubscriptionStatusScheduled, true).
		Count(&scheduledCount).Error)
	assert.EqualValues(t, MaxScheduledMembershipsPerUser, scheduledCount)
	assertSingleActiveMembership(t, user.Id)
}
