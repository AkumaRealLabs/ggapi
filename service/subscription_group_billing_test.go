package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedPlan(t *testing.T, plan *model.SubscriptionPlan) {
	t.Helper()
	require.NoError(t, model.DB.Create(plan).Error)
}

func seedSub(t *testing.T, sub *model.UserSubscription) {
	t.Helper()
	require.NoError(t, model.DB.Create(sub).Error)
}

func getSub(t *testing.T, id int) model.UserSubscription {
	t.Helper()
	var sub model.UserSubscription
	require.NoError(t, model.DB.Where("id = ?", id).First(&sub).Error)
	return sub
}

func getUserQuotaForGroupBilling(t *testing.T, id int) int {
	t.Helper()
	q, err := model.GetUserQuota(id, true)
	require.NoError(t, err)
	return q
}

func seedActiveSubForGroup(t *testing.T, id, userId, planId int, upgradeGroup string, total, used int64, allowOverflow bool) {
	t.Helper()
	now := time.Now().Unix()
	seedSub(t, &model.UserSubscription{
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

func billingTestContext(t *testing.T, tokenQuota int) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Set("token_quota", tokenQuota)
	return c
}

func makeGroupBillingRelayInfo(userId, tokenId int, tokenKey, requestId, usingGroup, pref string, userQuota int) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		UserId:   userId,
		TokenId:  tokenId,
		TokenKey: tokenKey,
		// Playground skips token-quota pre-consume (avoids commonKeyCol init in service TestMain).
		// ForcePreConsume still forces full funding pre-consume for wallet/subscription.
		IsPlayground:    true,
		TokenUnlimited:  true,
		RequestId:       requestId,
		UsingGroup:      usingGroup,
		OriginModelName: "gpt-4",
		UserQuota:       userQuota,
		ForcePreConsume: true,
		UserSetting: dto.UserSetting{
			BillingPreference: pref,
		},
	}
}

// A + F: vip-only strict sub + default group + subscription_first → wallet, vip AmountUsed unchanged.
func TestSubscriptionFirstMismatchedGroupUsesWallet(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 301, 5000)
	seedToken(t, 301, 301, "tk-301", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9101, Title: "VIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9301, 301, 9101, "vip", 1000, 0, false)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(301, 301, "tk-301", "svc-req-a", "default", "subscription_first", 5000)

	session, apiErr := NewBillingSession(c, info, 100)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceWallet, session.funding.Source())
	assert.EqualValues(t, 0, getSub(t, 9301).AmountUsed)
	assert.Equal(t, 4900, getUserQuotaForGroupBilling(t, 301))
}

// B: vip sub + vip group → subscription pre-consume.
func TestSubscriptionFirstMatchingGroupUsesSubscription(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 302, 5000)
	seedToken(t, 302, 302, "tk-302", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9102, Title: "VIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9302, 302, 9102, "vip", 1000, 0, true)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(302, 302, "tk-302", "svc-req-b", "vip", "subscription_first", 5000)

	session, apiErr := NewBillingSession(c, info, 120)
	require.Nil(t, apiErr)
	require.NotNil(t, session)
	assert.Equal(t, BillingSourceSubscription, session.funding.Source())
	assert.EqualValues(t, 120, getSub(t, 9302).AmountUsed)
	assert.Equal(t, 5000, getUserQuotaForGroupBilling(t, 302))
}

// C: empty UpgradeGroup works for any group via SubscriptionFunding.
func TestSubscriptionFundingEmptyUpgradeGroupAnyGroup(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 303, 100)
	seedToken(t, 303, 303, "tk-303", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9103, Title: "Legacy", PriceAmount: 5, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 500, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9303, 303, 9103, "", 500, 0, true)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(303, 303, "tk-303", "svc-req-c", "pro", "subscription_only", 100)

	session, apiErr := NewBillingSession(c, info, 40)
	require.Nil(t, apiErr)
	assert.Equal(t, BillingSourceSubscription, session.funding.Source())
	assert.EqualValues(t, 40, getSub(t, 9303).AmountUsed)
}

// E: matching strict sub insufficient → reject, no wallet.
func TestSubscriptionFirstMatchingStrictInsufficientRejects(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 305, 5000)
	seedToken(t, 305, 305, "tk-305", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9105, Title: "Strict", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 100, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9305, 305, 9105, "vip", 100, 90, false)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(305, 305, "tk-305", "svc-req-e", "vip", "subscription_first", 5000)

	session, apiErr := NewBillingSession(c, info, 50)
	require.NotNil(t, apiErr)
	assert.Nil(t, session)
	assert.Equal(t, types.ErrorCodeInsufficientUserQuota, apiErr.GetErrorCode())
	assert.EqualValues(t, 90, getSub(t, 9305).AmountUsed)
	assert.Equal(t, 5000, getUserQuotaForGroupBilling(t, 305))
}

// F (service): unmatched strict vip must not block wallet fallback for default.
func TestSubscriptionFirstUnmatchedStrictDoesNotBlockWallet(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 306, 3000)
	seedToken(t, 306, 306, "tk-306", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9106, Title: "StrictVIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 100, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9306, 306, 9106, "vip", 100, 0, false)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(306, 306, "tk-306", "svc-req-f", "default", "subscription_first", 3000)

	session, apiErr := NewBillingSession(c, info, 200)
	require.Nil(t, apiErr)
	assert.Equal(t, BillingSourceWallet, session.funding.Source())
	assert.EqualValues(t, 0, getSub(t, 9306).AmountUsed)
	assert.Equal(t, 2800, getUserQuotaForGroupBilling(t, 306))
}

// wallet_first: wallet empty → only matching-group subscription.
func TestWalletFirstFallsBackOnlyToMatchingSubscription(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 307, 0)
	seedToken(t, 307, 307, "tk-307", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9107, Title: "VIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9307, 307, 9107, "vip", 1000, 0, true)

	c := billingTestContext(t, 5000)

	// default group: vip sub must not fund
	infoDefault := makeGroupBillingRelayInfo(307, 307, "tk-307", "svc-req-wf-d", "default", "wallet_first", 0)
	session, apiErr := NewBillingSession(c, infoDefault, 50)
	require.NotNil(t, apiErr)
	assert.Nil(t, session)
	assert.EqualValues(t, 0, getSub(t, 9307).AmountUsed)

	// vip group: wallet empty, matching sub funds
	infoVip := makeGroupBillingRelayInfo(307, 307, "tk-307", "svc-req-wf-v", "vip", "wallet_first", 0)
	session, apiErr = NewBillingSession(c, infoVip, 50)
	require.Nil(t, apiErr)
	assert.Equal(t, BillingSourceSubscription, session.funding.Source())
	assert.EqualValues(t, 50, getSub(t, 9307).AmountUsed)
}

// H: pre-consume group equals first-dispatch resolved group; cross-group retry same request_id keeps binding.
func TestBillingSessionAutoResolvedGroupConsistentWithPreConsume(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 308, 100)
	seedToken(t, 308, 308, "tk-308", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9108, Title: "VIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 1000, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9308, 308, 9108, "vip", 1000, 0, true)

	// Simulate HandleGroupRatio: token group was "auto", Distributor set auto_group=vip,
	// ModelPriceHelper wrote UsingGroup=vip before PreConsumeBilling.
	resolvedFirstDispatchGroup := "vip"
	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(308, 308, "tk-308", "svc-req-h", resolvedFirstDispatchGroup, "subscription_first", 100)

	session, apiErr := NewBillingSession(c, info, 80)
	require.Nil(t, apiErr)
	assert.Equal(t, BillingSourceSubscription, session.funding.Source())
	assert.Equal(t, 9308, info.SubscriptionId)
	assert.EqualValues(t, 80, getSub(t, 9308).AmountUsed)

	// Cross-group retry must not re-preconsume another subscription (idempotent by request_id).
	funding := &SubscriptionFunding{
		requestId:  "svc-req-h",
		userId:     308,
		modelName:  "gpt-4",
		usingGroup: "default", // would-be retry group
		amount:     80,
	}
	require.NoError(t, funding.PreConsume(80))
	assert.Equal(t, 9308, funding.subscriptionId)
	assert.EqualValues(t, 80, getSub(t, 9308).AmountUsed)

	// Unresolved "auto" string must not debit restricted subscription.
	fundingAuto := &SubscriptionFunding{
		requestId:  "svc-req-h-auto",
		userId:     308,
		modelName:  "gpt-4",
		usingGroup: "auto",
		amount:     10,
	}
	require.Error(t, fundingAuto.PreConsume(10))
	assert.EqualValues(t, 80, getSub(t, 9308).AmountUsed)
}

// I: failed pre-consume leaves user, token, subscription unchanged.
func TestBillingSessionFailedPreConsumeNoPartialDebit(t *testing.T) {
	truncate(t)
	now := time.Now().Unix()
	seedUser(t, 309, 5000)
	seedToken(t, 309, 309, "tk-309", 5000)
	seedPlan(t, &model.SubscriptionPlan{
		Id: 9109, Title: "VIP", PriceAmount: 10, DurationUnit: model.SubscriptionDurationMonth,
		DurationValue: 1, TotalAmount: 50, CreatedAt: now, UpdatedAt: now,
	})
	seedActiveSubForGroup(t, 9309, 309, 9109, "vip", 50, 40, false)

	c := billingTestContext(t, 5000)
	info := makeGroupBillingRelayInfo(309, 309, "tk-309", "svc-req-i", "vip", "subscription_only", 5000)

	session, apiErr := NewBillingSession(c, info, 30)
	require.NotNil(t, apiErr)
	assert.Nil(t, session)
	assert.EqualValues(t, 40, getSub(t, 9309).AmountUsed)
	assert.Equal(t, 5000, getUserQuotaForGroupBilling(t, 309))

	// Token pre-consume is skipped in playground fixtures; funding path is the
	// invariant under test (subscription AmountUsed + wallet quota unchanged).
	var n int64
	require.NoError(t, model.DB.Model(&model.SubscriptionPreConsumeRecord{}).Where("request_id = ?", "svc-req-i").Count(&n).Error)
	assert.Zero(t, n)
}
