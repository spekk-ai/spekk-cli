package install

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

// InstallStatus describes whether a remote skill is already installed on disk.
type InstallStatus int

const (
	// StatusNotInstalled means neither the local nor global scope has the file.
	StatusNotInstalled InstallStatus = iota
	// StatusLocal means <cwd>/.spekk/skills/<agent>/<skill>.md exists.
	StatusLocal
	// StatusGlobal means ~/.spekk/skills/<agent>/<skill>.md exists.
	StatusGlobal
	// StatusBoth means the skill exists in both scopes.
	StatusBoth
)

// Annotation returns the human-readable suffix for a status, suitable for
// the trailing column of `spekk install --list` output.
func (s InstallStatus) Annotation() string {
	switch s {
	case StatusLocal:
		return "installed (local)"
	case StatusGlobal:
		return "installed (global)"
	case StatusBoth:
		return "installed (local, global)"
	default:
		return "not installed"
	}
}

// RemoteSkill is one entry returned by ListRemoteSkills.
type RemoteSkill struct {
	Name   string
	Status InstallStatus
}

// contentsEntry mirrors the subset of GitHub's contents API JSON we read.
type contentsEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ListURL returns the GitHub contents API URL for an agent's skill directory.
func ListURL(agent string) string {
	return fmt.Sprintf("%s/%s", APIBase(), agent)
}

// FetchListRaw fetches the contents listing for an agent and returns the
// raw body plus the HTTP status code. It is split out so tests can inject
// a stub and so the 403 rate-limit case can be detected by callers.
func FetchListRaw(url string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request for %s: %w", url, err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := defaultHTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, fmt.Errorf("read body from %s: %w", url, readErr)
	}
	return body, resp.StatusCode, nil
}

// FetchListFn is the contract used by ListRemoteSkills so tests can inject
// an httptest-backed stub.
type FetchListFn func(url string) ([]byte, int, error)

// ListRemoteSkills queries the registry contents API for an agent and
// returns one RemoteSkill per `.md` file entry, annotated with whether the
// skill is already present in the local or global scope.
//
// Errors are wrapped to surface user-friendly messages. A 403 from the
// upstream is recognized and the error mentions the GitHub unauthenticated
// 60 req/hr limit.
func ListRemoteSkills(agent, cwd, home string, fetch FetchListFn) ([]RemoteSkill, error) {
	if !isValidAgent(agent) {
		return nil, unknownAgentError(agent)
	}

	url := ListURL(agent)
	body, status, err := fetch(url)
	if err != nil {
		return nil, err
	}

	switch {
	case status == http.StatusForbidden:
		return nil, fmt.Errorf("registry list rejected with 403 at %s (GitHub unauthenticated API limit is 60 requests/hour per IP)", url)
	case status == http.StatusNotFound:
		return nil, fmt.Errorf("registry directory not found: %s", url)
	case status < 200 || status >= 300:
		return nil, fmt.Errorf("http %d listing %s", status, url)
	}

	var entries []contentsEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("parse registry response from %s: %w", url, err)
	}

	var skills []RemoteSkill
	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		if !strings.HasSuffix(e.Name, ".md") {
			continue
		}
		stem := strings.TrimSuffix(e.Name, ".md")
		skills = append(skills, RemoteSkill{
			Name:   stem,
			Status: classifyInstalled(cwd, home, agent, stem),
		})
	}
	return skills, nil
}

// classifyInstalled checks the on-disk presence of <skill>.md in the local
// and global scopes and returns the corresponding InstallStatus.
func classifyInstalled(cwd, home, agent, skill string) InstallStatus {
	local := cwd != "" && fileExists(filepath.Join(cwd, ".spekk", "skills", agent, skill+".md"))
	var global bool
	if globalDir, err := config.GlobalConfigDir(); err == nil {
		global = fileExists(filepath.Join(globalDir, "skills", agent, skill+".md"))
	}
	switch {
	case local && global:
		return StatusBoth
	case local:
		return StatusLocal
	case global:
		return StatusGlobal
	default:
		return StatusNotInstalled
	}
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// FormatList renders the remote skill listing for stdout. Each line is
// `<name>  —  <annotation>`. Returns a leading header and a trailing newline
// so callers can print the result directly.
func FormatList(agent string, skills []RemoteSkill) string {
	var b strings.Builder
	if len(skills) == 0 {
		fmt.Fprintf(&b, "No skills found in registry for agent %q.\n", agent)
		return b.String()
	}
	fmt.Fprintf(&b, "Remote skills for %s:\n", agent)
	for _, s := range skills {
		fmt.Fprintf(&b, "  %s  —  %s\n", s.Name, s.Status.Annotation())
	}
	return b.String()
}
