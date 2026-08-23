package service

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBillingSessionRefundRetryDoesNotRepeatCompletedStages(t *testing.T) {
	truncate(t)
	const userID, tokenID, refundedQuota = 707, 707, 100
	seedUser(t, userID, 1000)
	seedToken(t, tokenID, userID, "billing-refund-retry", common.MaxQuota-50)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("used_quota", refundedQuota).Error)

	info := &relaycommon.RelayInfo{UserId: userID, TokenId: tokenID, TokenKey: "billing-refund-retry"}
	session := &BillingSession{
		relayInfo:     info,
		funding:       &WalletFunding{userId: userID, consumed: refundedQuota},
		tokenConsumed: refundedQuota,
	}
	ctx := billingTestContext(t, 1000)

	err := session.Refund(ctx)
	require.ErrorIs(t, err, model.ErrTokenQuotaLimitExceeded)
	assert.Equal(t, 1000, getUserQuota(t, userID))
	assert.False(t, session.refunded)

	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("remain_quota", 1000).Error)
	require.NoError(t, session.Refund(ctx))
	assert.Equal(t, 1100, getUserQuota(t, userID))
	assert.Equal(t, 1100, getTokenRemainQuota(t, tokenID))
	assert.True(t, session.refunded)
}

func TestBillingSessionSubscriptionRefundRollsBackWithTokenFailure(t *testing.T) {
	truncate(t)
	const userID, tokenID, subscriptionID = 708, 708, 708
	const baseQuota, extraQuota, tokenQuota = 100, 50, 150
	const subscriptionUsed int64 = 500
	seedUser(t, userID, 0)
	seedToken(t, tokenID, userID, "billing-subscription-refund", common.MaxQuota-100)
	seedSubscription(t, subscriptionID, userID, 1_000, subscriptionUsed)
	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("used_quota", tokenQuota).Error)
	record := model.SubscriptionPreConsumeRecord{
		RequestId:          "billing-subscription-refund",
		UserId:             userID,
		UserSubscriptionId: subscriptionID,
		PreConsumed:        baseQuota,
		Status:             "consumed",
	}
	require.NoError(t, model.DB.Create(&record).Error)

	info := &relaycommon.RelayInfo{UserId: userID, TokenId: tokenID, TokenKey: "billing-subscription-refund"}
	session := &BillingSession{
		relayInfo: info,
		funding: &SubscriptionFunding{
			requestId:      record.RequestId,
			subscriptionId: subscriptionID,
			preConsumed:    baseQuota,
		},
		tokenConsumed: tokenQuota,
		extraReserved: extraQuota,
	}
	ctx := billingTestContext(t, 0)

	require.ErrorIs(t, session.Refund(ctx), model.ErrTokenQuotaLimitExceeded)
	assert.Equal(t, subscriptionUsed, getSubscriptionUsed(t, subscriptionID))
	require.NoError(t, model.DB.First(&record, record.Id).Error)
	assert.Equal(t, "consumed", record.Status)

	require.NoError(t, model.DB.Model(&model.Token{}).Where("id = ?", tokenID).Update("remain_quota", 1000).Error)
	require.NoError(t, session.Refund(ctx))
	assert.Equal(t, subscriptionUsed-baseQuota-extraQuota, getSubscriptionUsed(t, subscriptionID))
	assert.Equal(t, 1000+tokenQuota, getTokenRemainQuota(t, tokenID))
	require.NoError(t, model.DB.First(&record, record.Id).Error)
	assert.Equal(t, "refunded", record.Status)
}

func TestBillingSessionReserveAdditionalSerializesConcurrentIncrements(t *testing.T) {
	truncate(t)
	const userID = 706
	seedUser(t, userID, 3_000)
	relayInfo := &relaycommon.RelayInfo{UserId: userID, IsPlayground: true}
	session := &BillingSession{
		relayInfo: relayInfo,
		funding:   &WalletFunding{userId: userID},
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- session.ReserveAdditional(1_000)
		}()
	}
	wg.Wait()
	for range 2 {
		require.NoError(t, <-errs)
	}

	assert.Equal(t, 2_000, session.GetPreConsumedQuota())
	quota, err := model.GetUserQuotaFromDB(userID)
	require.NoError(t, err)
	assert.Equal(t, 1_000, quota)
}

func TestWalletBillingReservesQuotaAboveTrustThreshold(t *testing.T) {
	truncate(t)

	const userID = 704
	initialQuota := common.GetTrustQuota() + 10_000
	const reservedQuota = 4_000
	seedUser(t, userID, initialQuota)

	ctx := billingTestContext(t, initialQuota)
	info := &relaycommon.RelayInfo{
		UserId:         userID,
		IsPlayground:   true,
		TokenUnlimited: true,
		UsingGroup:     "default",
		UserQuota:      initialQuota,
		UserSetting: dto.UserSetting{
			BillingPreference: "wallet_only",
		},
	}

	apiErr := PreConsumeBilling(ctx, reservedQuota, info)
	require.Nil(t, apiErr)
	require.NotNil(t, info.Billing)
	assert.Equal(t, reservedQuota, info.FinalPreConsumedQuota)

	quota, err := model.GetUserQuotaFromDB(userID)
	require.NoError(t, err)
	assert.Equal(t, initialQuota-reservedQuota, quota)
}

func TestRealtimeReservationsCannotOverdrawTokenQuota(t *testing.T) {
	truncate(t)

	const (
		userID   = 705
		tokenID  = 705
		tokenKey = "realtime-race-token"
	)
	seedUser(t, userID, 100)
	seedToken(t, tokenID, userID, tokenKey, 10)

	oldRatios := ratio_setting.ModelRatio2JSONString()
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"realtime-race-test":10}`))
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(oldRatios))
	})

	newRelayInfo := func() *relaycommon.RelayInfo {
		info := &relaycommon.RelayInfo{
			UserId:          userID,
			TokenId:         tokenID,
			TokenKey:        tokenKey,
			UsingGroup:      "default",
			UserGroup:       "default",
			OriginModelName: "realtime-race-test",
		}
		session := &BillingSession{
			relayInfo: info,
			funding:   &WalletFunding{userId: userID},
		}
		info.Billing = session
		return info
	}

	usage := &dto.RealtimeUsage{}
	usage.InputTokenDetails.TextTokens = 1
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		info := newRelayInfo()
		ctx := billingTestContext(t, 10)
		go func() {
			<-start
			results <- PreWssConsumeQuota(ctx, info, usage)
		}()
	}
	close(start)

	succeeded := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			succeeded++
		}
	}
	assert.Equal(t, 1, succeeded)

	var token model.Token
	require.NoError(t, model.DB.First(&token, tokenID).Error)
	assert.Equal(t, 0, token.RemainQuota)
	assert.Equal(t, 10, token.UsedQuota)
	quota, err := model.GetUserQuotaFromDB(userID)
	require.NoError(t, err)
	assert.Equal(t, 90, quota)
}
