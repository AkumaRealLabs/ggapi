package relay

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDropUnsupportedOfficialResponsesPenalties(t *testing.T) {
	tests := []struct {
		name        string
		channelType int
		drop        bool
	}{
		{name: "OpenAI", channelType: constant.ChannelTypeOpenAI, drop: true},
		{name: "Azure", channelType: constant.ChannelTypeAzure, drop: true},
		{name: "compatible provider", channelType: constant.ChannelTypeAdvancedCustom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &dto.OpenAIResponsesRequest{
				FrequencyPenalty: json.RawMessage(`0`),
				PresencePenalty:  json.RawMessage(`1.5`),
			}
			dropUnsupportedOfficialResponsesPenalties(&relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{ChannelType: tt.channelType},
			}, request)

			if tt.drop {
				assert.Nil(t, request.FrequencyPenalty)
				assert.Nil(t, request.PresencePenalty)
				return
			}
			assert.JSONEq(t, `0`, string(request.FrequencyPenalty))
			assert.JSONEq(t, `1.5`, string(request.PresencePenalty))
		})
	}
}

func TestSanitizeOfficialResponsesPassThroughBody(t *testing.T) {
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI}}
	body, err := sanitizeOfficialResponsesPassThroughBody(info, []byte(`{"model":"gpt-5","frequency_penalty":0,"presence_penalty":1.5,"metadata":{"keep":true}}`))
	require.NoError(t, err)
	var got map[string]json.RawMessage
	require.NoError(t, common.Unmarshal(body, &got))
	assert.NotContains(t, got, "frequency_penalty")
	assert.NotContains(t, got, "presence_penalty")
	assert.JSONEq(t, `{"keep":true}`, string(got["metadata"]))

	compatible := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeAdvancedCustom}}
	original := []byte(`{"frequency_penalty":0}`)
	unchanged, err := sanitizeOfficialResponsesPassThroughBody(compatible, original)
	require.NoError(t, err)
	assert.Equal(t, original, unchanged)
}
