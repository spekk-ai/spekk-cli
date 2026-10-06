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
#        sudo SPEKK_SHARE_USER=<login> ./prepare-machine.sh  # also let <login>
#          read/write /opt/spekk/workspace (a box you use interactively, e.g. a Pi)
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
apt-get install -y git jq htop tmux vim unzip ca-certificates curl gnupg acl

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

echo "==> spekk CLI"
# Install as agent into a directory agent owns, so `spekk update` needs no sudo,
# and outside agent's home, which Ubuntu creates 0750, so the link on the system
# PATH works for every user, not only agent and root.
install -d -o agent -g agent -m 0755 /opt/spekk/cli
if [ ! -x /opt/spekk/cli/spekk ]; then
  su - agent -c 'curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | SPEKK_INSTALL_DIR=/opt/spekk/cli sh'
fi
# An earlier version of this script installed into agent's home. That copy would
# shadow this one on agent's own PATH and stop receiving updates.
rm -f /home/agent/.local/bin/spekk
ln -sf /opt/spekk/cli/spekk /usr/local/bin/spekk

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

# Optional: let a human login user share the agent's workspace. On a box you also
# use interactively (e.g. a Raspberry Pi), set SPEKK_SHARE_USER to your login name
# so you can read/write /opt/spekk/workspace alongside the agent. ACLs, not chgrp:
# deploy re-runs `chown -R agent:agent /opt/spekk`, which preserves ACLs but would
# wipe a group change, and a default ACL also grants access to the files the agent
# creates later, which its umask would otherwise deny.
if [ -n "${SPEKK_SHARE_USER:-}" ]; then
  echo "==> Sharing /opt/spekk/workspace with ${SPEKK_SHARE_USER}"
  if ! id "${SPEKK_SHARE_USER}" >/dev/null 2>&1; then
    echo "SPEKK_SHARE_USER=${SPEKK_SHARE_USER} is not an existing user." >&2
    exit 1
  fi
  # deploy creates workspace at deploy time; make it now so the ACL is in place.
  mkdir -p /opt/spekk/workspace
  chown agent:agent /opt/spekk/workspace
  setfacl -R    -m "u:${SPEKK_SHARE_USER}:rwX" /opt/spekk/workspace
  setfacl -R -d -m "u:${SPEKK_SHARE_USER}:rwX" /opt/spekk/workspace
fi

echo "==> Marking provisioning complete"
touch /opt/spekk/.provisioned

cat <<EOF

Done. This machine now carries /opt/spekk/.provisioned.

Register it from your workstation (add --ssh-user if you do not log in as root):

  spekk sandbox create --provider none --ip <this-machine-ip> --name <name> --ssh-key <path>

If a create already recorded the sandbox, finish it instead with:

  spekk sandbox provision <name>
EOF
