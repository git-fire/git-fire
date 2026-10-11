# AGENTS.md — git-fire

Guidance for **AI coding agents** (Cursor, Claude Code, Codex, Cloud Agents, etc.) and humans automating the CLI.

**Sibling project:** [git-rain](https://github.com/git-fire/git-rain) restores/syncs from remotes; use git-fire to **push checkpoints** and git-rain to **pull them back**. Keep automation flags aligned (`--non-interactive`, `--json`, env vars).

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

## What git-fire does

`git-fire` scans for git repositories (from a path and/or a persistent registry), optionally **auto-commits dirty work**, and **pushes** backup branches to remotes using configurable modes and conflict strategies. It shells out to the system **`git`** binary (via [git-harness](https://github.com/git-fire/git-harness)); it is not a hosted service.

**Invocation:** `git-fire` and `git fire` are equivalent when the binary is on `PATH`.

## When an agent should use it

| Situation | Recommended command |
|-----------|---------------------|
| End of session / before risky refactor | `git-fire --dry-run --json --non-interactive --path <root>` then ask the human before a live run |
| Push already-committed work only | `git-fire --skip-auto-commit --json --non-interactive --path <root>` |
| Inspect auth + repo dirtiness | `git-fire --status --json --non-interactive --path <root>` |
| Human at keyboard choosing repos | `git-fire --fire` (interactive TUI only) |

**Do not** use `git-fire --fire` from unattended agents (requires a real TTY).

**Prefer** scoping with `--path` to the user’s workspace instead of scanning the entire home directory.

## Install

```bash
# Recommended: latest release
go install github.com/git-fire/git-fire@latest

# macOS / Linuxbrew
brew install git-fire/tap/git-fire

# Linux install script
curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/install.sh | bash
```

Ensure `git-fire` is on `PATH` (`go install` → `$HOME/go/bin`). Requires **Go 1.24.2+** and **git** on `PATH`.

Generate default config: `git-fire --init` (use `--force` when overwriting non-interactively).

## Commands (copy-paste)

```bash
# Preview plan (no git mutations) — default for agents
git-fire --dry-run --json --non-interactive --path .

# Live checkpoint (COMMITS dirty repos + PUSHES) — needs human approval
git-fire --json --non-interactive --path .

# Push committed refs only (no auto-commit)
git-fire --skip-auto-commit --json --non-interactive --path .

# Status: SSH keys/agent + discovered repos
git-fire --status --json --non-interactive --path .

# Registry-only run (no filesystem walk)
git-fire --no-scan --dry-run --json --non-interactive

# Registry maintenance
git-fire repos list
git-fire repos ignore /absolute/path/to/repo
git-fire repos scan ~/projects

# Version / help
git-fire --version
git-fire --help
```

Machine-readable schemas: [docs/JSON_OUTPUT.md](docs/JSON_OUTPUT.md).

## Non-interactive behavior

Set **`--non-interactive`** or **`GIT_FIRE_NON_INTERACTIVE=1`** (also set automatically when `CI` or `GITHUB_ACTIONS` is set):

- No stdin prompts for config overwrite (`--init` needs `--force` if config exists).
- Stale registry locks are removed automatically; **live** locks require `--force-unlock-registry` (dangerous if another instance is running).
- After backups finish, a slow scan is **cancelled** instead of waiting for Enter.
- Git network operations use **`GIT_TERMINAL_PROMPT=0`** (via git-harness) so git does not block on password prompts.

**Aliases:** `--yes` is the same as `--non-interactive`.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success, user abort (`--fire` cancel), or intentional no-op (e.g. no repos / dry-run complete) |
| `1` | Error (missing git, config failure, push failures, plugin `fail_run`, invalid flags, etc.) |

The process prints `Error: ...` on stderr and exits `1` on failure (`main.go`).

## Safety notes

| Action | Destructive? | Notes |
|--------|----------------|-------|
| `--dry-run` / `--fire-drill` | No | Plans and secret warnings only; no git mutations |
| `--status` | No | Read-only inspection |
| Default live `git-fire` | **Yes** | May **create commits** on dirty repos and **push** to remotes |
| `--skip-auto-commit` | Push only | Does not commit uncommitted changes |
| `--fire` TUI | **Yes** if user confirms | Interactive selection then same push pipeline |
| `git-fire repos ignore` | Local registry | Stops future backups for that path |
| `--init --force` | Overwrites config file | Does not touch git repos |
| `--force-unlock-registry` | Risky | Can corrupt registry if another process holds the lock |

- Normal flow avoids **force-push**; divergence may create `git-fire-*` backup branches (see [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md)).
- **Auth:** SSH agent / keys for SSH remotes; HTTPS remotes need credentials already configured (no interactive git prompts).
- **Secrets:** scanner warns on suspected secrets; committing secrets remains the user’s responsibility.
- **State:** writes `~/.config/git-fire/config.toml`, `repos.toml`, and logs under `~/.cache/git-fire/logs/`.

## Documentation map

- [llms.txt](llms.txt) — llmstxt.org index
- [README.md](README.md) — human-oriented quickstart
- [docs/agentic-flows.md](docs/agentic-flows.md) — hooks and workflows
- [docs/agent-integration.md](docs/agent-integration.md) — future MCP/skill design
- [CLAUDE.md](CLAUDE.md) — contributor architecture

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
