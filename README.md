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
  <a href="https://discord.gg/pjkVMSpT7j"><img src="https://img.shields.io/badge/Discord-5865F2?logo=discord&logoColor=white" alt="Discord"></a>
</p>

> In case of fire:
> 1. `git-fire`
> 2. Leave building

`git-fire` is a multi-repo checkpoint command for people managing many Git repos: discover repositories, optionally auto-commit dirty work, and push backup branches/remotes with safety rails.

Manual push loops can fail silently in real life (network drops, auth problems, or tool hiccups). `git-fire` gives you an auditable recovery path and more peace of mind when you need consistency across many repos.

Invocation note: `git-fire` and `git fire` are equivalent when `git-fire` is on your PATH.

### TUI screenshot

Current `git-fire` TUI: multi-repo selection, per-repo status, and one-screen checkpoint workflow.

![git-fire TUI screenshot showing repository selection and status view](assets/git-fire-tui-screenshot-gh.png)

## Alpha Status

`git-fire` is alpha software. Core multi-repo backup flows are usable today. Some roadmap items (plugin CLI auto-loading and `--backup-to`) are intentionally not wired yet.

## Quick Start

### Recommended first run (safe)

```bash
# preview first (safe)
git-fire --dry-run --path ~/projects

# then run the default streamed checkpoint flow
git-fire --path ~/projects
```

Use `--path` on first run to scope discovery and avoid unexpectedly broad scans.

### One-line emergency mode

Use this for urgent situations only. `curl | bash` executes remote code directly.
Inspect `scripts/emergency.sh` first and prefer release assets plus checksum verification when you have time.

```bash
# replace v0.1.0-alpha with the release tag you want to run
curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/v0.1.0-alpha/scripts/emergency.sh | bash
```

### Install

| Method | Command | Platform |
|---|---|---|
| Go | `go install github.com/git-fire/git-fire@latest` | All (Go 1.24.2+) |
| Binary | [GitHub Releases](https://github.com/git-fire/git-fire/releases) | All |

For the alpha phase, `git-fire` is distributed via Go install and GitHub release binaries only.
Homebrew/Scoop publishing will be enabled in a later stable release.

#### Verify release checksums (recommended)

The emergency/install scripts currently download release assets over HTTPS but do not verify checksums for you.
For higher trust, verify release hashes manually before running binaries.

```bash
# from a release assets directory containing the binary archive and checksums.txt
sha256sum -c checksums.txt
```
On macOS, use `shasum -a 256 -c checksums.txt`.

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

## Who Is This For

- **Primary users:** developers, infra/platform engineers, and agent-workflow users who manage many repos and need reliable checkpoints.
- **Also useful for:** security/ops and research/docs teams working across multiple Git repositories.
- **Not the target:** single-repo users or monorepo teams with strong one-repo checkpoint discipline.

## Good Fit / Not a Fit

**Good fit if:**
- you manage many repos and regularly context-switch
- you want one repeatable command with dry-run and audit logs
- you value safety defaults over minimal command surface

**Not a fit if:**
- you only use one repo and already have a simple backup habit
- you want a tiny one-purpose shell script with minimal behavior
- you do not want optional auto-commit capabilities in your workflow

## Relationship to classic `git-fire`

`git-fire` (this repo) is inspired by the original `qw3rtman/git-fire` emergency script, but targets a different problem size:
- original: single-repo emergency push flow in a lightweight script
- this project: multi-repo discovery, persistent registry, dry-run planning, safety rails, and structured logs

Which is "better" depends on perspective. If you want minimal one-repo behavior, the classic script may feel simpler. If you need repeatable multi-repo checkpointing, this tool is built for that use case.

## Use Cases

- **Daily checkpoints:** end-of-day, before context switches, before risky refactors.
- **Agent workflow safety net:** run at session stop, keep logs for review, dry-run in guarded environments.
- **Ops/security windows:** checkpoint tooling/config repos before maintenance, teardown, or incident response changes.
- **Emergency mode:** if your build is on fire, run `git-fire`.

Workflow guides:
- [docs/agentic-flows.md](docs/agentic-flows.md)
- [docs/security-ops.md](docs/security-ops.md)

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
- **Execution-mode control:** `--dry-run` for no git commit/push mutations (it still updates discovered repos in the registry), `--fire` for interactive selection, `--path` for scoped discovery.
- **Registry-only mode:** use `--no-scan` to back up only repos already in the registry for this run.
- **Config trust boundary:** only `~/.config/git-fire/config.toml` is loaded by default; use `--config <path>` to opt into a project-local file.
- **Auto-commit strategy control:** choose whether dirty working trees are included with default behavior or skipped via `--skip-auto-commit`.
- **Session logging:** each run writes structured logs under `~/.cache/git-fire/logs/` for auditability and debugging.
- **Workflow composition:** combine with hooks, wrappers, task runners, or CI helper scripts for consistent team or solo automation.
## Why It Is Trustworthy in Alpha

- No force push in normal flows.
- Conflict and auto-commit strategies create explicit backup branches (`git-fire-backup-*`, `git-fire-staged-*`, `git-fire-full-*`) when needed.
- Dry-run gives a no git side-effect plan preview.
- Secret detection blocks the auto-commit path by default (override in config if you explicitly accept risk).
- Structured logs create a machine-readable audit trail.
- Built to reduce risk from silent failure modes in manual workflows (network, auth, and command-sequencing errors across many repos).
- 250+ tests cover core non-UI packages.

## Core Commands

```bash
# default streamed checkpoint flow
git-fire
git fire

# non-destructive preview
git-fire --dry-run
git-fire --fire-drill

# TUI selector mode
git-fire --fire

# scan specific root
git-fire --path ~/projects

# push existing commits only (no auto-commit)
git-fire --skip-auto-commit

# inspect auth + repo status
git-fire --status

# use explicit config path (project-local opt-in)
git-fire --config ./git-fire.toml

# use only known registry repos for this run
git-fire --no-scan

# generate config template
git-fire --init
```

## Set-and-Forget Repeatability

`git-fire` persists discovered repositories in `~/.config/git-fire/repos.toml`. Once discovered, those repos stay in scope for future runs unless you explicitly ignore them.

See [docs/REGISTRY.md](docs/REGISTRY.md).

## Extensible via Plugins (`v0.2`)

Plugin support is in active development. Command plugin internals exist, but runtime integration in the default CLI path (including config-based auto-loading) remains a `v0.2` target.

See [docs/agentic-flows.md](docs/agentic-flows.md).

## USB Mode (planned, not in alpha)

USB mode is a planned roadmap feature for first-class backup destinations. It is intentionally not part of current alpha behavior.

See `docs/USB_MODE.md` for current status and planned scope.

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

Planned command plugins will let you trigger extra backup/notification steps (for example S3 sync, webhook calls via curl, local archive scripts) once `v0.2` runtime wiring lands.

See [PLUGINS.md](PLUGINS.md) and [examples/plugins/s3-upload.md](examples/plugins/s3-upload.md).

## Documentation

Start with [docs/README.md](docs/README.md).

- Agentic workflows: [docs/agentic-flows.md](docs/agentic-flows.md)
- Security and operations workflows: [docs/security-ops.md](docs/security-ops.md)
- Behavior spec: [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Security policy: [SECURITY.md](SECURITY.md)
- Historical validation archive: [docs/REQUIREMENTS_VALIDATION.md](docs/REQUIREMENTS_VALIDATION.md)

## Security Notes

Before running broad backups:
- keep secrets out of tracked files
- rely on `.gitignore` and `.git/info/exclude` for local secret files
- run `git-fire --dry-run` regularly to inspect what would be committed

`git-fire` includes secret detection warnings, but commit responsibility remains with the user.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

