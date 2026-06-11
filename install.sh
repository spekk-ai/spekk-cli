#!/bin/sh
# Install the latest spekk CLI release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
#
# Options (environment variables):
#   SPEKK_INSTALL_DIR   Install directory (default: /usr/local/bin)
set -eu

REPO="spekk-ai/spekk-cli"
INSTALL_DIR="${SPEKK_INSTALL_DIR:-/usr/local/bin}"

err() {
    echo "Error: $1" >&2
    exit 1
}

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
    darwin|linux) ;;
    *) err "unsupported OS: $os — for Windows, download spekk-windows-*.exe from https://github.com/$REPO/releases/latest" ;;
esac

arch=$(uname -m)
case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    aarch64|arm64) arch=arm64 ;;
    *) err "unsupported architecture: $arch" ;;
esac

binary="spekk-$os-$arch"
url="https://github.com/$REPO/releases/latest/download/$binary"

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

echo "Downloading $binary ..."
curl -fsSL "$url" -o "$tmp" || err "download failed: $url"
chmod +x "$tmp"

if [ -w "$INSTALL_DIR" ]; then
    mv "$tmp" "$INSTALL_DIR/spekk"
else
    echo "Installing to $INSTALL_DIR (requires sudo)"
    sudo mv "$tmp" "$INSTALL_DIR/spekk"
fi
trap - EXIT

version=$("$INSTALL_DIR/spekk" version 2>/dev/null || echo "unknown")
echo "Installed spekk $version to $INSTALL_DIR/spekk"
echo
echo "Get started:"
echo "  spekk help"
echo "  https://github.com/$REPO#quick-start"
