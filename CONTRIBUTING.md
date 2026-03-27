# Contributing to git-fire

Thanks for your interest in contributing!

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
```

All tests must pass before submitting a PR.

## Submitting a PR

- Keep PRs focused — one fix or feature per PR.
- New behaviour must include tests. Bug fixes should include a regression test.
- Run `go vet ./...` locally before pushing; CI will enforce this too.
- Write a clear PR description explaining *why*, not just *what*.

## Package Overview

| Package | Purpose |
|---|---|
| `cmd/` | Cobra CLI entry point and flag handling |
| `internal/git` | Repository scanning and git operations (commit, push, branch) |
| `internal/executor` | Execution planner, runner, rate limiter, and structured logger |
| `internal/safety` | Secret detection — pattern matching and filename heuristics |
| `internal/auth` | SSH key discovery and ssh-agent management |
| `internal/config` | TOML config loading, defaults, and validation |
| `internal/ui` | Bubble Tea TUI (repo selector, fire background animation) |
| `internal/plugins` | Plugin system — command execution and registry |
| `internal/testutil` | Shared test helpers: repo fixtures, scenarios, snapshots |

## Reporting Issues

Open a GitHub issue with a minimal reproduction case. For security issues, please email the maintainer directly rather than opening a public issue.
