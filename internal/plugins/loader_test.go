package plugins

import (
	"testing"
	"time"

	"github.com/TBRX103/git-fire/internal/config"
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

func TestParseTrigger(t *testing.T) {
	tests := []struct {
		input string
		want  Trigger
	}{
		{"before-push", TriggerBeforePush},
		{"after-push", TriggerAfterPush},
		{"on-success", TriggerOnSuccess},
		{"on-failure", TriggerOnFailure},
		{"always", TriggerAlways},
		{"unknown", TriggerAfterPush}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTrigger(tt.input)
			if got != tt.want {
				t.Errorf("parseTrigger(%s) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetEnabledPlugins(t *testing.T) {
	globalRegistry.Clear()

	// Register some plugins
	Register(NewCommandPlugin("plugin1", "echo", []string{"1"}))
	Register(NewCommandPlugin("plugin2", "echo", []string{"2"}))
	Register(NewCommandPlugin("plugin3", "echo", []string{"3"}))

	tests := []struct {
		name         string
		enabledList  []string
		wantCount    int
		wantErr      bool
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

func TestFilterPluginsByTrigger(t *testing.T) {
	plugin1 := NewCommandPlugin("after", "echo", []string{"1"})
	plugin1.SetTrigger(TriggerAfterPush)

	plugin2 := NewCommandPlugin("before", "echo", []string{"2"})
	plugin2.SetTrigger(TriggerBeforePush)

	plugin3 := NewCommandPlugin("always", "echo", []string{"3"})
	plugin3.SetTrigger(TriggerAlways)

	allPlugins := []Plugin{plugin1, plugin2, plugin3}

	tests := []struct {
		name      string
		trigger   Trigger
		wantCount int
		wantNames []string
	}{
		{
			name:      "after-push trigger",
			trigger:   TriggerAfterPush,
			wantCount: 2, // after + always
			wantNames: []string{"after", "always"},
		},
		{
			name:      "before-push trigger",
			trigger:   TriggerBeforePush,
			wantCount: 2, // before + always
			wantNames: []string{"before", "always"},
		},
		{
			name:      "on-success trigger",
			trigger:   TriggerOnSuccess,
			wantCount: 1, // only always
			wantNames: []string{"always"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterPluginsByTrigger(allPlugins, tt.trigger)

			if len(filtered) != tt.wantCount {
				t.Errorf("FilterPluginsByTrigger() count = %d, want %d", len(filtered), tt.wantCount)
			}

			// Check names
			for _, wantName := range tt.wantNames {
				found := false
				for _, p := range filtered {
					if p.Name() == wantName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected plugin %s in filtered results", wantName)
				}
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
