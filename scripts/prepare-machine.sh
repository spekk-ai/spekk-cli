#!/usr/bin/env bash
#
# prepare-machine.sh -- provision a machine you already have so `spekk sandbox
# create --provider none` (or `spekk sandbox provision`) can equip it.
#
# spekk does not provision a machine it did not create: on a droplet spekk made,
# cloud-init runs internal/sandbox/cloud-init.yaml; on a machine you bring, you
# run the equivalent yourself. This script is that equivalent, kept in step with
# cloud-init.yaml -- change one and change the other.
#
# It installs ONLY what the agent needs (the `agent` user, Docker, Node + the
# Claude Code CLI, git/gh, the spekk directories) and ends by writing
# /opt/spekk/.provisioned. It deliberately leaves out the droplet-only hardening
# in cloud-init.yaml -- a full `apt upgrade`, a default-deny UFW policy that
# allows only port 22, and fail2ban -- because on a machine you already use those
# can lock you out or disrupt whatever else it serves. Apply your own firewall
# policy separately if you want one.
#
# It is arch-aware (Debian/Ubuntu, amd64 or arm64, e.g. a Raspberry Pi) and safe
# to re-run.
#
# Usage (as root, or via sudo):
#   sudo ./prepare-machine.sh
#
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "prepare-machine.sh must run as root; re-run with sudo." >&2
  exit 1
fi

if ! command -v apt-get >/dev/null 2>&1; then
  echo "This script targets Debian/Ubuntu (apt-get). Adapt it for other distros." >&2
  exit 1
fi

export DEBIAN_FRONTEND=noninteractive
ARCH="$(dpkg --print-architecture)"
CODENAME="$(. /etc/os-release && echo "$VERSION_CODENAME")"

echo "==> Base packages"
apt-get update -y
apt-get install -y git jq htop tmux vim unzip ca-certificates curl gnupg

echo "==> agent user"
# The service account the agent runs as: a home directory (credentials land in
# ~agent), passwordless sudo, and the docker + systemd-journal groups.
if ! id agent >/dev/null 2>&1; then
  useradd --create-home --shell /bin/bash agent
fi
usermod -aG sudo,systemd-journal agent
echo 'agent ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/90-spekk-agent
chmod 0440 /etc/sudoers.d/90-spekk-agent

echo "==> Docker"
if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
fi
usermod -aG docker agent
systemctl enable --now docker

echo "==> Node.js LTS + Claude Code CLI"
if ! command -v node >/dev/null 2>&1; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y nodejs
fi
npm install -g @anthropic-ai/claude-code

echo "==> GitHub CLI"
if ! command -v gh >/dev/null 2>&1; then
  curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
  chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg
  echo "deb [arch=${ARCH} signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list
  apt-get update -y
  apt-get install -y gh
fi

echo "==> spekk directories"
# spekk chowns /opt/spekk to agent when it deploys, but the tree must exist.
mkdir -p /opt/spekk /etc/spekk /var/log/spekk
chown agent:agent /var/log/spekk
su - agent -c 'git config --global init.defaultBranch main'

echo "==> Marking provisioning complete"
touch /opt/spekk/.provisioned

cat <<EOF

Done. This machine now carries /opt/spekk/.provisioned.

Register it from your workstation (add --ssh-user if you do not log in as root):

  spekk sandbox create --provider none --ip <this-machine-ip> --name <name> --ssh-key <path>

If a create already recorded the sandbox, finish it instead with:

  spekk sandbox provision <name>
EOF
