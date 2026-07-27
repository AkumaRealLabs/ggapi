package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
)

type githubReleaseResponse struct {
	TagName          string `json:"tag_name"`
	Name             string `json:"name"`
	Body             string `json:"body"`
	HTMLURL          string `json:"html_url"`
	PublishedAt      string `json:"published_at"`
	Message          string `json:"message"` // GitHub error payload
	DocumentationURL string `json:"documentation_url"`
}

// CheckUpdate fetches the configured release endpoint server-side so private
// repos can use a PAT that never leaves the backend. Root-only.
func CheckUpdate(c *gin.Context) {
	checkUpdate(c, nil)
}

func checkUpdate(c *gin.Context, client *http.Client) {
	common.OptionMapRWMutex.RLock()
	apiURL := strings.TrimSpace(common.OptionMap["UpdateCheckRepoAPIURL"])
	token := strings.TrimSpace(common.OptionMap["UpdateCheckGitHubToken"])
	common.OptionMapRWMutex.RUnlock()

	if apiURL == "" {
		apiURL = system_setting.DefaultUpdateCheckRepoAPIURL
	}

	parsed, err := url.Parse(apiURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid UpdateCheckRepoAPIURL",
		})
		return
	}
	if err := validateUpdateCheckURL(parsed, token != ""); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to build update check request",
		})
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "ggapi-update-checker")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	if client == nil {
		client = service.NewPublicHTTPClient(15 * time.Second)
	}
	client.CheckRedirect = updateCheckRedirectPolicy(token != "")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": fmt.Sprintf("failed to contact release API: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "failed to read release API response",
		})
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "release not found (404). For private repos, set a PAT with Contents: Read, and confirm the API URL.",
		})
		return
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": fmt.Sprintf("release API returned %d (auth). Check PAT scope and repository access.", resp.StatusCode),
		})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": fmt.Sprintf("release API returned HTTP %d", resp.StatusCode),
		})
		return
	}

	var release githubReleaseResponse
	if err := common.Unmarshal(body, &release); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "release API returned non-JSON or unexpected payload",
		})
		return
	}
	if release.TagName == "" {
		msg := release.Message
		if msg == "" {
			msg = "unexpected release payload (missing tag_name)"
		}
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"tag_name":         release.TagName,
			"name":             release.Name,
			"body":             release.Body,
			"html_url":         release.HTMLURL,
			"published_at":     release.PublishedAt,
			"current":          common.Version,
			"is_latest":        release.TagName == common.Version,
			"source_url":       apiURL,
			"token_configured": token != "",
		},
	})
}

func validateUpdateCheckURL(parsed *url.URL, authenticated bool) error {
	if parsed == nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("invalid UpdateCheckRepoAPIURL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("UpdateCheckRepoAPIURL must use http or https")
	}
	if !authenticated {
		return nil
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("authenticated update checks require HTTPS")
	}
	if !strings.EqualFold(parsed.Hostname(), "api.github.com") || (parsed.Port() != "" && parsed.Port() != "443") {
		return fmt.Errorf("GitHub PAT may only be sent to https://api.github.com")
	}
	return nil
}

func updateCheckRedirectPolicy(authenticated bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("stopped after 10 redirects")
		}
		if err := validateUpdateCheckURL(req.URL, authenticated); err != nil {
			return fmt.Errorf("update check redirect blocked: %w", err)
		}
		return nil
	}
}
