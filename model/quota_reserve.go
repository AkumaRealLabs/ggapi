package model

import (
	"context"
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type cacheQuotaResult int

const (
	cacheQuotaInsufficient cacheQuotaResult = iota
	cacheQuotaOK
	cacheQuotaMiss
)

const userQuotaReserveScript = `
if redis.call('EXISTS', KEYS[2]) == 1
  or tonumber(redis.call('HGET', KEYS[1], 'Id') or '0') ~= tonumber(ARGV[2])
  or tonumber(redis.call('HGET', KEYS[1], 'CacheSchema') or '0') ~= tonumber(ARGV[3])
  or redis.call('HEXISTS', KEYS[1], 'Quota') == 0 then
  return -1
end
local quota = tonumber(redis.call('HGET', KEYS[1], 'Quota'))
if quota == nil or quota < tonumber(ARGV[1]) then
  return 0
end
redis.call('HINCRBY', KEYS[1], 'Quota', -tonumber(ARGV[1]))
redis.call('INCR', KEYS[3])
redis.call('EXPIRE', KEYS[3], ARGV[4])
return 1`

const tokenQuotaReserveScript = `
if redis.call('EXISTS', KEYS[2]) == 1
  or tonumber(redis.call('HGET', KEYS[1], 'Id') or '0') ~= tonumber(ARGV[2])
  or redis.call('HEXISTS', KEYS[1], 'RemainQuota') == 0
  or redis.call('HEXISTS', KEYS[1], 'UsedQuota') == 0 then
  return -1
end
local remain = tonumber(redis.call('HGET', KEYS[1], 'RemainQuota'))
if remain == nil or remain < tonumber(ARGV[1]) then
  return 0
end
redis.call('HINCRBY', KEYS[1], 'RemainQuota', -tonumber(ARGV[1]))
redis.call('HINCRBY', KEYS[1], 'UsedQuota', tonumber(ARGV[1]))
redis.call('HSET', KEYS[1], 'AccessedTime', ARGV[3])
redis.call('INCR', KEYS[3])
redis.call('EXPIRE', KEYS[3], ARGV[4])
return 1`

func quotaResultFromLua(result int, err error) (cacheQuotaResult, error) {
	if err != nil {
		return cacheQuotaMiss, err
	}
	switch result {
	case 1:
		return cacheQuotaOK, nil
	case 0:
		return cacheQuotaInsufficient, nil
	default:
		return cacheQuotaMiss, nil
	}
}

func cacheTryReserveUserQuota(userID int, amount int64) (cacheQuotaResult, error) {
	result, err := common.RDB.Eval(context.Background(), userQuotaReserveScript,
		[]string{getUserCacheKey(userID), getUserQuotaCacheFenceKey(userID), getUserQuotaCacheDirtyKey(userID)},
		amount, userID, userCacheSchemaVersion, userAuthFenceTTLSeconds()).Int()
	return quotaResultFromLua(result, err)
}

func cacheTryReserveTokenQuota(id int, key string, amount int64) (cacheQuotaResult, error) {
	result, err := common.RDB.Eval(context.Background(), tokenQuotaReserveScript,
		[]string{getTokenCacheKey(key), getTokenCacheFenceKey(key), getTokenCacheQuotaDirtyKey(key)},
		amount, id, common.GetTimestamp(), tokenCacheMutationLeaseSeconds()).Int()
	return quotaResultFromLua(result, err)
}

func reserveUserQuotaDB(id int, quota int) (bool, error) {
	result := DB.Model(&User{}).
		Where("id = ? AND quota >= ?", id, quota).
		Update("quota", gorm.Expr("quota - ?", quota))
	return result.RowsAffected == 1, result.Error
}

func reserveUserQuotaDBAndInvalidate(id int, quota int) (bool, error) {
	reserved, err := reserveUserQuotaDB(id, quota)
	if invalidateErr := invalidateUserQuotaCacheForMutation(id); invalidateErr != nil {
		common.SysError("failed to invalidate user quota cache after database reservation: " + invalidateErr.Error())
	}
	return reserved, err
}

func reserveTokenQuotaDB(id int, key string, quota int, unlimited bool) (bool, error) {
	query := DB.Model(&Token{}).Where(&Token{Id: id, Key: key})
	if unlimited {
		query = query.Where("unlimited_quota = ?", true)
	} else {
		query = query.Where("unlimited_quota = ? AND remain_quota >= ?", false, quota)
	}
	result := query.Updates(map[string]interface{}{
		"remain_quota":  gorm.Expr("remain_quota - ?", quota),
		"used_quota":    gorm.Expr("used_quota + ?", quota),
		"accessed_time": common.GetTimestamp(),
	})
	return result.RowsAffected == 1, result.Error
}

func reserveTokenQuotaDBAndInvalidate(id int, key string, quota int, unlimited bool) (bool, error) {
	reserved, err := reserveTokenQuotaDB(id, key, quota, unlimited)
	if invalidateErr := invalidateTokenCacheForMutation(key); invalidateErr != nil {
		common.SysError("failed to invalidate token quota cache after database reservation: " + invalidateErr.Error())
	}
	return reserved, err
}

// TryReserveUserQuota atomically checks and deducts a user's wallet quota.
// Redis serializes the fast-path reservation, while the database conditionally
// confirms and persists every debit before success is returned.
func TryReserveUserQuota(id int, quota int) (bool, error) {
	if quota < 0 {
		return false, errors.New("quota 不能为负数！")
	}
	if quota == 0 {
		return true, nil
	}
	if !common.RedisEnabled {
		return reserveUserQuotaDB(id, quota)
	}

	result, err := cacheTryReserveUserQuota(id, int64(quota))
	if err == nil && result == cacheQuotaMiss {
		if _, hydrateErr := GetUserCache(id); hydrateErr == nil {
			result, err = cacheTryReserveUserQuota(id, int64(quota))
		}
	}
	if err != nil || result == cacheQuotaMiss {
		if err != nil {
			common.SysLog("user quota cache reserve unavailable, falling back to database: " + err.Error())
		}
		return reserveUserQuotaDBAndInvalidate(id, quota)
	}
	if result == cacheQuotaInsufficient {
		persisted, err := reserveUserQuotaDB(id, quota)
		if err != nil || !persisted {
			return false, err
		}
		if invalidateErr := invalidateUserQuotaCacheForMutation(id); invalidateErr != nil {
			common.SysError("failed to invalidate stale user quota cache: " + invalidateErr.Error())
		}
		return true, nil
	}
	persisted, err := reserveUserQuotaDB(id, quota)
	if err != nil || !persisted {
		if cacheErr := finalizeUserQuotaReservation(id, false); cacheErr != nil {
			common.SysError("failed to invalidate stale user quota cache: " + cacheErr.Error())
		}
		return false, err
	}
	if cacheErr := finalizeUserQuotaReservation(id, true); cacheErr != nil {
		common.SysError("failed to check user quota cache fence after reservation: " + cacheErr.Error())
	}
	return true, nil
}

// TryReserveTokenQuota atomically checks and deducts a token quota. Unlimited
// tokens skip the balance check but still update remain/used accounting. Every
// debit is persisted immediately, including when batch updates are enabled.
func TryReserveTokenQuota(id int, key string, quota int, unlimited bool) (bool, error) {
	if quota < 0 {
		return false, errors.New("quota 不能为负数！")
	}
	if quota == 0 {
		return true, nil
	}
	if unlimited {
		return reserveTokenQuotaDBAndInvalidate(id, key, quota, true)
	}
	if !common.RedisEnabled {
		return reserveTokenQuotaDB(id, key, quota, false)
	}

	result, err := cacheTryReserveTokenQuota(id, key, int64(quota))
	if err == nil && result == cacheQuotaMiss {
		if _, hydrateErr := GetTokenByKey(key, true); hydrateErr == nil {
			result, err = cacheTryReserveTokenQuota(id, key, int64(quota))
		}
	}
	if err != nil || result == cacheQuotaMiss {
		if err != nil {
			common.SysLog("token quota cache reserve unavailable, falling back to database: " + err.Error())
		}
		return reserveTokenQuotaDBAndInvalidate(id, key, quota, false)
	}
	if result == cacheQuotaInsufficient {
		persisted, err := reserveTokenQuotaDBAndInvalidate(id, key, quota, false)
		if err != nil || !persisted {
			return false, err
		}
		return true, nil
	}
	persisted, err := reserveTokenQuotaDB(id, key, quota, false)
	if err != nil || !persisted {
		if cacheErr := finalizeTokenQuotaReservation(key, false); cacheErr != nil {
			common.SysError("failed to invalidate stale token quota cache: " + cacheErr.Error())
		}
		return false, err
	}
	if cacheErr := finalizeTokenQuotaReservation(key, true); cacheErr != nil {
		common.SysError("failed to check token cache fence after reservation: " + cacheErr.Error())
	}
	return true, nil
}
