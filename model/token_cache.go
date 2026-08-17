package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

func getTokenCacheKey(key string) string {
	return fmt.Sprintf("token:%s", common.GenerateHMAC(key))
}

func getTokenCacheFenceKey(key string) string {
	return fmt.Sprintf("token:fence:%s", common.GenerateHMAC(key))
}

func getTokenCacheQuotaDirtyKey(key string) string {
	return fmt.Sprintf("token:quota-dirty:%s", common.GenerateHMAC(key))
}

func tokenCacheTTLSeconds() int {
	ttl := common.RedisKeyCacheSeconds()
	if ttl <= 0 {
		return 60
	}
	return ttl
}

// tokenCacheFenceSeconds protects one-shot cache invalidations after quota or
// user-state changes.
const tokenCacheFenceSeconds = 10

// A metadata-mutation lease outlives every token-cache entry that could have
// been read before the fence, but eventually releases after a crashed writer.
func tokenCacheMutationLeaseSeconds() int {
	cacheTTL := tokenCacheTTLSeconds()
	extra := cacheTTL
	if extra < 60 {
		extra = 60
	}
	return cacheTTL + extra
}

// A deleted-token tombstone only needs to outlive stale cache readers; it is
// bounded so deleting distinct tokens cannot grow Redis forever.
func tokenCacheTombstoneSeconds() int {
	return tokenCacheMutationLeaseSeconds() + tokenCacheTTLSeconds()
}

// invalidateTokenCacheForMutation raises a short fence and drops the cached
// hash. It never replaces an owned metadata-mutation fence or shortens a
// longer recovery fence/tombstone already protecting the key.
func invalidateTokenCacheForMutation(key string) error {
	if !common.RedisEnabled || key == "" {
		return nil
	}
	const script = `
local current = redis.call('GET', KEYS[1])
local window = tonumber(ARGV[1])
if not current then
  redis.call('SET', KEYS[1], 1, 'EX', window)
elseif current == '1' then
  local ttl = redis.call('TTL', KEYS[1])
  if ttl >= 0 and ttl < window then
    redis.call('EXPIRE', KEYS[1], window)
	elseif ttl < 0 then
		redis.call('SET', KEYS[1], 1, 'EX', window)
	end
end
if current and current ~= '1' then
  if redis.call('EXISTS', KEYS[3]) == 0 then
    redis.call('SET', KEYS[3], 1, 'EX', ARGV[2])
  else
    redis.call('EXPIRE', KEYS[3], ARGV[2])
  end
end
redis.call('DEL', KEYS[2])
return 1`
	return common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheFenceKey(key), getTokenCacheKey(key), getTokenCacheQuotaDirtyKey(key),
	}, tokenCacheFenceSeconds, tokenCacheMutationLeaseSeconds()).Err()
}

// finalizeTokenQuotaReservation accounts for one Redis reservation after its
// database CAS finishes. A dirty marker keeps metadata writers from publishing
// a snapshot read before the reservation commit.
func finalizeTokenQuotaReservation(key string, committed bool) error {
	if !common.RedisEnabled || key == "" {
		return nil
	}
	const script = `
local pending = tonumber(redis.call('GET', KEYS[3]) or '0')
if pending > 1 then
  pending = redis.call('DECR', KEYS[3])
else
  pending = 0
  redis.call('DEL', KEYS[3])
end
local current = redis.call('GET', KEYS[1])
if ARGV[1] == '1' then
  if current and current ~= '1' then
    redis.call('DEL', KEYS[2])
    if pending < 1 then pending = 1 end
    redis.call('SET', KEYS[3], pending, 'EX', ARGV[2])
  elseif current then
    redis.call('DEL', KEYS[2])
  end
  return 1
end
redis.call('DEL', KEYS[2])
if current and current ~= '1' and pending > 0 then
  redis.call('SET', KEYS[3], pending, 'EX', ARGV[2])
elseif not current then
  redis.call('SET', KEYS[1], '1', 'EX', ARGV[3])
	end
return 1`
	committedValue := 0
	if committed {
		committedValue = 1
	}
	return common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheFenceKey(key), getTokenCacheKey(key), getTokenCacheQuotaDirtyKey(key),
	}, committedValue, tokenCacheMutationLeaseSeconds(), tokenCacheFenceSeconds).Err()
}

// beginTokenCacheMutation installs a caller-owned lease. The database mutation
// must not proceed if this fails.
func beginTokenCacheMutation(key string) (string, error) {
	if !common.RedisEnabled {
		return "", nil
	}
	if key == "" {
		return "", fmt.Errorf("token key is empty")
	}
	marker := common.GetUUID()
	const script = `
local current = redis.call('GET', KEYS[1])
if current and current ~= '1' then
  return 0
end
redis.call('SET', KEYS[1], ARGV[1], 'EX', ARGV[2])
redis.call('DEL', KEYS[2])
return 1`
	result, err := common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheFenceKey(key), getTokenCacheKey(key), getTokenCacheQuotaDirtyKey(key),
	}, marker, tokenCacheMutationLeaseSeconds()).Int()
	if err != nil {
		return "", err
	}
	if result != 1 {
		return "", fmt.Errorf("token cache mutation is already pending")
	}
	return marker, nil
}

func tokenCacheMutationRenewInterval() time.Duration {
	interval := tokenCacheMutationLeaseSeconds() / 3
	if interval < 1 {
		interval = 1
	}
	return time.Duration(interval) * time.Second
}

func renewTokenCacheMutation(key string, marker string) error {
	if !common.RedisEnabled || marker == "" {
		return nil
	}
	const script = `
local current = redis.call('GET', KEYS[1])
if current ~= ARGV[1] then
  return 0
end
redis.call('EXPIRE', KEYS[1], ARGV[2])
return 1`
	result, err := common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheFenceKey(key),
	}, marker, tokenCacheMutationLeaseSeconds()).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("token cache mutation fence ownership was lost")
	}
	return nil
}

// watchTokenCacheMutation renews the caller-owned lease until the returned
// stop function is invoked. A crashed watcher still lets the lease expire.
func watchTokenCacheMutation(key string, marker string) func() {
	if !common.RedisEnabled || marker == "" {
		return func() {}
	}
	done := make(chan struct{})
	stopped := make(chan struct{})
	var stopOnce sync.Once
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(tokenCacheMutationRenewInterval())
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := renewTokenCacheMutation(key, marker); err != nil {
					return
				}
			}
		}
	}()
	return func() {
		stopOnce.Do(func() { close(done) })
		<-stopped
	}
}

const tokenCacheWriteFields = `
redis.call('HSET', KEYS[1],
  'Id', ARGV[1], 'UserId', ARGV[2], 'Status', ARGV[3], 'Name', ARGV[4],
  'CreatedTime', ARGV[5], 'AccessedTime', ARGV[6], 'ExpiredTime', ARGV[7],
  'UnlimitedQuota', ARGV[8], 'ModelLimitsEnabled', ARGV[9], 'ModelLimits', ARGV[10],
  'AllowIps', ARGV[11], 'Group', ARGV[12], 'CrossGroupRetry', ARGV[13],
  'AutoGroups', ARGV[14], 'RemainQuota', ARGV[15], 'UsedQuota', ARGV[16])`

func tokenCacheArgs(token Token) []interface{} {
	allowIps := ""
	if token.AllowIps != nil {
		allowIps = *token.AllowIps
	}
	return []interface{}{
		token.Id, token.UserId, token.Status, token.Name,
		token.CreatedTime, token.AccessedTime, token.ExpiredTime,
		strconv.FormatBool(token.UnlimitedQuota), strconv.FormatBool(token.ModelLimitsEnabled),
		token.ModelLimits, allowIps, token.Group, strconv.FormatBool(token.CrossGroupRetry),
		token.AutoGroups, token.RemainQuota, token.UsedQuota,
	}
}

// finishTokenCacheMutation publishes the current database row and releases
// only this caller's fence. A deleted token keeps a bounded tombstone so an
// in-flight pre-delete snapshot cannot repopulate Redis during the cache
// lifetime.
func finishTokenCacheMutation(key string, marker string) error {
	if !common.RedisEnabled || marker == "" {
		return nil
	}
	var token Token
	if err := DB.Where(commonKeyCol+" = ?", key).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			const script = `
local current = redis.call('GET', KEYS[1])
if current ~= ARGV[1] then
	redis.call('DEL', KEYS[2])
	if not current or current == '1' or current == 'deleted' then
		redis.call('SET', KEYS[1], 'deleted', 'EX', ARGV[2])
	end
	return 0
end
redis.call('SET', KEYS[1], 'deleted', 'EX', ARGV[2])
redis.call('DEL', KEYS[2], KEYS[3])
return 1`
			result, redisErr := common.RDB.Eval(context.Background(), script, []string{
				getTokenCacheFenceKey(key), getTokenCacheKey(key), getTokenCacheQuotaDirtyKey(key),
			}, marker, tokenCacheTombstoneSeconds()).Int()
			if redisErr != nil {
				return redisErr
			}
			if result != 1 {
				return fmt.Errorf("token cache mutation fence ownership was lost")
			}
			return nil
		}
		if invalidateErr := invalidateTokenCacheForMutation(key); invalidateErr != nil {
			return fmt.Errorf("%w; failed to clear token cache: %v", err, invalidateErr)
		}
		return err
	}
	args := append(tokenCacheArgs(token), tokenCacheTTLSeconds(), marker, tokenCacheTombstoneSeconds(), tokenCacheMutationLeaseSeconds())
	const script = `
local current = redis.call('GET', KEYS[2])
if current ~= ARGV[18] then
	redis.call('DEL', KEYS[1])
	if not current or current == '1' then
		redis.call('SET', KEYS[2], '1', 'EX', ARGV[19])
	end
	return 0
end
if redis.call('EXISTS', KEYS[3]) == 1 then
  redis.call('DEL', KEYS[1], KEYS[3])
  redis.call('SET', KEYS[2], '1', 'EX', ARGV[20])
  return 0
end
` + tokenCacheWriteFields + `
redis.call('EXPIRE', KEYS[1], ARGV[17])
redis.call('DEL', KEYS[2])
return 1`
	result, err := common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheKey(key), getTokenCacheFenceKey(key), getTokenCacheQuotaDirtyKey(key),
	}, args...).Int()
	if err != nil {
		return err
	}
	if result != 1 {
		return fmt.Errorf("token cache mutation fence ownership was lost")
	}
	return nil
}

// cacheInitToken publishes a database snapshot only when no mutation fence is
// active and the hash is cold. An existing hash only gets its TTL refreshed:
// its RemainQuota may already be ahead of this snapshot because atomic
// pre-consume decrements Redis first, so a snapshot must never overwrite any
// field of a live hash.
// 返回值：0=被 fence 拦截，1=完成初始化，2=哈希已存在，仅刷新 TTL。
func cacheInitToken(token Token) (int, error) {
	if !common.RedisEnabled {
		return 0, nil
	}
	const script = `
if redis.call('EXISTS', KEYS[2]) == 1 then
  return 0
end
if redis.call('EXISTS', KEYS[1]) == 1 then
  if redis.call('HEXISTS', KEYS[1], 'Id') == 0 then
    if redis.call('EXISTS', KEYS[3]) == 1 then
      return 0
    end
` + tokenCacheWriteFields + `
    redis.call('EXPIRE', KEYS[1], ARGV[17])
    return 1
  end
  redis.call('EXPIRE', KEYS[1], ARGV[17])
  return 2
end
if redis.call('EXISTS', KEYS[3]) == 1 then
  return 0
end
` + tokenCacheWriteFields + `
redis.call('EXPIRE', KEYS[1], ARGV[17])
return 1`

	args := append(tokenCacheArgs(token), tokenCacheTTLSeconds())
	return common.RDB.Eval(context.Background(), script, []string{
		getTokenCacheKey(token.Key), getTokenCacheFenceKey(token.Key), getTokenCacheQuotaDirtyKey(token.Key),
	}, args...).Int()
}

// cacheGetTokenByKey 从缓存读取 token；不完整的哈希（如仅有配额字段）会被拒绝。
func cacheGetTokenByKey(key string) (*Token, error) {
	if !common.RedisEnabled {
		return nil, fmt.Errorf("redis is not enabled")
	}
	fenced, err := common.RDB.Exists(context.Background(), getTokenCacheFenceKey(key)).Result()
	if err != nil {
		return nil, err
	}
	if fenced > 0 {
		return nil, fmt.Errorf("token cache is fenced")
	}
	var token Token
	if err := common.RedisHGetObj(getTokenCacheKey(key), &token); err != nil {
		return nil, err
	}
	if token.Id <= 0 {
		return nil, fmt.Errorf("token cache is incomplete")
	}
	token.Key = key
	return &token, nil
}
