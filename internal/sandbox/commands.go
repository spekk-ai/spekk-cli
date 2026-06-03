package sandbox

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
}

// --- Create ---

// Create creates a new sandbox droplet with cloud-init provisioning.
func Create(opts CreateOptions) error {
	if opts.Region == "" {
		opts.Region = "nyc1"
	}
	if opts.Size == "" {
		opts.Size = "s-2vcpu-4gb"
	}

	dropletName := "spekk-" + opts.Name

	// Check required env vars
	requiredVars := []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION", "GITHUB_TOKEN", "SPEKK_HOST"}
	var missing []string
	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	client, err := NewClient()
	if err != nil {
		return err
	}

	// Generate agent token
	agentToken := generateAgentToken()

	// Resolve project if specified
	var projectID, projectName string
	if opts.Project != "" {
		projectID, projectName, err = resolveProject(client, opts.Project)
		if err != nil {
			return err
		}
	}

	// Generate SSH key pair
	fmt.Fprintln(os.Stderr, "Generating SSH key pair...")
	keyPath, err := generateSSHKeyPair(opts.Name)
	if err != nil {
		return fmt.Errorf("generating SSH key: %w", err)
	}

	// Upload public key to DO
	pubKeyData, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return fmt.Errorf("reading public key: %w", err)
	}
	keyName := fmt.Sprintf("spekk-%s", opts.Name)
	doKey, err := client.CreateSSHKey(keyName, strings.TrimSpace(string(pubKeyData)))
	if err != nil {
		return fmt.Errorf("uploading SSH key to DO: %w", err)
	}
	fmt.Fprintf(os.Stderr, "SSH key uploaded to DigitalOcean (ID: %d)\n", doKey.ID)

	// Also fetch existing account keys so user can SSH in with their own keys
	existingKeys, listErr := client.ListSSHKeys()
	if listErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not list existing SSH keys (%s); only the generated key will be authorized\n", listErr)
	}
	sshKeyIDs := []int{doKey.ID}
	for _, k := range existingKeys {
		if k.ID != doKey.ID {
			sshKeyIDs = append(sshKeyIDs, k.ID)
		}
	}

	// Create droplet
	fmt.Fprintf(os.Stderr, "Creating droplet %q in %s (%s)...\n", dropletName, opts.Region, opts.Size)
	droplet, err := client.CreateDroplet(CreateDropletRequest{
		Name:    dropletName,
		Region:  opts.Region,
		Size:    opts.Size,
		SSHKeys: sshKeyIDs,
		VpcUUID: opts.VPC,
	})
	if err != nil {
		// Roll back the SSH key we uploaded so it doesn't leak in the DO account
		if delErr := client.DeleteSSHKey(doKey.ID); delErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not remove orphaned SSH key %d from DO: %s\n", doKey.ID, delErr)
		}
		os.Remove(keyPath)
		os.Remove(keyPath + ".pub")
		return fmt.Errorf("creating droplet: %w", err)
	}
	dropletID := droplet.ID
	fmt.Fprintf(os.Stderr, "Droplet created (ID: %d). Waiting for it to become active...\n", dropletID)

	// Wait for droplet to become active
	ip, err := waitForDroplet(client, dropletID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\nDroplet ID: %d -- not auto-destroyed, debug manually.\n", err, dropletID)
		return err
	}
	fmt.Fprintf(os.Stderr, "Droplet active at %s\n", ip)

	// Wait for SSH and provisioning
	if err := waitForProvisioning(ip, keyPath, opts.Name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\nDroplet IP: %s (ID: %d) -- not auto-destroyed, debug manually.\n", err, ip, dropletID)
		return err
	}
	fmt.Fprintln(os.Stderr, "Provisioning complete.")

	// Inject credentials
	fmt.Fprintln(os.Stderr, "Injecting credentials...")
	if err := injectCredentials(ip, keyPath, opts.Name, agentToken); err != nil {
		return fmt.Errorf("injecting credentials: %w", err)
	}

	// Configure git credentials
	fmt.Fprintln(os.Stderr, "Configuring git credentials...")
	if err := configureGitCredentials(ip, keyPath, opts.Name); err != nil {
		return fmt.Errorf("configuring git credentials: %w", err)
	}

	// Assign to project
	if projectID != "" {
		fmt.Fprintf(os.Stderr, "Assigning droplet to project %q...\n", projectName)
		urn := fmt.Sprintf("do:droplet:%d", dropletID)
		if err := client.AssignToProject(projectID, []string{urn}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not assign to project: %s\n", err)
		}
	}

	// Save metadata
	meta := &SandboxMeta{
		DropletID:  dropletID,
		IP:         ip,
		Region:     opts.Region,
		Size:       opts.Size,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Status:     "active",
		Project:    projectName,
		SSHKeyID:   doKey.ID,
		SSHKeyPath: keyPath,
	}
	if err := SaveSandbox(opts.Name, meta); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(os.Getenv("SPEKK_HOST"), "https://"), "http://"), "/")

	fmt.Fprintf(os.Stderr, `
Sandbox created successfully:
  Name:           spekk-%s
  IP:             %s
  AGENT_TOKEN:    %s

Next: Add this agent in Django admin at https://%s/staff/agent/agent/add/
  - Name: %s
  - Sandbox ID: spekk-%s
  - Auth token: %s
`, opts.Name, ip, agentToken, bareHost, opts.Name, opts.Name, agentToken)

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

// Status shows detailed info for a named sandbox.
func Status(name string) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	fmt.Printf("Sandbox: %s\n", name)
	fmt.Printf("Droplet ID: %d\n", sandbox.DropletID)
	fmt.Printf("IP: %s\n", orUnknown(sandbox.IP))
	fmt.Printf("Region: %s\n", orUnknown(sandbox.Region))
	fmt.Printf("Size: %s\n", orUnknown(sandbox.Size))
	fmt.Printf("Created: %s\n", orUnknown(sandbox.CreatedAt))

	// Fetch live status from DO API
	dropletStatus := orUnknown(sandbox.Status)
	client, err := NewClient()
	if err == nil {
		droplet, err := client.GetDroplet(sandbox.DropletID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not fetch live droplet data: %s\n", err)
		} else if droplet != nil {
			dropletStatus = droplet.Status
		}
	}
	fmt.Printf("Droplet status: %s\n", dropletStatus)

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

// Destroy tears down a sandbox droplet and removes local metadata.
func Destroy(name string, force bool) error {
	sandbox, err := GetSandbox(name)
	if err != nil {
		return err
	}
	if sandbox == nil {
		return fmt.Errorf("sandbox %q not found", name)
	}

	if !force {
		fmt.Fprintf(os.Stderr, "Destroy sandbox %q (droplet %d)? [y/N] ", name, sandbox.DropletID)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	client, err := NewClient()
	if err != nil {
		return err
	}

	if err := client.DeleteDroplet(sandbox.DropletID); err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
			fmt.Fprintf(os.Stderr, "Warning: Droplet %d was already deleted.\n", sandbox.DropletID)
		} else {
			return fmt.Errorf("deleting droplet: %w", err)
		}
	}

	// Remove SSH key from DO
	if sandbox.SSHKeyID != 0 {
		if err := client.DeleteSSHKey(sandbox.SSHKeyID); err != nil {
			if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 404 {
				fmt.Fprintf(os.Stderr, "Warning: SSH key %d was already deleted.\n", sandbox.SSHKeyID)
			} else {
				fmt.Fprintf(os.Stderr, "Warning: could not remove SSH key from DO: %s\n", err)
			}
		} else {
			fmt.Fprintln(os.Stderr, "SSH key removed from DigitalOcean.")
		}
	}

	// Remove local SSH key files
	if sandbox.SSHKeyPath != "" {
		os.Remove(sandbox.SSHKeyPath)
		os.Remove(sandbox.SSHKeyPath + ".pub")
	}

	// Remove per-sandbox known_hosts file
	os.Remove(KnownHostsFile(name))

	if err := RemoveSandbox(name); err != nil {
		return fmt.Errorf("removing metadata: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Sandbox %q destroyed.\n", name)
	return nil
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

	// Download latest release binary from GitHub
	fmt.Fprintln(os.Stderr, "Downloading latest spekk binary...")
	downloadScript := `set -e
cd /tmp
LATEST=$(curl -sL https://api.github.com/repos/spekk-ai/spekk-cli/releases/latest | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
if [ -z "$LATEST" ]; then echo "Failed to determine latest release"; exit 1; fi
echo "Downloading $LATEST..."
curl -sL "https://github.com/spekk-ai/spekk-cli/releases/download/${LATEST}/spekk-linux-amd64" -o /tmp/spekk-new
chmod +x /tmp/spekk-new
mv /tmp/spekk-new /usr/local/bin/spekk
echo "Installed spekk $LATEST"
systemctl restart spekk-agent`

	args := sshArgs(sandbox, name)
	args = append(args, downloadScript)
	cmd := exec.Command("ssh", args...)
	out, err := cmd.CombinedOutput()
	os.Stderr.Write(out)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Agent redeployed to %q.\n", name)
	return nil
}

// --- Helpers ---

// KnownHostsFile returns the path to the per-sandbox known_hosts file.
func KnownHostsFile(name string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".spekk", "known_hosts", name)
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	keysDir := filepath.Join(home, ".spekk", "keys")
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

func runSSH(ip, keyPath, name, command string) string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10")
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", ip), command)
	cmd := exec.Command("ssh", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func sshCheck(sandbox *SandboxMeta, name, command string) string {
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=5", "-o", "BatchMode=yes")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", sandbox.IP), command)
	cmd := exec.Command("ssh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func injectCredentials(ip, keyPath, name, agentToken string) error {
	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(os.Getenv("SPEKK_HOST"), "https://"), "http://"), "/")

	envLines := []string{
		"AWS_ACCESS_KEY_ID=" + os.Getenv("AWS_ACCESS_KEY_ID"),
		"AWS_SECRET_ACCESS_KEY=" + os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"AWS_DEFAULT_REGION=" + os.Getenv("AWS_DEFAULT_REGION"),
		"GITHUB_TOKEN=" + os.Getenv("GITHUB_TOKEN"),
		"SPEKK_HOST=" + bareHost,
		"SPEKK_AGENT_TOKEN=" + agentToken,
		"CLAUDE_CODE_USE_BEDROCK=1",
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-" + name,
	}

	envContent := strings.Join(envLines, "\n") + "\n"

	// Base64-encode the env content in Go to avoid any shell interpolation.
	// The remote side decodes it, so credential values with newlines, quotes,
	// or heredoc terminators cannot break out of the intended context.
	encoded := base64.StdEncoding.EncodeToString([]byte(envContent))
	script := fmt.Sprintf(
		`echo '%s' | base64 -d > /etc/spekk/agent.env && chown agent:agent /etc/spekk/agent.env && chmod 600 /etc/spekk/agent.env`,
		encoded,
	)

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
// Exported for testing.
func buildEnvContent(envVars map[string]string, name, agentToken, spekkHost string) string {
	bareHost := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(spekkHost, "https://"), "http://"), "/")

	envLines := []string{
		"AWS_ACCESS_KEY_ID=" + envVars["AWS_ACCESS_KEY_ID"],
		"AWS_SECRET_ACCESS_KEY=" + envVars["AWS_SECRET_ACCESS_KEY"],
		"AWS_DEFAULT_REGION=" + envVars["AWS_DEFAULT_REGION"],
		"GITHUB_TOKEN=" + envVars["GITHUB_TOKEN"],
		"SPEKK_HOST=" + bareHost,
		"SPEKK_AGENT_TOKEN=" + agentToken,
		"CLAUDE_CODE_USE_BEDROCK=1",
		"WORKSPACE=/opt/spekk/workspace",
		"SPEKK_AGENT_NAME=spekk-" + name,
	}
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

	// Base64-encode the token to avoid interpolating it into shell strings.
	// The remote script decodes it and writes files without shell interpretation.
	encodedToken := base64.StdEncoding.EncodeToString([]byte(ghToken))
	script := fmt.Sprintf(`set -e
TOKEN=$(echo '%s' | base64 -d)
su - agent -c 'git config --global credential.helper store'
su - agent -c "cat > ~/.git-credentials" <<< "https://x-access-token:${TOKEN}@github.com"
su - agent -c 'chmod 600 ~/.git-credentials'
echo "${TOKEN}" | su - agent -c 'gh auth login --with-token 2>/dev/null || true'`,
		encodedToken,
	)

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
