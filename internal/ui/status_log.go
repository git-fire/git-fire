package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-fire/git-fire/internal/executor"
)

func statusGlyph(e executor.LogEntry) string {
	switch e.Level {
	case "success":
		return "✅"
	case "error":
		return "❌"
	default:
		switch {
		case strings.Contains(e.Action, "scan"):
			return "🔍"
		case strings.Contains(e.Action, "export"):
			return "📦"
		default:
			return "ℹ️"
		}
	}
}

func renderLogExportText(entries []executor.LogEntry) string {
	var b strings.Builder
	for _, e := range entries {
		ts := e.Timestamp.Format(time.RFC3339)
		fmt.Fprintf(&b, "%s [%s] %s %s", ts, e.Level, e.Action, e.Description)
		if e.Error != "" {
			fmt.Fprintf(&b, " err=%s", e.Error)
		}
		if e.Duration != "" {
			fmt.Fprintf(&b, " duration=%s", e.Duration)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// defaultExportDir places exports beside the session log directory so both
// share the same cache-dir fallback chain.
func defaultExportDir() string {
	return filepath.Join(filepath.Dir(executor.DefaultLogDir()), "exports")
}

func exportLogEntriesText(entries []executor.LogEntry) (string, error) {
	exportDir := defaultExportDir()
	if err := os.MkdirAll(exportDir, 0o700); err != nil {
		return "", fmt.Errorf("create export dir: %w", err)
	}
	path := filepath.Join(exportDir, fmt.Sprintf("git-fire-ui-log-%s.txt", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, []byte(renderLogExportText(entries)), 0o600); err != nil {
		return "", fmt.Errorf("write export file: %w", err)
	}
	return path, nil
}
