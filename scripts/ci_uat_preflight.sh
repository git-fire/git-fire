#!/usr/bin/env bash
# Lightweight diagnostics for GitHub Actions UAT failures (runner image, tools).
# Invoked from .github/workflows/ci.yml before scripts/uat_test.sh.
set -euo pipefail
echo "==> ci_uat_preflight"
echo "    runner: $(uname -a 2>/dev/null || true)"
echo "    bash: ${BASH_VERSION:-?}"
echo "    go: $(command -v go >/dev/null && go version || echo missing)"
echo "    git: $(command -v git >/dev/null && git --version || echo missing)"
echo "    realpath: $(command -v realpath >/dev/null && realpath --version 2>/dev/null | head -1 || echo missing)"
echo "    CI=${CI:-} GITHUB_ACTIONS=${GITHUB_ACTIONS:-} GIT_FIRE_NON_INTERACTIVE=${GIT_FIRE_NON_INTERACTIVE:-} GIT_FIRE_VERBOSE=${GIT_FIRE_VERBOSE:-}"
echo "    XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-} XDG_CACHE_HOME=${XDG_CACHE_HOME:-}"
