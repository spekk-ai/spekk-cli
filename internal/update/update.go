// Package update implements self-update functionality for the spekk CLI.
// It checks Gemfury for newer versions and replaces the running binary in-place.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/version"
)

const defaultAccount = "thinknimble"

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the HTTP client used for Gemfury API calls. Override in tests.
var Client HTTPClient = http.DefaultClient

// Run performs the self-update. If checkOnly is true, it prints the available
// version without installing.
func Run(checkOnly bool) error {
	user := os.Getenv("GEMFURY_USER")
	if user == "" {
		return fmt.Errorf("GEMFURY_USER environment variable is required\nSet this to your personal Gemfury username")
	}

	token := os.Getenv("GEMFURY_TOKEN")
	if token == "" {
		return fmt.Errorf("GEMFURY_TOKEN environment variable is required\nGet your token from https://manage.fury.io")
	}

	account := os.Getenv("GEMFURY_ACCOUNT")
	if account == "" {
		account = defaultAccount
	}

	current := version.Version
	if current == "dev" {
		return fmt.Errorf("cannot update a development build; install a released version first")
	}

	latest, err := FetchLatestVersion(user, token, account, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if latest == "" {
		return fmt.Errorf("no releases found for %s/%s on Gemfury", runtime.GOOS, runtime.GOARCH)
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

	binaryName := BinaryName(runtime.GOOS, runtime.GOARCH, latest)
	downloadURL := fmt.Sprintf("https://fury.io/%s/%s", account, binaryName)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("cannot resolve executable path: %w", err)
	}

	if err := downloadAndReplace(downloadURL, user, token, exePath); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	fmt.Printf("Updated successfully: %s → %s\n", current, latest)
	return nil
}

// FetchLatestVersion queries the Gemfury API for the latest version available
// for the given OS and architecture.
func FetchLatestVersion(user, token, account, goos, goarch string) (string, error) {
	url := fmt.Sprintf("https://api.fury.io/v1/users/%s/packages", account)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(user, token)

	resp, err := Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemfury API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var packages []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&packages); err != nil {
		return "", fmt.Errorf("failed to parse Gemfury response: %w", err)
	}

	var names []string
	for _, p := range packages {
		names = append(names, p.Name)
	}
	return LatestVersionFromNames(names, goos, goarch), nil
}

// LatestVersionFromNames extracts the latest version from a list of artifact names
// matching the pattern spekk-{os}-{arch}-v{version}.
func LatestVersionFromNames(names []string, goos, goarch string) string {
	pattern := fmt.Sprintf(`^spekk-%s-%s-v(.+?)(?:\.exe)?$`, regexp.QuoteMeta(goos), regexp.QuoteMeta(goarch))
	re := regexp.MustCompile(pattern)

	var versions []string
	for _, name := range names {
		if m := re.FindStringSubmatch(name); m != nil {
			versions = append(versions, m[1])
		}
	}

	if len(versions) == 0 {
		return ""
	}

	sort.Slice(versions, func(i, j int) bool {
		return IsNewer(versions[i], versions[j])
	})

	return versions[0]
}

func downloadAndReplace(url, user, token, destPath string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, token)

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

// BinaryName returns the expected Gemfury artifact name for a given platform and version.
func BinaryName(goos, goarch, ver string) string {
	name := fmt.Sprintf("spekk-%s-%s-v%s", goos, goarch, ver)
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
