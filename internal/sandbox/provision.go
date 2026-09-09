package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	if err := validateSSHUser(opts.SSHUser); err != nil {
		return err
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
	meta.SSHUser = opts.SSHUser
	return nil
}

var sshUserRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]*$`)

// validateSSHUser checks the login user before it reaches an ssh argument.
// A value that starts with "-" is read by ssh as an option rather than as
// part of the destination, so "-oProxyCommand=..." would run a command on
// the operator's own machine, and it would run again on every later status,
// ssh, deploy and destroy, because the value is stored. An empty user is
// valid and means root.
func validateSSHUser(user string) error {
	if user == "" {
		return nil
	}
	if !sshUserRe.MatchString(user) {
		return fmt.Errorf("invalid --ssh-user %q: must match [a-zA-Z0-9_][a-zA-Z0-9._-]* (a login name)", user)
	}
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
	// file, so name both rather than guess. What to do about it depends on
	// who provisions the machine: the operator, for one they brought; cloud-
	// init, for one spekk created, which may still be running.
	advice := "spekk does not provision a machine it did not create. Prepare it first, then re-run this command"
	if meta.Provider != ProviderNone {
		advice = fmt.Sprintf("cloud-init may still be running. Watch it with: spekk sandbox ssh %s tail -f %s\n"+
			"Then re-run this command", name, cloudInitLog)
	}
	return fmt.Errorf("could not confirm %s on %s as %s: the machine is not provisioned, or the connection failed.\n%s",
		provisionedMarker, meta.IP, user, advice)
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
	args := append(sshBatchArgs(sandbox, name), privilegedScript(sshUser(sandbox), teardownCommand()))
	if out, err := sshExec(args); err != nil {
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
