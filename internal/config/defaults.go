package config

import "time"

const DefaultPushWorkers = 4
const DefaultUIFireTickMS = 180

// MinUIFireTickMS and MaxUIFireTickMS bound ui.fire_tick_ms after load so the TUI
// tick interval cannot become a busy loop (tiny values) or an impractically long
// sleep (huge values).
const MinUIFireTickMS = 30
const MaxUIFireTickMS = 60000

// DefaultConfig returns safe default configuration
func DefaultConfig() Config {
	return Config{
		UI: UIConfig{
			ShowFireAnimation: true,
			FireTickMS:        DefaultUIFireTickMS,
			ColorProfile:      UIColorProfileClassic,
		},
		Global: GlobalConfig{
			DefaultMode:      "push-known-branches",
			ConflictStrategy: "new-branch",
			AutoCommitDirty:  true,
			ScanPath:         ".",
			ScanExclude: []string{
				".cache",
				"node_modules",
				".venv",
				"venv",
				"vendor",
				"dist",
				"build",
				"target",
			},
			ScanDepth:        10,
			ScanWorkers:      8,
			PushWorkers:      DefaultPushWorkers,
			CacheTTL:         24 * time.Hour,
			RescanSubmodules: false,
			DisableScan:      false,
		},
		Backup: BackupConfig{
			Platform:         "github",
			RepoTemplate:     "backup-{repo}-{date}",
			GenerateManifest: true,
		},
		Auth: AuthConfig{
			UseSSHAgent: true,
		},
		Repos: []RepoOverride{},
	}
}

// ExampleConfigTOML returns an example configuration file
func ExampleConfigTOML() string {
	return `# Git Fire Configuration
# Place this file at ~/.config/git-fire/config.toml or ./git-fire.toml

[global]
# Default push mode for repositories
# Options: "push-known-branches", "push-all", "leave-untouched"
default_mode = "push-known-branches"

# How to handle conflicts between local and remote
# Options: "new-branch" (create git-fire-backup-* branch), "abort"
conflict_strategy = "new-branch"

# Auto-commit uncommitted changes before pushing
auto_commit_dirty = true

# Directory to scan for git repos
scan_path = "."

# Directories to exclude from scanning
scan_exclude = [
    ".cache",
    "node_modules",
    ".venv",
    "venv",
    "vendor",
    "dist",
    "build",
    "target"
]

# Maximum directory depth to scan
scan_depth = 10

# Number of parallel workers for scanning
scan_workers = 8

# Number of parallel workers for pushing repositories
push_workers = 4

# Cache TTL (e.g., "24h", "1h30m")
cache_ttl = "24h"

# Re-scan known repos for new submodules (global default; overridable per-repo in registry)
rescan_submodules = false

# Skip filesystem walk; only back up repos already in the registry.
# Use --no-scan flag to override for a single run without changing this file.
disable_scan = false

[ui]
# Show the fire animation in the TUI repo selector.
# Toggle live during a session with the 'f' key.
# The animation is always suppressed when the terminal is too short regardless of this setting.
show_fire_animation = true

# Fire animation speed in milliseconds per frame.
# Lower = faster/smoother but higher CPU usage.
# Recommended range for most terminals: 120-300.
fire_tick_ms = 180

# Built-in color profile for fire + borders/accents in the TUI.
# Options: "classic", "synthwave", "forest", "arctic"
color_profile = "classic"

[backup]
# Backup mode: Push to a different remote (creates repos automatically)
# Leave empty to use existing remotes
target_remote = ""

# Platform for backup mode: "github", "gitlab", "gitea"
platform = "github"

# API token for creating repositories (prefer env var: GIT_FIRE_API_TOKEN)
api_token = ""

# Repository naming template
# Available variables: {repo}, {date}, {hostname}
repo_template = "backup-{repo}-{date}"

# Organization/user to create repos under
organization = ""

# Generate backup manifest (JSON metadata file)
generate_manifest = true

[auth]
# SSH passphrase (prefer env var: GIT_FIRE_SSH_PASSPHRASE)
ssh_passphrase = ""

# Use ssh-agent for authentication
use_ssh_agent = true

# Per-repository overrides
# [[repos]]
# path = "/home/user/critical-project"
# mode = "push-all"
# skip_auto_commit = false
# rescan_submodules = true   # override global rescan_submodules for this repo

# [[repos]]
# remote = "github.com/company/*"
# mode = "push-known-branches"
`
}
