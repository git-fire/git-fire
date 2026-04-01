# Git Fire Specification (Current Behavior)

This document reflects current implemented behavior in `cmd/root.go`, `internal/config`, `internal/executor`, and `internal/git`.

## CLI Surface

`git-fire` supports these root flags:

- `--dry-run`: plan-only run, no writes
- `--fire-drill`: alias for `--dry-run`
- `--fire`: interactive TUI selection mode
- `--path`: scan root override (default `"."`)
- `--skip-auto-commit`: disable auto-commit for this run
- `--no-scan`: disable filesystem scan for this run (registry-only)
- `--init`: write example config and exit
- `--force`: allow overwriting config when used with `--init`
- `--backup-to`: accepted by parser but intentionally not implemented
- `--config`: explicit config file path
- `--status`: show SSH + repository status and exit

### Explicit CLI Constraints

- `--fire` and `--dry-run` are mutually exclusive and return an explicit error.
- `--fire` and `--fire-drill` are also invalid together (`--fire-drill` maps to `--dry-run`).
- `--backup-to` returns an explicit not-implemented error:
  - `--backup-to is not implemented yet; use configured remotes instead`

## Runtime Flow

There is no "Is the building on fire?" confirmation prompt in current code.

Current execution routes:

1. `--status` -> `handleStatus()` and exit
2. `--init` -> `handleInit()` and exit
3. `--fire` -> streaming TUI flow (`runFireStream`)
4. `--dry-run` / `--fire-drill` -> batch plan flow (`runBatch`)
5. default -> streaming scan+backup flow (`runStream`)

## Conflict Handling

`global.conflict_strategy` supports:

- `new-branch`:
  - planner detects divergence for current-branch mode repos
  - schedules `create-fire-branch` and pushes backup branch(es)
- `abort`:
  - planner detects divergence and marks repo `Skip`
  - skip reason explicitly states `conflict_strategy=abort` and diverged branch/remote

## Auto-Commit Safety (Dual-Branch Strategy)

`internal/git/operations.go` `AutoCommitDirtyWithStrategy`:

- Captures original `HEAD` SHA at start.
- Uses original SHA for all resets.
- On any failure after creating one or more backup commits, performs cleanup reset to original SHA.
- On success with `ReturnToOriginal=true`, resets back to original SHA (not `HEAD~N` math).

This avoids orphaned temporary commits and avoids reset failures when commit depth is smaller than the number of temporary commits created in the run.

## Config Schema and Defaults

Canonical schema is `internal/config/types.go`; defaults are in `internal/config/defaults.go`.

```toml
[global]
default_mode = "push-known-branches"      # push-known-branches | push-all | leave-untouched
conflict_strategy = "new-branch"          # new-branch | abort
auto_commit_dirty = true
block_on_secrets = true
scan_path = "."
scan_exclude = [".cache", "node_modules", ".venv", "venv", "vendor", "dist", "build", "target"]
scan_depth = 10
scan_workers = 8
push_workers = 4
cache_ttl = "24h"
rescan_submodules = false
disable_scan = false

[ui]
show_fire_animation = true
fire_tick_ms = 180
color_profile = "classic"                 # classic | synthwave | forest | arctic

[backup]
target_remote = ""
platform = "github"                       # github | gitlab | gitea
api_token = ""
repo_template = "backup-{repo}-{date}"
organization = ""
generate_manifest = true

[auth]
ssh_passphrase = ""
use_ssh_agent = true

[plugins]
enabled = []
command = []
webhook = []
```

Per-repo overrides are represented by `[[repos]]` entries with:

- `path`
- `remote`
- `mode`
- `skip_auto_commit`
- `rescan_submodules`
