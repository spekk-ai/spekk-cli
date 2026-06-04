package update

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/version"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"v1.2.3", []int{1, 2, 3}},
		{"0.1.0", []int{0, 1, 0}},
		{"10.20.30", []int{10, 20, 30}},
		{"1.2.3-dirty", []int{1, 2, 3}},
		{"1", []int{1}},
		{"dev", []int{0}},
	}

	for _, tt := range tests {
		got := ParseVersion(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseVersion(%q) = %v, want %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.1.0", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.1.0", false},
		{"0.9.0", "1.0.0", false},
		{"v1.1.0", "v1.0.0", true},
		{"1.1.0", "v1.0.0", true},
	}

	for _, tt := range tests {
		got := IsNewer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestBinaryName(t *testing.T) {
	tests := []struct {
		goos, goarch, ver string
		want              string
	}{
		{"darwin", "arm64", "1.0.0", "spekk-darwin-arm64-v1.0.0"},
		{"linux", "amd64", "2.1.3", "spekk-linux-amd64-v2.1.3"},
		{"windows", "amd64", "1.0.0", "spekk-windows-amd64-v1.0.0.exe"},
	}

	for _, tt := range tests {
		got := BinaryName(tt.goos, tt.goarch, tt.ver)
		if got != tt.want {
			t.Errorf("BinaryName(%q, %q, %q) = %q, want %q", tt.goos, tt.goarch, tt.ver, got, tt.want)
		}
	}
}

func TestLatestVersionFromNames(t *testing.T) {
	names := []string{
		"spekk-darwin-arm64-v1.0.0",
		"spekk-darwin-arm64-v1.2.0",
		"spekk-darwin-arm64-v1.1.0",
		"spekk-linux-amd64-v1.3.0",
		"spekk-darwin-amd64-v2.0.0",
		"spekk-windows-amd64-v1.0.0.exe",
	}

	tests := []struct {
		goos, goarch string
		want         string
	}{
		{"darwin", "arm64", "1.2.0"},
		{"linux", "amd64", "1.3.0"},
		{"darwin", "amd64", "2.0.0"},
		{"windows", "amd64", "1.0.0"},
		{"linux", "arm64", ""}, // no matching artifacts
	}

	for _, tt := range tests {
		got := LatestVersionFromNames(names, tt.goos, tt.goarch)
		if got != tt.want {
			t.Errorf("LatestVersionFromNames(..., %q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

// roundTripFunc adapts a function to http.RoundTripper for test mocking.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchLatestVersion(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	body := `[{"name":"spekk-darwin-arm64-v1.0.0"},{"name":"spekk-darwin-arm64-v1.2.0"},{"name":"spekk-linux-amd64-v1.1.0"}]`

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Verify auth
			user, pass, ok := req.BasicAuth()
			if !ok || user != "test-user" || pass != "test-token" {
				t.Error("expected basic auth with user:token")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}),
	}

	ver, err := FetchLatestVersion("test-user", "test-token", "thinknimble", "darwin", "arm64")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ver != "1.2.0" {
		t.Errorf("got %q, want %q", ver, "1.2.0")
	}
}

func TestFetchLatestVersionAPIError(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 401,
				Body:       io.NopCloser(bytes.NewBufferString("Unauthorized")),
			}, nil
		}),
	}

	_, err := FetchLatestVersion("bad-user", "bad-token", "thinknimble", "darwin", "arm64")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestRunMissingUser(t *testing.T) {
	t.Setenv("GEMFURY_USER", "")
	t.Setenv("GEMFURY_TOKEN", "test-token")
	err := Run(false)
	if err == nil || err.Error() != "GEMFURY_USER environment variable is required\nSet this to your personal Gemfury username" {
		t.Errorf("expected user error, got: %v", err)
	}
}

func TestRunMissingToken(t *testing.T) {
	t.Setenv("GEMFURY_USER", "test-user")
	t.Setenv("GEMFURY_TOKEN", "")
	err := Run(false)
	if err == nil || err.Error() != "GEMFURY_TOKEN environment variable is required\nGet your token from https://manage.fury.io" {
		t.Errorf("expected token error, got: %v", err)
	}
}

func TestRunDevBuild(t *testing.T) {
	t.Setenv("GEMFURY_USER", "test-user")
	t.Setenv("GEMFURY_TOKEN", "test-token")
	original := version.Version
	version.Version = "dev"
	defer func() { version.Version = original }()

	err := Run(false)
	if err == nil || err.Error() != "cannot update a development build; install a released version first" {
		t.Errorf("expected dev build error, got: %v", err)
	}
}

func TestDownloadAndReplace(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	newContent := []byte("#!/bin/sh\necho new-binary")

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			// Verify auth header uses user:token
			user, pass, ok := req.BasicAuth()
			if !ok || user != "test-user" || pass != "test-token" {
				t.Error("expected basic auth with user:token on download request")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewReader(newContent)),
			}, nil
		}),
	}

	// Create a temp "binary" to replace
	dir := t.TempDir()
	binPath := filepath.Join(dir, "spekk")
	if err := os.WriteFile(binPath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	err := downloadAndReplace("https://example.com/binary", "test-user", "test-token", binPath)
	if err != nil {
		t.Fatalf("downloadAndReplace failed: %v", err)
	}

	got, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newContent) {
		t.Errorf("binary content = %q, want %q", got, newContent)
	}

	// Verify executable permission
	info, _ := os.Stat(binPath)
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		t.Error("binary should be executable")
	}
}
