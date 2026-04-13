# Contributing to git-fire

Thanks for your interest in contributing!

Your feedback, feature requests, and ideas are genuinely wanted — no idea is too small or too ambitious to discuss. If you're thinking about forking to add something, please open an issue first. The goal is a tool the community actually relies on, and I'd rather build it with you than have it fragment. Open an issue, open a PR, or just tell me what you need.

For project orientation, start with [README.md](README.md) and the docs hub at [docs/README.md](docs/README.md). Detailed behavior expectations, edge cases, and validation targets are documented in [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md); user-facing summaries and shipped code on `main` are the practical source of truth when wording drifts during beta.

## Prerequisites

- Go 1.24.2 or later
- `git` in your PATH

## Build & Test

```bash
# Build all packages (matches what CI checks)
go build ./...

# Build the CLI binary
go build -o git-fire .

# Run all tests
go test ./...

# Run tests with race detector (used in CI)
go test -race -count=1 ./...

# Vet
go vet ./...

# One-shot local parity with CI (build, UAT, vet, race tests, plugin contract; optional tools if installed)
./scripts/validate.sh
# or: make validate

# MVP UAT — end-to-end scenarios with local bare remotes (needs ./git-fire from make build)
bash scripts/uat_test.sh
# or: make uat  /  ./scripts/uat.sh
# Verbose pre-flight (pwd, CI env, git/git-fire versions): GIT_FIRE_VERBOSE=1 bash scripts/uat_test.sh
```

All tests must pass before submitting a PR.

## Submitting a PR

- Keep PRs focused — one fix or feature per PR.
- State what change you are making and why it is needed.
- If the PR resolves a bug, include reproducibility steps when feasible (especially for complex issues).
- All code changes must include tests. Bug fixes should include regression coverage.
- Run `go vet ./...` locally before pushing; CI will enforce this too.
- Write a clear PR description explaining *why*, not just *what*.

## Maintainer

Maintainers are listed on the GitHub repository: `github.com/git-fire/git-fire`.

## Package Overview

| Package | Purpose |
|---|---|
| `cmd/` | Cobra CLI entry point and flag handling |
| `github.com/git-fire/git-harness/git` | Repository scanning and git operations (commit, push, branch); vendored as a module dependency |
| `internal/executor` | Execution planner, runner, rate limiter, and structured logger |
| `github.com/git-fire/git-harness/safety` | Secret detection — pattern matching and filename heuristics; module dependency |
| `internal/auth` | SSH key discovery and ssh-agent management |
| `internal/config` | TOML config loading, defaults, and validation |
| `internal/ui` | Bubble Tea TUI (repo selector, fire background animation) |
| `internal/plugins` | Plugin system — command execution and registry |
| `internal/testutil` | Shared test helpers: repo fixtures, scenarios, snapshots |

## Current Beta Limitations

- Command plugin auto-loading from config is now shipped: plugins defined under `[[plugins.command]]` in config.toml are loaded and executed automatically after each run.
- `--backup-to` is exposed but not yet implemented (`v0.2` target).
- Webhook/reference plugin execution paths are planned but not implemented in the runtime path yet (`v0.2` target).

## Reporting Issues

Open a GitHub issue with a minimal reproduction case. For security issues, do not open a public issue; use the private reporting path in [SECURITY.md](SECURITY.md).
