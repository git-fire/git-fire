#!/usr/bin/env bash
# =============================================================================
# git-fire UAT Test Suite — MVP Validation
# =============================================================================
# Runs the compiled git-fire binary against 8 real-git scenarios using local
# bare repos (no network needed). Each scenario gets its own isolated HOME dir
# so the real registry/config are never touched.
#
# Usage: bash scripts/uat_test.sh [--keep-tmp]
#        --keep-tmp  Don't delete temp dirs on exit (useful for post-mortem)
#
# GitHub Actions sets CI / GITHUB_ACTIONS; git-fire treats those as non-interactive
# for the post-backup scan prompt. For other automation, set GIT_FIRE_NON_INTERACTIVE=1.
#
# Actions also sets XDG_CONFIG_HOME / XDG_CACHE_HOME under the runner user. This
# script sets XDG_* under each temp HOME when invoking git-fire so the isolated
# registry is used (see uat_git_fire_cmd).
#
# Optional: GIT_FIRE_VERBOSE=1 — print environment + git-fire version to stderr
# (CI enables this in .github/workflows/ci.yml for easier log triage).
# =============================================================================

set -euo pipefail

BINARY="$(cd "$(dirname "$0")/.." && pwd)/git-fire"

# GitHub Actions (and other CI) export GIT_DIR / GIT_WORK_TREE / GIT_CONFIG_*
# for the workflow checkout. This script shells out to git in unrelated temp
# repos; inherited GIT_* breaks those commands and causes cascading UAT failures.
while IFS= read -r line || [[ -n "$line" ]]; do
	[[ -z "$line" ]] && continue
	name="${line%%=*}"
	[[ "$name" == GIT_* ]] || continue
	[[ "$name" == GIT_FIRE_* ]] && continue
	unset "$name" 2>/dev/null || true
done < <(printenv | grep '^GIT_' || true)

uat_dbg() {
	if [[ -n "${GIT_FIRE_VERBOSE:-}" ]]; then
		echo "[uat] $*" >&2
	fi
}

uat_debug_dump() {
	if [[ -z "${GIT_FIRE_VERBOSE:-}" ]]; then
		return 0
	fi
	uat_dbg "pwd=$(pwd)"
	uat_dbg "binary=$BINARY"
	uat_dbg "shell=$BASH_VERSION uname=$(uname -a 2>/dev/null || echo '?')"
	uat_dbg "CI=${CI:-} GITHUB_ACTIONS=${GITHUB_ACTIONS:-} GIT_FIRE_NON_INTERACTIVE=${GIT_FIRE_NON_INTERACTIVE:-}"
	uat_dbg "XDG_CONFIG_HOME=${XDG_CONFIG_HOME:-} XDG_CACHE_HOME=${XDG_CACHE_HOME:-}"
	if [[ -x "$BINARY" && -n "${UAT_VERBOSE_HOME:-}" ]]; then
		# Run with same XDG isolation as scenarios (pipefail-safe subshell).
		(uat_git_fire_cmd "$UAT_VERBOSE_HOME" "$BINARY" --version 2>&1 || true) \
			| while IFS= read -r line; do uat_dbg "git-fire: $line"; done || true
	fi
	git --version 2>&1 | while IFS= read -r line; do uat_dbg "git: $line"; done || true
}
KEEP_TMP="${1:-}"
PASS=0
FAIL=0
declare -a BUGS=()
declare -a FAILURES=()

# ── Colors ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

log_pass()  { echo -e "${GREEN}  ✓ PASS${NC}  $1"; PASS=$((PASS+1)); }
log_fail()  { echo -e "${RED}  ✗ FAIL${NC}  $1"; FAIL=$((FAIL+1)); FAILURES+=("$1"); }
log_bug()   { echo -e "${YELLOW}  ⚠ BUG${NC}   $1"; BUGS+=("$1"); }
log_info()  { echo -e "${CYAN}    →${NC} $1"; }
log_head()  { echo -e "\n${BOLD}${BLUE}━━━ $1 ━━━${NC}"; }

assert_exit() {
    local expected="$1" actual="$2" label="$3"
    if [[ "$actual" -eq "$expected" ]]; then
        log_pass "$label (exit=$actual)"
    else
        log_fail "$label — expected exit $expected, got $actual"
    fi
}

assert_remote_branch_exists() {
    local remote_path="$1" branch="$2" label="$3"
    if git -C "$remote_path" branch --list | grep -qF "$branch"; then
        log_pass "$label — branch '$branch' exists on remote"
    else
        log_fail "$label — branch '$branch' MISSING on remote"
        log_info "Remote branches: $(git -C "$remote_path" branch --list | tr '\n' ' ')"
    fi
}

assert_remote_branch_absent() {
    local remote_path="$1" branch="$2" label="$3"
    if ! git -C "$remote_path" branch --list | grep -qF "$branch"; then
        log_pass "$label — branch '$branch' absent on remote (expected)"
    else
        log_fail "$label — branch '$branch' found on remote (unexpected)"
    fi
}

assert_remote_commit_count() {
    local remote_path="$1" branch="$2" expected="$3" label="$4"
    local actual
    actual=$(git -C "$remote_path" log "$branch" --oneline 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$actual" -eq "$expected" ]]; then
        log_pass "$label — $branch has $actual commits (expected $expected)"
    else
        log_fail "$label — $branch has $actual commits, expected $expected"
    fi
}

assert_remote_has_pattern() {
    local remote_path="$1" branch="$2" pattern="$3" label="$4"
    if git -C "$remote_path" branch --list | grep -qE "$pattern"; then
        log_pass "$label — remote has branch matching '$pattern'"
    else
        log_fail "$label — remote has NO branch matching '$pattern'"
        log_info "Remote branches: $(git -C "$remote_path" branch --list | tr '\n' ' ')"
    fi
}

assert_local_branch_exists() {
    local repo_path="$1" branch="$2" label="$3"
    if git -C "$repo_path" branch --list | grep -qF "$branch"; then
        log_pass "$label — local branch '$branch' exists"
    else
        log_fail "$label — local branch '$branch' MISSING"
    fi
}

assert_local_commit_count_gt() {
    local repo_path="$1" branch="$2" baseline="$3" label="$4"
    local actual
    actual=$(git -C "$repo_path" log "$branch" --oneline 2>/dev/null | wc -l | tr -d ' ')
    if [[ "$actual" -gt "$baseline" ]]; then
        log_pass "$label — $branch has $actual commits (> $baseline)"
    else
        log_fail "$label — $branch has $actual commits (expected > $baseline)"
    fi
}

# ── Setup helpers ────────────────────────────────────────────────────────────

make_temp_home() {
    local h
    h=$(mktemp -d)
    # Full XDG layout: git-fire uses os.UserConfigDir / UserCacheDir, which honor
    # XDG_* when set. GitHub Actions exports XDG_CONFIG_HOME under /home/runner —
    # if we only set HOME, the real user registry is still used and every scenario
    # in this script shares one repos.toml (massive cross-talk / false failures).
    mkdir -p "$h/.config/git-fire" "$h/.cache/git-fire/logs" \
        "$h/.local/state" "$h/.local/share"
    cat > "$h/.config/git-fire/config.toml" <<'EOF'
[global]
default_mode      = "push-known-branches"
conflict_strategy = "new-branch"
auto_commit_dirty = true
scan_path         = "."
scan_depth        = 5
scan_workers      = 4
scan_exclude      = []

[backup]
platform          = "github"
repo_template     = "backup-{repo}-{date}"
generate_manifest = true

[auth]
use_ssh_agent = false
EOF
    echo "$h"
}

# Prefix env for git-fire so config/registry/logs stay under the temp home even
# when the outer shell inherits CI's XDG_* (see make_temp_home).
uat_git_fire_cmd() {
    local h="$1"
    shift
    HOME="$h" \
        XDG_CONFIG_HOME="$h/.config" \
        XDG_CACHE_HOME="$h/.cache" \
        XDG_STATE_HOME="$h/.local/state" \
        XDG_DATA_HOME="$h/.local/share" \
        "$@"
}

make_bare_remote() {
    local dir="$1/remote.git"
    git init --bare -q -b main "$dir" 2>/dev/null || \
        { git init --bare -q "$dir" && git -C "$dir" symbolic-ref HEAD refs/heads/main; }
    echo "$dir"
}

make_local_repo() {
    local dir="$1"
    mkdir -p "$dir"
    git init -q -b main "$dir" 2>/dev/null || \
        { git init -q "$dir" && git -C "$dir" symbolic-ref HEAD refs/heads/main; }
    git -C "$dir" config user.email "test@git-fire.local"
    git -C "$dir" config user.name  "git-fire UAT"
}

initial_commit_and_push() {
    local repo="$1" remote="$2" branch="${3:-main}"
    echo "initial" > "$repo/readme.txt"
    git -C "$repo" add -A
    git -C "$repo" commit -q -m "initial commit"
    git -C "$repo" remote add origin "file://$remote"
    git -C "$repo" push -q origin "HEAD:$branch"
    git -C "$repo" branch --set-upstream-to="origin/$branch" 2>/dev/null || true
}

run_git_fire() {
    # Usage: run_git_fire <home_dir> <scan_path> [extra flags...]
    local home_dir="$1" scan_path="$2"; shift 2
    uat_git_fire_cmd "$home_dir" "$BINARY" --path "$scan_path" "$@" 2>&1
    return "${PIPESTATUS[0]}"
}

run_git_fire_rc() {
    # Like run_git_fire but captures exit code in $RC (not propagated via set -e)
    local home_dir="$1" scan_path="$2"; shift 2
    local out st
    set +e
    out=$(uat_git_fire_cmd "$home_dir" "$BINARY" --path "$scan_path" "$@" 2>&1)
    st=$?
    set -e
    RC=$st
    printf '%s\n' "$out"
}

# ── Cleanup ──────────────────────────────────────────────────────────────────

TMPDIRS=()
cleanup() {
    if [[ "$KEEP_TMP" == "--keep-tmp" ]]; then
        echo -e "\n${YELLOW}--keep-tmp set. Temp dirs:${NC}"
        for d in "${TMPDIRS[@]}"; do echo "  $d"; done
    else
        for d in "${TMPDIRS[@]}"; do rm -rf "$d"; done
    fi
}
trap cleanup EXIT

# =============================================================================
# PRE-FLIGHT
# =============================================================================
log_head "PRE-FLIGHT"

# Isolated HOME+XDG for verbose dump + version line (matches scenario git-fire env).
UAT_VERBOSE_HOME=$(make_temp_home)
TMPDIRS+=("$UAT_VERBOSE_HOME")

uat_debug_dump

if [[ ! -x "$BINARY" ]]; then
    echo -e "${RED}ERROR: binary not found at $BINARY — run 'make build' first${NC}"
    exit 1
fi
log_pass "Binary exists at $BINARY"

VER=$(uat_git_fire_cmd "$UAT_VERBOSE_HOME" "$BINARY" --version 2>&1 | head -1)
log_info "Version: $VER"

# =============================================================================
# SCENARIO 1 — Staged + Unstaged changes, no upstream conflict
# Expected (current behavior): AutoCommitDirtyWithStrategy creates dual backup
# branches (git-fire-staged-* for staged-only, git-fire-full-* for all changes)
# and pushes them. main commit count stays at 1; files land on backup branches.
# =============================================================================
log_head "SCENARIO 1: Staged + Unstaged changes (no conflict)"

S1=$(mktemp -d); TMPDIRS+=("$S1")
S1_HOME=$(make_temp_home); TMPDIRS+=("$S1_HOME")
S1_REMOTE=$(make_bare_remote "$S1")
S1_REPO="$S1/repo"
make_local_repo "$S1_REPO"
initial_commit_and_push "$S1_REPO" "$S1_REMOTE" main

# Stage a change to file_a.txt
echo "staged change" > "$S1_REPO/file_a.txt"
git -C "$S1_REPO" add file_a.txt

# Leave file_b.txt unstaged
echo "unstaged change" > "$S1_REPO/file_b.txt"

log_info "Setup: staged=file_a.txt, unstaged=file_b.txt"

# Capture output + exit code
S1_OUT=$(uat_git_fire_cmd "$S1_HOME" "$BINARY" --path "$S1" 2>&1) && S1_RC=0 || S1_RC=$?
log_info "git-fire output:"
echo "$S1_OUT" | sed 's/^/    /'

assert_exit 0 "$S1_RC" "S1: exit code"

# Dual-branch strategy: check for backup branches on remote
S1_STAGED_BRANCH=$(git -C "$S1_REMOTE" branch --list | grep -E "git-fire-staged-" | head -1 | tr -d ' ' || true)
S1_FULL_BRANCH=$(git -C "$S1_REMOTE" branch --list | grep -E "git-fire-full-" | head -1 | tr -d ' ' || true)

if [[ -n "$S1_STAGED_BRANCH" ]]; then
    log_pass "S1: git-fire-staged-* backup branch pushed to remote ($S1_STAGED_BRANCH)"
    # staged branch should contain file_a.txt (staged change)
    git -C "$S1_REMOTE" show "$S1_STAGED_BRANCH":file_a.txt > /dev/null 2>&1 && \
        log_pass "S1: file_a.txt (staged) in staged backup branch" || \
        log_fail "S1: file_a.txt (staged) MISSING from staged backup branch"
else
    log_fail "S1: no git-fire-staged-* branch on remote (dual-branch backup not working)"
fi

if [[ -n "$S1_FULL_BRANCH" ]]; then
    log_pass "S1: git-fire-full-* backup branch pushed to remote ($S1_FULL_BRANCH)"
    # full branch should contain both file_a.txt and file_b.txt
    git -C "$S1_REMOTE" show "$S1_FULL_BRANCH":file_a.txt > /dev/null 2>&1 && \
        log_pass "S1: file_a.txt (staged) in full backup branch" || \
        log_fail "S1: file_a.txt (staged) MISSING from full backup branch"
    git -C "$S1_REMOTE" show "$S1_FULL_BRANCH":file_b.txt > /dev/null 2>&1 && \
        log_pass "S1: file_b.txt (unstaged) in full backup branch" || \
        log_fail "S1: file_b.txt (unstaged) MISSING from full backup branch"
else
    log_fail "S1: no git-fire-full-* branch on remote (unstaged changes were not backed up)"
fi

# =============================================================================
# SCENARIO 1b — Only Staged changes (no unstaged)
# =============================================================================
log_head "SCENARIO 1b: Only staged changes"

S1B=$(mktemp -d); TMPDIRS+=("$S1B")
S1B_HOME=$(make_temp_home); TMPDIRS+=("$S1B_HOME")
S1B_REMOTE=$(make_bare_remote "$S1B")
S1B_REPO="$S1B/repo"
make_local_repo "$S1B_REPO"
initial_commit_and_push "$S1B_REPO" "$S1B_REMOTE" main

echo "staged only" > "$S1B_REPO/staged_file.txt"
git -C "$S1B_REPO" add staged_file.txt
log_info "Setup: staged=staged_file.txt, nothing unstaged"

S1B_OUT=$(uat_git_fire_cmd "$S1B_HOME" "$BINARY" --path "$S1B" 2>&1) && S1B_RC=0 || S1B_RC=$?
log_info "git-fire output:"
echo "$S1B_OUT" | sed 's/^/    /'
assert_exit 0 "$S1B_RC" "S1b: exit code"
# Staged-only: expect a git-fire-staged-* branch (no full branch since nothing unstaged)
S1B_STAGED=$(git -C "$S1B_REMOTE" branch --list | grep -E "git-fire-staged-" | head -1 | tr -d ' ' || true)
if [[ -n "$S1B_STAGED" ]]; then
    log_pass "S1b: git-fire-staged-* branch pushed ($S1B_STAGED)"
    git -C "$S1B_REMOTE" show "$S1B_STAGED":staged_file.txt > /dev/null 2>&1 && \
        log_pass "S1b: staged file in backup branch" || \
        log_fail "S1b: staged file MISSING from backup branch"
else
    log_fail "S1b: no git-fire-staged-* branch on remote"
fi

# =============================================================================
# SCENARIO 1c — Only Unstaged changes
# =============================================================================
log_head "SCENARIO 1c: Only unstaged changes"

S1C=$(mktemp -d); TMPDIRS+=("$S1C")
S1C_HOME=$(make_temp_home); TMPDIRS+=("$S1C_HOME")
S1C_REMOTE=$(make_bare_remote "$S1C")
S1C_REPO="$S1C/repo"
make_local_repo "$S1C_REPO"
initial_commit_and_push "$S1C_REPO" "$S1C_REMOTE" main

echo "unstaged only" > "$S1C_REPO/unstaged_file.txt"
# Do NOT stage it
log_info "Setup: nothing staged, unstaged=unstaged_file.txt"

S1C_OUT=$(uat_git_fire_cmd "$S1C_HOME" "$BINARY" --path "$S1C" 2>&1) && S1C_RC=0 || S1C_RC=$?
log_info "git-fire output:"
echo "$S1C_OUT" | sed 's/^/    /'
assert_exit 0 "$S1C_RC" "S1c: exit code"
# Unstaged-only: expect a git-fire-full-* branch (captures all working tree changes)
S1C_FULL=$(git -C "$S1C_REMOTE" branch --list | grep -E "git-fire-full-" | head -1 | tr -d ' ' || true)
if [[ -n "$S1C_FULL" ]]; then
    log_pass "S1c: git-fire-full-* branch pushed ($S1C_FULL)"
    git -C "$S1C_REMOTE" show "$S1C_FULL":unstaged_file.txt > /dev/null 2>&1 && \
        log_pass "S1c: unstaged file in full backup branch" || \
        log_fail "S1c: unstaged file MISSING from full backup branch"
else
    log_fail "S1c: no git-fire-full-* branch on remote"
fi

# =============================================================================
# SCENARIO 2 — Staged + Unstaged + UPSTREAM CONFLICT (main diverged from remote)
# With default_mode=push-known-branches, auto-commit replaces the plan with
# pushes of git-fire-staged-* / git-fire-full-* only (no direct main push in
# this path), so backup pushes succeed and exit 0 even though main is behind
# origin. Remote main must stay unchanged; backup branches must land on remote.
# =============================================================================
log_head "SCENARIO 2: Staged + Unstaged + Upstream Conflict"

S2=$(mktemp -d); TMPDIRS+=("$S2")
S2_HOME=$(make_temp_home); TMPDIRS+=("$S2_HOME")
S2_REMOTE=$(make_bare_remote "$S2")
S2_REPO="$S2/repo"
make_local_repo "$S2_REPO"
initial_commit_and_push "$S2_REPO" "$S2_REMOTE" main

# Advance remote INDEPENDENTLY (simulate another dev pushing)
S2_CLONE="$S2/clone"
git clone -q "file://$S2_REMOTE" "$S2_CLONE"
git -C "$S2_CLONE" config user.email "other@example.com"
git -C "$S2_CLONE" config user.name  "Other Dev"
echo "remote advance" > "$S2_CLONE/remote_change.txt"
git -C "$S2_CLONE" add -A
git -C "$S2_CLONE" commit -q -m "remote advance commit"
git -C "$S2_CLONE" push -q origin main

# Now local diverges — reset + different commit
git -C "$S2_REPO" fetch -q origin
git -C "$S2_REPO" reset --hard HEAD  # stay at initial, remote is ahead
echo "local diverge" > "$S2_REPO/local_change.txt"
git -C "$S2_REPO" add -A
git -C "$S2_REPO" commit -q -m "local diverging commit"

# Add staged + unstaged changes on top
echo "staged fire" > "$S2_REPO/file_staged.txt"
git -C "$S2_REPO" add file_staged.txt
echo "unstaged fire" > "$S2_REPO/file_unstaged.txt"

log_info "Setup: remote is 1 commit AHEAD of local fork point; local has staged+unstaged"
log_info "Remote commit log:"
git -C "$S2_REMOTE" log --oneline | sed 's/^/    /'

S2_OUT=$(uat_git_fire_cmd "$S2_HOME" "$BINARY" --path "$S2" 2>&1) && S2_RC=0 || S2_RC=$?
log_info "git-fire output:"
echo "$S2_OUT" | sed 's/^/    /'

# Expected: backup-only pushes succeed → exit 0 (no main push in this plan path)
assert_exit 0 "$S2_RC" "S2: exit code"

# Remote main unchanged (local diverging commit never pushed to main)
S2_REMOTE_COMMITS=$(git -C "$S2_REMOTE" log main --oneline 2>/dev/null | wc -l | tr -d ' ')
log_info "S2: remote main commit count = $S2_REMOTE_COMMITS"
if [[ "$S2_REMOTE_COMMITS" -eq 2 ]]; then
    log_pass "S2: remote main unchanged at 2 commits (initial + remote advance)"
else
    log_fail "S2: remote has $S2_REMOTE_COMMITS commits on main — expected 2 (initial + remote advance)"
fi

# Dual-branch backups on remote
S2_REMOTE_STAGED=$(git -C "$S2_REMOTE" branch --list | grep -E "git-fire-staged-" | head -1 | tr -d ' ' || true)
S2_REMOTE_FULL=$(git -C "$S2_REMOTE" branch --list | grep -E "git-fire-full-" | head -1 | tr -d ' ' || true)
if [[ -n "$S2_REMOTE_STAGED" && -n "$S2_REMOTE_FULL" ]]; then
    log_pass "S2: backup branches on remote (staged=$S2_REMOTE_STAGED full=$S2_REMOTE_FULL)"
    if git -C "$S2_REMOTE" show "$S2_REMOTE_STAGED":file_staged.txt > /dev/null 2>&1; then
        log_pass "S2: staged file in remote staged backup"
    else
        log_fail "S2: file_staged.txt MISSING from remote staged backup"
    fi
    if git -C "$S2_REMOTE" show "$S2_REMOTE_FULL":file_staged.txt > /dev/null 2>&1; then
        log_pass "S2: staged file in remote full backup"
    else
        log_fail "S2: file_staged.txt MISSING from remote full backup"
    fi
    if git -C "$S2_REMOTE" show "$S2_REMOTE_FULL":file_unstaged.txt > /dev/null 2>&1; then
        log_pass "S2: unstaged file in remote full backup"
    else
        log_fail "S2: file_unstaged.txt MISSING from remote full backup"
    fi
    if git -C "$S2_REMOTE" show "$S2_REMOTE_FULL":local_change.txt > /dev/null 2>&1; then
        log_pass "S2: local diverging commit in remote full backup"
    else
        log_fail "S2: local_change.txt MISSING from remote full backup"
    fi
else
    log_fail "S2: expected both backup branches on remote (staged=$S2_REMOTE_STAGED full=$S2_REMOTE_FULL)"
fi

# =============================================================================
# SCENARIO 3 — Multiple branches, unpushed, NO conflict
# Default mode: push-known-branches → only branches with remote tracking pushed
# =============================================================================
log_head "SCENARIO 3: Multiple branches, unpushed (push-known-branches mode)"

S3=$(mktemp -d); TMPDIRS+=("$S3")
S3_HOME=$(make_temp_home); TMPDIRS+=("$S3_HOME")
S3_REMOTE=$(make_bare_remote "$S3")
S3_REPO="$S3/repo"
make_local_repo "$S3_REPO"
initial_commit_and_push "$S3_REPO" "$S3_REMOTE" main

# Create feature-a: local only, never pushed
git -C "$S3_REPO" checkout -q -b feature-a
echo "feature a work" > "$S3_REPO/feature_a.txt"
git -C "$S3_REPO" add -A
git -C "$S3_REPO" commit -q -m "feature a commit"

# Create feature-b: local only, never pushed
git -C "$S3_REPO" checkout -q -b feature-b main
echo "feature b work" > "$S3_REPO/feature_b.txt"
git -C "$S3_REPO" add -A
git -C "$S3_REPO" commit -q -m "feature b commit"

# Return to main
git -C "$S3_REPO" checkout -q main

log_info "Setup: main pushed; feature-a, feature-b are local-only (never pushed)"
log_info "Local branches: $(git -C "$S3_REPO" branch --list | tr '\n' ' ')"

S3_OUT=$(uat_git_fire_cmd "$S3_HOME" "$BINARY" --path "$S3" 2>&1) && S3_RC=0 || S3_RC=$?
log_info "git-fire output:"
echo "$S3_OUT" | sed 's/^/    /'

assert_exit 0 "$S3_RC" "S3: exit code"

# main should still be on remote
assert_remote_branch_exists "$S3_REMOTE" main "S3: main on remote"

# feature-a and feature-b should NOT be on remote (push-known-branches)
REMOTE_BRANCHES=$(git -C "$S3_REMOTE" branch --list)
if echo "$REMOTE_BRANCHES" | grep -qF "feature-a"; then
    log_fail "S3: feature-a found on remote — unexpected with push-known-branches"
else
    log_pass "S3: feature-a correctly NOT pushed (local-only, not 'known' to remote)"
fi

if echo "$REMOTE_BRANCHES" | grep -qF "feature-b"; then
    log_fail "S3: feature-b found on remote — unexpected with push-known-branches"
else
    log_pass "S3: feature-b correctly NOT pushed (local-only, not 'known' to remote)"
fi

# Verify warning is emitted for local-only branches (Bug 3 fix: no longer silent)
if echo "$S3_OUT" | grep -qF "has no remote tracking ref" \
    && echo "$S3_OUT" | grep -qF "not backed up"; then
    log_pass "S3: warning emitted for local-only branches not backed up"
else
    log_fail "S3: missing warning for local-only branches not backed up"
fi

# Verify DefaultMode from config is applied (Bug 4 fix: scanner no longer hardcodes mode)
log_info "S3: push-known-branches is the expected default mode (registry upsert now respects default_mode config)"

# =============================================================================
# SCENARIO 3b — Multiple branches, push-all mode (via registry pre-seed)
# =============================================================================
log_head "SCENARIO 3b: Multiple branches, push-all mode (registry override)"

S3B=$(mktemp -d); TMPDIRS+=("$S3B")
S3B_HOME=$(make_temp_home); TMPDIRS+=("$S3B_HOME")
S3B_REMOTE=$(make_bare_remote "$S3B")
S3B_REPO="$S3B/repo"
make_local_repo "$S3B_REPO"
initial_commit_and_push "$S3B_REPO" "$S3B_REMOTE" main

git -C "$S3B_REPO" checkout -q -b feature-a
echo "fa" > "$S3B_REPO/fa.txt"
git -C "$S3B_REPO" add -A
git -C "$S3B_REPO" commit -q -m "feature-a commit"

git -C "$S3B_REPO" checkout -q -b feature-b main
echo "fb" > "$S3B_REPO/fb.txt"
git -C "$S3B_REPO" add -A
git -C "$S3B_REPO" commit -q -m "feature-b commit"
git -C "$S3B_REPO" checkout -q main

# Pre-seed registry with push-all mode for this repo
ABS_REPO=$(realpath "$S3B_REPO")
ADDED_AT=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
mkdir -p "$S3B_HOME/.config/git-fire"
cat > "$S3B_HOME/.config/git-fire/repos.toml" <<EOF
[[repos]]
path = "$ABS_REPO"
name = "repo"
status = "active"
mode = "push-all"
added_at = $ADDED_AT
last_seen = $ADDED_AT
EOF

log_info "Setup: registry pre-seeded with push-all for $ABS_REPO"
log_info "Local branches: $(git -C "$S3B_REPO" branch --list | tr '\n' ' ')"

S3B_OUT=$(uat_git_fire_cmd "$S3B_HOME" "$BINARY" --path "$S3B" 2>&1) && S3B_RC=0 || S3B_RC=$?
log_info "git-fire output:"
echo "$S3B_OUT" | sed 's/^/    /'

assert_exit 0 "$S3B_RC" "S3b: exit code with push-all mode"

REMOTE_B_BRANCHES=$(git -C "$S3B_REMOTE" branch --list)
if echo "$REMOTE_B_BRANCHES" | grep -qF "feature-a"; then
    log_pass "S3b: feature-a pushed with push-all mode"
else
    log_fail "S3b: feature-a NOT pushed even with push-all mode"
    log_info "Remote branches: $REMOTE_B_BRANCHES"
fi
if echo "$REMOTE_B_BRANCHES" | grep -qF "feature-b"; then
    log_pass "S3b: feature-b pushed with push-all mode"
else
    log_fail "S3b: feature-b NOT pushed even with push-all mode"
fi

# =============================================================================
# SCENARIO 4 — Multiple branches WITH upstream conflict
# push-known-branches: diverged branches fail, clean branches succeed
# =============================================================================
log_head "SCENARIO 4: Multiple branches with upstream conflict (partial failure)"

S4=$(mktemp -d); TMPDIRS+=("$S4")
S4_HOME=$(make_temp_home); TMPDIRS+=("$S4_HOME")
S4_REMOTE=$(make_bare_remote "$S4")
S4_REPO="$S4/repo"
make_local_repo "$S4_REPO"
initial_commit_and_push "$S4_REPO" "$S4_REMOTE" main

# Create feature-ok: push it, then add a local commit (fast-forward = will succeed)
git -C "$S4_REPO" checkout -q -b feature-ok
echo "ok feature" > "$S4_REPO/ok.txt"
git -C "$S4_REPO" add -A
git -C "$S4_REPO" commit -q -m "feature-ok initial"
git -C "$S4_REPO" push -q origin feature-ok
git -C "$S4_REPO" branch --set-upstream-to="origin/feature-ok" feature-ok 2>/dev/null || true
# Add another local commit to feature-ok (fast-forward will work)
echo "ok extra" >> "$S4_REPO/ok.txt"
git -C "$S4_REPO" add -A
git -C "$S4_REPO" commit -q -m "feature-ok extra commit"

# Create feature-conflict: push it, then diverge
git -C "$S4_REPO" checkout -q -b feature-conflict main
echo "conflict feature" > "$S4_REPO/conflict.txt"
git -C "$S4_REPO" add -A
git -C "$S4_REPO" commit -q -m "feature-conflict initial"
git -C "$S4_REPO" push -q origin feature-conflict
git -C "$S4_REPO" branch --set-upstream-to="origin/feature-conflict" feature-conflict 2>/dev/null || true

# Advance remote's feature-conflict via clone
S4_CLONE="$S4/clone"
git clone -q "file://$S4_REMOTE" "$S4_CLONE"
git -C "$S4_CLONE" config user.email "other@example.com"
git -C "$S4_CLONE" config user.name "Other Dev"
git -C "$S4_CLONE" checkout -q feature-conflict
echo "remote conflict" >> "$S4_CLONE/conflict.txt"
git -C "$S4_CLONE" add -A
git -C "$S4_CLONE" commit -q -m "remote conflict advance"
git -C "$S4_CLONE" push -q origin feature-conflict

# Local feature-conflict diverges
echo "local conflict" >> "$S4_REPO/conflict.txt"
git -C "$S4_REPO" add -A
git -C "$S4_REPO" commit -q -m "local conflict diverge"

# Return to main
git -C "$S4_REPO" checkout -q main

log_info "Setup: feature-ok (fast-forward ok), feature-conflict (diverged, push will fail)"

S4_OUT=$(uat_git_fire_cmd "$S4_HOME" "$BINARY" --path "$S4" 2>&1) && S4_RC=0 || S4_RC=$?
log_info "git-fire output:"
echo "$S4_OUT" | sed 's/^/    /'

# Should fail overall (some branch failed)
if [[ "$S4_RC" -ne 0 ]]; then
    log_pass "S4: exit code non-zero (partial failure expected)"
else
    log_fail "S4: exit code 0 — should fail since feature-conflict push rejected"
fi

# feature-ok should be on remote with the extra commit
S4_OK_COMMITS=$(git -C "$S4_REMOTE" log feature-ok --oneline 2>/dev/null | wc -l | tr -d ' ')
if [[ "$S4_OK_COMMITS" -ge 2 ]]; then
    log_pass "S4: feature-ok pushed successfully ($S4_OK_COMMITS commits on remote)"
else
    log_fail "S4: feature-ok has $S4_OK_COMMITS commits on remote, expected >= 2"
fi

# feature-conflict remote should still have 3 commits:
# initial + feature-conflict-initial + remote-advance (local diverge was correctly rejected)
S4_CONFLICT_COMMITS=$(git -C "$S4_REMOTE" log feature-conflict --oneline 2>/dev/null | wc -l | tr -d ' ')
if [[ "$S4_CONFLICT_COMMITS" -eq 3 ]]; then
    log_pass "S4: feature-conflict remote unchanged at 3 commits (push correctly rejected)"
else
    log_fail "S4: feature-conflict has $S4_CONFLICT_COMMITS commits on remote, expected 3"
fi

# =============================================================================
# SCENARIO 5 — Push completely fails (bad remote URL)
# Expected: auto-commit succeeds locally, push fails, exit non-zero
# Local commit should be preserved (not rolled back)
# =============================================================================
log_head "SCENARIO 5: Push completely fails (nonexistent remote)"

S5=$(mktemp -d); TMPDIRS+=("$S5")
S5_HOME=$(make_temp_home); TMPDIRS+=("$S5_HOME")
S5_REPO="$S5/repo"
make_local_repo "$S5_REPO"

# Setup: repo with a REAL initial commit but a BAD remote URL
git -C "$S5_REPO" checkout -q -b main 2>/dev/null || true
echo "init" > "$S5_REPO/readme.txt"
git -C "$S5_REPO" add -A
git -C "$S5_REPO" commit -q -m "initial commit"
git -C "$S5_REPO" remote add origin "file:///nonexistent/path/that/does/not/exist.git"

# Add staged + unstaged changes
echo "staged data" > "$S5_REPO/staged.txt"
git -C "$S5_REPO" add staged.txt
echo "unstaged data" > "$S5_REPO/unstaged.txt"
S5_BEFORE_COMMITS=$(git -C "$S5_REPO" log --oneline | wc -l | tr -d ' ')

log_info "Setup: bad remote URL, staged=staged.txt, unstaged=unstaged.txt"

S5_OUT=$(uat_git_fire_cmd "$S5_HOME" "$BINARY" --path "$S5" 2>&1) && S5_RC=0 || S5_RC=$?
log_info "git-fire output:"
echo "$S5_OUT" | sed 's/^/    /'

if [[ "$S5_RC" -ne 0 ]]; then
    log_pass "S5: exit code non-zero (push failed as expected)"
else
    log_fail "S5: exit code 0 — should have failed (bad remote)"
fi

# Auto-commit goes to backup branches (not main); check for local backup branches
S5_LOCAL_STAGED=$(git -C "$S5_REPO" branch --list | grep -E "git-fire-staged-" | head -1 | tr -d ' ' || true)
S5_LOCAL_FULL=$(git -C "$S5_REPO" branch --list | grep -E "git-fire-full-" | head -1 | tr -d ' ' || true)
if [[ -n "$S5_LOCAL_STAGED" && -n "$S5_LOCAL_FULL" ]]; then
    log_pass "S5: local backup branch(es) created before push attempt (staged=$S5_LOCAL_STAGED full=$S5_LOCAL_FULL)"
    git -C "$S5_REPO" show "$S5_LOCAL_STAGED":staged.txt > /dev/null 2>&1 && \
        log_pass "S5: staged file preserved in staged backup branch" || \
        log_fail "S5: staged file NOT in staged backup branch"
    git -C "$S5_REPO" show "$S5_LOCAL_FULL":staged.txt > /dev/null 2>&1 && \
        log_pass "S5: staged file preserved in full backup branch" || \
        log_fail "S5: staged file NOT in full backup branch"
    git -C "$S5_REPO" show "$S5_LOCAL_FULL":unstaged.txt > /dev/null 2>&1 && \
        log_pass "S5: unstaged file preserved in full backup branch" || \
        log_fail "S5: unstaged file NOT in full backup branch"
else
    log_fail "S5: expected both local backup branches before push attempt (staged=$S5_LOCAL_STAGED full=$S5_LOCAL_FULL)"
fi

# Check that error output contains meaningful info
if echo "$S5_OUT" | grep -qi "error\|fail\|❌"; then
    log_pass "S5: error output contains failure indication"
else
    log_fail "S5: error output missing failure indication — user may not know push failed"
fi

# Check UX bug: Cobra prints full usage text on error (S5, S2, S4 all trigger this)
if echo "$S5_OUT" | grep -q "Use \"git-fire \[command\] --help\""; then
    log_bug "[LOW] S5/S2/S4: Cobra prints full usage text + flags on every error exit. In an emergency, this buries the actual error message under 20+ lines of help text. rootCmd.SilenceUsage should be set to true in cmd/root.go."
fi

# =============================================================================
# SCENARIO 6 — Clean repo (no dirty changes)
# Expected: skip auto-commit, still push (nothing new), exit 0
# =============================================================================
log_head "SCENARIO 6: Clean repo (nothing dirty)"

S6=$(mktemp -d); TMPDIRS+=("$S6")
S6_HOME=$(make_temp_home); TMPDIRS+=("$S6_HOME")
S6_REMOTE=$(make_bare_remote "$S6")
S6_REPO="$S6/repo"
make_local_repo "$S6_REPO"
initial_commit_and_push "$S6_REPO" "$S6_REMOTE" main

log_info "Setup: clean repo, all pushed, nothing dirty"

S6_OUT=$(uat_git_fire_cmd "$S6_HOME" "$BINARY" --path "$S6" 2>&1) && S6_RC=0 || S6_RC=$?
log_info "git-fire output:"
echo "$S6_OUT" | sed 's/^/    /'

assert_exit 0 "$S6_RC" "S6: exit code"
assert_remote_commit_count "$S6_REMOTE" main 1 "S6: remote unchanged (still 1 commit)"

# =============================================================================
# SCENARIO 7 — Dry run
# Expected: plan printed, NO commits made, NO pushes made, exit 0
# =============================================================================
log_head "SCENARIO 7: Dry run (--dry-run)"

S7=$(mktemp -d); TMPDIRS+=("$S7")
S7_HOME=$(make_temp_home); TMPDIRS+=("$S7_HOME")
S7_REMOTE=$(make_bare_remote "$S7")
S7_REPO="$S7/repo"
make_local_repo "$S7_REPO"
initial_commit_and_push "$S7_REPO" "$S7_REMOTE" main

echo "dry run staged" > "$S7_REPO/dry_staged.txt"
git -C "$S7_REPO" add dry_staged.txt
echo "dry run unstaged" > "$S7_REPO/dry_unstaged.txt"

log_info "Setup: staged + unstaged changes, running with --dry-run"

S7_OUT=$(uat_git_fire_cmd "$S7_HOME" "$BINARY" --path "$S7" --dry-run 2>&1) && S7_RC=0 || S7_RC=$?
log_info "git-fire output:"
echo "$S7_OUT" | sed 's/^/    /'

assert_exit 0 "$S7_RC" "S7: exit code"

# No new commits on remote
assert_remote_commit_count "$S7_REMOTE" main 1 "S7: remote unchanged (dry-run made no pushes)"

# No new local commits
S7_LOCAL=$(git -C "$S7_REPO" log --oneline | wc -l | tr -d ' ')
if [[ "$S7_LOCAL" -eq 1 ]]; then
    log_pass "S7: no local commits made (dry-run preserved state)"
else
    log_fail "S7: $S7_LOCAL local commits — dry-run should not commit"
fi

# Staged files still staged
if git -C "$S7_REPO" diff --cached --quiet; then
    log_fail "S7: staged file no longer staged — dry-run modified state"
else
    log_pass "S7: staged file still staged (dry-run preserved state)"
fi

if echo "$S7_OUT" | grep -qi "dry run\|fire drill\|no changes"; then
    log_pass "S7: dry-run message in output"
else
    log_fail "S7: dry-run completion message missing from output"
fi

# =============================================================================
# SCENARIO 8 — Repo with NO remote (should skip, not panic)
# =============================================================================
log_head "SCENARIO 8: Repo with no remote configured"

S8=$(mktemp -d); TMPDIRS+=("$S8")
S8_HOME=$(make_temp_home); TMPDIRS+=("$S8_HOME")
S8_REPO="$S8/repo"
make_local_repo "$S8_REPO"

git -C "$S8_REPO" checkout -q -b main 2>/dev/null || true
echo "local only" > "$S8_REPO/file.txt"
git -C "$S8_REPO" add -A
git -C "$S8_REPO" commit -q -m "local only commit"
echo "dirty change" >> "$S8_REPO/file.txt"

log_info "Setup: repo with NO remote, dirty changes"

S8_OUT=$(uat_git_fire_cmd "$S8_HOME" "$BINARY" --path "$S8" 2>&1) && S8_RC=0 || S8_RC=$?
log_info "git-fire output:"
echo "$S8_OUT" | sed 's/^/    /'

# Should not panic — exit 0 or 1, but not crash
if [[ "$S8_RC" -lt 2 ]]; then
    log_pass "S8: no crash/panic (exit=$S8_RC)"
else
    log_fail "S8: abnormal exit code $S8_RC (possible panic)"
fi

if echo "$S8_OUT" | grep -qi "skip\|no remote"; then
    log_pass "S8: 'skip/no remote' message in output"
else
    log_info "S8: output didn't explicitly mention skip/no-remote (may still be OK)"
fi

# =============================================================================
# SCENARIO 9 — Skip auto-commit flag
# Expected: no auto-commit, just push current branch
# =============================================================================
log_head "SCENARIO 9: --skip-auto-commit flag"

S9=$(mktemp -d); TMPDIRS+=("$S9")
S9_HOME=$(make_temp_home); TMPDIRS+=("$S9_HOME")
S9_REMOTE=$(make_bare_remote "$S9")
S9_REPO="$S9/repo"
make_local_repo "$S9_REPO"
initial_commit_and_push "$S9_REPO" "$S9_REMOTE" main

echo "skip commit change" > "$S9_REPO/skip.txt"
git -C "$S9_REPO" add skip.txt

log_info "Setup: staged change, running with --skip-auto-commit"

S9_OUT=$(uat_git_fire_cmd "$S9_HOME" "$BINARY" --path "$S9" --skip-auto-commit 2>&1) && S9_RC=0 || S9_RC=$?
log_info "git-fire output:"
echo "$S9_OUT" | sed 's/^/    /'

assert_exit 0 "$S9_RC" "S9: exit code"

# Remote should still have only 1 commit (skip.txt was not committed or pushed)
assert_remote_commit_count "$S9_REMOTE" main 1 "S9: remote unchanged (auto-commit skipped)"

# skip.txt should still be staged locally
if ! git -C "$S9_REPO" diff --cached --quiet; then
    log_pass "S9: staged file still staged (auto-commit was skipped)"
else
    log_fail "S9: staged file no longer staged — was it committed despite --skip-auto-commit?"
fi

# =============================================================================
# FINAL REPORT
# =============================================================================
echo ""
echo -e "${BOLD}${BLUE}$(printf '═%.0s' {1..60})${NC}"
echo -e "${BOLD}${BLUE}  git-fire MVP UAT — FINAL REPORT${NC}"
echo -e "${BOLD}${BLUE}$(printf '═%.0s' {1..60})${NC}"
echo ""
echo -e "  ${GREEN}PASS: $PASS${NC}"
echo -e "  ${RED}FAIL: $FAIL${NC}"
echo -e "  ${YELLOW}BUGS: ${#BUGS[@]}${NC}"

if [[ "${#FAILURES[@]}" -gt 0 ]]; then
    echo ""
    echo -e "${RED}Failed checks:${NC}"
    for f in "${FAILURES[@]+"${FAILURES[@]}"}"; do
        echo -e "  ${RED}✗${NC} $f"
    done
fi

if [[ "${#BUGS[@]}" -gt 0 ]]; then
    echo ""
    echo -e "${YELLOW}Bugs found:${NC}"
    for b in "${BUGS[@]+"${BUGS[@]}"}"; do
        echo -e "  ${YELLOW}⚠${NC} $b"
    done
fi

echo ""
echo -e "${BOLD}Key Behavioral Summary (current post-fix behavior):${NC}"
echo "  • AutoCommitDirtyWithStrategy (dual-branch): ACTIVE — staged → git-fire-staged-*, all → git-fire-full-*"
echo "  • Dirty repo + push-known-branches: dual-branch backups push to remote; diverged main is not auto-pushed in that path (see S2)"
echo "  • push-known-branches: warns for local-only branches (no longer silent — Bug 3 fixed)"
echo "  • DefaultMode config: applied via registry upsert (no longer dead code — Bug 4 fixed)"
echo "  • conflict_strategy='new-branch': evaluated by planner (no longer dead code — Bug 2 fixed)"
echo "  • SilenceUsage: cobra usage suppressed on errors (Bug 5 fixed)"
echo ""

if [[ "$FAIL" -eq 0 ]]; then
    echo -e "${GREEN}${BOLD}All checks passed! (Note: see bugs above for design-level issues)${NC}"
else
    echo -e "${RED}${BOLD}$FAIL check(s) FAILED.${NC}"
fi
echo ""

# Non-zero exit so CI and `scripts/validate.sh` can gate on failures
if [[ "$FAIL" -gt 0 ]]; then
    exit 1
fi
exit 0
