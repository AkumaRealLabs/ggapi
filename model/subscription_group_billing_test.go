package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedGroupBillingPlan(t *testing.T, plan *SubscriptionPlan) {
	t.Helper()
	require.NoError(t, DB.Create(plan).Error)
}

func seedGroupBillingSub(t *testing.T, sub *UserSubscription) {
	t.Helper()
	require.NoError(t, DB.Create(sub).Error)
}

func getGroupBillingSub(t *testing.T, id int) UserSubscription {
	t.Helper()
	var sub UserSubscription
	require.NoError(t, DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

func seedActiveSub(t *testing.T, id, userId, planId int, upgradeGroup string, total, used int64, allowOverflow bool) {
	t.Helper()
	now := GetDBTimestamp()
	seedGroupBillingSub(t, &UserSubscription{
		Id:                  id,
		UserId:              userId,
		PlanId:              planId,
		AmountTotal:         total,
		AmountUsed:          used,
		StartTime:           now - 3600,
		EndTime:             now + 30*24*3600,
		Status:              "active",
		UpgradeGroup:        upgradeGroup,
		AllowWalletOverflow: allowOverflow,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
}

// A: vip subscription + default group request must not consume vip quota.
func TestPreConsumeSkipsMismatchedUpgradeGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8101, Title: "VIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8201, 201, 8101, "vip", 1000, 0, true)

	_, err := PreConsumeUserSubscription("req-a", 201, "gpt-4", 0, 100, "default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active subscription")
	assert.EqualValues(t, 0, getGroupBillingSub(t, 8201).AmountUsed)
}

// B: vip subscription + vip group request pre-consumes correctly.
func TestPreConsumeMatchesUpgradeGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8102, Title: "VIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8202, 202, 8102, "vip", 1000, 0, true)

	res, err := PreConsumeUserSubscription("req-b", 202, "gpt-4", 0, 150, "vip")
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 8202, res.UserSubscriptionId)
	assert.EqualValues(t, 150, res.PreConsumed)
	assert.EqualValues(t, 0, res.AmountUsedBefore)
	assert.EqualValues(t, 150, res.AmountUsedAfter)
	assert.EqualValues(t, 150, getGroupBillingSub(t, 8202).AmountUsed)
}

// C: empty UpgradeGroup remains usable for any group (backward compatible).
func TestPreConsumeEmptyUpgradeGroupAllowsAnyGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8103, Title: "Legacy", PriceAmount: 5, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 500, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8203, 203, 8103, "", 500, 0, true)

	for i, group := range []string{"default", "vip", "pro"} {
		reqId := "req-c-" + group
		res, err := PreConsumeUserSubscription(reqId, 203, "gpt-4", 0, 10, group)
		require.NoError(t, err, "group=%s", group)
		assert.Equal(t, 8203, res.UserSubscriptionId)
		assert.EqualValues(t, int64(10*(i+1)), getGroupBillingSub(t, 8203).AmountUsed)
	}
}

// D: multi-sub — skip mismatched, pick matching with enough quota.
func TestPreConsumeSelectsMatchingSubscriptionAmongMany(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8104, Title: "Mix", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	// earlier end_time first in Order — put mismatched first so filter is exercised
	seedActiveSub(t, 8204, 204, 8104, "vip", 1000, 0, true)
	seedActiveSub(t, 8205, 204, 8104, "pro", 1000, 950, true) // matching but insufficient (remain 50)
	seedActiveSub(t, 8206, 204, 8104, "pro", 1000, 0, true)   // matching and enough

	// Order is end_time asc, id asc → walk 8204 skip, 8205 insufficient, 8206 take
	res, err := PreConsumeUserSubscription("req-d", 204, "gpt-4", 0, 100, "pro")
	require.NoError(t, err)
	assert.Equal(t, 8206, res.UserSubscriptionId)
	assert.EqualValues(t, 0, getGroupBillingSub(t, 8204).AmountUsed)
	assert.EqualValues(t, 950, getGroupBillingSub(t, 8205).AmountUsed)
	assert.EqualValues(t, 100, getGroupBillingSub(t, 8206).AmountUsed)
}

// E: matching sub allow_wallet_overflow=false + insufficient → quota insufficient (not "no active").
func TestPreConsumeMatchingInsufficientReturnsQuotaError(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8105, Title: "Strict", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 100, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8207, 205, 8105, "vip", 100, 90, false)

	_, err := PreConsumeUserSubscription("req-e", 205, "gpt-4", 0, 50, "vip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription quota insufficient")
	assert.EqualValues(t, 90, getGroupBillingSub(t, 8207).AmountUsed)
}

// F: HasActive / AllowWalletOverflow ignore non-matching strict subs.
func TestGroupAwareActiveAndOverflowHelpers(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8106, Title: "StrictVIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 100, CreatedAt: now, UpdatedAt: now,
	})
	// strict vip sub only
	seedActiveSub(t, 8208, 206, 8106, "vip", 100, 0, false)

	hasDefault, err := HasActiveUserSubscription(206, "default")
	require.NoError(t, err)
	assert.False(t, hasDefault, "vip-only sub must not count for default group")

	hasVip, err := HasActiveUserSubscription(206, "vip")
	require.NoError(t, err)
	assert.True(t, hasVip)

	// unmatched group: no matching sub → overflow allowed (wallet path free)
	allowDefault, err := UserActiveSubscriptionsAllowWalletOverflow(206, "default")
	require.NoError(t, err)
	assert.True(t, allowDefault, "unmatched strict sub must not block wallet for other groups")

	// matching strict → no overflow
	allowVip, err := UserActiveSubscriptionsAllowWalletOverflow(206, "vip")
	require.NoError(t, err)
	assert.False(t, allowVip)

	// empty UpgradeGroup sub is universal
	seedActiveSub(t, 8209, 207, 8106, "", 100, 0, false)
	hasAny, err := HasActiveUserSubscription(207, "default")
	require.NoError(t, err)
	assert.True(t, hasAny)
	allowAny, err := UserActiveSubscriptionsAllowWalletOverflow(207, "pro")
	require.NoError(t, err)
	assert.False(t, allowAny)
}

// G: same request_id retries keep original pre-consume record / subscription.
func TestPreConsumeIdempotentSameRequestId(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8107, Title: "VIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8210, 208, 8107, "vip", 1000, 0, true)
	seedActiveSub(t, 8211, 208, 8107, "default", 1000, 0, true)

	res1, err := PreConsumeUserSubscription("req-g", 208, "gpt-4", 0, 200, "vip")
	require.NoError(t, err)
	assert.Equal(t, 8210, res1.UserSubscriptionId)

	// Retry with a different usingGroup must still return the original record, not rebind.
	res2, err := PreConsumeUserSubscription("req-g", 208, "gpt-4", 0, 200, "default")
	require.NoError(t, err)
	assert.Equal(t, res1.UserSubscriptionId, res2.UserSubscriptionId)
	assert.EqualValues(t, res1.PreConsumed, res2.PreConsumed)
	assert.EqualValues(t, 200, getGroupBillingSub(t, 8210).AmountUsed)
	assert.EqualValues(t, 0, getGroupBillingSub(t, 8211).AmountUsed)

	var records []SubscriptionPreConsumeRecord
	require.NoError(t, DB.Where("request_id = ?", "req-g").Find(&records).Error)
	assert.Len(t, records, 1)
	assert.Equal(t, 8210, records[0].UserSubscriptionId)
}

// H: "auto" is not a concrete billing group — restricted subs do not match;
// after resolution to a real group, pre-consume uses that same group.
func TestPreConsumeAutoIsNotConcreteBillingGroup(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8108, Title: "VIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8212, 209, 8108, "vip", 1000, 0, true)

	// unresolved auto must not debit vip sub
	_, err := PreConsumeUserSubscription("req-h-auto", 209, "gpt-4", 0, 50, "auto")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active subscription")
	assert.EqualValues(t, 0, getGroupBillingSub(t, 8212).AmountUsed)

	// resolved first-dispatch group (as HandleGroupRatio would set from auto_group)
	// must match the same concrete group used for the first channel selection
	res, err := PreConsumeUserSubscription("req-h-vip", 209, "gpt-4", 0, 50, "vip")
	require.NoError(t, err)
	assert.Equal(t, 8212, res.UserSubscriptionId)
	assert.EqualValues(t, 50, getGroupBillingSub(t, 8212).AmountUsed)

	// Cross-group retry with same request_id must not rebind to another sub
	// (pre-consume is once per request; idempotent path preserves funding).
	res2, err := PreConsumeUserSubscription("req-h-vip", 209, "gpt-4", 0, 50, "default")
	require.NoError(t, err)
	assert.Equal(t, 8212, res2.UserSubscriptionId)
	assert.EqualValues(t, 50, getGroupBillingSub(t, 8212).AmountUsed)
}

// I: pre-consume failure leaves subscription AmountUsed unchanged (no partial debit).
func TestPreConsumeFailureNoPartialDebit(t *testing.T) {
	truncateTables(t)
	now := GetDBTimestamp()
	seedGroupBillingPlan(t, &SubscriptionPlan{
		Id: 8109, Title: "VIP", PriceAmount: 10, DurationUnit: SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 100, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSub(t, 8213, 210, 8109, "vip", 100, 80, true)

	_, err := PreConsumeUserSubscription("req-i", 210, "gpt-4", 0, 50, "vip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscription quota insufficient")
	assert.EqualValues(t, 80, getGroupBillingSub(t, 8213).AmountUsed)

	var n int64
	require.NoError(t, DB.Model(&SubscriptionPreConsumeRecord{}).Where("request_id = ?", "req-i").Count(&n).Error)
	assert.Zero(t, n)

	// invalid amount
	_, err = PreConsumeUserSubscription("req-i2", 210, "gpt-4", 0, 0, "vip")
	require.Error(t, err)
	assert.EqualValues(t, 80, getGroupBillingSub(t, 8213).AmountUsed)
}

func TestSubscriptionAppliesToGroup(t *testing.T) {
	assert.True(t, subscriptionAppliesToGroup("", "default"))
	assert.True(t, subscriptionAppliesToGroup("", "auto"))
	assert.True(t, subscriptionAppliesToGroup("vip", "vip"))
	assert.False(t, subscriptionAppliesToGroup("vip", "default"))
	assert.False(t, subscriptionAppliesToGroup("vip", "auto"))
	assert.False(t, subscriptionAppliesToGroup("vip", ""))
	assert.True(t, subscriptionAppliesToGroup("  vip  ", "vip"))
}
