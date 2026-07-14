#!/usr/bin/env bash

# install.sh - Installs the GoDoctor Antigravity Plugin directly using pre-compiled binaries

set -euo pipefail

# 1. Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "${OS}" in
  darwin)  OS="darwin" ;;
  linux)   OS="linux" ;;
  *)
    echo "❌ Error: OS '${OS}' is not supported by this installer." >&2
    exit 1
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64) ARCH="x64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "❌ Error: Architecture '${ARCH}' is not supported." >&2
    exit 1
    ;;
esac

echo "🔍 Detected platform: ${OS}/${ARCH}"

# 3. Fetch latest release version from GitHub API
echo "🌐 Fetching latest release tag..."
LATEST_RELEASE=$(curl -s https://api.github.com/repos/danicat/godoctor/releases/latest | grep -o '"tag_name": "[^"]*' | head -n1 | cut -d'"' -f4)

if [ -z "${LATEST_RELEASE}" ]; then
  echo "❌ Error: Failed to fetch the latest release tag. Please try again." >&2
  exit 1
fi

echo "🏷️  Latest release: ${LATEST_RELEASE}"

# 4. Construct download URL
FILENAME="${OS}.${ARCH}.godoctor.tar.gz"
DOWNLOAD_URL="https://github.com/danicat/godoctor/releases/download/${LATEST_RELEASE}/${FILENAME}"

# 5. Create a temporary directory for extraction
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "${TEMP_DIR}"' EXIT

echo "📥 Downloading ${FILENAME}..."
if ! curl -sSL -o "${TEMP_DIR}/${FILENAME}" "${DOWNLOAD_URL}"; then
  echo "❌ Error: Failed to download the release asset from ${DOWNLOAD_URL}" >&2
  exit 1
fi

# 6. Extract the archive
echo "📦 Extracting package..."
tar -xzf "${TEMP_DIR}/${FILENAME}" -C "${TEMP_DIR}"
rm "${TEMP_DIR}/${FILENAME}"

# 7. Install via agy plugin install
echo "🔌 Installing plugin via agy..."
if ! agy plugin install "${TEMP_DIR}"; then
  echo "❌ Error: 'agy plugin install' failed." >&2
  exit 1
fi

echo "✅ Success! GoDoctor has been successfully installed as an Antigravity plugin."
