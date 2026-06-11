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

func TestAssetName(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
	}{
		{"darwin", "arm64", "spekk-darwin-arm64"},
		{"linux", "amd64", "spekk-linux-amd64"},
		{"windows", "amd64", "spekk-windows-amd64.exe"},
	}

	for _, tt := range tests {
		got := AssetName(tt.goos, tt.goarch)
		if got != tt.want {
			t.Errorf("AssetName(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

// roundTripFunc lets us inject a custom HTTP transport in tests.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchLatestRelease(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	body := `{"tag_name":"v1.2.0","assets":[{"name":"spekk-darwin-arm64","browser_download_url":"https://github.com/spekk-ai/spekk-cli/releases/download/v1.2.0/spekk-darwin-arm64"},{"name":"spekk-linux-amd64","browser_download_url":"https://github.com/spekk-ai/spekk-cli/releases/download/v1.2.0/spekk-linux-amd64"}]}`

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Authorization"); got != "token test-token" {
				t.Errorf("Authorization = %q, want %q", got, "token test-token")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}),
	}

	release, err := FetchLatestRelease("test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release.TagName != "v1.2.0" {
		t.Errorf("TagName = %q, want %q", release.TagName, "v1.2.0")
	}
	if len(release.Assets) != 2 {
		t.Fatalf("got %d assets, want 2", len(release.Assets))
	}
}

func TestFetchLatestReleaseAPIError(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 401,
				Body:       io.NopCloser(bytes.NewBufferString("Bad credentials")),
			}, nil
		}),
	}

	_, err := FetchLatestRelease("bad-token")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestRunMissingToken(t *testing.T) {
	t.Setenv("GH_SPEKK_TOKEN", "")
	err := Run(false)
	if err == nil {
		t.Fatal("expected error for missing GH_SPEKK_TOKEN")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("GH_SPEKK_TOKEN")) {
		t.Errorf("error should mention GH_SPEKK_TOKEN, got: %v", err)
	}
}

func TestRunDevBuild(t *testing.T) {
	t.Setenv("GH_SPEKK_TOKEN", "test-token")
	original := version.Version
	version.Version = "dev"
	defer func() { version.Version = original }()

	err := Run(false)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("development build")) {
		t.Errorf("expected dev build error, got: %v", err)
	}
}

func TestDownloadAndReplace(t *testing.T) {
	original := Client
	defer func() { Client = original }()

	newContent := []byte("new-binary-content")

	Client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got := req.Header.Get("Authorization"); got != "token test-token" {
				t.Errorf("Authorization = %q, want %q", got, "token test-token")
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBuffer(newContent)),
			}, nil
		}),
	}

	dir := t.TempDir()
	binPath := filepath.Join(dir, "spekk")
	if err := os.WriteFile(binPath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	err := downloadAndReplace("https://example.com/binary", "test-token", binPath)
	if err != nil {
		t.Fatalf("downloadAndReplace failed: %v", err)
	}

	got, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newContent) {
		t.Errorf("binary content = %q, want %q", got, newContent)
	}

	info, _ := os.Stat(binPath)
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		t.Error("binary should be executable")
	}
}
