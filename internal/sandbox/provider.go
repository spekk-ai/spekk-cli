package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

// ErrNoMachineRecorded reports that a sandbox's metadata names no machine
// for the provider to tear down. Destroy refuses on this unless forced,
// because the usual cause is a record whose identifier was lost, and the
// machine is still running.
var ErrNoMachineRecorded = errors.New("no machine recorded for this sandbox")

// ErrSandboxNotFound reports that no sandbox by that name is in the store.
var ErrSandboxNotFound = errors.New("sandbox not found")

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
	// it defaulted.
	//
	// On error, fill in only what identifies a machine that really
	// exists. The caller saves meta whenever it names one, on the error
	// path as well as on success, so an address written before the
	// machine is real becomes a record pointing at nothing.
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
	name := meta.Provider
	if name == "" {
		name = "digitalocean"
	}
	return ProviderByName(name)
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
// therefore delete it. Spekk writes every key it generates into
// generatedKeysDir, so the test is whether the private key is a file in that
// directory. An operator-supplied --ssh-key is anywhere else.
//
// The recorded path and the file it resolves to must both be inside the keys
// directory. The path is cleaned and made absolute first, so a
// `keys/../../.ssh/id_rsa` does not pass on its prefix. Then symlinks are
// followed on both sides, and the resolved paths are compared: a link inside
// the keys directory that points out of it is the operator's file, and a link
// outside it that points in is a path the operator arranged, so both are kept.
// A path with nothing on disk at it passes on the cleaned path alone; there
// is nothing at it to protect, and the removal is a no-op.
//
// One known gap: a key still sitting at the pre-XDG path under ~/.spekk/keys
// answers false, so destroy leaves those two files behind. That direction is
// the safe one. Widening the test is what would risk the operator's own key,
// and a destroy that deletes that key is not recoverable.
func ownsKeyPair(path string) bool {
	if path == "" {
		return false
	}
	keys, err := filepath.Abs(generatedKeysDir())
	if err != nil {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil || !insideDir(abs, keys) {
		return false
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	resolvedKeys, err := filepath.EvalSymlinks(keys)
	if err != nil {
		return false
	}
	return insideDir(resolved, resolvedKeys)
}

// insideDir reports whether path is under dir. Both must be clean and
// absolute. The separator is part of the test so that a sibling directory
// with the same prefix, such as keys-old beside keys, does not pass.
func insideDir(path, dir string) bool {
	return strings.HasPrefix(path, dir+string(filepath.Separator))
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
		return nil, fmt.Errorf("%w: %q", ErrSandboxNotFound, name)
	}
	return ProviderFromMeta(meta)
}
