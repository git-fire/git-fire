package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

type LoadOptions struct {
	ConfigFile string
}

// Load loads configuration from files and environment variables
// Priority (highest to lowest):
//  1. Environment variables (GIT_FIRE_*)
//  2. Explicit --config file (optional)
//  3. user config dir/git-fire/config.toml (user config)
//  4. Default config
func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

// LoadWithOptions loads config with optional explicit config file override.
func LoadWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("toml")

	// Add user config path via a shared resolver so read/write behavior stays aligned.
	userCfgDir, cfgWarning := resolvedUserConfigDir()
	v.AddConfigPath(userCfgDir)
	if cfgWarning != "" {
		fmt.Fprintf(os.Stderr, "warning: %s\n", cfgWarning)
	}
	v.AddConfigPath("/etc/git-fire")          // System config
	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)
	}

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
		// Config file not found - use defaults only when not explicitly requested.
		if opts.ConfigFile != "" {
			return nil, fmt.Errorf("config file not found: %s", opts.ConfigFile)
		}
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
	v.SetDefault("global.block_on_secrets", defaults.Global.BlockOnSecrets)
	v.SetDefault("global.scan_path", defaults.Global.ScanPath)
	v.SetDefault("global.scan_exclude", defaults.Global.ScanExclude)
	v.SetDefault("global.scan_depth", defaults.Global.ScanDepth)
	v.SetDefault("global.scan_workers", defaults.Global.ScanWorkers)
	v.SetDefault("global.push_workers", defaults.Global.PushWorkers)
	v.SetDefault("global.cache_ttl", defaults.Global.CacheTTL)
	v.SetDefault("global.rescan_submodules", defaults.Global.RescanSubmodules)
	v.SetDefault("global.disable_scan", defaults.Global.DisableScan)

	// Backup defaults
	v.SetDefault("backup.platform", defaults.Backup.Platform)
	v.SetDefault("backup.repo_template", defaults.Backup.RepoTemplate)
	v.SetDefault("backup.generate_manifest", defaults.Backup.GenerateManifest)

	// Auth defaults
	v.SetDefault("auth.use_ssh_agent", defaults.Auth.UseSSHAgent)

	// UI defaults
	v.SetDefault("ui.show_fire_animation", defaults.UI.ShowFireAnimation)
	v.SetDefault("ui.fire_tick_ms", defaults.UI.FireTickMS)
	v.SetDefault("ui.color_profile", defaults.UI.ColorProfile)
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

	// Validate UI color profile
	validProfiles := map[string]bool{}
	for _, name := range UIColorProfiles() {
		validProfiles[name] = true
	}
	if !validProfiles[c.UI.ColorProfile] {
		return fmt.Errorf("invalid ui.color_profile: %s (must be one of %s)", c.UI.ColorProfile, strings.Join(UIColorProfiles(), ", "))
	}

	// ui.fire_tick_ms: normalize and clamp before any time.Duration conversion.
	// Callers (cmd + TUI) use this as the scheduler period; reject absurd inputs here
	// so we never pass a sub-millisecond busy loop or multi-minute stall to tea.Tick.
	if c.UI.FireTickMS <= 0 {
		c.UI.FireTickMS = DefaultUIFireTickMS
	} else {
		if c.UI.FireTickMS < MinUIFireTickMS {
			c.UI.FireTickMS = MinUIFireTickMS
		} else if c.UI.FireTickMS > MaxUIFireTickMS {
			c.UI.FireTickMS = MaxUIFireTickMS
		}
	}

	if c.Global.PushWorkers <= 0 {
		c.Global.PushWorkers = DefaultPushWorkers
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

// DefaultConfigPath returns the default user config file path.
func DefaultConfigPath() string {
	userCfgDir, cfgWarning := resolvedUserConfigDir()
	if cfgWarning != "" {
		fmt.Fprintf(os.Stderr, "warning: %s\n", cfgWarning)
	}
	return filepath.Join(userCfgDir, "config.toml")
}

func userConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine user config directory: %w", err)
	}
	return filepath.Join(base, "git-fire"), nil
}

func fallbackUserConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory for fallback: %w", err)
	}
	if !filepath.IsAbs(home) {
		abs, absErr := filepath.Abs(home)
		if absErr != nil {
			return "", fmt.Errorf("fallback home directory is not absolute (%q): %w", home, absErr)
		}
		home = abs
	}
	return filepath.Join(home, ".config", "git-fire"), nil
}

func resolvedUserConfigDir() (string, string) {
	if dir, err := userConfigDir(); err == nil {
		return dir, ""
	}
	if dir, err := fallbackUserConfigDir(); err == nil {
		return dir, fmt.Sprintf("using fallback user config directory %q", dir)
	}
	if wd, err := os.Getwd(); err == nil {
		if !filepath.IsAbs(wd) {
			if abs, absErr := filepath.Abs(wd); absErr == nil {
				wd = abs
			}
		}
		if filepath.IsAbs(wd) {
			dir := filepath.Join(wd, "git-fire")
			return dir, fmt.Sprintf("using working-directory config fallback %q", dir)
		}
	}
	tempBase := os.TempDir()
	if !filepath.IsAbs(tempBase) {
		if abs, absErr := filepath.Abs(tempBase); absErr == nil {
			tempBase = abs
		}
	}
	dir := filepath.Join(tempBase, "git-fire")
	return dir, fmt.Sprintf("using temporary config fallback %q; this path may not persist across reboots", dir)
}

// ParseDuration parses duration strings (supports Viper's format)
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// tomlUnmarshal decodes TOML bytes into v using the same library as SaveConfig.
// Used by tests to verify round-trip correctness without going through viper.
func tomlUnmarshal(data []byte, v interface{}) error {
	return toml.Unmarshal(data, v)
}

// SaveConfig marshals cfg to TOML and atomically writes it to path.
// It creates the parent directory if necessary.
func SaveConfig(cfg *Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}
