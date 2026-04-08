# Manual Smoke Fixtures

These scripts create repeatable local Git repo states for manual `git-fire` validation.

- Designed for Unix-like shells (`bash`).
- Uses local filesystem remotes (`file://...`), no network hosting required.
- Defaults to `${TMPDIR:-/tmp}` so fixture data stays in temp space.

## Scripts

- `scripts/setup-manual-smoke-fixtures.sh`
  - Builds one fixture root with a selected profile.
  - Profiles:
    - `stage1`: clean baseline only
    - `stage2`: `stage1` plus dirty/no-remote/local-only branch cases
    - `stage3` / `full`: `stage2` plus divergence and multi-remote conflict cases
- `scripts/setup-manual-smoke-stages.sh`
  - Builds `stage1`, `stage2`, and `stage3` roots in one command.
  - Writes `STAGE_INDEX.md` with suggested commands.
- `scripts/run-manual-smoke-stage.sh`
  - Safe wrapper to run one stage with explicit `--path` + `--config`.
  - Uses isolated `HOME` so your real `~/.config/git-fire` and cache remain untouched.

## Quick Start

Create all staged fixtures:

```bash
scripts/setup-manual-smoke-stages.sh --reset
```

Run stage 1 preview:

```bash
scripts/run-manual-smoke-stage.sh --stage 1 --mode dry-run
```

Run stage 2 known-branches mode:

```bash
scripts/run-manual-smoke-stage.sh --stage 2 --mode push-known
```

Run stage 3 conflict behavior:

```bash
scripts/run-manual-smoke-stage.sh --stage 3 --mode abort
scripts/run-manual-smoke-stage.sh --stage 3 --mode new-branch
```

## Data Layout

Default stage root:

```text
${TMPDIR:-/tmp}/git-fire-manual-smoke-stages
```

Each stage contains:

- `repos/` local working repos to scan
- `remotes/` local bare remotes
- `clones/` helper clones used to create remote divergence
- `config_abort.toml`, `config_new_branch.toml`, `config_push_known.toml`
- `MANUAL_SMOKE_RUNBOOK.md` (generated command hints)

## Cleanup and Reset

Recreate all stages from scratch:

```bash
scripts/setup-manual-smoke-stages.sh --reset
```

Use a custom root:

```bash
scripts/setup-manual-smoke-stages.sh --root "/path/to/tmp-area" --reset
scripts/run-manual-smoke-stage.sh --stage-root "/path/to/tmp-area" --stage 1 --mode dry-run
```
