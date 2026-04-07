#!/usr/bin/env bash

set -euo pipefail

ROOT=""
RESET=false
PROFILE="full"

usage() {
    cat <<'EOF'
Usage: scripts/setup-manual-smoke-fixtures.sh [--root DIR] [--profile PROFILE] [--reset]

Creates repeatable local git repos/remotes for manual smoke testing git-fire.

Options:
  --root DIR   Target directory for fixtures
               (default: /tmp/git-fire-manual-smoke)
  --profile    Fixture profile: stage1 | stage2 | stage3 | full
               stage1: clean baseline only
               stage2: stage1 + dirty/no-remote/local-only-branch
               stage3/full: stage2 + conflict + multi-remote divergence
  --reset      Remove existing fixture dir first
  -h, --help   Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --root)
            ROOT="${2:-}"
            shift 2
            ;;
        --profile)
            PROFILE="${2:-}"
            shift 2
            ;;
        --reset)
            RESET=true
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage
            exit 1
            ;;
    esac
done

DEFAULT_TMP_ROOT="${TMPDIR:-/tmp}"
if [[ -z "${ROOT}" ]]; then
    ROOT="${DEFAULT_TMP_ROOT}/git-fire-manual-smoke"
fi

case "${PROFILE}" in
    stage1|stage2|stage3|full) ;;
    *)
        echo "Invalid --profile '${PROFILE}'. Use: stage1|stage2|stage3|full" >&2
        exit 1
        ;;
esac

if ! command -v git >/dev/null 2>&1; then
    echo "git is required on PATH" >&2
    exit 1
fi

if [[ -d "${ROOT}" ]]; then
    if [[ "${RESET}" == true ]]; then
        rm -rf "${ROOT}"
    else
        echo "Target already exists: ${ROOT}" >&2
        echo "Use --reset to recreate it." >&2
        exit 1
    fi
fi

mkdir -p "${ROOT}/repos" "${ROOT}/remotes" "${ROOT}/clones"

init_repo() {
    local repo_path="$1"
    mkdir -p "${repo_path}"
    git init -q -b main "${repo_path}" 2>/dev/null || {
        git init -q "${repo_path}"
        git -C "${repo_path}" symbolic-ref HEAD refs/heads/main
    }
    git -C "${repo_path}" config user.name "git-fire smoke"
    git -C "${repo_path}" config user.email "smoke@git-fire.local"
}

init_bare_remote() {
    local remote_path="$1"
    git init --bare -q -b main "${remote_path}" 2>/dev/null || {
        git init --bare -q "${remote_path}"
        git -C "${remote_path}" symbolic-ref HEAD refs/heads/main
    }
}

commit_file() {
    local repo_path="$1"
    local file="$2"
    local content="$3"
    local message="$4"
    mkdir -p "$(dirname "${repo_path}/${file}")"
    printf "%s\n" "${content}" > "${repo_path}/${file}"
    git -C "${repo_path}" add "${file}"
    git -C "${repo_path}" commit -q -m "${message}"
}

clone_and_configure() {
    local remote_path="$1"
    local clone_path="$2"
    git clone -q "file://${remote_path}" "${clone_path}"
    git -C "${clone_path}" config user.name "remote writer"
    git -C "${clone_path}" config user.email "remote@git-fire.local"
}

write_config() {
    local output_path="$1"
    local mode="$2"
    local conflict="$3"
    cat > "${output_path}" <<EOF
[global]
default_mode = "${mode}"
conflict_strategy = "${conflict}"
auto_commit_dirty = true
scan_path = "${ROOT}/repos"
scan_depth = 6
scan_workers = 4
scan_exclude = []
disable_scan = false
EOF
}

include_stage2=false
include_stage3=false
if [[ "${PROFILE}" == "stage2" || "${PROFILE}" == "stage3" || "${PROFILE}" == "full" ]]; then
    include_stage2=true
fi
if [[ "${PROFILE}" == "stage3" || "${PROFILE}" == "full" ]]; then
    include_stage3=true
fi

declare -a REPO_SUMMARY=()

# Stage 1 baseline: clean and synced.
init_bare_remote "${ROOT}/remotes/clean.git"
init_repo "${ROOT}/repos/clean-repo"
commit_file "${ROOT}/repos/clean-repo" "README.md" "clean repo" "initial"
git -C "${ROOT}/repos/clean-repo" remote add origin "file://${ROOT}/remotes/clean.git"
git -C "${ROOT}/repos/clean-repo" push -q -u origin main
REPO_SUMMARY+=("clean-repo: clean and fully synced")

if [[ "${include_stage2}" == true ]]; then
    # Stage 2: dirty/no-remote/local-only branch behaviors.
    init_bare_remote "${ROOT}/remotes/dirty.git"
    init_repo "${ROOT}/repos/dirty-repo"
    commit_file "${ROOT}/repos/dirty-repo" "README.md" "dirty baseline" "initial"
    git -C "${ROOT}/repos/dirty-repo" remote add origin "file://${ROOT}/remotes/dirty.git"
    git -C "${ROOT}/repos/dirty-repo" push -q -u origin main
    printf "staged change\n" > "${ROOT}/repos/dirty-repo/staged.txt"
    git -C "${ROOT}/repos/dirty-repo" add staged.txt
    printf "unstaged change\n" > "${ROOT}/repos/dirty-repo/unstaged.txt"
    REPO_SUMMARY+=("dirty-repo: staged + unstaged changes")

    init_repo "${ROOT}/repos/no-remote-repo"
    commit_file "${ROOT}/repos/no-remote-repo" "README.md" "no remote baseline" "initial"
    printf "local only dirty\n" >> "${ROOT}/repos/no-remote-repo/README.md"
    REPO_SUMMARY+=("no-remote-repo: dirty with no remotes")

    init_bare_remote "${ROOT}/remotes/local-known.git"
    init_repo "${ROOT}/repos/local-only-branches-repo"
    commit_file "${ROOT}/repos/local-only-branches-repo" "README.md" "local-only baseline" "initial"
    git -C "${ROOT}/repos/local-only-branches-repo" remote add origin "file://${ROOT}/remotes/local-known.git"
    git -C "${ROOT}/repos/local-only-branches-repo" push -q -u origin main
    git -C "${ROOT}/repos/local-only-branches-repo" checkout -q -b feature-local
    commit_file "${ROOT}/repos/local-only-branches-repo" "feature.txt" "feature local only" "feature local commit"
    git -C "${ROOT}/repos/local-only-branches-repo" checkout -q main
    REPO_SUMMARY+=("local-only-branches-repo: unpushed feature-local branch")
fi

if [[ "${include_stage3}" == true ]]; then
    # Stage 3: divergence and multi-remote conflict behavior.
    init_bare_remote "${ROOT}/remotes/conflict.git"
    init_repo "${ROOT}/repos/conflict-repo"
    commit_file "${ROOT}/repos/conflict-repo" "README.md" "conflict baseline" "initial"
    git -C "${ROOT}/repos/conflict-repo" remote add origin "file://${ROOT}/remotes/conflict.git"
    git -C "${ROOT}/repos/conflict-repo" push -q -u origin main
    clone_and_configure "${ROOT}/remotes/conflict.git" "${ROOT}/clones/conflict-writer"
    commit_file "${ROOT}/clones/conflict-writer" "REMOTE.txt" "remote change" "remote advance"
    git -C "${ROOT}/clones/conflict-writer" push -q origin main
    commit_file "${ROOT}/repos/conflict-repo" "LOCAL.txt" "local change" "local diverge"
    REPO_SUMMARY+=("conflict-repo: origin/main diverged local vs remote")

    init_bare_remote "${ROOT}/remotes/multi-origin.git"
    init_bare_remote "${ROOT}/remotes/multi-backup.git"
    init_repo "${ROOT}/repos/multi-remote-repo"
    commit_file "${ROOT}/repos/multi-remote-repo" "README.md" "multi baseline" "initial"
    git -C "${ROOT}/repos/multi-remote-repo" remote add origin "file://${ROOT}/remotes/multi-origin.git"
    git -C "${ROOT}/repos/multi-remote-repo" remote add backup "file://${ROOT}/remotes/multi-backup.git"
    git -C "${ROOT}/repos/multi-remote-repo" push -q -u origin main
    git -C "${ROOT}/repos/multi-remote-repo" push -q backup main
    clone_and_configure "${ROOT}/remotes/multi-origin.git" "${ROOT}/clones/multi-origin-writer"
    commit_file "${ROOT}/clones/multi-origin-writer" "ORIGIN_ONLY.txt" "remote origin change" "origin diverge"
    git -C "${ROOT}/clones/multi-origin-writer" push -q origin main
    commit_file "${ROOT}/repos/multi-remote-repo" "LOCAL_ONLY.txt" "local branch change" "local ahead"
    REPO_SUMMARY+=("multi-remote-repo: origin diverged, backup still fast-forwardable")
fi

write_config "${ROOT}/config_abort.toml" "push-current-branch" "abort"
write_config "${ROOT}/config_new_branch.toml" "push-current-branch" "new-branch"
write_config "${ROOT}/config_push_known.toml" "push-known-branches" "new-branch"

cat > "${ROOT}/MANUAL_SMOKE_RUNBOOK.md" <<EOF
# Manual Smoke Fixtures

Fixture root: \`${ROOT}\`
Profile: \`${PROFILE}\`

## Included repos
EOF

for summary in "${REPO_SUMMARY[@]}"; do
    printf -- "- \`repos/%s\`\n" "${summary}" >> "${ROOT}/MANUAL_SMOKE_RUNBOOK.md"
done

cat >> "${ROOT}/MANUAL_SMOKE_RUNBOOK.md" <<EOF

## Suggested manual commands

Preview:
\`\`\`bash
git-fire --dry-run --path "${ROOT}/repos" --config "${ROOT}/config_abort.toml"
\`\`\`

Conflict behavior (\`abort\`):
\`\`\`bash
git-fire --path "${ROOT}/repos" --config "${ROOT}/config_abort.toml"
\`\`\`

Conflict behavior (\`new-branch\`):
\`\`\`bash
git-fire --path "${ROOT}/repos" --config "${ROOT}/config_new_branch.toml"
\`\`\`

Known-branches mode (local-only branch warning behavior):
\`\`\`bash
git-fire --path "${ROOT}/repos" --config "${ROOT}/config_push_known.toml"
\`\`\`

Registry-only run after first scan:
\`\`\`bash
git-fire --no-scan --config "${ROOT}/config_abort.toml"
\`\`\`

TUI run:
\`\`\`bash
git-fire --fire --path "${ROOT}/repos" --config "${ROOT}/config_abort.toml"
\`\`\`

## Reset fixtures
\`\`\`bash
scripts/setup-manual-smoke-fixtures.sh --root "${ROOT}" --profile "${PROFILE}" --reset
\`\`\`
EOF

echo "Created manual smoke fixtures at: ${ROOT} (profile=${PROFILE})"
echo "Runbook: ${ROOT}/MANUAL_SMOKE_RUNBOOK.md"
