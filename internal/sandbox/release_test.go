package sandbox

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// sandboxRoundTrip injects a custom HTTP transport for the release download
// client in tests.
type sandboxRoundTrip func(*http.Request) (*http.Response, error)

func (f sandboxRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// The agent runs on the sandbox, so its build is chosen from the machine's
// `uname -m`, not the operator's. Deploying the wrong one would report success
// and then fail to start, so an architecture spekk has no agent for is refused
// by name rather than guessed at.
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
// what they pinned, or the latest published release when they pinned nothing.
func TestReleaseTag(t *testing.T) {
	if got := releaseTag(""); got != "latest" {
		t.Errorf("releaseTag(\"\") = %q, want \"latest\"", got)
	}
	if got := releaseTag("exp-sandbox-arm64-2"); got != "exp-sandbox-arm64-2" {
		t.Errorf("releaseTag pinned = %q, want the pinned tag", got)
	}
}

// downloadAgentBinary fetches the build for the target arch, so it must request
// the sandbox-linux-<arch> asset and write exactly those bytes to BinaryPath.
func TestDownloadAgentBinarySelectsArchAsset(t *testing.T) {
	orig := githubHTTPClient
	defer func() { githubHTTPClient = orig }()

	const payload = "arm64-agent-bytes"
	var requested string
	githubHTTPClient = &http.Client{
		Transport: sandboxRoundTrip(func(req *http.Request) (*http.Response, error) {
			requested = req.URL.Path
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
		}),
	}

	a := &releaseArtifacts{
		token: "tok",
		release: &githubRelease{
			TagName: "exp-x",
			Assets: []githubAsset{
				{ID: 11, Name: "sandbox-linux-amd64"},
				{ID: 22, Name: "sandbox-linux-arm64"},
			},
		},
	}
	if err := a.downloadAgentBinary("arm64"); err != nil {
		t.Fatalf("downloadAgentBinary: %v", err)
	}
	defer os.Remove(a.BinaryPath)

	if !strings.HasSuffix(requested, "/releases/assets/22") {
		t.Errorf("expected the arm64 asset (id 22) to be fetched, hit %q", requested)
	}
	got, err := os.ReadFile(a.BinaryPath)
	if err != nil {
		t.Fatalf("reading downloaded binary: %v", err)
	}
	if string(got) != payload {
		t.Errorf("BinaryPath contents = %q, want %q", got, payload)
	}
}

// A release only carries the architectures it published. Asking for one it does
// not have must fail before deploy, naming the missing asset, rather than fall
// back to a binary the machine cannot run.
func TestDownloadAgentBinaryMissingArch(t *testing.T) {
	a := &releaseArtifacts{
		token:   "tok",
		release: &githubRelease{TagName: "v1.0.0", Assets: []githubAsset{{ID: 11, Name: "sandbox-linux-amd64"}}},
	}
	err := a.downloadAgentBinary("arm64")
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
			for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN"} {
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
	for _, v := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN"} {
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
