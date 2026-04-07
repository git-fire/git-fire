# Git Fire — behavior spec (implementation-aligned)

**Status:** Beta. This document describes **what the current codebase does**. Roadmap-only ideas are labeled **planned** or **deferred** explicitly.
This document also serves as written behavior/spec context for requirements validation and exploratory sections, but beta release decisions should defer to `README.md` and shipped code on `main` when mismatches exist.

**Canonical sources:** `cmd/root.go`, `internal/config/{types,defaults,loader}.go`, `internal/git`, `internal/executor`, `internal/registry`, `PLUGINS.md`, `.github/workflows/ci.yml`.

Related docs: [README.md](README.md), [docs/README.md](docs/README.md), [PLUGINS.md](PLUGINS.md), [docs/REGISTRY.md](docs/REGISTRY.md), [docs/PROJECT_OVERVIEW.md](docs/PROJECT_OVERVIEW.md).

---

## Phase 4 doc alignment — decision coverage (D-05–D-35)

This pass maps **D-05 … D-35** to spec sections (themes from the former MVP validation matrix). Each ID is **addressed** if the spec now states implemented vs deferred behavior accurately.

| ID | Theme | Spec section(s) |
|----|--------|------------------|
| D-05 | Primary run flow (no legacy prompt) | [Runtime modes](#runtime-modes) |
| D-06 | CLI flags | [CLI flags](#cli-flags) |
| D-07 | Environment variables | [Environment variables](#environment-variables) |
| D-08 | Config file + TOML keys | [Configuration](#configuration) |
| D-09 | Log directory and session files | [Logging](#logging) |
| D-10 | Persistent registry | [Registry](#registry) |
| D-11 | Scanning (walk, depth, exclude, workers) | [Scanning](#scanning) |
| D-12 | Dry-run (batch plan, no mutations) | [Runtime modes](#runtime-modes) |
| D-13 | `--fire` TUI | [Runtime modes](#runtime-modes), [UI](#ui) |
| D-14 | Confirmation / “prompt screen” | **Deferred** — see [Deferred / planned](#deferred--planned) |
| D-15 | Security notice (live runs) | [Runtime modes](#runtime-modes) |
| D-16 | Auto-commit when dirty | [Auto-commit and branches](#auto-commit-and-branches) |
| D-17 | Dual-branch staged / full backup | [Auto-commit and branches](#auto-commit-and-branches) |
| D-18 | Conflict handling (`new-branch` / `abort`) | [Conflict handling](#conflict-handling) |
| D-19 | Push modes | [Push modes](#push-modes) |
| D-20 | Per-repo overrides (`[[repos]]`) | [Configuration](#configuration) |
| D-21 | Per-host / global push limits | [Execution](#execution) |
| D-22 | SSH detection and `--status` | [Authentication](#authentication) |
| D-23 | Secret detection / block | [Safety](#safety) |
| D-24 | Command plugins (internals) | [Plugins](#plugins) |
| D-25 | Webhook plugins | **Deferred** — config type exists; loader not wired |
| D-26 | `--backup-to` / backup API mode | **Deferred** — flag returns error |
| D-27 | CI (build, vet, race tests) | [CI](#ci) |
| D-28 | Repository layout / packages | [Architecture](#architecture) |
| D-29 | Data model (high level) | [Core types (summary)](#core-types-summary) |
| D-30 | Worktrees | [Worktrees](#worktrees) |
| D-31 | HTTPS tokens for git | **Not implemented** for push auth |
| D-32 | UI color profiles / fire animation | [UI](#ui) |
| D-33 | `disable_scan` / `--no-scan` | [Scanning](#scanning) |
| D-34 | `push_workers` / `scan_workers` | [Scanning](#scanning), [Execution](#execution) |
| D-35 | `cache_ttl` vs JSON repo cache file | [Configuration](#configuration) / [Scanning](#scanning) — no `repos-cache.json`; `cache_ttl` not wired to scan |

---

## CLI flags

Defined on the root command in `cmd/root.go` (see also `git-fire repos` subcommands in `cmd/repos.go`).

| Flag | Purpose |
|------|---------|
| `--dry-run` | Collect repos, build plan, print summary; **no** git mutations. |
| `--fire-drill` | Alias for `--dry-run`. |
| `--fire` | Bubble Tea repo selector; repos stream in as discovered; then same backup pipeline for selection. |
| `--path <dir>` | Scan root (overrides `global.scan_path` for this run). Default `.` |
| `--skip-auto-commit` | Sets `global.auto_commit_dirty` false for this run. |
| `--no-scan` | Skip filesystem walk; only known registry paths (plus configured behavior). Same as toggling `disable_scan` for this run. |
| `--init` | Write example config to `~/.config/git-fire/config.toml` (or prompt unless `--force`). |
| `--force` | With `--init`, overwrite config without prompting. |
| `--config <file>` | Explicit config file path. |
| `--status` | SSH / key / agent snapshot and repo-oriented status; no backup run. |
| `--backup-to <url>` | **Not implemented** — returns error (planned v0.2+). |

Root invocation: `git-fire` with no subcommand runs the default streamed backup flow.

**Not implemented** (do not document as available): `--full-scan`, `--reindex`, `--token`, `--platform` (as flags), `--auth-check`, `--quiet`, `-v` / `--verbose`, per-key SSH passphrase flags from the old design doc.

---

## Environment variables

- **`GIT_FIRE_API_TOKEN`** — If set, overrides `backup.api_token` after load (for future backup features; backup mode is not active in the default CLI path).
- **`GIT_FIRE_SSH_PASSPHRASE`** — If set, overrides `auth.ssh_passphrase` after load.

**Viper** (`internal/config/loader.go`): prefix `GIT_FIRE`, nested keys use `_` for `.` (e.g. `GIT_FIRE_GLOBAL_SCAN_PATH` for `global.scan_path`).

**Not used:** There is **no** `GIT_FIRE_CONFIG` binding. Choose config path with **`--config`**.

---

## Configuration

- **Default file:** `~/.config/git-fire/config.toml` (optional; defaults apply if missing).
- **System path:** `/etc/git-fire/config.toml` is on the search path but optional.
- **Explicit file:** `--config` (required to exist if passed).

See `internal/config/types.go` and `config.ExampleConfigTOML()` in `defaults.go` for the exact TOML shape.

### `global`

| Key | Meaning |
|-----|---------|
| `default_mode` | `push-known-branches`, `push-all`, `push-current-branch`, or `leave-untouched`. |
| `conflict_strategy` | `new-branch` or `abort` (not `skip`). |
| `auto_commit_dirty` | bool |
| `block_on_secrets` | bool |
| `scan_path` | default directory root for scanning |
| `scan_exclude` | path segments to skip while walking |
| `scan_depth` | max directory depth |
| `scan_workers` | parallel scan analysis workers |
| `push_workers` | worker count for repo execution |
| `cache_ttl` | Parsed in config; **not** currently passed from `cmd/root.go` into `git.ScanOptions` / unused by the walk (reserved). No separate JSON repo list file. |
| `rescan_submodules` | re-walk known repos for new submodules |
| `disable_scan` | when true, no filesystem walk; registry-known paths only |

**Not in config (do not add to templates from old spec):** `branch_name_template`, `push_to_all_remotes`, `preferred_remotes`, `quick_scan_paths`, `full_scan_root`, `repos_cache_file`, `[logging]` section — those were design-only or unimplemented.

### `ui`

| Key | Meaning |
|-----|---------|
| `show_fire_animation` | Fire layer in TUI (also toggled with `f` in session). |
| `fire_tick_ms` | Bubble Tea tick interval; clamped 30–60000 ms on load. |
| `color_profile` | `classic`, `synthwave`, `forest`, `arctic`. |

### `backup`

Structured for future “new remote” backup; **creating repos / manifest / target push** is **not** wired in the default CLI. Keys include `target_remote`, `platform`, `api_token`, `repo_template`, `organization`, `generate_manifest`.

### `auth`

`ssh_passphrase`, `use_ssh_agent`.

### `plugins`

`enabled`, `[[plugins.command]]`, `[[plugins.webhook]]` — see [Plugins](#plugins).

### `[[repos]]`

Per-repo overrides: `path` (glob), `remote` (substring), `mode`, `skip_auto_commit`, `rescan_submodules`.

---

## Runtime modes

| Mode | Behavior |
|------|----------|
| **Default** | `ScanRepositoriesStream` feeds the executor as repos appear; registry upsert; pushes run with worker pool. |
| **`--dry-run`** | Full scan (or registry-only if scan disabled), then plan summary; no modifying git operations. |
| **`--fire`** | Stream repos into TUI; user selects repos; then planning/execution for selection. |

On non-dry live runs, `safety.SecurityNotice()` is printed once before work begins.

There is **no** “Is the building on fire?” prompt, countdown, or YES/NO gate in the current CLI.

---

## Registry

- **Path:** `~/.config/git-fire/repos.toml` (see `internal/registry`).
- Every discovered repo is upserted; entries use **absolute** paths.
- **Opt-out:** repos are active unless marked ignored in the registry.
- **`git-fire repos`** — list, scan, remove, ignore, unignore subcommands.

---

## Scanning

- Walk starts at `global.scan_path` (or `--path`), respects `scan_depth`, `scan_exclude`, and `scan_workers`.
- **`--no-scan` / `disable_scan`:** filesystem walk skipped; known paths from registry still processed.
- **No** `~/.config/git-fire/repos-cache.json` quick-scan list; persistence is **`repos.toml`**, not a separate JSON cache file from the legacy spec.

---

## Auto-commit and branches

Dirty repos (when `auto_commit_dirty` is true and not blocked by secrets):

- Uses **dual-branch** strategy in `internal/git` (`AutoCommitDirtyWithStrategy`): e.g. `git-fire-staged-*` and `git-fire-full-*` backup branches, then working tree restored where applicable.
- Commit messages include ISO8601 timestamps.

---

## Conflict handling

- **`new-branch`:** create/push `git-fire-backup-*` style branches when divergence is detected (planner/runner + `internal/git`), no user-editable branch template in config today.
- **`abort`:** skip pushing when conflicts would require unsafe behavior.

---

## Push modes

Resolved per repo from `default_mode` and `[[repos]]` overrides:

- `leave-untouched` — skip.
- `push-known-branches` — push branches that exist on the remote.
- `push-all` — push all local branches.
- `push-current-branch` — push only the checked-out branch.

Remotes: planner schedules pushes across **each** configured remote on the repo (not “origin only” unless others are absent).

**Deferred:** `preferred_remotes` / “push_to_all_remotes false” style ordering is **not** configurable.

---

## Execution

- **Planner:** `internal/executor/planner.go` — builds per-repo actions.
- **Runner:** `internal/executor/runner.go` — runs actions, structured logging, **HostLimiter** (`internal/executor/ratelimit.go`) for global and per-host concurrency (including conservative defaults for `github.com`).

---

## Logging

- **Directory:** `~/.cache/git-fire/logs/` (`executor.DefaultLogDir()`).
- **Files:** `git-fire-YYYYMMDD-HHMMSS.log`, one per run session.
- **Format:** JSON lines (`internal/executor/logger.go`); entries sanitized via `internal/safety` where applicable.

---

## Authentication

- **`git-fire --status`:** SSH key discovery, encrypted-key detection, agent status (see `internal/auth`, `cmd/root.go`).
- Git HTTPS credential helpers / `GIT_FIRE_GITHUB_TOKEN` style push auth: **not** implemented for git operations.

---

## Safety

- Pattern-based secret detection can **block** auto-commit/push when `block_on_secrets` is true (`internal/safety`).

---

## UI

- Bubble Tea selector in `--fire`; fire animation and `color_profile` per `internal/ui`.
- **Deferred:** dedicated prompt screen, separate “completion report” screen, and rich push-progress UI from the legacy design doc — current UX is TUI + console output.

---

## Plugins

- **Command plugins:** implemented in `internal/plugins` (command execution, triggers, template vars). **`LoadFromConfig` is not invoked from the default `git-fire` root path** — CLI auto-loading is **planned (v0.2)** per [PLUGINS.md](PLUGINS.md).
- **Webhook plugins:** TOML types exist; `loader.go` has a TODO — **deferred**.
- **Go `.so` plugins:** removed from roadmap (see PLUGINS.md).

---

## Worktrees

Worktrees are discoverable (`internal/git`); treat parallel execution across worktrees as **best-effort** — not all historical matrix rows are E2E guaranteed.

---

## CI

- **ci.yml** (on push/PR): `go build ./...`, `go vet ./...`, `go test -race -count=1 ./...`.
- **golangci-lint** job is commented out (v2 migration).
- **release.yml:** manual `workflow_dispatch` for tagging/releases.

---

## Architecture

```text
main.go
└── cmd/
    ├── root.go      # root flags, run routing, stream/batch/TUI
    └── repos.go     # registry subcommands
internal/config      # Viper/TOML, validation
internal/registry    # repos.toml persistence
internal/git         # scan + native git exec
internal/executor    # planner, runner, logger, rate limiter
internal/auth        # SSH
internal/safety      # secrets, redaction
internal/plugins     # command plugins (load from config not wired on default path)
internal/ui          # Bubble Tea
internal/testutil    # tests
```

All git operations shell out to the **`git`** binary — no go-git.

---

## Core types (summary)

Authoritative definitions live in `internal/git/types.go`, `internal/executor/types.go`, `internal/config/types.go`. The legacy spec’s long pseudo-Go blocks mixed **planned** fields with shipped types; prefer **godoc / source** over those excerpts.

---

## Deferred / planned

| Item | Notes |
|------|--------|
| `--backup-to` | Flag present; returns error until implemented. |
| Backup-to-new-remote (API create repo, manifest, etc.) | `[backup]` config is forward-looking. |
| Interactive fire prompt + countdown | Not in codebase. |
| `GIT_FIRE_CONFIG` | Not bound; use `--config`. |
| JSON `repos-cache.json`, quick-scan-only paths | Replaced by registry + full walk under `scan_path`. |
| Webhook plugin loading | TODO in `plugins/loader.go`. |
| Plugin auto-run from config on default CLI | v0.2 target. |
| `preferred_remotes` / remote subset toggles | Not implemented. |

---

## Hypothetical product directions (not specifications)

The items below are **exploratory roadmap notes**. They do **not** define current behavior, MVP scope, or acceptance criteria until promoted into an explicit phase or section of this document.

### General non-emergency mode

- A **general / everyday** entry point framed for routine use: sync, backup, or “save my work” without emergency metaphors, fire-drill wording, or dramatic prompts.
- Could reuse the same scanner, registry, planner, and executor but offer calmer copy, different defaults, and workflows tuned for non-panic use (e.g. quieter progress, confirmation rules appropriate for casual runs).

### Lazy uploads (product framing)

- Position multi-repo **commit-and-push** as a **low-friction** action: one invocation covers many repositories when the user does not want to open each repo and run git manually.
- Treated primarily as **positioning and UX** around existing capabilities unless a separate mode is specified later.

### Automated git pushes

- **User-orchestrated automation (pattern):** Invoke `git-fire` from **cron**, **systemd timers**, other OS schedulers, or **tool/editor/agent hooks** so remotes stay current without a manual `git push` per repository. Document operational expectations when this is promoted (SSH agent, non-interactive runs, logging, exit codes, failure alerts).
- **First-class automation (hypothetical):** Future flags, subcommands, or config (e.g. schedule definitions, a small supervisor, or “unattended profile”) remain undefined until designed; this spec does not commit to a shape.

### Standalone git integration test library

- In-tree helpers that drive the **real `git` binary** to build temporary repos for tests may be **extracted as an open-source Go module** for reuse by other projects. Licensing, module path, and attribution would be decided at extraction time; see repository README and `internal/testutil` for context.
