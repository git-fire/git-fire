# CLAUDE.md — git-fire

## Project Overview

`git-fire` is an emergency Go CLI tool that backs up all dirty local git repositories to their remotes with a single command. It is the OSS successor to `qw3rtman/git-fire`, targeting developer audiences on HN and Reddit.

Module: `github.com/TBRX103/git-fire`
Go version: 1.24.2

---

## Commands

```bash
make build       # compile binary to ./git-fire
make run ARGS="--dry-run"  # build and run with flags
make test        # run all tests
make test-race   # run tests with race detector (used in CI)
make lint        # go vet ./...
make install     # install to $GOPATH/bin
make clean       # remove binary
```

Run directly:

```bash
go build ./...
go test -race -count=1 ./...
go vet ./...
```

---

## Architecture

```
main.go
  └── cmd/root.go            # Cobra CLI: flags, orchestration
      ├── internal/config    # Load config (~/.config/git-fire/config.toml)
      ├── internal/auth      # SSH key detection and agent status
      ├── internal/git       # Repo scanning + git operations (shells out to git binary)
      ├── internal/executor  # Plan builder + runner + rate limiter + JSON logger
      ├── internal/safety    # Secret pattern detection (warns, does not block)
      ├── internal/plugins   # Command/webhook plugin system
      ├── internal/ui        # Bubble Tea TUI: interactive repo selector + fire animation
      └── internal/testutil  # Shared test helpers, fixtures, scenario builders
```

**Key design decisions:**
- Uses native `git` binary via `exec.Command` — not go-git. Do not change this.
- Repo scanning is parallel (goroutine pool). Pushing is sequential to avoid SSH contention.
- Rate limiter caps concurrent pushes to the same host at 2.
- All operations are logged as structured JSON lines under `~/.cache/git-fire/logs/` (session files `git-fire-*.log`); user config lives in `~/.config/git-fire/`.
- `internal/ui` has no tests — Bubble Tea TUI testing is deferred intentionally.

**Registry invariant (opt-out model):**
Every repo git-fire discovers is immediately upserted into the persistent registry (`~/.config/git-fire/repos.toml`, beside `config.toml`; legacy `~/.git-fire/repos.toml` is migrated on first load) and the registry is saved before the run ends. Backup is opt-out — all `active` repos are backed up by default; users explicitly set a repo to `ignored` to exclude it. Registry entries persist their absolute paths, so repos found from one working directory are included in future runs from any directory.

**Scan→backup pipeline (non-`--fire` live runs):**
`cmd/root.go` uses `git.ScanRepositoriesStream` to pipeline scanning and backup: as soon as a repo is discovered it is upserted into the registry and queued for backup via `executor.Runner.ExecuteStream`. Backup workers block when the queue is temporarily empty rather than waiting for the full scan to complete. `--fire` (TUI) and `--dry-run` modes still collect the full repo list first, since the TUI and plan summary both need it.

---

## Testing

**Target: 80% coverage or better on all non-UI packages.**

- Write tests for every new function in `internal/` packages (except `internal/ui`).
- Use `internal/testutil` helpers — `fixtures.go` for temp repos, `scenarios.go` for complex multi-repo setups.
- Always run `make test-race` before considering a change done; the CI pipeline uses `-race`.
- Prefer table-driven tests (`t.Run` subtests) for functions with multiple input/output cases.
- Integration-style tests that shell out to `git` are fine and preferred for `internal/git` — do not mock the git binary.
- Coverage gaps in `internal/ui` are acceptable and expected.

---

## Conventions

- **No go-git**: all git interactions shell out to the system `git` binary.
- **Cobra for CLI**: flags and subcommands live in `cmd/`. Do not add business logic there.
- **Config via Viper/TOML**: user config at `~/.config/git-fire/config.toml`; env vars override.
- **Error handling**: return errors up to the caller; only `log.Fatal`/`os.Exit` in `main.go` or `cmd/`.
- **No global state** outside of the config loader.
- **Bubble Tea `Update`** returns `(Model, tea.Cmd)` — no error in the signature. Handle errors by embedding them in the model.
- Plugin execution is non-fatal: log and continue.

---

## CI

- `.github/workflows/ci.yml`: build + vet + `go test -race` on every push and PR.
- `.github/workflows/release.yml`: manual trigger; builds binaries for 8 platforms (Linux amd64/arm64/armv6, macOS amd64/arm64, Windows amd64/arm64/386) and creates a GitHub Release.
- golangci-lint is configured but currently disabled in CI (v2 migration in progress — do not enable without checking).
