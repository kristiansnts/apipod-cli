#!/bin/sh
set -e

REPO="rpay/apipod-cli"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i location | sed 's/.*tag\/v//' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
  VERSION="0.1.0"
fi

BINARY_NAME="apipod-cli-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}"

echo "Installing apipod-cli v${VERSION} (${OS}/${ARCH})..."

TMPDIR=$(mktemp -d)
curl -fsSL "$URL" -o "${TMPDIR}/apipod-cli"
chmod +x "${TMPDIR}/apipod-cli"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/apipod-cli" "${INSTALL_DIR}/apipod-cli"
else
  sudo mv "${TMPDIR}/apipod-cli" "${INSTALL_DIR}/apipod-cli"
fi

rm -rf "$TMPDIR"

echo "✓ Installed apipod-cli to ${INSTALL_DIR}/apipod-cli"
echo "  Run 'apipod-cli login' to get started."
