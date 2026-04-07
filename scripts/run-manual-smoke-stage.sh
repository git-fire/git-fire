#!/usr/bin/env bash

set -euo pipefail

STAGE_ROOT="${TMPDIR:-/tmp}/git-fire-manual-smoke-stages"
STAGE=""
MODE="live"
KEEP_HOME=false

usage() {
    cat <<'EOF'
Usage: scripts/run-manual-smoke-stage.sh --stage {1|2|3} [options]

Runs git-fire against a staged manual smoke fixture with:
- explicit --path and --config
- isolated HOME (so ~/.config/git-fire is untouched)

Options:
  --stage N          Stage number: 1, 2, or 3 (required)
  --stage-root DIR   Stage root directory (default: /tmp/git-fire-manual-smoke-stages)
  --mode MODE        One of:
                     dry-run     -> --dry-run with abort config
                     fire        -> --fire with abort config
                     live        -> default run with abort config
                     abort       -> default run with abort config
                     new-branch  -> default run with new-branch config
                     push-known  -> default run with push-known config
                     status      -> --status with abort config
  --keep-home        Keep isolated HOME directory after run (printed on exit)
  -h, --help         Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --stage)
            STAGE="${2:-}"
            shift 2
            ;;
        --stage-root)
            STAGE_ROOT="${2:-}"
            shift 2
            ;;
        --mode)
            MODE="${2:-}"
            shift 2
            ;;
        --keep-home)
            KEEP_HOME=true
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

if [[ -z "${STAGE}" ]]; then
    echo "--stage is required" >&2
    usage
    exit 1
fi

case "${STAGE}" in
    1|2|3) ;;
    *)
        echo "Invalid --stage '${STAGE}' (must be 1, 2, or 3)" >&2
        exit 1
        ;;
esac

case "${MODE}" in
    dry-run|fire|live|abort|new-branch|push-known|status) ;;
    *)
        echo "Invalid --mode '${MODE}'" >&2
        usage
        exit 1
        ;;
esac

if ! command -v git-fire >/dev/null 2>&1; then
    echo "git-fire must be available on PATH" >&2
    exit 1
fi

STAGE_DIR="${STAGE_ROOT}/stage${STAGE}"
REPOS_DIR="${STAGE_DIR}/repos"
ABORT_CFG="${STAGE_DIR}/config_abort.toml"
NEW_BRANCH_CFG="${STAGE_DIR}/config_new_branch.toml"
PUSH_KNOWN_CFG="${STAGE_DIR}/config_push_known.toml"

if [[ ! -d "${REPOS_DIR}" ]]; then
    echo "Missing repos dir: ${REPOS_DIR}" >&2
    echo "Run: scripts/setup-manual-smoke-stages.sh --root \"${STAGE_ROOT}\" --reset" >&2
    exit 1
fi

HOME_ISOLATED="${STAGE_DIR}/.home"
mkdir -p "${HOME_ISOLATED}/.config/git-fire" "${HOME_ISOLATED}/.cache/git-fire/logs"

cleanup() {
    if [[ "${KEEP_HOME}" == true ]]; then
        echo "Isolated HOME preserved: ${HOME_ISOLATED}"
    else
        rm -rf "${HOME_ISOLATED}"
    fi
}
trap cleanup EXIT

CFG="${ABORT_CFG}"
EXTRA_ARGS=()
case "${MODE}" in
    dry-run)
        CFG="${ABORT_CFG}"
        EXTRA_ARGS+=(--dry-run)
        ;;
    fire)
        CFG="${ABORT_CFG}"
        EXTRA_ARGS+=(--fire)
        ;;
    live|abort)
        CFG="${ABORT_CFG}"
        ;;
    new-branch)
        CFG="${NEW_BRANCH_CFG}"
        ;;
    push-known)
        CFG="${PUSH_KNOWN_CFG}"
        ;;
    status)
        CFG="${ABORT_CFG}"
        EXTRA_ARGS+=(--status)
        ;;
esac

if [[ ! -f "${CFG}" ]]; then
    echo "Missing config file: ${CFG}" >&2
    exit 1
fi

echo "Stage: stage${STAGE}"
echo "Mode: ${MODE}"
echo "Repos path: ${REPOS_DIR}"
echo "Config: ${CFG}"
echo "HOME (isolated): ${HOME_ISOLATED}"
echo

HOME="${HOME_ISOLATED}" git-fire --path "${REPOS_DIR}" --config "${CFG}" "${EXTRA_ARGS[@]}"
