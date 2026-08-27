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
	SSHKey  string // private key that reaches that machine as root
	Auth    AuthMode

	// CloudInit is the provisioning payload from the release artifacts.
	// Create fills it in; no flag sets it. A provider that does not use
	// cloud-init ignores it.
	CloudInit []byte
}

// --- Create ---

// Create provisions a sandbox using the given Provider for VM lifecycle and
// then runs generic provisioning (SSH wait, credential injection, agent deploy).
func Create(p Provider, opts CreateOptions) error {
	// Check the credentials this sandbox's auth mode actually needs, so a
	// subscription sandbox is not blocked by AWS keys it will never use.
	requiredVars := requiredEnvVars(opts.Auth)
	var missing []string
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
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

	// A failure from here on leaves a real machine running. Say so, and
	// say which one, every time.
	fail := func(stage string, err error) error {
		fmt.Fprintf(os.Stderr, "%s -- not auto-destroyed. Debug it, or run: spekk sandbox destroy %s\n", machineRef(meta), opts.Name)
		return fmt.Errorf("%s: %w", stage, err)
	}

	if p == nil {
		// Nothing to wait for: the operator provisioned this machine.
		// Confirm that before spending credentials on it.
		if err := checkReady(meta, opts.Name); err != nil {
			return fail("checking the machine", err)
		}
	} else if err := waitReady(meta.IP, meta.SSHKeyPath, opts.Name); err != nil {
		return fail("waiting for provisioning", err)
	}
	fmt.Fprintln(os.Stderr, "Provisioning complete.")

	fmt.Fprintln(os.Stderr, "Injecting credentials...")
	if err := injectCredentials(meta.IP, meta.SSHKeyPath, opts.Name, agentToken, opts.Auth); err != nil {
		return fail("injecting credentials", err)
	}

	fmt.Fprintln(os.Stderr, "Configuring git credentials...")
	if err := configureGitCredentials(meta.IP, meta.SSHKeyPath, opts.Name); err != nil {
		return fail("configuring git credentials", err)
	}

	fmt.Fprintln(os.Stderr, "Deploying agent binary...")
	if err := deployAgent(meta.IP, meta.SSHKeyPath, opts.Name, artifacts); err != nil {
		return fail("deploying agent", err)
	}

	meta.Status = "active"
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(os.Getenv("SPEKK_HOST"), "https://"), "http://"), "/")

	fmt.Fprintf(os.Stderr, `
Sandbox created successfully:
  Name:           spekk-%s
  IP:             %s
  AGENT_TOKEN:    %s

Next: Register this agent on the control host admin at https://%s/
  - Name: %s
  - Sandbox ID: spekk-%s
  - Auth token: %s
`, opts.Name, meta.IP, agentToken, bareHost, opts.Name, opts.Name, agentToken)

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

	// Remove local SSH key files, but only the ones spekk generated. An
	// operator-supplied key belongs to the operator.
	if ownsKeyPair(sandbox.SSHKeyPath) {
		os.Remove(sandbox.SSHKeyPath)
		os.Remove(sandbox.SSHKeyPath + ".pub")
	}

	// Remove per-sandbox known_hosts file.
	os.Remove(KnownHostsFile(name))

	if err := RemoveSandbox(name); err != nil {
		return fmt.Errorf("removing metadata: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Sandbox %q destroyed.\n", name)
	return nil
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

	if err := deployAgent(sandbox.IP, sandbox.SSHKeyPath, name, artifacts); err != nil {
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
func deployAgent(ip, keyPath, name string, artifacts *releaseArtifacts) error {
	// Copy the binary up via scp.
	scp := sshHostKeyOpts(name)
	scp = append(scp, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		scp = append(scp, "-i", keyPath)
	}
	scp = append(scp, artifacts.BinaryPath, fmt.Sprintf("root@%s:/opt/spekk/agent-client", ip))
	if out, err := exec.Command("scp", scp...).CombinedOutput(); err != nil {
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

	if out, err := runSSHCombined(ip, keyPath, name, script); err != nil {
		return fmt.Errorf("installing service: %s\n%s", err, out)
	}
	return nil
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
	args = append(args, fmt.Sprintf("root@%s", sandbox.IP))
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
// can reach the steps after machine creation without a ten-minute wait.
var waitReady = waitForProvisioning

func waitForProvisioning(ip, keyPath, name string) error {
	deadline := time.Now().Add(10 * time.Minute)

	// Wait for SSH connectivity
	fmt.Fprintln(os.Stderr, "Waiting for SSH connectivity...")
	for time.Now().Before(deadline) {
		if checkTCPPort(ip, 22) {
			break
		}
		time.Sleep(5 * time.Second)
	}

	// Wait for provisioning marker
	fmt.Fprintln(os.Stderr, "Waiting for cloud-init provisioning to complete...")
	for time.Now().Before(deadline) {
		out := runSSH(ip, keyPath, name, "test -f /opt/spekk/.provisioned && echo ok")
		if strings.TrimSpace(out) == "ok" {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("provisioning did not complete within 10 minutes on %s", ip)
}

func checkTCPPort(ip string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// runSSH runs command on the host and returns trimmed stdout, or "" on error.
func runSSH(ip, keyPath, name, command string) string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", ip), command)
	out, err := exec.Command("ssh", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runSSHCombined runs command on the host, returning combined output and any error.
func runSSHCombined(ip, keyPath, name, command string) (string, error) {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", ip), command)
	out, err := exec.Command("ssh", args...).CombinedOutput()
	return string(out), err
}

// sshBatchArgs builds the ssh arguments for a non-interactive command on a
// sandbox: host-key options, a connect timeout, no prompting, the sandbox key
// if there is one, and the root account.
func sshBatchArgs(sandbox *SandboxMeta, name string) []string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10", "-o", "BatchMode=yes")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	return append(args, fmt.Sprintf("root@%s", sandbox.IP))
}

func sshCheck(sandbox *SandboxMeta, name, command string) string {
	cmd := exec.Command("ssh", append(sshBatchArgs(sandbox, name), command)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func injectCredentials(ip, keyPath, name, agentToken string, mode AuthMode) error {
	envVars := map[string]string{
		"AWS_ACCESS_KEY_ID":     os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY": os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_DEFAULT_REGION":    os.Getenv("AWS_DEFAULT_REGION"),
		"GITHUB_TOKEN":          os.Getenv("GITHUB_TOKEN"),
		oauthTokenVar:           os.Getenv(oauthTokenVar),
	}
	spekkHost := os.Getenv("SPEKK_HOST")

	envContent := buildEnvContent(mode, envVars, name, agentToken, spekkHost)
	script := buildInjectScript(envContent)

	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", ip), script)
	cmd := exec.Command("ssh", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
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

func configureGitCredentials(ip, keyPath, name string) error {
	ghToken := os.Getenv("GITHUB_TOKEN")

	script := buildGitCredentialScript(ghToken)

	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", ip), script)
	cmd := exec.Command("ssh", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
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
