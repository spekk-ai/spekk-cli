// Package update implements self-update functionality for the spekk CLI.
// It checks GitHub Releases for newer versions and replaces the running binary in-place.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/version"
)

const (
	repoOwner = "spekk-ai"
	repoName  = "spekk-cli"
)

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the HTTP client used for GitHub API calls. Override in tests.
var Client HTTPClient = http.DefaultClient

// releaseResponse represents the GitHub Releases API response (subset of fields).
type releaseResponse struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Run performs the self-update. If checkOnly is true, it prints the available
// version without installing.
func Run(checkOnly bool) error {
	token := os.Getenv("GH_SPEKK_TOKEN")
	if token == "" {
		return fmt.Errorf("GH_SPEKK_TOKEN environment variable is required\nSet this to a fine-grained PAT with contents:read on %s/%s", repoOwner, repoName)
	}

	current := version.Version
	if current == "dev" {
		return fmt.Errorf("cannot update a development build; install a released version first")
	}

	release, err := FetchLatestRelease(token)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return fmt.Errorf("no releases found on GitHub")
	}

	if !IsNewer(latest, current) {
		fmt.Printf("Already on latest version (%s)\n", current)
		return nil
	}

	if checkOnly {
		fmt.Printf("Current version: %s\nLatest version:  %s\nRun 'spekk update' to install\n", current, latest)
		return nil
	}

	fmt.Printf("Updating %s → %s ...\n", current, latest)

	assetName := AssetName(runtime.GOOS, runtime.GOARCH)
	downloadURL := ""
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		return fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	if err := downloadAndReplace(downloadURL, token, exePath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Updated successfully: %s → %s\n", current, latest)
	return nil
}

// FetchLatestRelease queries the GitHub Releases API for the latest release.
func FetchLatestRelease(token string) (*releaseResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var release releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}
	return &release, nil
}

func downloadAndReplace(url, token, destPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := Client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Write to a temp file in the same directory so we can atomically rename.
	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, ".spekk-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		tmp.Close()
		os.Remove(tmpName) // clean up on any error path
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return fmt.Errorf("cannot write new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot close temp file: %w", err)
	}

	// Preserve the executable bit.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(tmpName, 0755); err != nil {
			return fmt.Errorf("cannot chmod new binary: %w", err)
		}
	}

	// On Windows the running binary cannot be replaced directly; rename the
	// old one out of the way first.
	if runtime.GOOS == "windows" {
		if err := os.Rename(destPath, destPath+".old"); err != nil {
			return fmt.Errorf("cannot move old binary: %w", err)
		}
	}

	if err := os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("cannot replace binary: %w", err)
	}
	return nil
}

// AssetName returns the expected GitHub Release asset name for a given platform.
func AssetName(goos, goarch string) string {
	name := fmt.Sprintf("spekk-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// ParseVersion splits a version string like "1.2.3" or "v1.2.3" into
// a slice of ints. Non-numeric suffixes (e.g. "-dirty") are ignored.
func ParseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	// Strip any suffix after a hyphen (e.g. "1.2.3-dirty" → "1.2.3")
	if idx := strings.IndexByte(v, '-'); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	nums := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			break
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 {
		return []int{0}
	}
	return nums
}

// IsNewer reports whether version a is strictly newer than version b.
func IsNewer(a, b string) bool {
	va := ParseVersion(a)
	vb := ParseVersion(b)
	n := len(va)
	if len(vb) > n {
		n = len(vb)
	}
	for i := 0; i < n; i++ {
		ai, bi := 0, 0
		if i < len(va) {
			ai = va[i]
		}
		if i < len(vb) {
			bi = vb[i]
		}
		if ai != bi {
			return ai > bi
		}
	}
	return false
}
