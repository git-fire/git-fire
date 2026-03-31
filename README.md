# Git Fire - Emergency Git Backup Tool

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

`git-fire` discovers repositories, auto-commits dirty work (unless you disable it), and pushes in parallel so work is not stranded locally. It is built for emergency backup and also works for routine multi-repo sync.

Invocation note: you can use either `git-fire` or `git fire` (Git resolves `git-fire` on PATH as a `git` subcommand).

## Alpha Status

`git-fire` is currently in alpha, and we are actively looking for testers and feedback.

## Project Snapshot

- **Project:** `git-fire` (`github.com/git-fire/git-fire`)
- **Language:** Go 1.24.2
- **License:** MIT
- **Status:** Alpha
- **Core promise:** one command to discover repos, auto-commit dirty work (unless disabled), and push backups so local-only work is not lost

Detailed product, architecture, safety, testing, and roadmap notes are in [docs/PROJECT_OVERVIEW.md](docs/PROJECT_OVERVIEW.md).

## Quick Start

### One-line emergency mode

> **Coming soon:** This emergency bootstrap URL is not live yet. Keep this command ready for the upcoming release.

```bash
curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/emergency.sh | bash
```

### Install

> **Coming soon:** Homebrew, Scoop, and packaged binary distribution are not published yet.  
> Keep the commands below as the planned install paths for beta rollout.

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

# run interactive backup
git-fire
```

### TUI screenshot

Current `git-fire` TUI: multi-repo selection, per-repo status, and one-screen checkpoint workflow.

![git-fire TUI screenshot showing repository selection and status view](assets/git-fire-tui-screenshot.png)

## Core Commands

```bash
# interactive emergency backup
git-fire
# same command via git subcommand aliasing
git fire

# non-destructive fire drill
git-fire --dry-run

# "fire mode" selector UI
git-fire --fire

# scan a specific root
git-fire --path ~/projects

# push existing commits only (no auto-commit)
git-fire --skip-auto-commit

# inspect auth/repo status
git-fire --status

# generate config template
git-fire --init
```

## Concepts at a Glance

### Safety model

`git-fire` is designed to avoid destructive behavior:
- never force-pushes in normal flows
- uses conflict backup branches (`git-fire-backup-*`) when needed
- supports dry-run planning before execution

See canonical behavior details in [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md).

### Persistent repo registry

The repo registry (`~/.config/git-fire/repos.toml`) tracks known repos so repeat runs are fast and manageable.

See [docs/REGISTRY.md](docs/REGISTRY.md).

### Agentic workflows

`git-fire` works well as an end-of-session safety net for AI coding agents and can be wired into hooks.

See [docs/agentic-flows.md](docs/agentic-flows.md).

## Release Roadmap

- **Beta goal (next 2 weeks):** begin beta rollout with expanded tester validation and feedback.
- **During beta:** begin publishing `git-fire` binaries to online package managers and address critical stabilization issues.
- **1.0 release target (next 2-4 months):** ship a stable production release after beta-critical items are closed.

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

See [PLUGINS.md](PLUGINS.md) and [examples/plugins/s3-upload.md](examples/plugins/s3-upload.md).

## Use Cases

- Emergency/disaster backup when immediate off-machine sync is needed
- End-of-day multi-repo commit and push
- Agent session checkpointing
- Scheduled automation with external orchestrators
- Layered backup strategy with plugins

## Documentation

Start with the docs hub: [docs/README.md](docs/README.md)

- Spec and behavior: [GIT_FIRE_SPEC.md](GIT_FIRE_SPEC.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Plugins: [PLUGINS.md](PLUGINS.md)
- Registry internals: [docs/REGISTRY.md](docs/REGISTRY.md)
- Agentic usage: [docs/agentic-flows.md](docs/agentic-flows.md)
- Validation status: [docs/REQUIREMENTS_VALIDATION.md](docs/REQUIREMENTS_VALIDATION.md)

## Security Notes

Before running broad backups:
- keep secrets out of tracked files
- rely on `.gitignore` and `.git/info/exclude` for local secret files
- run `git-fire --dry-run` regularly to inspect what would be committed

`git-fire` includes secret detection warnings, but commit responsibility remains with the user.

## Alpha Risk and Warranty

The product is stable in many common workflows, but it is still alpha and should not be fully trusted yet. Use at your own risk.

No warranty is provided (express or implied), including merchantability or fitness for a particular purpose. Maintain your own backup strategy, verify backup results, and keep updating as fixes are released.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for build/test expectations and PR guidelines.

## License

MIT. See [LICENSE](LICENSE).

