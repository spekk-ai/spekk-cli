package sandbox

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

const (
	releaseRepo = "spekk-ai/spekk-cli"

	// cloudInitKeyPlaceholder is the line in the template that is replaced with
	// the sandbox's generated public key.
	cloudInitKeyPlaceholder = "ssh-ed25519 AAAA... your-key-here"
)

// sandboxAssetName is the agent binary published for a given GOARCH — the
// sandbox machine's arch, not the operator's.
func sandboxAssetName(arch string) string {
	return "sandbox-linux-" + arch
}

// githubHTTPClient downloads release assets. spekk-cli is a public repo, so we
// fetch from the anonymous release CDN (github.com/<repo>/releases/download/...)
// rather than the api.github.com asset endpoint: no token, and no exposure to
// the 60/hr unauthenticated API rate limit. The default redirect policy follows
// the CDN's 302 to the presigned objects.githubusercontent.com URL, which
// carries its own auth — nothing of ours to leak across the hop.
var githubHTTPClient = &http.Client{Timeout: 60 * time.Second}

// releaseArtifacts are the files needed to provision and deploy a sandbox.
// CloudInit stays in memory because it is sent straight to the DO API as
// droplet user-data; only the binary is written to disk so scp can copy it.
type releaseArtifacts struct {
	Version    string // concrete tag the binary is pulled from, e.g. "v1.30.0"
	CloudInit  []byte
	BinaryPath string // temp file; set by downloadAgentBinary, caller removes when done
}

// fetchArtifacts is the seam Create fetches through. It is a variable so a
// test can exercise Create without a network call.
var fetchArtifacts = fetchReleaseArtifacts

// fetchReleaseArtifacts resolves tag to a concrete release and pairs it with the
// cloud-init template. tag may be empty/"latest" or a specific tag. The agent
// binary is downloaded later, by downloadAgentBinary, since which build to
// fetch depends on the machine's arch — not known until it is reached.
func fetchReleaseArtifacts(tag string) (*releaseArtifacts, error) {
	version, err := resolveReleaseTag(tag)
	if err != nil {
		return nil, err
	}

	return &releaseArtifacts{
		Version:   version,
		CloudInit: cloudInitTemplate,
	}, nil
}

// resolveReleaseTag turns "" / "latest" into the concrete tag of the newest
// release, and returns any other tag unchanged. "latest" is resolved through
// the CDN's redirect (github.com/<repo>/releases/latest -> .../releases/tag/<tag>),
// so recording an exact version needs no API call or token.
func resolveReleaseTag(tag string) (string, error) {
	if tag != "" && tag != "latest" {
		return tag, nil
	}

	// A dedicated client that stops at the redirect so we can read Location; the
	// shared client follows redirects, which is what downloads need.
	resolver := &http.Client{
		Timeout:       githubHTTPClient.Timeout,
		Transport:     githubHTTPClient.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	url := fmt.Sprintf("https://github.com/%s/releases/latest", releaseRepo)
	resp, err := resolver.Get(url)
	if err != nil {
		return "", fmt.Errorf("resolving latest release: %w", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("resolving latest release from %s: HTTP %d without a redirect", releaseRepo, resp.StatusCode)
	}
	return path.Base(loc), nil
}

// downloadAgentBinary fetches the agent build for arch into a temp file and
// records it in BinaryPath; the caller os.Removes it when done.
func (a *releaseArtifacts) downloadAgentBinary(arch string) error {
	name := sandboxAssetName(arch)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", releaseRepo, a.Version, name)

	binary, err := downloadReleaseAsset(url)
	if err != nil {
		// A release only carries the architectures it published; a 404 means this
		// one has no build for arch, not a transient failure.
		if err == errAssetNotFound {
			return fmt.Errorf("release %q has no asset %q", a.Version, name)
		}
		return fmt.Errorf("downloading %s: %w", name, err)
	}

	f, err := os.CreateTemp("", "spekk-sandbox-*")
	if err != nil {
		return fmt.Errorf("creating temp file for binary: %w", err)
	}
	if _, err := f.Write(binary); err != nil {
		f.Close()
		os.Remove(f.Name())
		return fmt.Errorf("writing binary: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return fmt.Errorf("closing binary: %w", err)
	}

	a.BinaryPath = f.Name()
	return nil
}

// errAssetNotFound is the 404 sentinel: the release exists but has no asset at
// the requested path (wrong arch, or a tag that was never published).
var errAssetNotFound = fmt.Errorf("release asset not found")

func downloadReleaseAsset(url string) ([]byte, error) {
	resp, err := githubHTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errAssetNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// renderCloudInit substitutes the sandbox's public key into the template's
// placeholder line.
func renderCloudInit(template []byte, sshPublicKey string) string {
	return strings.Replace(string(template), cloudInitKeyPlaceholder, sshPublicKey, 1)
}
