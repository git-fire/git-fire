# 🔥 Git Fire - Emergency Git Backup Tool

<p align="center">
  <img src="https://img.shields.io/badge/status-MVP-green" alt="Status: MVP">
  <img src="https://img.shields.io/badge/tests-43%2F43-brightgreen" alt="Tests: 43/43">
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
- 🛡️ **Respects .gitignore** - Won't commit ignored files

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

# Use fancy fire UI mode
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
# Run all tests (43/43 passing)
go test ./...

# With coverage
go test -cover ./...
```

## 📊 Architecture

CLI-first design with background scanning, parallel execution, and plugin support.

See [architecture docs](./docs) for details.

## ⚠️ Security Notice

Git-fire will auto-commit ALL uncommitted files in emergency mode.

**Before using:**
- ✓ Use `.env` files for secrets (add to `.gitignore`)
- ✓ Never commit: `.env`, `credentials.json`, `*.pem`, `*.key`
- ✓ **ALWAYS run `--dry-run` first** to preview commits

Git-fire includes secret detection to warn you, but **you** are responsible for your commits.

## 🔥 Comparison to Other Tools

This project was inspired by [qw3rtman/git-fire](https://github.com/qw3rtman/git-fire) (Ruby, 2015) but is a complete rewrite with modern features:

| Feature | This (Go, 2026) | qw3rtman (Ruby, 2015) |
|---------|----------------|----------------------|
| Multi-repo | ✅ Parallel | ❌ Single repo |
| Secret detection | ✅ Yes | ❌ No |
| Dry-run | ✅ Yes | ❌ No |
| SSH key mgmt | ✅ Auto-detect | ❌ Manual |
| Config | ✅ TOML + env | ❌ None |
| Background scan | ✅ Yes | ❌ No |
| Plugins | ✅ Extensible | ❌ No |
| Tests | ✅ 43 tests | ❌ No tests |
| Active | ✅ 2026 | ❌ 2015 (archived) |

## 📝 License

MIT License

## 🙏 Credits

- Inspired by the "in case of fire" meme and [qw3rtman/git-fire](https://github.com/qw3rtman/git-fire)
- Built with Cobra, Viper, Bubble Tea, Lipgloss

---

<p align="center">
  <strong>🔥 In case of fire: git-fire 🔥</strong>
</p>