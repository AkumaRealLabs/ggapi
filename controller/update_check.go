package controller

import (
	"cmp"
	"context"
	"errors"
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
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	Message     string `json:"message"` // GitHub error payload
}

// updateCheckRelease is the release shape consumed by the dashboard's system
// update checker.
type updateCheckRelease struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt *string `json:"published_at"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
}

func newUpdateCheckRelease(release githubReleaseResponse) updateCheckRelease {
	var publishedAt *string
	if release.PublishedAt != "" {
		publishedAt = &release.PublishedAt
	}
	return updateCheckRelease{
		TagName:     release.TagName,
		Name:        release.Name,
		Body:        release.Body,
		HTMLURL:     release.HTMLURL,
		PublishedAt: publishedAt,
		Draft:       release.Draft,
		Prerelease:  release.Prerelease,
	}
}

// parseUpdateCheckReleases accepts either a single release (…/releases/latest)
// or a release list (…/releases) and always returns a non-nil list on success.
// A nil list without error carries GitHub's error message, if any.
func parseUpdateCheckReleases(body []byte) ([]updateCheckRelease, string, error) {
	if strings.HasPrefix(strings.TrimSpace(string(body)), "[") {
		var releases []githubReleaseResponse
		if err := common.Unmarshal(body, &releases); err != nil {
			return nil, "", err
		}
		result := make([]updateCheckRelease, 0, len(releases))
		for _, release := range releases {
			if release.TagName != "" {
				result = append(result, newUpdateCheckRelease(release))
			}
		}
		return result, "", nil
	}
	var release githubReleaseResponse
	if err := common.Unmarshal(body, &release); err != nil {
		return nil, "", err
	}
	if release.TagName == "" {
		return nil, release.Message, nil
	}
	return []updateCheckRelease{newUpdateCheckRelease(release)}, "", nil
}

// updateCheckFailedMessage is the system update panel's text for a failed check.
const updateCheckFailedMessage = "Failed to check for updates"

func isGitHubRateLimited(resp *http.Response) bool {
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	return resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0"
}

// CheckUpdate fetches the configured release endpoint server-side so private
// repos can use a PAT that never leaves the backend. Any administrator can
// check; only root can change the endpoint and PAT.
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

	// A URL that does not parse leaves parsed nil, which validateUpdateCheckURL
	// rejects without echoing the URL and any credentials in it to the log.
	parsed, _ := url.Parse(apiURL)
	if err := validateUpdateCheckURL(parsed, token != ""); err != nil {
		failUpdateCheck(c, http.StatusBadRequest, updateCheckFailedMessage,
			common.LogText("update check URL rejected: %s", err.Error()))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		failUpdateCheck(c, http.StatusInternalServerError, updateCheckFailedMessage,
			common.LogText("update check request could not be built: %s", err.Error()))
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
		failUpdateCheck(c, http.StatusBadGateway, updateCheckFailedMessage,
			common.LogText("update check could not reach the release API: %s", err.Error()))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20)) // 2 MiB
	if err != nil {
		failUpdateCheck(c, http.StatusBadGateway, updateCheckFailedMessage,
			common.LogText("update check could not read the release API response: %s", err.Error()))
		return
	}

	switch {
	case isGitHubRateLimited(resp):
		failUpdateCheck(c, http.StatusTooManyRequests, "GitHub rate limit reached. Try again later.",
			common.LogText("update check hit the release API rate limit"))
		return
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		failUpdateCheck(c, http.StatusBadGateway, updateCheckFailedMessage,
			common.LogText("update check got HTTP %d from the release API; for a private repository, set a PAT with Contents read access and check the API URL", resp.StatusCode))
		return
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		failUpdateCheck(c, http.StatusBadGateway, updateCheckFailedMessage,
			common.LogText("update check got HTTP %d from the release API", resp.StatusCode))
		return
	}

	releases, githubMessage, err := parseUpdateCheckReleases(body)
	if err == nil && releases == nil {
		err = errors.New(cmp.Or(githubMessage, "missing tag_name"))
	}
	if err != nil {
		failUpdateCheck(c, http.StatusBadGateway, "Unexpected release payload",
			common.LogText("update check got an unexpected release payload: %s", err.Error()))
		return
	}

	common.ApiSuccess(c, releases)
}

// failUpdateCheck logs why a check failed and answers with the message the
// system update panel shows for that kind of failure; the panel tells only a
// rate limit apart, so the detailed reason stays in the server log.
func failUpdateCheck(c *gin.Context, status int, key string, logLine string) {
	common.SysError(logLine)
	common.ApiErrorStatus(c, status, common.NewMessage(key))
}

func validateUpdateCheckURL(parsed *url.URL, authenticated bool) error {
	if parsed == nil || !parsed.IsAbs() || parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil {
		return errors.New("invalid UpdateCheckRepoAPIURL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("UpdateCheckRepoAPIURL must use http or https")
	}
	if !authenticated {
		return nil
	}
	if parsed.Scheme != "https" {
		return errors.New("authenticated update checks require HTTPS")
	}
	if !strings.EqualFold(parsed.Hostname(), "api.github.com") || (parsed.Port() != "" && parsed.Port() != "443") {
		return errors.New("GitHub PAT may only be sent to https://api.github.com")
	}
	return nil
}

func updateCheckRedirectPolicy(authenticated bool) func(*http.Request, []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		if err := validateUpdateCheckURL(req.URL, authenticated); err != nil {
			return fmt.Errorf("update check redirect blocked: %w", err)
		}
		return nil
	}
}
