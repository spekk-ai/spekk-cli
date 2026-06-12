#!/bin/sh
# Install the latest spekk CLI release.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/spekk-ai/spekk-cli/main/install.sh | sh
#
# Options (environment variables):
#   SPEKK_INSTALL_DIR   Install directory (default: ~/.local/bin)
set -eu

REPO="spekk-ai/spekk-cli"
INSTALL_DIR="${SPEKK_INSTALL_DIR:-$HOME/.local/bin}"

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

if [ ! -d "$INSTALL_DIR" ]; then
    if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
        echo "Creating $INSTALL_DIR (requires sudo)"
        sudo mkdir -p "$INSTALL_DIR"
    fi
fi

if [ -w "$INSTALL_DIR" ]; then
    mv "$tmp" "$INSTALL_DIR/spekk"
else
    echo "Installing to $INSTALL_DIR (requires sudo)"
    sudo mv "$tmp" "$INSTALL_DIR/spekk"
fi
trap - EXIT

version=$("$INSTALL_DIR/spekk" version 2>/dev/null || echo "unknown")
echo "Installed spekk $version to $INSTALL_DIR/spekk"

# Warn if another spekk earlier on PATH would shadow this install.
existing=$(command -v spekk 2>/dev/null || true)
if [ -n "$existing" ] && [ "$existing" != "$INSTALL_DIR/spekk" ]; then
    echo "Note: found another spekk at $existing, which may shadow this install." >&2
fi

# Warn (with a copy-paste fix) if the install dir is not on PATH.
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        case "$INSTALL_DIR" in
            "$HOME"/*) path_line="export PATH=\"\$HOME${INSTALL_DIR#"$HOME"}:\$PATH\"" ;;
            *)         path_line="export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
        esac
        case "${SHELL:-}" in
            */zsh) rc_file="~/.zshrc" ;;
            */bash)
                if [ "$os" = "darwin" ]; then
                    rc_file="~/.bash_profile"
                else
                    rc_file="~/.bashrc"
                fi
                ;;
            *) rc_file="your shell config file" ;;
        esac
        {
            echo
            echo "Warning: $INSTALL_DIR is not on your PATH."
            echo "Add this line to $rc_file:"
            echo
            echo "  $path_line"
            echo
            echo "Then restart your shell (or run: source $rc_file)."
        } >&2
        ;;
esac

echo
echo "Get started:"
echo "  spekk help"
echo "  https://github.com/$REPO#quick-start"
