#!/usr/bin/env bash
set -euo pipefail

VERSION="${INPUT_VERSION:-latest}"
REPO="${INPUT_REPOSITORY:-jordan-simonovski/helmver}"

if [[ "$VERSION" == "latest" ]]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
fi

TAG="${VERSION#v}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

BIN="helmver-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/v${TAG}/${BIN}"

install_dir="${RUNNER_TEMP:-/tmp}/helmver-bin"
mkdir -p "$install_dir"

echo "Installing helmver ${TAG} (${BIN})..."
curl -fsSL "$URL" -o "${install_dir}/helmver"
chmod +x "${install_dir}/helmver"

if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "$install_dir" >> "$GITHUB_PATH"
fi

"${install_dir}/helmver" --version
