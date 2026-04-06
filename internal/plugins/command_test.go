package plugins

import (
	"errors"
	"testing"
	"time"
)

type testLogger struct {
	messages []string
}

func (l *testLogger) Info(msg string) {
	l.messages = append(l.messages, "INFO: "+msg)
}

func (l *testLogger) Success(msg string) {
	l.messages = append(l.messages, "SUCCESS: "+msg)
}

func (l *testLogger) Error(msg string, err error) {
	if err != nil {
		l.messages = append(l.messages, "ERROR: "+msg+" "+err.Error())
		return
	}
	l.messages = append(l.messages, "ERROR: "+msg)
}

func (l *testLogger) Debug(msg string) {
	l.messages = append(l.messages, "DEBUG: "+msg)
}

func TestCommandPlugin_Basic(t *testing.T) {
	plugin := NewCommandPlugin("test-echo", "echo", []string{"hello", "world"})

	if plugin.Name() != "test-echo" {
		t.Errorf("Expected name 'test-echo', got '%s'", plugin.Name())
	}

	if plugin.Type() != PluginTypeCommand {
		t.Errorf("Expected type PluginTypeCommand, got %v", plugin.Type())
	}
}

func TestCommandPlugin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  *CommandPlugin
		wantErr bool
	}{
		{
			name:    "valid plugin",
			plugin:  NewCommandPlugin("test", "echo", []string{"hello"}),
			wantErr: false,
		},
		{
			name:    "missing name",
			plugin:  &CommandPlugin{command: "echo"},
			wantErr: true,
		},
		{
			name:    "missing command",
			plugin:  &CommandPlugin{name: "test"},
			wantErr: true,
		},
		{
			name:    "invalid command",
			plugin:  NewCommandPlugin("test", "nonexistent-command-xyz", []string{}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandPlugin_Execute(t *testing.T) {
	logger := &testLogger{}

	ctx := Context{
		RepoPath:  "/test/repo",
		RepoName:  "test-repo",
		Branch:    "main",
		CommitSHA: "abc123",
		Timestamp: time.Now(),
		Logger:    logger,
		DryRun:    false,
	}

	plugin := NewCommandPlugin("test-echo", "echo", []string{"hello"})

	err := plugin.Execute(ctx)
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}

	// Check that logger was called
	if len(logger.messages) == 0 {
		t.Error("Expected logger messages, got none")
	}
}

func TestCommandPlugin_DryRun(t *testing.T) {
	logger := &testLogger{}

	ctx := Context{
		RepoPath: "/test/repo",
		Logger:   logger,
		DryRun:   true,
	}

	plugin := NewCommandPlugin("test", "echo", []string{"hello"})

	err := plugin.Execute(ctx)
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}

	// In dry run, should only log
	if len(logger.messages) != 1 {
		t.Errorf("Expected 1 log message, got %d", len(logger.messages))
	}

	if !contains(logger.messages[0], "DRY RUN") {
		t.Error("Expected DRY RUN message")
	}
}

func TestCommandPlugin_VariableExpansion(t *testing.T) {
	plugin := NewCommandPlugin("test", "echo", []string{
		"{repo_name}",
		"{branch}",
		"{timestamp}",
	})

	ctx := Context{
		RepoName:  "my-repo",
		Branch:    "main",
		Timestamp: time.Date(2026, 2, 12, 15, 30, 0, 0, time.UTC),
	}

	expanded := plugin.expandVars("{repo_name}-{branch}-{timestamp}", ctx)

	if !contains(expanded, "my-repo") {
		t.Error("Expected expanded string to contain repo name")
	}

	if !contains(expanded, "main") {
		t.Error("Expected expanded string to contain branch")
	}

	if !contains(expanded, "20260212") {
		t.Error("Expected expanded string to contain timestamp")
	}
}

func TestCommandPlugin_Timeout(t *testing.T) {
	logger := &testLogger{}

	ctx := Context{
		RepoPath: "/test/repo",
		Logger:   logger,
		DryRun:   false,
	}

	// Create plugin with very short timeout
	plugin := NewCommandPlugin("test-sleep", "sleep", []string{"10"})
	plugin.SetTimeout(100 * time.Millisecond)

	err := plugin.Execute(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !contains(err.Error(), "timeout") && !contains(err.Error(), "timed out") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}

func TestCommandPlugin_Cleanup(t *testing.T) {
	plugin := NewCommandPlugin("cleanup-test", "echo", []string{"ok"})
	if err := plugin.Cleanup(); err != nil {
		t.Fatalf("Cleanup() should return nil, got %v", err)
	}
}

func TestCommandPlugin_Execute_SanitizesStderr(t *testing.T) {
	logger := &testLogger{}

	ctx := Context{
		RepoPath: "/test/repo",
		Logger:   logger,
		DryRun:   false,
	}

	plugin := NewCommandPlugin("test-stderr", "sh", []string{"-c", "echo 'fatal https://user:secret@github.com/repo.git' 1>&2; exit 1"})

	err := plugin.Execute(ctx)
	if err == nil {
		t.Fatal("Expected command error")
	}

	if contains(err.Error(), "user:secret@") {
		t.Fatalf("Expected sanitized error, got %q", err.Error())
	}
	if !contains(err.Error(), "[REDACTED]") {
		t.Fatalf("Expected masked secret marker in error, got %q", err.Error())
	}
}

func TestCommandPlugin_Execute_SanitizesStdoutDebug(t *testing.T) {
	logger := &testLogger{}

	ctx := Context{
		RepoPath: "/test/repo",
		Logger:   logger,
		DryRun:   false,
	}

	plugin := NewCommandPlugin("test-stdout", "sh", []string{"-c", "echo 'https://user:secret@github.com/repo.git'"})

	err := plugin.Execute(ctx)
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	joined := ""
	for _, m := range logger.messages {
		joined += m
	}
	if contains(joined, "user:secret@") {
		t.Fatalf("Expected sanitized logger output, got %q", joined)
	}
	if !contains(joined, "[REDACTED]") {
		t.Fatalf("Expected masked secret marker in logger output, got %q", joined)
	}
}

func TestTestLoggerErrorIncludesErr(t *testing.T) {
	logger := &testLogger{}
	logger.Error("boom", errors.New("details"))
	if !contains(logger.messages[0], "details") {
		t.Fatalf("expected logger to include error details, got %q", logger.messages[0])
	}
}

// Helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
