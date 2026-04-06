# AGENTS.md

## Cursor Cloud specific instructions

### Project overview

`git-fire` is a pure Go CLI tool (no servers, databases, or containers). It shells out to the system `git` binary for all git operations. See `CLAUDE.md` for full architecture, commands, and conventions.

### Prerequisites

- **Go 1.24.2** and **git** must be on `PATH`. No other system dependencies.
- Go module dependencies are fetched automatically on `go build` / `go test`.

### Common commands

All standard dev commands are in the `Makefile` and documented in `CLAUDE.md`:

| Task | Command |
|------|---------|
| Build | `make build` |
| Lint | `make lint` |
| Test | `make test` |
| Test (CI, with race detector) | `make test-race` |
| Run with flags | `make run ARGS="--dry-run"` |

### Non-obvious caveats

- `internal/executor` and `internal/git` test suites create many temporary git repos and take ~3-4 minutes each with `-race`. Total `make test-race` takes ~4 minutes.
- Tests do **not** require network access or remote git hosts — all tests use local bare repos created via `internal/testutil` fixtures.
- The binary is built to `./git-fire` in the repo root by `make build`. It is `.gitignore`d.
- Config lives at `~/.config/git-fire/config.toml`; generate a template with `./git-fire --init`.
- The repo registry at `~/.config/git-fire/repos.toml` persists discovered repos across runs. This file is auto-created on first run.
- `internal/ui` has no tests by design (Bubble Tea TUI testing is deferred).
- golangci-lint v2 migration is in progress — do not enable it in CI without checking compatibility first.

### Release automation caveat

- Homebrew automation is configured via `.goreleaser.yaml` and `.github/workflows/release.yml`, using `HOMEBREW_TAP_TOKEN` for writes to `git-fire/homebrew-tap`.
- If `HOMEBREW_TAP_TOKEN` is missing, release automation intentionally skips Homebrew with `--skip=homebrew` so release publishing still succeeds.
