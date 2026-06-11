#!/usr/bin/env bash
#
# deploy-agent.sh — DEV/DEBUG ESCAPE HATCH
#
# Builds the Go agent client from your local checkout and deploys it directly
# to a sandbox droplet. Useful for rapid iteration without cutting a release.
#
# The production path is:
#   1. Push a version tag (e.g. git tag v1.2.3 && git push origin v1.2.3)
#   2. CI builds and publishes the release automatically
#   3. spekk sandbox deploy <name>  ← downloads from the release and deploys
#
# Usage:
#   ./deploy-agent.sh DROPLET_IP
#
set -euo pipefail

DROPLET_IP="${1:?Usage: ./deploy-agent.sh DROPLET_IP}"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

echo "==> Building Go binary for Linux/amd64"
LDFLAGS="-X main.version=$(git describe --tags --always) \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${REPO_ROOT}/sandbox/sandbox" "${REPO_ROOT}/sandbox"

echo "==> Deploying agent client to ${DROPLET_IP}"

# Copy compiled binary
rsync -az --progress "${REPO_ROOT}/sandbox/sandbox" "root@${DROPLET_IP}:/opt/spekk/agent-client"
ssh "root@${DROPLET_IP}" "chmod +x /opt/spekk/agent-client"

# Enable and start the service
ssh "root@${DROPLET_IP}" "systemctl daemon-reload && systemctl enable --now spekk-agent"

echo "==> Agent deployed. Check status:"
echo "    ssh root@${DROPLET_IP} journalctl -u spekk-agent -f"
