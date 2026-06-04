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
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required\nSet this to a fine-grained PAT with contents:read on %s/%s", repoOwner, repoName)
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
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Write to temp file in same directory for atomic rename
	dir := filepath.Dir(destPath)
	tmp, err := os.CreateTemp(dir, "spekk-update-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w (check directory permissions for %s)", err, dir)
	}
	tmpPath := tmp.Name()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("download interrupted: %w", err)
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot set permissions: %w", err)
	}

	// Replace binary (Windows needs special handling for running binaries)
	if runtime.GOOS == "windows" {
		oldPath := destPath + ".old"
		os.Remove(oldPath)
		if err := os.Rename(destPath, oldPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot move old binary: %w", err)
		}
		if err := os.Rename(tmpPath, destPath); err != nil {
			os.Rename(oldPath, destPath) // try to restore
			os.Remove(tmpPath)
			return fmt.Errorf("cannot install new binary: %w", err)
		}
		os.Remove(oldPath)
	} else {
		if err := os.Rename(tmpPath, destPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot replace binary: %w (you may need to run with sudo)", err)
		}
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

// IsNewer returns true if version a is strictly newer than version b.
func IsNewer(a, b string) bool {
	ap := ParseVersion(a)
	bp := ParseVersion(b)

	maxLen := len(ap)
	if len(bp) > maxLen {
		maxLen = len(bp)
	}

	for i := 0; i < maxLen; i++ {
		var ai, bi int
		if i < len(ap) {
			ai = ap[i]
		}
		if i < len(bp) {
			bi = bp[i]
		}
		if ai != bi {
			return ai > bi
		}
	}
	return false
}

// ParseVersion splits a version string like "v1.2.3" into integer components [1, 2, 3].
func ParseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		// Strip non-numeric suffix (e.g., "3-dirty" → 3)
		numStr := ""
		for _, c := range p {
			if c >= '0' && c <= '9' {
				numStr += string(c)
			} else {
				break
			}
		}
		if numStr != "" {
			result[i], _ = strconv.Atoi(numStr)
		}
	}
	return result
}
