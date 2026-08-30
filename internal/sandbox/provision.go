package sandbox

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// provisionScript generates an idempotent shell script that replicates
// the cloud-init provisioning. It creates the agent user, installs all
// required packages, configures the firewall, and marks the machine as
// provisioned. Running it on an already-provisioned machine is safe.
//
// sshPubKey is the SSH public key to authorize for the agent user.
func provisionScript(sshPubKey string) string {
	// The key is carried base64-encoded. Go's %q is not bash quoting: it
	// emits a double-quoted string, and bash expands $(...) inside those,
	// so an SSH key whose free-form comment held a command substitution
	// would run it as root. Base64 has no character bash acts on. This is
	// the same defense buildInjectScript already uses.
	encoded := base64.StdEncoding.EncodeToString([]byte(sshPubKey))
	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

echo "=== Spekk sandbox provisioning ==="

# --- Agent user (idempotent) ---
if ! id -u agent >/dev/null 2>&1; then
  useradd -m -s /bin/bash agent
  echo "agent ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/agent
  chmod 440 /etc/sudoers.d/agent
  echo "Created agent user"
else
  echo "Agent user already exists"
fi

# Add agent to groups (idempotent — usermod -aG is a no-op if already a member).
# Docker group may not exist yet; create it if needed so usermod succeeds.
getent group docker >/dev/null 2>&1 || groupadd docker
usermod -aG docker,sudo,systemd-journal agent

# Authorize SSH key for agent user.
AGENT_HOME=$(eval echo ~agent)
mkdir -p "${AGENT_HOME}/.ssh"
AUTHORIZED_KEYS="${AGENT_HOME}/.ssh/authorized_keys"
PUB_KEY=$(printf %%s %s | base64 -d)
if ! grep -qF "${PUB_KEY}" "${AUTHORIZED_KEYS}" 2>/dev/null; then
  echo "${PUB_KEY}" >> "${AUTHORIZED_KEYS}"
  echo "Added SSH key for agent user"
fi
chmod 700 "${AGENT_HOME}/.ssh"
chmod 600 "${AUTHORIZED_KEYS}"
chown -R agent:agent "${AGENT_HOME}/.ssh"

# --- Packages ---
echo "Updating packages..."
apt-get update -y
apt-get upgrade -y
apt-get install -y git jq htop tmux vim unzip ca-certificates curl gnupg ufw fail2ban

# --- Config files ---
mkdir -p /etc/spekk /opt/spekk /var/log/spekk /workspace
chown agent:agent /var/log/spekk /workspace

cat > /etc/fail2ban/jail.d/sandbox-sshd.local << 'JAILEOF'
[sshd]
enabled = true
maxretry = 5
bantime = 1h
findtime = 10m
JAILEOF

# --- Docker (official repo, idempotent) ---
if ! command -v docker >/dev/null 2>&1; then
  echo "Installing Docker..."
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
else
  echo "Docker already installed"
fi

# --- Firewall (UFW) ---
echo "Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp

# UFW/Docker integration (idempotent — check before appending).
if ! grep -q "BEGIN UFW AND DOCKER" /etc/ufw/after.rules 2>/dev/null; then
cat >> /etc/ufw/after.rules <<'UFW_DOCKER'

# BEGIN UFW AND DOCKER
*filter
:ufw-user-forward - [0:0]
:ufw-docker-logging-deny - [0:0]
:DOCKER-USER - [0:0]
-A DOCKER-USER -j ufw-user-forward
-A DOCKER-USER -j RETURN -s 10.0.0.0/8
-A DOCKER-USER -j RETURN -s 172.16.0.0/12
-A DOCKER-USER -j RETURN -s 192.168.0.0/16
-A DOCKER-USER -p udp -m udp --sport 53 --dport 1024:65535 -j RETURN
-A DOCKER-USER -j ufw-docker-logging-deny -p tcp -m tcp --tcp-flags FIN,SYN,RST,ACK SYN -d 192.168.0.0/16
-A DOCKER-USER -j ufw-docker-logging-deny -p tcp -m tcp --tcp-flags FIN,SYN,RST,ACK SYN -d 10.0.0.0/8
-A DOCKER-USER -j ufw-docker-logging-deny -p tcp -m tcp --tcp-flags FIN,SYN,RST,ACK SYN -d 172.16.0.0/12
-A DOCKER-USER -j ufw-docker-logging-deny -p udp -m udp --dport 0:32767 -d 192.168.0.0/16
-A DOCKER-USER -j ufw-docker-logging-deny -p udp -m udp --dport 0:32767 -d 10.0.0.0/8
-A DOCKER-USER -j ufw-docker-logging-deny -p udp -m udp --dport 0:32767 -d 172.16.0.0/12
-A DOCKER-USER -j RETURN
-A ufw-docker-logging-deny -m limit --limit 3/min --limit-burst 10 -j LOG --log-prefix "[UFW DOCKER BLOCK] "
-A ufw-docker-logging-deny -j DROP
COMMIT
# END UFW AND DOCKER
UFW_DOCKER
fi

ufw --force enable
systemctl enable fail2ban
systemctl restart fail2ban

# --- Node.js LTS ---
if ! command -v node >/dev/null 2>&1; then
  echo "Installing Node.js..."
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y nodejs
else
  echo "Node.js already installed"
fi

# --- GitHub CLI ---
if ! command -v gh >/dev/null 2>&1; then
  echo "Installing GitHub CLI..."
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg 2>/dev/null
  chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list
  apt-get update -y
  apt-get install -y gh
else
  echo "GitHub CLI already installed"
fi

# --- Claude Code CLI ---
if ! command -v claude >/dev/null 2>&1; then
  echo "Installing Claude Code..."
  npm install -g @anthropic-ai/claude-code
else
  echo "Claude Code already installed"
fi

# --- Git config for agent ---
su - agent -c 'git config --global init.defaultBranch main'

# --- Mark provisioning complete ---
touch /opt/spekk/.provisioned
echo "=== Provisioning complete ==="
`, encoded)
}

// provisionViaSSH runs the provisioning script on a remote machine over SSH.
// It reads the public key from the SSH key path, generates the script, and
// executes it on the remote host.
func provisionViaSSH(ip, keyPath, name string) error {
	// Read the public key to authorize for the agent user.
	pubKeyPath := keyPath + ".pub"
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("reading SSH public key %s: %w (ensure the .pub file exists alongside the private key)", pubKeyPath, err)
	}
	pubKey := strings.TrimSpace(string(pubKeyBytes))

	script := provisionScript(pubKey)

	fmt.Fprintln(os.Stderr, "Provisioning machine via SSH (this may take several minutes)...")
	out, err := runSSHCombined(ip, keyPath, name, script)
	if err != nil {
		return fmt.Errorf("provisioning failed: %w\n%s", err, out)
	}

	// Verify the provisioned marker was created. runSSH returns "" for a
	// failed connection as well as for a missing file, so say only what
	// is certain rather than naming the wrong cause.
	if strings.TrimSpace(runSSH(ip, keyPath, name, "test -f /opt/spekk/.provisioned && echo ok")) != "ok" {
		return fmt.Errorf("provisioning script finished, but could not confirm /opt/spekk/.provisioned on %s: the file is missing, or the check could not connect", ip)
	}

	return nil
}

// stopAgentService stops the spekk-agent service on a manual sandbox and
// removes the credentials spekk put there.
//
// For a manual machine this is the whole of teardown: the machine survives,
// so anything left behind stays live. It returns an error rather than only
// warning, because the caller must not delete the local record of a machine
// that may still be running an agent with these credentials on it.
func stopAgentService(sandbox *SandboxMeta, name string) error {
	fmt.Fprintln(os.Stderr, "Stopping agent service and removing credentials...")
	args := sshHostKeyOpts(name)
	args = append(args, "-o", "ConnectTimeout=10", "-o", "BatchMode=yes")
	if sandbox.SSHKeyPath != "" {
		args = append(args, "-i", sandbox.SSHKeyPath)
	}
	args = append(args, fmt.Sprintf("root@%s", sandbox.IP), strings.Join([]string{
		"set -e",
		"systemctl stop spekk-agent",
		"systemctl disable spekk-agent",
		"rm -f /etc/spekk/agent.env",
		"rm -f /home/agent/.git-credentials",
		// gh auth login writes the same token here, so removing only
		// the two files above leaves it live on a surviving machine.
		"rm -rf /home/agent/.config/gh",
		// is-active exits non-zero when the unit is stopped, which is
		// what we want to see. Turn that into the success case.
		"! systemctl is-active --quiet spekk-agent",
	}, " && "))

	cmd := exec.Command("ssh", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// stopAgent is the seam Destroy stops through, so a test can drive the
// paths around it without an SSH connection.
var stopAgent = stopAgentService
