package system_setting

// DefaultUpdateCheckRepoAPIURL is the default GitHub Releases API endpoint used
// when the admin has not configured UpdateCheckRepoAPIURL. ggapi defaults to
// the private team repo; operators can override it in System Settings.
const DefaultUpdateCheckRepoAPIURL = "https://api.github.com/repos/AkumaRealLabs/ggapi/releases/latest"

// UpdateCheckRepoAPIURL is the full URL used by the server-side update checker
// (typically .../repos/{owner}/{repo}/releases/latest). Empty means use DefaultUpdateCheckRepoAPIURL.
var UpdateCheckRepoAPIURL = DefaultUpdateCheckRepoAPIURL

// UpdateCheckGitHubToken is an optional GitHub PAT (or fine-grained token) sent
// as Authorization: Bearer when fetching the release API. Required for private
// repositories. Never expose this value via GetOptions (key ends with Token).
var UpdateCheckGitHubToken = ""
