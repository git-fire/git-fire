# 🔥 Git Fire - Emergency Git Backup Tool

<p align="center">
  <img src="https://img.shields.io/badge/status-MVP-green" alt="Status: MVP">
  <img src="https://img.shields.io/badge/tests-153%2F153-brightgreen" alt="Tests: 153/153">
  <img src="https://img.shields.io/badge/go-1.24.2-blue" alt="Go 1.24.2">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="License: MIT">
</p>

> **In case of fire:**
> 1. `git-fire`
> 2. Leave building

Emergency git backup tool that automatically commits and pushes all your repositories when disaster strikes.

## 🚨 Quick Start

### One-Line Emergency Mode

If the building is on fire RIGHT NOW:

```bash
curl -fsSL https://raw.githubusercontent.com/TBRX103/git-fire/main/scripts/emergency.sh | bash
```

This will:
- ✓ Find all git repos in current directory
- ✓ Auto-commit uncommitted changes
- ✓ Push to all remotes
- ✓ Report success/failures
- ✓ No installation required

### Install Properly

```bash
curl -fsSL https://raw.githubusercontent.com/TBRX103/git-fire/main/scripts/install.sh | bash
```

Or with Go:

```bash
go install github.com/TBRX103/git-fire@latest
```

Or build from source:

```bash
git clone https://github.com/TBRX103/git-fire.git
cd git-fire
go build -o git-fire .
```

## Getting started

After you install git-fire, **populate the registry** so future runs know about your repositories. The registry lives at `~/.config/git-fire/repos.toml` (next to `config.toml`).

1. **Discover repos (safe)** — From each directory tree where you keep git repositories, run a fire drill so paths are recorded without pushing:

   ```bash
   git-fire --dry-run --path ~/projects
   ```

   Repeat with `--path` for other roots, or use `git-fire repos scan [path]` (defaults to your config `scan_path`).

2. **Exclude repos you do not want backed up** — In `git-fire --fire`, press `x` on a repo to mark it **ignored** in the registry, or run `git-fire repos ignore <path>`. Ignored repos are hidden from the fire selector and are not backed up.

3. **Track a repo again** — Run `git-fire repos unignore <path>`, or in `git-fire --fire` press `i` to open the ignored list, then `enter` or `u` on a row to restore tracking.

4. **Inspect the registry** — `git-fire repos list` shows every tracked path and its status.

See **Usage** below for `--fire`, `--dry-run`, and `--path`.

## 🎯 Features

### Core Features

- ✅ **Auto-commit dirty repos** - Commits uncommitted changes with timestamp
- ✅ **Multi-repo scanning** - Finds all git repos recursively
- ✅ **Parallel processing** - Scans and pushes repos in parallel
- ✅ **Conflict detection** - Creates fire branches when local/remote diverge
- ✅ **Fire branches** - Format: `git-fire-backup-main-20260212-abc1234`
- ✅ **Push modes** - Push all branches, known branches, or specific branch
- ✅ **SSH key detection** - Auto-detects and validates SSH keys
- ✅ **Secret detection** - Warns about AWS keys, tokens, .env files
- ✅ **Dry-run mode** - Preview what would happen (fire drill)
- ✅ **Background scanning** - Scans repos and SSH keys while waiting for input
- ✅ **Structured logging** - JSON logs with full reversibility tracking
- ✅ **Zero-config** - Works out of the box, configure if needed

### Safety Features

- 🛡️ **Secret detection** - Detects 10+ types of secrets (AWS keys, GitHub tokens, private keys, etc.)
- 🛡️ **Dry-run validation** - Test before pushing
- 🛡️ **User confirmation** - Requires confirmation before pushing
- 🛡️ **Reversible** - Full logs for undoing changes
- 🛡️ **Respects .gitignore for untracked files** - Won't add new ignored files (tracked files in `.gitignore` can still be committed)

## 📖 Usage

### Basic Commands

```bash
# Emergency push (interactive)
git-fire

# Fire drill - preview what would happen
git-fire --dry-run

# Status check - see repos and SSH keys
git-fire --status

# Generate config file
git-fire --init

# Scan specific directory
git-fire --path ~/projects

# Skip auto-commit (only push existing commits)
git-fire --skip-auto-commit

# Fire mode: TUI repo selector, skips confirmation prompt
git-fire --fire
```

### Advanced Usage

```bash
# Backup to specific remote
git-fire --backup-to git@github.com:user/emergency-backup.git

# Scan with custom settings
git-fire --path ~/critical-projects --skip-auto-commit --dry-run
```

## ⚙️ Configuration

Git-fire works with zero configuration, but you can customize it:

```bash
# Generate example config
git-fire --init
```

See [example config](https://github.com/TBRX103/git-fire/blob/main/internal/config/defaults.go) for all options.

## 🔌 Extensibility

Git-fire is designed to be extensible beyond just git. See [PLUGINS.md](PLUGINS.md) for the plugin architecture.

Quick example - upload to S3:
```toml
[[plugins.command]]
name = "s3-backup"
command = "aws"
args = ["s3", "sync", "{repo_path}", "s3://emergency/{repo_name}"]
```

## 🧪 Testing

```bash
# Run all tests (153/153 passing)
go test ./...

# With coverage
go test -cover ./...
```

### Git integration test helpers

The [`internal/testutil`](internal/testutil) package drives the **real `git` binary** to create temporary repositories, commits, remotes, branches, and dirty trees. That lets integration tests exercise the same behavior users see, without mocking git. The same building blocks are useful for **other Go projects** that need reproducible repo fixtures in tests.

We intend to **extract and open source** this helper library as a standalone module when it is mature enough to stand on its own. If you publish a compatible extraction or fork **before** we do, please **link back to this repository** (and ideally mention git-fire in the readme) so people can discover the upstream project. We will **review and, where it makes sense, adopt or align** with a well-maintained community version rather than duplicate effort.

**License and credit:** git-fire is released under the **MIT License**. MIT already requires that the **copyright notice and permission text** be preserved in copies and substantial portions—that is the legal baseline for credit. A clear **link or citation to git-fire** in addition to that notice is appreciated and helps users find the canonical source; it does **not** require changing away from MIT. A standalone spin-out of the test helpers can remain **MIT** (or another permissive license you choose) as long as you comply with MIT’s notice requirement for any code derived from this repo.

## 📊 Architecture

CLI-first design with background scanning, parallel execution, and plugin support.

See [architecture docs](./docs) for details.

## ⚠️ Security Notice

Git-fire will auto-commit tracked changes plus untracked files not excluded by `.gitignore` in emergency mode.
Note: `.gitignore` only prevents *untracked* files from being added. If a secret file was previously committed and is now in `.gitignore`, Git still tracks it — changes to it will be staged and committed.

**Before using:**
- ✓ Use `.env` files for secrets (add to `.gitignore`)
- ✓ Never commit: `.env`, `credentials.json`, `*.pem`, `*.key`
- ✓ **ALWAYS run `--dry-run` first** to preview commits

Git-fire includes secret detection to warn you, but **you** are responsible for your commits.

## 🔒 Best Practices

Before relying on git-fire in an emergency, make sure your secrets are excluded from git:

**`.gitignore`** — prevents untracked secret files from ever being staged:

```gitignore
.env
.env.*
!.env.example
*.pem
*.key
credentials.json
secrets.yaml
config/secrets.yml
```

**`.git/info/exclude`** — machine-local exclusions that don't get committed (useful for files you can't add to a shared `.gitignore`):

```bash
echo "my-local-secrets.txt" >> .git/info/exclude
```

Note: neither of these helps if a secret file was already committed. In that case, remove it from history with `git filter-repo` and rotate the secret.

Run `git-fire --dry-run` regularly to see exactly what would be committed before an emergency happens.

## 🔥 Comparison to Other Tools

**Note:** There's an old [qw3rtman/git-fire](https://github.com/qw3rtman/git-fire) (bash, 2015, archived) with the same name, but this is an independent project with different goals:

| Feature | This (Go, 2026) | qw3rtman (bash, 2015) |
|---------|----------------|----------------------|
| Multi-repo | ✅ Parallel | ❌ Single repo |
| Secret detection | ✅ Yes | ❌ No |
| Dry-run | ✅ Yes | ❌ No |
| SSH key mgmt | ✅ Auto-detect | ❌ Manual |
| Config | ✅ TOML + env | ❌ None |
| Background scan | ✅ Yes | ❌ No |
| Plugins | ✅ Extensible | ❌ No |
| Tests | ✅ 200+ tests | ❌ No tests |
| Active | ✅ 2026 | ❌ 2015 (archived) |

## 🌐 Website

[git-fire.com](https://git-fire.com) — coming soon.

## 📝 License

MIT License

## 🤖 Agentic Coding

AI coding agents edit files at high speed across multiple repos without committing. Git-fire is a natural safety net: run it at the end of every agent session to ensure nothing is lost.

**Quick setup with Claude Code** — add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "mkdir -p ~/.cache/git-fire && git-fire --path . >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

This runs git-fire automatically after every Claude Code session ends.

See [docs/agentic-flows.md](docs/agentic-flows.md) for the full integration guide, including plugin callbacks, registry management, and the roadmap for MCP server mode and structured JSON output.

## 😴 End-of-Day Use

git-fire isn't just for emergencies. It's also a perfectly valid end-of-day tool for the days when you don't feel like figuring out which repos you touched:

```bash
git-fire
```

One command. Every dirty repo in your working directory (or `--path ~/projects`) gets committed and pushed. You leave with a clean slate and nothing left unsaved. No need to context-switch back through eight terminals figuring out what you were working on.

The `--dry-run` flag lets you preview what it would commit before actually doing it.

## 🐶 Dogfooding

Git-fire is developed using git-fire. During development, `make run` was accidentally run without `--dry-run` — and git-fire immediately committed and pushed its own source code mid-development. It saved itself. That's the whole pitch.

---

<p align="center">
  <strong>🔥 In case of fire: git-fire 🔥</strong>
</p>