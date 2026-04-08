package plugins

import (
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/config"
)

func TestLoadFromConfig(t *testing.T) {
	// Clear registry before test
	globalRegistry.Clear()

	cfg := &config.Config{
		Plugins: config.PluginsConfig{
			Command: []config.CommandPluginConfig{
				{
					Name:    "test-echo",
					Command: "echo",
					Args:    []string{"hello"},
					When:    "after-push",
					Timeout: "5m",
				},
				{
					Name:    "test-ls",
					Command: "ls",
					Args:    []string{"-la"},
					Env: map[string]string{
						"TEST_VAR": "test_value",
					},
				},
			},
		},
	}

	err := LoadFromConfig(cfg)
	if err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Check that plugins were registered
	plugin1, err := Get("test-echo")
	if err != nil {
		t.Errorf("Expected plugin test-echo to be registered")
	}

	if plugin1.Name() != "test-echo" {
		t.Errorf("Expected name test-echo, got %s", plugin1.Name())
	}

	plugin2, err := Get("test-ls")
	if err != nil {
		t.Errorf("Expected plugin test-ls to be registered")
	}

	if plugin2.Name() != "test-ls" {
		t.Errorf("Expected name test-ls, got %s", plugin2.Name())
	}
}

func TestLoadFromConfig_InvalidPlugin(t *testing.T) {
	globalRegistry.Clear()

	cfg := &config.Config{
		Plugins: config.PluginsConfig{
			Command: []config.CommandPluginConfig{
				{
					Name:    "invalid",
					Command: "", // Missing command
				},
			},
		},
	}

	err := LoadFromConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid plugin config")
	}
}

func TestCreateCommandPlugin(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.CommandPluginConfig
		wantErr bool
	}{
		{
			name: "valid plugin",
			cfg: config.CommandPluginConfig{
				Name:    "test",
				Command: "echo",
				Args:    []string{"hello"},
				Timeout: "30s",
				When:    "after-push",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			cfg: config.CommandPluginConfig{
				Command: "echo",
			},
			wantErr: true,
		},
		{
			name: "missing command",
			cfg: config.CommandPluginConfig{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			cfg: config.CommandPluginConfig{
				Name:    "test",
				Command: "echo",
				Timeout: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, err := createCommandPlugin(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("createCommandPlugin() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && plugin == nil {
				t.Error("Expected non-nil plugin")
			}
		})
	}
}

func TestCreateCommandPlugin_DefaultTriggerOnSuccess(t *testing.T) {
	cfg := config.CommandPluginConfig{
		Name:    "default-trigger",
		Command: "echo",
		Args:    []string{"hello"},
	}

	plugin, err := createCommandPlugin(cfg)
	if err != nil {
		t.Fatalf("createCommandPlugin() error = %v", err)
	}
	if plugin.when != TriggerOnSuccess {
		t.Fatalf("expected default trigger %q, got %q", TriggerOnSuccess, plugin.when)
	}
}

func TestGetEnabledPlugins(t *testing.T) {
	globalRegistry.Clear()

	// Register some plugins
	_ = Register(NewCommandPlugin("plugin1", "echo", []string{"1"}))
	_ = Register(NewCommandPlugin("plugin2", "echo", []string{"2"}))
	_ = Register(NewCommandPlugin("plugin3", "echo", []string{"3"}))

	tests := []struct {
		name        string
		enabledList []string
		wantCount   int
		wantErr     bool
	}{
		{
			name:        "no enabled list returns all",
			enabledList: []string{},
			wantCount:   3,
			wantErr:     false,
		},
		{
			name:        "specific plugins enabled",
			enabledList: []string{"plugin1", "plugin2"},
			wantCount:   2,
			wantErr:     false,
		},
		{
			name:        "non-existent plugin",
			enabledList: []string{"plugin1", "nonexistent"},
			wantCount:   0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Plugins: config.PluginsConfig{
					Enabled: tt.enabledList,
				},
			}

			plugins, err := GetEnabledPlugins(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEnabledPlugins() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(plugins) != tt.wantCount {
				t.Errorf("GetEnabledPlugins() count = %d, want %d", len(plugins), tt.wantCount)
			}
		})
	}
}

func TestTimeoutParsing(t *testing.T) {
	cfg := config.CommandPluginConfig{
		Name:    "test",
		Command: "echo",
		Args:    []string{"hello"},
		Timeout: "2m30s",
	}

	plugin, err := createCommandPlugin(cfg)
	if err != nil {
		t.Fatalf("createCommandPlugin() error = %v", err)
	}

	expectedTimeout := 2*time.Minute + 30*time.Second
	if plugin.timeout != expectedTimeout {
		t.Errorf("Expected timeout %v, got %v", expectedTimeout, plugin.timeout)
	}
}
