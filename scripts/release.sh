#!/usr/bin/env bash
set -euo pipefail

# Release script: builds cross-compiled binaries and uploads them to Gemfury
#
# Required environment variables:
#   GEMFURY_TOKEN   - API token for Gemfury authentication
#
# Optional environment variables:
#   GEMFURY_ACCOUNT - Gemfury account name (default: spekk)
#   VERSION         - Version to embed and tag artifacts with (default: git describe)

GEMFURY_ACCOUNT="${GEMFURY_ACCOUNT:-spekk}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
DIST="dist"
BINARY="spekk"
PLATFORMS=(
    darwin/amd64
    darwin/arm64
    linux/amd64
    linux/arm64
    windows/amd64
    windows/arm64
)

# --- Validation ---

if [ -z "${GEMFURY_TOKEN:-}" ]; then
    echo "ERROR: GEMFURY_TOKEN environment variable is required" >&2
    echo "Get your token from https://manage.fury.io/manage/$GEMFURY_ACCOUNT/tokens" >&2
    exit 1
fi

echo "=== Spekk Release ==="
echo "Version:  $VERSION"
echo "Account:  $GEMFURY_ACCOUNT"
echo "Binaries: ${#PLATFORMS[@]} platforms"
echo ""

# --- Build ---

echo "--- Building binaries ---"
make build-all VERSION="$VERSION"
echo ""

# --- Upload ---

echo "--- Uploading to Gemfury ---"
failed=0

for platform in "${PLATFORMS[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"
    src="${DIST}/${BINARY}-${os}-${arch}"
    versioned="${BINARY}-${os}-${arch}-v${VERSION}"

    if [ "$os" = "windows" ]; then
        src="${src}.exe"
        versioned="${versioned}.exe"
    fi

    if [ ! -f "$src" ]; then
        echo "ERROR: Binary not found: $src" >&2
        failed=1
        continue
    fi

    # Copy to versioned name for upload
    cp "$src" "${DIST}/${versioned}"

    echo "Uploading ${versioned} ..."
    if ! curl -sSf -F "package=@${DIST}/${versioned}" \
        -u "${GEMFURY_TOKEN}:" \
        "https://push.fury.io/${GEMFURY_ACCOUNT}/"; then
        echo "ERROR: Failed to upload ${versioned}" >&2
        failed=1
    fi
done

echo ""

if [ "$failed" -ne 0 ]; then
    echo "ERROR: One or more uploads failed" >&2
    exit 1
fi

echo "=== Release complete ==="
echo "All ${#PLATFORMS[@]} binaries uploaded as v${VERSION}"
echo ""
echo "Download URLs:"
for platform in "${PLATFORMS[@]}"; do
    os="${platform%/*}"
    arch="${platform#*/}"
    name="${BINARY}-${os}-${arch}-v${VERSION}"
    if [ "$os" = "windows" ]; then name="${name}.exe"; fi
    echo "  https://TOKEN@fury.io/${GEMFURY_ACCOUNT}/${name}"
done
