package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckUpdateRejectsNonHTTPURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap["UpdateCheckRepoAPIURL"] = "file:///etc/passwd"
	common.OptionMap["UpdateCheckGitHubToken"] = ""
	common.OptionMapRWMutex.Unlock()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/option/check_update", nil)

	CheckUpdate(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
}

func TestCheckUpdateUsesDefaultWhenEmpty(t *testing.T) {
	// Only validates that empty option falls back to default constant without panicking
	// on URL parse (network call may fail in CI; we only assert not 400 invalid URL).
	gin.SetMode(gin.TestMode)

	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap["UpdateCheckRepoAPIURL"] = ""
	common.OptionMap["UpdateCheckGitHubToken"] = ""
	common.OptionMapRWMutex.Unlock()

	assert.NotEmpty(t, system_setting.DefaultUpdateCheckRepoAPIURL)
	assert.True(t, len(system_setting.DefaultUpdateCheckRepoAPIURL) > 8)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/option/check_update", nil)

	CheckUpdate(c)

	// Should not fail URL validation; may be 502 if network blocked.
	assert.NotEqual(t, http.StatusBadRequest, w.Code)
}
