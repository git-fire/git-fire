# Alpha regression validation — 2026-02-10

## Scope

- **Branch:** `main` (synced with `origin/main` at time of run)
- **Automated:** `go vet ./...`, `go test ./... -race -count=1`
- **Manual / TUI:** Not exercised in this environment (no reliable interactive TTY for Bubble Tea). Core logic is covered by unit/integration tests under `internal/*` and `cmd/*`.

## Automated results

| Check | Result |
|-------|--------|
| `go vet ./...` | **PASS** |
| `go test ./... -race -count=1` | **PASS** (executor + git packages ~3.5m wall time) |

## Defect found and fixed

| ID | Severity | Description | Status |
|----|----------|-------------|--------|
| **INIT-001** | High (UX / docs mismatch) | `git-fire --init --config /path/to/file.toml` ignored `--config` and always wrote to the default XDG/user config path. Users following “explicit config” workflows would see success text for the wrong path and subsequent runs with `--config` would fail (file missing). | **Fixed** in `handleInit()`: when `configFile` is set, init writes and conflict checks use that path. Regression test: `TestHandleInit_UsesExplicitConfigPath`. |

## Video / screen capture

No screen recordings were produced: this run did not include interactive TUI validation. For alpha sign-off, recommend a short manual recording of: `--fire` selector, dirty repo flow, `--dry-run`, and `repos` subcommands on a real machine.

## Alpha readiness assessment

**Recommendation:** **Conditional go** for alpha — ship with release notes calling out known limitations below; complete a short manual TUI smoke pass before tagging.

### Strengths

- Broad automated coverage across config, git operations, executor, registry, plugins, safety/redaction, and UI helpers (non-TTY tested).
- Race detector clean on full suite.

### Known gaps (product / roadmap — not regressions)

- `--backup-to` is documented but returns “not yet implemented” (v0.2).
- Plugin CLI auto-loading called out as not fully wired in README.
- End-to-end human validation of the TUI (`--fire`) still required for alpha confidence.

## Follow-up plan

1. **Ship INIT-001 fix** with the next release or patch (included in this commit).
2. **Manual alpha checklist** (15–20 min): `--init` default + `--config` custom path; `--status`; `--dry-run` on a small multi-repo tree; `repos list/add/remove`; `--fire` selection and confirm no panic on resize.
3. **Done:** Subprocess guardrail `TestCLI_InitHonorsConfigFlag` in `cmd/cli_integration_test.go` (skipped with `go test -short`).
4. **Track:** Implement or hide `--backup-to` until implemented to avoid user confusion post-alpha.
