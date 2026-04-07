# Project Overview

`git-fire` is an emergency multi-repo Git backup CLI inspired by "in case of fire."

Core value: in a panic (or at end-of-day), run one command to discover repositories, auto-commit dirty work, and push backups to remotes so local-only work is not lost.

## Basics

- **Project:** `git-fire`
- **Repository:** `github.com/git-fire/git-fire`
- **Language:** Go 1.24.2
- **License:** MIT
- **Status:** Alpha

## User Promise

- Fast, low-friction, low-config backup across many repositories.
- Safety-first defaults (no force-push in normal flows).
- Useful for emergency recovery and routine daily sync.

## Primary Commands

- `git-fire` (default streamed backup flow)
- `git fire` (Git subcommand alias support)
- `git-fire --dry-run` (plan only, non-destructive)
- `git-fire --fire` (TUI selector + animation mode)
- `git-fire --path <dir>` (scan specific root)
- `git-fire --skip-auto-commit` (push existing commits only)
- `git-fire --status` (auth/repo status)
- `git-fire --init` (generate config template)
- `git-fire repos …` (list / scan / ignore / unignore / remove registry entries)

## Core Functionality

### 1) Repository Discovery and Scanning

- Scans filesystem roots for Git repositories.
- Uses parallel scanning for speed.
- Supports streaming discovery in default live mode so backup can start before the full scan completes.

### 2) Persistent Repository Registry (Opt-out)

- Registry location: `~/.config/git-fire/repos.toml`
- Every discovered repository is upserted and persisted.
- Backups are opt-out: active repos are included by default; users explicitly mark ignored repos.
- Registry stores absolute paths so behavior is stable across working directories.

See [REGISTRY.md](REGISTRY.md).

### 3) Scan-to-Backup Pipeline

- **Live run (default):** streamed scan results feed executor queue as repos are found.
- **`--fire`:** discovered repos stream into the TUI selector progressively.
- **`--dry-run`:** full collection first, then plan summary.

### 4) Auto-commit and Branch Strategy

- Dirty repositories can be auto-committed unless disabled.
- Backup branch strategy may create `git-fire-staged-*` and `git-fire-full-*`.
- Conflict strategy supports a new-branch approach when divergence is detected.

### 5) Push Execution Model

- Repository execution is parallelized with configurable worker concurrency (`global.push_workers`, default `4`).
- Per-host rate limiting is applied during push actions to avoid overloading a single remote host.
- Default host limits are conservative for common providers (for example `github.com` is capped lower than generic hosts).

### 6) Safety Model

- Normal flows avoid force pushes.
- Conflict safety branches (`git-fire-backup-*`) are used for `push-current-branch` conflict flows.
- Dry-run mode supports preflight verification.
- Secret-pattern detection blocks by default (configurable).

See [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md).

### 7) UX and TUI

- Interactive repository selector.
- Fire-themed terminal UX with color profiles: `classic`, `synthwave`, `forest`, `arctic`.
- Configurable under `[ui]` in TOML.
- Custom hex palettes are planned, not yet shipped.

### 8) Logging and Observability

- Structured JSONL logs in `~/.cache/git-fire/logs/`
- Session files: `git-fire-*.log`
- User config path: `~/.config/git-fire/config.toml`

### 9) Extensibility and Plugins

- Command-plugin scaffolding exists, but default CLI auto-loading is not yet wired.
- Webhook plugin loading is planned and not implemented yet.
- Plugin execution is non-fatal (errors are logged and run continues).
- Typical use cases: object storage sync, notifications, archive steps.

See [../PLUGINS.md](../PLUGINS.md).

## Architecture

- `main.go`
- `cmd/root.go` (Cobra orchestration and flags)
- `internal/config` (Viper/TOML config loading)
- `internal/auth` (SSH key and agent checks)
- `internal/git` (repo scan + native git operations through system `git`)
- `internal/executor` (planner/runner/rate limiting/structured logging)
- `internal/safety` (secret warnings)
- `internal/plugins` (plugin execution layer)
- `internal/ui` (Bubble Tea TUI)
- `internal/testutil` (fixtures and scenario builders)

## Design Constraints

- Must shell out to native `git` (`exec.Command`); no `go-git`.
- `cmd/` should not contain business logic.
- Errors should bubble up; only CLI entry paths should terminate process.
- Avoid global mutable state beyond configuration loading.
- Plugin failures should never crash a run.

## Testing and Quality Posture

- 250+ tests (see README badge).
- CI runs build, vet, and race tests.
- Coverage is tracked per package with a risk-based focus, not a single global gate.
- UI testing remains intentionally limited compared to non-UI packages.
- Integration tests using real `git` are preferred over mocking.
- Manual smoke fixtures for OSS testers live in `scripts/setup-manual-smoke-fixtures.sh`, `scripts/setup-manual-smoke-stages.sh`, and `scripts/run-manual-smoke-stage.sh`.
- Future enhancement target: add stage outcome verification script for pass/fail assertions after manual runs.

### Post-Release OS Polish Backlog (Low Risk)

- Keep Unix-first shell script assumptions for manual smoke tooling, while documenting expected behavior for non-Unix users.
- Consider expanding config/cache path tests around Windows `USERPROFILE`/`APPDATA` edge cases in isolated test environments.
- Revisit non-critical path/layout fallback behavior for additional cross-OS consistency after launch feedback.

## Maturity and Risk

- Alpha, but stable for many common flows.
- Not intended to be a sole backup system yet.
- Users should run dry-runs, verify results, and keep independent backup layers.

## Roadmap Direction

### Near-term (Beta Window)

- Expand tester pool and feedback loops.
- Prioritize stabilization and critical bug fixes.
- Improve packaging and distribution reliability.

### Beta

- Broaden package manager publication.
- Tighten edge-case handling and operational confidence.
- Improve onboarding and safe-usage documentation.

### 1.0 (timing depends on alpha feedback)

- Close beta-critical issues.
- Ship stable production-ready core flows.
- Harden reliability and conflict handling semantics.

## Review Focus Areas

For architecture and release readiness reviews, prioritize:

1. Scan -> plan -> execute boundaries and package cohesion.
2. Safety guardrails around divergence/conflict states.
3. Performance opportunities for large multi-repo runs.
4. Onboarding clarity for first-time users.
5. Coverage gaps in critical Git edge-case scenarios.
