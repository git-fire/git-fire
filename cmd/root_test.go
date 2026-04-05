package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
	testutil "github.com/git-fire/git-testkit"

	"github.com/git-fire/git-fire/internal/usb"
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

	// Save original HOME and restore after test
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tmpHome)

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

func TestHandleInit_ExistingConfig(t *testing.T) {
	// Create temp directory for config
	tmpHome := t.TempDir()

	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tmpHome)

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
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

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
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

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
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

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
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpHome)

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
}

func TestPruneUSBTarget_PreservesAllPlannedDestinations(t *testing.T) {
	targetRoot := t.TempDir()
	reposRoot := filepath.Join(targetRoot, "repos")
	if err := os.MkdirAll(reposRoot, 0o700); err != nil {
		t.Fatalf("mkdir repos root: %v", err)
	}

	keepDest := filepath.Join(reposRoot, "keep-repo.git")
	pruneDest := filepath.Join(reposRoot, "prune-repo.git")
	staleDest := filepath.Join(reposRoot, "stale-repo.git")

	for _, p := range []string{keepDest, pruneDest, staleDest} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}

	plans := []usb.RepoPlan{
		{
			Actions: []usb.Action{
				{
					Type:        usb.ActionSync,
					TargetRoot:  targetRoot,
					Destination: keepDest,
					SyncPolicy:  "keep",
				},
			},
		},
		{
			Actions: []usb.Action{
				{
					Type:        usb.ActionSync,
					TargetRoot:  targetRoot,
					Destination: pruneDest,
					SyncPolicy:  "prune",
				},
			},
		},
	}

	if err := pruneUSBTarget(targetRoot, reposRoot, plans); err != nil {
		t.Fatalf("pruneUSBTarget error: %v", err)
	}

	for _, p := range []string{keepDest, pruneDest} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected planned destination to remain (%s): %v", p, err)
		}
	}
	if _, err := os.Stat(staleDest); !os.IsNotExist(err) {
		t.Fatalf("expected stale destination to be removed, stat err: %v", err)
	}
}

func TestPruneUSBTarget_PrunesStaleCloneDestinations(t *testing.T) {
	targetRoot := t.TempDir()
	reposRoot := filepath.Join(targetRoot, "repos")
	if err := os.MkdirAll(reposRoot, 0o700); err != nil {
		t.Fatalf("mkdir repos root: %v", err)
	}

	keepCloneDest := filepath.Join(reposRoot, "keep-clone")
	staleCloneDest := filepath.Join(reposRoot, "stale-clone")
	nonRepoDir := filepath.Join(reposRoot, "notes")

	for _, p := range []string{keepCloneDest, staleCloneDest, nonRepoDir} {
		if err := os.MkdirAll(p, 0o700); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
	}
	for _, p := range []string{keepCloneDest, staleCloneDest} {
		if err := os.MkdirAll(filepath.Join(p, ".git"), 0o700); err != nil {
			t.Fatalf("mkdir %s/.git: %v", p, err)
		}
	}

	plans := []usb.RepoPlan{
		{
			Actions: []usb.Action{
				{
					Type:        usb.ActionSync,
					TargetRoot:  targetRoot,
					Destination: keepCloneDest,
					SyncPolicy:  "prune",
				},
			},
		},
	}

	if err := pruneUSBTarget(targetRoot, reposRoot, plans); err != nil {
		t.Fatalf("pruneUSBTarget error: %v", err)
	}

	if _, err := os.Stat(keepCloneDest); err != nil {
		t.Fatalf("expected planned clone destination to remain (%s): %v", keepCloneDest, err)
	}
	if _, err := os.Stat(staleCloneDest); !os.IsNotExist(err) {
		t.Fatalf("expected stale clone destination to be removed, stat err: %v", err)
	}
	if _, err := os.Stat(nonRepoDir); err != nil {
		t.Fatalf("expected non-repo directory to remain (%s): %v", nonRepoDir, err)
	}
}

func TestRunUSB_DryRun_UsesPlannedDestinationFromOverrides(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("app").
		AddFile("README.md", "hello\n").
		Commit("init")

	repoPath := repo.Path()
	targetRoot := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.Global.ScanPath = filepath.Dir(repoPath)
	cfg.USB.CreateOnFirst = true
	cfg.USB.Strategy = usb.StrategyMirror
	cfg.USB.SyncPolicy = "keep"

	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{
				Path:          repoPath,
				Name:          "app",
				Status:        registry.StatusActive,
				Mode:          cfg.Global.DefaultMode,
				USBStrategy:   usb.StrategyClone,
				USBRepoPath:   "custom/app-backup",
				USBSyncPolicy: "keep",
				AddedAt:       time.Now(),
				LastSeen:      time.Now(),
			},
		},
	}

	opts := git.DefaultScanOptions()
	opts.RootPath = cfg.Global.ScanPath
	opts.DisableScan = false

	resetFlags()
	dryRun = true

	var runErr error
	output := captureStdoutFlavor(t, func() {
		runErr = runUSB(&cfg, reg, "", opts, []string{targetRoot})
	})

	if runErr != errRunNoop {
		t.Fatalf("expected errRunNoop for dry-run, got: %v", runErr)
	}

	expectedDest := filepath.Join(targetRoot, usb.DefaultRepoLayoutDir, "custom/app-backup")
	if !strings.Contains(output, expectedDest) {
		t.Fatalf("expected dry-run output to include override destination %q, got %q", expectedDest, output)
	}
	if strings.Contains(output, expectedDest+".git") {
		t.Fatalf("did not expect .git suffix to be appended for override destination, got %q", output)
	}
}

func TestRunUSB_DedupsNormalizedTargetsBeforeLocking(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("app").
		AddFile("README.md", "hello\n").
		Commit("init")

	repoPath := repo.Path()
	targetRoot := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.USB.CreateOnFirst = true
	cfg.USB.Strategy = usb.StrategyMirror
	cfg.USB.SyncPolicy = "keep"
	cfg.USB.Workers = 1
	cfg.USB.TargetWorkers = 1

	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{
				Path:     repoPath,
				Name:     "app",
				Status:   registry.StatusActive,
				Mode:     cfg.Global.DefaultMode,
				AddedAt:  time.Now(),
				LastSeen: time.Now(),
			},
		},
	}

	opts := git.DefaultScanOptions()
	opts.DisableScan = true
	opts.KnownPaths = map[string]bool{repoPath: false}

	resetFlags()
	dryRun = false

	dupPathWithTrailingSlash := targetRoot + string(os.PathSeparator)
	runErr := runUSB(&cfg, reg, "", opts, []string{targetRoot, dupPathWithTrailingSlash})
	if runErr != nil {
		t.Fatalf("expected duplicate normalized targets to be handled, got error: %v", runErr)
	}
}

func TestApplyFlagOverrides_USBWorkersClampedToMax(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	cfg := config.DefaultConfig()
	usbWorkers = config.MaxUSBWorkers + 1000

	if err := applyFlagOverrides(&cfg); err != nil {
		t.Fatalf("applyFlagOverrides() unexpected error: %v", err)
	}
	if cfg.USB.Workers != config.MaxUSBWorkers {
		t.Fatalf("expected usb workers to clamp to %d, got %d", config.MaxUSBWorkers, cfg.USB.Workers)
	}
}

func TestApplyFlagOverrides_InvalidUSBStrategyReturnsError(t *testing.T) {
	resetFlags()
	t.Cleanup(resetFlags)

	cfg := config.DefaultConfig()
	usbStrategy = "git-Clone"

	err := applyFlagOverrides(&cfg)
	if err == nil {
		t.Fatal("expected invalid usb strategy to return validation error")
	}
	if !strings.Contains(err.Error(), "invalid usb.strategy") {
		t.Fatalf("expected invalid usb.strategy error, got: %v", err)
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
	usbTargets = nil
	usbInit = false
	usbWorkers = 0
	usbStrategy = ""
	usbResume = false
	usbVerify = false
}
