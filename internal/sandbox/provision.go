package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// provisionedMarker is the file a provisioned machine carries. Cloud-init
// writes it on a machine spekk created; an operator's own setup writes it on
// a machine spekk did not.
const provisionedMarker = "/opt/spekk/.provisioned"

// registerMachine records a machine the operator already has.
//
// This is the whole of "create" for an unmanaged machine: there is nothing to
// provision, because spekk does not own the box and will not rewrite its
// firewall or upgrade its packages. Everything after this point — credential
// injection, agent deploy — is the same work the cloud path does.
func registerMachine(opts CreateOptions, meta *SandboxMeta) error {
	if opts.IP == "" {
		return fmt.Errorf("an existing machine needs --ip")
	}
	if opts.SSHKey == "" {
		return fmt.Errorf("an existing machine needs --ssh-key")
	}

	// Resolve to an absolute path: every later command runs from a
	// different working directory than create did.
	key, err := filepath.Abs(opts.SSHKey)
	if err != nil {
		return fmt.Errorf("resolving --ssh-key %q: %w", opts.SSHKey, err)
	}
	// Check the key before recording anything. A key that is not there is
	// the operator's typo, and it should not cost them a metadata entry
	// they then have to force away.
	if _, err := os.Stat(key); err != nil {
		return fmt.Errorf("--ssh-key %s: %w", key, err)
	}

	meta.IP = opts.IP
	meta.SSHKeyPath = key
	return nil
}

// checkProvisioned reports whether a machine already carries the marker that
// says it has the packages, the agent user, and the directories a sandbox
// needs.
func checkProvisioned(meta *SandboxMeta, name string) error {
	fmt.Fprintln(os.Stderr, "Checking the machine is provisioned...")
	out := strings.TrimSpace(runSSH(meta.IP, meta.SSHKeyPath, name, "test -f "+provisionedMarker+" && echo ok"))
	if out == "ok" {
		return nil
	}
	// runSSH returns "" for a failed connection as well as for a missing
	// file, so name both rather than guess.
	return fmt.Errorf("could not confirm %s on %s as root: the machine is not provisioned, or the connection failed.\n"+
		"spekk does not provision a machine it did not create. Prepare it first, then re-run this command",
		provisionedMarker, meta.IP)
}

// stopAgentService stops the spekk-agent service on an unmanaged machine and
// removes the credentials spekk put there.
//
// For a machine that survives destroy this is the whole of teardown, so
// anything left behind stays live. Removing the secrets is unconditional: a
// create that died before the unit was installed still injected them, and a
// stop that fails on a unit that was never there must not be what keeps a
// GitHub token on somebody's server.
func stopAgentService(sandbox *SandboxMeta, name string) error {
	fmt.Fprintln(os.Stderr, "Stopping agent service and removing credentials...")
	args := append(sshBatchArgs(sandbox, name), teardownCommand())
	if out, err := exec.Command("ssh", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// teardownCommand is the shell run on an unmanaged machine to stop the agent
// and take spekk's credentials off it.
//
// Removing the secrets is deliberately not chained behind stopping the
// service. A create that died before the agent was installed still injected
// them, and on that machine `systemctl stop` fails on a unit that was never
// there. If the removals sat behind it, every later destroy would fail, and
// the only way out — --force — would leave a GitHub token and AWS keys on
// somebody's server for good.
func teardownCommand() string {
	return strings.Join([]string{
		"if systemctl cat spekk-agent >/dev/null 2>&1; then systemctl stop spekk-agent && systemctl disable spekk-agent; fi",
		"rm -f /etc/spekk/agent.env",
		"rm -f /home/agent/.git-credentials",
		// gh auth login writes the same token here.
		"rm -rf /home/agent/.config/gh",
		// is-active exits non-zero when the unit is stopped, which is
		// the outcome we want. Turn that into the success case.
		"! systemctl is-active --quiet spekk-agent",
	}, " && ")
}

// stopAgent is the seam Destroy stops through, so a test can drive the paths
// around it without an SSH connection.
var stopAgent = stopAgentService
