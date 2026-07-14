package model

import (
	"errors"
	"math"
	"path/filepath"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAffiliateTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
	oldRate := common.AffiliateCommissionRate
	oldQuotaPerUnit := common.QuotaPerUnit
	paymentSetting := operation_setting.GetPaymentSetting()
	oldConfirmed := paymentSetting.ComplianceConfirmed
	oldVersion := paymentSetting.ComplianceTermsVersion
	common.AffiliateCommissionRate = 10
	common.QuotaPerUnit = 1000
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion
	t.Cleanup(func() {
		common.AffiliateCommissionRate = oldRate
		common.QuotaPerUnit = oldQuotaPerUnit
		paymentSetting.ComplianceConfirmed = oldConfirmed
		paymentSetting.ComplianceTermsVersion = oldVersion
	})
}

func createAffiliateTestUser(t *testing.T, id int, username string, code string, inviterId int, rate *float64) {
	t.Helper()
	require.NoError(t, DB.Create(&User{
		Id:                id,
		Username:          username,
		AffCode:           code,
		InviterId:         inviterId,
		AffCommissionRate: rate,
		Status:            common.UserStatusEnabled,
	}).Error)
}

func TestValidateAffiliateCommissionRate(t *testing.T) {
	for _, rate := range []float64{0, 0.01, 12.34, 100} {
		assert.NoError(t, ValidateAffiliateCommissionRate(rate))
	}
	for _, rate := range []float64{-0.01, 12.345, 100.01} {
		assert.Error(t, ValidateAffiliateCommissionRate(rate))
	}
}

func TestAffiliateSchemaCanMigrateSQLiteTwice(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "affiliate.db")
	db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{})
	require.NoError(t, err)

	for range 2 {
		require.NoError(t, db.AutoMigrate(&User{}, &AffiliateCommission{}))
	}
}

func TestAffiliateCodeNormalizationAndCaseInsensitiveUniqueness(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "legacy", "Legacy_Code", 0, nil)
	createAffiliateTestUser(t, 2, "target", "target", 0, nil)

	code, err := NormalizeAffiliateCode("  New_Code-1 ")
	require.NoError(t, err)
	assert.Equal(t, "new_code-1", code)
	require.ErrorIs(t, SetAffiliateUserConfig(2, "legacy_code", nil, 0), ErrAffiliateCodeUnavailable)
	assert.Error(t, SetAffiliateUserConfig(2, "bad code", nil, 0))
}

func TestSetAffiliateUserConfigPreservesLegacyCode(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "legacy", "ABC", 0, nil)
	rate := 12.5

	require.NoError(t, SetAffiliateUserConfig(1, "ABC", &rate, 0))
	var user User
	require.NoError(t, DB.First(&user, 1).Error)
	assert.Equal(t, "ABC", user.AffCode)
	require.NotNil(t, user.AffCommissionRate)
	assert.Equal(t, rate, *user.AffCommissionRate)
}

func TestGetAffiliateUserDetailGeneratesMissingCode(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "missing-code", "", 0, nil)

	detail, err := GetAffiliateUserDetail(1)
	require.NoError(t, err)
	assert.Regexp(t, affiliateCodePattern, detail.AffCode)
	assert.Len(t, detail.AffCode, 8)

	var user User
	require.NoError(t, DB.First(&user, 1).Error)
	assert.Equal(t, detail.AffCode, user.AffCode)
}

func TestEnsureAffiliateCodeConcurrentCallsReturnSameCode(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "missing-code", "", 0, nil)

	codes := make(chan string, 2)
	errCh := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			code, err := EnsureAffiliateCode(1)
			codes <- code
			errCh <- err
		}()
	}
	waitGroup.Wait()
	close(codes)
	close(errCh)

	for err := range errCh {
		require.NoError(t, err)
	}
	var generated []string
	for code := range codes {
		generated = append(generated, code)
	}
	require.Len(t, generated, 2)
	assert.Equal(t, generated[0], generated[1])
	assert.Regexp(t, affiliateCodePattern, generated[0])
}

func TestConcurrentAdminCodeUpdateAllowsOneOwner(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "one", "one1", 0, nil)
	createAffiliateTestUser(t, 2, "two", "two2", 0, nil)

	errCh := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, userId := range []int{1, 2} {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errCh <- SetAffiliateUserConfig(userId, "shared_code", nil, 0)
		}()
	}
	waitGroup.Wait()
	close(errCh)

	successCount := 0
	conflictCount := 0
	for err := range errCh {
		if err == nil {
			successCount++
			continue
		}
		if errors.Is(err, ErrAffiliateCodeUnavailable) {
			conflictCount++
			continue
		}
		require.NoError(t, err)
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, conflictCount)
}

func TestBindAffiliateInviterIsOneTimeAndDoesNotGrantRegistrationReward(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "InviteMe", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 0, nil)

	require.NoError(t, BindAffiliateInviterByCode(2, "inviteme"))
	require.ErrorIs(t, BindAffiliateInviterByCode(2, "InviteMe"), ErrAffiliateAlreadyBound)

	var inviter User
	var invitee User
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	assert.Equal(t, 1, inviter.AffCount)
	assert.Zero(t, inviter.AffQuota)
	assert.Zero(t, inviter.AffHistoryQuota)
	assert.Equal(t, 1, invitee.InviterId)
	assert.Zero(t, invitee.Quota)
}

func TestAffiliateRelationRejectsSelfAndCyclesAndMaintainsCounts(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "one", "one1", 0, nil)
	createAffiliateTestUser(t, 2, "two", "two2", 0, nil)
	createAffiliateTestUser(t, 3, "three", "three3", 0, nil)

	require.NoError(t, SetAffiliateUserConfig(2, "two2", nil, 1))
	require.NoError(t, SetAffiliateUserConfig(3, "three3", nil, 2))
	require.ErrorIs(t, SetAffiliateUserConfig(1, "one1", nil, 1), ErrAffiliateSelfInvite)
	require.ErrorIs(t, SetAffiliateUserConfig(1, "one1", nil, 3), ErrAffiliateRelationCycle)

	require.NoError(t, SetAffiliateUserConfig(3, "three3", nil, 1))
	var one User
	var two User
	require.NoError(t, DB.First(&one, 1).Error)
	require.NoError(t, DB.First(&two, 2).Error)
	assert.Equal(t, 2, one.AffCount)
	assert.Zero(t, two.AffCount)
}

func TestAffiliateRelationRejectsCycleThroughDeletedUser(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "one", "one1", 0, nil)
	createAffiliateTestUser(t, 2, "two", "two2", 3, nil)
	createAffiliateTestUser(t, 3, "three", "three3", 1, nil)

	var three User
	require.NoError(t, DB.First(&three, 3).Error)
	require.NoError(t, three.Delete())
	require.ErrorIs(t, SetAffiliateUserConfig(1, "one1", nil, 2), ErrAffiliateRelationCycle)
}

func TestHardDeleteAfterSoftDeleteDoesNotDecrementInviterTwice(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "inviter", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Update("aff_count", 1).Error)

	var invitee User
	require.NoError(t, DB.First(&invitee, 2).Error)
	require.NoError(t, invitee.Delete())
	require.NoError(t, invitee.HardDelete())

	var inviter User
	require.NoError(t, DB.First(&inviter, 1).Error)
	assert.Zero(t, inviter.AffCount)
}

func TestRechargeCreatesOneAffiliateCommission(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "rate10", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          10,
		Money:           10,
		TradeNo:         "affiliate-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	require.NoError(t, RechargeEpay("affiliate-topup", "wechat", "127.0.0.1"))
	require.NoError(t, RechargeEpay("affiliate-topup", "wechat", "127.0.0.1"))

	var inviter User
	var invitee User
	var commissions []AffiliateCommission
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	require.NoError(t, DB.Find(&commissions).Error)
	assert.Equal(t, 10_000, invitee.Quota)
	assert.Equal(t, 1000, inviter.AffQuota)
	assert.Equal(t, 1000, inviter.AffHistoryQuota)
	require.Len(t, commissions, 1)
	assert.Equal(t, AffiliateCommissionSourceTopUp, commissions[0].SourceType)
	assert.Equal(t, 10_000, commissions[0].BaseQuota)
	assert.Equal(t, 10.0, commissions[0].CommissionRate)
	assert.Equal(t, 1000, commissions[0].CommissionQuota)
}

func TestSettleAffiliateCommissionReportsSkippedAndSettled(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "settle", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)

	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		commission, settled, err := SettleAffiliateCommissionTx(
			tx, 2, AffiliateCommissionSourceTopUp, 1, "skip-zero", PaymentProviderEpay, 0, 1,
		)
		require.NoError(t, err)
		assert.False(t, settled)
		assert.Nil(t, commission)

		commission, settled, err = SettleAffiliateCommissionTx(
			tx, 2, AffiliateCommissionSourceTopUp, 2, "settled", PaymentProviderEpay, 1000, 1,
		)
		require.NoError(t, err)
		assert.True(t, settled)
		require.NotNil(t, commission)
		assert.Equal(t, 100, commission.CommissionQuota)
		return nil
	}))
}

func TestConcurrentTopUpCompletionCreatesOneAffiliateCommission(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "concurrent", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          10,
		Money:           10,
		TradeNo:         "concurrent-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	errors := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			errors <- RechargeEpay("concurrent-topup", "alipay", "127.0.0.1")
		}()
	}
	waitGroup.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	var inviter User
	var invitee User
	var count int64
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
	assert.Equal(t, 10_000, invitee.Quota)
	assert.Equal(t, 1000, inviter.AffQuota)
	assert.Equal(t, int64(1), count)
}

func TestAllTopUpCompletionPathsCreateCommission(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		amount   int64
		money    float64
		complete func(tradeNo string) error
	}{
		{
			name:     "epay",
			provider: PaymentProviderEpay,
			amount:   10,
			money:    10,
			complete: func(tradeNo string) error { return RechargeEpay(tradeNo, "alipay", "127.0.0.1") },
		},
		{
			name:     "stripe",
			provider: PaymentProviderStripe,
			amount:   10,
			money:    10,
			complete: func(tradeNo string) error { return Recharge(tradeNo, "cus_test", "127.0.0.1") },
		},
		{
			name:     "creem",
			provider: PaymentProviderCreem,
			amount:   10_000,
			money:    10,
			complete: func(tradeNo string) error { return RechargeCreem(tradeNo, "", "", "127.0.0.1") },
		},
		{
			name:     "waffo",
			provider: PaymentProviderWaffo,
			amount:   10,
			money:    10,
			complete: func(tradeNo string) error { return RechargeWaffo(tradeNo, "127.0.0.1") },
		},
		{
			name:     "waffo pancake",
			provider: PaymentProviderWaffoPancake,
			amount:   10,
			money:    10,
			complete: RechargeWaffoPancake,
		},
		{
			name:     "admin completion",
			provider: PaymentProviderEpay,
			amount:   10,
			money:    10,
			complete: func(tradeNo string) error { return ManualCompleteTopUp(tradeNo, "127.0.0.1") },
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setupAffiliateTest(t)
			createAffiliateTestUser(t, 1, "inviter", "channelrate", 0, nil)
			createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
			tradeNo := "commission-" + testCase.provider + "-" + testCase.name
			require.NoError(t, DB.Create(&TopUp{
				UserId:          2,
				AffInviterId:    1,
				Amount:          testCase.amount,
				Money:           testCase.money,
				TradeNo:         tradeNo,
				PaymentMethod:   testCase.provider,
				PaymentProvider: testCase.provider,
				Status:          common.TopUpStatusPending,
			}).Error)

			require.NoError(t, testCase.complete(tradeNo))
			var commission AffiliateCommission
			require.NoError(t, DB.Where("source_type = ?", AffiliateCommissionSourceTopUp).First(&commission).Error)
			assert.Equal(t, 10_000, commission.BaseQuota)
			assert.Equal(t, 1000, commission.CommissionQuota)
		})
	}
}

func TestCustomZeroCommissionRateDisablesCommission(t *testing.T) {
	setupAffiliateTest(t)
	zero := 0.0
	createAffiliateTestUser(t, 1, "inviter", "ratezero", 0, &zero)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          5,
		Money:           5,
		TradeNo:         "zero-rate-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	require.NoError(t, RechargeEpay("zero-rate-topup", "alipay", "127.0.0.1"))
	var count int64
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCustomCommissionRateOverridesAndCanInheritDefault(t *testing.T) {
	setupAffiliateTest(t)
	customRate := 12.5
	createAffiliateTestUser(t, 1, "inviter", "rate125", 0, &customRate)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)

	for _, tradeNo := range []string{"custom-rate-topup", "default-rate-topup"} {
		require.NoError(t, DB.Create(&TopUp{
			UserId:          2,
			AffInviterId:    1,
			Amount:          10,
			Money:           10,
			TradeNo:         tradeNo,
			PaymentMethod:   "alipay",
			PaymentProvider: PaymentProviderEpay,
			Status:          common.TopUpStatusPending,
		}).Error)
		if tradeNo == "default-rate-topup" {
			require.NoError(t, SetAffiliateUserConfig(1, "rate125", nil, 0))
		}
		require.NoError(t, RechargeEpay(tradeNo, "alipay", "127.0.0.1"))
	}

	var commissions []AffiliateCommission
	require.NoError(t, DB.Order("id asc").Find(&commissions).Error)
	require.Len(t, commissions, 2)
	assert.Equal(t, 12.5, commissions[0].CommissionRate)
	assert.Equal(t, 1250, commissions[0].CommissionQuota)
	assert.Equal(t, 10.0, commissions[1].CommissionRate)
	assert.Equal(t, 1000, commissions[1].CommissionQuota)
}

func TestCommissionSkipsUnavailableInviterAndUnconfirmedCompliance(t *testing.T) {
	testCases := []struct {
		name  string
		setup func()
	}{
		{
			name: "disabled inviter",
			setup: func() {
				require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Update("status", common.UserStatusDisabled).Error)
			},
		},
		{
			name: "unconfirmed compliance",
			setup: func() {
				paymentSetting := operation_setting.GetPaymentSetting()
				paymentSetting.ComplianceConfirmed = false
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setupAffiliateTest(t)
			createAffiliateTestUser(t, 1, "inviter", "unavailable", 0, nil)
			createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
			testCase.setup()
			require.NoError(t, DB.Create(&TopUp{
				UserId:          2,
				AffInviterId:    1,
				Amount:          10,
				Money:           10,
				TradeNo:         "skip-" + testCase.name,
				PaymentMethod:   "alipay",
				PaymentProvider: PaymentProviderEpay,
				Status:          common.TopUpStatusPending,
			}).Error)

			require.NoError(t, RechargeEpay("skip-"+testCase.name, "alipay", "127.0.0.1"))
			var count int64
			require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
			assert.Zero(t, count)
		})
	}
}

func TestExternalSubscriptionCreatesCommissionButBalancePurchaseDoesNot(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "subrate", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	plan := &SubscriptionPlan{
		Id:            10,
		Title:         "Affiliate Plan",
		PriceAmount:   20,
		Currency:      "USD",
		DurationUnit:  SubscriptionDurationMonth,
		DurationValue: 1,
		Enabled:       true,
		TotalAmount:   1000,
	}
	require.NoError(t, DB.Create(plan).Error)
	order := &SubscriptionOrder{
		UserId:          2,
		PlanId:          plan.Id,
		TradeNo:         "affiliate-subscription",
		PaymentMethod:   PaymentMethodStripe,
		PaymentProvider: PaymentProviderStripe,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, order.Insert())
	require.NoError(t, CompleteSubscriptionOrder(order.TradeNo, "", PaymentProviderStripe, PaymentMethodStripe))

	var commission AffiliateCommission
	require.NoError(t, DB.Where("source_type = ?", AffiliateCommissionSourceSubscription).First(&commission).Error)
	assert.Equal(t, 20_000, commission.BaseQuota)
	assert.Equal(t, 2000, commission.CommissionQuota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", 2).Update("quota", 100_000).Error)
	require.NoError(t, PurchaseSubscriptionWithBalance(2, plan.Id))
	var count int64
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestCommissionFailureRollsBackTopUpAndUserQuota(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "rollback", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	topUp := &TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          5,
		Money:           5,
		TradeNo:         "rollback-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}
	require.NoError(t, DB.Create(topUp).Error)
	require.NoError(t, DB.Create(&AffiliateCommission{
		InviterId:       1,
		InviteeId:       2,
		SourceType:      AffiliateCommissionSourceTopUp,
		SourceId:        topUp.Id,
		SourceOrderNo:   topUp.TradeNo,
		BaseQuota:       5000,
		CommissionRate:  10,
		CommissionQuota: 500,
	}).Error)

	require.Error(t, RechargeEpay(topUp.TradeNo, "alipay", "127.0.0.1"))
	var gotTopUp TopUp
	var invitee User
	require.NoError(t, DB.First(&gotTopUp, topUp.Id).Error)
	require.NoError(t, DB.First(&invitee, 2).Error)
	assert.Equal(t, common.TopUpStatusPending, gotTopUp.Status)
	assert.Zero(t, invitee.Quota)
}

func TestCommissionUsesOrderTimeInviterSnapshot(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter-a", "snap-a", 0, nil)
	createAffiliateTestUser(t, 3, "inviter-b", "snap-b", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          10,
		Money:           10,
		TradeNo:         "snapshot-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	// Rebind after order creation must not reroute this pending order.
	require.NoError(t, SetAffiliateUserConfig(2, "invitee", nil, 3))
	require.NoError(t, RechargeEpay("snapshot-topup", "alipay", "127.0.0.1"))

	var commissions []AffiliateCommission
	require.NoError(t, DB.Find(&commissions).Error)
	require.Len(t, commissions, 1)
	assert.Equal(t, 1, commissions[0].InviterId)

	var inviterA, inviterB User
	require.NoError(t, DB.First(&inviterA, 1).Error)
	require.NoError(t, DB.First(&inviterB, 3).Error)
	assert.Equal(t, 1000, inviterA.AffQuota)
	assert.Zero(t, inviterB.AffQuota)
}

func TestCommissionOverflowDoesNotBlockTopUp(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "inviter", "overflow", 0, nil)
	createAffiliateTestUser(t, 2, "invitee", "invitee", 1, nil)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Updates(map[string]any{
		"aff_quota":   math.MaxInt32 - 10,
		"aff_history": math.MaxInt32 - 10,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId:          2,
		AffInviterId:    1,
		Amount:          10,
		Money:           10,
		TradeNo:         "overflow-topup",
		PaymentMethod:   "alipay",
		PaymentProvider: PaymentProviderEpay,
		Status:          common.TopUpStatusPending,
	}).Error)

	require.NoError(t, RechargeEpay("overflow-topup", "alipay", "127.0.0.1"))
	var invitee, inviter User
	var count int64
	require.NoError(t, DB.First(&invitee, 2).Error)
	require.NoError(t, DB.First(&inviter, 1).Error)
	require.NoError(t, DB.Model(&AffiliateCommission{}).Count(&count).Error)
	assert.Equal(t, 10_000, invitee.Quota)
	assert.Equal(t, math.MaxInt32-10, inviter.AffQuota)
	assert.Zero(t, count)
}

func TestGetUserIdByAffCodeRejectsAmbiguousLegacyCodes(t *testing.T) {
	setupAffiliateTest(t)
	createAffiliateTestUser(t, 1, "user-a", "AbCd", 0, nil)
	createAffiliateTestUser(t, 2, "user-b", "aBcD", 0, nil)

	id, err := GetUserIdByAffCode("AbCd")
	require.NoError(t, err)
	assert.Equal(t, 1, id)

	_, err = GetUserIdByAffCode("abcd")
	require.ErrorIs(t, err, ErrAffiliateCodeAmbiguous)
}
