// Package plugins loads and runs optional command and webhook hooks around backup runs.
package plugins

import (
	"errors"
	"time"

	"github.com/git-fire/git-fire/internal/executor"
)

// Plugin is the core interface all plugins must implement
type Plugin interface {
	// Name returns the plugin name (e.g., "s3-upload")
	Name() string

	// Type returns the plugin type (command, webhook, go-plugin, etc.)
	Type() PluginType

	// Validate checks if the plugin configuration is valid
	Validate() error

	// Execute runs the plugin with the given context
	Execute(ctx Context) error

	// Cleanup performs any cleanup after execution
	Cleanup() error
}

// PluginType defines the type of plugin
type PluginType string

const (
	PluginTypeCommand PluginType = "command"
	PluginTypeWebhook PluginType = "webhook"
	PluginTypeGo      PluginType = "go"
)

var ErrPluginFailed = errors.New("plugin failed")

// Context provides execution context to plugins
type Context struct {
	// Repository information
	RepoPath  string
	RepoName  string
	Branch    string
	CommitSHA string
	Remotes   []string

	// Execution context
	Timestamp time.Time
	DryRun    bool
	Emergency bool // True if running in emergency mode

	// Resources
	Logger Logger
	Config map[string]interface{} // Plugin-specific config

	// Results from previous steps
	PushResult *executor.RepoResult
}

// Logger interface for plugin logging
type Logger interface {
	Info(message string)
	Success(message string)
	Error(message string, err error)
	Debug(message string)
}

// Result represents the result of plugin execution
type Result struct {
	Success  bool
	Message  string
	Error    error
	Data     map[string]interface{} // Plugin-specific result data
	Duration time.Duration
}

// Trigger defines when a plugin should execute
type Trigger string

const (
	TriggerBeforePush Trigger = TriggerOnSuccess
	TriggerAfterPush  Trigger = TriggerOnSuccess
	TriggerOnSuccess  Trigger = "on-success"
	TriggerOnFailure  Trigger = "on-failure"
	TriggerAlways     Trigger = "always"
)

// PluginConfig represents plugin configuration from config file
type PluginConfig struct {
	Name    string                 `mapstructure:"name"`
	Type    string                 `mapstructure:"type"`
	When    string                 `mapstructure:"when"`
	Enabled bool                   `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

// TemplateVars contains variables available for template substitution
type TemplateVars struct {
	RepoPath  string
	RepoName  string
	Branch    string
	CommitSHA string
	Timestamp string
	Date      string
	Time      string
	Hostname  string
	Username  string
}
