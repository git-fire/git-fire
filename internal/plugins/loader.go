package plugins

import (
	"fmt"
	"time"

	"github.com/git-fire/git-fire/internal/config"
)

// LoadFromConfig loads and registers plugins from configuration
func LoadFromConfig(cfg *config.Config) error {
	// Load command plugins
	for _, cmdCfg := range cfg.Plugins.Command {
		plugin, err := createCommandPlugin(cmdCfg)
		if err != nil {
			return fmt.Errorf("failed to create command plugin %s: %w", cmdCfg.Name, err)
		}

		if plugin.When() == TriggerMergeConflict {
			if err := RegisterConflictResolver(plugin); err != nil {
				return fmt.Errorf("failed to register conflict resolver %s: %w", cmdCfg.Name, err)
			}
			// Also register in the global plugin list so `plugins.enabled` can name it;
			// post-run dispatch skips on-merge-conflict triggers.
			if err := Register(plugin); err != nil {
				return fmt.Errorf("failed to register plugin %s: %w", cmdCfg.Name, err)
			}
			continue
		}

		if err := Register(plugin); err != nil {
			return fmt.Errorf("failed to register plugin %s: %w", cmdCfg.Name, err)
		}
	}

	// TODO: Load webhook plugins when implemented
	// TODO: Load Go plugins when implemented

	return nil
}

// createCommandPlugin creates a command plugin from config
func createCommandPlugin(cfg config.CommandPluginConfig) (*CommandPlugin, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}

	if cfg.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	plugin := NewCommandPlugin(cfg.Name, cfg.Command, cfg.Args)

	// Set environment variables
	for key, value := range cfg.Env {
		plugin.SetEnv(key, value)
	}

	// Parse and set timeout
	if cfg.Timeout != "" {
		timeout, err := time.ParseDuration(cfg.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid timeout %s: %w", cfg.Timeout, err)
		}
		plugin.SetTimeout(timeout)
	}

	// Parse and set trigger
	if cfg.When != "" {
		trigger := parseTrigger(cfg.When)
		plugin.SetTrigger(trigger)
	}
	plugin.SetFailRun(cfg.FailRun)

	return plugin, nil
}

// parseTrigger converts string to Trigger type
func parseTrigger(when string) Trigger {
	switch when {
	case "before-push":
		return TriggerOnSuccess
	case "after-push":
		return TriggerOnSuccess
	case "on-success":
		return TriggerOnSuccess
	case "on-failure":
		return TriggerOnFailure
	case "always":
		return TriggerAlways
	case "on-merge-conflict":
		return TriggerMergeConflict
	default:
		return TriggerOnSuccess // Default
	}
}

// GetEnabledPlugins returns only enabled plugins from config
func GetEnabledPlugins(cfg *config.Config) ([]Plugin, error) {
	if len(cfg.Plugins.Enabled) == 0 {
		// If no enabled list, return all registered plugins
		return List(), nil
	}

	var enabled []Plugin
	for _, name := range cfg.Plugins.Enabled {
		plugin, err := Get(name)
		if err != nil {
			return nil, fmt.Errorf("enabled plugin %s not found: %w", name, err)
		}
		enabled = append(enabled, plugin)
	}

	return enabled, nil
}

// GetEnabledConflictResolvers returns conflict resolvers that should run during
// planning, respecting the same enabled list as other plugins. When enabled is
// empty, all registered resolvers run.
func GetEnabledConflictResolvers(cfg *config.Config) ([]ConflictResolver, error) {
	all := ListConflictResolvers()
	if cfg == nil || len(cfg.Plugins.Enabled) == 0 {
		return all, nil
	}
	enabledSet := make(map[string]struct{}, len(cfg.Plugins.Enabled))
	for _, name := range cfg.Plugins.Enabled {
		enabledSet[name] = struct{}{}
	}
	var out []ConflictResolver
	for _, r := range all {
		if _, ok := enabledSet[r.Name()]; ok {
			out = append(out, r)
		}
	}
	return out, nil
}

// FilterPluginsByTrigger filters plugins by when they should run
func FilterPluginsByTrigger(plugins []Plugin, trigger Trigger) []Plugin {
	var filtered []Plugin
	want := canonicalTrigger(trigger)
	for _, p := range plugins {
		// Check if plugin is a command plugin (has trigger)
		if cmd, ok := p.(*CommandPlugin); ok {
			if cmd.when == TriggerMergeConflict {
				continue
			}
			if canonicalTrigger(cmd.when) == want {
				filtered = append(filtered, p)
			}
		}
	}
	return filtered
}

func canonicalTrigger(trigger Trigger) Trigger {
	switch trigger {
	case TriggerOnFailure, TriggerAlways, TriggerMergeConflict:
		return trigger
	default:
		return TriggerOnSuccess
	}
}
