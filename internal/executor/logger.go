package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// secretPatterns matches common credential patterns that should be redacted
// from log output before persisting to disk.
var secretPatterns = []*regexp.Regexp{
	// HTTPS URLs with embedded credentials: https://user:pass@host
	regexp.MustCompile(`(?i)(https?://)[^:@/\s]+:[^@/\s]+@`),
	// AWS access keys
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	// Generic token/key/password/secret in key=value form
	regexp.MustCompile(`(?i)(token|key|password|secret|passwd|api_key|apikey)\s*[:=]\s*\S+`),
	// GitHub tokens
	regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b`),
}

// maskSecrets redacts common credential patterns from a string.
func maskSecrets(s string) string {
	for _, re := range secretPatterns {
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			// For URL credentials keep the scheme and host visible.
			if strings.HasPrefix(strings.ToLower(m), "http") {
				// Replace user:pass@ with [REDACTED]@
				return re.ReplaceAllString(m, "${1}[REDACTED]@")
			}
			// For key=value forms keep the key name.
			idx := strings.IndexAny(m, ":=")
			if idx >= 0 {
				return m[:idx+1] + "[REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return s
}

// Logger handles structured logging of git-fire operations
type Logger struct {
	logPath string
	file    *os.File
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	Repo        string    `json:"repo,omitempty"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	Error       string    `json:"error,omitempty"`
	Duration    string    `json:"duration,omitempty"`
}

// NewLogger creates a new logger
func NewLogger(logDir string) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file with timestamp
	logFilename := fmt.Sprintf("git-fire-%s.log", time.Now().Format("20060102-150405"))
	logPath := filepath.Join(logDir, logFilename)

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logger := &Logger{
		logPath: logPath,
		file:    file,
	}

	// Write header
	logger.Info("", "git-fire-start", "Git-fire session started")

	return logger, nil
}

// Log writes a log entry
func (l *Logger) Log(entry LogEntry) error {
	if l.file == nil {
		return fmt.Errorf("logger not initialized")
	}

	entry.Timestamp = time.Now()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return l.file.Sync()
}

// Info logs an info message
func (l *Logger) Info(repo, action, description string) {
	l.Log(LogEntry{
		Level:       "info",
		Repo:        repo,
		Action:      action,
		Description: description,
	})
}

// Error logs an error message. The error string is masked for common secret
// patterns (e.g. HTTPS credentials embedded in git remote URLs) before being
// written to disk.
func (l *Logger) Error(repo, action, description string, err error) {
	entry := LogEntry{
		Level:       "error",
		Repo:        repo,
		Action:      action,
		Description: description,
	}
	if err != nil {
		entry.Error = maskSecrets(err.Error())
	}
	l.Log(entry)
}

// Success logs a successful action with duration
func (l *Logger) Success(repo, action, description string, duration time.Duration) {
	l.Log(LogEntry{
		Level:       "success",
		Repo:        repo,
		Action:      action,
		Description: description,
		Duration:    duration.String(),
	})
}

// LogResult logs the execution result
func (l *Logger) LogResult(result *ExecutionResult) {
	l.Info("", "execution-complete", fmt.Sprintf(
		"Execution complete: %d success, %d failed, %d skipped (duration: %s)",
		result.Success, result.Failed, result.Skipped, result.Duration,
	))

	for _, repoResult := range result.RepoResults {
		if repoResult.Success {
			l.Success(
				repoResult.Path,
				"repo-complete",
				fmt.Sprintf("Successfully pushed %d branches", len(repoResult.PushedBranches)),
				repoResult.Duration,
			)
		} else if repoResult.Error != nil {
			l.Error(
				repoResult.Path,
				"repo-failed",
				"Repository push failed",
				repoResult.Error,
			)
		}
	}
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file == nil {
		return nil
	}

	l.Info("", "git-fire-end", "Git-fire session ended")
	return l.file.Close()
}

// LogPath returns the path to the log file
func (l *Logger) LogPath() string {
	return l.logPath
}

// DefaultLogDir returns the default log directory
func DefaultLogDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "git-fire", "logs")
}
