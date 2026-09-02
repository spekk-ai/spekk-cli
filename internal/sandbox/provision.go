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

// checkReady is the seam Create checks through, so a test can drive the
// paths around it without an SSH connection.
var checkReady = checkProvisioned

// checkProvisioned reports whether a machine already carries the marker that
// says it has the packages, the agent user, and the directories a sandbox
// needs.
func checkProvisioned(meta *SandboxMeta, name string) error {
	fmt.Fprintln(os.Stderr, "Checking the machine is provisioned...")
	user := sshUser(meta)
	out := strings.TrimSpace(runSSH(meta.IP, meta.SSHKeyPath, name, user, "test -f "+provisionedMarker+" && echo ok"))
	if out == "ok" {
		return nil
	}
	// runSSH returns "" for a failed connection as well as for a missing
	// file, so name both rather than guess.
	return fmt.Errorf("could not confirm %s on %s as %s: the machine is not provisioned, or the connection failed.\n"+
		"spekk does not provision a machine it did not create. Prepare it first, then re-run this command",
		provisionedMarker, meta.IP, user)
}

// agentSecrets are the files spekk puts on a sandbox. On a machine that
// survives destroy, anything left here stays live.
var agentSecrets = []string{
	"/etc/spekk/agent.env",
	"/home/agent/.git-credentials",
	// gh auth login writes the same token here.
	"/home/agent/.config/gh",
	// An older setup-credentials.sh exported ANTHROPIC_API_KEY into the
	// agent's login shell. Nothing writes it now, but a machine
	// credentialed before then still carries it.
	"/home/agent/.bashrc.d/spekk.sh",
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
	// Teardown touches root-owned units and files, so a non-root user
	// runs it under sudo.
	command := teardownCommand()
	if sshUser(sandbox) != "root" {
		command = sudoWrap(command)
	}
	args := append(sshBatchArgs(sandbox, name), command)
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
	// Every removal is attempted, and the command succeeds only if the
	// agent is stopped and none of the secrets is still there.
	//
	// "&&" would stop at the first failure, so a machine whose unit was
	// never installed would keep its credentials. ";" alone would run
	// them all but report only the last one, so a read-only /etc would
	// remove nothing and still report success. So the status is
	// accumulated, and then the outcome is checked rather than trusted:
	// what matters is that the file is gone, not that rm said so.
	steps := []string{
		"rc=0",
		// Best effort: the unit may never have been installed, and the
		// check below is what decides whether the agent stopped.
		"systemctl stop spekk-agent 2>/dev/null || true",
		"systemctl disable spekk-agent 2>/dev/null || true",
	}
	for _, path := range agentSecrets {
		steps = append(steps, fmt.Sprintf("rm -rf %s || rc=1", path))
	}
	for _, path := range agentSecrets {
		steps = append(steps, fmt.Sprintf("test -e %s && rc=1", path))
	}
	steps = append(steps,
		// is-active exits 0 while the agent runs, which is a failure.
		"systemctl is-active --quiet spekk-agent && rc=1",
		"exit $rc",
	)
	return strings.Join(steps, "; ")
}

// stopAgent is the seam Destroy stops through, so a test can drive the paths
// around it without an SSH connection.
var stopAgent = stopAgentService
