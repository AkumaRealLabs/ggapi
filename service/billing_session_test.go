package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
