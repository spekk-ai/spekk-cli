package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

// SandboxMeta holds local metadata for a sandbox.
//
// Provider names which implementation owns the machine's lifecycle. It is
// additive: an entry written before the field existed reads back empty, and
// ProviderFromMeta treats empty as "digitalocean", because that was the only
// provider at the time. Every other field keeps the name and meaning it has
// always had, so an old sandboxes.json stays readable and, more importantly,
// stays destroyable.
//
// DropletID and SSHKeyID are DigitalOcean's. The generic layer never reads
// them; only DOProvider does. Holding them as concrete fields rather than an
// opaque blob is deliberate while there is one cloud provider: it keeps the
// on-disk format unchanged and it keeps "no droplet" distinguishable from
// "droplet id not loaded". A second cloud provider adds its own fields.
type SandboxMeta struct {
	Provider   string `json:"provider,omitempty"`
	DropletID  int    `json:"dropletId,omitempty"`
	IP         string `json:"ip"`
	Region     string `json:"region"`
	Size       string `json:"size"`
	CreatedAt  string `json:"createdAt"`
	Status     string `json:"status"`
	Project    string `json:"project,omitempty"`
	SSHKeyID   int    `json:"sshKeyId,omitempty"`
	SSHKeyPath string `json:"sshKeyPath,omitempty"`
}

// sandboxesFile returns the path to the sandboxes metadata file.
// Overridable in tests.
var sandboxesFile = func() string {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		dir = config.DefaultDir()
	}
	return filepath.Join(dir, "sandboxes.json")
}

// LoadSandboxes reads all sandbox metadata from the config dir's sandboxes.json.
func LoadSandboxes() (map[string]*SandboxMeta, error) {
	data, err := os.ReadFile(sandboxesFile())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*SandboxMeta), nil
		}
		return nil, err
	}
	var result map[string]*SandboxMeta
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parsing sandboxes.json: %w", err)
	}
	if result == nil {
		result = make(map[string]*SandboxMeta)
	}
	for _, meta := range result {
		meta.SSHKeyPath = remapLegacyKeyPath(meta.SSHKeyPath)
	}
	return result, nil
}

// remapLegacyKeyPath rewrites an SSH key path that still points into the
// pre-XDG ~/.spekk directory after that directory has been migrated to the
// XDG config location. Returns the path unchanged unless the old path is
// gone and the same relative path exists under the new config dir.
func remapLegacyKeyPath(p string) string {
	if p == "" {
		return p
	}
	if _, err := os.Stat(p); err == nil {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	oldPrefix := filepath.Join(home, ".spekk") + string(os.PathSeparator)
	if !strings.HasPrefix(p, oldPrefix) {
		return p
	}
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return p
	}
	candidate := filepath.Join(dir, strings.TrimPrefix(p, oldPrefix))
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return p
}

// GetSandbox returns metadata for a named sandbox, or nil if not found.
func GetSandbox(name string) (*SandboxMeta, error) {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return nil, err
	}
	return sandboxes[name], nil
}

// SaveSandbox writes or updates metadata for a named sandbox.
func SaveSandbox(name string, meta *SandboxMeta) error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	sandboxes[name] = meta

	return writeSandboxes(sandboxes)
}

// RemoveSandbox deletes a sandbox from the metadata store.
func RemoveSandbox(name string) error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	delete(sandboxes, name)

	return writeSandboxes(sandboxes)
}

func writeSandboxes(sandboxes map[string]*SandboxMeta) error {
	dir := filepath.Dir(sandboxesFile())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	data, err := json.MarshalIndent(sandboxes, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(sandboxesFile(), data, 0o600)
}
