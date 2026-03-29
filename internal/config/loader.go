package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Load loads configuration from files and environment variables
// Priority (highest to lowest):
//  1. Environment variables (GIT_FIRE_*)
//  2. ./git-fire.toml (current directory)
//  3. ~/.config/git-fire/config.toml (user config)
//  4. Default config
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("toml")

	// Add config paths
	v.AddConfigPath(".")                           // Current directory (./git-fire.toml)
	v.AddConfigPath("$HOME/.config/git-fire")      // User config
	v.AddConfigPath("/etc/git-fire")               // System config

	// Environment variables
	v.SetEnvPrefix("GIT_FIRE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional - don't error if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Config file found but has errors
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found - use defaults (this is OK)
	}

	// Unmarshal into struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override with env vars for sensitive data
	if token := os.Getenv("GIT_FIRE_API_TOKEN"); token != "" {
		cfg.Backup.APIToken = token
	}
	if passphrase := os.Getenv("GIT_FIRE_SSH_PASSPHRASE"); passphrase != "" {
		cfg.Auth.SSHPassphrase = passphrase
	}

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// LoadOrDefault loads config or returns defaults if no config found
func LoadOrDefault() *Config {
	cfg, err := Load()
	if err != nil {
		// Fall back to defaults
		defaultCfg := DefaultConfig()
		return &defaultCfg
	}
	return cfg
}

// setDefaults sets default values in Viper
func setDefaults(v *viper.Viper) {
	defaults := DefaultConfig()

	// Global defaults
	v.SetDefault("global.default_mode", defaults.Global.DefaultMode)
	v.SetDefault("global.conflict_strategy", defaults.Global.ConflictStrategy)
	v.SetDefault("global.auto_commit_dirty", defaults.Global.AutoCommitDirty)
	v.SetDefault("global.scan_path", defaults.Global.ScanPath)
	v.SetDefault("global.scan_exclude", defaults.Global.ScanExclude)
	v.SetDefault("global.scan_depth", defaults.Global.ScanDepth)
	v.SetDefault("global.scan_workers", defaults.Global.ScanWorkers)
	v.SetDefault("global.cache_ttl", defaults.Global.CacheTTL)

	// Backup defaults
	v.SetDefault("backup.platform", defaults.Backup.Platform)
	v.SetDefault("backup.repo_template", defaults.Backup.RepoTemplate)
	v.SetDefault("backup.generate_manifest", defaults.Backup.GenerateManifest)

	// Auth defaults
	v.SetDefault("auth.use_ssh_agent", defaults.Auth.UseSSHAgent)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate mode
	validModes := map[string]bool{
		"push-known-branches": true,
		"push-all":            true,
		"leave-untouched":     true,
		"push-current-branch": true,
	}
	if !validModes[c.Global.DefaultMode] {
		return fmt.Errorf("invalid default_mode: %s (must be push-known-branches, push-all, push-current-branch, or leave-untouched)", c.Global.DefaultMode)
	}

	// Validate conflict strategy
	validStrategies := map[string]bool{
		"new-branch": true,
		"abort":      true,
	}
	if !validStrategies[c.Global.ConflictStrategy] {
		return fmt.Errorf("invalid conflict_strategy: %s (must be new-branch or abort)", c.Global.ConflictStrategy)
	}

	// Validate backup platform
	if c.Backup.Platform != "" {
		validPlatforms := map[string]bool{
			"github": true,
			"gitlab": true,
			"gitea":  true,
		}
		if !validPlatforms[c.Backup.Platform] {
			return fmt.Errorf("invalid platform: %s (must be github, gitlab, or gitea)", c.Backup.Platform)
		}
	}

	return nil
}

// FindRepoOverride finds a matching override for a repo path
func (c *Config) FindRepoOverride(repoPath, remoteURL string) *RepoOverride {
	for _, override := range c.Repos {
		// Match by path pattern
		if override.PathPattern != "" {
			matched, _ := filepath.Match(override.PathPattern, repoPath)
			if matched {
				return &override
			}
		}

		// Match by remote URL pattern
		if override.RemotePattern != "" && remoteURL != "" {
			// Simple substring match for now
			if strings.Contains(remoteURL, override.RemotePattern) {
				return &override
			}
		}
	}
	return nil
}

// WriteExampleConfig writes an example config file to the specified path
func WriteExampleConfig(path string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write example config
	content := ExampleConfigTOML()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultConfigPath returns the default user config path
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "git-fire", "config.toml")
}

// ParseDuration parses duration strings (supports Viper's format)
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}
