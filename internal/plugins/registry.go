package plugins

import (
	"fmt"
	"sync"
)

// Registry manages all registered plugins
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// Global registry instance
var globalRegistry = &Registry{
	plugins: make(map[string]Plugin),
}

// Register adds a plugin to the global registry
func Register(plugin Plugin) error {
	return globalRegistry.Register(plugin)
}

// Get retrieves a plugin by name from the global registry
func Get(name string) (Plugin, error) {
	return globalRegistry.Get(name)
}

// List returns all registered plugins
func List() []Plugin {
	return globalRegistry.List()
}

// Register adds a plugin to this registry
func (r *Registry) Register(plugin Plugin) error {
	if plugin == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Validate plugin before registration
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("plugin %s validation failed: %w", name, err)
	}

	r.plugins[name] = plugin
	return nil
}

// Get retrieves a plugin by name
func (r *Registry) Get(name string) (Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// List returns all registered plugins
func (r *Registry) List() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// Clear removes all plugins (useful for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]Plugin)
}

// Exists checks if a plugin is registered
func (r *Registry) Exists(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.plugins[name]
	return exists
}
