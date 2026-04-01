# Git Fire - Multi-Repo Checkpoint CLI

<p align="center">
  <img src="assets/git-fire-lockup.svg#gh-light-mode-only" alt="git-fire: flame and git node with wordmark" width="280" height="160">
  <img src="assets/git-fire-lockup-dark.svg#gh-dark-mode-only" alt="git-fire: flame and git node with wordmark" width="280" height="160">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/status-alpha-orange" alt="Status: alpha">
  <img src="https://img.shields.io/badge/tests-250%2B-brightgreen" alt="Tests: 250+">
  <img src="https://img.shields.io/badge/go-1.24.2-blue" alt="Go 1.24.2">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License: MIT">
</p>

> In case of fire:
> 1. `git-fire`
> 2. Leave building

`git-fire` is one command to checkpoint many repositories: discover, auto-commit dirty work (optional), and push backup branches/remotes with safety rails. It helps automate multi-repo push/checkpoint cycles for anyone who uses Git, from daily development to docs, data, and ops workflows.

Manual push loops can fail silently in real life (network drops, auth problems, or tool hiccups). `git-fire` gives you an auditable recovery path and more peace of mind when you need consistency across many repos.

Invocation note: `git-fire` and `git fire` are equivalent when `git-fire` is on your PATH.

### TUI screenshot

Current `git-fire` TUI: multi-repo selection, per-repo status, and one-screen checkpoint workflow.

![git-fire TUI screenshot showing repository selection and status view](assets/git-fire-tui-screenshot-gh.png)

## Alpha Status

`git-fire` is alpha software. Core multi-repo backup flows are usable today. Some roadmap items (plugin CLI auto-loading and `--backup-to`) are intentionally not wired yet.

## Quick Start

### Install

> **Coming soon:** Homebrew, Scoop, and packaged binary distribution are not published yet.

| Method | Command | Platform |
|---|---|---|
| Homebrew (coming soon) | `brew tap git-fire/homebrew-tap && brew install git-fire` | macOS / Linux |
| Scoop (coming soon) | `scoop bucket add git-fire https://github.com/git-fire/scoop-bucket && scoop install git-fire` | Windows |
| Go | `go install github.com/git-fire/git-fire@latest` | All (Go 1.24.2+) |
| Binary (coming soon) | [GitHub Releases](https://github.com/git-fire/git-fire/releases/latest) | All |

### First run

```bash
# preview first (safe)
git-fire --dry-run --path ~/projects

# run interactive checkpoint
git-fire
```

## Who Is This For

- **Anyone using Git across multiple repos:** you want one reliable checkpoint command before context switches, travel, maintenance, or riskier changes.
- **Developers and platform/infra engineers:** you maintain many code/IaC/config repos and want consistent, auditable bulk checkpoints.
- **Agent workflow users:** you run Claude/Cursor-style coding sessions and want a stop-hook safety net.
- **Security/ops practitioners:** you need fast state preservation before teardown, maintenance, or incident-driven system change.
- **Data/research/documentation teams using Git:** you track analysis, notebooks, or docs in many repos and need repeatable backup behavior.
- **Not the target:** single-repo users and monorepo teams that already have one-repo checkpoint discipline.

## Use Cases

### Daily developer checkpoint

- End of day
- Before context switch
- Before travel
- Before large refactor

### Non-developer multi-repo checkpoint

- Before publishing docs/content from multiple repositories
- Before data-analysis environment changes
- Before operational change windows where Git state should be preserved

### Creative and content workflows

- Keep many writing/media/site repos checkpointed before publishing
- Snapshot cross-repo changes before major editing or migration passes
- Standardize backup behavior for mixed technical and non-technical contributors

### Agent session safety net

- Run at session stop to avoid losing uncommitted agent output
- Keep logs for post-session review
- Use dry-run in guarded environments

See [docs/agentic-flows.md](docs/agentic-flows.md).

### IT/infra maintenance windows

- Bulk checkpoint tooling and config repos before maintenance
- Consistent push behavior across many repos
- Registry-backed repeatability across runs

### Security and operations workflows

- Red team session teardown
- Purple team exercise sync before debrief
- Incident response state preservation

See [docs/security-ops.md](docs/security-ops.md).

### Emergency hail mary

If your build is literally on fire, run `git-fire`.

## Integrations and Toolchains

`git-fire` can be integrated into your existing toolchains, IDE workflows, and automation hooks (for example session-stop hooks, task runners, CI helpers, or wrapper scripts).

If you want first-class support for a specific workflow or application, please open a feature request or submit a PR. We would love to support your use case.

## Roadmap Direction: Integrations + Redundancy Layers

Roadmap focus is practical integrations and emergency redundancy layers, especially for cases like SSH auth/key failures during high-pressure moments.

The goal is "paranoid and lazy" at the same time: set up layers once, then run one command when it counts.

## Key Features

- **One-command multi-repo checkpoint:** discover repositories and execute a repeatable backup flow from a single command.
- **Optional dirty-work auto-commit:** include uncommitted changes when you choose, or use `--skip-auto-commit` to push committed work only.
- **Safety-first conflict handling:** avoid force-push in normal flow and create backup branches when needed.
- **Dry-run planning:** preview exactly what would happen before making changes.
- **Auditable execution logs:** structured JSON logs make troubleshooting and post-run review practical.
- **Registry-backed repeatability:** discovered repos persist across runs so your workflow gets more reliable over time.

## Advanced Configuration and Behaviors

- **Persistent repo registry:** discovered repos are saved in `~/.config/git-fire/repos.toml`, so future runs include them unless explicitly ignored.
- **Status and auth checks:** `git-fire --status` gives a quick snapshot of SSH/auth and repo readiness before a full run.
- **Execution-mode control:** `--dry-run` for zero-side-effect planning, `--fire` for interactive selection, `--path` for scoped discovery.
- **Auto-commit strategy control:** choose whether dirty working trees are included with default behavior or skipped via `--skip-auto-commit`.
- **Session logging:** each run writes structured logs under `~/.cache/git-fire/logs/` for auditability and debugging.
- **Workflow composition:** combine with hooks, wrappers, task runners, or CI helper scripts for consistent team or solo automation.

## Feature to Use-Case Map

| Feature | Daily Dev | Agentic | IT/Infra | Red Team | Emergency |
|---|---|---|---|---|---|
| Parallel multi-repo execution | ✅ | ✅ | ✅ | ✅ | ✅ |
| Persistent repo registry | ✅ | ✅ | ✅ | ✅ | ✅ |
| Dry-run planning | ✅ | ✅ | ✅ | ✅ | ✅ |
| Secret detection warnings | ✅ | ✅ | ✅ | ✅ | ✅ |
| Structured JSON logs (`~/.cache/git-fire/logs/`) | ⚪ Optional | ✅ | ✅ | ✅ | ⚪ Optional |
| `--status` SSH/repo snapshot | ✅ | ✅ | ✅ | ✅ | ⚪ Optional |
| Conflict-safe backup branches (no force push in normal flow) | ✅ | ✅ | ✅ | ✅ | ✅ |
| Plugin internals (`v0.2` CLI auto-loading target) | 🔜 | 🔜 | 🔜 | 🔜 | 🔜 |

## Why It Is Trustworthy in Alpha

- No force push in normal flows.
- Conflict strategy creates backup branches (`git-fire-backup-*`) when needed.
- Dry-run gives a no-side-effect plan preview.
- Secret detection warns before push.
- Structured logs create a machine-readable audit trail.
- Built to reduce risk from silent failure modes in manual workflows (network, auth, and command-sequencing errors across many repos).
- 250+ tests cover core non-UI packages.

## How Git-Fire Works (and Why It Is Worth Trusting)

`git-fire` is intentionally simple in how it works, while aiming to become powerful in what it can do for emergency data safety over time.

- Built in Go for a fast, typed, testable codebase with predictable cross-platform behavior.
- Scans repository roots you explicitly provide (for example via `--path`) plus configured/default scan roots; it does not blindly crawl your entire system unless you point it there.
- Uses concurrent scanning and worker-based execution.
- Uses the native `git` binary and gives control points (`--dry-run`, `--skip-auto-commit`, `--status`) before making changes.

## Core Commands

```bash
# interactive checkpoint flow
git-fire
git fire

# non-destructive preview
git-fire --dry-run

# TUI selector mode
git-fire --fire

# scan specific root
git-fire --path ~/projects

# push existing commits only (no auto-commit)
git-fire --skip-auto-commit

# inspect auth + repo status
git-fire --status

# generate config template
git-fire --init
```

## Set-and-Forget Repeatability

`git-fire` persists discovered repositories in `~/.config/git-fire/repos.toml`. Once discovered, those repos stay in scope for future runs unless you explicitly ignore them.

See [docs/REGISTRY.md](docs/REGISTRY.md).

## Extensible via Plugins (`v0.2`)

Plugin support is in active development. Command plugin internals exist, but default CLI auto-loading from config is a `v0.2` target.

Practical workaround today: `git-fire && your-script`

See [PLUGINS.md](PLUGINS.md) and [examples/plugins/s3-upload.md](examples/plugins/s3-upload.md).

## Documentation

Start with [docs/README.md](docs/README.md).

- Agentic workflows: [docs/agentic-flows.md](docs/agentic-flows.md)
- Security and operations workflows: [docs/security-ops.md](docs/security-ops.md)
- Behavior spec: [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Validation status: [docs/REQUIREMENTS_VALIDATION.md](docs/REQUIREMENTS_VALIDATION.md)

## Alpha Risk and Warranty

`git-fire` is alpha software. Keep independent backups, verify results, and treat this as a fast checkpointing layer, not your only data safety mechanism.

No warranty is provided (express or implied), including merchantability or fitness for a particular purpose.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

