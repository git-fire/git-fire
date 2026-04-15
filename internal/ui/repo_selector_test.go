package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-harness/git"
)

// press builds a key message for a printable character (e.g. 'j', 'q', ' ').
func press(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// pressSpecial builds a key message for a named key (Enter, Up, Down, Space, etc.).
func pressSpecial(t tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: t}
}

func wheelVertical(btn tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: btn,
	}
}

func wheelHorizontal(btn tea.MouseButton) tea.MouseMsg {
	return tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: btn,
	}
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
	if m.repos[0].Mode != git.ModePushCurrentBranch {
		t.Errorf("after third 'm': mode = %v, want ModePushCurrentBranch", m.repos[0].Mode)
	}

	m = updateLite(t, m, press('m'))
	if m.repos[0].Mode != git.ModeLeaveUntouched {
		t.Errorf("after fourth 'm': mode = %v, want ModeLeaveUntouched (wraps around)", m.repos[0].Mode)
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

func TestRepoSelectorModel_MouseWheel_NavigateMainList(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	m = updateMain(t, m, wheelVertical(tea.MouseButtonWheelDown))
	if m.cursor != 1 {
		t.Errorf("after wheel down cursor = %d, want 1", m.cursor)
	}
	m = updateMain(t, m, wheelVertical(tea.MouseButtonWheelUp))
	if m.cursor != 0 {
		t.Errorf("after wheel up cursor = %d, want 0", m.cursor)
	}
}

func TestRepoSelectorModel_MouseWheel_ConfigViewMovesCursor(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.view = repoViewConfig
	m.configCursor = 0
	m = updateMain(t, m, wheelVertical(tea.MouseButtonWheelDown))
	if m.configCursor != 1 {
		t.Errorf("config cursor = %d, want 1", m.configCursor)
	}
	m = updateMain(t, m, wheelVertical(tea.MouseButtonWheelUp))
	if m.configCursor != 0 {
		t.Errorf("config cursor = %d, want 0", m.configCursor)
	}
}

func manySampleRepos(n int) []git.Repository {
	root := filepath.Join(os.TempDir(), "gitfire-ui-many")
	out := make([]git.Repository, 0, n)
	for i := range n {
		name := fmt.Sprintf("repo%d", i)
		out = append(out, git.Repository{
			Path:     filepath.Join(root, name),
			Name:     name,
			Selected: true,
			Mode:     git.ModeLeaveUntouched,
		})
	}
	return out
}

func TestRepoSelectorModel_PageKeys_HomeEnd(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m = updateMain(t, m, pressSpecial(tea.KeyEnd))
	if m.cursor != 2 {
		t.Fatalf("after End cursor = %d, want 2", m.cursor)
	}
	m = updateMain(t, m, pressSpecial(tea.KeyHome))
	if m.cursor != 0 {
		t.Fatalf("after Home cursor = %d, want 0", m.cursor)
	}
}

func TestRepoSelectorModel_PageKeys_PgUpPgDown(t *testing.T) {
	repos := manySampleRepos(30)
	m := NewRepoSelectorModel(repos, nil, "")
	m.windowWidth = 80
	m.windowHeight = 40
	m.showFire = false // stable list viewport for predictable page step

	m = updateMain(t, m, pressSpecial(tea.KeyPgDown))
	if m.cursor <= 0 {
		t.Fatalf("expected PgDown to advance cursor, got %d", m.cursor)
	}
	firstJump := m.cursor
	m = updateMain(t, m, pressSpecial(tea.KeyPgUp))
	if m.cursor != 0 {
		t.Fatalf("after PgUp from first page want cursor 0, got %d", m.cursor)
	}
	// Second PgDown should land at same index as first (deterministic viewport).
	m = updateMain(t, m, pressSpecial(tea.KeyPgDown))
	if m.cursor != firstJump {
		t.Fatalf("second PgDown cursor = %d, want %d", m.cursor, firstJump)
	}
}

func TestRepoSelectorModel_ConfigView_PageKeys(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.view = repoViewConfig
	cfg := config.DefaultConfig()
	m.cfg = &cfg
	m.configCursor = 0

	m = updateMain(t, m, pressSpecial(tea.KeyEnd))
	wantLast := len(configRows) - 1
	if m.configCursor != wantLast {
		t.Fatalf("End: configCursor = %d, want %d", m.configCursor, wantLast)
	}
	m = updateMain(t, m, pressSpecial(tea.KeyHome))
	if m.configCursor != 0 {
		t.Fatalf("Home: configCursor = %d, want 0", m.configCursor)
	}
	step := configViewPageStep()
	m = updateMain(t, m, pressSpecial(tea.KeyPgDown))
	if got := m.configCursor; got != step && got != wantLast {
		t.Fatalf("PgDown from 0: configCursor = %d, want %d or %d", got, step, wantLast)
	}
}

func TestRepoSelectorLiteModel_PageKeys(t *testing.T) {
	repos := manySampleRepos(25)
	m := NewRepoSelectorLiteModel(repos, nil, "")
	m = updateLite(t, m, pressSpecial(tea.KeyEnd))
	if m.cursor != len(repos)-1 {
		t.Fatalf("End: cursor = %d", m.cursor)
	}
	m = updateLite(t, m, pressSpecial(tea.KeyHome))
	if m.cursor != 0 {
		t.Fatalf("Home: cursor = %d", m.cursor)
	}
	m = updateLite(t, m, pressSpecial(tea.KeyPgDown))
	if m.cursor <= 0 {
		t.Fatalf("PgDown should advance, cursor = %d", m.cursor)
	}
}

func TestRepoSelectorModel_MouseWheel_HorizontalScrollsPath(t *testing.T) {
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
	m = updateMain(t, m, wheelHorizontal(tea.MouseButtonWheelRight))
	if m.pathScrollOffset <= 0 {
		t.Fatalf("expected pathScrollOffset > 0 after wheel right, got %d", m.pathScrollOffset)
	}
	off := m.pathScrollOffset
	m = updateMain(t, m, wheelHorizontal(tea.MouseButtonWheelLeft))
	if m.pathScrollOffset >= off {
		t.Fatalf("expected pathScrollOffset to decrease after wheel left, had %d then %d", off, m.pathScrollOffset)
	}
}

func TestRepoSelectorLiteModel_MouseWheel_NavigateList(t *testing.T) {
	m := NewRepoSelectorLiteModel(sampleRepos(), nil, "")
	m = updateLite(t, m, wheelVertical(tea.MouseButtonWheelDown))
	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
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
	// Hint may wrap across lines at narrow widths; match parts that stay stable.
	if !strings.Contains(view, "SCROLL") || !strings.Contains(view, "PATH >>") {
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

func TestRepoSelectorModel_View_HandlesSuppressedIndicatorsInTinyViewport(t *testing.T) {
	repos := []git.Repository{
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "alpha"), Name: "alpha", Selected: true, Mode: git.ModeLeaveUntouched},
		{Path: filepath.Join(os.TempDir(), "gitfire-ui-sample", "beta"), Name: "beta", Selected: true, Mode: git.ModeLeaveUntouched},
	}
	m := NewRepoSelectorModel(repos, nil, "")
	m.showFire = false
	m.windowWidth = 80
	m.windowHeight = 12 // intentionally tiny to force suppressed indicators and warning fallback behavior

	view := m.View()
	if strings.Contains(view, "↑ 1 more") || strings.Contains(view, "↓ 1 more") {
		t.Fatalf("did not expect explicit scroll indicators in tiny viewport, got: %q", view)
	}
	if !strings.Contains(view, "alpha") && !strings.Contains(view, "beta") {
		t.Fatalf("expected at least one repo row to still render, got: %q", view)
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

func TestRepoSelectorModel_FireTickFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.FireTickMS = 150

	m := NewRepoSelectorModelStream(nil, nil, true, false, &cfg, "", nil, "")
	if got, want := m.fireTick.Milliseconds(), int64(150); got != want {
		t.Errorf("fireTick = %dms, want %dms", got, want)
	}
}

func TestRepoSelectorModel_StartupQuoteConfigFromStreamModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowStartupQuote = true
	cfg.UI.StartupQuoteBehavior = config.UIQuoteBehaviorHide
	cfg.UI.StartupQuoteIntervalSec = 10

	m := NewRepoSelectorModelStream(nil, nil, true, false, &cfg, "", nil, "")
	if !m.showStartupQuote {
		t.Fatal("showStartupQuote should be true from config")
	}
	if m.startupQuoteBehavior != config.UIQuoteBehaviorHide {
		t.Fatalf("startupQuoteBehavior = %q, want %q", m.startupQuoteBehavior, config.UIQuoteBehaviorHide)
	}
	if got := int(m.startupQuoteInterval.Seconds()); got != 10 {
		t.Fatalf("startupQuoteInterval = %ds, want 10s", got)
	}
	if !m.startupQuoteVisible {
		t.Fatal("startupQuoteVisible should start true when quotes enabled")
	}
}

func TestRepoSelectorModel_QuoteTick_HideBehavior(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = true
	m.startupQuoteBehavior = config.UIQuoteBehaviorHide
	m.startupQuoteInterval = 10 * time.Second
	m.currentStartupQuote = "Fire walk with me."
	m.startupQuoteVisible = true

	updated, cmd := m.Update(quoteTickMsg(time.Now()))
	got, ok := updated.(RepoSelectorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorModel", updated)
	}
	if got.startupQuoteVisible {
		t.Fatal("quote should be hidden after tick when behavior=hide")
	}
	if got.quoteTickActive {
		t.Fatal("quote tick should stop after hide behavior hides the quote")
	}
	if cmd != nil {
		t.Fatal("hide behavior should not schedule another quote tick")
	}
}

func TestRepoSelectorModel_QuoteTick_HideDeferredWhileScanStreaming(t *testing.T) {
	scanCh := make(chan git.Repository)
	progCh := make(chan string)
	cfg := config.DefaultConfig()
	cfg.UI.StartupQuoteBehavior = config.UIQuoteBehaviorHide
	cfg.UI.StartupQuoteIntervalSec = 10

	m := NewRepoSelectorModelStream(scanCh, progCh, false, false, &cfg, "", nil, "")
	if m.scanDone {
		t.Fatal("sanity: scan should not be done at stream start")
	}
	m.currentStartupQuote = "Still scanning."
	m.startupQuoteVisible = true

	updated, cmd := m.Update(quoteTickMsg(time.Now()))
	got, ok := updated.(RepoSelectorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorModel", updated)
	}
	if !got.startupQuoteVisible {
		t.Fatal("quote should stay visible while scan is streaming and hide tick fires")
	}
	if !got.quoteTickActive {
		t.Fatal("quote tick should reschedule until scan finishes")
	}
	if cmd == nil {
		t.Fatal("expected a follow-up quote tick cmd while scan is in progress")
	}

	got.scanDone = true
	updated2, cmd2 := got.Update(quoteTickMsg(time.Now()))
	after := updated2.(RepoSelectorModel)
	if after.startupQuoteVisible {
		t.Fatal("quote should hide on hide tick after scan completes")
	}
	if cmd2 != nil {
		t.Fatal("hide should not schedule another tick after hiding")
	}
}

func TestRepoSelectorModel_QuoteTick_NoOpWhenQuotesDisabled(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = false
	m.startupQuoteBehavior = config.UIQuoteBehaviorRefresh
	m.startupQuoteInterval = 10 * time.Second
	m.currentStartupQuote = "unchanged"
	m.startupQuoteVisible = true

	updated, _ := m.Update(quoteTickMsg(time.Now()))
	got, ok := updated.(RepoSelectorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorModel", updated)
	}
	if got.currentStartupQuote != "unchanged" {
		t.Fatalf("quote changed while disabled: got %q", got.currentStartupQuote)
	}
	if !got.startupQuoteVisible {
		t.Fatal("quote visibility should remain unchanged when quotes are disabled")
	}
}

func TestRepoSelectorModel_QuoteTick_RefreshBehavior(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = true
	m.startupQuoteBehavior = config.UIQuoteBehaviorRefresh
	m.startupQuoteInterval = 10 * time.Second
	m.currentStartupQuote = ""
	m.startupQuoteVisible = false

	updated, _ := m.Update(quoteTickMsg(time.Now()))
	got, ok := updated.(RepoSelectorModel)
	if !ok {
		t.Fatalf("Update() returned %T, want RepoSelectorModel", updated)
	}
	if !got.startupQuoteVisible {
		t.Fatal("quote should be visible after tick when behavior=refresh")
	}
	if got.currentStartupQuote == "" {
		t.Fatal("quote should refresh to a non-empty value")
	}
}

func TestRepoSelectorModel_SyncRuntimeFromConfig_DoesNotDuplicateQuoteTickOrReshow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowStartupQuote = true
	cfg.UI.StartupQuoteIntervalSec = 10

	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.cfg = &cfg
	m.showStartupQuote = true
	m.startupQuoteVisible = false
	m.quoteTickActive = true

	updated, cmds := m.syncRuntimeFromConfig(nil)

	if updated.startupQuoteVisible {
		t.Fatal("syncRuntimeFromConfig should not force hidden quote visible on unrelated config changes")
	}
	if len(cmds) != 0 {
		t.Fatalf("syncRuntimeFromConfig should not enqueue duplicate quote ticks when one is already active; got %d cmds", len(cmds))
	}
	if !updated.quoteTickActive {
		t.Fatal("quote tick should remain active when already running")
	}
}

func TestRepoSelectorModel_SyncRuntimeFromConfig_ToggleOnReshowsAndSchedulesTick(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowStartupQuote = true
	cfg.UI.StartupQuoteIntervalSec = 10

	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.cfg = &cfg
	m.showStartupQuote = false
	m.startupQuoteVisible = false
	m.quoteTickActive = false

	updated, cmds := m.syncRuntimeFromConfig(nil)

	if !updated.startupQuoteVisible {
		t.Fatal("syncRuntimeFromConfig should show quote when startup quote is toggled on")
	}
	if len(cmds) != 1 {
		t.Fatalf("syncRuntimeFromConfig should enqueue one quote tick when enabling startup quote; got %d cmds", len(cmds))
	}
	if !updated.quoteTickActive {
		t.Fatal("quote tick should be marked active after scheduling")
	}
}

func TestRepoSelectorModel_QuoteVisibleHelper(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = true
	m.startupQuoteVisible = true
	m.currentStartupQuote = "A light in the dark provides hope."
	if !m.quoteVisible() {
		t.Fatal("quoteVisible should be true when all quote flags are set")
	}

	m.currentStartupQuote = ""
	if m.quoteVisible() {
		t.Fatal("quoteVisible should be false with empty quote text")
	}
}

func TestRepoSelectorModel_View_HidesQuoteBannerWhenNotVisible(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = true
	m.startupQuoteVisible = false
	m.currentStartupQuote = "A light in the dark provides hope."

	view := m.View()
	if strings.Contains(view, `🔥 "A light in the dark provides hope."`) {
		t.Fatalf("did not expect startup quote banner in view, got: %q", view)
	}
}

func TestRepoSelectorModel_View_ShowsQuoteBannerWhenVisible(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.showStartupQuote = true
	m.startupQuoteVisible = true
	m.currentStartupQuote = "A light in the dark provides hope."

	view := m.View()
	if !strings.Contains(view, `🔥 "A light in the dark provides hope."`) {
		t.Fatalf("expected startup quote banner in view, got: %q", view)
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

func TestRepoSelectorModel_IgnoredListVisibleCount_FireOverhead(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.windowWidth = 80
	m.windowHeight = 25
	m.fireBg = NewFireBackground(70, 5)

	m.showFire = true
	visibleFireOn := m.ignoredListVisibleCount()
	reserve := m.fireSectionReserveLines()
	if reserve != 5+2 {
		t.Fatalf("fireSectionReserveLines() = %d, want 7 for height 5 + 2", reserve)
	}

	m.showFire = false
	visibleFireOff := m.ignoredListVisibleCount()
	if d := visibleFireOff - visibleFireOn; d != reserve {
		t.Errorf("visible delta fire off−on = %d, want fireSectionReserveLines()=%d", d, reserve)
	}

	m.showFire = true
	m.windowHeight = fireHeightThreshold // fire suppressed by short terminal
	visibleShortWithFireFlag := m.ignoredListVisibleCount()
	m.showFire = false
	visibleShortFireOff := m.ignoredListVisibleCount()
	if visibleShortWithFireFlag != visibleShortFireOff {
		t.Errorf("short terminal: fire on vs off visible = %d vs %d, want equal (fire not shown)", visibleShortWithFireFlag, visibleShortFireOff)
	}
}

func TestRepoSelectorModel_IgnoredListVisibleCount_NarrowWidthMoreChrome(t *testing.T) {
	wide := NewRepoSelectorModel(sampleRepos(), nil, "")
	wide.windowWidth = 120
	wide.windowHeight = 40
	wide.fireBg = NewFireBackground(70, 5)
	wide.showFire = false

	narrow := NewRepoSelectorModel(sampleRepos(), nil, "")
	narrow.windowWidth = 32
	narrow.windowHeight = 40
	narrow.fireBg = NewFireBackground(70, 5)
	narrow.showFire = false

	if g, w := narrow.ignoredListVisibleCount(), wide.ignoredListVisibleCount(); g > w {
		t.Errorf("narrow width should not reserve fewer list rows than wide: narrow=%d wide=%d", g, w)
	}
	if narrow.ignoredViewNonListHeight() <= wide.ignoredViewNonListHeight() {
		t.Errorf("ignored chrome height narrow=%d should exceed wide=%d when help wraps",
			narrow.ignoredViewNonListHeight(), wide.ignoredViewNonListHeight())
	}
}

func TestRepoSelectorModel_QuoteWrappingAffectsMeasuredHeights(t *testing.T) {
	m := NewRepoSelectorModel(sampleRepos(), nil, "")
	m.windowWidth = 40
	m.windowHeight = 40
	m.showFire = false
	m.showStartupQuote = true
	m.startupQuoteVisible = true

	m.currentStartupQuote = "short quote"
	shortIgnored := m.ignoredViewNonListHeight()
	shortVisible := m.repoListVisibleCount()

	m.currentStartupQuote = strings.Repeat("From ember to branch, every change deserves shelter. ", 3)
	longIgnored := m.ignoredViewNonListHeight()
	longVisible := m.repoListVisibleCount()

	if longIgnored <= shortIgnored {
		t.Fatalf("expected long wrapped quote to increase ignored view non-list height: long=%d short=%d", longIgnored, shortIgnored)
	}
	if longVisible >= shortVisible {
		t.Fatalf("expected long wrapped quote to reduce visible repo rows: long=%d short=%d", longVisible, shortVisible)
	}
}

func TestRepoSelectorModel_ShowFireAnimationConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowFireAnimation = true

	// Row 5 is "Show fire animation"
	val := configRowValue(5, &cfg)
	if val != "true" {
		t.Errorf("configRowValue(5) = %q, want %q", val, "true")
	}

	applyConfigChange(5, &cfg, 0) // direction ignored for bools
	if cfg.UI.ShowFireAnimation {
		t.Error("applyConfigChange(5) should have toggled ShowFireAnimation to false")
	}

	val = configRowValue(5, &cfg)
	if val != "false" {
		t.Errorf("configRowValue(5) after toggle = %q, want %q", val, "false")
	}
}

func TestRepoSelectorModel_ShowStartupQuoteConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ShowStartupQuote = true

	// Row 6 is "Show flavor quotes" (ui.show_startup_quote)
	val := configRowValue(6, &cfg)
	if val != "true" {
		t.Errorf("configRowValue(6) = %q, want %q", val, "true")
	}

	applyConfigChange(6, &cfg, 0) // direction ignored for bools
	if cfg.UI.ShowStartupQuote {
		t.Error("applyConfigChange(6) should have toggled ShowStartupQuote to false")
	}
}

func TestRepoSelectorModel_StartupQuoteBehaviorConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.StartupQuoteBehavior = config.UIQuoteBehaviorRefresh

	// Row 7 is "Flavor quote behavior"
	val := configRowValue(7, &cfg)
	if val != config.UIQuoteBehaviorRefresh {
		t.Errorf("configRowValue(7) = %q, want %q", val, config.UIQuoteBehaviorRefresh)
	}

	applyConfigChange(7, &cfg, +1)
	if cfg.UI.StartupQuoteBehavior != config.UIQuoteBehaviorHide {
		t.Errorf("applyConfigChange(7,+1) = %q, want %q", cfg.UI.StartupQuoteBehavior, config.UIQuoteBehaviorHide)
	}
}

func TestRepoSelectorModel_StartupQuoteIntervalConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.StartupQuoteIntervalSec = 10

	// Row 8 is "Flavor quote interval (s)"
	val := configRowValue(8, &cfg)
	if val != "10" {
		t.Errorf("configRowValue(8) = %q, want %q", val, "10")
	}

	applyConfigChange(8, &cfg, +1)
	if cfg.UI.StartupQuoteIntervalSec != 15 {
		t.Errorf("applyConfigChange(8,+1) = %d, want %d", cfg.UI.StartupQuoteIntervalSec, 15)
	}

	applyConfigChange(8, &cfg, -1)
	if cfg.UI.StartupQuoteIntervalSec != 10 {
		t.Errorf("applyConfigChange(8,-1) = %d, want %d", cfg.UI.StartupQuoteIntervalSec, 10)
	}
}

func TestRepoSelectorModel_FireSpeedConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.FireTickMS = 180

	// Row 9 is "Fire speed (ms)"
	val := configRowValue(9, &cfg)
	if val != "180" {
		t.Errorf("configRowValue(9) = %q, want %q", val, "180")
	}

	applyConfigChange(9, &cfg, +1)
	if cfg.UI.FireTickMS != 220 {
		t.Errorf("applyConfigChange(9,+1) = %d, want %d", cfg.UI.FireTickMS, 220)
	}

	applyConfigChange(9, &cfg, -1)
	if cfg.UI.FireTickMS != 180 {
		t.Errorf("applyConfigChange(9,-1) = %d, want %d", cfg.UI.FireTickMS, 180)
	}
}

func TestRepoSelectorModel_FireSpeedConfigRow_FromCustomOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.FireTickMS = 175 // manual override not present in presets

	applyConfigChange(9, &cfg, +1)
	if cfg.UI.FireTickMS != 180 {
		t.Errorf("applyConfigChange(9) custom +1 = %d, want %d", cfg.UI.FireTickMS, 180)
	}

	cfg.UI.FireTickMS = 175
	applyConfigChange(9, &cfg, -1)
	if cfg.UI.FireTickMS != 150 {
		t.Errorf("applyConfigChange(9) custom -1 = %d, want %d", cfg.UI.FireTickMS, 150)
	}
}

func TestRepoSelectorModel_PushWorkersConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.PushWorkers = 4

	// Row 4 is "Push workers".
	val := configRowValue(4, &cfg)
	if val != "4" {
		t.Errorf("configRowValue(4) = %q, want %q", val, "4")
	}

	applyConfigChange(4, &cfg, +1)
	if cfg.Global.PushWorkers != 8 {
		t.Errorf("applyConfigChange(4,+1) = %d, want %d", cfg.Global.PushWorkers, 8)
	}

	applyConfigChange(4, &cfg, -1)
	if cfg.Global.PushWorkers != 4 {
		t.Errorf("applyConfigChange(4,-1) = %d, want %d", cfg.Global.PushWorkers, 4)
	}
}

func TestRepoSelectorModel_ColorProfileConfigRow(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.UI.ColorProfile = config.UIColorProfileClassic

	// Row 10 is "Color profile"
	val := configRowValue(10, &cfg)
	if val != config.UIColorProfileClassic {
		t.Errorf("configRowValue(10) = %q, want %q", val, config.UIColorProfileClassic)
	}

	applyConfigChange(10, &cfg, +1)
	if cfg.UI.ColorProfile == config.UIColorProfileClassic {
		t.Error("applyConfigChange(10,+1) should move to next color profile")
	}
}

func TestRepoSelectorModel_CustomPaletteRowComingSoon(t *testing.T) {
	cfg := config.DefaultConfig()
	beforeProfile := cfg.UI.ColorProfile

	// Row 11 is "Custom hex palette" and should be non-editable for now.
	val := configRowValue(11, &cfg)
	if val == "" {
		t.Fatal("configRowValue(11) should show a placeholder/preview string")
	}

	applyConfigChange(11, &cfg, +1)
	if cfg.UI.ColorProfile != beforeProfile {
		t.Error("coming-soon row should not mutate config")
	}
}
