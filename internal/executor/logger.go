package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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

// Error logs an error message
func (l *Logger) Error(repo, action, description string, err error) {
	entry := LogEntry{
		Level:       "error",
		Repo:        repo,
		Action:      action,
		Description: description,
	}
	if err != nil {
		entry.Error = err.Error()
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
