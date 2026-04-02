#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="$ROOT_DIR/docs/launch-posts"
BACKUP_DIR="${HOME}/.local/share/git-fire/launch-posts"

mkdir -p "$BACKUP_DIR"

usage() {
  echo "Usage: $0 {backup|restore|status}"
}

copy_if_exists() {
  local src="$1"
  local dst="$2"
  if [[ -f "$src" ]]; then
    cp "$src" "$dst"
  fi
}

case "${1:-}" in
  backup)
    mkdir -p "$BACKUP_DIR"
    copy_if_exists "$SOURCE_DIR/2026-03-31-show-hn.md" "$BACKUP_DIR/2026-03-31-show-hn.md"
    copy_if_exists "$SOURCE_DIR/2026-03-31-reddit-golang.md" "$BACKUP_DIR/2026-03-31-reddit-golang.md"
    copy_if_exists "$SOURCE_DIR/2026-03-31-reddit-programming.md" "$BACKUP_DIR/2026-03-31-reddit-programming.md"
    copy_if_exists "$SOURCE_DIR/2026-03-31-reddit-devops.md" "$BACKUP_DIR/2026-03-31-reddit-devops.md"
    echo "Backed up launch posts to: $BACKUP_DIR"
    ;;
  restore)
    mkdir -p "$SOURCE_DIR"
    copy_if_exists "$BACKUP_DIR/2026-03-31-show-hn.md" "$SOURCE_DIR/2026-03-31-show-hn.md"
    copy_if_exists "$BACKUP_DIR/2026-03-31-reddit-golang.md" "$SOURCE_DIR/2026-03-31-reddit-golang.md"
    copy_if_exists "$BACKUP_DIR/2026-03-31-reddit-programming.md" "$SOURCE_DIR/2026-03-31-reddit-programming.md"
    copy_if_exists "$BACKUP_DIR/2026-03-31-reddit-devops.md" "$SOURCE_DIR/2026-03-31-reddit-devops.md"
    echo "Restored launch posts from: $BACKUP_DIR"
    ;;
  status)
    echo "Source: $SOURCE_DIR"
    echo "Backup: $BACKUP_DIR"
    ls -1 "$SOURCE_DIR" 2>/dev/null || true
    echo "---"
    ls -1 "$BACKUP_DIR" 2>/dev/null || true
    ;;
  *)
    usage
    exit 1
    ;;
esac
