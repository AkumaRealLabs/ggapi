package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedRetryChainLogs(t *testing.T, userId int, tokenId int, requestId string) (err1, err2, consume *Log) {
	t.Helper()
	otherErr := common.MapToJsonStr(map[string]interface{}{
		"error_code":   "upstream_error",
		"channel_id":   101,
		"channel_name": "retry-a",
		"channel_type": 1,
		"admin_info": map[string]interface{}{
			"use_channel":      []string{"101", "102"},
			"channel_id":       101,
			"channel_name":     "retry-a",
			"is_multi_key":     true,
			"multi_key_index":  0,
			"channel_affinity": map[string]interface{}{"hit": true},
		},
	})
	otherConsume := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.01,
		"admin_info": map[string]interface{}{
			"use_channel": []string{"101", "102", "103"},
			"channel_id":  103,
		},
	})

	err1 = &Log{
		UserId:            userId,
		Username:          "alice",
		CreatedAt:         1000,
		Type:              LogTypeError,
		Content:           "attempt 1 failed",
		TokenName:         "tok",
		ModelName:         "gpt-test",
		ChannelId:         101,
		TokenId:           tokenId,
		RequestId:         requestId,
		UpstreamRequestId: "up-1",
		Other:             otherErr,
	}
	err2 = &Log{
		UserId:            userId,
		Username:          "alice",
		CreatedAt:         1001,
		Type:              LogTypeError,
		Content:           "attempt 2 failed",
		TokenName:         "tok",
		ModelName:         "gpt-test",
		ChannelId:         102,
		TokenId:           tokenId,
		RequestId:         requestId,
		UpstreamRequestId: "up-2",
		Other:             otherErr,
	}
	consume = &Log{
		UserId:            userId,
		Username:          "alice",
		CreatedAt:         1002,
		Type:              LogTypeConsume,
		Content:           "ok",
		TokenName:         "tok",
		ModelName:         "gpt-test",
		Quota:             42,
		PromptTokens:      10,
		CompletionTokens:  5,
		ChannelId:         103,
		TokenId:           tokenId,
		RequestId:         requestId,
		UpstreamRequestId: "up-3",
		Other:             otherConsume,
	}
	require.NoError(t, LOG_DB.Create(err1).Error)
	require.NoError(t, LOG_DB.Create(err2).Error)
	require.NoError(t, LOG_DB.Create(consume).Error)
	return err1, err2, consume
}

func assertUserLogSanitized(t *testing.T, log *Log) {
	t.Helper()
	require.NotNil(t, log)
	assert.Equal(t, 0, log.ChannelId, "channel id must be cleared for user views")
	assert.Empty(t, log.ChannelName, "channel name must be cleared for user views")
	assert.Empty(t, log.UpstreamRequestId, "upstream_request_id must be cleared for user views")

	other, err := common.StrToMap(log.Other)
	require.NoError(t, err)
	if other == nil {
		return
	}
	for _, key := range []string{
		"admin_info", "audit_info", "stream_status",
		"channel", "channel_id", "channel_name", "channel_type",
		"use_channel", "channel_affinity", "is_multi_key", "multi_key_index",
		"upstream_request_id",
	} {
		_, present := other[key]
		assert.Falsef(t, present, "sensitive other key %q must be stripped", key)
	}
}

// A: admin query keeps every retry attempt, consume row, channel ids, and admin_info.
func TestGetAllLogsKeepsRetryChainAndAdminInfo(t *testing.T) {
	truncateTables(t)
	require.NoError(t, DB.Create(&Channel{Id: 101, Name: "retry-a"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 102, Name: "retry-b"}).Error)
	require.NoError(t, DB.Create(&Channel{Id: 103, Name: "retry-c"}).Error)

	_, _, consume := seedRetryChainLogs(t, 1, 11, "req-success")

	logs, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "req-success", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, logs, 3)

	// Admin order is created_at desc, id desc — consume first.
	require.Equal(t, LogTypeConsume, logs[0].Type)
	require.Equal(t, consume.Id, logs[0].Id)
	assert.Equal(t, 103, logs[0].ChannelId)
	assert.Equal(t, "retry-c", logs[0].ChannelName)
	assert.Equal(t, "up-3", logs[0].UpstreamRequestId)

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	adminInfo, ok := other["admin_info"].(map[string]interface{})
	require.True(t, ok, "admin_info must remain for admin views")
	assert.Contains(t, adminInfo, "use_channel")

	// Two error attempts still present with channel ids.
	errCount := 0
	for _, log := range logs {
		if log.Type == LogTypeError {
			errCount++
			assert.NotZero(t, log.ChannelId)
			assert.NotEmpty(t, log.ChannelName)
		}
	}
	assert.Equal(t, 2, errCount)
}

// B: user self query collapses success chain to final consume and sanitizes fields.
func TestGetUserLogsProjectsSuccessChainToConsumeOnly(t *testing.T) {
	truncateTables(t)
	seedRetryChainLogs(t, 1, 11, "req-success")

	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "logical request total must be 1")
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeConsume, logs[0].Type)
	assert.Equal(t, 42, logs[0].Quota)
	assert.Equal(t, "req-success", logs[0].RequestId)
	assert.Equal(t, "gpt-test", logs[0].ModelName)
	assert.Equal(t, "tok", logs[0].TokenName)
	assertUserLogSanitized(t, logs[0])

	// Billing-friendly fields remain.
	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	assert.Contains(t, other, "model_price")
}

// Filtering by LogTypeError must not re-surface intermediate retries of a
// request that ultimately consumed successfully (project first, then filter).
func TestGetUserLogsTypeFilterDoesNotRevealHiddenRetries(t *testing.T) {
	truncateTables(t)
	seedRetryChainLogs(t, 1, 11, "req-success")
	// A genuine all-failed request should still appear under type=error.
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 1, CreatedAt: 1100, Type: LogTypeError, Content: "only-fail",
		TokenName: "tok", ModelName: "gpt-test", ChannelId: 9, TokenId: 11,
		RequestId: "req-only-fail",
	}).Error)

	logs, total, err := GetUserLogs(1, LogTypeError, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "req-only-fail", logs[0].RequestId)
	assert.Equal(t, LogTypeError, logs[0].Type)
	assertUserLogSanitized(t, logs[0])
}

// Filtering by an intermediate upstream_request_id must not resurrect hidden
// retry rows when the logical winner is a later consume with a different ID.
func TestGetUserLogsUpstreamFilterDoesNotRevealHiddenRetries(t *testing.T) {
	truncateTables(t)
	seedRetryChainLogs(t, 1, 11, "req-success")

	// Intermediate attempt used up-1; final consume used up-3.
	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "up-1")
	require.NoError(t, err)
	require.Equal(t, int64(0), total)
	require.Empty(t, logs)

	// Filtering by the winner's upstream id still returns the consume row
	// (field is blanked in the response by formatUserLogs).
	logs, total, err = GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "up-3")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeConsume, logs[0].Type)
	assertUserLogSanitized(t, logs[0])
}

// C: all-failed request_id returns only the last failure, sanitized.
func TestGetUserLogsProjectsAllFailedToFinalError(t *testing.T) {
	truncateTables(t)
	rid := "req-fail"
	for i, ch := range []int{201, 202, 203} {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId:            1,
			CreatedAt:         int64(2000 + i),
			Type:              LogTypeError,
			Content:           "fail",
			TokenName:         "tok",
			ModelName:         "gpt-test",
			ChannelId:         ch,
			TokenId:           11,
			RequestId:         rid,
			UpstreamRequestId: "up-fail",
			Other: common.MapToJsonStr(map[string]interface{}{
				"channel_id":   ch,
				"channel_name": "ch",
				"channel_type": 1,
				"admin_info":   map[string]interface{}{"use_channel": []string{"201", "202", "203"}},
			}),
		}).Error)
	}

	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", rid, "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeError, logs[0].Type)
	// Last inserted failure should win (highest id).
	assert.Equal(t, rid, logs[0].RequestId)
	assertUserLogSanitized(t, logs[0])
}

// D: different request_ids stay separate; empty request_id rows never merge.
func TestGetUserLogsDoesNotMergeDistinctOrEmptyRequestIds(t *testing.T) {
	truncateTables(t)
	// Two distinct request ids (each with 2 errors) → 2 logical rows.
	for _, rid := range []string{"req-a", "req-b"} {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId: 1, CreatedAt: 3000, Type: LogTypeError, Content: "e1",
			RequestId: rid, ChannelId: 1, TokenId: 11, ModelName: "m",
		}).Error)
		require.NoError(t, LOG_DB.Create(&Log{
			UserId: 1, CreatedAt: 3001, Type: LogTypeError, Content: "e2",
			RequestId: rid, ChannelId: 2, TokenId: 11, ModelName: "m",
		}).Error)
	}
	// Empty request_id historical rows must not collapse.
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 1, CreatedAt: 3002, Type: LogTypeError, Content: "legacy-1",
		RequestId: "", ChannelId: 9, TokenId: 11, ModelName: "m",
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 1, CreatedAt: 3003, Type: LogTypeError, Content: "legacy-2",
		RequestId: "", ChannelId: 9, TokenId: 11, ModelName: "m",
	}).Error)

	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(4), total)
	require.Len(t, logs, 4)

	var emptyCount, aCount, bCount int
	for _, log := range logs {
		assertUserLogSanitized(t, log)
		switch log.RequestId {
		case "":
			emptyCount++
		case "req-a":
			aCount++
		case "req-b":
			bCount++
		}
	}
	assert.Equal(t, 2, emptyCount)
	assert.Equal(t, 1, aCount)
	assert.Equal(t, 1, bCount)
}

// E: token log endpoint applies the same isolation rules as self logs.
func TestGetLogByTokenIdAppliesLogicalProjectionAndSanitization(t *testing.T) {
	truncateTables(t)
	seedRetryChainLogs(t, 1, 11, "req-token")
	// Noise for another token must not appear.
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 1, CreatedAt: 4000, Type: LogTypeConsume, Content: "other",
		RequestId: "req-other", ChannelId: 1, TokenId: 99, ModelName: "m", Quota: 1,
	}).Error)

	logs, err := GetLogByTokenId(11)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, LogTypeConsume, logs[0].Type)
	assert.Equal(t, "req-token", logs[0].RequestId)
	assert.Equal(t, 11, logs[0].TokenId)
	assertUserLogSanitized(t, logs[0])
}

// F: request_id filter, pagination, and total use logical request counts.
func TestGetUserLogsFilterAndPaginationUseLogicalTotal(t *testing.T) {
	truncateTables(t)
	// 3 logical requests, each with 2 attempts.
	for i, rid := range []string{"p1", "p2", "p3"} {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId: 1, CreatedAt: int64(5000 + i*2), Type: LogTypeError, Content: "e",
			RequestId: rid, ChannelId: 1, TokenId: 11, ModelName: "m",
		}).Error)
		require.NoError(t, LOG_DB.Create(&Log{
			UserId: 1, CreatedAt: int64(5001 + i*2), Type: LogTypeConsume, Content: "ok",
			RequestId: rid, ChannelId: 2, TokenId: 11, ModelName: "m", Quota: 1,
		}).Error)
	}

	// Filter single request_id.
	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "p2", "")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
	assert.Equal(t, "p2", logs[0].RequestId)
	assert.Equal(t, LogTypeConsume, logs[0].Type)

	// Page size 1 over all logical requests → total still 3.
	page1, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 1, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, page1, 1)

	page2, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 1, 1, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, page2, 1)
	assert.NotEqual(t, page1[0].RequestId, page2[0].RequestId)

	page3, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 2, 1, "", "", "")
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, page3, 1)

	seen := map[string]bool{
		page1[0].RequestId: true,
		page2[0].RequestId: true,
		page3[0].RequestId: true,
	}
	assert.Equal(t, 3, len(seen))
}

// G: formatUserLogs strips legacy top-level channel diagnostics and admin_info.
func TestFormatUserLogsStripsLegacyChannelFields(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price":         0.004,
		"channel_id":          9,
		"channel_name":        "secret-channel",
		"channel_type":        42,
		"use_channel":         []string{"9", "10"},
		"channel_affinity":    map[string]interface{}{"hit": true},
		"is_multi_key":        true,
		"multi_key_index":     1,
		"upstream_request_id": "should-not-leak-in-other",
		"admin_info": map[string]interface{}{
			"channel_id":  9,
			"use_channel": []string{"9"},
		},
		"audit_info":    map[string]interface{}{"op": "x"},
		"stream_status": map[string]interface{}{"status": "ok"},
	})
	logs := []*Log{{
		ChannelId:         9,
		ChannelName:       "secret-channel",
		UpstreamRequestId: "up-legacy",
		Other:             other,
	}}

	formatUserLogs(logs, 0)

	assert.Equal(t, 0, logs[0].ChannelId)
	assert.Empty(t, logs[0].ChannelName)
	assert.Empty(t, logs[0].UpstreamRequestId)
	// Display id assignment still works.
	assert.Equal(t, 1, logs[0].Id)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Contains(t, parsed, "model_price")
	for _, key := range userLogSensitiveOtherKeys {
		_, present := parsed[key]
		assert.Falsef(t, present, "key %q must be stripped", key)
	}
}

// Pure-Go projection helper locks the ClickHouse-equivalent selection rules.
func TestPickUserLogicalRequestWinners(t *testing.T) {
	logs := []*Log{
		{Id: 1, RequestId: "r1", Type: LogTypeError, CreatedAt: 1, UseTime: 1},
		{Id: 2, RequestId: "r1", Type: LogTypeError, CreatedAt: 2, UseTime: 2},
		{Id: 3, RequestId: "r1", Type: LogTypeConsume, CreatedAt: 3, UseTime: 3},
		{Id: 4, RequestId: "r2", Type: LogTypeError, CreatedAt: 4, UseTime: 1},
		{Id: 5, RequestId: "r2", Type: LogTypeError, CreatedAt: 5, UseTime: 2},
		{Id: 6, RequestId: "", Type: LogTypeError, CreatedAt: 6},
		{Id: 7, RequestId: "", Type: LogTypeError, CreatedAt: 7},
	}
	winners := pickUserLogicalRequestWinners(logs)
	require.Len(t, winners, 4)

	byReq := map[string][]int{}
	for _, w := range winners {
		byReq[w.RequestId] = append(byReq[w.RequestId], w.Id)
	}
	assert.Equal(t, []int{3}, byReq["r1"])
	assert.Equal(t, []int{5}, byReq["r2"])
	assert.ElementsMatch(t, []int{6, 7}, byReq[""])
}

// ClickHouse-style same-second retries (id=0) must prefer higher use_time.
func TestPickUserLogicalRequestWinnersSameSecondUsesUseTime(t *testing.T) {
	logs := []*Log{
		{Id: 0, RequestId: "r-ch", Type: LogTypeError, CreatedAt: 100, UseTime: 1, ChannelId: 10, Content: "first"},
		{Id: 0, RequestId: "r-ch", Type: LogTypeError, CreatedAt: 100, UseTime: 3, ChannelId: 20, Content: "later"},
		{Id: 0, RequestId: "r-ch", Type: LogTypeError, CreatedAt: 100, UseTime: 2, ChannelId: 30, Content: "mid"},
	}
	winners := pickUserLogicalRequestWinners(logs)
	require.Len(t, winners, 1)
	assert.Equal(t, 3, winners[0].UseTime)
	assert.Equal(t, "later", winners[0].Content)
}

// SumUsedQuota must still sum every consume row (not the projected view).
func TestSumUsedQuotaUnaffectedByUserLogProjection(t *testing.T) {
	truncateTables(t)
	// Two consume rows for the same request_id (unusual but possible for tasks);
	// stat path must count both, while user list projects to one.
	for i := 0; i < 2; i++ {
		require.NoError(t, LOG_DB.Create(&Log{
			UserId: 1, Username: "alice", CreatedAt: int64(6000 + i),
			Type: LogTypeConsume, Content: "ok", ModelName: "m", TokenName: "tok",
			Quota: 10, RequestId: "req-stat", ChannelId: 1,
		}).Error)
	}
	// Intermediate error does not affect quota sum.
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 1, Username: "alice", CreatedAt: 5999,
		Type: LogTypeError, Content: "fail", ModelName: "m", TokenName: "tok",
		Quota: 0, RequestId: "req-stat", ChannelId: 2,
	}).Error)

	stat, err := SumUsedQuota(LogTypeUnknown, 0, 0, "", "alice", "", 0, "")
	require.NoError(t, err)
	assert.Equal(t, 20, stat.Quota)

	logs, total, err := GetUserLogs(1, LogTypeUnknown, 0, 0, "", "", 0, 20, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, logs, 1)
}
