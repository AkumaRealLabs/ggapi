package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/thanhpk/randstr"
)

type SubscriptionWaffoPancakePayRequest struct {
	PlanId int `json:"plan_id"`
}

func SubscriptionRequestWaffoPancakePay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req SubscriptionWaffoPancakePayRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.PlanId <= 0 {
		common.ApiErrorMsg(c, "Invalid parameters")
		return
	}

	plan, err := model.GetSubscriptionPlanById(req.PlanId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !plan.Enabled {
		common.ApiError(c, model.ErrSubscriptionPlanDisabled)
		return
	}
	if strings.TrimSpace(plan.WaffoPancakeProductId) == "" {
		common.ApiErrorMsg(c, "This plan is not configured for the selected payment provider.")
		return
	}
	// Plan targets its own Pancake product, so we only require credentials
	// here — not the gateway-level WaffoPancakeProductID.
	if strings.TrimSpace(setting.WaffoPancakeMerchantID) == "" ||
		strings.TrimSpace(setting.WaffoPancakePrivateKey) == "" {
		common.ApiErrorMsg(c, "The selected payment provider is not configured correctly.")
		return
	}

	userId := c.GetInt("id")
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user == nil {
		common.ApiErrorMsg(c, "User not found.")
		return
	}

	// WAFFO_PANCAKE_SUB- prefix (vs. wallet's WAFFO_PANCAKE-) drives webhook
	// dispatch in WaffoPancakeWebhook.
	tradeNo := fmt.Sprintf("WAFFO_PANCAKE_SUB-%d-%d-%s", userId, time.Now().UnixMilli(), randstr.String(6))

	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         tradeNo,
		PaymentMethod:   model.PaymentMethodWaffoPancake,
		PaymentProvider: model.PaymentProviderWaffoPancake,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	lockedPlan, err := model.InsertPendingSubscriptionOrder(order)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Waffo Pancake 订阅订单创建失败 user_id=%d plan_id=%d trade_no=%s error=%q", userId, plan.Id, tradeNo, err.Error()))
		common.ApiError(c, err)
		return
	}

	expiresInSeconds := 45 * 60
	session, err := service.CreateWaffoPancakeCheckoutSession(c.Request.Context(), &service.WaffoPancakeCreateSessionParams{
		ProductID:     lockedPlan.WaffoPancakeProductId,
		BuyerIdentity: service.WaffoPancakeBuyerIdentityFromUserID(user.Id),
		PriceSnapshot: &service.WaffoPancakePriceSnapshot{
			Amount:      decimal.NewFromFloat(lockedPlan.PriceAmount).StringFixed(2),
			TaxCategory: "saas",
		},
		BuyerEmail:              getWaffoPancakeBuyerEmail(user),
		ExpiresInSeconds:        &expiresInSeconds,
		OrderMerchantExternalID: tradeNo,
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Waffo Pancake 订阅结账会话创建失败 user_id=%d plan_id=%d trade_no=%s error=%q", userId, plan.Id, tradeNo, err.Error()))
		_ = model.ExpireSubscriptionOrder(tradeNo, model.PaymentProviderWaffoPancake)
		common.ApiErrorMsg(c, "Payment request failed")
		return
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf("Waffo Pancake 订阅订单创建成功 user_id=%d plan_id=%d trade_no=%s session_id=%s money=%.2f", userId, plan.Id, tradeNo, session.SessionID, lockedPlan.PriceAmount))

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url":     session.CheckoutURL,
			"session_id":       session.SessionID,
			"expires_at":       session.ExpiresAt,
			"order_id":         tradeNo,
			"token":            session.Token,
			"token_expires_at": session.TokenExpiresAt,
		},
	})
}
