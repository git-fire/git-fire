package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/git-fire/git-fire/internal/executor"
)

func TestStatusGlyphFromEntry(t *testing.T) {
	if got := statusGlyph(executor.LogEntry{Level: "error", Action: "scan-failed"}); got != "❌" {
		t.Fatalf("statusGlyph(error) = %q, want ❌", got)
	}
	if got := statusGlyph(executor.LogEntry{Level: "info", Action: "scan-progress"}); got != "🔍" {
		t.Fatalf("statusGlyph(scan) = %q, want 🔍", got)
	}
}

func TestRenderLogExportText(t *testing.T) {
	out := renderLogExportText([]executor.LogEntry{
		{
			Timestamp:   time.Date(2026, 4, 19, 12, 0, 0, 0, time.UTC),
			Level:       "info",
			Action:      "scan-progress",
			Description: "found repo",
		},
	})
	if !strings.Contains(out, "scan-progress") {
		t.Fatalf("missing action in output: %q", out)
	}
	if !strings.Contains(out, "found repo") {
		t.Fatalf("missing description in output: %q", out)
	}
}

func TestExportLogEntriesText(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	path, err := exportLogEntriesText([]executor.LogEntry{
		{Timestamp: time.Now(), Level: "info", Action: "scan", Description: "ok"},
	})
	if err != nil {
		t.Fatalf("exportLogEntriesText() error = %v", err)
	}
	if !strings.Contains(path, "git-fire-ui-log-") {
		t.Fatalf("unexpected export path: %s", path)
	}
}

func TestRepoSelectorModel_ToggleLogPanel(t *testing.T) {
	m := NewRepoSelectorModel(nil, nil, "")
	if m.showLogPanel {
		t.Fatal("showLogPanel should start false")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	next := updated.(RepoSelectorModel)
	if !next.showLogPanel {
		t.Fatal("showLogPanel should be true after pressing l")
	}
}
