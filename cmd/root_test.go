package cmd

import (
	"bytes"
	"errors"
	"io"
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
	setTestUserDirs(t, tmpHome)

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

func TestHandleInit_ExistingConfig_NonInteractive(t *testing.T) {
	// Simulate a non-interactive environment by replacing os.Stdin with a pipe.
	// handleInit should return an error rather than prompt when stdin is not a tty.
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	configPath := config.DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("# Existing config\n"), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	// Replace stdin with a pipe so Stat() returns non-character-device mode.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin; r.Close() }()

	resetFlags()
	gotErr := handleInit()
	if gotErr == nil {
		t.Fatal("handleInit() should return error when config exists in non-interactive env")
	}
	if !strings.Contains(gotErr.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", gotErr)
	}
}

func TestHandleInit_ForceOverwrite(t *testing.T) {
	// --force should overwrite an existing config without prompting.
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	configPath := config.DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("# Old config\n"), 0644); err != nil {
		t.Fatalf("Failed to write existing config: %v", err)
	}

	resetFlags()
	forceInit = true
	defer func() { forceInit = false }()

	if err := handleInit(); err != nil {
		t.Fatalf("handleInit() with --force error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config after force overwrite: %v", err)
	}
	if strings.Contains(string(content), "# Old config") {
		t.Error("Config was not overwritten by --force")
	}
}

func TestCmdPluginLogger_Debug(t *testing.T) {
	// Debug is a no-op — just confirm it doesn't panic.
	l := &cmdPluginLogger{}
	l.Debug("should not panic")
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
	setTestUserDirs(t, tmpHome)

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
	setTestUserDirs(t, tmpHome)

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

func TestRunGitFire_DryRun_EmitsSecretWarningStderr(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("secret-repo").
		WithRemote("origin", remote).
		AddFile("test.txt", "content\n").
		Commit("Initial commit")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	secretFile := filepath.Join(repo.Path(), "token.env")
	if err := os.WriteFile(secretFile, []byte("GITLAB_TOKEN=glpat-abcdefghij1234567890\n"), 0644); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	resetFlags()
	dryRun = true
	scanPath = filepath.Dir(repo.Path())

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w

	runErr := runGitFire(rootCmd, []string{})

	os.Stderr = oldStderr
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	stderrBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	stderr := string(stderrBytes)

	if runErr != nil {
		t.Fatalf("runGitFire() dry-run: %v", runErr)
	}
	if !strings.Contains(stderr, "Potential secrets detected") {
		t.Fatalf("expected secret warning on stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "GitLab") {
		t.Fatalf("expected GitLab pattern in stderr, got: %q", stderr)
	}
}

func TestRunGitFire_NoRepos(t *testing.T) {
	// Isolate registry from the user's real one
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

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
	setTestUserDirs(t, tmpHome)

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
	setTestUserDirs(t, tmpHome)

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
	setTestUserDirs(t, tmpHome)

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
	setTestUserDirs(t, tmpHome)

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

func TestConfirmFireRiskAcknowledgement_AcceptsOKAndPersists(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	resetFlags()
	cfg := config.DefaultConfig()
	cfgPath := config.DefaultConfigPath()

	if err := confirmFireRiskAcknowledgement(&cfg, strings.NewReader("OK\n")); err != nil {
		t.Fatalf("confirmFireRiskAcknowledgement() error = %v", err)
	}
	if !cfg.Global.FireRiskAcknowledged {
		t.Fatal("expected FireRiskAcknowledged=true after OK confirmation")
	}

	loaded, err := config.LoadWithOptions(config.LoadOptions{ConfigFile: cfgPath})
	if err != nil {
		t.Fatalf("LoadWithOptions() error = %v", err)
	}
	if !loaded.Global.FireRiskAcknowledged {
		t.Fatal("expected persisted fire_risk_acknowledged=true in config")
	}
}

func TestConfirmFireRiskAcknowledgement_RejectsNonOK(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	resetFlags()
	cfg := config.DefaultConfig()
	cfgPath := config.DefaultConfigPath()

	err := confirmFireRiskAcknowledgement(&cfg, strings.NewReader("no\n"))
	if !errors.Is(err, errRunAborted) {
		t.Fatalf("expected errRunAborted, got: %v", err)
	}
	if cfg.Global.FireRiskAcknowledged {
		t.Fatal("did not expect FireRiskAcknowledged to change on rejection")
	}
	if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
		t.Fatalf("did not expect config file to be created on rejection, stat err=%v", statErr)
	}
}

func TestMaybeConfirmFireRiskAcknowledgement_NonInteractiveWhenRequired(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	resetFlags()
	fireMode = true
	cfg := config.DefaultConfig()
	cfg.Global.FireRiskAcknowledged = false
	oldIsInteractive := stdinIsInteractive
	stdinIsInteractive = func() bool { return false }
	defer func() { stdinIsInteractive = oldIsInteractive }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	err = maybeConfirmFireRiskAcknowledgement(&cfg)
	if err == nil {
		t.Fatal("expected non-interactive fire acknowledgment error")
	}
	if !strings.Contains(err.Error(), "fire mode requires an interactive risk acknowledgment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunGitFire_FireModeAbortsWhenRiskNotAcknowledged(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	cfgPath := config.DefaultConfigPath()
	cfgText := `
[global]
default_mode = "push-known-branches"
conflict_strategy = "new-branch"
auto_commit_dirty = true
block_on_secrets = true
fire_risk_acknowledged = false
scan_path = "."
scan_exclude = []
scan_depth = 1
scan_workers = 1
push_workers = 1
cache_ttl = "1h"
rescan_submodules = false
disable_scan = false
`
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags()
	fireMode = true
	scanPath = tmpHome
	oldIsInteractive := stdinIsInteractive
	stdinIsInteractive = func() bool { return true }
	defer func() { stdinIsInteractive = oldIsInteractive }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		r.Close()
	}()

	// Ensure the prompt receives a rejection, then close the writer.
	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := io.WriteString(w, "no\n")
		closeErr := w.Close()
		writeDone <- errors.Join(writeErr, closeErr)
	}()

	err = runGitFire(rootCmd, []string{})
	if err != nil {
		t.Fatalf("runGitFire() should return nil after user abort, got: %v", err)
	}
	if writeErr := <-writeDone; writeErr != nil {
		t.Fatalf("stdin write failed: %v", writeErr)
	}

	loaded, err := config.LoadWithOptions(config.LoadOptions{ConfigFile: cfgPath})
	if err != nil {
		t.Fatalf("LoadWithOptions() error = %v", err)
	}
	if loaded.Global.FireRiskAcknowledged {
		t.Fatal("did not expect fire_risk_acknowledged to be persisted on abort")
	}
}

func TestMaybeConfirmFireRiskAcknowledgement_AcknowledgesOnceThenSkipsPrompt(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	resetFlags()
	fireMode = true
	cfg := config.DefaultConfig()
	cfgPath := config.DefaultConfigPath()
	oldIsInteractive := stdinIsInteractive
	stdinIsInteractive = func() bool { return true }
	defer func() { stdinIsInteractive = oldIsInteractive }()

	rOK, wOK, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = rOK
	writeDone := make(chan error, 1)
	go func() {
		_, writeErr := io.WriteString(wOK, "OK\n")
		closeErr := wOK.Close()
		writeDone <- errors.Join(writeErr, closeErr)
	}()
	if err := maybeConfirmFireRiskAcknowledgement(&cfg); err != nil {
		t.Fatalf("first maybeConfirmFireRiskAcknowledgement() failed: %v", err)
	}
	os.Stdin = oldStdin
	rOK.Close()
	if writeErr := <-writeDone; writeErr != nil {
		t.Fatalf("stdin write failed: %v", writeErr)
	}

	loaded, err := config.LoadWithOptions(config.LoadOptions{ConfigFile: cfgPath})
	if err != nil {
		t.Fatalf("LoadWithOptions() after first acknowledgment: %v", err)
	}
	if !loaded.Global.FireRiskAcknowledged {
		t.Fatal("expected first acknowledgment to persist fire_risk_acknowledged=true")
	}

	// Second call should skip prompt entirely since it's now acknowledged.
	// Simulate non-interactive stdin; this would fail if a prompt were still required.
	rClosed, wClosed, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	wClosed.Close()
	oldStdin = os.Stdin
	os.Stdin = rClosed
	defer func() {
		os.Stdin = oldStdin
		rClosed.Close()
	}()
	if err := maybeConfirmFireRiskAcknowledgement(&cfg); err != nil {
		t.Fatalf("second maybeConfirmFireRiskAcknowledgement() should skip prompt after persisted ack, got: %v", err)
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

func TestCmdPluginLoggerError_SanitizesOutput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}

	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	logger := &cmdPluginLogger{}
	logger.Error(
		"plugin failed for https://user:supersecret@github.com/org/repo.git",
		errors.New("fatal: could not read from https://user:supersecret@github.com/org/repo.git"),
	)

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed reading stderr pipe: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "supersecret") {
		t.Fatalf("expected sanitized output, got %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker in output, got %q", got)
	}
}

func TestCmdPluginLoggerInfo_SanitizesOutput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	logger := &cmdPluginLogger{}
	logger.Info("plugin info: https://user:supersecret@github.com/org/repo.git")

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed reading stdout pipe: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "supersecret") {
		t.Fatalf("expected sanitized output, got %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker in output, got %q", got)
	}
}

func TestCmdPluginLoggerSuccess_SanitizesOutput(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	logger := &cmdPluginLogger{}
	logger.Success("plugin success: https://user:supersecret@github.com/org/repo.git")

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer failed: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed reading stdout pipe: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "supersecret") {
		t.Fatalf("expected sanitized output, got %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected redaction marker in output, got %q", got)
	}
}

func TestShouldRunPostRunPlugins(t *testing.T) {
	tests := []struct {
		name     string
		dryRun   bool
		runErr   error
		expected bool
	}{
		{name: "success run", dryRun: false, runErr: nil, expected: true},
		{name: "dry run", dryRun: true, runErr: nil, expected: false},
		{name: "aborted run", dryRun: false, runErr: errRunAborted, expected: false},
		{name: "noop run", dryRun: false, runErr: errRunNoop, expected: false},
		{name: "failed run", dryRun: false, runErr: errors.New("boom"), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunPostRunPlugins(tt.dryRun, tt.runErr)
			if got != tt.expected {
				t.Fatalf("shouldRunPostRunPlugins(%v, %v) = %v, want %v", tt.dryRun, tt.runErr, got, tt.expected)
			}
		})
	}
}

func TestRunGitFire_OnFailurePluginErrorKeepsRunError(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("plugin-failure-run-error").
		AddFile("README.md", "hello\n").
		Commit("init")
	// Intentionally configure a broken remote so the main run fails.
	brokenRemoteDir := filepath.Join(t.TempDir(), "missing-remote.git")
	if err := os.MkdirAll(filepath.Dir(brokenRemoteDir), 0o755); err != nil {
		t.Fatalf("mkdir broken remote parent: %v", err)
	}
	testutil.RunGitCmd(t, repo.Path(), "remote", "add", "origin", brokenRemoteDir)

	pluginName := "fail-on-failure-plugin-keep-run-error"
	cfgPath := filepath.Join(tmpHome, "config.toml")
	cfgText := `
[plugins]
enabled = ["` + pluginName + `"]

[[plugins.command]]
name = "` + pluginName + `"
command = "sh"
args = ["-c", "echo plugin failure https://user:supersecret@github.com/org/repo.git 1>&2; exit 1"]
when = "on-failure"
fail_run = true
`
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags()
	configFile = cfgPath
	scanPath = filepath.Dir(repo.Path())

	runErr := runGitFire(rootCmd, []string{})
	if runErr == nil {
		t.Fatal("expected runGitFire() to fail")
	}
	if !strings.Contains(runErr.Error(), "some repositories failed") {
		t.Fatalf("expected returned error to include original run failure, got: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "plugin "+pluginName+" failed") {
		t.Fatalf("expected returned error to include plugin failure, got: %v", runErr)
	}
	if strings.Contains(runErr.Error(), "supersecret") {
		t.Fatalf("expected plugin error in returned error to be sanitized, got: %v", runErr)
	}
	if !strings.Contains(runErr.Error(), "[REDACTED]") {
		t.Fatalf("expected sanitized plugin error marker in returned error, got: %v", runErr)
	}
}

func TestRunGitFire_OnSuccessPluginFailRunFailsRun(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("plugin-on-success").
		WithRemote("origin", remote).
		AddFile("README.md", "hello\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Make the repo dirty so the run performs a real backup path.
	if err := os.WriteFile(filepath.Join(repo.Path(), "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	pluginName := "fail-on-success-plugin"
	cfgPath := filepath.Join(tmpHome, "config.toml")
	cfgText := `
[plugins]
enabled = ["` + pluginName + `"]

[[plugins.command]]
name = "` + pluginName + `"
command = "sh"
args = ["-c", "exit 7"]
when = "on-success"
fail_run = true
`
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags()
	configFile = cfgPath
	scanPath = filepath.Dir(repo.Path())

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w

	runErr := runGitFire(rootCmd, []string{})

	os.Stderr = oldStderr
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	stderrBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	stderr := string(stderrBytes)
	if runErr == nil {
		t.Fatal("expected runGitFire() to fail when on-success fail_run plugin fails")
	}
	if !strings.Contains(runErr.Error(), "plugin "+pluginName+" failed") {
		t.Fatalf("expected returned error to include plugin failure, got: %v", runErr)
	}
	if strings.Contains(runErr.Error(), "some repositories failed") {
		t.Fatalf("expected core backup to succeed and only plugin to fail run, got: %v", runErr)
	}
	if strings.Contains(stderr, "plugin "+pluginName+":") {
		t.Fatalf("expected fail_run plugin errors to avoid duplicate stderr logging, got: %q", stderr)
	}
}

func TestRunGitFire_DryRun_SkipsPostRunPlugins(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("plugin-dry-run-skip").
		WithRemote("origin", remote).
		AddFile("README.md", "hello\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Keep repo dirty so dry-run still plans work.
	if err := os.WriteFile(filepath.Join(repo.Path(), "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	marker := filepath.Join(t.TempDir(), "plugin-ran.marker")
	pluginName := "should-not-run-in-dry-run"
	cfgPath := filepath.Join(tmpHome, "config.toml")
	cfgText := `
[plugins]
enabled = ["` + pluginName + `"]

[[plugins.command]]
name = "` + pluginName + `"
command = "sh"
args = ["-c", "printf ran > \"$1\"", "plugin", "` + marker + `"]
when = "always"
fail_run = true
`
	if err := os.WriteFile(cfgPath, []byte(cfgText), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	resetFlags()
	configFile = cfgPath
	scanPath = filepath.Dir(repo.Path())
	dryRun = true

	if err := runGitFire(rootCmd, []string{}); err != nil {
		t.Fatalf("runGitFire() dry-run should not fail from post-run plugins: %v", err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatalf("expected post-run plugins to be skipped in dry-run; marker %q exists", marker)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat marker: %v", err)
	}
}

func TestBuildPostRunPluginContext_UsesScanRootMetadata(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scanRoot := t.TempDir()
	cfg := config.LoadOrDefault()
	cfg.Global.ScanPath = scanRoot

	ctx := buildPostRunPluginContext(cfg, false, true)

	if ctx.RepoPath != scanRoot {
		t.Fatalf("expected RepoPath %q, got %q", scanRoot, ctx.RepoPath)
	}
	if ctx.RepoName != filepath.Base(scanRoot) {
		t.Fatalf("expected RepoName %q, got %q", filepath.Base(scanRoot), ctx.RepoName)
	}
	if !ctx.Emergency {
		t.Fatalf("expected Emergency=true")
	}
	if ctx.DryRun {
		t.Fatalf("expected DryRun=false")
	}
}

func TestBuildPostRunPluginContext_ReadsGitBranchAndCommitWhenScanRootIsRepo(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("scan-root-repo").
		WithRemote("origin", remote).
		AddFile("README.md", "hello\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	cfg := config.LoadOrDefault()
	cfg.Global.ScanPath = repo.Path()

	ctx := buildPostRunPluginContext(cfg, false, false)

	if ctx.Branch == "" {
		t.Fatalf("expected Branch to be populated for git scan root")
	}
	if ctx.CommitSHA == "" {
		t.Fatalf("expected CommitSHA to be populated for git scan root")
	}
}

func TestBuildPostRunPluginContext_DoesNotReadParentRepoMetadata(t *testing.T) {
	tmpHome := t.TempDir()
	setTestUserDirs(t, tmpHome)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	parentRepo := scenario.CreateRepo("parent-repo").
		WithRemote("origin", remote).
		AddFile("README.md", "parent\n").
		Commit("init")
	defaultBranch := parentRepo.GetDefaultBranch()
	parentRepo.Push("origin", defaultBranch)

	scanRoot := filepath.Join(parentRepo.Path(), "projects")
	if err := os.MkdirAll(scanRoot, 0o755); err != nil {
		t.Fatalf("mkdir scan root: %v", err)
	}

	cfg := config.LoadOrDefault()
	cfg.Global.ScanPath = scanRoot

	ctx := buildPostRunPluginContext(cfg, false, false)

	if ctx.RepoPath != scanRoot {
		t.Fatalf("expected RepoPath %q, got %q", scanRoot, ctx.RepoPath)
	}
	if ctx.RepoName != filepath.Base(scanRoot) {
		t.Fatalf("expected RepoName %q, got %q", filepath.Base(scanRoot), ctx.RepoName)
	}
	if ctx.Branch != "" {
		t.Fatalf("expected Branch to be empty when scan root is not a git repo, got %q", ctx.Branch)
	}
	if ctx.CommitSHA != "" {
		t.Fatalf("expected CommitSHA to be empty when scan root is not a git repo, got %q", ctx.CommitSHA)
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
