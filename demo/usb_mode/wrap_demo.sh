#!/usr/bin/env bash
set -euo pipefail
_REPO="$(cd "$(dirname "$0")/../.." && pwd)"
export DEMO_HOME="${DEMO_HOME:-/tmp/git-fire-usb-demo-record}"
export GIT_FIRE_BIN="${GIT_FIRE_BIN:-$_REPO/git-fire}"
exec bash "$(dirname "$0")/usb_mode_demo_run.sh"
