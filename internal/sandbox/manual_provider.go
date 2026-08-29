package sandbox

import (
	"fmt"
	"path/filepath"
)

// ManualProvider implements Provider for pre-existing SSH-reachable machines.
type ManualProvider struct{}

// Name reports the provider name stored in SandboxMeta.Provider.
func (p *ManualProvider) Name() string { return "manual" }

func (p *ManualProvider) Create(name string, opts CreateOptions, meta *SandboxMeta) error {
	if opts.IP == "" {
		return fmt.Errorf("manual provider requires --ip")
	}
	if opts.SSHKey == "" {
		return fmt.Errorf("manual provider requires --ssh-key")
	}
	// Store an absolute path: every later command resolves it from a
	// different working directory than the one create ran in.
	key, err := filepath.Abs(opts.SSHKey)
	if err != nil {
		return fmt.Errorf("resolving --ssh-key %q: %w", opts.SSHKey, err)
	}
	meta.IP = opts.IP
	meta.SSHKeyPath = key
	return nil
}

// Destroy has no cloud resources to tear down. The generic Destroy stops the
// agent service; the machine itself is not ours to remove.
func (p *ManualProvider) Destroy(meta *SandboxMeta) error { return nil }

// Status has no API to query, so it reports nothing and lets the caller fall
// back to the SSH checks it already runs.
func (p *ManualProvider) Status(meta *SandboxMeta) (string, error) { return "", nil }
