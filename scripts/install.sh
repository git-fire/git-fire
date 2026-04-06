#!/usr/bin/env bash
set -euo pipefail

# Install git-fire from GitHub release assets with checksum verification.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/install.sh | bash
# Optional env vars:
#   VERSION=v0.2.0
#   INSTALL_DIR=$HOME/.local/bin
#   REPO=git-fire/git-fire

REPO="${REPO:-git-fire/git-fire}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-}"
BINARY_NAME="git-fire"

log() {
  printf '[git-fire install] %s\n' "$1"
}

fail() {
  printf '[git-fire install] ERROR: %s\n' "$1" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

download_to() {
  src="$1"
  dst="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$src" -o "$dst"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$dst" "$src"
  else
    fail "curl or wget is required"
  fi
}

sha256_file() {
  target="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$target" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$target" | awk '{print $1}'
  else
    fail "sha256sum or shasum is required"
  fi
}

normalize_os() {
  local raw_os
  raw_os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$raw_os" in
    linux) echo "linux" ;;
    darwin) echo "darwin" ;;
    *)
      fail "unsupported OS: $raw_os (expected linux or darwin)"
      ;;
  esac
}

normalize_arch() {
  local raw_arch
  raw_arch="$(uname -m)"
  case "$raw_arch" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv6l|armv7l) echo "armv6" ;;
    i386|i686) echo "386" ;;
    *)
      fail "unsupported architecture: $raw_arch"
      ;;
  esac
}

resolve_version() {
  if [ -n "$VERSION" ]; then
    printf '%s\n' "$VERSION"
    return
  fi

  api_url="https://api.github.com/repos/$REPO/releases/latest"
  if command -v curl >/dev/null 2>&1; then
    response="$(curl -fsSL "$api_url")"
  elif command -v wget >/dev/null 2>&1; then
    response="$(wget -qO- "$api_url")"
  else
    fail "curl or wget is required"
  fi

  tag="$(printf '%s\n' "$response" | awk -F '"' '/"tag_name"[[:space:]]*:/ {print $4; exit}')"
  [ -n "$tag" ] || fail "could not resolve latest release tag from GitHub API"
  printf '%s\n' "$tag"
}

install_binary() {
  local src_bin="$1"
  local target_dir="$2"
  local target_bin="$target_dir/$BINARY_NAME"

  if [ -w "$target_dir" ]; then
    install -m 0755 "$src_bin" "$target_bin"
    return
  fi

  if [ ! -e "$target_dir" ] && mkdir -p "$target_dir"; then
    install -m 0755 "$src_bin" "$target_bin"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$target_dir"
    sudo install -m 0755 "$src_bin" "$target_bin"
    return
  fi

  fail "install directory is not writable and sudo is unavailable: $target_dir"
}

need_cmd tar
os="$(normalize_os)"
arch="$(normalize_arch)"
version_tag="$(resolve_version)"
version="${version_tag#v}"

archive_name="${BINARY_NAME}_${version}_${os}_${arch}.tar.gz"
archive_url="https://github.com/$REPO/releases/download/${version_tag}/${archive_name}"
checksums_url="https://github.com/$REPO/releases/download/${version_tag}/checksums.txt"

log "installing ${BINARY_NAME} ${version_tag} for ${os}/${arch}"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

archive_path="$tmp_dir/$archive_name"
checksums_path="$tmp_dir/checksums.txt"

log "downloading release archive"
download_to "$archive_url" "$archive_path"

log "downloading checksum file"
download_to "$checksums_url" "$checksums_path"

expected_sum="$(awk -v file="$archive_name" '$2 == file {print $1; exit}' "$checksums_path")"
[ -n "$expected_sum" ] || fail "could not find checksum entry for $archive_name"

actual_sum="$(sha256_file "$archive_path")"
if [ "$expected_sum" != "$actual_sum" ]; then
  fail "checksum mismatch for $archive_name"
fi

log "checksum verified"
tar -xzf "$archive_path" -C "$tmp_dir"
[ -f "$tmp_dir/$BINARY_NAME" ] || fail "archive did not contain $BINARY_NAME"

install_binary "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR"

log "installed to $INSTALL_DIR/$BINARY_NAME"
log "verify with: $BINARY_NAME --version"
