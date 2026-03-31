package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TBRX103/git-fire/internal/config"
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

// assertUpdateNoPanic runs m.Update(msg) and fails the test if it panics.
func assertUpdateNoPanic(t *testing.T, m tea.Model, msg tea.Msg) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Update panicked for empty repos and msg %T: %v", msg, r)
		}
	}()
	_, _ = m.Update(msg)
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
	root := filepath.Join(os.TempDir(), "gitfire-ui-sample")
	return []git.Repository{
		{Path: filepath.Join(root, "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
		{Path: filepath.Join(root, "beta"), Name: "beta", Selected: false, Mode: git.ModePushKnownBranches},
		{Path: filepath.Join(root, "gamma"), Name: "gamma", Selected: true, Mode: git.ModePushAll},
	}
}

// --- RepoSelectorLiteModel tests ---

func TestNewRepoSelectorLiteModel_DefaultSelection(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorLiteModel(repos, nil, "")

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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
}

func TestRepoSelectorLiteModel_Init(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil for the lite model")
	}
}

func TestRepoSelectorLiteModel_GetSelectedRepos(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	selected := m.GetSelectedRepos()

	if len(selected) != 2 {
		t.Fatalf("GetSelectedRepos() returned %d repos, want 2", len(selected))
	}
	names := make(map[string]struct{})
	for _, r := range selected {
		names[r.Name] = struct{}{}
	}
	if len(names) != 2 {
		t.Errorf("expected 2 distinct repo names in selection, got %d distinct: %v", len(names), names)
	}
	if _, ok := names["alpha"]; !ok {
		t.Error(`GetSelectedRepos() missing expected repo "alpha"`)
	}
	if _, ok := names["gamma"]; !ok {
		t.Error(`GetSelectedRepos() missing expected repo "gamma"`)
	}
}

func TestRepoSelectorLiteModel_GetSelectedRepos_NoneSelected(t *testing.T) {
	repos := []git.Repository{
		{Name: "a", Selected: false},
		{Name: "b", Selected: false},
	}
	m := NewRepoSelectorLiteModel(repos, nil, "")
	if len(m.GetSelectedRepos()) != 0 {
		t.Error("expected no selected repos")
	}
}

// --- View output ---

func TestRepoSelectorLiteModel_View_Confirmed(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m.quitting = true
	m.confirmed = true

	view := m.View()
	if !strings.Contains(view, "Selected 2 repositories for backup") {
		t.Errorf("confirmed view should contain 'Selected 2 repositories for backup', got: %q", view)
	}
}

func TestRepoSelectorLiteModel_View_Cancelled(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m.quitting = true
	m.confirmed = false

	view := m.View()
	if !strings.Contains(view, "Cancelled") {
		t.Errorf("cancelled view should contain 'Cancelled', got: %q", view)
	}
}

func TestRepoSelectorLiteModel_View_ShowsRepos(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorLiteModel(repos, nil, "")
	view := m.View()

	for _, r := range repos {
		// View now shows repo name and parent directory separately.
		if !strings.Contains(view, r.Name) {
			t.Errorf("view should contain repo name %q", r.Name)
		}
		wantParent := AbbreviateUserHome(filepath.Dir(r.Path))
		if !strings.Contains(view, wantParent) {
			t.Errorf("view should contain parent path %q", wantParent)
		}
	}
}

// --- Key handling ---

func TestRepoSelectorLiteModel_Key_CursorDown(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")

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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")

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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m = updateLite(t, m, press('a'))

	for i := range m.repos {
		if !m.selected[i] {
			t.Errorf("'a' should have selected all repos; repo %d is unselected", i)
		}
	}
}

func TestRepoSelectorLiteModel_Key_SelectNone(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
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
	m := NewRepoSelectorLiteModel(repos, nil, "")

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
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m = updateLite(t, m, press('q'))

	if !m.quitting {
		t.Error("'q' should set quitting=true")
	}
	if m.confirmed {
		t.Error("'q' should not set confirmed=true")
	}
}

func TestRepoSelectorLiteModel_Key_Enter(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m = updateLite(t, m, pressSpecial(tea.KeyEnter))

	if !m.quitting {
		t.Error("enter should set quitting=true")
	}
	if !m.confirmed {
		t.Error("enter should set confirmed=true")
	}
}

func TestRepoSelectorLiteModel_Key_EmptyRepos_NoPanic(t *testing.T) {
	m := NewRepoSelectorLiteModel(nil, nil, "")

	assertUpdateNoPanic(t, m, press('m'))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeySpace))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeyUp))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeyDown))
}

// --- RepoSelectorModel (full, animated) ---
// The full model shares identical key-handling logic as the lite model.
// These tests confirm construction and GetSelectedRepos work correctly.

func TestNewRepoSelectorModel_DefaultSelection(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorModel(repos, nil, "")

	if !m.selected[0] {
		t.Error("repo 0 (Selected=true) should be selected initially")
	}
	if m.selected[1] {
		t.Error("repo 1 (Selected=false) should not be selected initially")
	}
}

func TestRepoSelectorModel_GetSelectedRepos(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
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
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.quitting = true
	m.confirmed = true

	view := m.View()
	if !strings.Contains(view, "Selected 2 repositories for backup") {
		t.Errorf("confirmed view should contain 'Selected 2 repositories for backup', got: %q", view)
	}
}

func TestRepoSelectorModel_View_Cancelled(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.quitting = true
	m.confirmed = false

	view := m.View()
	if !strings.Contains(view, "Cancelled") {
		t.Errorf("cancelled view should contain 'Cancelled', got: %q", view)
	}
}

func TestRepoSelectorModel_View_ShowsRepos(t *testing.T) {
	repos := sampleRepos()
	m := NewRepoSelectorModel(repos, nil, "")
	view := m.View()

	for _, r := range repos {
		// View now shows repo name and parent directory separately.
		if !strings.Contains(view, r.Name) {
			t.Errorf("view should contain repo name %q", r.Name)
		}
		wantParent := AbbreviateUserHome(filepath.Dir(r.Path))
		if !strings.Contains(view, wantParent) {
			t.Errorf("view should contain parent path %q", wantParent)
		}
	}
}

func TestRepoSelectorModel_Key_EmptyRepos_NoPanic(t *testing.T) {
	m := NewRepoSelectorModel(nil, nil, "")

	assertUpdateNoPanic(t, m, press('m'))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeySpace))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeyUp))
	assertUpdateNoPanic(t, m, pressSpecial(tea.KeyDown))
}

func TestRepoSelectorLiteModel_View_ShowsScrollHintWhenPathTruncated(t *testing.T) {
	longParent := filepath.Join(
		os.TempDir(),
		"gitfire-ui-sample",
		"very",
		"long",
		"parent",
		"path",
		"that",
		"will",
		"truncate",
	)
	repos := []git.Repository{
		{Path: filepath.Join(longParent, "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorLiteModel(repos, nil, "")
	m.windowWidth = 45

	view := m.View()
	if !strings.Contains(view, "SCROLL PATH") {
		t.Fatalf("expected lite view to show scroll hint when truncated, got: %q", view)
	}
}

func TestRepoSelectorModel_View_ShowsScrollHintWhenPathTruncated(t *testing.T) {
	longParent := filepath.Join(
		os.TempDir(),
		"gitfire-ui-sample",
		"very",
		"long",
		"parent",
		"path",
		"that",
		"will",
		"truncate",
	)
	repos := []git.Repository{
		{Path: filepath.Join(longParent, "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorModel(repos, nil, "")
	m.windowWidth = 45

	view := m.View()
	if !strings.Contains(view, "SCROLL PATH") {
		t.Fatalf("expected full view to show scroll hint when truncated, got: %q", view)
	}
}

func TestRepoSelectorModel_View_SmallHeightStillShowsAtLeastOneRepoRow(t *testing.T) {
	repos := []git.Repository{
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "beta"), Name: "beta", Selected: true, Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorModel(repos, nil, "")
	m.windowWidth = 80
	m.windowHeight = 26 // small terminal — exercises the at-least-one-row floor

	view := m.View()
	if !strings.Contains(view, "alpha") && !strings.Contains(view, "beta") {
		t.Fatalf("expected at least one repo row to render at small height, got: %q", view)
	}
}

func TestRepoSelectorModel_View_ShowsViewportWarningWhenIndicatorsSuppressed(t *testing.T) {
	repos := []git.Repository{
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "beta"), Name: "beta", Selected: true, Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorModel(repos, nil, "")
	m.showFire = false
	m.windowWidth = 80
	m.windowHeight = 12 // intentionally tiny to force 1 list row and suppress ↑/↓ lines

	view := m.View()
	if !strings.Contains(view, "More repos exist, but ↑/↓ indicators are hidden") {
		t.Fatalf("expected compact-height warning when indicators are suppressed, got: %q", view)
	}
}

// --- Fire animation toggle tests ---

// updateMain sends a key to the model and returns the updated RepoSelectorModel.
func updateMain(t *testing.T, m RepoSelectorModel, msg tea.Msg) RepoSelectorModel {
	t.Helper()
	updated, _ := m.Update(msg)
	typed, ok := updated.(RepoSelectorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorModel", updated)
	}
	return typed
}

func TestRepoSelectorModel_DefaultShowFire(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	if !m.showFire {
		t.Error("showFire should be true by default")
	}
}

func TestRepoSelectorModel_ShowFireFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowFireAnimation = false

	m := NewRepoSelectorModelStream(nil, nil, true, false, &cfg, "", nil, "")
	if m.showFire {
		t.Error("showFire should be false when cfg.UI.ShowFireAnimation = false")
	}
}

func TestRepoSelectorModel_FKeyTogglesShowFire(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.windowHeight = 40 // large enough that auto-suppress doesn't interfere

	if !m.showFire {
		t.Fatal("precondition: showFire should start true")
	}

	m = updateMain(t, m, press('f'))
	if m.showFire {
		t.Error("after first 'f', showFire should be false")
	}

	m = updateMain(t, m, press('f'))
	if !m.showFire {
		t.Error("after second 'f', showFire should be true again")
	}
}

func TestRepoSelectorModel_FKeyNoOpInIgnoredView(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.view = repoViewIgnored
	m.showFire = true

	m = updateMain(t, m, press('f'))
	if !m.showFire {
		t.Error("'f' in ignored view should not toggle showFire")
	}
}

func TestRepoSelectorModel_FKeyPersistsToConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowFireAnimation = true

	m := NewRepoSelectorModelStream(nil, nil, true, false, &cfg, "", nil, "")
	m.windowHeight = 40

	m = updateMain(t, m, press('f'))

	if m.cfg.UI.ShowFireAnimation {
		t.Error("cfg.UI.ShowFireAnimation should be false after toggling off")
	}
	if m.showFire {
		t.Error("showFire should be false after toggling off")
	}
}

func TestRepoSelectorModel_ViewShowsFireWhenEnabled(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showFire = true
	m.windowWidth = 80
	m.windowHeight = 40 // above threshold

	if !m.fireVisible() {
		t.Fatal("precondition: fireVisible() must be true for this test")
	}

	// With fire enabled the rendered output is taller; measure both and confirm
	// fire-on renders more lines than fire-off.
	viewFireOn := m.View()

	m.showFire = false
	viewFireOff := m.View()

	heightOn := strings.Count(viewFireOn, "\n")
	heightOff := strings.Count(viewFireOff, "\n")
	if heightOn <= heightOff {
		t.Errorf("fire-on view (%d lines) should be taller than fire-off view (%d lines)", heightOn, heightOff)
	}
}

func TestRepoSelectorModel_ViewHidesFireWhenDisabled(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showFire = false
	m.windowWidth = 80
	m.windowHeight = 40

	view := m.View()
	// List and controls must still be present
	if !strings.Contains(view, "GIT FIRE") {
		t.Error("title must still appear when fire is hidden")
	}
	if !strings.Contains(view, "Toggle fire") {
		t.Error("help text must still show 'Toggle fire' hint")
	}
}

func TestRepoSelectorModel_ViewSuppressesFireOnSmallTerminal(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showFire = true // user preference is on, but terminal is too short
	m.windowWidth = 80
	m.windowHeight = fireHeightThreshold - 1 // below threshold

	if m.fireVisible() {
		t.Error("fireVisible() should return false when windowHeight <= fireHeightThreshold")
	}

	view := m.View()
	// List must still be present
	if !strings.Contains(view, "GIT FIRE") {
		t.Error("title must appear even when fire is suppressed due to small terminal")
	}
}

func TestRepoSelectorModel_FireVisibleThreshold(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showFire = true

	m.windowHeight = fireHeightThreshold
	if m.fireVisible() {
		t.Errorf("fireVisible() should be false at exactly windowHeight=%d (threshold)", fireHeightThreshold)
	}

	m.windowHeight = fireHeightThreshold + 1
	if !m.fireVisible() {
		t.Errorf("fireVisible() should be true at windowHeight=%d (threshold+1)", fireHeightThreshold+1)
	}
}

func TestRepoSelectorModel_ShowFireAnimationConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowFireAnimation = true

	// Row 4 is "Show fire animation"
	val := configRowValue(4, &cfg)
	if val != "true" {
		t.Errorf("configRowValue(4) = %q, want %q", val, "true")
	}

	applyConfigChange(4, &cfg, 0) // direction ignored for bools
	if cfg.UI.ShowFireAnimation {
		t.Error("applyConfigChange(4) should have toggled ShowFireAnimation to false")
	}

	val = configRowValue(4, &cfg)
	if val != "false" {
		t.Errorf("configRowValue(4) after toggle = %q, want %q", val, "false")
	}
}

func TestRepoSelectorModel_ColorProfileConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ColorProfile = config.UIColorProfileClassic

	// Row 5 is "Color profile"
	val := configRowValue(5, &cfg)
	if val != config.UIColorProfileClassic {
		t.Errorf("configRowValue(5) = %q, want %q", val, config.UIColorProfileClassic)
	}

	applyConfigChange(5, &cfg, +1)
	if cfg.UI.ColorProfile == config.UIColorProfileClassic {
		t.Error("applyConfigChange(5,+1) should move to next color profile")
	}
}
