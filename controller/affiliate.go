package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type bindAffiliateInviterRequest struct {
	AffCode string `json:"aff_code"`
}

type updateAffiliateUserRequest struct {
	AffCode              string   `json:"aff_code"`
	CustomCommissionRate *float64 `json:"custom_commission_rate"`
	InviterId            int      `json:"inviter_id"`
}

type selfAffiliateCommission struct {
	Id              int     `json:"id"`
	SourceType      string  `json:"source_type"`
	SourceOrderNo   string  `json:"source_order_no"`
	PaymentProvider string  `json:"payment_provider"`
	BaseQuota       int     `json:"base_quota"`
	CommissionRate  float64 `json:"commission_rate"`
	CommissionQuota int     `json:"commission_quota"`
	SettledAt       int64   `json:"settled_at"`
}

func GetSelfAffiliate(c *gin.Context) {
	detail, err := model.GetAffiliateUserDetail(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func BindSelfAffiliate(c *gin.Context) {
	var request bindAffiliateInviterRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if err := model.BindAffiliateInviterByCode(c.GetInt("id"), request.AffCode); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetSelfAffiliateCommissions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListAffiliateCommissions(pageInfo, model.AffiliateCommissionFilters{
		InviterId:  c.GetInt("id"),
		OrderNo:    c.Query("order_no"),
		SourceType: c.Query("source_type"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	responses := make([]selfAffiliateCommission, 0, len(items))
	for _, item := range items {
		// Omit source_order_no: some providers embed the invitee user ID in the
		// trade number (e.g. USR{id}NO..., WAFFO-{id}-...), which would leak
		// invitee identity to the inviter.
		responses = append(responses, selfAffiliateCommission{
			Id:              item.Id,
			SourceType:      item.SourceType,
			PaymentProvider: item.PaymentProvider,
			BaseQuota:       item.BaseQuota,
			CommissionRate:  item.CommissionRate,
			CommissionQuota: item.CommissionQuota,
			SettledAt:       item.SettledAt,
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(responses)
	common.ApiSuccess(c, pageInfo)
}

func GetAffiliateCommissions(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	inviterId, _ := strconv.Atoi(strings.TrimSpace(c.Query("inviter_id")))
	items, total, err := model.ListAffiliateCommissions(pageInfo, model.AffiliateCommissionFilters{
		InviterId:  inviterId,
		OrderNo:    c.Query("order_no"),
		SourceType: c.Query("source_type"),
		Inviter:    c.Query("inviter"),
		Invitee:    c.Query("invitee"),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func SearchAffiliateUsers(c *gin.Context) {
	users, err := model.SearchAffiliateUsers(c.Query("keyword"), 20)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, users)
}

func GetAffiliateUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的用户 ID"})
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canManageTargetRole(c.GetInt("role"), user.Role) {
		common.ApiErrorMsg(c, "无权查看同级或更高等级用户")
		return
	}
	detail, err := model.GetAffiliateUserDetail(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func UpdateAffiliateUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的用户 ID"})
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !canManageTargetRole(c.GetInt("role"), user.Role) {
		common.ApiErrorMsg(c, "无权管理同级或更高等级用户")
		return
	}
	var request updateAffiliateUserRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	request.AffCode = strings.TrimSpace(request.AffCode)
	if err := model.SetAffiliateUserConfig(userId, request.AffCode, request.CustomCommissionRate, request.InviterId); err != nil {
		common.ApiError(c, err)
		return
	}
	detail, err := model.GetAffiliateUserDetail(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}
