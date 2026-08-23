package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type TopUp struct {
	Id              int     `json:"id"`
	UserId          int     `json:"user_id" gorm:"index"`
	Amount          int64   `json:"amount"`
	Money           float64 `json:"money"`
	TradeNo         string  `json:"trade_no" gorm:"unique;type:varchar(255);index"`
	PaymentMethod   string  `json:"payment_method" gorm:"type:varchar(50)"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50);default:''"`
	// AffInviterId is the invitee's inviter at order creation time. Settlement
	// uses this snapshot so later rebinds only affect subsequent orders.
	AffInviterId int    `json:"aff_inviter_id" gorm:"type:int;default:0;column:aff_inviter_id;index"`
	CreateTime   int64  `json:"create_time"`
	CompleteTime int64  `json:"complete_time"`
	Status       string `json:"status"`
}

const (
	PaymentMethodStripe       = "stripe"
	PaymentMethodCreem        = "creem"
	PaymentMethodWaffo        = "waffo"
	PaymentMethodWaffoPancake = "waffo_pancake"
	PaymentMethodBalance      = "balance"
)

const (
	PaymentProviderEpay         = "epay"
	PaymentProviderStripe       = "stripe"
	PaymentProviderCreem        = "creem"
	PaymentProviderWaffo        = "waffo"
	PaymentProviderWaffoPancake = "waffo_pancake"
	PaymentProviderBalance      = "balance"
)

var (
	ErrPaymentMethodMismatch   = errors.New("payment method mismatch")
	ErrTopUpNotFound           = errors.New("topup not found")
	ErrTopUpStatusInvalid      = errors.New("topup status invalid")
	ErrInvalidTopUpQuota       = errors.New("invalid top-up quota")
	ErrTopUpQuotaLimitExceeded = errors.New("top-up quota limit exceeded")
)

func (topUp *TopUp) Insert() error {
	if topUp == nil {
		return errors.New("topup is nil")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		topUp.AffInviterId = resolveAffiliateInviterIdTx(tx, topUp.UserId)
		return tx.Create(topUp).Error
	})
}

func topUpQuotaMaxCurrent(creditedQuota int) (int, error) {
	if creditedQuota <= 0 {
		return 0, ErrInvalidTopUpQuota
	}
	maxCurrentQuota, err := userQuotaMaxCurrent(creditedQuota)
	if err != nil {
		return 0, ErrInvalidTopUpQuota
	}
	return maxCurrentQuota, nil
}

// ValidateTopUpQuotaCapacity performs the user-facing pre-payment check. The
// settlement path repeats the same invariant with an atomic conditional
// update, because the wallet balance can change after checkout creation.
func ValidateTopUpQuotaCapacity(userId int, creditedQuota int) error {
	maxCurrentQuota, err := topUpQuotaMaxCurrent(creditedQuota)
	if err != nil {
		return err
	}

	var user User
	if err := DB.Select("quota").Where("id = ?", userId).First(&user).Error; err != nil {
		return err
	}
	if user.Quota > maxCurrentQuota {
		return ErrTopUpQuotaLimitExceeded
	}
	return nil
}

// creditTopUpQuota atomically enforces the int32 wallet ceiling while adding
// quota. Keeping the predicate and increment in one UPDATE prevents two
// concurrent callbacks from both passing a separate read/check.
func creditTopUpQuota(tx *gorm.DB, userId int, creditedQuota int, updates map[string]interface{}) error {
	maxCurrentQuota, err := topUpQuotaMaxCurrent(creditedQuota)
	if err != nil {
		return err
	}

	updateFields := make(map[string]interface{}, len(updates)+1)
	for key, value := range updates {
		updateFields[key] = value
	}
	updateFields["quota"] = gorm.Expr("quota + ?", creditedQuota)

	result := tx.Model(&User{}).
		Where("id = ? AND quota <= ?", userId, maxCurrentQuota).
		Updates(updateFields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}

	var count int64
	if err := tx.Model(&User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return ErrTopUpQuotaLimitExceeded
}

func (topUp *TopUp) Update() error {
	var err error
	err = DB.Save(topUp).Error
	return err
}

func GetTopUpById(id int) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("id = ?", id).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

func GetTopUpByTradeNo(tradeNo string) *TopUp {
	var topUp *TopUp
	var err error
	err = DB.Where("trade_no = ?", tradeNo).First(&topUp).Error
	if err != nil {
		return nil
	}
	return topUp
}

type topUpCompletionOptions struct {
	ExpectedPaymentProvider string
	ActualPaymentMethod     string
	StripeCustomer          string
	CustomerEmail           string
}

func calculateTopUpQuota(topUp *TopUp) (int, error) {
	if topUp == nil {
		return 0, errors.New("充值订单不存在")
	}
	// Provider amount semantics are historical: Creem stores quota directly,
	// Stripe stores paid money, and the remaining gateways store money units.
	switch topUp.PaymentProvider {
	case PaymentProviderCreem:
		if topUp.Amount <= 0 {
			return 0, errors.New("无效的充值额度")
		}
		quota, clamp := common.QuotaFromDecimalChecked(decimal.NewFromInt(topUp.Amount))
		if clamp != nil {
			return 0, errors.New("充值额度超出系统范围")
		}
		return quota, nil
	case PaymentProviderStripe:
		return common.QuotaFromMoney(topUp.Money)
	default:
		if topUp.Amount <= 0 || common.QuotaPerUnit <= 0 {
			return 0, errors.New("无效的充值额度")
		}
		// Match historical gateway settlement: Amount * QuotaPerUnit truncates toward zero.
		intPart := decimal.NewFromInt(topUp.Amount).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
		if intPart > int64(common.MaxQuota) || intPart <= 0 {
			return 0, errors.New("充值额度超出系统范围")
		}
		return int(intPart), nil
	}
}

func completeTopUp(tradeNo string, options topUpCompletionOptions) (*TopUp, int, bool, error) {
	if tradeNo == "" {
		return nil, 0, false, errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}
	var completed TopUp
	var quotaToAdd int
	alreadyCompleted := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(&completed).Error; err != nil {
			return ErrTopUpNotFound
		}
		if options.ExpectedPaymentProvider != "" && completed.PaymentProvider != options.ExpectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if completed.Status == common.TopUpStatusSuccess {
			alreadyCompleted = true
			return nil
		}
		if completed.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		var err error
		quotaToAdd, err = calculateTopUpQuota(&completed)
		if err != nil {
			return err
		}
		if quotaToAdd <= 0 {
			return errors.New("无效的充值额度")
		}
		if _, _, err := SettleAffiliateCommissionTx(
			tx,
			completed.UserId,
			AffiliateCommissionSourceTopUp,
			completed.Id,
			completed.TradeNo,
			completed.PaymentProvider,
			quotaToAdd,
			completed.AffInviterId,
		); err != nil {
			return err
		}

		completed.CompleteTime = common.GetTimestamp()
		completed.Status = common.TopUpStatusSuccess
		if options.ActualPaymentMethod != "" {
			completed.PaymentMethod = options.ActualPaymentMethod
		}
		if err := tx.Save(&completed).Error; err != nil {
			return err
		}

		updates := map[string]any{}
		if options.StripeCustomer != "" {
			updates["stripe_customer"] = options.StripeCustomer
		}
		if err := creditTopUpQuota(tx, completed.UserId, quotaToAdd, updates); err != nil {
			return err
		}
		if options.CustomerEmail != "" {
			if err := tx.Model(&User{}).
				Where("id = ? AND (email = '' OR email IS NULL)", completed.UserId).
				Update("email", options.CustomerEmail).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, false, err
	}
	if !alreadyCompleted {
		syncCreditUserQuotaCache(completed.UserId, quotaToAdd, completed.PaymentProvider+" topup")
	}
	return &completed, quotaToAdd, alreadyCompleted, nil
}

func finishTopUp(tradeNo string, options topUpCompletionOptions, logIfNew func(*TopUp, int)) error {
	topUp, quota, alreadyCompleted, err := completeTopUp(tradeNo, options)
	if err != nil {
		return err
	}
	if !alreadyCompleted && logIfNew != nil {
		logIfNew(topUp, quota)
	}
	return nil
}

func UpdatePendingTopUpStatus(tradeNo string, expectedPaymentProvider string, targetStatus string) error {
	if tradeNo == "" {
		return errors.New("未提供支付单号")
	}

	refCol := "`trade_no`"
	if common.UsingMainDatabase(common.DatabaseTypePostgreSQL) {
		refCol = `"trade_no"`
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		topUp := &TopUp{}
		if err := lockForUpdate(tx).Where(refCol+" = ?", tradeNo).First(topUp).Error; err != nil {
			return ErrTopUpNotFound
		}
		if expectedPaymentProvider != "" && topUp.PaymentProvider != expectedPaymentProvider {
			return ErrPaymentMethodMismatch
		}
		if topUp.Status != common.TopUpStatusPending {
			return ErrTopUpStatusInvalid
		}

		topUp.Status = targetStatus
		return tx.Save(topUp).Error
	})
}

func Recharge(referenceId string, customerId string, callerIp string) error {
	err := finishTopUp(referenceId, topUpCompletionOptions{
		ExpectedPaymentProvider: PaymentProviderStripe,
		StripeCustomer:          customerId,
	}, func(topUp *TopUp, quota int) {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%d", logger.FormatQuota(quota), topUp.Amount), callerIp, topUp.PaymentMethod, PaymentMethodStripe)
	})
	if err != nil {
		common.SysError("topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	return nil
}

// topUpQueryWindowSeconds 限制充值记录查询的时间窗口（秒）。
const topUpQueryWindowSeconds int64 = 30 * 24 * 60 * 60

// topUpQueryCutoff 返回允许查询的最早 create_time（秒级 Unix 时间戳）。
func topUpQueryCutoff() int64 {
	return common.GetTimestamp() - topUpQueryWindowSeconds
}

func GetUserTopUps(userId int, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	cutoff := topUpQueryCutoff()

	// Get total count within transaction
	err = tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, cutoff).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated topups within same transaction
	err = tx.Where("user_id = ? AND create_time >= ?", userId, cutoff).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// GetAllTopUps 获取全平台的充值记录（管理员使用，不限制时间窗口）
func GetAllTopUps(pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err = tx.Model(&TopUp{}).Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	return topups, total, nil
}

// searchTopUpCountHardLimit 搜索充值记录时 COUNT 的安全上限，
// 防止对超大表执行无界 COUNT 触发 DoS。
const searchTopUpCountHardLimit = 10000

// SearchUserTopUps 按订单号搜索某用户的充值记录
func SearchUserTopUps(userId int, keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{}).Where("user_id = ? AND create_time >= ?", userId, topUpQueryCutoff())
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// SearchAllTopUps 按订单号搜索全平台充值记录（管理员使用，不限制时间窗口）
func SearchAllTopUps(keyword string, pageInfo *common.PageInfo) (topups []*TopUp, total int64, err error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := tx.Model(&TopUp{})
	if keyword != "" {
		pattern, perr := sanitizeLikePattern(keyword)
		if perr != nil {
			tx.Rollback()
			return nil, 0, perr
		}
		query = query.Where("trade_no LIKE ? ESCAPE '!'", pattern)
	}

	if err = query.Limit(searchTopUpCountHardLimit).Count(&total).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to count search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&topups).Error; err != nil {
		tx.Rollback()
		common.SysError("failed to search topups: " + err.Error())
		return nil, 0, errors.New("搜索充值记录失败")
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return topups, total, nil
}

// ManualCompleteTopUp 管理员手动完成订单并给用户充值
func ManualCompleteTopUp(tradeNo string, callerIp string) error {
	return finishTopUp(tradeNo, topUpCompletionOptions{}, func(topUp *TopUp, quotaToAdd int) {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("管理员补单成功，充值金额: %v，支付金额：%f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, "admin")
	})
}

func RechargeCreem(referenceId string, customerEmail string, customerName string, callerIp string) error {
	err := finishTopUp(referenceId, topUpCompletionOptions{
		ExpectedPaymentProvider: PaymentProviderCreem,
		CustomerEmail:           customerEmail,
	}, func(topUp *TopUp, quota int) {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("使用Creem充值成功，充值额度: %v，支付金额：%.2f", quota, topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodCreem)
	})
	if err != nil {
		common.SysError("creem topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	return nil
}

func RechargeWaffo(tradeNo string, callerIp string) error {
	err := finishTopUp(tradeNo, topUpCompletionOptions{
		ExpectedPaymentProvider: PaymentProviderWaffo,
	}, func(topUp *TopUp, quotaToAdd int) {
		RecordTopupLog(topUp.UserId, fmt.Sprintf("Waffo充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentMethodWaffo)
	})
	if err != nil {
		common.SysError("waffo topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	return nil
}

func RechargeWaffoPancake(tradeNo string) error {
	err := finishTopUp(tradeNo, topUpCompletionOptions{
		ExpectedPaymentProvider: PaymentProviderWaffoPancake,
	}, func(topUp *TopUp, quotaToAdd int) {
		RecordLog(topUp.UserId, LogTypeTopup, fmt.Sprintf("Waffo Pancake充值成功，充值额度: %v，支付金额: %.2f", logger.FormatQuota(quotaToAdd), topUp.Money))
	})
	if err != nil {
		common.SysError("waffo pancake topup failed: " + err.Error())
		return errors.New("充值失败，请稍后重试")
	}
	return nil
}

// RechargeEpay 原子完成易支付订单：订单行锁、状态校验、成功更新、用户额度增加
// 与返佣结算在同一个事务内完成，因此同一订单的并发/重复回调（包括多实例部署下）
// 最多充值一次。alreadyDone=true 表示订单此前已完成，本次为幂等重复回调。
// 进程内的 LockOrder 只是优化，正确性由本函数的数据库行锁保证。
func RechargeEpay(tradeNo string, actualPaymentMethod string, callerIp string) (alreadyDone bool, err error) {
	topUp, quotaToAdd, alreadyDone, err := completeTopUp(tradeNo, topUpCompletionOptions{
		ExpectedPaymentProvider: PaymentProviderEpay,
		ActualPaymentMethod:     actualPaymentMethod,
	})
	if err != nil {
		if !errors.Is(err, ErrTopUpNotFound) && !errors.Is(err, ErrPaymentMethodMismatch) && !errors.Is(err, ErrTopUpStatusInvalid) {
			common.SysError("epay topup failed: " + err.Error())
		}
		return false, err
	}
	if alreadyDone {
		return true, nil
	}
	RecordTopupLog(topUp.UserId, fmt.Sprintf("使用在线充值成功，充值金额: %v，支付金额：%f", logger.LogQuota(quotaToAdd), topUp.Money), callerIp, topUp.PaymentMethod, PaymentProviderEpay)
	return false, nil
}
