package model

import (
	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	retiredMembershipPlanTable         = "subscription_plans"
	retiredMembershipSubscriptionTable = "user_subscriptions"
	retiredMembershipColumn            = "membership_only"
	retiredMembershipScheduledStatus   = "scheduled"
)

// retireMembershipSubscriptions cleans up data left by the removed ggapi
// membership-only plans (GG-006). Those plans have a zero total amount, which
// upstream treats as an unlimited subscription, so they must not stay usable:
// membership plans are disabled and their pending orders expired, queued
// memberships are cancelled, and active memberships end now so the regular
// expiration task restores the user group. Every processed row also clears
// membership_only in the same transaction, so a restart never repeats the
// cleanup or overrides later admin changes. Fresh installs have no such column.
func retireMembershipSubscriptions() error {
	migrator := DB.Migrator()
	planHasColumn := migrator.HasColumn(retiredMembershipPlanTable, retiredMembershipColumn)
	subHasColumn := migrator.HasColumn(retiredMembershipSubscriptionTable, retiredMembershipColumn)
	if !planHasColumn && !subHasColumn {
		return nil
	}

	var retiredPlans, expiredOrders, cancelledSubs, endedSubs int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		now := GetDBTimestamp()
		if planHasColumn {
			var planIds []int
			if err := tx.Table(retiredMembershipPlanTable).
				Where(retiredMembershipColumn+" = ?", true).
				Pluck("id", &planIds).Error; err != nil {
				return err
			}
			if len(planIds) > 0 {
				res := tx.Model(&SubscriptionOrder{}).
					Where("plan_id IN ? AND status = ?", planIds, common.TopUpStatusPending).
					Update("status", common.TopUpStatusExpired)
				if res.Error != nil {
					return res.Error
				}
				expiredOrders = res.RowsAffected
				res = tx.Table(retiredMembershipPlanTable).
					Where("id IN ?", planIds).
					Updates(map[string]any{
						"enabled":               false,
						retiredMembershipColumn: false,
					})
				if res.Error != nil {
					return res.Error
				}
				retiredPlans = res.RowsAffected
			}
		}
		if subHasColumn {
			res := tx.Table(retiredMembershipSubscriptionTable).
				Where("status = ?", retiredMembershipScheduledStatus).
				Updates(map[string]any{
					"status":                "cancelled",
					retiredMembershipColumn: false,
					"updated_at":            now,
				})
			if res.Error != nil {
				return res.Error
			}
			cancelledSubs = res.RowsAffected
			res = tx.Table(retiredMembershipSubscriptionTable).
				Where(retiredMembershipColumn+" = ? AND status = ?", true, "active").
				Updates(map[string]any{
					"end_time":              now,
					retiredMembershipColumn: false,
					"updated_at":            now,
				})
			if res.Error != nil {
				return res.Error
			}
			endedSubs = res.RowsAffected
			if err := tx.Table(retiredMembershipSubscriptionTable).
				Where(retiredMembershipColumn+" = ?", true).
				Update(retiredMembershipColumn, false).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if retiredPlans+expiredOrders+cancelledSubs+endedSubs > 0 {
		common.SysLog(common.LogText("retired membership data: %d plans disabled, %d pending orders expired, %d queued subscriptions cancelled, %d active subscriptions ended",
			retiredPlans, expiredOrders, cancelledSubs, endedSubs))
	}
	return nil
}
