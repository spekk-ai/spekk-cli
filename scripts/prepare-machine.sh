#!/usr/bin/env bash
#
# prepare-machine.sh -- provision a machine you already have so `spekk sandbox
# create --provider none` (or `spekk sandbox provision`) can equip it.
#
# spekk does not provision a machine it did not create: on a droplet spekk made,
# cloud-init runs internal/sandbox/cloud-init.yaml; on a machine you bring, you
# run the equivalent yourself. This script mirrors the agent-relevant parts of
# cloud-init.yaml -- keep them in step -- but adapts for a machine you already
# have: it detects Debian vs Ubuntu (cloud-init assumes an Ubuntu droplet) and
# installs the Claude Code native binary rather than the npm package.
#
# It installs ONLY what the agent needs (the `agent` user, Docker, the Claude
# Code CLI native binary, git/gh, the spekk directories) and ends by writing
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
# shellcheck disable=SC1091
. /etc/os-release
ARCH="$(dpkg --print-architecture)"
CODENAME="${VERSION_CODENAME:-}"
# Docker publishes separate repos per distro. A droplet is Ubuntu; a Raspberry
# Pi is Debian (ID=debian, or raspbian on 32-bit) -- so the Ubuntu repo has no
# Release file for a Debian codename like "trixie". Everything not Ubuntu uses
# the Debian repo.
case "${ID:-}" in
  ubuntu) DOCKER_DISTRO=ubuntu ;;
  *)      DOCKER_DISTRO=debian ;;
esac

# A 64-bit kernel over a 32-bit userland (uname -m reports aarch64, but the
# userland is armhf) is a common Raspberry Pi setup and a dead end: the arm64
# agent has no loader here, and Claude Code publishes no 32-bit ARM build. Fail
# now rather than install a binary that cannot execute.
if [ "$(uname -m)" = "aarch64" ] && [ "$ARCH" = "armhf" ]; then
  echo "This machine has a 64-bit kernel but a 32-bit (armhf) userland." >&2
  echo "Claude Code has no 32-bit ARM build. Install a 64-bit OS (64-bit" >&2
  echo "Raspberry Pi OS or Debian arm64) and re-run this script." >&2
  exit 1
fi

echo "==> Base packages"
# Drop a Docker repo file a previous run may have left (an earlier version
# pinned the wrong distro), so this first update does not choke on it. The
# Docker section below recreates it for the detected distro.
rm -f /etc/apt/sources.list.d/docker.list
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
  curl -fsSL "https://download.docker.com/linux/${DOCKER_DISTRO}/gpg" -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  echo "deb [arch=${ARCH} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${DOCKER_DISTRO} ${CODENAME} stable" \
    > /etc/apt/sources.list.d/docker.list
  # Docker CE if the repo carries this codename; otherwise the distro's own
  # docker.io, so a codename Docker has not published yet (a fresh Debian) still
  # gets a working engine.
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
# Install it as the agent user so it owns its
# versions directory (~agent/.local/share/claude) and `claude update` works,
# and so the launcher resolves under the agent service's $HOME at runtime.
if [ ! -x /home/agent/.local/bin/claude ]; then
  su - agent -c 'curl -fsSL https://claude.ai/install.sh | bash -s stable'
fi
# Put it on the system PATH the spekk-agent service uses (systemd's default
# PATH has /usr/local/bin; a service running as agent does not read ~/.local/bin).
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
