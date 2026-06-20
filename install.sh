#!/usr/bin/env bash
set -e

REPO="kernelcode0/aptify"

echo "Installing aptify-cli..."

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*) os="linux" ;;
  darwin*) os="darwin" ;;
  msys*|cygwin*|mingw*) os="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

EXT=""
if [ "$os" = "windows" ]; then
  EXT=".exe"
fi

BINARY_NAME="aptify-cli-${os}-${arch}${EXT}"

# Fetch latest release URL
API_URL="https://api.github.com/repos/$REPO/releases/latest"
echo "Fetching latest release from $API_URL..."
DOWNLOAD_URL=$(curl -s "$API_URL" | grep -o "https://github.com/[^\"]*$BINARY_NAME")

if [ -z "$DOWNLOAD_URL" ]; then
  echo "Error: Could not find release asset $BINARY_NAME for your platform."
  echo "You may need to compile from source or check the Releases page manually."
  exit 1
fi

echo "Downloading $DOWNLOAD_URL..."
TMP_BIN="/tmp/aptify-cli$EXT"
curl -sSL "$DOWNLOAD_URL" -o "$TMP_BIN"

echo "Fetching checksums..."
CHECKSUM_URL=$(curl -s "$API_URL" | grep -o "https://github.com/[^\"]*checksums.txt" || true)

if [ -n "$CHECKSUM_URL" ]; then
  curl -sSL "$CHECKSUM_URL" -o "/tmp/checksums.txt"
  EXPECTED_HASH=$(grep "$BINARY_NAME" "/tmp/checksums.txt" | awk '{print $1}')
  
  if [ -n "$EXPECTED_HASH" ]; then
    echo "Verifying checksum..."
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL_HASH=$(sha256sum "$TMP_BIN" | awk '{print $1}')
    else
      ACTUAL_HASH=$(shasum -a 256 "$TMP_BIN" | awk '{print $1}')
    fi
    
    if [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
      echo "Error: Checksum mismatch! Expected $EXPECTED_HASH, got $ACTUAL_HASH."
      echo "This could indicate a corrupted download or a supply chain attack."
      rm -f "$TMP_BIN"
      exit 1
    fi
    echo "Checksum verified successfully."
  else
    echo "Warning: No checksum found for $BINARY_NAME in checksums.txt. Proceeding without verification."
  fi
fi

chmod +x "$TMP_BIN"

DEST="/usr/local/bin/aptify-cli$EXT"

if [ "$os" = "windows" ]; then
  # For windows via Git Bash/MSYS
  DEST="$HOME/bin/aptify-cli.exe"
  mkdir -p "$HOME/bin"
  mv "$TMP_BIN" "$DEST"
  echo "Installed to $DEST. Make sure $HOME/bin is in your PATH."
else
  echo "Moving to $DEST (you may be prompted for your password)..."
  sudo mv "$TMP_BIN" "$DEST"
  echo "Installed successfully! Run 'aptify-cli --help' to get started."
fi
