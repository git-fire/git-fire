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

`git-fire` is one command to checkpoint many repositories: discover, auto-commit dirty work (optional), and push backup branches/remotes with safety rails. It is useful in emergencies and in normal daily developer and agent workflows.

<<<<<<< HEAD
Invocation note: you can use either `git-fire` or `git fire` (Git resolves `git-fire` on PATH as a `git` subcommand).

## Quick Start

### One-line emergency mode

```bash
curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/emergency.sh | bash
```

### Install

=======
Invocation note: `git-fire` and `git fire` are equivalent when `git-fire` is on your PATH.

## Alpha Status

`git-fire` is alpha software. Core multi-repo backup flows are usable today. Some roadmap items (plugin CLI auto-loading and `--backup-to`) are intentionally not wired yet.

## Quick Start

### Install

> **Coming soon:** Homebrew, Scoop, and packaged binary distribution are not published yet.

>>>>>>> origin/main
| Method | Command | Platform |
|---|---|---|
| Go | `go install github.com/git-fire/git-fire@latest` | All (Go 1.24.2+) |
| Binary | [GitHub Releases](https://github.com/git-fire/git-fire/releases) | All |

For the alpha phase, `git-fire` is distributed via Go install and GitHub release binaries only.
Homebrew/Scoop publishing will be enabled in a later stable release.

#### PATH setup (required)

After install, make sure the binary location is on your `PATH`.

**Go install (Linux/macOS):**
```bash
export PATH="$HOME/go/bin:$PATH"
```
Add that line to `~/.zshrc` or `~/.bashrc` to persist.

**Manual binary install (Linux/macOS):**
```bash
# after extracting the release archive:
chmod +x git-fire
sudo mv git-fire /usr/local/bin/
```

**Manual binary install (Windows PowerShell):**
```powershell
# after extracting the release archive:
New-Item -ItemType Directory -Force "$env:USERPROFILE\bin" | Out-Null
Move-Item .\git-fire.exe "$env:USERPROFILE\bin\git-fire.exe" -Force
```
Then add `$env:USERPROFILE\bin` to your user `PATH` if not already present.

#### Verify install

```bash
git-fire --version
which git-fire
```
On Windows PowerShell:
```powershell
git-fire.exe --version
Get-Command git-fire.exe
```

### First run

```bash
# preview first (safe)
git-fire --dry-run --path ~/projects

# run interactive checkpoint
git-fire
```

## Who Is This For

- **Polyrepo developers:** you touch 5-20+ repos and want one end-of-day or pre-travel checkpoint command.
- **Platform/infra engineers:** you maintain many IaC/config/tooling repos and need consistent, auditable bulk checkpoints.
- **Agent workflow users:** you run Claude/Cursor-style coding sessions and want a stop-hook safety net.
- **Security/red team practitioners:** you need fast state preservation before teardown, maintenance, or incident-driven system change.
- **Not the target:** single-repo users and monorepo teams that already have one-repo checkpoint discipline.

## Use Cases

### Daily developer checkpoint

- End of day
- Before context switch
- Before travel
- Before large refactor

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
- 250+ tests cover core non-UI packages.

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

<<<<<<< HEAD
See [docs/agentic-flows.md](docs/agentic-flows.md).

### TUI color profiles

You can reskin both the fire effect and border/accent colors in `git-fire --fire`:

| Profile | Style |
|---------|-------|
| `classic` | Original orange/yellow fire |
| `synthwave` | 80s neon purple/pink/cyan |
| `forest` | Green ember palette |
| `arctic` | Cool cyan/ice palette |

| Method | How |
|--------|-----|
| In-TUI settings | Press **`c`** → **Color profile** → `space` / `←` / `→` |
| Config file | Set `color_profile` under `[ui]` |

```toml
[ui]
show_fire_animation = true
color_profile = "synthwave"
```

Custom hex palettes are planned but not enabled yet. A future release will allow user-defined hex lists for fire and accent colors.

### Extensibility with plugins

Command plugins let you trigger extra backup/notification steps (for example S3 sync, webhook calls via curl, local archive scripts).
=======
Practical workaround today: `git-fire && your-script`
>>>>>>> origin/main

See [PLUGINS.md](PLUGINS.md) and [examples/plugins/s3-upload.md](examples/plugins/s3-upload.md).

## Documentation

Start with [docs/README.md](docs/README.md).

- Agentic workflows: [docs/agentic-flows.md](docs/agentic-flows.md)
- Security and operations workflows: [docs/security-ops.md](docs/security-ops.md)
- Behavior spec: [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Validation status: [docs/REQUIREMENTS_VALIDATION.md](docs/REQUIREMENTS_VALIDATION.md)

<<<<<<< HEAD
## Security Notes

Before running broad backups:
- keep secrets out of tracked files
- rely on `.gitignore` and `.git/info/exclude` for local secret files
- run `git-fire --dry-run` regularly to inspect what would be committed

`git-fire` includes secret detection warnings, but commit responsibility remains with the user.
=======
## Alpha Risk and Warranty

`git-fire` is alpha software. Keep independent backups, verify results, and treat this as a fast checkpointing layer, not your only data safety mechanism.

No warranty is provided (express or implied), including merchantability or fitness for a particular purpose.
>>>>>>> origin/main

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

