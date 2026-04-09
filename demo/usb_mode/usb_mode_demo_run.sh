#!/usr/bin/env bash
# Self-contained USB mode demo for screen recording (isolated HOME).
set -euo pipefail

SAY() { printf "\n\033[1;36m━━ %s ━━\033[0m\n" "$*"; sleep 0.8; }
PAUSE() { printf "\033[90m… pause …\033[0m\n"; sleep "${1:-1.2}"; }

DEMO_HOME="${DEMO_HOME:-/tmp/git-fire-usb-demo-$$}"
export HOME="$DEMO_HOME"
export XDG_CONFIG_HOME="$HOME/.config"
export XDG_CACHE_HOME="$HOME/.cache"
export XDG_DATA_HOME="$HOME/.local/share"
export XDG_STATE_HOME="$HOME/.local/state"

_REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
GIT_FIRE_BIN="${GIT_FIRE_BIN:-$_REPO_ROOT/git-fire}"
VIRTUAL_USB="$DEMO_HOME/VIRTUAL_USB_DRIVE"
PROJECT="$DEMO_HOME/projects/demo-app"
CONFIG="$XDG_CONFIG_HOME/git-fire/config.toml"

mkdir -p "$XDG_CONFIG_HOME/git-fire" "$PROJECT" "$VIRTUAL_USB"

SAY "Isolated demo HOME: $DEMO_HOME"
SAY "Virtual USB / backup folder: $VIRTUAL_USB"

# --- Config v1: minimal USB section, scan limited to demo project tree ---
SAY "config.toml — initial version (USB targets via CLI only)"
cat >"$CONFIG" <<EOF
[global]
scan_path = "$(dirname "$PROJECT")"
scan_depth = 4
default_mode = "push-known-branches"
auto_commit_dirty = true
disable_scan = false

[ui]
show_startup_quote = false
show_fire_animation = false

[usb]
strategy = "git-mirror"
workers = 1
target_workers = 1
create_on_first_use = false
sync_policy = "keep"
EOF
echo "----- $CONFIG -----"
cat "$CONFIG"
PAUSE 1.5

# --- Git repo with several branches ---
SAY "Create demo-app repo with multiple branches"
cd "$PROJECT"
git init -b main
printf "# demo-app\n\nPortable backup demo for git-fire USB mode.\n" >README.md
git add README.md
git config user.email "demo@git-fire.local"
git config user.name "USB Demo"
git commit -m "init: main"

git checkout -b feature/payments
printf "\n## Payments\nStub integration.\n" >>README.md
git add README.md && git commit -m "feature: payments section"

git checkout -b hotfix/critical
echo "fix.txt" >fix.txt && git add fix.txt && git commit -m "hotfix: add fix marker"

git checkout -b experiment/ideas
echo "idea.md" >idea.md && git add idea.md && git commit -m "experiment: ideas file"

git checkout main
mkdir -p docs
echo "Overview for mainline." >docs/overview.txt
git add docs/overview.txt && git commit -m "docs: overview"

git checkout feature/payments
SAY "Branch tips (diverse branches)"
git branch -vv
PAUSE 1

# --- First USB run: marker missing ---
SAY "First USB sync: target has no .git-fire marker yet (expect error)"
set +e
"$GIT_FIRE_BIN" --config "$CONFIG" --path "$(dirname "$PROJECT")" --usb "$VIRTUAL_USB" 2>&1
EC=$?
set -e
echo "(exit code $EC — expected non-zero)"
PAUSE 1.5

# --- Bootstrap marker + first successful sync ---
SAY "Bootstrap volume with --usb-init and mirror to virtual drive"
"$GIT_FIRE_BIN" --config "$CONFIG" --path "$(dirname "$PROJECT")" --usb "$VIRTUAL_USB" --usb-init
PAUSE 1

SAY "Volume marker (.git-fire) and mirror layout"
ls -la "$VIRTUAL_USB"
echo "----- $VIRTUAL_USB/.git-fire -----"
cat "$VIRTUAL_USB/.git-fire"
echo "----- repos (bare mirrors) -----"
find "$VIRTUAL_USB/repos" -maxdepth 2 -type d 2>/dev/null | head -20
PAUSE 1

SAY "Branches visible on USB mirror (git ls-remote)"
BARE=$(find "$VIRTUAL_USB/repos" -name "*.git" -type d | head -1)
git ls-remote --heads "$BARE" | head -15
PAUSE 1.5

# --- Working tree change + re-sync ---
SAY "Dirty working tree on feature/payments → auto-commit + re-sync"
git checkout feature/payments
printf "\n### Status\nReady for USB backup round 2.\n" >>README.md
# uncommitted change
"$GIT_FIRE_BIN" --config "$CONFIG" --path "$(dirname "$PROJECT")" --usb "$VIRTUAL_USB" --usb-init
PAUSE 1

git ls-remote --heads "$BARE" | grep -E 'feature/payments|main' || true
PAUSE 1

# --- Evolve config: file-backed target + create_on_first_use ---
SAY "config.toml — evolved: [[usb.targets]], workers=2, create_on_first_use=true"
cat >"$CONFIG" <<EOF
[global]
scan_path = "$(dirname "$PROJECT")"
scan_depth = 4
default_mode = "push-known-branches"
auto_commit_dirty = true
disable_scan = false

[ui]
show_startup_quote = false
show_fire_animation = false

[usb]
strategy = "git-mirror"
workers = 2
target_workers = 1
create_on_first_use = true
sync_policy = "keep"

[[usb.targets]]
name = "virtual-travel-stick"
path = "$VIRTUAL_USB"
enabled = true
EOF
echo "----- updated $CONFIG -----"
cat "$CONFIG"
PAUSE 1.5

SAY "Sync using only config file targets (resolved from [[usb.targets]])"
"$GIT_FIRE_BIN" --config "$CONFIG" --path "$(dirname "$PROJECT")"
PAUSE 1

# --- Registry snippet ---
SAY "repos.toml — registry entry after discovery (excerpt)"
REG="$XDG_CONFIG_HOME/git-fire/repos.toml"
if [[ -f "$REG" ]]; then
  head -80 "$REG"
else
  echo "(no repos.toml yet)"
fi
PAUSE 1

SAY "Optional: per-repo USB overrides (append usb_repo_path example)"
if [[ -f "$REG" ]]; then
  echo ""
  echo "Tip: edit repos.toml to set usb_strategy, usb_repo_path, usb_sync_policy per path."
fi

SAY "Manifest from last run"
if [[ -f "$VIRTUAL_USB/git-fire-usb-manifest.json" ]]; then
  head -40 "$VIRTUAL_USB/git-fire-usb-manifest.json"
fi

SAY "Demo complete."
PAUSE 0.5
