package relay

import (
	"encoding/json"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
)

// ggapi GG-010: the official OpenAI/Azure Responses API rejects
// frequency_penalty and presence_penalty, so they are removed before a request
// reaches an official channel. Compatible providers keep explicit values.

func isOfficialResponsesChannel(info *relaycommon.RelayInfo) bool {
	return info != nil && info.ChannelMeta != nil &&
		(info.ChannelType == constant.ChannelTypeOpenAI || info.ChannelType == constant.ChannelTypeAzure)
}

func dropUnsupportedOfficialResponsesPenalties(info *relaycommon.RelayInfo, request *dto.OpenAIResponsesRequest) {
	if request == nil || !isOfficialResponsesChannel(info) {
		return
	}
	request.FrequencyPenalty = nil
	request.PresencePenalty = nil
}

func hasOfficialResponsesPenalties(info *relaycommon.RelayInfo, request *dto.OpenAIResponsesRequest) bool {
	return request != nil && isOfficialResponsesChannel(info) &&
		(len(request.FrequencyPenalty) > 0 || len(request.PresencePenalty) > 0)
}

func sanitizeOfficialResponsesPassThroughBody(info *relaycommon.RelayInfo, body []byte) ([]byte, error) {
	if !isOfficialResponsesChannel(info) {
		return body, nil
	}
	var fields map[string]json.RawMessage
	if err := common.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	changed := false
	for _, field := range []string{"frequency_penalty", "presence_penalty"} {
		if _, ok := fields[field]; ok {
			delete(fields, field)
			changed = true
		}
	}
	if !changed {
		return body, nil
	}
	return common.Marshal(fields)
}
