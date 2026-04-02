#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="$ROOT_DIR/docs/launch-posts"
BACKUP_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/git-fire/launch-posts"

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
    backed=0
    for f in 2026-03-31-show-hn.md 2026-03-31-reddit-golang.md 2026-03-31-reddit-programming.md 2026-03-31-reddit-devops.md; do
      if [[ -f "$SOURCE_DIR/$f" ]]; then
        cp "$SOURCE_DIR/$f" "$BACKUP_DIR/$f"
        ((backed++)) || true
      fi
    done
    if [[ $backed -eq 0 ]]; then
      echo "Warning: No launch post files found in $SOURCE_DIR"
    else
      echo "Backed up $backed file(s) to: $BACKUP_DIR"
    fi
    ;;
  restore)
    mkdir -p "$SOURCE_DIR"
    restored=0
    for f in 2026-03-31-show-hn.md 2026-03-31-reddit-golang.md 2026-03-31-reddit-programming.md 2026-03-31-reddit-devops.md; do
      if [[ -f "$BACKUP_DIR/$f" ]]; then
        cp "$BACKUP_DIR/$f" "$SOURCE_DIR/$f"
        ((restored++)) || true
      fi
    done
    if [[ $restored -eq 0 ]]; then
      echo "Warning: No backup files found in $BACKUP_DIR"
    else
      echo "Restored $restored file(s) from: $BACKUP_DIR"
    fi
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
