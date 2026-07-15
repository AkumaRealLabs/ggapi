package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

type SubscriptionCreemPayRequest struct {
	PlanId int `json:"plan_id"`
}

func SubscriptionRequestCreemPay(c *gin.Context) {
	if !requirePaymentCompliance(c) {
		return
	}

	var req SubscriptionCreemPayRequest

	// Keep body for debugging consistency (like RequestCreemPay)
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Creem 订阅支付请求读取失败 error=%q", err.Error()))
		common.ApiErrorMsg(c, "Failed to read payment request.")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
	if plan.CreemProductId == "" {
		common.ApiErrorMsg(c, "This plan is not configured for the selected payment provider.")
		return
	}
	if setting.CreemWebhookSecret == "" && !setting.CreemTestMode {
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

	reference := "sub-creem-ref-" + randstr.String(6)
	referenceId := "sub_ref_" + common.Sha1([]byte(reference+time.Now().String()+user.Username))

	// create pending order first (locks plan; use returned snapshot for checkout)
	order := &model.SubscriptionOrder{
		UserId:          userId,
		PlanId:          plan.Id,
		Money:           plan.PriceAmount,
		TradeNo:         referenceId,
		PaymentMethod:   model.PaymentMethodCreem,
		PaymentProvider: model.PaymentProviderCreem,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}
	lockedPlan, err := model.InsertPendingSubscriptionOrder(order)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Reuse Creem checkout generator by building a lightweight product reference.
	currency := "USD"
	switch operation_setting.GetGeneralSetting().QuotaDisplayType {
	case operation_setting.QuotaDisplayTypeCNY:
		currency = "CNY"
	case operation_setting.QuotaDisplayTypeUSD:
		currency = "USD"
	default:
		currency = "USD"
	}
	product := &CreemProduct{
		ProductId: lockedPlan.CreemProductId,
		Name:      lockedPlan.Title,
		Price:     lockedPlan.PriceAmount,
		Currency:  currency,
		Quota:     0,
	}

	checkoutUrl, err := genCreemLink(c.Request.Context(), referenceId, product, user.Email, user.Username)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("Creem 订阅支付链接创建失败 trade_no=%s product_id=%s error=%q", referenceId, product.ProductId, err.Error()))
		_ = model.ExpireSubscriptionOrder(referenceId, model.PaymentProviderCreem)
		common.ApiErrorMsg(c, "Payment request failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data": gin.H{
			"checkout_url": checkoutUrl,
			"order_id":     referenceId,
		},
	})
}
