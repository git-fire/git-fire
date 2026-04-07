package cmd

import (
	"bytes"
	"fmt"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-fire/internal/registry"
	testutil "github.com/git-fire/git-testkit"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	setTestUserDirs(t, tmpHome)

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
	regPath := testRegistryPath(t)
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
	setTestUserDirs(t, tmpHome)

	// Write a corrupt registry file at the same path the app resolves (e.g. macOS Application Support).
	regPath := testRegistryPath(t)
	if err := os.MkdirAll(filepath.Dir(regPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(regPath, []byte("not valid toml [[["), 0o600); err != nil {
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
	setTestUserDirs(t, tmpHome)

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
	regPath := testRegistryPath(t)
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
	setTestUserDirs(t, tmpHome)

	regPath := testRegistryPath(t)
	if err := os.MkdirAll(filepath.Dir(regPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(regPath, []byte("not valid toml [[["), 0o600); err != nil {
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
	setTestUserDirs(t, tmpHome)

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

// ---- repos subcommand function coverage ----

// isolateHome redirects HOME so loadRegistry() uses a temp registry.
func isolateHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	setTestUserDirs(t, tmp)
	return tmp
}

// setTestUserDirs normalizes user-dir environment variables for tests that
// depend on UserConfigDir/UserCacheDir path resolution (Linux/macOS/Windows).
func setTestUserDirs(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
}

// testRegistryPath returns the path handleStatus/runGitFire use after setTestUserDirs (matches macOS vs XDG layout).
func testRegistryPath(t *testing.T) string {
	t.Helper()
	p, err := registry.DefaultRegistryPath()
	if err != nil {
		t.Fatalf("DefaultRegistryPath: %v", err)
	}
	return p
}

func TestLoadRegistry(t *testing.T) {
	isolateHome(t)

	reg, path, err := loadRegistry()
	if err != nil {
		t.Fatalf("loadRegistry() error = %v", err)
	}
	if reg == nil {
		t.Error("expected non-nil registry")
	}
	if path == "" {
		t.Error("expected non-empty path")
	}
}

func TestReposList_FunctionDirectly(t *testing.T) {
	isolateHome(t)
	// Empty registry — should print "No repositories" message
	if err := reposList(reposListCmd, nil); err != nil {
		t.Errorf("reposList() error = %v", err)
	}
}

func TestReposList_WithData(t *testing.T) {
	tmpHome := isolateHome(t)

	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	reg := &registry.Registry{}
	reg.Upsert(registry.RegistryEntry{
		Path: "/some/repo", Name: "repo", Status: registry.StatusActive,
		Mode: "push-known-branches",
	})
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	if err := reposList(reposListCmd, nil); err != nil {
		t.Errorf("reposList() error = %v", err)
	}
}

func TestReposScan_Function(t *testing.T) {
	isolateHome(t)

	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{Name: "scanme"})
	scanRoot := filepath.Dir(repoPath)

	if err := reposScan(reposScanCmd, []string{scanRoot}); err != nil {
		t.Errorf("reposScan() error = %v", err)
	}
}

func TestReposScan_DefaultPath(t *testing.T) {
	isolateHome(t)

	// Create a real repo and scan its parent so reposScan actually discovers it
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{Name: "scanme2"})
	scanRoot := filepath.Dir(repoPath)

	if err := reposScan(reposScanCmd, []string{scanRoot}); err != nil {
		t.Errorf("reposScan() with explicit path error = %v", err)
	}
}

func TestReposScan_NewEntriesUseConfiguredDefaultMode(t *testing.T) {
	tmpHome := isolateHome(t)

	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{Name: "mode-default"})
	scanRoot := filepath.Dir(repoPath)

	cfgPath := filepath.Join(tmpHome, ".config", "git-fire", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte("[global]\ndefault_mode = \"push-all\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := reposScan(reposScanCmd, []string{scanRoot}); err != nil {
		t.Fatalf("reposScan() error = %v", err)
	}

	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	reg, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	absRepoPath, _ := filepath.Abs(repoPath)
	entry := reg.FindByPath(absRepoPath)
	if entry == nil {
		t.Fatalf("expected repo entry %s in registry", absRepoPath)
	}
	if entry.Mode != "push-all" {
		t.Fatalf("expected mode push-all, got %q", entry.Mode)
	}
}

func TestRunGitFire_HonorsConfiguredScanDepth(t *testing.T) {
	tmpHome := isolateHome(t)

	scanRoot := t.TempDir()
	nestedRepo := filepath.Join(scanRoot, "a", "b", "deep-repo")
	if err := os.MkdirAll(nestedRepo, 0o755); err != nil {
		t.Fatalf("mkdir nested repo: %v", err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = nestedRepo
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(nestedRepo, "README.md"), []byte("hi\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "init")

	cfgPath := filepath.Join(tmpHome, ".config", "git-fire", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	cfg := fmt.Sprintf("[global]\nscan_path = %q\nscan_depth = 0\nauto_commit_dirty = false\n", scanRoot)
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags()
	dryRun = true

	var runErr error
	out := captureStdout(t, func() {
		runErr = runGitFire(rootCmd, []string{})
	})
	if runErr != nil {
		t.Fatalf("runGitFire() error = %v", runErr)
	}
	if !strings.Contains(out, "✓ Found 0 repositories") {
		t.Fatalf("expected configured scan_depth to exclude deep repo, output:\n%s", out)
	}
}

func TestReposRemove_Function(t *testing.T) {
	tmpHome := isolateHome(t)

	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	reg := &registry.Registry{}
	abs, _ := filepath.Abs("/fake/path")
	reg.Upsert(registry.RegistryEntry{Path: abs, Name: "fake", Status: registry.StatusActive})
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	if err := reposRemove(reposRemoveCmd, []string{abs}); err != nil {
		t.Errorf("reposRemove() error = %v", err)
	}

	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if loaded.FindByPath(abs) != nil {
		t.Error("entry should have been removed")
	}
}

func TestReposRemove_NotFound_Function(t *testing.T) {
	isolateHome(t)
	if err := reposRemove(reposRemoveCmd, []string{"/definitely/not/there"}); err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestSetRepoStatus_IgnoreUnignore(t *testing.T) {
	tmpHome := isolateHome(t)

	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	reg := &registry.Registry{}
	abs, _ := filepath.Abs("/some/repo")
	reg.Upsert(registry.RegistryEntry{Path: abs, Name: "repo", Status: registry.StatusActive})
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	if err := setRepoStatus(abs, registry.StatusIgnored, "ignored"); err != nil {
		t.Errorf("setRepoStatus(ignored) error = %v", err)
	}
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if e := loaded.FindByPath(abs); e == nil || e.Status != registry.StatusIgnored {
		t.Error("status should be ignored")
	}

	if err := setRepoStatus(abs, registry.StatusActive, "active"); err != nil {
		t.Errorf("setRepoStatus(active) error = %v", err)
	}
	loaded2, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if e := loaded2.FindByPath(abs); e == nil || e.Status != registry.StatusActive {
		t.Error("status should be active after unignore")
	}
}

func TestSetRepoStatus_NotFound(t *testing.T) {
	isolateHome(t)
	if err := setRepoStatus("/not/tracked", registry.StatusIgnored, "ignored"); err == nil {
		t.Error("expected error for untracked path")
	}
}

func TestReposIgnoreAndUnignore_Wrappers(t *testing.T) {
	tmpHome := isolateHome(t)

	regPath := filepath.Join(tmpHome, ".config", "git-fire", "repos.toml")
	reg := &registry.Registry{}
	abs, _ := filepath.Abs("/wrapper/repo")
	reg.Upsert(registry.RegistryEntry{Path: abs, Name: "repo", Status: registry.StatusActive})
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	if err := reposIgnore(reposIgnoreCmd, []string{abs}); err != nil {
		t.Fatalf("reposIgnore() error = %v", err)
	}
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if e := loaded.FindByPath(abs); e == nil || e.Status != registry.StatusIgnored {
		t.Fatalf("reposIgnore should mark status as ignored")
	}

	if err := reposUnignore(reposUnignoreCmd, []string{abs}); err != nil {
		t.Fatalf("reposUnignore() error = %v", err)
	}
	loaded, err = registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if e := loaded.FindByPath(abs); e == nil || e.Status != registry.StatusActive {
		t.Fatalf("reposUnignore should mark status as active")
	}
}

func TestReposRemove_UsesXDGRegistryPath(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)
	cfgRoot := filepath.Join(tmpHome, ".xdg_config")
	t.Setenv("XDG_CONFIG_HOME", cfgRoot)
	t.Setenv("APPDATA", cfgRoot)
	t.Setenv("LOCALAPPDATA", cfgRoot)

	regPath := filepath.Join(cfgRoot, "git-fire", "repos.toml")
	reg := &registry.Registry{}
	abs, _ := filepath.Abs("/xdg/repo")
	reg.Upsert(registry.RegistryEntry{Path: abs, Name: "xdg", Status: registry.StatusActive})
	if err := registry.Save(reg, regPath); err != nil {
		t.Fatalf("seed registry: %v", err)
	}

	if err := reposRemove(reposRemoveCmd, []string{abs}); err != nil {
		t.Fatalf("reposRemove() error = %v", err)
	}
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if loaded.FindByPath(abs) != nil {
		t.Fatal("entry should be removed when using XDG config path")
	}
}

func TestStatusLabel_AllValues(t *testing.T) {
	tests := []struct{ in, want string }{
		{registry.StatusActive, "active "},
		{"", "active "},
		{registry.StatusMissing, "MISSING"},
		{registry.StatusIgnored, "ignored"},
		{"custom-status", "custom-status"},
	}
	for _, tt := range tests {
		if got := statusLabel(tt.in); got != tt.want {
			t.Errorf("statusLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHandleInit_ForceFlag(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	// First call creates the file
	forceInit = false
	if err := handleInit(); err != nil {
		t.Fatalf("first handleInit(): %v", err)
	}

	// Second call with --force should overwrite without prompting
	forceInit = true
	defer func() { forceInit = false }()
	if err := handleInit(); err != nil {
		t.Fatalf("handleInit() with force: %v", err)
	}
}

func TestPrintResult(t *testing.T) {
	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w

	printResult(&executor.ExecutionResult{
		Success:  3,
		Failed:   1,
		Skipped:  2,
		Duration: 5 * time.Second,
	}, "/tmp/fake.log")

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "3") {
		t.Error("expected success count in output")
	}
}
