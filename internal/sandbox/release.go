package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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

// cliAssetName is the spekk CLI published for a given GOARCH of the machine.
func cliAssetName(arch string) string {
	return "spekk-linux-" + arch
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
// droplet user-data; only the binaries are written to disk so scp can copy
// them.
type releaseArtifacts struct {
	Version   string // concrete tag the binaries are pulled from, e.g. "v1.30.0"
	CloudInit []byte
	// BinaryPath (the agent) and CLIPath (the spekk CLI) are set by
	// downloadBinaries, in a temp directory that cleanup removes. Each
	// file has its staged name, so one scp puts both in the login home.
	BinaryPath string
	CLIPath    string
	dir        string
	// AgentSHA256 and CLISHA256 are the hex digests that the release
	// publishes for the two files. The machine checks the uploads with them.
	AgentSHA256 string
	CLISHA256   string
}

// cleanup removes the downloaded binaries.
func (a *releaseArtifacts) cleanup() {
	if a.dir != "" {
		os.RemoveAll(a.dir)
	}
}

// fetchArtifacts is the seam Create fetches through. It is a variable so a
// test can exercise Create without a network call.
var fetchArtifacts = fetchReleaseArtifacts

// fetchReleaseArtifacts resolves tag to a concrete release and pairs it with the
// cloud-init template. tag may be empty, "latest", or a specific tag. The agent
// and CLI binaries are downloaded by downloadBinaries after the machine's
// architecture is known.
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

// downloadBinaries fetches the agent and the CLI builds for arch from the
// release into a temp directory, with the sha256 that the release publishes
// for each. The caller runs cleanup when done.
func (a *releaseArtifacts) downloadBinaries(arch string) (err error) {
	digests, err := fetchDigests(a.Version)
	if err != nil {
		return err
	}
	agent, cli := sandboxAssetName(arch), cliAssetName(arch)
	for _, name := range []string{agent, cli} {
		if digests[name] == "" {
			return fmt.Errorf("release %q publishes no sha256 for %q, so spekk cannot check it; choose another release with --release", a.Version, name)
		}
	}

	dir, err := os.MkdirTemp("", "spekk-sandbox-*")
	if err != nil {
		return fmt.Errorf("creating temp dir for binaries: %w", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	for _, f := range []struct{ asset, file string }{{agent, stagedBinary}, {cli, stagedCLI}} {
		if err := a.downloadAsset(f.asset, filepath.Join(dir, f.file)); err != nil {
			return err
		}
	}

	a.dir, a.BinaryPath, a.CLIPath = dir, filepath.Join(dir, stagedBinary), filepath.Join(dir, stagedCLI)
	a.AgentSHA256, a.CLISHA256 = digests[agent], digests[cli]
	return nil
}

// downloadAsset writes the release asset name to dest.
func (a *releaseArtifacts) downloadAsset(name, dest string) error {
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
	if err := os.WriteFile(dest, binary, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", name, err)
	}
	return nil
}

// releaseAPIURL is the GitHub API path of a release by tag.
const releaseAPIURL = "https://api.github.com/repos/" + releaseRepo + "/releases/tags/"

// fetchDigests is the seam downloadBinaries reads the digests through.
var fetchDigests = releaseDigests

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

// releaseDigests returns the hex sha256 of each asset of the release tag, as
// GitHub publishes it in the asset's digest field. This is one anonymous API
// call for each deploy, well inside the 60 an hour that GitHub allows.
func releaseDigests(tag string) (map[string]string, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPIURL+url.PathEscape(tag), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reading the sha256 digests of release %q: %w", tag, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reading the sha256 digests of release %q: HTTP %d from the GitHub API", tag, resp.StatusCode)
	}
	var release struct {
		Assets []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("reading the sha256 digests of release %q: %w", tag, err)
	}
	digests := map[string]string{}
	for _, asset := range release.Assets {
		// Only a well-formed digest goes into the install script.
		if sum, ok := strings.CutPrefix(asset.Digest, "sha256:"); ok && sha256Re.MatchString(sum) {
			digests[asset.Name] = sum
		}
	}
	return digests, nil
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
