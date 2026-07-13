package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGeminiChatBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewGeminiChatBillingUsage(nil))
	require.Nil(t, NewGeminiChatBillingUsage(&GeminiUsageMetadata{}))

	billingUsage := NewGeminiChatBillingUsage(&GeminiUsageMetadata{PromptTokenCount: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.GeminiUsageMetadata)
	assert.Equal(t, BillingUsageSourceGeminiChat, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticGemini, billingUsage.Semantic)
	assert.False(t, billingUsage.Estimated)
}

func TestNewClaudeMessagesBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewClaudeMessagesBillingUsage(nil))
	require.Nil(t, NewClaudeMessagesBillingUsage(&ClaudeUsage{}))
	require.Nil(t, NewClaudeMessagesBillingUsage(&ClaudeUsage{CacheCreation: &ClaudeCacheCreationUsage{}}))

	billingUsage := NewClaudeMessagesBillingUsage(&ClaudeUsage{InputTokens: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.ClaudeUsage)
	assert.Equal(t, BillingUsageSourceClaudeMessages, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticAnthropic, billingUsage.Semantic)

	cacheOnly := NewClaudeMessagesBillingUsage(&ClaudeUsage{
		CacheCreation: &ClaudeCacheCreationUsage{Ephemeral5mInputTokens: 4},
	})
	require.NotNil(t, cacheOnly)
}

func TestNewOpenAIChatBillingUsageRequiresTokenContent(t *testing.T) {
	require.Nil(t, NewOpenAIChatBillingUsage(nil))
	require.Nil(t, NewOpenAIChatBillingUsage(&Usage{}))

	billingUsage := NewOpenAIChatBillingUsage(&Usage{PromptTokens: 1})
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.OpenAIUsage)
	assert.Equal(t, BillingUsageSourceOAIChat, billingUsage.Source)
	assert.Equal(t, BillingUsageSemanticOpenAI, billingUsage.Semantic)
	assert.Equal(t, 1, billingUsage.OpenAIUsage.PromptTokens)
}

func TestNewEstimatedGeminiChatBillingUsage(t *testing.T) {
	billingUsage := NewEstimatedGeminiChatBillingUsage(&Usage{
		PromptTokens:     11,
		CompletionTokens: 7,
	})

	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.GeminiUsageMetadata)
	assert.True(t, billingUsage.Estimated)
	assert.Equal(t, 11, billingUsage.GeminiUsageMetadata.PromptTokenCount)
	assert.Equal(t, 7, billingUsage.GeminiUsageMetadata.CandidatesTokenCount)
	assert.Equal(t, 18, billingUsage.GeminiUsageMetadata.TotalTokenCount)
}

func TestNewEstimatedGeminiChatBillingUsagePreservesDetails(t *testing.T) {
	usage := &Usage{
		PromptTokens:     100,
		CompletionTokens: 23, // includes 3 reasoning tokens
		TotalTokens:      123,
	}
	usage.PromptTokensDetails.CachedTokens = 7
	usage.PromptTokensDetails.TextTokens = 80
	usage.PromptTokensDetails.ImageTokens = 12
	usage.PromptTokensDetails.AudioTokens = 8
	usage.CompletionTokenDetails.ReasoningTokens = 3
	usage.CompletionTokenDetails.TextTokens = 15
	usage.CompletionTokenDetails.ImageTokens = 5

	billingUsage := NewEstimatedGeminiChatBillingUsage(usage)
	require.NotNil(t, billingUsage)
	require.NotNil(t, billingUsage.GeminiUsageMetadata)
	meta := billingUsage.GeminiUsageMetadata

	assert.True(t, billingUsage.Estimated)
	assert.Equal(t, 100, meta.PromptTokenCount)
	assert.Equal(t, 20, meta.CandidatesTokenCount)
	assert.Equal(t, 3, meta.ThoughtsTokenCount)
	assert.Equal(t, 123, meta.TotalTokenCount)
	assert.Equal(t, 7, meta.CachedContentTokenCount)
	assert.Equal(t, []GeminiPromptTokensDetails{
		{Modality: "TEXT", TokenCount: 80},
		{Modality: "IMAGE", TokenCount: 12},
		{Modality: "AUDIO", TokenCount: 8},
	}, meta.PromptTokensDetails)
	assert.Equal(t, []GeminiPromptTokensDetails{
		{Modality: "TEXT", TokenCount: 15},
		{Modality: "IMAGE", TokenCount: 5},
	}, meta.CandidatesTokensDetails)
}

func TestBillingUsageJSONUsesProtocolNamedFields(t *testing.T) {
	billingUsage := &BillingUsage{
		OpenAIUsage:         &Usage{PromptTokens: 1, BillingUsage: NewClaudeMessagesBillingUsage(&ClaudeUsage{InputTokens: 9})},
		ClaudeUsage:         &ClaudeUsage{InputTokens: 2, BillingUsage: NewOpenAIChatBillingUsage(&Usage{PromptTokens: 8})},
		GeminiUsageMetadata: &GeminiUsageMetadata{PromptTokenCount: 3, BillingUsage: NewOpenAIChatBillingUsage(&Usage{PromptTokens: 7})},
	}

	data, err := common.Marshal(billingUsage)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"openai_usage"`)
	assert.Contains(t, string(data), `"claude_usage"`)
	assert.Contains(t, string(data), `"gemini_usage_metadata"`)
	assert.NotContains(t, string(data), `"usage":`)
	assert.NotContains(t, string(data), `"usage_metadata"`)

	clone := CloneBillingUsage(billingUsage)
	require.NotNil(t, clone.OpenAIUsage)
	require.NotNil(t, clone.ClaudeUsage)
	require.NotNil(t, clone.GeminiUsageMetadata)
	assert.Nil(t, clone.OpenAIUsage.BillingUsage)
	assert.Nil(t, clone.ClaudeUsage.BillingUsage)
	assert.Nil(t, clone.GeminiUsageMetadata.BillingUsage)
}

func TestBillingUsageNotSerializedOnWireDTOs(t *testing.T) {
	u := &Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3, BillingUsage: &BillingUsage{Source: BillingUsageSourceOAIChat, Semantic: BillingUsageSemanticOpenAI}}
	b, err := common.Marshal(u)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "billing_usage")

	c := &ClaudeUsage{InputTokens: 1, BillingUsage: &BillingUsage{Source: BillingUsageSourceClaudeMessages}}
	b2, err := common.Marshal(c)
	require.NoError(t, err)
	assert.NotContains(t, string(b2), "billing_usage")

	g := &GeminiUsageMetadata{PromptTokenCount: 1, BillingUsage: &BillingUsage{Source: BillingUsageSourceGeminiChat}}
	b3, err := common.Marshal(g)
	require.NoError(t, err)
	assert.NotContains(t, string(b3), "billing_usage")
}
