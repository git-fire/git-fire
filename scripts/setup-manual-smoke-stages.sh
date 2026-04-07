#!/usr/bin/env bash

set -euo pipefail

ROOT="/tmp/git-fire-manual-smoke-stages"
RESET=false

usage() {
    cat <<'EOF'
Usage: scripts/setup-manual-smoke-stages.sh [--root DIR] [--reset]

Creates three fixture stages for manual smoke testing:
  stage1 -> baseline clean
  stage2 -> stage1 + dirty/no-remote/local-only branch
  stage3 -> stage2 + divergence/multi-remote conflict
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --root)
            ROOT="${2:-}"
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

if [[ -d "${ROOT}" ]]; then
    if [[ "${RESET}" == true ]]; then
        rm -rf "${ROOT}"
    else
        echo "Target already exists: ${ROOT}" >&2
        echo "Use --reset to recreate it." >&2
        exit 1
    fi
fi

mkdir -p "${ROOT}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/setup-manual-smoke-fixtures.sh" --root "${ROOT}/stage1" --profile stage1 --reset
"${SCRIPT_DIR}/setup-manual-smoke-fixtures.sh" --root "${ROOT}/stage2" --profile stage2 --reset
"${SCRIPT_DIR}/setup-manual-smoke-fixtures.sh" --root "${ROOT}/stage3" --profile stage3 --reset

cat > "${ROOT}/STAGE_INDEX.md" <<EOF
# Manual Smoke Stage Index

Root: \`${ROOT}\`

## Stage Directories
- \`stage1\` baseline clean fixture set
- \`stage2\` adds dirty/no-remote/local-only branch behaviors
- \`stage3\` adds divergence and multi-remote conflict cases

## Run one stage at a time

Recommended wrapper (enforces explicit path/config + isolated HOME):
\`\`\`bash
scripts/run-manual-smoke-stage.sh --stage-root "${ROOT}" --stage 1 --mode dry-run
scripts/run-manual-smoke-stage.sh --stage-root "${ROOT}" --stage 2 --mode push-known
scripts/run-manual-smoke-stage.sh --stage-root "${ROOT}" --stage 3 --mode abort
scripts/run-manual-smoke-stage.sh --stage-root "${ROOT}" --stage 3 --mode new-branch
\`\`\`

Direct commands (equivalent, without HOME isolation):

Stage 1:
\`\`\`bash
git-fire --dry-run --path "${ROOT}/stage1/repos" --config "${ROOT}/stage1/config_abort.toml"
\`\`\`

Stage 2:
\`\`\`bash
git-fire --path "${ROOT}/stage2/repos" --config "${ROOT}/stage2/config_push_known.toml"
\`\`\`

Stage 3:
\`\`\`bash
git-fire --path "${ROOT}/stage3/repos" --config "${ROOT}/stage3/config_abort.toml"
git-fire --path "${ROOT}/stage3/repos" --config "${ROOT}/stage3/config_new_branch.toml"
\`\`\`

## Recreate all stages
\`\`\`bash
scripts/setup-manual-smoke-stages.sh --root "${ROOT}" --reset
\`\`\`

## Future Enhancements
- Add \`scripts/verify-manual-smoke-stage.sh\` to assert expected outcomes after each run.
- Add optional \`--build\` mode to run local \`./git-fire\` binary instead of PATH-installed \`git-fire\`.
- Add JSON summary export for OSS tester bug reports.
EOF

echo "Created staged fixtures at: ${ROOT}"
echo "Stage index: ${ROOT}/STAGE_INDEX.md"
