#!/usr/bin/env bash
set -euo pipefail

VERSION="${INPUT_VERSION:-latest}"
REPO="${INPUT_REPOSITORY:-jordan-simonovski/helmver}"

install_dir="${RUNNER_TEMP:-/tmp}/helmver-bin"
mkdir -p "$install_dir"

if [[ "$VERSION" == "dev" ]]; then
  if ! command -v go >/dev/null 2>&1; then
    echo "go is not available; run actions/setup-go before using version=dev, or pin a released version instead." >&2
    exit 1
  fi
  src_dir="${GITHUB_WORKSPACE:-.}"
  if [[ ! -f "${src_dir}/main.go" ]]; then
    echo "helmver source not found at ${src_dir}" >&2
    exit 1
  fi
  echo "Building helmver from source..."
  (cd "$src_dir" && go build -ldflags "-s -w" -o "${install_dir}/helmver" .)
  chmod +x "${install_dir}/helmver"
  if [[ -n "${GITHUB_PATH:-}" ]]; then
    echo "$install_dir" >> "$GITHUB_PATH"
  fi
  "${install_dir}/helmver" --version
  exit 0
fi

if [[ "$VERSION" == "latest" ]]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
fi

VERSION_NUM="${VERSION#v}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
esac

case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*)
    OS=windows
    ;;
  *)
    echo "unsupported OS: $OS" >&2
    exit 1
    ;;
esac

ASSET="helmver_${VERSION_NUM}_${OS}_${ARCH}"
echo "Installing helmver ${VERSION} (${ASSET})..."

if [[ "$OS" == "windows" ]]; then
  ARCHIVE="${install_dir}/${ASSET}.zip"
  curl -fsSL "https://github.com/${REPO}/releases/download/v${VERSION_NUM}/${ASSET}.zip" -o "$ARCHIVE"
  unzip -o "$ARCHIVE" -d "$install_dir"
  bin_name="helmver.exe"
else
  ARCHIVE="${install_dir}/${ASSET}.tar.gz"
  curl -fsSL "https://github.com/${REPO}/releases/download/v${VERSION_NUM}/${ASSET}.tar.gz" -o "$ARCHIVE"
  tar -xzf "$ARCHIVE" -C "$install_dir"
  bin_name="helmver"
fi

bin_path="$(find "$install_dir" -type f -name "$bin_name" | head -n 1)"
if [[ -z "$bin_path" ]]; then
  echo "helmver binary not found after extracting ${ASSET}" >&2
  exit 1
fi

target="${install_dir}/${bin_name}"
if [[ "$bin_path" != "$target" ]]; then
  mv "$bin_path" "$target"
fi
chmod +x "$target"

if [[ -n "${GITHUB_PATH:-}" ]]; then
  echo "$install_dir" >> "$GITHUB_PATH"
fi

"$target" --version
