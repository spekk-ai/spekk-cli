package sandbox

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

// Provider owns the machine lifecycle for a sandbox: it creates the VM,
// tears it down, and reports its live state. Everything after the machine
// exists — waiting for provisioning, injecting credentials, deploying the
// agent — is generic and stays in this package's Create.
//
// Each method takes the sandbox's *SandboxMeta rather than an opaque handle.
// A provider reads and writes the fields it owns and ignores the rest. This
// is a deliberate choice for one cloud provider: it needs no encoding step,
// it leaves the on-disk format legible, and it cannot confuse "this sandbox
// has no machine to destroy" with "the identifier did not survive a load".
type Provider interface {
	// Name reports the provider name stored in SandboxMeta.Provider.
	Name() string

	// Create provisions a VM and fills the fields of meta that it owns:
	// IP and SSHKeyPath always, plus its own identifiers and any setting
	// it defaulted. On error meta may be partly filled; the caller saves
	// it only on success.
	Create(name string, opts CreateOptions, meta *SandboxMeta) error

	// Destroy tears down every resource this provider created for the
	// sandbox. It does not remove local files or metadata.
	Destroy(meta *SandboxMeta) error

	// Status returns the live machine state, for example "active" or "off".
	// An empty string means the provider has no live state to report.
	Status(meta *SandboxMeta) (string, error)
}

// ProviderFromMeta returns the Provider that owns a sandbox's machine.
//
// An empty Provider field means the entry was written before the field
// existed, when DigitalOcean was the only provider. Reading it as
// DigitalOcean is what lets `destroy` still delete the droplet of a sandbox
// created by an older binary.
func ProviderFromMeta(meta *SandboxMeta) (Provider, error) {
	switch meta.Provider {
	case "", "digitalocean":
		return NewDOProvider()
	default:
		return nil, fmt.Errorf("unknown provider %q", meta.Provider)
	}
}

// generatedKeysDir is the directory holding key pairs that spekk created.
func generatedKeysDir() string {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		dir = config.DefaultDir()
	}
	return filepath.Join(dir, "keys")
}

// ownsKeyPair reports whether spekk generated the key pair at path and may
// therefore delete it.
//
// Today every key is generated, so this always answers true. It exists so
// that a provider which accepts an operator-supplied key — a machine spekk
// did not create — cannot reach the unconditional os.Remove in Destroy. A
// destroy that deletes the operator's own ~/.ssh key is not recoverable, so
// the guard goes in before the caller that needs it, not after.
func ownsKeyPair(path string) bool {
	if path == "" {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	keys, err := filepath.Abs(generatedKeysDir())
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, keys+string(filepath.Separator))
}

// ProviderForName returns the Provider that owns the named sandbox.
// It reports an error the caller can downgrade to a warning when the
// command can still do useful work without a provider.
func ProviderForName(name string) (Provider, error) {
	meta, err := GetSandbox(name)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("sandbox %q not found", name)
	}
	return ProviderFromMeta(meta)
}
