// Package config defines the git-fire configuration schema and related constants.
package config

import "time"

// Config represents the complete git-fire configuration
type Config struct {
	Global  GlobalConfig   `mapstructure:"global"   toml:"global"`
	UI      UIConfig       `mapstructure:"ui"       toml:"ui"`
	Backup  BackupConfig   `mapstructure:"backup"   toml:"backup"`
	Auth    AuthConfig     `mapstructure:"auth"     toml:"auth"`
	Plugins PluginsConfig  `mapstructure:"plugins"  toml:"plugins"`
	Repos   []RepoOverride `mapstructure:"repos"    toml:"repos"`

	// File-backed secret snapshots captured at load time and used by SaveConfig
	// to avoid persisting environment-injected secret values.
	fileBackupAPIToken    string
	fileSSHPassphrase     string
	hasFileBackupAPIToken bool
	hasFileSSHPassphrase  bool
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	// Default mode for repos: "push-known-branches", "push-all", "leave-untouched"
	DefaultMode string `mapstructure:"default_mode" toml:"default_mode"`

	// Conflict strategy: "new-branch" or "abort"
	ConflictStrategy string `mapstructure:"conflict_strategy" toml:"conflict_strategy"`

	// Auto-commit uncommitted changes before pushing
	AutoCommitDirty bool `mapstructure:"auto_commit_dirty" toml:"auto_commit_dirty"`

	// Block auto-commit/push when suspicious secrets are detected.
	BlockOnSecrets bool `mapstructure:"block_on_secrets" toml:"block_on_secrets"`

	// Scan configuration
	ScanPath    string        `mapstructure:"scan_path"    toml:"scan_path"`
	ScanExclude []string      `mapstructure:"scan_exclude" toml:"scan_exclude"`
	ScanDepth   int           `mapstructure:"scan_depth"   toml:"scan_depth"`
	ScanWorkers int           `mapstructure:"scan_workers" toml:"scan_workers"`
	PushWorkers int           `mapstructure:"push_workers" toml:"push_workers"`
	CacheTTL    time.Duration `mapstructure:"cache_ttl"    toml:"cache_ttl"`

	// Re-scan known repos for new submodules (global default; overridable per-repo in registry)
	RescanSubmodules bool `mapstructure:"rescan_submodules" toml:"rescan_submodules"`

	// Skip filesystem walk; only back up repos already in the registry.
	// Set via --no-scan flag (this run only) or disable_scan = true in config.
	DisableScan bool `mapstructure:"disable_scan" toml:"disable_scan"`

	// Allow destructive local branch realignment during `git-fire rain`.
	// When false, local-only commits are never rewritten.
	RainRiskyMode bool `mapstructure:"rain_risky_mode" toml:"rain_risky_mode"`
}

// BackupConfig contains backup mode settings
type BackupConfig struct {
	// Target remote URL (for backup mode)
	TargetRemote string `mapstructure:"target_remote" toml:"target_remote"`

	// Platform: "github", "gitlab", "gitea"
	Platform string `mapstructure:"platform" toml:"platform"`

	// API token for creating repos
	APIToken string `mapstructure:"api_token" toml:"api_token"`

	// Repo naming template
	// Available vars: {repo}, {date}, {hostname}
	RepoTemplate string `mapstructure:"repo_template" toml:"repo_template"`

	// Organization/user to create repos under
	Organization string `mapstructure:"organization" toml:"organization"`

	// Generate backup manifest
	GenerateManifest bool `mapstructure:"generate_manifest" toml:"generate_manifest"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	// SSH passphrase (prefer env var)
	SSHPassphrase string `mapstructure:"ssh_passphrase" toml:"ssh_passphrase"`

	// Use ssh-agent
	UseSSHAgent bool `mapstructure:"use_ssh_agent" toml:"use_ssh_agent"`
}

// RepoOverride allows per-repo configuration
type RepoOverride struct {
	// Match by path pattern (glob)
	PathPattern string `mapstructure:"path" toml:"path"`

	// Match by remote URL pattern
	RemotePattern string `mapstructure:"remote" toml:"remote"`

	// Override mode for this repo
	Mode string `mapstructure:"mode" toml:"mode"`

	// Skip auto-commit for this repo
	SkipAutoCommit bool `mapstructure:"skip_auto_commit" toml:"skip_auto_commit"`

	// Re-scan this repo for submodules. nil/unset = use global default.
	RescanSubmodules *bool `mapstructure:"rescan_submodules" toml:"rescan_submodules"`
}

// UIConfig contains TUI/display settings
type UIConfig struct {
	// Show the fire animation in the repo selector. Toggle live with 'f'.
	// Automatically suppressed when the terminal is too short.
	ShowFireAnimation bool `mapstructure:"show_fire_animation" toml:"show_fire_animation"`

	// Show flavor quotes: TUI banner plus CLI motivation lines (success/failure).
	// TOML key remains show_startup_quote. Toggle in Settings as "Show flavor quotes".
	ShowStartupQuote bool `mapstructure:"show_startup_quote" toml:"show_startup_quote"`

	// Flavor quote behavior after interval elapses (TUI). Options: "refresh", "hide".
	StartupQuoteBehavior string `mapstructure:"startup_quote_behavior" toml:"startup_quote_behavior"`

	// Interval in seconds for flavor quote behavior in the TUI.
	StartupQuoteIntervalSec int `mapstructure:"startup_quote_interval_sec" toml:"startup_quote_interval_sec"`

	// Fire animation tick interval in milliseconds.
	// Lower values animate faster but can increase terminal CPU usage.
	FireTickMS int `mapstructure:"fire_tick_ms" toml:"fire_tick_ms"`

	// Color profile for fire and TUI accents.
	// Options: "classic", "synthwave", "forest", "arctic".
	ColorProfile string `mapstructure:"color_profile" toml:"color_profile"`
}

const (
	UIColorProfileClassic   = "classic"
	UIColorProfileSynthwave = "synthwave"
	UIColorProfileForest    = "forest"
	UIColorProfileArctic    = "arctic"

	UIQuoteBehaviorRefresh = "refresh"
	UIQuoteBehaviorHide    = "hide"
)

// UIColorProfiles returns valid built-in UI color profile names.
func UIColorProfiles() []string {
	return []string{
		UIColorProfileClassic,
		UIColorProfileSynthwave,
		UIColorProfileForest,
		UIColorProfileArctic,
	}
}

// PluginsConfig contains plugin configuration
type PluginsConfig struct {
	// List of enabled plugin names
	Enabled []string `mapstructure:"enabled" toml:"enabled"`

	// Command plugins
	Command []CommandPluginConfig `mapstructure:"command" toml:"command"`

	// Webhook plugins
	Webhook []WebhookPluginConfig `mapstructure:"webhook" toml:"webhook"`
}

// CommandPluginConfig configures a command plugin
type CommandPluginConfig struct {
	Name    string            `mapstructure:"name"    toml:"name"`
	Command string            `mapstructure:"command" toml:"command"`
	Args    []string          `mapstructure:"args"    toml:"args"`
	Env     map[string]string `mapstructure:"env"     toml:"env"`
	When    string            `mapstructure:"when"    toml:"when"`
	Timeout string            `mapstructure:"timeout" toml:"timeout"`
	FailRun bool              `mapstructure:"fail_run" toml:"fail_run"`
}

// WebhookPluginConfig configures a webhook plugin
type WebhookPluginConfig struct {
	Name    string            `mapstructure:"name"    toml:"name"`
	URL     string            `mapstructure:"url"     toml:"url"`
	Method  string            `mapstructure:"method"  toml:"method"`
	Headers map[string]string `mapstructure:"headers" toml:"headers"`
	Body    string            `mapstructure:"body"    toml:"body"`
	When    string            `mapstructure:"when"    toml:"when"`
	Timeout string            `mapstructure:"timeout" toml:"timeout"`
}
