# AGENTS.md

<!-- agentic-guardrails:start -->
## Agentic guardrails

These notes apply to humans and to automated agents (including Cloud Agents).

1. **Autonomous work comes first**  
   The default posture is to **finish useful work without unnecessary human gates**. Prefer actions that keep momentum (fetch, build, test, open PRs) over asking for permission for routine steps—while still honoring the hard safety rules below.

2. **Multi-repository work is encouraged**  
   When a task genuinely spans more than one codebase (services, libraries, docs, sibling CLIs), work across **multiple repositories or local clones** as needed. Open **one PR per repository**, keep each PR scoped, and **link related PRs** in every description so reviewers see the coordinated change.

3. **Stay on a fresh enough tip; rebase is optional**  
   Before substantive work: `git fetch origin`, check out your branch, then prefer **`git merge --ff-only origin/<branch>`** (or `git pull --ff-only` when upstream exists). If you cannot fast-forward, pause and use the repository’s normal merge or rebase workflow—**do not** silently work on a stale checkout. **`git pull --rebase`** is *ideal* when updating an active feature branch, but it is **not required**; a merge commit or team-standard flow is fine when it avoids needless churn.

4. **Force-push is never automatic and needs explicit human buy-in**  
   Do **not** run `git push --force`, `git push --force-with-lease`, or rewrite published history on your own. If you believe it might be warranted, **stop** and give the **human** explicit **reasoning**, **effects** on collaborators, CI, and open PRs, and **why** a force-push would be needed versus safer alternatives (new branch, revert, merge). Proceed **only** after they **explicitly approve** that exact repository and branch.

5. **Focused changes and verification**  
   Keep pull requests focused; run this repository’s standard build, test, and lint commands (see `README`, `Makefile`, or `CLAUDE.md`) before requesting review.

6. **Workflow shape is yours**  
   Using **git worktree** versus a single working directory is an **operator choice**; these docs do **not** require worktrees.

<!-- agentic-guardrails:end -->

---


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
