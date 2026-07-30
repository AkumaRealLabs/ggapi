package model

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	AffiliateCommissionSourceTopUp        = "topup"
	AffiliateCommissionSourceSubscription = "subscription"
	AffiliateCommissionSourceRedemption   = "redemption"
)

var (
	affiliateCodePattern           = regexp.MustCompile(`^[a-z0-9_-]{4,32}$`)
	ErrAffiliateAlreadyBound       = errors.New("邀请人已绑定，不能重复修改")
	ErrAffiliateCodeNotFound       = errors.New("邀请码不存在")
	ErrAffiliateCodeUnavailable    = errors.New("邀请码已被使用")
	ErrAffiliateCodeAmbiguous      = errors.New("邀请码存在大小写歧义，请使用精确邀请码")
	ErrAffiliateInviterUnavailable = errors.New("邀请人不可用")
	ErrAffiliateRelationCycle      = errors.New("邀请关系不能形成循环")
	ErrAffiliateSelfInvite         = errors.New("不能邀请自己")
	ErrAffiliateRelationChanged    = errors.New("邀请关系已变化，请重试")
)

type AffiliateCommission struct {
	Id              int     `json:"id"`
	InviterId       int     `json:"inviter_id" gorm:"index"`
	InviteeId       int     `json:"invitee_id" gorm:"index"`
	SourceType      string  `json:"source_type" gorm:"type:varchar(32);uniqueIndex:idx_affiliate_commission_source,priority:1"`
	SourceId        int     `json:"source_id" gorm:"uniqueIndex:idx_affiliate_commission_source,priority:2"`
	SourceOrderNo   string  `json:"source_order_no" gorm:"type:varchar(255);index"`
	PaymentProvider string  `json:"payment_provider" gorm:"type:varchar(50)"`
	BaseQuota       int     `json:"base_quota" gorm:"type:int"`
	CommissionRate  float64 `json:"commission_rate"`
	CommissionQuota int     `json:"commission_quota" gorm:"type:int"`
	SettledAt       int64   `json:"settled_at" gorm:"index"`
}

type AffiliateUserSummary struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	AffCode  string `json:"aff_code"`
	Status   int    `json:"status"`
}

type AffiliateUserDetail struct {
	Id                      int                   `json:"id"`
	Username                string                `json:"username"`
	AffCode                 string                `json:"aff_code"`
	Inviter                 *AffiliateUserSummary `json:"inviter"`
	DefaultCommissionRate   float64               `json:"default_commission_rate"`
	CustomCommissionRate    *float64              `json:"custom_commission_rate"`
	EffectiveCommissionRate float64               `json:"effective_commission_rate"`
	AffQuota                int                   `json:"aff_quota"`
	AffHistoryQuota         int                   `json:"aff_history_quota"`
	InviteCount             int64                 `json:"invite_count"`
}

type AffiliateCommissionListItem struct {
	Id              int     `json:"id"`
	InviterId       int     `json:"inviter_id"`
	InviterUsername string  `json:"inviter_username"`
	InviteeId       int     `json:"invitee_id"`
	InviteeUsername string  `json:"invitee_username"`
	SourceType      string  `json:"source_type"`
	SourceId        int     `json:"source_id"`
	SourceOrderNo   string  `json:"source_order_no"`
	PaymentProvider string  `json:"payment_provider"`
	BaseQuota       int     `json:"base_quota"`
	CommissionRate  float64 `json:"commission_rate"`
	CommissionQuota int     `json:"commission_quota"`
	SettledAt       int64   `json:"settled_at"`
}

type AffiliateCommissionFilters struct {
	InviterId  int
	OrderNo    string
	SourceType string
	Inviter    string
	Invitee    string
}

func ValidateAffiliateCommissionRate(rate float64) error {
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 || rate > 100 {
		return errors.New("返佣比例必须在 0.00 到 100.00 之间")
	}
	scaled := decimal.NewFromFloat(rate).Mul(decimal.NewFromInt(100))
	if !scaled.Equal(scaled.Truncate(0)) {
		return errors.New("返佣比例最多保留两位小数")
	}
	return nil
}

func NormalizeAffiliateCode(code string) (string, error) {
	code = strings.ToLower(strings.TrimSpace(code))
	if !affiliateCodePattern.MatchString(code) {
		return "", errors.New("邀请码只能包含小写字母、数字、下划线或短横线，长度为 4 到 32 位")
	}
	return code, nil
}

func effectiveAffiliateCommissionRate(user *User) float64 {
	if user != nil && user.AffCommissionRate != nil {
		return *user.AffCommissionRate
	}
	return common.AffiliateCommissionRate
}

// resolveAffiliateInviterIdTx snapshots the invitee's current inviter inside
// the same transaction that creates the payment order, so concurrent rebinds
// cannot be applied to an order that starts after the rebind commits.
func resolveAffiliateInviterIdTx(tx *gorm.DB, userId int) int {
	if tx == nil || userId <= 0 {
		return 0
	}
	var inviterId int
	if err := tx.Model(&User{}).Select("inviter_id").Where("id = ?", userId).Scan(&inviterId).Error; err != nil {
		return 0
	}
	return inviterId
}

// findUserIdByAffiliateCode prefers an exact code match. Case-folded lookup is
// only used when exactly one row matches, so mixed-case legacy duplicates cannot
// silently credit the wrong inviter on PostgreSQL/SQLite.
func findUserIdByAffiliateCode(db *gorm.DB, code string) (int, error) {
	if db == nil {
		db = DB
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, ErrAffiliateCodeNotFound
	}

	var exact User
	err := db.Select("id").Where("aff_code = ?", code).First(&exact).Error
	if err == nil {
		return exact.Id, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	var matches []User
	if err := db.Select("id", "aff_code").Where("LOWER(aff_code) = ?", strings.ToLower(code)).Find(&matches).Error; err != nil {
		return 0, err
	}
	switch len(matches) {
	case 0:
		return 0, ErrAffiliateCodeNotFound
	case 1:
		return matches[0].Id, nil
	default:
		return 0, ErrAffiliateCodeAmbiguous
	}
}

func inviteUser(inviterId int, reward int) error {
	updates := map[string]any{
		"aff_count": gorm.Expr("aff_count + 1"),
	}
	if reward > 0 {
		updates["aff_quota"] = gorm.Expr("aff_quota + ?", reward)
		updates["aff_history"] = gorm.Expr("aff_history + ?", reward)
	}
	return DB.Model(&User{}).Where("id = ?", inviterId).Updates(updates).Error
}

func finishAffiliateInvite(inviteeId int, inviterId int) {
	if inviterId == 0 {
		return
	}
	inviterReward := 0
	if operation_setting.IsPaymentComplianceConfirmed() {
		if common.QuotaForInvitee > 0 {
			_ = IncreaseUserQuota(inviteeId, common.QuotaForInvitee, true)
			RecordLog(inviteeId, LogTypeSystem, fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee)))
		}
		inviterReward = common.QuotaForInviter
	}
	if err := inviteUser(inviterId, inviterReward); err != nil {
		common.SysError("failed to update inviter reward: " + err.Error())
		return
	}
	if inviterReward > 0 {
		RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请用户赠送 %s", logger.LogQuota(inviterReward)))
	}
}

func GetAffiliateUserDetail(userId int) (*AffiliateUserDetail, error) {
	var user User
	if err := DB.First(&user, userId).Error; err != nil {
		return nil, err
	}
	if user.AffCode == "" {
		code, err := EnsureAffiliateCode(userId)
		if err != nil {
			return nil, err
		}
		user.AffCode = code
	}

	detail := &AffiliateUserDetail{
		Id:                      user.Id,
		Username:                user.Username,
		AffCode:                 user.AffCode,
		DefaultCommissionRate:   common.AffiliateCommissionRate,
		CustomCommissionRate:    user.AffCommissionRate,
		EffectiveCommissionRate: effectiveAffiliateCommissionRate(&user),
		AffQuota:                user.AffQuota,
		AffHistoryQuota:         user.AffHistoryQuota,
	}
	if err := DB.Model(&User{}).Where("inviter_id = ?", user.Id).Count(&detail.InviteCount).Error; err != nil {
		return nil, err
	}
	if user.InviterId > 0 {
		// Include soft-deleted inviters so the invitee still appears bound and
		// is not offered a bind form that always fails with already-bound.
		var inviter AffiliateUserSummary
		err := DB.Unscoped().Model(&User{}).
			Select("id", "username", "aff_code", "status").
			First(&inviter, user.InviterId).Error
		if err == nil {
			detail.Inviter = &inviter
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			detail.Inviter = &AffiliateUserSummary{Id: user.InviterId}
		} else {
			return nil, err
		}
	}
	return detail, nil
}

func EnsureAffiliateCode(userId int) (string, error) {
	if userId <= 0 {
		return "", errors.New("无效的用户 ID")
	}
	var user User
	if err := DB.Select("id", "aff_code").First(&user, userId).Error; err != nil {
		return "", err
	}
	if user.AffCode != "" {
		return user.AffCode, nil
	}

	for range 10 {
		candidate := strings.ToLower(common.GetRandomString(8))
		var count int64
		if err := DB.Unscoped().Model(&User{}).Where("LOWER(aff_code) = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count > 0 {
			continue
		}

		result := DB.Model(&User{}).
			Where("id = ? AND (aff_code = '' OR aff_code IS NULL)", userId).
			Update("aff_code", candidate)
		if isUniqueViolation(result.Error) {
			continue
		}
		if result.Error != nil {
			return "", result.Error
		}
		if result.RowsAffected > 0 {
			return candidate, nil
		}

		if err := DB.Select("aff_code").First(&user, userId).Error; err != nil {
			return "", err
		}
		if user.AffCode != "" {
			return user.AffCode, nil
		}
	}
	return "", errors.New("生成邀请码失败，请重试")
}

func loadAffiliateRelationSnapshot(tx *gorm.DB, userId int, inviterId int) (map[int]int, []int, error) {
	expectedInviters := map[int]int{}
	userIds := make([]int, 0, 4)

	var user User
	if err := tx.Unscoped().Select("id", "inviter_id", "deleted_at").First(&user, userId).Error; err != nil {
		return nil, nil, err
	}
	expectedInviters[user.Id] = user.InviterId
	userIds = append(userIds, user.Id)
	if user.InviterId > 0 {
		userIds = append(userIds, user.InviterId)
	}

	seen := map[int]struct{}{}
	currentId := inviterId
	for currentId > 0 {
		if _, ok := seen[currentId]; ok {
			return nil, nil, ErrAffiliateRelationCycle
		}
		seen[currentId] = struct{}{}

		var current User
		err := tx.Unscoped().Select("id", "inviter_id", "deleted_at").First(&current, currentId).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if currentId == inviterId {
				return nil, nil, ErrAffiliateInviterUnavailable
			}
			break
		}
		if err != nil {
			return nil, nil, err
		}
		if currentId == inviterId && current.DeletedAt.Valid {
			return nil, nil, ErrAffiliateInviterUnavailable
		}
		expectedInviters[current.Id] = current.InviterId
		userIds = append(userIds, current.Id)
		currentId = current.InviterId
	}

	return expectedInviters, userIds, nil
}

func lockAffiliateUsersById(tx *gorm.DB, userIds []int) ([]User, error) {
	uniqueIds := make(map[int]struct{}, len(userIds))
	for _, userId := range userIds {
		if userId > 0 {
			uniqueIds[userId] = struct{}{}
		}
	}
	userIds = userIds[:0]
	for userId := range uniqueIds {
		userIds = append(userIds, userId)
	}
	sort.Ints(userIds)

	var users []User
	err := lockForUpdate(tx.Unscoped()).
		Select("id", "username", "status", "aff_code", "aff_count", "inviter_id", "aff_commission_rate", "deleted_at").
		Where("id IN ?", userIds).
		Order("id asc").
		Find(&users).Error
	return users, err
}

func verifyAffiliateRelationSnapshot(users []User, expectedInviters map[int]int) error {
	for userId, inviterId := range expectedInviters {
		user := affiliateUserById(users, userId)
		if user == nil || user.InviterId != inviterId {
			return ErrAffiliateRelationChanged
		}
	}
	return nil
}

func lockAffiliatePath(tx *gorm.DB, userId int, inviterId int) ([]User, error) {
	expectedInviters, userIds, err := loadAffiliateRelationSnapshot(tx, userId, inviterId)
	if err != nil {
		return nil, err
	}
	users, err := lockAffiliateUsersById(tx, userIds)
	if err != nil {
		return nil, err
	}
	if err := verifyAffiliateRelationSnapshot(users, expectedInviters); err != nil {
		return nil, err
	}
	return users, nil
}

func affiliateUserById(users []User, userId int) *User {
	for i := range users {
		if users[i].Id == userId {
			return &users[i]
		}
	}
	return nil
}

func validateAffiliateRelation(users []User, userId int, inviterId int) error {
	if inviterId == 0 {
		return nil
	}
	if userId == inviterId {
		return ErrAffiliateSelfInvite
	}
	inviter := affiliateUserById(users, inviterId)
	if inviter == nil || inviter.DeletedAt.Valid || inviter.Status != common.UserStatusEnabled {
		return ErrAffiliateInviterUnavailable
	}

	seen := map[int]struct{}{}
	currentId := inviterId
	for currentId > 0 {
		if currentId == userId {
			return ErrAffiliateRelationCycle
		}
		if _, ok := seen[currentId]; ok {
			return ErrAffiliateRelationCycle
		}
		seen[currentId] = struct{}{}
		current := affiliateUserById(users, currentId)
		if current == nil {
			return nil
		}
		currentId = current.InviterId
	}
	return nil
}

func updateAffiliateRelationTx(tx *gorm.DB, user *User, inviterId int) error {
	if user.InviterId == inviterId {
		return nil
	}
	if user.InviterId > 0 {
		if err := tx.Unscoped().Model(&User{}).Where("id = ?", user.InviterId).
			Update("aff_count", gorm.Expr("CASE WHEN aff_count > 0 THEN aff_count - 1 ELSE 0 END")).Error; err != nil {
			return err
		}
	}
	if inviterId > 0 {
		if err := tx.Unscoped().Model(&User{}).Where("id = ?", inviterId).
			Update("aff_count", gorm.Expr("aff_count + 1")).Error; err != nil {
			return err
		}
	}
	user.InviterId = inviterId
	return nil
}

func removeAffiliateRelationBeforeDeleteTx(tx *gorm.DB, userId int) error {
	users, err := lockAffiliatePath(tx, userId, 0)
	if err != nil {
		return err
	}
	user := affiliateUserById(users, userId)
	if user == nil || user.DeletedAt.Valid || user.InviterId <= 0 {
		return nil
	}
	return tx.Unscoped().Model(&User{}).Where("id = ?", user.InviterId).
		Update("aff_count", gorm.Expr("CASE WHEN aff_count > 0 THEN aff_count - 1 ELSE 0 END")).Error
}

func BindAffiliateInviterByCode(userId int, code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return ErrAffiliateCodeNotFound
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		inviterId, err := findUserIdByAffiliateCode(tx, code)
		if err != nil {
			return err
		}
		users, err := lockAffiliatePath(tx, userId, inviterId)
		if err != nil {
			return err
		}
		user := affiliateUserById(users, userId)
		if user == nil || user.DeletedAt.Valid {
			return gorm.ErrRecordNotFound
		}
		if user.InviterId != 0 {
			return ErrAffiliateAlreadyBound
		}

		inviter := affiliateUserById(users, inviterId)
		if inviter == nil || !strings.EqualFold(inviter.AffCode, code) {
			return ErrAffiliateCodeNotFound
		}
		if inviter.Status != common.UserStatusEnabled {
			return ErrAffiliateInviterUnavailable
		}
		if err := validateAffiliateRelation(users, user.Id, inviter.Id); err != nil {
			return err
		}
		if err := updateAffiliateRelationTx(tx, user, inviter.Id); err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", user.Id).Update("inviter_id", inviter.Id).Error
	})
}

func SetAffiliateUserConfig(userId int, affCode string, customRate *float64, inviterId int) error {
	if userId <= 0 || inviterId < 0 {
		return errors.New("无效的用户 ID")
	}
	if customRate != nil {
		if err := ValidateAffiliateCommissionRate(*customRate); err != nil {
			return err
		}
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		users, err := lockAffiliatePath(tx, userId, inviterId)
		if err != nil {
			return err
		}
		user := affiliateUserById(users, userId)
		if user == nil || user.DeletedAt.Valid {
			return gorm.ErrRecordNotFound
		}
		normalizedCode := user.AffCode
		if !strings.EqualFold(strings.TrimSpace(affCode), user.AffCode) {
			var err error
			normalizedCode, err = NormalizeAffiliateCode(affCode)
			if err != nil {
				return err
			}
		}
		if err := validateAffiliateRelation(users, userId, inviterId); err != nil {
			return err
		}

		var count int64
		if err := tx.Unscoped().Model(&User{}).
			Where("LOWER(aff_code) = ? AND id <> ?", strings.ToLower(normalizedCode), userId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrAffiliateCodeUnavailable
		}
		if err := updateAffiliateRelationTx(tx, user, inviterId); err != nil {
			return err
		}

		result := tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]any{
			"aff_code":            normalizedCode,
			"aff_commission_rate": customRate,
			"inviter_id":          inviterId,
		})
		if isUniqueViolation(result.Error) {
			return ErrAffiliateCodeUnavailable
		}
		return result.Error
	})
}

func SearchAffiliateUsers(keyword string, limit int) ([]AffiliateUserSummary, error) {
	keyword = strings.TrimSpace(keyword)
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	query := DB.Model(&User{}).Select("id", "username", "aff_code", "status").Order("id asc").Limit(limit)
	if keyword != "" {
		like := "%" + strings.ToLower(keyword) + "%"
		if id, err := strconv.Atoi(keyword); err == nil && id > 0 {
			query = query.Where("LOWER(username) LIKE ? OR LOWER(aff_code) LIKE ? OR id = ?", like, like, id)
		} else {
			query = query.Where("LOWER(username) LIKE ? OR LOWER(aff_code) LIKE ?", like, like)
		}
	}
	var users []AffiliateUserSummary
	return users, query.Find(&users).Error
}

func addAffiliateIdentityFilter(query *gorm.DB, alias string, value string) *gorm.DB {
	value = strings.TrimSpace(value)
	if value == "" {
		return query
	}
	like := "%" + strings.ToLower(value) + "%"
	condition := fmt.Sprintf("(LOWER(%s.username) LIKE ? OR LOWER(%s.aff_code) LIKE ?", alias, alias)
	args := []any{like, like}
	if id, err := strconv.Atoi(value); err == nil && id > 0 {
		condition += fmt.Sprintf(" OR %s.id = ?", alias)
		args = append(args, id)
	}
	condition += ")"
	return query.Where(condition, args...)
}

func ListAffiliateCommissions(pageInfo *common.PageInfo, filters AffiliateCommissionFilters) ([]AffiliateCommissionListItem, int64, error) {
	query := DB.Table("affiliate_commissions AS commissions").
		Joins("LEFT JOIN users AS inviter ON inviter.id = commissions.inviter_id").
		Joins("LEFT JOIN users AS invitee ON invitee.id = commissions.invitee_id")
	if filters.InviterId > 0 {
		query = query.Where("commissions.inviter_id = ?", filters.InviterId)
	}
	if filters.OrderNo != "" {
		query = query.Where("commissions.source_order_no LIKE ?", "%"+strings.TrimSpace(filters.OrderNo)+"%")
	}
	if filters.SourceType != "" {
		query = query.Where("commissions.source_type = ?", filters.SourceType)
	}
	query = addAffiliateIdentityFilter(query, "inviter", filters.Inviter)
	query = addAffiliateIdentityFilter(query, "invitee", filters.Invitee)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]AffiliateCommissionListItem, 0)
	err := query.Select(
		"commissions.id, commissions.inviter_id, inviter.username AS inviter_username, " +
			"commissions.invitee_id, invitee.username AS invitee_username, commissions.source_type, " +
			"commissions.source_id, commissions.source_order_no, commissions.payment_provider, " +
			"commissions.base_quota, commissions.commission_rate, commissions.commission_quota, commissions.settled_at",
	).Order("commissions.id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Scan(&items).Error
	return items, total, err
}

// SettleAffiliateCommissionTx credits the inviter snapshot supplied by the
// caller. Payment orders use their creation-time snapshot, while redemptions
// use the relationship read inside the redemption transaction.
// Overflow on aff_quota/aff_history skips commission without failing the
// parent transaction.
func SettleAffiliateCommissionTx(tx *gorm.DB, inviteeId int, sourceType string, sourceId int, orderNo string, paymentProvider string, baseQuota int, affInviterId int) (*AffiliateCommission, bool, error) {
	if tx == nil {
		return nil, false, errors.New("tx is nil")
	}
	if sourceType != AffiliateCommissionSourceTopUp &&
		sourceType != AffiliateCommissionSourceSubscription &&
		sourceType != AffiliateCommissionSourceRedemption {
		return nil, false, errors.New("无效的返佣来源")
	}
	if inviteeId <= 0 || sourceId <= 0 || baseQuota <= 0 || !operation_setting.IsPaymentComplianceConfirmed() {
		return nil, false, nil
	}
	if sourceType != AffiliateCommissionSourceRedemption && affInviterId <= 0 {
		return nil, false, nil
	}
	if inviteeId == affInviterId {
		return nil, false, nil
	}

	userIds := []int{inviteeId}
	if affInviterId > 0 {
		userIds = append(userIds, affInviterId)
	}
	sort.Ints(userIds)
	var locked []User
	if err := lockForUpdate(tx).
		Select("id", "status", "inviter_id", "aff_commission_rate", "aff_quota", "aff_history").
		Where("id IN ?", userIds).
		Order("id asc").
		Find(&locked).Error; err != nil {
		return nil, false, err
	}
	invitee := affiliateUserById(locked, inviteeId)
	if invitee == nil {
		return nil, false, gorm.ErrRecordNotFound
	}
	// Payment orders intentionally keep their creation-time inviter. A
	// redemption snapshot must still match after the user rows are locked.
	if sourceType == AffiliateCommissionSourceRedemption && invitee.InviterId != affInviterId {
		return nil, false, ErrAffiliateRelationChanged
	}
	if affInviterId <= 0 {
		return nil, false, nil
	}
	inviter := affiliateUserById(locked, affInviterId)
	if inviter == nil || inviter.Status != common.UserStatusEnabled {
		return nil, false, nil
	}

	rate := effectiveAffiliateCommissionRate(inviter)
	if err := ValidateAffiliateCommissionRate(rate); err != nil || rate == 0 {
		return nil, false, nil
	}
	commissionQuota, clamp := common.QuotaFromDecimalChecked(
		decimal.NewFromInt(int64(baseQuota)).Mul(decimal.NewFromFloat(rate)).Div(decimal.NewFromInt(100)),
	)
	if clamp != nil {
		return nil, false, errors.New("返佣额度超出系统范围")
	}
	if commissionQuota <= 0 {
		return nil, false, nil
	}
	if int64(inviter.AffQuota)+int64(commissionQuota) > math.MaxInt32 ||
		int64(inviter.AffHistoryQuota)+int64(commissionQuota) > math.MaxInt32 {
		common.SysError(fmt.Sprintf(
			"skip affiliate commission due to aff balance overflow inviter=%d invitee=%d source=%s/%d commission=%d aff_quota=%d aff_history=%d",
			inviter.Id, invitee.Id, sourceType, sourceId, commissionQuota, inviter.AffQuota, inviter.AffHistoryQuota,
		))
		return nil, false, nil
	}

	commission := &AffiliateCommission{
		InviterId:       inviter.Id,
		InviteeId:       invitee.Id,
		SourceType:      sourceType,
		SourceId:        sourceId,
		SourceOrderNo:   orderNo,
		PaymentProvider: paymentProvider,
		BaseQuota:       baseQuota,
		CommissionRate:  rate,
		CommissionQuota: commissionQuota,
		SettledAt:       common.GetTimestamp(),
	}
	if err := tx.Create(commission).Error; err != nil {
		return nil, false, err
	}
	if err := tx.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"aff_quota":   gorm.Expr("aff_quota + ?", commissionQuota),
		"aff_history": gorm.Expr("aff_history + ?", commissionQuota),
	}).Error; err != nil {
		return nil, false, err
	}
	return commission, true, nil
}
