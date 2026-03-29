package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TBRX103/git-fire/internal/registry"
	"github.com/TBRX103/git-fire/internal/testutil"
)

// ---- buildKnownPaths ----

func TestBuildKnownPaths_ActiveMissingEmptyNotIgnored(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing")
	emptyStatusPath := filepath.Join(dir, "empty-status")
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: dir, Status: registry.StatusActive},
			{Path: filepath.Join(dir, "ignored"), Status: registry.StatusIgnored},
			{Path: missingPath, Status: registry.StatusMissing},
			{Path: emptyStatusPath, Status: ""},
		},
	}

	m := buildKnownPaths(reg, false)

	abs, _ := filepath.Abs(dir)
	if _, ok := m[abs]; !ok {
		t.Error("active entry should be in KnownPaths")
	}
	absMissing, _ := filepath.Abs(missingPath)
	if _, ok := m[absMissing]; !ok {
		t.Error("missing entry should be in KnownPaths (scanner skips if path gone)")
	}
	absEmpty, _ := filepath.Abs(emptyStatusPath)
	if _, ok := m[absEmpty]; !ok {
		t.Error("empty-status entry should be in KnownPaths")
	}
	if len(m) != 3 {
		t.Errorf("expected 3 entries (active+missing+empty), got %d", len(m))
	}
}

func TestBuildKnownPaths_GlobalRescanDefault(t *testing.T) {
	dir := t.TempDir()
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: dir, Status: registry.StatusActive, RescanSubmodules: nil},
		},
	}

	m := buildKnownPaths(reg, true)
	abs, _ := filepath.Abs(dir)
	if v, ok := m[abs]; !ok || !v {
		t.Error("should inherit global rescan=true when per-repo override is nil")
	}

	m2 := buildKnownPaths(reg, false)
	if v, ok := m2[abs]; !ok || v {
		t.Error("should inherit global rescan=false when per-repo override is nil")
	}
}

func TestBuildKnownPaths_PerRepoOverride(t *testing.T) {
	dir := t.TempDir()
	override := false
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: dir, Status: registry.StatusActive, RescanSubmodules: &override},
		},
	}

	// Global says true, but per-repo says false
	m := buildKnownPaths(reg, true)
	abs, _ := filepath.Abs(dir)
	if v, ok := m[abs]; !ok || v {
		t.Error("per-repo RescanSubmodules=false should override global=true")
	}
}

func TestBuildKnownPaths_EmptyRegistry(t *testing.T) {
	m := buildKnownPaths(&registry.Registry{}, false)
	if len(m) != 0 {
		t.Errorf("expected empty map for empty registry, got %d entries", len(m))
	}
}

func TestStatusLabel_EmptyString(t *testing.T) {
	if got := statusLabel(""); got != "active " {
		t.Errorf("statusLabel(\"\") = %q, want \"active \"", got)
	}
	if got := statusLabel(registry.StatusActive); got != "active " {
		t.Errorf("statusLabel(active) = %q, want \"active \"", got)
	}
}

// ---- handleStatus registry integration ----

func TestHandleStatus_IncludesRegistryRepos(t *testing.T) {
	// Set HOME to a temp dir so registry.DefaultRegistryPath() points there
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Create a real git repo to track
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("tracked").
		AddFile("readme.txt", "hello\n").
		Commit("init")

	// Write a registry that knows about this repo
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: repo.Path(), Name: "tracked", Status: registry.StatusActive},
		},
	}
	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	// handleStatus should not error even with registry populated
	err := handleStatus()
	if err != nil {
		if strings.Contains(err.Error(), "failed to get SSH status") {
			t.Skipf("skipping: SSH precondition not met in test environment: %v", err)
		}
		t.Fatalf("handleStatus() unexpected error: %v", err)
	}
}

func TestHandleStatus_CorruptRegistry_DoesNotPanic(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Write a corrupt registry file
	regDir := filepath.Join(tmpHome, ".config", "git-fire")
	if err := os.MkdirAll(regDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, "repos.toml"), []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}

	// Should fall back gracefully, not panic or hard-error on the registry
	err := handleStatus()
	if err != nil {
		if strings.Contains(err.Error(), "failed to get SSH status") {
			t.Skipf("skipping: SSH precondition not met in test environment: %v", err)
		}
		t.Fatalf("handleStatus() unexpected error: %v", err)
	}
}

// ---- registry status validation (os.Stat error handling) ----

func TestRunGitFire_PermissionDenied_DoesNotReactivateMissingRepo(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission denied as root")
	}

	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Create a path inside an unreadable parent dir so os.Stat returns EACCES,
	// not ENOENT. The parent has mode 0o000 so traversal is denied.
	lockedParent := filepath.Join(t.TempDir(), "locked")
	if err := os.MkdirAll(lockedParent, 0o700); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(lockedParent, "repo")
	if err := os.MkdirAll(targetPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(lockedParent, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(lockedParent, 0o700)

	// Register the path-inside-locked-dir as missing
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: targetPath, Name: "locked", Status: registry.StatusMissing},
		},
	}
	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	resetFlags()
	dryRun = true
	scanPath = t.TempDir()

	if err := runGitFire(rootCmd, []string{}); err != nil {
		t.Fatalf("runGitFire() error = %v", err)
	}

	// Reload registry — the repo must still be missing, not flipped to active
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	entry := loaded.FindByPath(targetPath)
	if entry == nil {
		t.Fatal("registry entry disappeared after run")
	}
	if entry.Status != registry.StatusMissing {
		t.Errorf("expected StatusMissing after non-ENOENT stat error, got %q", entry.Status)
	}
}

// ---- runGitFire registry graceful fallback ----

func TestRunGitFire_CorruptRegistry_DoesNotAbort(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Corrupt the registry file
	regDir := filepath.Join(tmpHome, ".config", "git-fire")
	if err := os.MkdirAll(regDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, "repos.toml"), []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}

	resetFlags()
	dryRun = true
	scanPath = t.TempDir()

	// The main flow must not abort due to an unreadable registry
	err := runGitFire(rootCmd, []string{})
	if err != nil {
		t.Errorf("runGitFire() should not abort when registry is corrupt, got: %v", err)
	}
}

// captureStdout redirects os.Stdout for the duration of fn and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("captureStdout: pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("captureStdout: read: %v", err)
	}
	return buf.String()
}

// ---- ignored-repo filtering (filepath.Abs failure safety) ----

func TestRunGitFire_IgnoredRepo_ExcludedFromBackup(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Create two repos
	scenario := testutil.NewScenario(t)
	kept := scenario.CreateRepo("kept").
		AddFile("a.txt", "a\n").
		Commit("init")
	ignored := scenario.CreateRepo("ignored").
		AddFile("b.txt", "b\n").
		Commit("init")

	// Register one as ignored
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: kept.Path(), Name: "kept", Status: registry.StatusActive},
			{Path: ignored.Path(), Name: "ignored", Status: registry.StatusIgnored},
		},
	}
	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	resetFlags()
	dryRun = true
	// Scan the parent dir so both repos are discovered
	scanPath = filepath.Dir(kept.Path())

	var runErr error
	out := captureStdout(t, func() {
		runErr = runGitFire(rootCmd, []string{})
	})
	if runErr != nil {
		t.Fatalf("runGitFire() error = %v", runErr)
	}

	// The "Selected repositories:" block lists repos by name with a bullet
	if strings.Contains(out, fmt.Sprintf("• %s", "ignored")) {
		t.Errorf("ignored repo should not appear in output:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("• %s", "kept")) {
		t.Errorf("kept repo should appear in output:\n%s", out)
	}
}
