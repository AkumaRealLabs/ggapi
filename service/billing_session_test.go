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
