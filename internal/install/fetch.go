package install

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultRawBase = "https://raw.githubusercontent.com/spekk-ai/spekk-skills/main"
	defaultAPIBase = "https://api.github.com/repos/spekk-ai/spekk-skills/contents"

	rawBaseEnv = "SPEKK_SKILLS_RAW_BASE"
	apiBaseEnv = "SPEKK_SKILLS_API_BASE"

	userAgent = "spekk-cli"
)

// defaultHTTPClient is used by FetchSkill and FetchURL. Tests inspect its
// timeout to verify the 30s contract.
var defaultHTTPClient = &http.Client{Timeout: 30 * time.Second}

// RawBase returns the base URL for raw skill content. Honors
// SPEKK_SKILLS_RAW_BASE if set; trailing slashes are trimmed.
func RawBase() string {
	if v := strings.TrimRight(os.Getenv(rawBaseEnv), "/"); v != "" {
		return v
	}
	return defaultRawBase
}

// APIBase returns the base URL for the registry contents API. Honors
// SPEKK_SKILLS_API_BASE if set; trailing slashes are trimmed.
func APIBase() string {
	if v := strings.TrimRight(os.Getenv(apiBaseEnv), "/"); v != "" {
		return v
	}
	return defaultAPIBase
}

// SkillURL returns the raw URL where `<agent>/<skill>.md` lives.
func SkillURL(agent, skill string) string {
	return fmt.Sprintf("%s/%s/%s.md", RawBase(), agent, skill)
}

// FetchSkill downloads the skill markdown for the given agent/skill from
// the configured raw base and returns the body bytes verbatim.
func FetchSkill(agent, skill string) ([]byte, error) {
	return FetchURL(SkillURL(agent, skill))
}

// FetchURL downloads the URL and returns the body bytes verbatim. A 404
// surfaces a "not found" error; other non-2xx responses surface a
// "http <code>: <url>" error.
func FetchURL(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found: %s", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body from %s: %w", url, err)
	}
	return body, nil
}
