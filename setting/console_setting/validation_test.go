package console_setting

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func announcementJSON(t *testing.T, content string) string {
	t.Helper()
	payload := []map[string]interface{}{
		{
			"content":     content,
			"publishDate": time.Now().UTC().Format(time.RFC3339),
			"type":        "default",
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return string(raw)
}

func faqJSON(t *testing.T, question, answer string) string {
	t.Helper()
	payload := []map[string]interface{}{
		{
			"question": question,
			"answer":   answer,
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return string(raw)
}

func TestValidateAnnouncements_RuneCountNotBytes(t *testing.T) {
	// 200 Chinese runes → 600 UTF-8 bytes. Old len(content) rejected this;
	// rune count must accept it under the 500-character limit.
	chinese200 := strings.Repeat("测", 200)
	require.Greater(t, len(chinese200), 500)
	require.Equal(t, 200, len([]rune(chinese200)))

	err := ValidateConsoleSettings(announcementJSON(t, chinese200), "Announcements")
	assert.NoError(t, err)

	// 501 Chinese runes must fail with the same character-limit message.
	chinese501 := strings.Repeat("测", 501)
	err = ValidateConsoleSettings(announcementJSON(t, chinese501), "Announcements")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过500字符")
}

func TestValidateFAQ_RuneCountNotBytes(t *testing.T) {
	// 150 Chinese runes for question (byte length 450) must pass 200-char limit.
	q := strings.Repeat("问", 150)
	a := strings.Repeat("答", 300)
	assert.NoError(t, ValidateConsoleSettings(faqJSON(t, q, a), "FAQ"))

	q201 := strings.Repeat("问", 201)
	err := ValidateConsoleSettings(faqJSON(t, q201, a), "FAQ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不能超过200字符")
}
