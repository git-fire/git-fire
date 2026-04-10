package config

import "time"

const DefaultPushWorkers = 4
const DefaultUIFireTickMS = 180
const DefaultUIStartupQuoteIntervalSec = 10

// MinUIFireTickMS and MaxUIFireTickMS clamp ui.fire_tick_ms after load (see
// Config.Validate). That field becomes the Bubble Tea program's tick period: the
// main event loop wakes on that interval even when the fire layer is hidden (only
// fire updates are skipped). Values far below ~30ms spin CPU without visible
// benefit; values above one minute make path scroll and other UI cadence feel
// stuck. These bounds favor reliability for an emergency backup tool over
// honoring every edge-case TOML number.
const MinUIFireTickMS = 30
const MaxUIFireTickMS = 60000

// DefaultConfig returns safe default configuration
func DefaultConfig() Config {
	return Config{
		UI: UIConfig{
			ShowFireAnimation:       true,
			ShowStartupQuote:        true,
			StartupQuoteBehavior:    UIQuoteBehaviorRefresh,
			StartupQuoteIntervalSec: DefaultUIStartupQuoteIntervalSec,
			FireTickMS:              DefaultUIFireTickMS,
			ColorProfile:            UIColorProfileClassic,
		},
		Global: GlobalConfig{
			DefaultMode:      "push-known-branches",
			ConflictStrategy: "new-branch",
			AutoCommitDirty:  true,
			BlockOnSecrets:   true,
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
# Place this file at ~/.config/git-fire/config.toml

[global]
# Default push mode for repositories
# Options: "push-known-branches", "push-all", "leave-untouched"
default_mode = "push-known-branches"

# How to handle conflicts between local and remote
# Options: "new-branch" (create git-fire-backup-* branch), "abort"
conflict_strategy = "new-branch"

# Auto-commit uncommitted changes before pushing
auto_commit_dirty = true

# Block auto-commit/push when suspicious secrets are detected
# Set false only if you explicitly accept the risk.
block_on_secrets = true

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

# Flavor quotes: TUI banner plus terminal motivation lines after runs.
# In Settings (TUI) this row is labeled "Show flavor quotes".
show_startup_quote = true

# Flavor quote behavior in the TUI:
# "refresh" = rotate to a new quote every startup_quote_interval_sec
# "hide" = remove quote after startup_quote_interval_sec
startup_quote_behavior = "refresh"

# Seconds between flavor quote refresh/hide actions in the TUI.
startup_quote_interval_sec = 10

# Fire animation speed in milliseconds per frame (also drives the TUI tick).
# Lower = faster/smoother but higher CPU usage. Recommended: 120-300.
# Values outside 30-60000 ms are clamped when the config is loaded.
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

# [plugins]
# enabled = ["my-resolver"]
#
# Merge-conflict plugins run during planning when the current branch has diverged
# from a remote (after fetch). Print "true" or "false" on the first line of stdout
# to report whether the divergence was resolved (e.g. after an automated merge).
# Template args: {repo_path} {repo_name} {branch} {remote} {local_sha} {remote_sha}
# [[plugins.command]]
# name = "my-resolver"
# command = "sh"
# args = ["-c", "your-tool; echo true"]
# when = "on-merge-conflict"
`
}
