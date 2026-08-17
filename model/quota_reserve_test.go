package model

import (
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createReserveTestUser(t *testing.T, quota int) User {
	t.Helper()
	user := User{
		Username:    "reserve-user-" + common.GetRandomString(6),
		Password:    "unused-password-hash",
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		AuthVersion: 1,
		Quota:       quota,
		AffCode:     "reserve-aff-" + common.GetRandomString(8),
	}
	require.NoError(t, DB.Create(&user).Error)
	return user
}

func createReserveTestToken(t *testing.T, remainQuota int) Token {
	t.Helper()
	token := Token{
		UserId:      1,
		Key:         "reserve-token-" + common.GetRandomString(8),
		Name:        "reserve-test",
		Status:      common.TokenStatusEnabled,
		ExpiredTime: -1,
		RemainQuota: remainQuota,
	}
	require.NoError(t, token.Insert())
	return token
}

func getUserQuotaFromDB(t *testing.T, id int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("quota").First(&user, id).Error)
	return user.Quota
}

func getTokenFromDB(t *testing.T, id int) Token {
	t.Helper()
	var token Token
	require.NoError(t, DB.First(&token, id).Error)
	return token
}

func resetBatchUpdateTestState(t *testing.T) {
	t.Helper()
	oldBatchEnabled := common.BatchUpdateEnabled
	common.BatchUpdateEnabled = false
	for i := 0; i < BatchUpdateTypeCount; i++ {
		batchUpdateLocks[i].Lock()
		batchUpdateStores[i] = make(map[int]int)
		batchUpdateLocks[i].Unlock()
	}
	t.Cleanup(func() {
		common.BatchUpdateEnabled = oldBatchEnabled
		for i := 0; i < BatchUpdateTypeCount; i++ {
			batchUpdateLocks[i].Lock()
			batchUpdateStores[i] = make(map[int]int)
			batchUpdateLocks[i].Unlock()
		}
	})
}

func TestTryReserveQuotaWithoutRedis(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)

	user := createReserveTestUser(t, 100)
	reserved, err := TryReserveUserQuota(user.Id, 60)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 40, getUserQuotaFromDB(t, user.Id))

	reserved, err = TryReserveUserQuota(user.Id, 41)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 40, getUserQuotaFromDB(t, user.Id))

	token := createReserveTestToken(t, 80)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 25, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	reloaded := getTokenFromDB(t, token.Id)
	assert.Equal(t, 55, reloaded.RemainQuota)
	assert.Equal(t, 25, reloaded.UsedQuota)

	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 56, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 55, getTokenFromDB(t, token.Id).RemainQuota)
}

func TestStaleUnlimitedTokenStateCannotBypassReservation(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)

	valid := createReserveTestToken(t, 5)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", valid.Id).Update("unlimited_quota", true).Error)
	reserved, err := TryReserveTokenQuota(valid.Id, valid.Key, 10, true)
	require.NoError(t, err)
	assert.True(t, reserved)
	stored := getTokenFromDB(t, valid.Id)
	assert.Equal(t, -5, stored.RemainQuota)
	assert.Equal(t, 10, stored.UsedQuota)

	finite := createReserveTestToken(t, 5)
	reserved, err = TryReserveTokenQuota(finite.Id, finite.Key, 10, true)
	require.NoError(t, err)
	assert.False(t, reserved)
	stored = getTokenFromDB(t, finite.Id)
	assert.Equal(t, 5, stored.RemainQuota)
	assert.Zero(t, stored.UsedQuota)

	deleted := createReserveTestToken(t, 5)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", deleted.Id).Update("unlimited_quota", true).Error)
	require.NoError(t, DB.Delete(&deleted).Error)
	reserved, err = TryReserveTokenQuota(deleted.Id, deleted.Key, 10, true)
	require.NoError(t, err)
	assert.False(t, reserved)
	stored = Token{}
	require.NoError(t, DB.Unscoped().First(&stored, deleted.Id).Error)
	assert.Equal(t, 5, stored.RemainQuota)
	assert.Zero(t, stored.UsedQuota)
}

func TestSelectUpdateDoesNotRestoreStaleQuotaSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)

	token := createReserveTestToken(t, 100)
	stale := getTokenFromDB(t, token.Id)
	reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved)

	stale.Status = common.TokenStatusDisabled
	require.NoError(t, stale.SelectUpdate())

	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, common.TokenStatusDisabled, stored.Status)
	assert.Equal(t, 90, stored.RemainQuota)
	assert.Equal(t, 10, stored.UsedQuota)
}

func TestMetadataUpdateDoesNotRestoreStaleQuotaSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)

	token := createReserveTestToken(t, 100)
	stale := getTokenFromDB(t, token.Id)
	reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved)

	stale.Name = "renamed-after-reserve"
	require.NoError(t, stale.Update())

	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, "renamed-after-reserve", stored.Name)
	assert.Equal(t, 90, stored.RemainQuota)
	assert.Equal(t, 10, stored.UsedQuota)
}

func TestStatusOnlyMembershipUpdateDoesNotRestoreStaleQuotaSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	user := createReserveTestUser(t, 100)
	token := Token{
		UserId:      user.Id,
		Key:         "status-only-token-" + common.GetRandomString(8),
		Name:        "status-only-test",
		Status:      common.TokenStatusEnabled,
		ExpiredTime: -1,
		RemainQuota: 100,
	}
	require.NoError(t, token.Insert())

	stale := getTokenFromDB(t, token.Id)
	reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved)

	stale.Status = common.TokenStatusDisabled
	stale.RemainQuota = 100
	stale.UsedQuota = 0
	require.NoError(t, stale.UpdateStatusWithMembershipGuard())

	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, common.TokenStatusDisabled, stored.Status)
	assert.Equal(t, 90, stored.RemainQuota)
	assert.Equal(t, 10, stored.UsedQuota)
}

func TestQuotaReservationsPersistImmediatelyWithBatchingEnabled(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)
	common.BatchUpdateEnabled = true

	user := createReserveTestUser(t, 10)
	reserved, err := TryReserveUserQuota(user.Id, 8)
	require.NoError(t, err)
	assert.True(t, reserved)
	// GG-008：钱包增量不进批量 map，缓存预扣成功后立即落库。
	assert.Equal(t, 2, getUserQuotaFromDB(t, user.Id), "wallet delta persists immediately even in batch mode")

	reserved, err = TryReserveUserQuota(user.Id, 3)
	require.NoError(t, err)
	assert.False(t, reserved, "cache-authoritative balance must not authorize a second spend")
	cachedUser, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 2, cachedUser.Quota)

	token := createReserveTestToken(t, 9)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 7, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 2, getTokenFromDB(t, token.Id).RemainQuota, "token reserve persists before returning")
	cachedToken, cacheErr := cacheGetTokenByKey(token.Key)
	require.NoError(t, cacheErr)
	assert.Equal(t, 2, cachedToken.RemainQuota)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 3, false)
	require.NoError(t, err)
	assert.False(t, reserved, "cache-insufficient balance must not authorize a second spend")
	assert.Equal(t, 2, getTokenFromDB(t, token.Id).RemainQuota)

	batchUpdate()
	assert.Equal(t, 2, getUserQuotaFromDB(t, user.Id))
	reloadedToken := getTokenFromDB(t, token.Id)
	assert.Equal(t, 2, reloadedToken.RemainQuota)
	assert.Equal(t, 7, reloadedToken.UsedQuota)
}

func TestConfirmedTokenCacheReservationStaysHot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	_, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	for range 2 {
		reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
		require.NoError(t, err)
		assert.True(t, reserved)
	}

	cached, err := cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 80, cached.RemainQuota)
	assert.Equal(t, 20, cached.UsedQuota)
	assert.False(t, server.Exists(getTokenCacheFenceKey(token.Key)))
}

func TestUserReservationBlocksPreCommitCacheRehydration(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 100)
	require.NoError(t, populateUserCache(user))

	result, err := cacheTryReserveUserQuota(user.Id, 10)
	require.NoError(t, err)
	require.Equal(t, cacheQuotaOK, result)
	require.NoError(t, common.RedisDelKey(getUserCacheKey(user.Id)))

	stale, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 100, stale.Quota, "the concurrent read must observe the pre-commit database row")
	assert.False(t, server.Exists(getUserCacheKey(user.Id)), "the pre-commit row must not repopulate Redis")

	reserved, err := reserveUserQuotaDB(user.Id, 10)
	require.NoError(t, err)
	require.True(t, reserved)
	require.NoError(t, finalizeUserQuotaReservation(user.Id, true))
	assert.False(t, server.Exists(getUserQuotaCacheDirtyKey(user.Id)))

	cached, err := GetUserCache(user.Id)
	require.NoError(t, err)
	assert.Equal(t, 90, cached.Quota)
}

func TestConfirmedTokenReservationDefersToMetadataFence(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	_, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)

	const callbackName = "test:begin-token-metadata-fence-after-cache-reserve"
	var marker string
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table != "tokens" || marker != "" {
			return
		}
		var fenceErr error
		marker, fenceErr = beginTokenCacheMutation(token.Key)
		if fenceErr != nil {
			tx.AddError(fenceErr)
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Update().Remove(callbackName) })

	reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	require.NotEmpty(t, marker)
	assert.Error(t, finishTokenCacheMutation(token.Key, marker))
	assert.False(t, server.Exists(getTokenCacheKey(token.Key)))
}

func TestStaleUserQuotaCacheCannotOverdrawDatabase(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 100)
	require.NoError(t, populateUserCache(user))
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 10).Error)

	reserved, err := TryReserveUserQuota(user.Id, 50)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 10, getUserQuotaFromDB(t, user.Id))
	_, err = cacheGetUserBase(user.Id)
	assert.Error(t, err, "the stale quota cache must be invalidated after the database CAS rejects it")
}

func TestStaleInsufficientCacheFallsBackToDatabaseCredit(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 5)
	require.NoError(t, populateUserCache(user))
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Update("quota", 20).Error)

	reserved, err := TryReserveUserQuota(user.Id, 10)
	require.NoError(t, err)
	assert.True(t, reserved, "a committed credit must be spendable despite stale cache")
	assert.Equal(t, 10, getUserQuotaFromDB(t, user.Id))

	token := createReserveTestToken(t, 0)
	_, err = GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("remain_quota", 20).Error)

	validated, err := ValidateUserToken(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 20, validated.RemainQuota, "authentication must confirm an exhausted cache against the database")
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved, "a committed token refund must be spendable despite stale cache")
	assert.Equal(t, 10, getTokenFromDB(t, token.Id).RemainQuota)

	disabled := createReserveTestToken(t, 0)
	_, err = GetTokenByKey(disabled.Key, true)
	require.NoError(t, err)
	require.NoError(t, DB.Model(&Token{}).Where("id = ?", disabled.Id).Updates(map[string]interface{}{
		"remain_quota": 20,
		"status":       common.TokenStatusDisabled,
	}).Error)
	_, err = ValidateUserToken(disabled.Key)
	assert.ErrorIs(t, err, ErrTokenInvalid, "database confirmation must not bypass a disabled token")
}

func TestReserveFallsBackToDatabaseWhenRedisIsUnavailable(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 20)
	require.NoError(t, populateUserCache(user))
	server.Close()

	// Redis 故障时降级为数据库条件更新：服务保持可用且不会超扣。
	reserved, err := TryReserveUserQuota(user.Id, 5)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 15, getUserQuotaFromDB(t, user.Id))

	reserved, err = TryReserveUserQuota(user.Id, 16)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Equal(t, 15, getUserQuotaFromDB(t, user.Id))
}

func TestDatabaseUserQuotaFallbackInvalidatesCachedQuota(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 20)
	require.NoError(t, populateUserCache(user))
	require.NoError(t, common.RDB.Set(t.Context(), getUserCacheKey(user.Id), "stale", 0).Err())

	reserved, err := TryReserveUserQuota(user.Id, 5)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 15, getUserQuotaFromDB(t, user.Id))
	_, err = cacheGetUserBase(user.Id)
	assert.Error(t, err)
}

func TestReserveInvalidatesCacheWhenPersistenceRejects(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 10)
	require.NoError(t, populateUserCache(user))
	require.NoError(t, DB.Delete(&user).Error)

	reserved, err := TryReserveUserQuota(user.Id, 6)
	assert.False(t, reserved)
	require.NoError(t, err)
	_, cacheErr := cacheGetUserBase(user.Id)
	assert.Error(t, cacheErr)

	token := createReserveTestToken(t, 12)
	_, err = GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	require.NoError(t, DB.Delete(&token).Error)
	reserved, err = TryReserveTokenQuota(token.Id, token.Key, 7, false)
	assert.False(t, reserved)
	require.NoError(t, err)
	_, cacheErr = cacheGetTokenByKey(token.Key)
	assert.Error(t, cacheErr)
}

func TestTokenCacheInitPreservesLiveQuotaAndFenceBlocksStaleSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	loaded, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	stale := *loaded

	result, err := cacheTryReserveTokenQuota(token.Id, token.Key, 70)
	require.NoError(t, err)
	require.Equal(t, cacheQuotaOK, result)
	require.NoError(t, finalizeTokenQuotaReservation(token.Key, true))

	// 已存在的哈希只刷新 TTL：数据库快照不得覆盖已被原子预扣的余额。
	code, err := cacheInitToken(stale)
	require.NoError(t, err)
	assert.Equal(t, 2, code)
	cached, err := cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 30, cached.RemainQuota)

	// 变更期间：fence 删除缓存并拦截并发读者手中的过期快照。
	require.NoError(t, invalidateTokenCacheForMutation(token.Key))
	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code, "the pre-mutation snapshot must not be published while fenced")
	_, err = cacheGetTokenByKey(token.Key)
	assert.Error(t, err)

	// 连续变更会续期短 fence；最后一次变更的保护窗口过后才可重新水合。
	server.FastForward(time.Duration(tokenCacheFenceSeconds-1) * time.Second)
	require.NoError(t, invalidateTokenCacheForMutation(token.Key))
	server.FastForward(2 * time.Second)
	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code)
	server.FastForward(time.Duration(tokenCacheFenceSeconds-1) * time.Second)
	fresh, err := GetTokenByKey(token.Key, false)
	require.NoError(t, err)
	assert.Equal(t, 100, fresh.RemainQuota)
	cached, err = cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, 100, cached.RemainQuota)
}

func TestIncompleteTokenCacheIsRehydratedFromDatabase(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	cacheKey := getTokenCacheKey(token.Key)
	require.NoError(t, common.RDB.HSet(t.Context(), cacheKey, "RemainQuota", 100).Err())

	loaded, err := GetTokenByKey(token.Key, false)
	require.NoError(t, err)
	assert.Equal(t, token.Id, loaded.Id)
	cached, err := cacheGetTokenByKey(token.Key)
	require.NoError(t, err)
	assert.Equal(t, token.Id, cached.Id)
	assert.Equal(t, token.RemainQuota, cached.RemainQuota)
}

func TestDatabaseQuotaFallbackCannotPublishPreReservationSnapshot(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	_, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	marker, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err)

	reserved, err := TryReserveTokenQuota(token.Id, token.Key, 10, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Equal(t, 90, getTokenFromDB(t, token.Id).RemainQuota)
	assert.False(t, server.Exists(getTokenCacheKey(token.Key)))
	assert.Error(t, finishTokenCacheMutation(token.Key, marker))
	assert.False(t, server.Exists(getTokenCacheKey(token.Key)))
}

func TestTokenMetadataFenceOutlivesShortInvalidationWindow(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	require.NoError(t, cacheSetTokenForTest(token))
	stale := token

	marker, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err)
	_, err = beginTokenCacheMutation(token.Key)
	require.Error(t, err, "a concurrent metadata mutation must not steal the fence")
	leaseTTL, err := common.RDB.TTL(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Greater(t, leaseTTL, time.Duration(tokenCacheTTLSeconds())*time.Second)
	server.FastForward(time.Duration(tokenCacheFenceSeconds+1) * time.Second)
	code, err := cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code, "the active owner lease must still block a stale snapshot")
	server.FastForward(time.Duration(tokenCacheMutationLeaseSeconds()+1) * time.Second)
	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Equal(t, 1, code, "an expired owner lease permits the stale writer to race the commit")

	require.NoError(t, DB.Model(&Token{}).Where("id = ?", token.Id).Update("status", common.TokenStatusDisabled).Error)
	assert.Error(t, finishTokenCacheMutation(token.Key, marker), "the expired owner must not publish its stale snapshot")
	require.NoError(t, invalidateTokenCacheForMutation(token.Key))
	assert.False(t, server.Exists(getTokenCacheKey(token.Key)))
	_, err = cacheGetTokenByKey(token.Key)
	assert.Error(t, err, "a cache fence must reject a hash written by an expired owner")
	server.FastForward(time.Duration(tokenCacheFenceSeconds+1) * time.Second)
	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code, "a recovery fence must not be shortened to the quota invalidation window")
	fresh := getTokenFromDB(t, token.Id)
	assert.Equal(t, common.TokenStatusDisabled, fresh.Status)

	require.NoError(t, token.Delete())
	tombstoneTTL := tokenCacheTombstoneSeconds()
	ttl, err := common.RDB.TTL(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(tokenCacheMutationLeaseSeconds())*time.Second)
	assert.LessOrEqual(t, ttl, time.Duration(tombstoneTTL)*time.Second)
	tombstone, err := common.RDB.Get(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Equal(t, "deleted", tombstone)
	_, err = beginTokenCacheMutation(token.Key)
	assert.Error(t, err, "a live deleted-token tombstone must reject later mutations")
	_, err = GetTokenByKey(token.Key, false)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	server.FastForward(time.Duration(tombstoneTTL+1) * time.Second)
	assert.False(t, server.Exists(getTokenCacheFenceKey(token.Key)), "the tombstone must eventually be reclaimed")

	batchToken := createReserveTestToken(t, 100)
	batchStale := batchToken
	require.NoError(t, cacheSetTokenForTest(batchToken))
	count, err := BatchDeleteTokens([]int{batchToken.Id}, batchToken.UserId)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	code, err = cacheInitToken(batchStale)
	require.NoError(t, err)
	assert.Zero(t, code, "batch deletion must keep the same bounded tombstone guarantee")
	_, err = cacheGetTokenByKey(batchToken.Key)
	assert.Error(t, err)
}

func TestTokenCacheMutationLeaseRenewalExtendsOwnerFence(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	marker, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err)
	server.FastForward(time.Duration(tokenCacheMutationLeaseSeconds()-1) * time.Second)
	require.NoError(t, renewTokenCacheMutation(token.Key, marker))
	ttl, err := common.RDB.TTL(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(tokenCacheMutationLeaseSeconds()-1)*time.Second)
	assert.Error(t, renewTokenCacheMutation(token.Key, "not-the-owner"))
	require.NoError(t, finishTokenCacheMutation(token.Key, marker))
}

func TestExpiredDeleteLeaseInstallsBoundedTombstone(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	require.NoError(t, cacheSetTokenForTest(token))
	stale := token
	marker, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err)
	require.NoError(t, DB.Delete(&token).Error)
	server.FastForward(time.Duration(tokenCacheMutationLeaseSeconds()+1) * time.Second)
	code, err := cacheInitToken(stale)
	require.NoError(t, err)
	assert.Equal(t, 1, code, "an expired lease allows a pre-delete snapshot to refill the hash")

	assert.Error(t, finishTokenCacheMutation(token.Key, marker))
	require.NoError(t, invalidateTokenCacheForMutation(token.Key))
	assert.False(t, server.Exists(getTokenCacheKey(token.Key)))
	tombstone, err := common.RDB.Get(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Equal(t, "deleted", tombstone)
	ttl, err := common.RDB.TTL(t.Context(), getTokenCacheFenceKey(token.Key)).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(tokenCacheMutationLeaseSeconds())*time.Second)
	assert.LessOrEqual(t, ttl, time.Duration(tokenCacheTombstoneSeconds())*time.Second)

	code, err = cacheInitToken(stale)
	require.NoError(t, err)
	assert.Zero(t, code)
	_, err = cacheGetTokenByKey(token.Key)
	assert.Error(t, err)

	server.FastForward(time.Duration(tokenCacheTombstoneSeconds()+1) * time.Second)
	assert.False(t, server.Exists(getTokenCacheFenceKey(token.Key)), "the tombstone must eventually be reclaimed")
}

func TestAbandonedTokenMetadataFenceLeaseExpires(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	_, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err)
	server.FastForward(time.Duration(tokenCacheMutationLeaseSeconds()+1) * time.Second)
	assert.False(t, server.Exists(getTokenCacheFenceKey(token.Key)))

	marker, err := beginTokenCacheMutation(token.Key)
	require.NoError(t, err, "a crashed mutation must not lock the token forever")
	require.NoError(t, finishTokenCacheMutation(token.Key, marker))
	assert.False(t, server.Exists(getTokenCacheFenceKey(token.Key)))
}

func TestValidateUserTokenWrapsExhaustedRefreshDatabaseError(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 0)
	_, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)

	forcedErr := errors.New("forced token query failure")
	const callbackName = "test:validate-token-exhausted-refresh-error"
	require.NoError(t, DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "tokens" {
			tx.AddError(forcedErr)
		}
	}))
	t.Cleanup(func() {
		assert.NoError(t, DB.Callback().Query().Remove(callbackName))
	})

	_, err = ValidateUserToken(token.Key)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDatabase)
	assert.False(t, errors.Is(err, ErrTokenInvalid))
}

func TestMutationsStopWhenCacheInvalidationFails(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	user := createReserveTestUser(t, 100)
	require.NoError(t, populateUserCache(user))
	token := createReserveTestToken(t, 100)
	_, err := GetTokenByKey(token.Key, true)
	require.NoError(t, err)
	batchToken := createReserveTestToken(t, 100)
	server.Close()

	assert.Error(t, OverrideUserQuota(user.Id, 10))
	assert.Equal(t, 100, getUserQuotaFromDB(t, user.Id))

	token.Status = common.TokenStatusDisabled
	assert.Error(t, token.UpdateWithMembershipGuard())
	stored := getTokenFromDB(t, token.Id)
	assert.Equal(t, common.TokenStatusEnabled, stored.Status)

	assert.Error(t, token.Delete())
	stored = getTokenFromDB(t, token.Id)
	assert.Equal(t, token.Id, stored.Id)

	_, err = BatchDeleteTokens([]int{batchToken.Id}, batchToken.UserId)
	assert.Error(t, err)
	stored = getTokenFromDB(t, batchToken.Id)
	assert.Equal(t, batchToken.Id, stored.Id)
}

func TestCommittedTokenDeleteReturnsSuccessWhenCachePublishFails(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	token := createReserveTestToken(t, 100)
	require.NoError(t, cacheSetTokenForTest(token))
	const callbackName = "test:close-redis-after-token-delete"
	require.NoError(t, DB.Callback().Delete().After("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "tokens" {
			server.Close()
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Delete().Remove(callbackName) })

	require.NoError(t, token.Delete())
	assert.ErrorIs(t, DB.First(&Token{}, token.Id).Error, gorm.ErrRecordNotFound)
}

func TestCommittedTokenBatchDeleteReturnsCountWhenCachePublishFails(t *testing.T) {
	truncateTables(t)
	resetBatchUpdateTestState(t)
	server := useUserCacheMiniRedis(t)

	first := createReserveTestToken(t, 100)
	second := createReserveTestToken(t, 100)
	const callbackName = "test:close-redis-after-token-batch-delete"
	require.NoError(t, DB.Callback().Delete().After("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "tokens" {
			server.Close()
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Delete().Remove(callbackName) })

	count, err := BatchDeleteTokens([]int{first.Id, second.Id}, first.UserId)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	var remaining int64
	require.NoError(t, DB.Model(&Token{}).Where("id IN ?", []int{first.Id, second.Id}).Count(&remaining).Error)
	assert.Zero(t, remaining)
}
