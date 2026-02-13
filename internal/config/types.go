package config

import "time"

// Config represents the complete git-fire configuration
type Config struct {
	Global  GlobalConfig   `mapstructure:"global"`
	Backup  BackupConfig   `mapstructure:"backup"`
	Auth    AuthConfig     `mapstructure:"auth"`
	Plugins PluginsConfig  `mapstructure:"plugins"`
	Repos   []RepoOverride `mapstructure:"repos"`
}

// GlobalConfig contains global settings
type GlobalConfig struct {
	// Default mode for repos: "push-known-branches", "push-all", "leave-untouched"
	DefaultMode string `mapstructure:"default_mode"`

	// Conflict strategy: "new-branch" or "abort"
	ConflictStrategy string `mapstructure:"conflict_strategy"`

	// Auto-commit uncommitted changes before pushing
	AutoCommitDirty bool `mapstructure:"auto_commit_dirty"`

	// Scan configuration
	ScanPath    string        `mapstructure:"scan_path"`
	ScanExclude []string      `mapstructure:"scan_exclude"`
	ScanDepth   int           `mapstructure:"scan_depth"`
	ScanWorkers int           `mapstructure:"scan_workers"`
	CacheTTL    time.Duration `mapstructure:"cache_ttl"`
}

// BackupConfig contains backup mode settings
type BackupConfig struct {
	// Target remote URL (for backup mode)
	TargetRemote string `mapstructure:"target_remote"`

	// Platform: "github", "gitlab", "gitea"
	Platform string `mapstructure:"platform"`

	// API token for creating repos
	APIToken string `mapstructure:"api_token"`

	// Repo naming template
	// Available vars: {repo}, {date}, {hostname}
	RepoTemplate string `mapstructure:"repo_template"`

	// Organization/user to create repos under
	Organization string `mapstructure:"organization"`

	// Generate backup manifest
	GenerateManifest bool `mapstructure:"generate_manifest"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	// SSH passphrase (prefer env var)
	SSHPassphrase string `mapstructure:"ssh_passphrase"`

	// Use ssh-agent
	UseSSHAgent bool `mapstructure:"use_ssh_agent"`
}

// RepoOverride allows per-repo configuration
type RepoOverride struct {
	// Match by path pattern (glob)
	PathPattern string `mapstructure:"path"`

	// Match by remote URL pattern
	RemotePattern string `mapstructure:"remote"`

	// Override mode for this repo
	Mode string `mapstructure:"mode"`

	// Skip auto-commit for this repo
	SkipAutoCommit bool `mapstructure:"skip_auto_commit"`
}

// PluginsConfig contains plugin configuration
type PluginsConfig struct {
	// List of enabled plugin names
	Enabled []string `mapstructure:"enabled"`

	// Command plugins
	Command []CommandPluginConfig `mapstructure:"command"`

	// Webhook plugins
	Webhook []WebhookPluginConfig `mapstructure:"webhook"`
}

// CommandPluginConfig configures a command plugin
type CommandPluginConfig struct {
	Name    string            `mapstructure:"name"`
	Command string            `mapstructure:"command"`
	Args    []string          `mapstructure:"args"`
	Env     map[string]string `mapstructure:"env"`
	When    string            `mapstructure:"when"`
	Timeout string            `mapstructure:"timeout"`
}

// WebhookPluginConfig configures a webhook plugin
type WebhookPluginConfig struct {
	Name    string            `mapstructure:"name"`
	URL     string            `mapstructure:"url"`
	Method  string            `mapstructure:"method"`
	Headers map[string]string `mapstructure:"headers"`
	Body    string            `mapstructure:"body"`
	When    string            `mapstructure:"when"`
	Timeout string            `mapstructure:"timeout"`
}
