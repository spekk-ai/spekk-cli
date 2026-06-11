#!/usr/bin/env bash
#
# setup-credentials.sh — Run on a freshly provisioned sandbox droplet to inject
# secrets. Run this from your local machine:
#
#   ssh root@DROPLET_IP 'bash -s' < setup-credentials.sh
#
# You'll be prompted for each secret interactively, or you can set them as
# environment variables before running:
#
#   SPEKK_AGENT_TOKEN=xxx SPEKK_HOST=xxx ANTHROPIC_API_KEY=xxx GITHUB_TOKEN=xxx \
#     ssh root@DROPLET_IP 'bash -s' < setup-credentials.sh
#
set -euo pipefail

echo "==> Spekk Sandbox Credential Setup"

# Wait for cloud-init to finish if it's still running
if [ -f /run/cloud-init/result.json ]; then
    echo "    Cloud-init already complete."
else
    echo "    Waiting for cloud-init to finish..."
    cloud-init status --wait || true
fi

# Check provisioning marker
if [ ! -f /opt/spekk/.provisioned ]; then
    echo "ERROR: Droplet not fully provisioned. Check /var/log/cloud-init-output.log"
    exit 1
fi

# Prompt for secrets if not provided via environment
if [ -z "${SPEKK_AGENT_TOKEN:-}" ]; then
    read -rp "Agent auth token (SPEKK_AGENT_TOKEN): " SPEKK_AGENT_TOKEN
fi
if [ -z "${SPEKK_HOST:-}" ]; then
    read -rp "Spekk app host (e.g. spekk-staging.herokuapp.com): " SPEKK_HOST
fi
if [ -z "${ANTHROPIC_API_KEY:-}" ]; then
    read -rp "Anthropic API key: " ANTHROPIC_API_KEY
fi
if [ -z "${GITHUB_TOKEN:-}" ]; then
    read -rp "GitHub PAT for agent: " GITHUB_TOKEN
fi

# Write agent env file
echo "==> Writing /etc/spekk/agent.env"
cat > /etc/spekk/agent.env <<EOF
SPEKK_AGENT_TOKEN=${SPEKK_AGENT_TOKEN}
SPEKK_HOST=${SPEKK_HOST}
ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
GITHUB_TOKEN=${GITHUB_TOKEN}
EOF
chmod 600 /etc/spekk/agent.env
echo "    Done."

# Configure git credentials for agent user
echo "==> Configuring git credentials for agent user"
su - agent -c "git config --global credential.helper store"
su - agent -c "echo 'https://agent:${GITHUB_TOKEN}@github.com' > ~/.git-credentials"
su - agent -c "chmod 600 ~/.git-credentials"

# Configure gh CLI for agent user
echo "==> Configuring gh CLI"
su - agent -c "echo '${GITHUB_TOKEN}' | gh auth login --with-token" 2>/dev/null || true

# Set Anthropic key in agent's profile so Claude Code can find it
echo "==> Setting ANTHROPIC_API_KEY in agent profile"
mkdir -p /home/agent/.bashrc.d
cat > /home/agent/.bashrc.d/spekk.sh <<EOF
export ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
EOF
chown -R agent:agent /home/agent/.bashrc.d

# Ensure .bashrc sources .bashrc.d
if ! grep -q 'bashrc.d' /home/agent/.bashrc 2>/dev/null; then
    cat >> /home/agent/.bashrc <<'EOF'

# Source drop-in scripts
for f in ~/.bashrc.d/*.sh; do [ -r "$f" ] && . "$f"; done
EOF
fi

echo "==> Credential setup complete."
echo ""
echo "Next steps:"
echo "  1. Run deploy-agent.sh to copy the Go binary to /opt/spekk/agent-client"
echo "  2. systemctl enable --now spekk-agent"
echo "  3. Check status: systemctl status spekk-agent"
echo "  4. Check logs: journalctl -u spekk-agent -f"
