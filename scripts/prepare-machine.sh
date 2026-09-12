#!/usr/bin/env bash
#
# prepare-machine.sh -- provision a machine you already have (`--provider none`)
# so `spekk sandbox create`/`provision` can equip it. The counterpart to
# internal/sandbox/cloud-init.yaml; keep them in step.
#
# Installs the agent user, Docker, the Claude Code binary, git/gh, and the spekk
# dirs, then writes /opt/spekk/.provisioned. Skips cloud-init's droplet-only
# hardening (apt upgrade, default-deny UFW, fail2ban) that could lock you out.
# Debian/Ubuntu, amd64/arm64, idempotent.
#
# Usage: sudo ./prepare-machine.sh
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
# shellcheck disable=SC1091
. /etc/os-release
ARCH="$(dpkg --print-architecture)"
CODENAME="${VERSION_CODENAME:-}"
# Docker publishes a repo per distro; Debian's codenames aren't in the Ubuntu repo.
case "${ID:-}" in
  ubuntu) DOCKER_DISTRO=ubuntu ;;
  *)      DOCKER_DISTRO=debian ;;
esac

# 64-bit kernel on a 32-bit userland: uname -m says aarch64 but there's no arm64
# loader, and Claude Code has no 32-bit ARM build. Dead end -- fail now.
if [ "$(uname -m)" = "aarch64" ] && [ "$ARCH" = "armhf" ]; then
  echo "This machine has a 64-bit kernel but a 32-bit (armhf) userland." >&2
  echo "Claude Code has no 32-bit ARM build. Install a 64-bit OS (64-bit" >&2
  echo "Raspberry Pi OS or Debian arm64) and re-run this script." >&2
  exit 1
fi

echo "==> Base packages"
# Drop a stale docker.list from a previous run so this update can't choke on it;
# the Docker section recreates it for the detected distro.
rm -f /etc/apt/sources.list.d/docker.list
apt-get update -y
apt-get install -y git jq htop tmux vim unzip ca-certificates curl gnupg

echo "==> agent user"
# Service account the agent runs as: home dir (credentials land here), docker +
# systemd-journal groups, passwordless sudo.
if ! id agent >/dev/null 2>&1; then
  useradd --create-home --shell /bin/bash agent
fi
usermod -aG sudo,systemd-journal agent
echo 'agent ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/90-spekk-agent
chmod 0440 /etc/sudoers.d/90-spekk-agent

echo "==> Docker"
if ! command -v docker >/dev/null 2>&1; then
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/${DOCKER_DISTRO}/gpg" -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${DOCKER_DISTRO} ${CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list
  # Docker CE if the repo has this codename, else the distro's docker.io.
  if apt-get update -y && apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin; then
    :
  else
    echo "   Docker CE repo has no release for ${DOCKER_DISTRO}/${CODENAME}; using the distro's docker.io"
    rm -f /etc/apt/sources.list.d/docker.list
    apt-get update -y
    apt-get install -y docker.io
    apt-get install -y docker-compose-v2 || true # compose plugin, best-effort
  fi
fi
usermod -aG docker agent
systemctl enable --now docker

echo "==> Claude Code CLI (native binary)"
# Install as agent so it owns ~agent/.local/share/claude and `claude update` works.
if [ ! -x /home/agent/.local/bin/claude ]; then
  su - agent -c 'curl -fsSL https://claude.ai/install.sh | bash -s stable'
fi
# Onto the system PATH: the spekk-agent service won't read agent's ~/.local/bin.
ln -sf /home/agent/.local/bin/claude /usr/local/bin/claude

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
