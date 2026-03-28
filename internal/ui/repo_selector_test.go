package ui

import (
	"strings"
	"testing"

	"github.com/TBRX103/git-fire/internal/git"
	tea "github.com/charmbracelet/bubbletea"
)

// press builds a key message for a printable character (e.g. 'j', 'q', ' ').
func press(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// pressSpecial builds a key message for a named key (Enter, Up, Down, Space, etc.).
func pressSpecial(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

// updateLite sends a message to a RepoSelectorLiteModel and returns the updated model.
func updateLite(t *testing.T, m RepoSelectorLiteModel, msg tea.Msg) RepoSelectorLiteModel {
	t.Helper()
	updated, _ := m.Update(msg)
	typed, ok := updated.(RepoSelectorLiteModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorLiteModel", updated)
	}
	return typed
}

// sampleRepos builds a small slice of git.Repository for tests.
func sampleRepos() []git.Repository {
	return []git.Repository{
		{Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
		{Name: "beta", Selected: false, Mode: git.ModePushKnownBranches},
		{Name: "gamma", Selected: true, Mode: git.ModePushAll},
	}
}

// --- RepoSelectorLiteModel tests ---

func TestNewRepoSelectorLiteModel_DefaultSelection(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorLiteModel(repos)

	// selected map should mirror repo.Selected
	if !m.selected[0] {
		t.Error("repo 0 (Selected=true) should be selected initially")
	}
	if m.selected[1] {
		t.Error("repo 1 (Selected=false) should not be selected initially")
	}
	if !m.selected[2] {
		t.Error("repo 2 (Selected=true) should be selected initially")
	}
}

func TestNewRepoSelectorLiteModel_InitialCursor(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
}

func TestRepoSelectorLiteModel_Init(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for the lite model")
	}
}

func TestRepoSelectorLiteModel_GetSelectedRepos(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	selected := m.GetSelectedRepos()

	// Only alpha (0) and gamma (2) are selected
	if len(selected) != 2 {
		t.Errorf("GetSelectedRepos() returned %d repos, want 2", len(selected))
	}
	for _, r := range selected {
		if r.Name != "alpha" && r.Name != "gamma" {
			t.Errorf("unexpected repo in selection: %s", r.Name)
		}
	}
}

func TestRepoSelectorLiteModel_GetSelectedRepos_NoneSelected(t *testing.T) {
	repos := []git.Repository{
		{Name: "a", Selected: false},
		{Name: "b", Selected: false},
	}
	m := NewRepoSelectorLiteModel(repos)
	if len(m.GetSelectedRepos()) != 0 {
		t.Error("expected no selected repos")
	}
}

// --- View output ---

func TestRepoSelectorLiteModel_View_Confirmed(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m.quitting = true
	m.confirmed = true

	view := m.View()
	if !strings.Contains(view, "Selected 2 repositories for backup") {
		t.Errorf("confirmed view should contain 'Selected 2 repositories for backup', got: %q", view)
	}
}

func TestRepoSelectorLiteModel_View_Cancelled(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m.quitting = true
	m.confirmed = false

	view := m.View()
	if !strings.Contains(view, "Cancelled") {
		t.Errorf("cancelled view should contain 'Cancelled', got: %q", view)
	}
}

func TestRepoSelectorLiteModel_View_ShowsRepos(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	view := m.View()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(view, name) {
			t.Errorf("view should contain repo name %q", name)
		}
	}
}

// --- Key handling ---

func TestRepoSelectorLiteModel_Key_CursorDown(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())

	m = updateLite(t, m, press('j'))
	if m.cursor != 1 {
		t.Errorf("cursor after 'j' = %d, want 1", m.cursor)
	}

	m = updateLite(t, m, pressSpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("cursor after Down = %d, want 2", m.cursor)
	}
}

func TestRepoSelectorLiteModel_Key_CursorUp(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m.cursor = 2

	m = updateLite(t, m, press('k'))
	if m.cursor != 1 {
		t.Errorf("cursor after 'k' = %d, want 1", m.cursor)
	}

	m = updateLite(t, m, pressSpecial(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("cursor after Up = %d, want 0", m.cursor)
	}
}

func TestRepoSelectorLiteModel_Key_CursorBounds(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())

	// Should not go below 0
	m = updateLite(t, m, pressSpecial(tea.KeyUp))
	if m.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", m.cursor)
	}

	// Move to last item, then try to go further
	m.cursor = 2
	m = updateLite(t, m, pressSpecial(tea.KeyDown))
	if m.cursor != 2 {
		t.Errorf("cursor should stay at 2 (last), got %d", m.cursor)
	}
}

func TestRepoSelectorLiteModel_Key_ToggleSelection(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	// Cursor is at 0 (alpha), which starts selected

	m = updateLite(t, m, pressSpecial(tea.KeySpace))
	if m.selected[0] {
		t.Error("space should have deselected repo 0")
	}

	m = updateLite(t, m, pressSpecial(tea.KeySpace))
	if !m.selected[0] {
		t.Error("second space should have re-selected repo 0")
	}
}

func TestRepoSelectorLiteModel_Key_SelectAll(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m = updateLite(t, m, press('a'))

	for i := range m.repos {
		if !m.selected[i] {
			t.Errorf("'a' should have selected all repos; repo %d is unselected", i)
		}
	}
}

func TestRepoSelectorLiteModel_Key_SelectNone(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m = updateLite(t, m, press('n'))

	for i := range m.repos {
		if m.selected[i] {
			t.Errorf("'n' should have deselected all repos; repo %d is still selected", i)
		}
	}
}

func TestRepoSelectorLiteModel_Key_CycleMode(t *testing.T) {
	repos := []git.Repository{
		{Name: "x", Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorLiteModel(repos)

	m = updateLite(t, m, press('m'))
	if m.repos[0].Mode != git.ModePushKnownBranches {
		t.Errorf("after 'm': mode = %v, want ModePushKnownBranches", m.repos[0].Mode)
	}

	m = updateLite(t, m, press('m'))
	if m.repos[0].Mode != git.ModePushAll {
		t.Errorf("after second 'm': mode = %v, want ModePushAll", m.repos[0].Mode)
	}

	m = updateLite(t, m, press('m'))
	if m.repos[0].Mode != git.ModeLeaveUntouched {
		t.Errorf("after third 'm': mode = %v, want ModeLeaveUntouched (wraps around)", m.repos[0].Mode)
	}
}

func TestRepoSelectorLiteModel_Key_Quit(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m = updateLite(t, m, press('q'))

	if !m.quitting {
		t.Error("'q' should set quitting=true")
	}
	if m.confirmed {
		t.Error("'q' should not set confirmed=true")
	}
}

func TestRepoSelectorLiteModel_Key_Enter(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos())
	m = updateLite(t, m, pressSpecial(tea.KeyEnter))

	if !m.quitting {
		t.Error("enter should set quitting=true")
	}
	if !m.confirmed {
		t.Error("enter should set confirmed=true")
	}
}

func TestRepoSelectorLiteModel_Key_EmptyRepos_NoPanic(t *testing.T) {
	m := NewRepoSelectorLiteModel(nil)

	assertNoPanic := func(msg tea.Msg) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Update panicked for empty repos and msg %T: %v", msg, r)
			}
		}()
		_, _ = m.Update(msg)
	}

	assertNoPanic(press('m'))
	assertNoPanic(pressSpecial(tea.KeySpace))
	assertNoPanic(pressSpecial(tea.KeyUp))
	assertNoPanic(pressSpecial(tea.KeyDown))
}

// --- RepoSelectorModel (full, animated) ---
// The full model shares identical key-handling logic as the lite model.
// These tests confirm construction and GetSelectedRepos work correctly.

func TestNewRepoSelectorModel_DefaultSelection(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorModel(repos)

	if !m.selected[0] {
		t.Error("repo 0 (Selected=true) should be selected initially")
	}
	if m.selected[1] {
		t.Error("repo 1 (Selected=false) should not be selected initially")
	}
}

func TestRepoSelectorModel_GetSelectedRepos(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos())
	selected := m.GetSelectedRepos()

	if len(selected) != 2 {
		t.Errorf("GetSelectedRepos() returned %d repos, want 2", len(selected))
	}
	for _, r := range selected {
		if r.Name != "alpha" && r.Name != "gamma" {
			t.Errorf("unexpected repo in selection: %s", r.Name)
		}
	}
}

func TestRepoSelectorModel_View_Confirmed(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos())
	m.quitting = true
	m.confirmed = true

	view := m.View()
	if !strings.Contains(view, "Selected 2 repositories for backup") {
		t.Errorf("confirmed view should contain 'Selected 2 repositories for backup', got: %q", view)
	}
}

func TestRepoSelectorModel_View_Cancelled(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos())
	m.quitting = true
	m.confirmed = false

	view := m.View()
	if !strings.Contains(view, "Cancelled") {
		t.Errorf("cancelled view should contain 'Cancelled', got: %q", view)
	}
}

func TestRepoSelectorModel_Key_EmptyRepos_NoPanic(t *testing.T) {
	m := NewRepoSelectorModel(nil)

	assertNoPanic := func(msg tea.Msg) {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Update panicked for empty repos and msg %T: %v", msg, r)
			}
		}()
		_, _ = m.Update(msg)
	}

	assertNoPanic(press('m'))
	assertNoPanic(pressSpecial(tea.KeySpace))
	assertNoPanic(pressSpecial(tea.KeyUp))
	assertNoPanic(pressSpecial(tea.KeyDown))
}
