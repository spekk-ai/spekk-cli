package sandbox

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	releaseRepo     = "spekk-ai/spekk-cli"
	binaryAssetName = "sandbox-linux-amd64"

	// cloudInitKeyPlaceholder is the line in the template that is replaced with
	// the sandbox's generated public key.
	cloudInitKeyPlaceholder = "ssh-ed25519 AAAA... your-key-here"
)

// githubHTTPClient is used for release downloads. Its default redirect policy
// strips the Authorization header on cross-host redirects, which is exactly
// what we need: the asset endpoint 302s to a presigned objects.githubusercontent.com
// URL that carries its own auth and rejects a forwarded token.
var githubHTTPClient = &http.Client{Timeout: 60 * time.Second}

// releaseArtifacts are the files needed to provision and deploy a sandbox.
// CloudInit stays in memory because it is sent straight to the DO API as
// droplet user-data; only the binary is written to disk so scp can copy it.
type releaseArtifacts struct {
	Version    string
	CloudInit  []byte
	BinaryPath string // temp file; caller removes when done
}

type githubAsset struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

// fetchReleaseArtifacts downloads the sandbox binary and cloud-init template
// from a spekk-app GitHub release. tag may be empty or "latest" for the latest
// published release, or a specific tag. The binary is written to a temp file
// whose path is returned in BinaryPath; callers should os.Remove it when done.
func fetchReleaseArtifacts(tag string) (*releaseArtifacts, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN is not set")
	}

	rel, err := fetchRelease(token, tag)
	if err != nil {
		return nil, err
	}

	binaryID, err := assetID(rel, binaryAssetName)
	if err != nil {
		return nil, err
	}

	binary, err := downloadAsset(token, binaryID)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", binaryAssetName, err)
	}

	f, err := os.CreateTemp("", "spekk-sandbox-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file for binary: %w", err)
	}
	if _, err := f.Write(binary); err != nil {
		f.Close()
		os.Remove(f.Name())
		return nil, fmt.Errorf("writing binary: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return nil, fmt.Errorf("closing binary: %w", err)
	}

	return &releaseArtifacts{
		Version:    rel.TagName,
		CloudInit:  cloudInitTemplate,
		BinaryPath: f.Name(),
	}, nil
}

func fetchRelease(token, tag string) (*githubRelease, error) {
	var url string
	if tag == "" || tag == "latest" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", releaseRepo)
	} else {
		url = fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", releaseRepo, tag)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		label := tag
		if label == "" {
			label = "latest"
		}
		return nil, fmt.Errorf("fetching release %q from %s: HTTP %d", label, releaseRepo, resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}
	return &rel, nil
}

func downloadAsset(token string, assetID int) ([]byte, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%d", releaseRepo, assetID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := githubHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func assetID(rel *githubRelease, name string) (int, error) {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a.ID, nil
		}
	}
	return 0, fmt.Errorf("release %q has no asset %q", rel.TagName, name)
}

// renderCloudInit substitutes the sandbox's public key into the template's
// placeholder line.
func renderCloudInit(template []byte, sshPublicKey string) string {
	return strings.Replace(string(template), cloudInitKeyPlaceholder, sshPublicKey, 1)
}
