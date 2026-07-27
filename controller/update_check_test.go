package controller

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type updateCheckRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn updateCheckRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func setUpdateCheckOptions(t *testing.T, apiURL string, token string) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	oldURL, hadURL := common.OptionMap["UpdateCheckRepoAPIURL"]
	oldToken, hadToken := common.OptionMap["UpdateCheckGitHubToken"]
	common.OptionMap["UpdateCheckRepoAPIURL"] = apiURL
	common.OptionMap["UpdateCheckGitHubToken"] = token
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if hadURL {
			common.OptionMap["UpdateCheckRepoAPIURL"] = oldURL
		} else {
			delete(common.OptionMap, "UpdateCheckRepoAPIURL")
		}
		if hadToken {
			common.OptionMap["UpdateCheckGitHubToken"] = oldToken
		} else {
			delete(common.OptionMap, "UpdateCheckGitHubToken")
		}
	})
}

func runUpdateCheck(t *testing.T, client *http.Client) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/option/check_update", nil)
	checkUpdate(c, client)
	return w
}

func TestCheckUpdateRejectsNonHTTPURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setUpdateCheckOptions(t, "file:///etc/passwd", "")
	w := runUpdateCheck(t, &http.Client{})

	require.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["success"])
}

func TestCheckUpdateSendsPATOnlyToTrustedGitHubAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setUpdateCheckOptions(t, "https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest", "secret-pat")
	client := &http.Client{Transport: updateCheckRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "Bearer secret-pat", req.Header.Get("Authorization"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"tag_name":"v1.0.0-rc.22.1"}`)),
			Request:    req,
		}, nil
	})}

	w := runUpdateCheck(t, client)

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "secret-pat")
}

func TestCheckUpdateRejectsUnsafeAuthenticatedTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []string{
		"http://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest",
		"https://example.com/releases/latest",
		"https://api.github.com.evil.example/releases/latest",
		"https://api.github.com:8443/releases/latest",
	}
	for _, apiURL := range tests {
		t.Run(apiURL, func(t *testing.T) {
			setUpdateCheckOptions(t, apiURL, "secret-pat")
			called := false
			client := &http.Client{Transport: updateCheckRoundTripFunc(func(*http.Request) (*http.Response, error) {
				called = true
				return nil, nil
			})}

			w := runUpdateCheck(t, client)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.False(t, called)
		})
	}
}

func TestCheckUpdateBlocksPATRedirectsOutsideTrustedHTTPSHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, redirectURL := range []string{
		"https://example.com/capture",
		"http://api.github.com/capture",
	} {
		t.Run(redirectURL, func(t *testing.T) {
			setUpdateCheckOptions(t, "https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest", "secret-pat")
			calls := 0
			client := &http.Client{Transport: updateCheckRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, 1, calls, "redirect target must not be contacted")
				assert.Equal(t, "Bearer secret-pat", req.Header.Get("Authorization"))
				return &http.Response{
					StatusCode: http.StatusFound,
					Status:     "302 Found",
					Header:     http.Header{"Location": []string{redirectURL}},
					Body:       http.NoBody,
					Request:    req,
				}, nil
			})}

			w := runUpdateCheck(t, client)

			require.Equal(t, http.StatusBadGateway, w.Code)
			assert.Equal(t, 1, calls)
			assert.NotContains(t, w.Body.String(), "secret-pat")
		})
	}
}
