package sandbox

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spekk-ai/spekk-cli/internal/config"
)

var sandboxNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateSandboxName checks that a sandbox name contains only safe characters.
// Valid names match ^[a-z0-9][a-z0-9-]*$ (lowercase alphanumeric and hyphens,
// must start with alphanumeric). This prevents shell injection when the name
// is used in commands or environment variables.
func ValidateSandboxName(name string) error {
	if name == "" {
		return fmt.Errorf("sandbox name cannot be empty")
	}
	if !sandboxNameRe.MatchString(name) {
		return fmt.Errorf("invalid sandbox name %q: must match [a-z0-9][a-z0-9-]* (lowercase alphanumeric and hyphens, starting with alphanumeric)", name)
	}
	return nil
}

// CreateOptions holds flags for the create subcommand.
type CreateOptions struct {
	Name    string
	Region  string
	Size    string
	Project string
	VPC     string
	IP      string // address of a machine that already exists
	SSHKey  string // private key that reaches that machine as the login user
	SSHUser string // login user for an existing machine (default: root)
	Auth    AuthMode

	// ProvisionTimeout is how long Create waits for cloud-init to write the
	// provisioned marker. Zero means DefaultProvisionTimeout.
	ProvisionTimeout time.Duration

	// CloudInit is the provisioning payload from the release artifacts.
	// Create fills it in; no flag sets it. A provider that does not use
	// cloud-init ignores it.
	CloudInit []byte
}

// ProvisionOptions holds flags for the provision subcommand.
type ProvisionOptions struct {
	// Auth overrides the mode the record carries. Empty keeps the recorded
	// mode, which is bedrock for a record written before the field existed.
	Auth AuthMode
	// Force provisions a record whose status is not "provisioning".
	Force bool
}

// checkRequiredEnv reports every variable the auth mode needs that is not
// set, so a subscription sandbox is not blocked by AWS keys it will never
// use. Create and Provision both call it before touching a machine.
func checkRequiredEnv(mode AuthMode) error {
	var missing []string
	for _, v := range requiredEnvVars(mode) {
		if strings.TrimSpace(os.Getenv(v)) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// --- Create ---

// Create provisions a sandbox using the given Provider for VM lifecycle and
// then runs generic provisioning (SSH wait, credential injection, agent deploy).
func Create(p Provider, opts CreateOptions) error {
	if err := checkRequiredEnv(opts.Auth); err != nil {
		return err
	}

	// A name that is already taken is nearly always a create that failed
	// partway. Overwriting its entry would drop the identifier of the
	// machine that failure left running.
	if existing, err := GetSandbox(opts.Name); err != nil {
		return err
	} else if existing != nil {
		return fmt.Errorf("sandbox %q already exists (%s); destroy it first: spekk sandbox destroy %s",
			opts.Name, machineRef(existing), opts.Name)
	}

	agentToken := generateAgentToken()

	// Fetch release artifacts before creating billable resources.
	fmt.Fprintln(os.Stderr, "Fetching sandbox release artifacts...")
	artifacts, err := fetchArtifacts("latest")
	if err != nil {
		return fmt.Errorf("fetching release artifacts: %w", err)
	}
	defer os.Remove(artifacts.BinaryPath)
	fmt.Fprintf(os.Stderr, "Using sandbox release %s\n", artifacts.Version)

	opts.CloudInit = artifacts.CloudInit

	// Delegate machine creation to the provider, which fills in the fields
	// of meta that it owns. A nil provider means no cloud owns this
	// machine: it already exists, and spekk only registers and equips it.
	meta := &SandboxMeta{
		Provider:  ProviderNone,
		Auth:      string(opts.Auth),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if p != nil {
		meta.Provider = p.Name()
	}
	// Record the machine as soon as one exists, and record it even when
	// the provider then fails. A machine with no metadata entry is one
	// that `spekk sandbox destroy` cannot reach: the operator has to go
	// to the provider's console and match it by name.
	meta.Status = "provisioning"
	var createErr error
	if p == nil {
		createErr = registerMachine(opts, meta)
	} else {
		createErr = p.Create(opts.Name, opts, meta)
	}
	if namesMachine(meta) {
		if err := SaveSandbox(opts.Name, meta); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not record %s: %s\n", machineRef(meta), err)
			if createErr == nil {
				return fmt.Errorf("saving metadata for %s: %w", machineRef(meta), err)
			}
		}
	}
	if createErr != nil {
		if namesMachine(meta) {
			fmt.Fprintf(os.Stderr, "%s -- not auto-destroyed. Run: spekk sandbox destroy %s\n", machineRef(meta), opts.Name)
		}
		return createErr
	}

	// --- Generic provisioning (provider-agnostic) ---

	// A failure from here on leaves a real machine running, and a record
	// at "provisioning". Say so, say which machine, and say how to finish:
	// `spekk sandbox provision` picks up from here, so a slow cloud-init
	// does not cost the operator the droplet.
	fail := func(stage string, err error) error {
		fmt.Fprintf(os.Stderr, "%s -- not auto-destroyed. Finish it with: spekk sandbox provision %s\nOr remove it with: spekk sandbox destroy %s\n",
			machineRef(meta), opts.Name, opts.Name)
		return fmt.Errorf("%s: %w", stage, err)
	}

	if p == nil {
		// Nothing to wait for: the operator provisioned this machine.
		// Confirm that before spending credentials on it.
		if err := checkReady(meta, opts.Name); err != nil {
			return fail("checking the machine", err)
		}
	} else if err := waitReady(meta.IP, meta.SSHKeyPath, opts.Name, opts.ProvisionTimeout); err != nil {
		return fail("waiting for provisioning", err)
	}
	fmt.Fprintln(os.Stderr, "Provisioning complete.")

	if err := equipSandbox(meta, opts.Name, agentToken, opts.Auth, artifacts); err != nil {
		return fail(err.stage, err.err)
	}

	meta.Status = "active"
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	printRegistration("Sandbox created successfully", opts.Name, meta.IP, agentToken)
	return nil
}

// stageError is a failure from one step of equipSandbox, with the step it
// came from, so Create and Provision can say which step stopped them.
type stageError struct {
	stage string
	err   error
}

func (e *stageError) Error() string { return e.stage + ": " + e.err.Error() }
func (e *stageError) Unwrap() error { return e.err }

// equipSandbox runs the steps that turn a provisioned machine into a sandbox:
// inject the credentials, configure git for the agent, deploy the agent
// binary. Create runs it after the provisioning wait, and Provision runs it
// on a record that wait left behind, so the two cannot drift apart.
func equipSandbox(meta *SandboxMeta, name, agentToken string, mode AuthMode, artifacts *releaseArtifacts) *stageError {
	// The login user for every SSH step. root on a machine spekk created;
	// whatever the operator gave for one they already had.
	user := sshUser(meta)

	fmt.Fprintln(os.Stderr, "Injecting credentials...")
	if err := injectCredentials(meta.IP, meta.SSHKeyPath, name, user, agentToken, mode); err != nil {
		return &stageError{"injecting credentials", err}
	}

	fmt.Fprintln(os.Stderr, "Configuring git credentials...")
	if err := configureGitCredentials(meta.IP, meta.SSHKeyPath, name, user); err != nil {
		return &stageError{"configuring git credentials", err}
	}

	fmt.Fprintln(os.Stderr, "Deploying agent binary...")
	if err := deployAgent(meta.IP, meta.SSHKeyPath, name, user, artifacts); err != nil {
		return &stageError{"deploying agent", err}
	}
	return nil
}

// printRegistration prints the token the operator has to register on the
// control host. The agent cannot connect until they do, so the message is
// the same whichever command finished the sandbox.
func printRegistration(headline, name, ip, agentToken string) {
	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(os.Getenv("SPEKK_HOST"), "https://"), "http://"), "/")

	fmt.Fprintf(os.Stderr, `
%s:
  Name:           spekk-%s
  IP:             %s
  AGENT_TOKEN:    %s

Next: Register this agent on the control host admin at https://%s/
  - Name: %s
  - Sandbox ID: spekk-%s
  - Auth token: %s
`, headline, name, ip, agentToken, bareHost, name, name, agentToken)
}

// --- Provision ---

// Provision finishes a sandbox that Create left at "provisioning": the
// machine exists and cloud-init may have finished after Create stopped
// waiting, but no credentials or agent reached it. It checks the marker,
// then runs the same steps Create runs after its wait, and records the
// sandbox as active.
func Provision(name string, opts ProvisionOptions) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	// A record that is not provisioning has already been equipped, or was
	// never a machine spekk should touch. --force says the operator knows.
	if sandbox.Status != "provisioning" && !opts.Force {
		return fmt.Errorf("sandbox %q is %s, not provisioning; use --force to provision it again", name, orUnknown(sandbox.Status))
	}

	mode := opts.Auth
	if mode == "" {
		if mode, err = ParseAuthMode(sandbox.Auth); err != nil {
			return fmt.Errorf("recorded auth mode: %w", err)
		}
	}

	// Fail before touching the machine when a credential is missing, for
	// the same reason Create does: a half-written env file is worse than
	// none.
	if err := checkRequiredEnv(mode); err != nil {
		return err
	}

	if err := checkReady(sandbox, name); err != nil {
		return fmt.Errorf("checking the machine: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Fetching sandbox release artifacts...")
	artifacts, err := fetchArtifacts("latest")
	if err != nil {
		return fmt.Errorf("fetching release artifacts: %w", err)
	}
	defer os.Remove(artifacts.BinaryPath)

	agentToken := generateAgentToken()
	if err := equipSandbox(sandbox, name, agentToken, mode, artifacts); err != nil {
		return err
	}

	sandbox.Auth = string(mode)
	sandbox.Status = "active"
	if err := SaveSandbox(name, sandbox); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	printRegistration("Sandbox provisioned successfully", name, sandbox.IP, agentToken)
	return nil
}

// --- List ---

// List displays all sandboxes in a table.
func List() error {
	sandboxes, err := LoadSandboxes()
	if err != nil {
		return err
	}
	if len(sandboxes) == 0 {
		fmt.Println("No sandboxes found.")
		return nil
	}

	header := padRow("Name", "IP", "Region", "Status", "Created")
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))

	for name, data := range sandboxes {
		fmt.Println(padRow(
			name,
			orDash(data.IP),
			orDash(data.Region),
			orDash(data.Status),
			orDash(data.CreatedAt),
		))
	}
	return nil
}

// --- Status ---

// Status shows detailed info for a named sandbox, using the Provider
// to fetch live VM state.
func Status(p Provider, name string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	fmt.Printf("Sandbox: %s\n", name)
	fmt.Printf("Provider: %s\n", orUnknown(providerName(sandbox)))
	if sandbox.DropletID != 0 {
		fmt.Printf("Droplet ID: %d\n", sandbox.DropletID)
	}
	fmt.Printf("IP: %s\n", orUnknown(sandbox.IP))
	fmt.Printf("Region: %s\n", orUnknown(sandbox.Region))
	fmt.Printf("Size: %s\n", orUnknown(sandbox.Size))
	fmt.Printf("Created: %s\n", orUnknown(sandbox.CreatedAt))

	// Fetch live status from the provider. A nil provider has two causes:
	// no cloud owns this machine, which is normal and silent, or one could
	// not be built — most often no API token — which is worth saying. Both
	// fall back to the stored value rather than refusing to print.
	vmStatus := orUnknown(sandbox.Status) + " (stored)"
	if p == nil {
		if sandbox.Provider != ProviderNone {
			fmt.Fprintln(os.Stderr, "Warning: no provider available; showing the stored status.")
		}
	} else if live, err := p.Status(sandbox); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch live VM status: %s\n", err)
	} else if live != "" {
		vmStatus = live
	}
	fmt.Printf("VM status: %s\n", vmStatus)

	// SSH checks
	provisioned := sshCheck(sandbox, name, "test -f /opt/spekk/.provisioned && echo yes || echo no")
	fmt.Printf("Provisioned: %s\n", provisioned)

	agentStatus := sshCheck(sandbox, name, "systemctl is-active spekk-agent 2>/dev/null || echo unknown")
	fmt.Printf("Agent service: %s\n", agentStatus)

	return nil
}

// --- SSH ---

// SSH opens an interactive SSH session to a sandbox.
func SSH(name string, extraArgs []string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	args := sshArgs(sandbox, name)
	args = append(args, extraArgs...)
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("SSH failed: %w", err)
	}
	return nil
}

// --- Destroy ---

// Destroy tears down a sandbox using the Provider for remote resource cleanup
// and then removes local files and metadata.
func Destroy(p Provider, name string, force bool) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	if !force {
		fmt.Fprintf(os.Stderr, "Destroy sandbox %q (%s)? [y/N] ", name, machineRef(sandbox))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// A manual machine survives destroy, so stopping the agent and
	// removing its credentials is the whole of teardown. Failing to do it
	// must stop the command: deleting the local record would leave an
	// agent running with live credentials and nothing pointing at it.
	// A cloud machine is about to be deleted whole, so this does not apply.
	if sandbox.Provider == ProviderNone {
		if err := stopAgent(sandbox, name); err != nil {
			if !force {
				return fmt.Errorf("stopping the agent on %s: %w\nThe machine may still be running it. Use --force to remove the record anyway", machineRef(sandbox), err)
			}
			fmt.Fprintf(os.Stderr, "Warning: could not stop the agent on %s: %s; removing the local record anyway because --force was given.\n", machineRef(sandbox), err)
		}
	}

	// Delegate remote resource cleanup to the provider. This is not
	// optional: if it fails, stop, because removing the metadata below is
	// what makes an orphaned machine unfindable. A nil provider means no
	// cloud owns the machine, so there is nothing of its to tear down.
	if err := destroyMachine(p, sandbox); err != nil {
		// A record that names no machine is normally a lost identifier,
		// and removing it would hide a running machine. With --force the
		// operator has said they checked; let them clear the entry.
		if !errors.Is(err, ErrNoMachineRecorded) || !force {
			return err
		}
		fmt.Fprintf(os.Stderr, "Warning: %s; removing the local record anyway because --force was given.\n", err)
	}

	removeGeneratedKeyPair(sandbox.SSHKeyPath)

	// Remove per-sandbox known_hosts file.
	os.Remove(KnownHostsFile(name))

	if err := RemoveSandbox(name); err != nil {
		return fmt.Errorf("removing metadata: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Sandbox %q destroyed.\n", name)
	return nil
}

// removeGeneratedKeyPair deletes the local key pair at path, but only when
// spekk generated it; ownsKeyPair holds the rule. An operator-supplied key
// belongs to the operator, and saying that it stayed is what tells them the
// destroy did not take it.
func removeGeneratedKeyPair(path string) {
	if path == "" {
		return
	}
	if !ownsKeyPair(path) {
		fmt.Fprintf(os.Stderr, "Kept SSH key %s: spekk did not generate it.\n", path)
		return
	}
	os.Remove(path)
	os.Remove(path + ".pub")
}

// providerName reports the provider that owns a sandbox, reading an entry
// written before the field existed as DigitalOcean.
func providerName(meta *SandboxMeta) string {
	if meta.Provider == "" {
		return "digitalocean"
	}
	return meta.Provider
}

// destroyMachine tears down the provider's resources, if a provider owns any.
func destroyMachine(p Provider, meta *SandboxMeta) error {
	if p == nil {
		return nil
	}
	return p.Destroy(meta)
}

// namesMachine reports whether meta identifies a machine that exists.
func namesMachine(meta *SandboxMeta) bool {
	return meta.DropletID != 0 || meta.IP != ""
}

// machineRef describes the machine a command is about to act on, so the
// operator can check it against the provider's console before saying yes.
func machineRef(meta *SandboxMeta) string {
	ref := "IP " + orUnknown(meta.IP)
	if meta.DropletID != 0 {
		ref += fmt.Sprintf(", droplet %d", meta.DropletID)
	}
	return ref
}

// --- Deploy ---

// Deploy downloads and deploys the agent binary to a sandbox.
func Deploy(name string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	fmt.Fprintf(os.Stderr, "Deploying agent to %s...\n", sandbox.IP)
	fmt.Fprintln(os.Stderr, "Fetching sandbox release artifacts...")
	artifacts, err := fetchReleaseArtifacts("latest")
	if err != nil {
		return fmt.Errorf("fetching release artifacts: %w", err)
	}
	defer os.Remove(artifacts.BinaryPath)

	if err := deployAgent(sandbox.IP, sandbox.SSHKeyPath, name, sshUser(sandbox), artifacts); err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Agent %s deployed to %q.\n", artifacts.Version, name)
	return nil
}

// spekkAgentUnit is the systemd unit installed on each sandbox. It is the single
// source of truth for the service definition (matches the release cloud-init).
const spekkAgentUnit = `[Unit]
Description=Spekk Agent Client
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
User=agent
WorkingDirectory=/opt/spekk
EnvironmentFile=/etc/spekk/agent.env
ExecStart=/opt/spekk/agent-client
Restart=always
RestartSec=5
StandardOutput=append:/var/log/spekk/agent.log
StandardError=append:/var/log/spekk/agent.log

[Install]
WantedBy=multi-user.target
`

// deployAgent copies the agent binary to /opt/spekk/agent-client, installs the
// systemd unit, and (re)starts the service. Shared by Create and Deploy.
func deployAgent(ip, keyPath, name, user string, artifacts *releaseArtifacts) error {
	// Copy the binary up via scp. A non-root user cannot write /opt/spekk,
	// so it stages in that user's home directory and the install script
	// moves it up under sudo. Not /tmp: a fixed name in a directory every
	// local user can write is a file another user can put there first, and
	// what root then moves into place is what systemd runs.
	scp := sshHostKeyOpts(name)
	scp = append(scp, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		scp = append(scp, "-i", keyPath)
	}
	scp = append(scp, artifacts.BinaryPath, scpTarget(user, ip))
	if out, err := scpExec(scp); err != nil {
		return fmt.Errorf("copying binary: %s\n%s", err, string(out))
	}

	// Install the unit, fix ownership, and start the service. The agent needs to
	// own /opt/spekk so it can create its WORKSPACE (/opt/spekk/workspace) at runtime.
	script := fmt.Sprintf(`set -e
chmod +x /opt/spekk/agent-client
mkdir -p /opt/spekk/workspace
chown -R agent:agent /opt/spekk
mkdir -p /var/log/spekk
chown agent:agent /var/log/spekk
cat > /etc/systemd/system/spekk-agent.service << 'UNITEOF'
%s
UNITEOF
systemctl daemon-reload
systemctl enable spekk-agent
systemctl restart spekk-agent`, spekkAgentUnit)

	if out, err := runSSHCombined(ip, keyPath, name, user, installCommand(user, script)); err != nil {
		return fmt.Errorf("installing service: %s\n%s", err, out)
	}
	return nil
}

// stagedBinary is where a non-root deploy puts the agent binary before root
// moves it into /opt/spekk. scp resolves a relative path against the login
// user's home directory, which only that user and root can write.
const stagedBinary = "agent-client.staged"

// sshExec runs an ssh command. It is a variable so a test can read what the
// privileged steps actually send, with no machine to send it to.
var sshExec = func(args []string) ([]byte, error) {
	return exec.Command("ssh", args...).CombinedOutput()
}

// scpExec copies a file with scp. It is a seam for the same reason sshExec
// is: the deploy decides where a non-root login stages the agent binary, and
// that decision is worth a test.
var scpExec = func(args []string) ([]byte, error) {
	return exec.Command("scp", args...).CombinedOutput()
}

// privilegedScript returns the remote command that runs script as root. A
// root login runs it as it is; any other login user escalates. Every
// privileged step goes through here, so the rule lives in one place rather
// than in a repeated conditional at each call site.
func privilegedScript(user, script string) string {
	if user == "root" {
		return script
	}
	return sudoWrap(script)
}

// scpTarget is where deployAgent copies the agent binary. root writes
// /opt/spekk directly; anybody else stages in their own home directory.
func scpTarget(user, ip string) string {
	if user == "root" {
		return fmt.Sprintf("root@%s:/opt/spekk/agent-client", ip)
	}
	return fmt.Sprintf("%s@%s:%s", user, ip, stagedBinary)
}

// installCommand is the remote command that installs and starts the agent.
// A non-root deploy first moves the staged binary into place, because only
// root can write /opt/spekk.
func installCommand(user, script string) string {
	if user == "root" {
		return script
	}
	return fmt.Sprintf(`sudo mv "$HOME/%s" /opt/spekk/agent-client && `, stagedBinary) + privilegedScript(user, script)
}

// sudoWrap base64-encodes a script and pipes it through `sudo bash`, running
// it as root without the quoting problems heredocs and nested quotes pose.
// The login user must have passwordless sudo.
func sudoWrap(script string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(script))
	return fmt.Sprintf("echo '%s' | base64 -d | sudo bash", encoded)
}

// sshUser returns the login user for a sandbox, defaulting to root.
func sshUser(meta *SandboxMeta) string {
	if meta.SSHUser != "" {
		return meta.SSHUser
	}
	return "root"
}

// --- Helpers ---

// KnownHostsFile returns the path to the per-sandbox known_hosts file.
func KnownHostsFile(name string) string {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		dir = config.DefaultDir()
	}
	return filepath.Join(dir, "known_hosts", name)
}

// sshHostKeyOpts returns SSH options for host key verification.
// On first connection (no known_hosts file), it uses accept-new to record the key.
// On subsequent connections, it uses strict verification against the stored key.
func sshHostKeyOpts(name string) []string {
	khFile := KnownHostsFile(name)
	if _, err := os.Stat(khFile); err == nil {
		// Known hosts file exists — verify host key strictly
		return []string{
			"-o", "StrictHostKeyChecking=yes",
			"-o", fmt.Sprintf("UserKnownHostsFile=%s", khFile),
		}
	}
	// First connection — accept and record the key
	os.MkdirAll(filepath.Dir(khFile), 0o700)
	return []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", fmt.Sprintf("UserKnownHostsFile=%s", khFile),
	}
}

// sshArgs returns the base SSH arguments for connecting to a sandbox,
// using the stored key if available.
func sshArgs(sandbox *SandboxMeta, name string) []string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", sshUser(sandbox), sandbox.IP))
	return args
}

// generateSSHKeyPair creates an ed25519 SSH key pair for a sandbox.
// Returns the path to the private key.
func generateSSHKeyPair(name string) (string, error) {
	dir, err := config.GlobalConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	keysDir := filepath.Join(dir, "keys")
	if err := os.MkdirAll(keysDir, 0o700); err != nil {
		return "", fmt.Errorf("creating keys dir: %w", err)
	}
	keyPath := filepath.Join(keysDir, name)

	// Remove existing key files if any
	os.Remove(keyPath)
	os.Remove(keyPath + ".pub")

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-C", fmt.Sprintf("spekk-%s", name))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	// Restrict permissions on private key
	if err := os.Chmod(keyPath, 0o600); err != nil {
		return "", fmt.Errorf("setting key permissions: %w", err)
	}

	return keyPath, nil
}

func waitForDroplet(client *Client, dropletID int) (string, error) {
	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		droplet, err := client.GetDroplet(dropletID)
		if err != nil {
			return "", err
		}
		if droplet.Status == "active" {
			ip := droplet.PublicIP()
			if ip != "" {
				return ip, nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return "", fmt.Errorf("droplet %d did not become active within 5 minutes", dropletID)
}

// waitReady is the seam Create waits through. It is a variable so a test
// can reach the steps after machine creation without a long wait.
var waitReady = waitForProvisioning

// DefaultProvisionTimeout is how long Create waits for cloud-init when the
// operator sets no --provision-timeout. Ten minutes was too short: an apt
// upgrade plus the nodesource repository has taken eighteen on a slow night,
// and the droplet then finished provisioning after Create had given up on it.
const DefaultProvisionTimeout = 30 * time.Minute

// ParseProvisionTimeout reads the --provision-timeout flag. An empty value
// is the default; anything else must be a positive Go duration such as 45m.
func ParseProvisionTimeout(s string) (time.Duration, error) {
	if strings.TrimSpace(s) == "" {
		return DefaultProvisionTimeout, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --provision-timeout %q: use a duration such as 30m or 1h", s)
	}
	if d <= 0 {
		return 0, fmt.Errorf("invalid --provision-timeout %q: must be longer than zero", s)
	}
	return d, nil
}

// provisionPollInterval is how often the wait asks the machine whether the
// marker is there. provisionReportInterval is how often it tells the
// operator what it saw, so a long wait reads as alive rather than hung.
// Both are variables so a test can run the loop without the real delays.
var (
	provisionPollInterval   = 5 * time.Second
	provisionReportInterval = time.Minute
)

// cloudInitLog is where cloud-init writes the output of every runcmd step,
// so its last line says what provisioning is doing right now.
const cloudInitLog = "/var/log/cloud-init-output.log"

// provisionProbeScript is one SSH round trip that reports everything the
// wait wants to know: the marker, what cloud-init says about itself, and the
// last line of its log. The log line goes through printf, quoted, so a line
// with a glob character or a run of spaces arrives as it was written.
const provisionProbeScript = `marker=no; test -f ` + provisionedMarker + ` && marker=yes
status=$(cloud-init status 2>/dev/null | sed -n 's/^status: //p' | head -n 1)
log=$(tail -n 1 ` + cloudInitLog + ` 2>/dev/null)
printf 'marker=%s\nstatus=%s\nlog=%s\n' "$marker" "$status" "$log"`

// provisionProbe is what one poll of a provisioning machine reports.
type provisionProbe struct {
	// marker is true once /opt/spekk/.provisioned exists.
	marker bool
	// cloudInit is what `cloud-init status` reports: running, done, error,
	// or "" when the machine could not be asked.
	cloudInit string
	// lastLog is the last line of cloud-init's output log.
	lastLog string
}

// parseProvisionProbe reads the output of provisionProbeScript. Output that
// is not in that form, including the empty string a failed connection
// returns, reads as "nothing known yet".
func parseProvisionProbe(out string) provisionProbe {
	var p provisionProbe
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "marker":
			p.marker = value == "yes"
		case "status":
			p.cloudInit = strings.TrimSpace(value)
		case "log":
			p.lastLog = strings.TrimSpace(value)
		}
	}
	return p
}

// probeProvisioning is the seam the wait polls through, so a test can drive
// the loop without a machine.
var probeProvisioning = func(ip, keyPath, name string) provisionProbe {
	return parseProvisionProbe(runSSH(ip, keyPath, name, "root", provisionProbeScript))
}

// provisionStopped reports why the marker will not appear, or nil while it
// still may. The marker is the last runcmd step, so cloud-init saying "done"
// without it means a step before it failed silently; "error" says so
// outright. Either way, waiting out the clock would tell the operator
// nothing more.
func provisionStopped(p provisionProbe) error {
	if p.marker {
		return nil
	}
	switch p.cloudInit {
	case "error":
		return fmt.Errorf("cloud-init reported an error before writing %s", provisionedMarker)
	case "done":
		return fmt.Errorf("cloud-init finished without writing %s", provisionedMarker)
	}
	return nil
}

// lastLogOrPlaceholder is the last cloud-init log line as a progress note.
func lastLogOrPlaceholder(p provisionProbe) string {
	if p.lastLog == "" {
		return "(no cloud-init output yet)"
	}
	return p.lastLog
}

func waitForProvisioning(ip, keyPath, name string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = DefaultProvisionTimeout
	}
	deadline := time.Now().Add(timeout)

	// Wait for SSH connectivity
	fmt.Fprintln(os.Stderr, "Waiting for SSH connectivity...")
	for time.Now().Before(deadline) {
		if checkTCPPort(ip, 22) {
			break
		}
		time.Sleep(provisionPollInterval)
	}

	fmt.Fprintf(os.Stderr, "Waiting up to %s for cloud-init provisioning to complete...\n", timeout)
	return waitForMarker(deadline, func() provisionProbe { return probeProvisioning(ip, keyPath, name) })
}

// waitForMarker polls probe until the marker appears, cloud-init says it
// will not, or the deadline passes. At most once a report interval it prints
// how long it has waited and what cloud-init last logged.
func waitForMarker(deadline time.Time, probe func() provisionProbe) error {
	start := time.Now()
	lastReport := start
	var last provisionProbe
	for time.Now().Before(deadline) {
		last = probe()
		if last.marker {
			return nil
		}
		if err := provisionStopped(last); err != nil {
			return fmt.Errorf("%w\nLast log line: %s", err, lastLogOrPlaceholder(last))
		}
		if time.Since(lastReport) >= provisionReportInterval {
			lastReport = time.Now()
			fmt.Fprintf(os.Stderr, "Still provisioning after %s. cloud-init: %s\n",
				time.Since(start).Round(time.Second), lastLogOrPlaceholder(last))
		}
		time.Sleep(provisionPollInterval)
	}
	return fmt.Errorf("provisioning did not complete within %s; cloud-init may still be running (last log line: %s)",
		time.Since(start).Round(time.Second), lastLogOrPlaceholder(last))
}

func checkTCPPort(ip string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// runSSH runs command on the host as user and returns trimmed stdout, or ""
// on error.
func runSSH(ip, keyPath, name, user, command string) string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)
	out, err := exec.Command("ssh", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runSSHCombined runs command on the host as user, returning combined output
// and any error.
func runSSHCombined(ip, keyPath, name, user, command string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), command)
	out, err := sshExec(args)
	return string(out), err
}

// sshBatchArgs builds the ssh arguments for a non-interactive command on a
// sandbox: host-key options, a connect timeout, no prompting, the sandbox key
// if there is one, and the login user.
func sshBatchArgs(sandbox *SandboxMeta, name string) []string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10", "-o", "BatchMode=yes")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	return append(args, fmt.Sprintf("%s@%s", sshUser(sandbox), sandbox.IP))
}

func sshCheck(sandbox *SandboxMeta, name, command string) string {
	cmd := exec.Command("ssh", append(sshBatchArgs(sandbox, name), command)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func injectCredentials(ip, keyPath, name, user, agentToken string, mode AuthMode) error {
	envVars := map[string]string{
		"AWS_ACCESS_KEY_ID":     os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY": os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_DEFAULT_REGION":    os.Getenv("AWS_DEFAULT_REGION"),
		"GITHUB_TOKEN":          os.Getenv("GITHUB_TOKEN"),
		oauthTokenVar:           os.Getenv(oauthTokenVar),
	}
	spekkHost := os.Getenv("SPEKK_HOST")

	envContent := buildEnvContent(mode, envVars, name, agentToken, spekkHost)
	script := privilegedScript(user, buildInjectScript(envContent))

	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), script)
	if out, err := sshExec(args); err != nil {
		return fmt.Errorf("SSH command failed: %s\n%s", err, string(out))
	}
	return nil
}

// buildEnvContent constructs the env file content for credential injection.
// The model credential comes from the auth mode; everything after it is the
// same whatever the sandbox pays with. Exported for testing.
func buildEnvContent(mode AuthMode, envVars map[string]string, name, agentToken, spekkHost string) string {
	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(spekkHost, "https://"), "http://"), "/")

	envLines := append(authLines(mode, envVars),
		"GITHUB_TOKEN="+envVars["GITHUB_TOKEN"],
		"SPEKK_HOST="+bareHost,
		"SPEKK_AGENT_TOKEN="+agentToken,
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-"+name,
	)
	return strings.Join(envLines, "\n") + "\n"
}

// buildInjectScript produces the SSH command string for injecting credentials.
// It base64-encodes the content to prevent shell injection. Exported for testing.
func buildInjectScript(envContent string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(envContent))
	return fmt.Sprintf(
		`echo '%s' | base64 -d > /etc/spekk/agent.env && chown agent:agent /etc/spekk/agent.env && chmod 600 /etc/spekk/agent.env`,
		encoded,
	)
}

func configureGitCredentials(ip, keyPath, name, user string) error {
	ghToken := os.Getenv("GITHUB_TOKEN")

	script := privilegedScript(user, buildGitCredentialScript(ghToken))

	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("%s@%s", user, ip), script)
	if out, err := sshExec(args); err != nil {
		return fmt.Errorf("SSH command failed: %s\n%s", err, string(out))
	}
	return nil
}

// buildGitCredentialScript produces the SSH command string for configuring git credentials.
// It base64-encodes the token to prevent shell injection. Exported for testing.
func buildGitCredentialScript(ghToken string) string {
	encodedToken := base64.StdEncoding.EncodeToString([]byte(ghToken))
	return fmt.Sprintf(`set -e
TOKEN=$(echo '%s' | base64 -d)
su - agent -c 'git config --global credential.helper store'
su - agent -c "cat > ~/.git-credentials" <<< "https://x-access-token:${TOKEN}@github.com"
su - agent -c 'chmod 600 ~/.git-credentials'
echo "${TOKEN}" | su - agent -c 'gh auth login --with-token 2>/dev/null || true'`,
		encodedToken,
	)
}

func resolveProject(client *Client, projectValue string) (string, string, error) {
	// If it looks like a UUID, use it directly
	if len(projectValue) == 36 && strings.Count(projectValue, "-") == 4 {
		return projectValue, projectValue, nil
	}

	projects, err := client.ListProjects()
	if err != nil {
		return "", "", fmt.Errorf("listing projects: %w", err)
	}
	for _, p := range projects {
		if p.Name == projectValue {
			return p.ID, p.Name, nil
		}
	}

	var names []string
	for _, p := range projects {
		names = append(names, "  - "+p.Name)
	}
	return "", "", fmt.Errorf("no project found with name %q\nAvailable projects:\n%s", projectValue, strings.Join(names, "\n"))
}

func generateAgentToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func padRow(name, ip, region, status, created string) string {
	return fmt.Sprintf("%-20s  %-18s  %-10s  %-12s  %s", name, ip, region, status, created)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func orUnknown(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
