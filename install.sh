#!/bin/sh
set -e

REPO="Gartner24/forge"
BINARY="forge"
INSTALL_DIR="/usr/local/bin"

die() {
    echo "error: $1" >&2
    exit 1
}

# OS check
OS=$(uname -s)
case "$OS" in
    Linux) ;;
    *) die "unsupported OS: $OS (only Linux is supported)" ;;
esac

# Architecture detection
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *) die "unsupported architecture: $ARCH" ;;
esac

ASSET="forge-linux-${ARCH}"

# Fetch latest release tag from GitHub API
echo "Fetching latest release..."
if command -v curl >/dev/null 2>&1; then
    RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
elif command -v wget >/dev/null 2>&1; then
    RELEASE_JSON=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest")
else
    die "curl or wget is required"
fi

# Extract download URL for the matching asset
DOWNLOAD_URL=$(echo "$RELEASE_JSON" | grep -o '"browser_download_url": *"[^"]*'"${ASSET}"'"' | grep -o 'https://[^"]*')
if [ -z "$DOWNLOAD_URL" ]; then
    die "could not find asset '${ASSET}' in the latest release"
fi

VERSION=$(echo "$RELEASE_JSON" | grep -o '"tag_name": *"[^"]*"' | grep -o 'core/v[^"]*')
echo "Installing Forge ${VERSION} (${ARCH})..."

# Download to a temp file
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$TMP" "$DOWNLOAD_URL"
else
    wget -qO "$TMP" "$DOWNLOAD_URL"
fi

chmod +x "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
    echo "Installing to ${INSTALL_DIR} requires sudo..."
    sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

# Verify
if command -v forge >/dev/null 2>&1; then
    echo "Installed: $(forge --version)"
else
    echo "Installed to ${INSTALL_DIR}/${BINARY}"
    echo "Make sure ${INSTALL_DIR} is in your PATH."
fi
