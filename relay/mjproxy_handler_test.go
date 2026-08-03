package relay

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relaydto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupMidjourneyChannelContextReplacesSelectedChannelSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/mj/submit/imagine", nil)
	common.SetContextKey(ctx, constant.ContextKeyChannelId, 7)
	common.SetContextKey(ctx, constant.ContextKeyChannelKey, "stale-key")
	common.SetContextKey(ctx, constant.ContextKeyChannelSetting, relaydto.ChannelSettings{Proxy: "http://stale.invalid"})

	baseURL := "https://midjourney.example.com"
	channel := &model.Channel{
		Id:      42,
		Type:    constant.ChannelTypeMidjourney,
		Name:    "origin-midjourney",
		Key:     "origin-key",
		BaseURL: &baseURL,
	}
	wantSettings := relaydto.ChannelSettings{
		Proxy:                 "http://proxy.example.com:8080",
		HTTPProtocol:          relaydto.HTTPProtocolAuto,
		HTTP2ConnectionShards: 3,
	}
	channel.SetSetting(wantSettings)
	info := &relaycommon.RelayInfo{OriginModelName: "mj_imagine"}

	require.NoError(t, setupMidjourneyChannelContext(ctx, channel, "mj_imagine", info))

	assert.Equal(t, channel.Id, common.GetContextKeyInt(ctx, constant.ContextKeyChannelId))
	assert.Equal(t, channel.Key, common.GetContextKeyString(ctx, constant.ContextKeyChannelKey))
	assert.Equal(t, baseURL, common.GetContextKeyString(ctx, constant.ContextKeyChannelBaseUrl))
	gotSettings, ok := common.GetContextKeyType[relaydto.ChannelSettings](ctx, constant.ContextKeyChannelSetting)
	require.True(t, ok)
	assert.Equal(t, wantSettings, gotSettings)
	require.NotNil(t, info.ChannelMeta)
	assert.Equal(t, channel.Id, info.ChannelId)
	assert.Equal(t, channel.Key, info.ApiKey)
	assert.Equal(t, wantSettings, info.ChannelSetting)
}
