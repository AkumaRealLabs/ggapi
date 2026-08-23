package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type Midjourney struct {
	Id          int    `json:"id"`
	Code        int    `json:"code"`
	UserId      int    `json:"user_id" gorm:"index"`
	Action      string `json:"action" gorm:"type:varchar(40);index"`
	MjId        string `json:"mj_id" gorm:"index"`
	Prompt      string `json:"prompt"`
	PromptEn    string `json:"prompt_en"`
	Description string `json:"description"`
	State       string `json:"state"`
	SubmitTime  int64  `json:"submit_time" gorm:"index"`
	StartTime   int64  `json:"start_time" gorm:"index"`
	FinishTime  int64  `json:"finish_time" gorm:"index"`
	ImageUrl    string `json:"image_url"`
	VideoUrl    string `json:"video_url"`
	VideoUrls   string `json:"video_urls"`
	Status      string `json:"status" gorm:"type:varchar(20);index"`
	Progress    string `json:"progress" gorm:"type:varchar(30);index"`
	FailReason  string `json:"fail_reason"`
	ChannelId   int    `json:"channel_id"`
	Quota       int    `json:"quota"`
	Buttons     string `json:"buttons"`
	Properties  string `json:"properties"`

	TokenId          int `json:"-" gorm:"default:0"`
	BillingChannelId int `json:"-" gorm:"default:0"`
}

// TaskQueryParams 用于包含所有搜索条件的结构体，可以根据需求添加更多字段
type TaskQueryParams struct {
	ChannelID      string
	MjID           string
	StartTimestamp string
	EndTimestamp   string
}

func GetAllUserTask(userId int, startIdx int, num int, queryParams TaskQueryParams) []*Midjourney {
	var tasks []*Midjourney
	var err error

	// 初始化查询构建器
	query := DB.Where("user_id = ?", userId)

	if queryParams.MjID != "" {
		query = query.Where("mj_id = ?", queryParams.MjID)
	}
	if queryParams.StartTimestamp != "" {
		// 假设您已将前端传来的时间戳转换为数据库所需的时间格式，并处理了时间戳的验证和解析
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != "" {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 获取数据
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func GetAllTasks(startIdx int, num int, queryParams TaskQueryParams) []*Midjourney {
	var tasks []*Midjourney
	var err error

	// 初始化查询构建器
	query := DB

	// 添加过滤条件
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.MjID != "" {
		query = query.Where("mj_id = ?", queryParams.MjID)
	}
	if queryParams.StartTimestamp != "" {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != "" {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}

	// 获取数据
	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func GetAllUnFinishTasks() []*Midjourney {
	var tasks []*Midjourney
	var err error
	err = DB.Where(
		"(progress != ? AND status != ?) OR (status = ? AND quota != 0 AND billing_channel_id != 0)",
		"100%", "FAILURE", "FAILURE",
	).Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

// HasUnfinishedMidjourneyTasks reports whether a task needs polling or a failed
// task still has a durable refund marker.
func HasUnfinishedMidjourneyTasks() bool {
	var id int
	err := DB.Model(&Midjourney{}).
		Where(
			"(progress != ? AND status != ?) OR (status = ? AND quota != 0 AND billing_channel_id != 0)",
			"100%", "FAILURE", "FAILURE",
		).
		Limit(1).
		Pluck("id", &id).Error
	return err == nil && id != 0
}

func GetByOnlyMJId(mjId string) *Midjourney {
	var mj *Midjourney
	var err error
	err = DB.Where("mj_id = ?", mjId).First(&mj).Error
	if err != nil {
		return nil
	}
	return mj
}

func GetByMJId(userId int, mjId string) *Midjourney {
	var mj *Midjourney
	var err error
	err = DB.Where("user_id = ? and mj_id = ?", userId, mjId).First(&mj).Error
	if err != nil {
		return nil
	}
	return mj
}

func GetByMJIds(userId int, mjIds []string) []*Midjourney {
	var mj []*Midjourney
	var err error
	err = DB.Where("user_id = ? and mj_id in (?)", userId, mjIds).Find(&mj).Error
	if err != nil {
		return nil
	}
	return mj
}

func GetMjByuId(id int) *Midjourney {
	var mj *Midjourney
	var err error
	err = DB.Where("id = ?", id).First(&mj).Error
	if err != nil {
		return nil
	}
	return mj
}

func UpdateProgress(id int, progress string) error {
	return DB.Model(&Midjourney{}).Where("id = ?", id).Update("progress", progress).Error
}

func (midjourney *Midjourney) Insert() error {
	var err error
	err = DB.Create(midjourney).Error
	return err
}

func (midjourney *Midjourney) UpdateBillingState() error {
	return DB.Model(midjourney).
		Select("quota", "token_id", "billing_channel_id").
		Updates(midjourney).Error
}

func (midjourney *Midjourney) GetBillingChannelId() int {
	if midjourney.BillingChannelId > 0 {
		return midjourney.BillingChannelId
	}
	return midjourney.ChannelId
}

type MidjourneyRefund struct {
	Quota            int
	TokenId          int
	BillingChannelId int
}

// RefundBilling atomically restores the wallet, token, and usage counters
// before clearing the task's durable refund marker.
func (midjourney *Midjourney) RefundBilling() (*MidjourneyRefund, error) {
	var refund *MidjourneyRefund
	var tokenKey string
	var tokenActive bool
	err := DB.Transaction(func(tx *gorm.DB) error {
		var current Midjourney
		if err := lockForUpdate(tx).First(&current, midjourney.Id).Error; err != nil {
			return err
		}
		if current.Status != "FAILURE" || current.Quota == 0 {
			return nil
		}

		quota := current.Quota
		if err := increaseUserQuotaWithDB(tx, current.UserId, quota); err != nil {
			return err
		}
		if current.TokenId > 0 {
			var token Token
			tokenTx := tx.Unscoped()
			if err := lockForUpdate(tokenTx).First(&token, current.TokenId).Error; err != nil {
				return err
			}
			if err := increaseTokenQuotaWithDB(tokenTx, current.TokenId, quota); err != nil {
				return err
			}
			tokenKey = token.Key
			tokenActive = !token.DeletedAt.Valid
		}

		billingChannelId := current.GetBillingChannelId()
		if err := tx.Model(&User{}).Where("id = ?", current.UserId).
			Update("used_quota", gorm.Expr("used_quota - ?", quota)).Error; err != nil {
			return err
		}
		if billingChannelId > 0 {
			if err := tx.Model(&Channel{}).Where("id = ?", billingChannelId).
				Update("used_quota", gorm.Expr("used_quota - ?", quota)).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&Midjourney{}).Where("id = ?", current.Id).
			Update("quota", 0).Error; err != nil {
			return err
		}

		refund = &MidjourneyRefund{
			Quota:            quota,
			TokenId:          current.TokenId,
			BillingChannelId: billingChannelId,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	midjourney.Quota = 0
	if refund == nil {
		return nil, nil
	}
	if err := invalidateUserQuotaCacheForMutation(midjourney.UserId); err != nil {
		common.SysLog("failed to invalidate user quota cache after Midjourney refund: " + err.Error())
	}
	if tokenActive {
		if err := invalidateTokenCacheForMutation(tokenKey); err != nil {
			common.SysLog("failed to invalidate token cache after Midjourney refund: " + err.Error())
		}
	}
	return refund, nil
}

// UpdateWithStatus performs a conditional UPDATE guarded by fromStatus (CAS).
// Terminal states cannot be reversed, and billing fields are never written by
// status polling or notifications.
func (midjourney *Midjourney) UpdateWithStatus(fromStatus string) (bool, error) {
	if (fromStatus == "FAILURE" || fromStatus == "SUCCESS") && midjourney.Status != fromStatus {
		return false, nil
	}
	result := DB.Model(&Midjourney{}).
		Where("id = ? AND status = ?", midjourney.Id, fromStatus).
		Select(
			"code", "progress", "prompt_en", "state", "submit_time", "start_time", "finish_time",
			"image_url", "video_url", "video_urls", "status", "fail_reason", "buttons", "properties",
		).
		Updates(midjourney)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func MjBulkUpdate(mjIds []string, params map[string]any) error {
	return DB.Model(&Midjourney{}).
		Where("mj_id in (?)", mjIds).
		Updates(params).Error
}

func MjBulkUpdateByTaskIds(taskIDs []int, params map[string]any) error {
	return DB.Model(&Midjourney{}).
		Where("id in (?)", taskIDs).
		Updates(params).Error
}

// CountAllTasks returns total midjourney tasks for admin query
func CountAllTasks(queryParams TaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Midjourney{})
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.MjID != "" {
		query = query.Where("mj_id = ?", queryParams.MjID)
	}
	if queryParams.StartTimestamp != "" {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != "" {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}

// CountAllUserTask returns total midjourney tasks for user
func CountAllUserTask(userId int, queryParams TaskQueryParams) int64 {
	var total int64
	query := DB.Model(&Midjourney{}).Where("user_id = ?", userId)
	if queryParams.MjID != "" {
		query = query.Where("mj_id = ?", queryParams.MjID)
	}
	if queryParams.StartTimestamp != "" {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != "" {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	_ = query.Count(&total).Error
	return total
}
