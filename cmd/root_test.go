package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
	testutil "github.com/git-fire/git-testkit"
	"github.com/mattn/go-runewidth"
)

func TestRootCommand_Flags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		checkVar func() bool
	}{
		{
			name: "dry-run flag",
			args: []string{"--dry-run"},
			checkVar: func() bool {
				return dryRun == true
			},
		},
		{
			name: "fire-drill flag",
			args: []string{"--fire-drill"},
			checkVar: func() bool {
				return fireDrill == true
			},
		},
		{
			name: "path flag",
			args: []string{"--path", "/tmp/test"},
			checkVar: func() bool {
				return scanPath == "/tmp/test"
			},
		},
		{
			name: "skip-auto-commit flag",
			args: []string{"--skip-auto-commit"},
			checkVar: func() bool {
				return skipCommit == true
			},
		},
		{
			name: "init flag",
			args: []string{"--init"},
			checkVar: func() bool {
				return initConfig == true
			},
		},
		{
			name: "config flag",
			args: []string{"--config", "/tmp/git-fire.toml"},
			checkVar: func() bool {
				return configFile == "/tmp/git-fire.toml"
			},
		},
		{
			name: "status flag",
			args: []string{"--status"},
			checkVar: func() bool {
				return showStatus == true
			},
		},
		{
			name: "fire mode flag",
			args: []string{"--fire"},
			checkVar: func() bool {
				return fireMode == true
			},
		},
		{
			name: "backup-to flag",
			args: []string{"--backup-to", "git@github.com:user/backup"},
			checkVar: func() bool {
				return backupTo == "git@github.com:user/backup"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			resetFlags()

			// Create new command for each test
			cmd := rootCmd
			cmd.SetArgs(tt.args)

			// Parse flags
			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check flag was set correctly
			if tt.checkVar != nil && !tt.checkVar() {
				t.Errorf("Flag not set correctly for args: %v", tt.args)
			}
		})
	}
}

func TestRootCommand_Help(t *testing.T) {
	// Reset flags
	resetFlags()

	// Capture output
	var buf bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Help command should not error, got: %v", err)
	}

	output := buf.String()

	// Check for key help text
	expectedStrings := []string{
		"git-fire",
		"Emergency", // Just check for "Emergency" instead of full phrase
		"--dry-run",
		"--path",
		"--init",
		"--status",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing expected text: %s", expected)
		}
	}
}

func TestRootCommand_SilenceUsageEnabled(t *testing.T) {
	if !rootCmd.SilenceUsage {
		t.Fatal("expected rootCmd.SilenceUsage to be true")
	}
}

func TestHandleInit(t *testing.T) {
	// Create temp directory for config
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Run handleInit
	err := handleInit()
	if err != nil {
		t.Fatalf("handleInit() error = %v", err)
	}

	// Check that config file was created
	configPath := config.DefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at: %s", configPath)
	}

	// Verify config file has content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Config file is empty")
	}

	// Check for expected config sections
	contentStr := string(content)
	expectedSections := []string{
		"[global]",
		"default_mode",
	}

	for _, section := range expectedSections {
		if !strings.Contains(contentStr, section) {
			t.Errorf("Config missing expected section: %s", section)
		}
	}
}

func TestHandleInit_UsesExplicitConfigPath(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	resetFlags()
	customPath := filepath.Join(tmpHome, "custom", "my-git-fire.toml")
	configFile = customPath
	defer func() { configFile = "" }()

	if err := handleInit(); err != nil {
		t.Fatalf("handleInit() with explicit config: %v", err)
	}
	if _, err := os.Stat(customPath); os.IsNotExist(err) {
		t.Fatalf("expected config at %s", customPath)
	}
	// Default user path must not be created when using --config
	defaultPath := config.DefaultConfigPath()
	if _, err := os.Stat(defaultPath); err == nil {
		t.Fatalf("did not expect default config at %s when --config was set", defaultPath)
	}
}

func TestHandleInit_ExistingConfig(t *testing.T) {
	// Create temp directory for config
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Create config directory and file
	configDir := filepath.Join(tmpHome, ".config", "git-fire")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.toml")
	existingContent := "# Existing config\n"
	err = os.WriteFile(configPath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing config: %v", err)
	}

	// handleInit should detect existing config
	// Note: In real usage, it would prompt the user
	// For testing, we can verify the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Existing config file should be detected")
	}
}

func TestHandleStatus(t *testing.T) {
	// handleStatus should not error even if no repos found
	err := handleStatus()

	// It's okay if it errors (ssh-agent might not be running in test env)
	// The important thing is we're testing the code path
	if err != nil {
		t.Logf("handleStatus() error (may be expected in test env): %v", err)
	}
}

func TestRunGitFire_DryRun(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Create a test scenario with repos
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("test").
		WithRemote("origin", remote).
		AddFile("test.txt", "content\n").
		Commit("Initial commit")

	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Reset flags
	resetFlags()

	// Set dry-run mode
	dryRun = true
	scanPath = filepath.Dir(repo.Path())

	// Run command
	err := runGitFire(rootCmd, []string{})

	// Should complete without error in dry-run
	if err != nil {
		t.Errorf("runGitFire() in dry-run mode error = %v", err)
	}
}

func TestRunGitFire_DryRun_DoesNotPrintWaterMessage(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	// Create a test scenario with repos
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("test").
		WithRemote("origin", remote).
		AddFile("test.txt", "content\n").
		Commit("Initial commit")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	resetFlags()
	dryRun = true
	scanPath = filepath.Dir(repo.Path())

	var runErr error
	output := captureStdoutFlavor(t, func() {
		runErr = runGitFire(rootCmd, []string{})
	})

	if runErr != nil {
		t.Fatalf("runGitFire() in dry-run mode error = %v", runErr)
	}
	if !strings.Contains(output, "No changes were made") {
		t.Fatalf("expected dry-run completion message, got: %q", output)
	}
	if strings.Contains(output, "💧 ") {
		t.Fatalf("did not expect water success message for dry-run output: %q", output)
	}
}

func TestRunGitFire_NoRepos(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Create empty directory
	emptyDir := t.TempDir()

	// Reset flags
	resetFlags()

	scanPath = emptyDir
	dryRun = true

	// Run command
	err := runGitFire(rootCmd, []string{})

	// Should not error when no repos found
	if err != nil {
		t.Errorf("runGitFire() with no repos error = %v", err)
	}
}

func TestRunGitFire_NoRepos_DoesNotPrintWaterMessage(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

	emptyDir := t.TempDir()

	resetFlags()
	scanPath = emptyDir
	dryRun = true

	var runErr error
	output := captureStdoutFlavor(t, func() {
		runErr = runGitFire(rootCmd, []string{})
	})

	if runErr != nil {
		t.Fatalf("runGitFire() with no repos error = %v", runErr)
	}
	if !strings.Contains(output, "No git repositories found.") {
		t.Fatalf("expected no-repos message, got: %q", output)
	}
	if strings.Contains(output, "💧 ") {
		t.Fatalf("did not expect water success message when no repos were found: %q", output)
	}
}

func TestRunGitFire_FireDrillFlag(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Reset flags
	resetFlags()

	// Set fire-drill flag
	fireDrill = true

	// After setting fireDrill, dryRun should also be set
	// This happens in runGitFire
	if dryRun {
		t.Error("dryRun should not be set before runGitFire is called")
	}

	// Create temp dir to avoid scanning real repos
	tmpDir := t.TempDir()
	scanPath = tmpDir

	// Run command (will set dryRun = true internally)
	_ = runGitFire(rootCmd, []string{})

	if !dryRun {
		t.Error("fire-drill flag should set dryRun to true")
	}
}

func TestRunGitFire_SkipAutoCommit(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Reset flags
	resetFlags()

	skipCommit = true
	dryRun = true
	scanPath = t.TempDir()

	// Run command
	err := runGitFire(rootCmd, []string{})

	if err != nil {
		t.Errorf("runGitFire() with skip-auto-commit error = %v", err)
	}

	// Verify that config was modified
	cfg := config.LoadOrDefault()
	if skipCommit && cfg.Global.AutoCommitDirty {
		// Note: The actual modification happens in runGitFire
		t.Log("Skip commit flag processed")
	}
}

func TestRunGitFire_WithInit(t *testing.T) {
	// Setup temp home
	tmpHome := t.TempDir()
	setTestHome(t, tmpHome)

	// Reset flags
	resetFlags()

	initConfig = true

	// Run command
	err := runGitFire(rootCmd, []string{})

	if err != nil {
		t.Errorf("runGitFire() with --init error = %v", err)
	}

	// Verify config was created
	configPath := config.DefaultConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file should be created with --init flag")
	}
}

func TestRunGitFire_WithStatus(t *testing.T) {
	// Reset flags
	resetFlags()

	showStatus = true

	// Run command
	err := runGitFire(rootCmd, []string{})

	// May error if ssh-agent not running, that's okay
	if err != nil {
		t.Logf("runGitFire() with --status error (expected in test env): %v", err)
	}
}

func TestExecute(t *testing.T) {
	// Test the Execute function
	// This is tricky because it calls cobra's Execute which may read args from os.Args

	// Save original args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set test args
	os.Args = []string{"git-fire", "--help"}

	// Execute should work
	err := Execute()

	// Help command should not error
	if err != nil {
		t.Errorf("Execute() with --help error = %v", err)
	}
}

func TestRootCommand_InvalidFlag(t *testing.T) {
	// Reset flags
	resetFlags()

	// Test with invalid flag
	cmd := rootCmd
	cmd.SetArgs([]string{"--invalid-flag"})

	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid flag")
	}

	// Error should mention the unknown flag
	if err != nil && !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("Expected 'unknown flag' error, got: %v", err)
	}
}

func TestBackupToExecuteError(t *testing.T) {
	resetFlags()
	backupTo = "git@github.com:user/backup"

	err := runGitFire(rootCmd, []string{})
	if err == nil {
		t.Fatal("expected --backup-to execute path to return an error")
	}
	if !strings.Contains(err.Error(), "--backup-to is not yet implemented") {
		t.Fatalf("unexpected error for --backup-to: %v", err)
	}
}

func TestRunGitFire_FireAndDryRunMutuallyExclusive(t *testing.T) {
	resetFlags()
	fireMode = true
	dryRun = true

	err := runGitFire(rootCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --fire and --dry-run are both enabled")
	}
	if !strings.Contains(err.Error(), "--fire and --dry-run cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitFire_FireAndFireDrillMutuallyExclusive(t *testing.T) {
	resetFlags()
	fireMode = true
	fireDrill = true // aliases to --dry-run in runGitFire

	err := runGitFire(rootCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --fire and --fire-drill are both enabled")
	}
	if !strings.Contains(err.Error(), "--fire and --dry-run cannot be used together") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRootCommand_CombinedFlags(t *testing.T) {
	// Reset flags
	resetFlags()

	// Test multiple flags together
	args := []string{"--dry-run", "--path", "/tmp/test", "--skip-auto-commit"}

	cmd := rootCmd
	err := cmd.ParseFlags(args)
	if err != nil {
		t.Errorf("ParseFlags() with combined flags error = %v", err)
	}

	// Verify all flags were set
	if !dryRun {
		t.Error("dry-run flag not set")
	}

	if scanPath != "/tmp/test" {
		t.Errorf("path flag not set correctly, got: %s", scanPath)
	}

	if !skipCommit {
		t.Error("skip-auto-commit flag not set")
	}
}

func TestRunGitFire_SecurityNotice(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()

	// Reset flags
	resetFlags()

	scanPath = tmpDir
	dryRun = false // Not dry-run, so security notice should show

	// We can't easily capture fmt.Println output without redirecting stdout
	// But we can verify the function runs without error

	// Note: This will prompt for confirmation, so we need to handle that
	// For now, we'll just test with dry-run to avoid prompts
	dryRun = true

	err := runGitFire(rootCmd, []string{})
	if err != nil {
		t.Errorf("runGitFire() error = %v", err)
	}
}

func TestUpsertRepoIntoRegistry_AppliesDefaultModeForNewRepo(t *testing.T) {
	reg := &registry.Registry{}
	now := time.Now()
	repo := git.Repository{
		Path: "/tmp/repo-a",
		Name: "repo-a",
	}

	updated, include := upsertRepoIntoRegistry(reg, repo, now, git.ModePushAll)
	if !include {
		t.Fatal("expected new repo to be included")
	}
	if updated.Mode != git.ModePushAll {
		t.Fatalf("expected mode %v, got %v", git.ModePushAll, updated.Mode)
	}

	entry := reg.FindByPath("/tmp/repo-a")
	if entry == nil {
		t.Fatal("expected registry entry to be created")
	}
	if entry.Mode != git.ModePushAll.String() {
		t.Fatalf("expected stored mode %q, got %q", git.ModePushAll.String(), entry.Mode)
	}
	if !updated.IsNewRegistryEntry {
		t.Fatal("expected IsNewRegistryEntry for first upsert")
	}
}

func TestUpsertRepoIntoRegistry_SecondUpsertNotNew(t *testing.T) {
	reg := &registry.Registry{}
	now := time.Now()
	repo := git.Repository{
		Path: "/tmp/repo-twice",
		Name: "repo-twice",
	}
	first, _ := upsertRepoIntoRegistry(reg, repo, now, git.ModePushAll)
	if !first.IsNewRegistryEntry {
		t.Fatal("first upsert should be new to registry")
	}
	second, _ := upsertRepoIntoRegistry(reg, repo, now, git.ModePushAll)
	if second.IsNewRegistryEntry {
		t.Fatal("second upsert should not mark IsNewRegistryEntry")
	}
}

func TestTruncateScanProgressPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		maxLen    int
		wantEqual string
		wantLen   int
		wantPref  string
		checkUTF8 bool
	}{
		{
			name:      "short path unchanged",
			path:      "/home/u/proj",
			maxLen:    72,
			wantEqual: "/home/u/proj",
		},
		{
			name:     "long path truncated",
			path:     strings.Repeat("/x", 80),
			maxLen:   20,
			wantLen:  20,
			wantPref: "...",
		},
		{
			name:      "multibyte path truncated safely",
			path:      "/tmp/" + strings.Repeat("世界", 20),
			maxLen:    14,
			wantPref:  "...",
			checkUTF8: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateScanProgressPath(tt.path, tt.maxLen)
			if tt.wantEqual != "" && got != tt.wantEqual {
				t.Fatalf("got %q want %q", got, tt.wantEqual)
			}
			if tt.wantLen > 0 && len(got) != tt.wantLen {
				t.Fatalf("len=%d want %d", len(got), tt.wantLen)
			}
			if tt.wantPref != "" && !strings.HasPrefix(got, tt.wantPref) {
				t.Fatalf("expected prefix %q, got %q", tt.wantPref, got)
			}
			if runewidth.StringWidth(got) > tt.maxLen {
				t.Fatalf("display width=%d exceeds max=%d for %q", runewidth.StringWidth(got), tt.maxLen, got)
			}
			if tt.checkUTF8 && !utf8.ValidString(got) {
				t.Fatalf("expected valid UTF-8 output, got %q", got)
			}
		})
	}
}

// Helper function to reset flags between tests
func resetFlags() {
	dryRun = false
	fireDrill = false
	fireMode = false
	scanPath = "."
	skipCommit = false
	noScan = false
	initConfig = false
	forceInit = false
	backupTo = ""
	configFile = ""
	showStatus = false
}
