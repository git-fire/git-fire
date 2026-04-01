package executor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	dir := t.TempDir()

	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	if logger.LogPath() == "" {
		t.Error("LogPath() should not be empty")
	}

	// Log file must be inside the dir
	if !strings.HasPrefix(logger.LogPath(), dir) {
		t.Errorf("log file %q not under dir %q", logger.LogPath(), dir)
	}

	// File should exist
	if _, err := os.Stat(logger.LogPath()); os.IsNotExist(err) {
		t.Error("log file was not created")
	}
}

func TestLogger_FilePermissions(t *testing.T) {
	dir := t.TempDir()

	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	info, err := os.Stat(logger.LogPath())
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", logger.LogPath(), err)
	}

	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Errorf("log file permissions = %04o, want 0600", perm)
	}
}

func TestLogger_Info(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Info("myrepo", "test-action", "hello world")

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	if !strings.Contains(string(data), "hello world") {
		t.Error("log file should contain the info message")
	}
}

func TestLogger_Error(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	testErr := &testError{"something failed"}
	logger.Error("myrepo", "push", "push failed", testErr)

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	if !strings.Contains(string(data), "something failed") {
		t.Error("log file should contain error message")
	}
}

func TestLogger_Error_MasksURLCredentials(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Error("myrepo", "push", "push failed",
		&testError{"remote: https://user:s3cr3tpassword@github.com/org/repo.git"})

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	content := string(data)
	if strings.Contains(content, "s3cr3tpassword") {
		t.Error("log file must not contain the raw password")
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Error("log file should contain [REDACTED] in place of credentials")
	}
}

func TestLogger_Error_MasksKeyValue(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Error("myrepo", "push", "auth failed",
		&testError{"API_KEY=supersecrettoken123"})

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	content := string(data)
	if strings.Contains(content, "supersecrettoken123") {
		t.Error("log file must not contain the raw API key value")
	}
	if !strings.Contains(content, "[REDACTED]") {
		t.Error("log file should contain [REDACTED] in place of key value")
	}
}

func TestLogger_Error_Nil(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	// Should not panic on nil error
	logger.Error("repo", "action", "desc", nil)
}

func TestLogger_Success(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Success("myrepo", "push", "pushed 3 branches", 2*time.Second)

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	if !strings.Contains(string(data), "pushed 3 branches") {
		t.Error("log file should contain success message")
	}
}

func TestLogger_LogResult(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	result := &ExecutionResult{
		Success:  2,
		Failed:   1,
		Skipped:  0,
		Duration: 5 * time.Second,
		RepoResults: []RepoResult{
			{
				Path:           "/repos/a",
				Success:        true,
				PushedBranches: []string{"main"},
				Duration:       time.Second,
			},
			{
				Path:    "/repos/b",
				Success: false,
				Error:   &testError{"network error"},
			},
		},
	}

	logger.LogResult(result)

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "2 success") {
		t.Error("LogResult should write success count")
	}
	if !strings.Contains(content, "network error") {
		t.Error("LogResult should write repo error")
	}
}

func TestLogger_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info("repo", "action", "description")
	logger.Error("repo", "action", "failed", &testError{"oops"})
	logger.Success("repo", "action", "done", time.Second)
	logger.Close()

	data, err := os.ReadFile(logger.LogPath())
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}

	// Every non-empty line must be valid JSON
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var entry LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("log line is not valid JSON: %q — error: %v", line, err)
		}
	}
}

func TestLogger_Close(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir)
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestDefaultLogDir(t *testing.T) {
	dir := DefaultLogDir()
	if dir == "" {
		t.Error("DefaultLogDir() should not be empty")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("DefaultLogDir() = %q, should be absolute", dir)
	}
}

func TestDefaultLogDir_UsesUserCacheDir(t *testing.T) {
	xdgCache := filepath.Join(t.TempDir(), "xdg-cache")
	t.Setenv("XDG_CACHE_HOME", xdgCache)

	dir := DefaultLogDir()
	want := filepath.Join(xdgCache, "git-fire", "logs")
	if dir != want {
		t.Fatalf("expected log dir %q, got %q", want, dir)
	}
}

// testError is a minimal error implementation for test use.
type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
