package sandbox

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spekk-ai/spekk-cli/internal/version"
)

// sandboxRoundTrip injects a custom HTTP transport for the release download
// client in tests.
type sandboxRoundTrip func(*http.Request) (*http.Response, error)

func (f sandboxRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// The agent runs on the sandbox, so its build is chosen from the machine's
// `uname -m`, not the operator's. Deploying the wrong one would report success
// and then fail to start, so an architecture spekk has no agent for is refused
// by name rather than guessed at.
// detectArch must read the machine name from stdout alone. ssh writes its
// own warnings to stderr, and a host whose sshd has no post-quantum key
// exchange gets three of them on every connection, so a combined read
// refuses a supported machine by the text of the warning.
func TestDetectArchIgnoresSSHWarnings(t *testing.T) {
	isolateConfig(t)
	bin := t.TempDir()
	fake := "#!/bin/sh\n" +
		"echo '** WARNING: connection is not using a post-quantum key exchange algorithm.' >&2\n" +
		"echo '** This session may be vulnerable to \"store now, decrypt later\" attacks.' >&2\n" +
		"echo aarch64\n"
	if err := os.WriteFile(filepath.Join(bin, "ssh"), []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	arch, err := detectArch(&SandboxMeta{IP: "192.0.2.1", SSHUser: "pi"}, "pi-box")
	if err != nil || arch != "arm64" {
		t.Errorf("detectArch() = %q, %v; want arm64", arch, err)
	}
}

func TestArchFromUname(t *testing.T) {
	ok := map[string]string{
		"x86_64":  "amd64",
		"amd64":   "amd64",
		"aarch64": "arm64",
		"arm64":   "arm64",
	}
	for in, want := range ok {
		got, err := archFromUname(in)
		if err != nil {
			t.Errorf("archFromUname(%q) errored: %v", in, err)
		}
		if got != want {
			t.Errorf("archFromUname(%q) = %q, want %q", in, got, want)
		}
	}

	for _, in := range []string{"armv7l", "i686", "riscv64", ""} {
		if _, err := archFromUname(in); err == nil {
			t.Errorf("archFromUname(%q) should be refused, not guessed", in)
		}
	}
}

// releaseTag turns an operator's --release into the tag artifacts come from:
// what they pinned, else the release of this CLI, else the latest release.
func TestReleaseTag(t *testing.T) {
	if got := releaseTag(""); got != "latest" {
		t.Errorf("releaseTag(\"\") = %q, want \"latest\" for a dev build", got)
	}
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "v1.40.2"
	if got := releaseTag(""); got != "v1.40.2" {
		t.Errorf("releaseTag(\"\") = %q, want the release of this CLI", got)
	}
	if got := releaseTag("exp-sandbox-arm64-2"); got != "exp-sandbox-arm64-2" {
		t.Errorf("releaseTag pinned = %q, want the pinned tag", got)
	}
}

// downloadBinaries fetches the agent and the CLI for the target arch from the
// anonymous release CDN (no token), and keeps the sha256 that the release
// publishes for each, which the machine checks before it installs.
func TestDownloadBinariesSelectsArchAssets(t *testing.T) {
	orig, origDigests := githubHTTPClient, fetchDigests
	defer func() { githubHTTPClient, fetchDigests = orig, origDigests }()
	fetchDigests = func(tag string) (map[string]string, error) {
		return map[string]string{"sandbox-linux-arm64": "aa", "spekk-linux-arm64": "bb", "spekk-linux-amd64": "cc"}, nil
	}

	var requested []string
	githubHTTPClient = &http.Client{
		Transport: sandboxRoundTrip(func(req *http.Request) (*http.Response, error) {
			requested = append(requested, req.URL.Path)
			if req.Header.Get("Authorization") != "" {
				t.Errorf("CDN download must be anonymous, got Authorization header")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("bytes of " + filepath.Base(req.URL.Path))), Header: make(http.Header)}, nil
		}),
	}

	a := &releaseArtifacts{Version: "exp-x"}
	if err := a.downloadBinaries("arm64"); err != nil {
		t.Fatalf("downloadBinaries: %v", err)
	}
	defer a.cleanup()

	if len(requested) != 2 || !strings.HasSuffix(requested[0], "/releases/download/exp-x/sandbox-linux-arm64") || !strings.HasSuffix(requested[1], "/releases/download/exp-x/spekk-linux-arm64") {
		t.Errorf("expected the two arm64 CDN assets to be fetched, hit %q", requested)
	}
	for path, want := range map[string]string{a.BinaryPath: "bytes of sandbox-linux-arm64", a.CLIPath: "bytes of spekk-linux-arm64"} {
		if got, err := os.ReadFile(path); err != nil || string(got) != want {
			t.Errorf("%s = %q (%v), want %q", path, got, err, want)
		}
	}
	if a.AgentSHA256 != "aa" || a.CLISHA256 != "bb" {
		t.Errorf("digests = %q, %q; want the ones of the arm64 assets", a.AgentSHA256, a.CLISHA256)
	}
}

// A release asset with no published sha256 cannot be checked, so nothing is
// downloaded, and the error names the asset.
func TestDownloadBinariesRefusesAnAssetWithNoDigest(t *testing.T) {
	origDigests := fetchDigests
	defer func() { fetchDigests = origDigests }()
	fetchDigests = func(tag string) (map[string]string, error) {
		return map[string]string{"sandbox-linux-amd64": "aa"}, nil
	}

	a := &releaseArtifacts{Version: "v1.0.0"}
	err := a.downloadBinaries("amd64")
	if err == nil || !strings.Contains(err.Error(), `no sha256 for "spekk-linux-amd64"`) {
		t.Fatalf("error = %v, want one that names the asset with no digest", err)
	}
	if a.BinaryPath != "" {
		t.Errorf("no file should be written, got %q", a.BinaryPath)
	}
}

// The digests go into a shell script, so only a well-formed sha256 is kept.
func TestReleaseDigests(t *testing.T) {
	orig := githubHTTPClient
	defer func() { githubHTTPClient = orig }()
	good := strings.Repeat("ab", 32)
	githubHTTPClient = &http.Client{
		Transport: sandboxRoundTrip(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://api.github.com/repos/spekk-ai/spekk-cli/releases/tags/v1.2.3" {
				t.Errorf("requested %s", req.URL)
			}
			body := `{"assets":[{"name":"a","digest":"sha256:` + good + `"},{"name":"b","digest":"sha256:x' ; rm -rf /"},{"name":"c","digest":null}]}`
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		}),
	}
	got, err := releaseDigests("v1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["a"] != good {
		t.Errorf("digests = %v, want only a", got)
	}
}

// A release only carries the architectures it published. A 404 for the asset
// must fail before deploy, naming the missing asset, rather than fall back to a
// binary the machine cannot run.
func TestDownloadBinariesMissingArch(t *testing.T) {
	orig, origDigests := githubHTTPClient, fetchDigests
	defer func() { githubHTTPClient, fetchDigests = orig, origDigests }()
	fetchDigests = func(tag string) (map[string]string, error) {
		return map[string]string{"sandbox-linux-arm64": "aa", "spekk-linux-arm64": "bb"}, nil
	}

	githubHTTPClient = &http.Client{
		Transport: sandboxRoundTrip(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("Not Found")), Header: make(http.Header)}, nil
		}),
	}

	a := &releaseArtifacts{Version: "v1.0.0"}
	err := a.downloadBinaries("arm64")
	if err == nil {
		t.Fatal("expected an error for an arch the release does not carry")
	}
	if !strings.Contains(err.Error(), "sandbox-linux-arm64") {
		t.Errorf("error should name the missing asset, got: %v", err)
	}
	if a.BinaryPath != "" {
		t.Errorf("no file should be written when the asset is missing, got %q", a.BinaryPath)
	}
}

// resolveReleaseTag turns ""/"latest" into a concrete tag via the CDN redirect
// and passes any pinned tag through untouched.
func TestResolveReleaseTag(t *testing.T) {
	orig := githubHTTPClient
	defer func() { githubHTTPClient = orig }()

	var hits int
	githubHTTPClient = &http.Client{
		Transport: sandboxRoundTrip(func(req *http.Request) (*http.Response, error) {
			hits++
			h := make(http.Header)
			h.Set("Location", "https://github.com/spekk-ai/spekk-cli/releases/tag/v1.30.0")
			return &http.Response{StatusCode: 302, Body: io.NopCloser(strings.NewReader("")), Header: h}, nil
		}),
	}

	for _, in := range []string{"", "latest"} {
		got, err := resolveReleaseTag(in)
		if err != nil {
			t.Fatalf("resolveReleaseTag(%q): %v", in, err)
		}
		if got != "v1.30.0" {
			t.Errorf("resolveReleaseTag(%q) = %q, want the tag from the redirect", in, got)
		}
	}

	if got, _ := resolveReleaseTag("exp-arm64"); got != "exp-arm64" {
		t.Errorf("a pinned tag must pass through unchanged, got %q", got)
	}
	if hits != 2 {
		t.Errorf("only \"\"/\"latest\" should hit the network; got %d requests", hits)
	}
}

// --release pins the release the artifacts come from. Create fetches before it
// touches the machine, so capturing the tag the fetch is asked for is enough to
// pin the behavior; an empty --release must fall back to "latest".
func TestCreatePassesReleaseTagToFetch(t *testing.T) {
	for _, tt := range []struct {
		name    string
		release string
		want    string
	}{
		{"pinned", "exp-sandbox-arm64-2", "exp-sandbox-arm64-2"},
		{"default", "", "latest"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfig(t)
			useTempStore(t)
			for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"} {
				t.Setenv(v, "x")
			}

			var gotTag string
			orig := fetchArtifacts
			fetchArtifacts = func(tag string) (*releaseArtifacts, error) {
				gotTag = tag
				return nil, fmt.Errorf("stop before touching anything")
			}
			t.Cleanup(func() { fetchArtifacts = orig })

			_ = Create(nil, CreateOptions{Name: "box", IP: "9.9.9.9", SSHKey: "irrelevant", Auth: AuthBedrock, Release: tt.release})
			if gotTag != tt.want {
				t.Errorf("Create fetched release %q, want %q", gotTag, tt.want)
			}
		})
	}
}

// Provision equips a record create left at "provisioning". It must pull the
// agent from the same --release the operator pins, defaulting to "latest".
func TestProvisionPassesReleaseTagToFetch(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"} {
		t.Setenv(v, "x")
	}
	if err := SaveSandbox("box", &SandboxMeta{Provider: "digitalocean", IP: "1.2.3.4", Status: "provisioning", Auth: string(AuthBedrock)}); err != nil {
		t.Fatal(err)
	}

	origCheck := checkReady
	checkReady = func(meta *SandboxMeta, name string) error { return nil }
	t.Cleanup(func() { checkReady = origCheck })

	var gotTag string
	origFetch := fetchArtifacts
	fetchArtifacts = func(tag string) (*releaseArtifacts, error) {
		gotTag = tag
		return nil, fmt.Errorf("stop after the marker check")
	}
	t.Cleanup(func() { fetchArtifacts = origFetch })

	_ = Provision("box", ProvisionOptions{Release: "exp-sandbox-arm64-2"})
	if gotTag != "exp-sandbox-arm64-2" {
		t.Errorf("Provision fetched release %q, want the pinned tag", gotTag)
	}
}

// Deploy redeploys the agent to a recorded sandbox and must honor --release the
// same way create and provision do.
func TestDeployPassesReleaseTagToFetch(t *testing.T) {
	isolateConfig(t)
	useTempStore(t)
	if err := SaveSandbox("box", &SandboxMeta{Provider: "digitalocean", IP: "1.2.3.4", Status: "active"}); err != nil {
		t.Fatal(err)
	}

	var gotTag string
	orig := fetchArtifacts
	fetchArtifacts = func(tag string) (*releaseArtifacts, error) {
		gotTag = tag
		return nil, fmt.Errorf("stop before scp")
	}
	t.Cleanup(func() { fetchArtifacts = orig })

	_ = Deploy("box", "exp-sandbox-arm64-2")
	if gotTag != "exp-sandbox-arm64-2" {
		t.Errorf("Deploy fetched release %q, want the pinned tag", gotTag)
	}
}

// A version that status reads on a machine warns when it is not the expected
// release, has no release stamp, or is not there. No SSH answer is not drift.
func TestVersionDrift(t *testing.T) {
	for _, tt := range []struct {
		got, want string
		warns     bool
	}{
		{"v1.38.1", "v1.38.1", false},
		{"v1.32.0", "v1.38.1", true},
		{"dev", "v1.38.1", true},
		{"dev", "", true},
		{"missing", "v1.38.1", true},
		{"v1.32.0", "", false},
		{"unknown", "v1.38.1", false},
	} {
		if got := versionDrift(tt.got, tt.want, tt.want+", the release of this spekk"); (got != "") != tt.warns {
			t.Errorf("versionDrift(%q, %q) = %q, want a warning: %v", tt.got, tt.want, got, tt.warns)
		}
	}
}
