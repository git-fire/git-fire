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
    return 0
  fi
  return 1
}

case "${1:-}" in
  backup)
    mkdir -p "$BACKUP_DIR"
    backed=0
    for src in "$SOURCE_DIR"/*.md; do
      [[ -e "$src" ]] || break
      f="$(basename "$src")"
      if copy_if_exists "$src" "$BACKUP_DIR/$f"; then
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
    for src in "$BACKUP_DIR"/*.md; do
      [[ -e "$src" ]] || break
      f="$(basename "$src")"
      if copy_if_exists "$src" "$SOURCE_DIR/$f"; then
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
