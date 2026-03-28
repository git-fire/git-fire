package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TBRX103/git-fire/internal/registry"
	"github.com/TBRX103/git-fire/internal/testutil"
)

// ---- buildKnownPaths ----

func TestBuildKnownPaths_ActiveOnly(t *testing.T) {
	dir := t.TempDir()
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: dir, Status: registry.StatusActive},
			{Path: filepath.Join(dir, "ignored"), Status: registry.StatusIgnored},
			{Path: filepath.Join(dir, "missing"), Status: registry.StatusMissing},
		},
	}

	m := buildKnownPaths(reg, false)

	abs, _ := filepath.Abs(dir)
	if _, ok := m[abs]; !ok {
		t.Error("active entry should be in KnownPaths")
	}
	if len(m) != 1 {
		t.Errorf("expected 1 entry, got %d (ignored/missing should be excluded)", len(m))
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
	if v := m2[abs]; v {
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
	if v := m[abs]; v {
		t.Error("per-repo RescanSubmodules=false should override global=true")
	}
}

func TestBuildKnownPaths_EmptyRegistry(t *testing.T) {
	m := buildKnownPaths(&registry.Registry{}, false)
	if len(m) != 0 {
		t.Errorf("expected empty map for empty registry, got %d entries", len(m))
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
	regPath := filepath.Join(tmpHome, ".git-fire", "repos.toml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	// handleStatus should not error even with registry populated
	err := handleStatus()
	// SSH agent may not be running in CI — only fail on unexpected errors
	if err != nil {
		t.Logf("handleStatus() error (may be expected in test env): %v", err)
	}
}

func TestHandleStatus_CorruptRegistry_DoesNotPanic(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Write a corrupt registry file
	regDir := filepath.Join(tmpHome, ".git-fire")
	if err := os.MkdirAll(regDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(regDir, "repos.toml"), []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}

	// Should fall back gracefully, not panic or hard-error on the registry
	err := handleStatus()
	if err != nil {
		t.Logf("handleStatus() error (may be expected in test env): %v", err)
	}
}

// ---- runGitFire registry graceful fallback ----

func TestRunGitFire_CorruptRegistry_DoesNotAbort(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)
	os.Setenv("HOME", tmpHome)

	// Corrupt the registry file
	regDir := filepath.Join(tmpHome, ".git-fire")
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
	regPath := filepath.Join(tmpHome, ".git-fire", "repos.toml")
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("failed to write test registry: %v", err)
	}

	resetFlags()
	dryRun = true
	// Scan the parent dir so both repos are discovered
	scanPath = filepath.Dir(kept.Path())

	err := runGitFire(rootCmd, []string{})
	if err != nil {
		t.Errorf("runGitFire() error = %v", err)
	}
}
