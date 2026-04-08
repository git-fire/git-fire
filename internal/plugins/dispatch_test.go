package plugins

import (
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/config"
)

func resetPluginRegistry(t *testing.T) {
	t.Helper()
	globalRegistry.Clear()
	t.Cleanup(func() {
		globalRegistry.Clear()
	})
}

func TestParseTrigger(t *testing.T) {
	tests := []struct {
		input string
		want  Trigger
	}{
		{"before-push", TriggerOnSuccess},
		{"after-push", TriggerOnSuccess},
		{"on-success", TriggerOnSuccess},
		{"on-failure", TriggerOnFailure},
		{"always", TriggerAlways},
		{"unknown", TriggerOnSuccess},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTrigger(tt.input)
			if got != tt.want {
				t.Fatalf("parseTrigger(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterPluginsByTrigger_OnSuccess(t *testing.T) {
	plugin := NewCommandPlugin("success", "echo", []string{"ok"})
	plugin.SetTrigger(TriggerOnSuccess)

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerOnSuccess)
	if len(filtered) != 1 || filtered[0].Name() != "success" {
		t.Fatalf("expected on-success plugin to match, got %d plugins", len(filtered))
	}
}

func TestFilterPluginsByTrigger_NoMatch(t *testing.T) {
	plugin := NewCommandPlugin("success", "echo", []string{"ok"})
	plugin.SetTrigger(TriggerOnSuccess)

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerOnFailure)
	if len(filtered) != 0 {
		t.Fatalf("expected no matches, got %d", len(filtered))
	}
}

func TestFilterPluginsByTrigger_AfterPushAliasesOnSuccess(t *testing.T) {
	plugin := NewCommandPlugin("alias", "echo", []string{"ok"})
	plugin.SetTrigger(parseTrigger("after-push"))

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerOnSuccess)
	if len(filtered) != 1 || filtered[0].Name() != "alias" {
		t.Fatalf("expected after-push alias to match on-success, got %d plugins", len(filtered))
	}
}

func TestFilterPluginsByTrigger_ProgrammaticAfterPushMatchesOnSuccess(t *testing.T) {
	plugin := NewCommandPlugin("programmatic-after", "echo", []string{"ok"})
	plugin.SetTrigger(TriggerAfterPush)

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerOnSuccess)
	if len(filtered) != 1 || filtered[0].Name() != "programmatic-after" {
		t.Fatalf("expected TriggerAfterPush to match on-success dispatch, got %d plugins", len(filtered))
	}
}

func TestFilterPluginsByTrigger_ProgrammaticBeforePushMatchesOnSuccess(t *testing.T) {
	plugin := NewCommandPlugin("programmatic-before", "echo", []string{"ok"})
	plugin.SetTrigger(TriggerBeforePush)

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerOnSuccess)
	if len(filtered) != 1 || filtered[0].Name() != "programmatic-before" {
		t.Fatalf("expected TriggerBeforePush to match on-success dispatch, got %d plugins", len(filtered))
	}
}

func TestFilterPluginsByTrigger_Always(t *testing.T) {
	plugin := NewCommandPlugin("always", "echo", []string{"ok"})
	plugin.SetTrigger(TriggerAlways)

	filtered := FilterPluginsByTrigger([]Plugin{plugin}, TriggerAlways)
	if len(filtered) != 1 || filtered[0].Name() != "always" {
		t.Fatalf("expected always plugin to match, got %d plugins", len(filtered))
	}
}

func TestCommandPlugin_FailRun_DefaultFalse(t *testing.T) {
	plugin := NewCommandPlugin("test", "echo", []string{"ok"})
	if plugin.FailRun() {
		t.Fatalf("expected fail_run default false")
	}
}

func TestCommandPlugin_FailRun_SetTrue(t *testing.T) {
	plugin := NewCommandPlugin("test", "echo", []string{"ok"})
	plugin.SetFailRun(true)
	if !plugin.FailRun() {
		t.Fatalf("expected fail_run true after SetFailRun(true)")
	}
}

func TestCommandPlugin_Execute_DryRun(t *testing.T) {
	logger := &testLogger{}
	plugin := NewCommandPlugin("dry", "echo", []string{"hello"})
	ctx := Context{
		DryRun:    true,
		Timestamp: time.Now(),
		Logger:    logger,
	}

	if err := plugin.Execute(ctx); err != nil {
		t.Fatalf("Execute() dry-run error = %v", err)
	}
	if len(logger.messages) != 1 {
		t.Fatalf("expected one dry-run log message, got %d", len(logger.messages))
	}
	if !contains(logger.messages[0], "DRY RUN") {
		t.Fatalf("expected dry-run log output, got %q", logger.messages[0])
	}
}

func TestLoadFromConfig_WiresFailRun(t *testing.T) {
	resetPluginRegistry(t)
	cfg := &config.Config{
		Plugins: config.PluginsConfig{
			Command: []config.CommandPluginConfig{
				{
					Name:    "fail-run-enabled",
					Command: "echo",
					Args:    []string{"hello"},
					FailRun: true,
				},
			},
		},
	}

	if err := LoadFromConfig(cfg); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	got, err := Get("fail-run-enabled")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	cmd, ok := got.(*CommandPlugin)
	if !ok {
		t.Fatalf("expected CommandPlugin, got %T", got)
	}
	if !cmd.FailRun() {
		t.Fatalf("expected fail_run to be wired as true")
	}
}

func TestLoadFromConfig_FailRunDefaultFalse(t *testing.T) {
	resetPluginRegistry(t)
	cfg := &config.Config{
		Plugins: config.PluginsConfig{
			Command: []config.CommandPluginConfig{
				{
					Name:    "fail-run-default",
					Command: "echo",
					Args:    []string{"hello"},
				},
			},
		},
	}

	if err := LoadFromConfig(cfg); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	got, err := Get("fail-run-default")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	cmd, ok := got.(*CommandPlugin)
	if !ok {
		t.Fatalf("expected CommandPlugin, got %T", got)
	}
	if cmd.FailRun() {
		t.Fatalf("expected fail_run default false when omitted")
	}
}
