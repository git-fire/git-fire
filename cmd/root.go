package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-fire/git-fire/internal/auth"
	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
	"github.com/git-fire/git-fire/internal/safety"
	"github.com/git-fire/git-fire/internal/ui"
)

// Version is set at build time via -ldflags "-X github.com/git-fire/git-fire/cmd.Version=vX.Y.Z"
var Version = "dev"

var (
	// Flags
	dryRun     bool
	fireDrill  bool
	fireMode   bool
	scanPath   string
	skipCommit bool
	noScan     bool
	initConfig bool
	forceInit  bool
	backupTo   string
	configFile string
	showStatus bool
)

var rootCmd = &cobra.Command{
	Use:   "git-fire",
	Short: "Emergency git backup tool",
	Long: `Git Fire - Emergency Git Backup Tool

In case of fire:
  1. git-fire
  2. Leave building

Git Fire will scan for repositories, auto-commit changes,
and push everything to your remotes.`,
	RunE: runGitFire,
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = Version
	rootCmd.SilenceUsage = true
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVar(&fireDrill, "fire-drill", false, "Alias for --dry-run")
	rootCmd.Flags().BoolVar(&fireMode, "fire", false, "Fire mode: TUI repo selector, skips confirmation prompt")
	rootCmd.Flags().StringVar(&scanPath, "path", ".", "Path to scan for repositories")
	rootCmd.Flags().BoolVar(&skipCommit, "skip-auto-commit", false, "Skip auto-committing dirty repos")
	rootCmd.Flags().BoolVar(&noScan, "no-scan", false, "Skip filesystem scan; back up only known (registry) repos this run")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "Generate example configuration file")
	rootCmd.Flags().BoolVar(&forceInit, "force", false, "Overwrite existing config without prompting (use with --init)")
	rootCmd.Flags().StringVar(&backupTo, "backup-to", "", "Backup to specified remote URL (planned v0.2; not yet implemented)")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Use an explicit config file path (default: ~/.config/git-fire/config.toml)")
	rootCmd.Flags().BoolVar(&showStatus, "status", false, "Show SSH and repo status")
}

func runGitFire(cmd *cobra.Command, args []string) error {
	// Handle --init flag
	if initConfig {
		return handleInit()
	}

	// Handle --status flag
	if showStatus {
		return handleStatus()
	}

	// Fire drill is same as dry run.
	if fireDrill {
		dryRun = true
	}
	if fireMode && dryRun {
		return fmt.Errorf("--fire and --dry-run cannot be used together")
	}

	if backupTo != "" {
		return fmt.Errorf("--backup-to is not implemented yet; use configured remotes instead")
	}

	// Verify git is available before doing anything else
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: please install git before using git-fire")
	}

	// Load configuration
	cfg, cfgErr := config.LoadWithOptions(config.LoadOptions{ConfigFile: configFile})
	if cfgErr != nil {
		return fmt.Errorf("failed to load config: %s", safety.SanitizeText(cfgErr.Error()))
	}

	// Override config with flags
	if skipCommit {
		cfg.Global.AutoCommitDirty = false
	}
	if scanPath != "." {
		cfg.Global.ScanPath = scanPath
	}
	if noScan {
		cfg.Global.DisableScan = true
	}

	// Show security notice
	if !dryRun {
		fmt.Println(safety.SecurityNotice())
	}

	// Load persistent registry (best-effort: fall back to empty in-memory registry on failure)
	reg := &registry.Registry{}
	regPath := ""
	if p, err := registry.DefaultRegistryPath(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: registry disabled: %v\n", err)
	} else if loaded, err := registry.Load(p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring unreadable registry %s: %v\n", p, err)
	} else {
		regPath = p
		reg = loaded
	}

	// Validate known paths and mark missing ones
	for i, entry := range reg.Repos {
		if entry.Status == registry.StatusIgnored {
			continue
		}
		if _, statErr := os.Stat(entry.Path); statErr != nil {
			if os.IsNotExist(statErr) && reg.Repos[i].Status != registry.StatusMissing {
				reg.Repos[i].Status = registry.StatusMissing
			}
			continue
		}
		if entry.Status == registry.StatusMissing || entry.Status == "" {
			reg.Repos[i].Status = registry.StatusActive
		}
	}

	// Build KnownPaths for the scanner (active, missing, and empty status).
	knownPaths := buildKnownPaths(reg, cfg.Global.RescanSubmodules)

	// Build scan options
	opts := git.DefaultScanOptions()
	opts.RootPath = cfg.Global.ScanPath
	opts.Exclude = cfg.Global.ScanExclude
	opts.Workers = cfg.Global.ScanWorkers
	opts.KnownPaths = knownPaths
	opts.DisableScan = cfg.Global.DisableScan

	fmt.Println("🔥 Git Fire - Emergency Backup Tool")
	if cfg.Global.DisableScan {
		if noScan {
			fmt.Println("⚠️  Scanning Disabled (this run only)")
		} else {
			fmt.Println("⚠️  Scanning Disabled")
		}
	}
	fmt.Println()

	// Routing:
	//   --fire         → streaming TUI (repos appear as discovered)
	//   --dry-run      → batch collect then plan summary (no changes made)
	//   default        → streaming backup pipeline
	if fireMode {
		return runFireStream(cfg, reg, regPath, opts)
	}
	if dryRun {
		return runBatch(cfg, reg, regPath, opts)
	}
	return runStream(cfg, reg, regPath, opts)
}

// runBatch is used for --fire (TUI) and --dry-run. It collects the full repo
// list before proceeding, which is necessary for the interactive selector and
// for showing a complete plan summary before any changes are made.
func runBatch(cfg *config.Config, reg *registry.Registry, regPath string, opts git.ScanOptions) error {
	var (
		repos     []git.Repository
		sshStatus *auth.SSHStatus
		scanErr   error
		sshErr    error
		wg        sync.WaitGroup
	)

	if opts.DisableScan {
		fmt.Printf("⏳ Loading %d known repositories from registry (filesystem scan disabled)...\n", len(opts.KnownPaths))
	} else {
		fmt.Printf("⏳ Loading %d known repositories and scanning for new ones...\n", len(opts.KnownPaths))
	}
	fmt.Println()

	wg.Add(2)
	go func() {
		defer wg.Done()
		repos, scanErr = git.ScanRepositories(opts)
	}()
	go func() {
		defer wg.Done()
		sshStatus, sshErr = auth.GetSSHStatus()
	}()
	wg.Wait()

	if scanErr != nil {
		return fmt.Errorf("repository scan failed: %w", scanErr)
	}
	if sshErr != nil {
		return fmt.Errorf("SSH status check failed: %w", sshErr)
	}

	// Upsert all discovered repos into the registry.
	now := time.Now()
	defaultMode := git.ParseMode(cfg.Global.DefaultMode)
	for i, repo := range repos {
		repos[i], _ = upsertRepoIntoRegistry(reg, repo, now, defaultMode)
	}
	saveRegistry(reg, regPath)

	// Exclude ignored repos from backup.
	activeRepos := make([]git.Repository, 0, len(repos))
	for _, repo := range repos {
		absPath, err := filepath.Abs(repo.Path)
		if err != nil {
			activeRepos = append(activeRepos, repo)
			continue
		}
		entry := reg.FindByPath(absPath)
		if entry != nil && entry.Status == registry.StatusIgnored {
			continue
		}
		activeRepos = append(activeRepos, repo)
	}
	repos = activeRepos

	fmt.Printf("✓ Found %d repositories\n", len(repos))
	fmt.Printf("✓ SSH Status: %d keys available", len(sshStatus.AvailableKeys))
	if sshStatus.Agent.Running {
		fmt.Printf(" (%d loaded in agent)", len(sshStatus.Agent.Keys))
	}
	fmt.Println()
	fmt.Println()

	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	// Repo selection: auto-select all (dry-run path only — fireMode uses streaming).
	for i := range repos {
		repos[i].Selected = true
	}

	// Show selected repos
	fmt.Println("Selected repositories:")
	for _, repo := range repos {
		if repo.Selected {
			status := ""
			if repo.IsDirty {
				status = " (dirty)"
			}
			fmt.Printf("  • %s%s\n", repo.Name, status)
		}
	}
	fmt.Println()

	// Build and validate plan
	planner := executor.NewPlanner(cfg)
	plan, err := planner.BuildPlan(repos, dryRun)
	if err != nil {
		return fmt.Errorf("failed to build plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	fmt.Println(plan.Summary())
	fmt.Println()

	if dryRun {
		fmt.Println("🔥 Fire Drill Complete - No changes were made")
		return nil
	}

	// Setup logging
	logger, err := executor.NewLogger(executor.DefaultLogDir())
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Close()

	// Execute plan
	fmt.Println()
	fmt.Println("🔥 Pushing repositories...")
	fmt.Println()

	runner := executor.NewRunner(cfg)
	defer runner.Close()

	go func() {
		for progress := range runner.ProgressChan() {
			fmt.Printf("[%d/%d] %s: %s (%s)\n",
				progress.CurrentRepo, progress.TotalRepos,
				progress.RepoName, progress.Action, progress.Status)
			if progress.Error != nil {
				fmt.Printf("  ❌ Error: %s\n", safety.SanitizeText(progress.Error.Error()))
			}
		}
	}()

	result, err := runner.Execute(plan)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	logger.LogResult(result)
	printResult(result, logger.LogPath())

	if result.Failed > 0 {
		fmt.Println("\n⚠️  Some repositories failed to push. Check the log for details.")
		return fmt.Errorf("some repositories failed")
	}
	return nil
}

// runFireStream runs the --fire TUI mode with progressive repo discovery.
// The scan runs in the background; repos stream into the TUI as they are found
// rather than waiting for the full scan to complete before showing the selector.
func runFireStream(cfg *config.Config, reg *registry.Registry, regPath string, opts git.ScanOptions) error {
	// Cancellable context so TUI can abort the scan on quit.
	ctx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()
	opts.Ctx = ctx

	folderProgress := make(chan string, 32)
	opts.FolderProgress = folderProgress

	scanChan := make(chan git.Repository, opts.Workers)
	// tuiRepoChan receives upserted, filtered repos for the TUI.
	tuiRepoChan := make(chan git.Repository, opts.Workers)

	var scanErr error
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanErr = git.ScanRepositoriesStream(opts, scanChan)
	}()

	now := time.Now()
	defaultMode := git.ParseMode(cfg.Global.DefaultMode)
	go func() {
		defer close(tuiRepoChan)
		for repo := range scanChan {
			repo, include := upsertRepoIntoRegistry(reg, repo, now, defaultMode)
			if include {
				repo.Selected = true
				tuiRepoChan <- repo
			}
		}
		saveRegistry(reg, regPath)
	}()

	selected, err := ui.RunRepoSelectorStream(
		tuiRepoChan,
		folderProgress,
		cfg.Global.DisableScan,
		noScan,
		cfg,
		config.DefaultConfigPath(),
		reg,
		regPath,
	)
	// Drain both channels BEFORE cancelling so neither the upsert goroutine
	// (tuiRepoChan) nor the scanner's walk goroutine (folderProgress) can block
	// on a send after the TUI exits. Only then cancel the scan and wait.
	go func() { for range tuiRepoChan {} }()
	go func() { for range folderProgress {} }()
	cancelScan()
	<-scanDone

	if err != nil {
		if errors.Is(err, ui.ErrCancelled) {
			fmt.Println("Aborted.")
			return nil
		}
		return fmt.Errorf("repo selection failed: %w", err)
	}
	if len(selected) == 0 {
		fmt.Println("No repositories selected.")
		return nil
	}

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "warning: scan error: %s\n", safety.SanitizeText(scanErr.Error()))
	}

	// Show selected repos
	fmt.Println("Selected repositories:")
	for _, repo := range selected {
		status := ""
		if repo.IsDirty {
			status = " (dirty)"
		}
		fmt.Printf("  • %s%s\n", repo.Name, status)
	}
	fmt.Println()

	// Build and execute plan
	planner := executor.NewPlanner(cfg)
	plan, err := planner.BuildPlan(selected, false)
	if err != nil {
		return fmt.Errorf("failed to build plan: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	fmt.Println(plan.Summary())
	fmt.Println()

	logger, err := executor.NewLogger(executor.DefaultLogDir())
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Close()

	fmt.Println("🔥 Pushing repositories...")
	fmt.Println()

	runner := executor.NewRunner(cfg)
	defer runner.Close()

	go func() {
		for progress := range runner.ProgressChan() {
			fmt.Printf("[%d/%d] %s: %s (%s)\n",
				progress.CurrentRepo, progress.TotalRepos,
				progress.RepoName, progress.Action, progress.Status)
			if progress.Error != nil {
				fmt.Printf("  ❌ Error: %s\n", safety.SanitizeText(progress.Error.Error()))
			}
		}
	}()

	result, err := runner.Execute(plan)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	logger.LogResult(result)
	printResult(result, logger.LogPath())

	if result.Failed > 0 {
		fmt.Println("\n⚠️  Some repositories failed to push. Check the log for details.")
		return fmt.Errorf("some repositories failed")
	}
	return nil
}

// runStream is the default live-run path. It pipelines scan → registry upsert →
// backup so that pushing starts as soon as the first repo is discovered, without
// waiting for the full scan to complete. Workers block when the queue is
// temporarily empty and drain naturally when scanning is done.
//
// After all backups complete, if the filesystem scan is still running the user
// is prompted to wait or press Ctrl+C / Enter to abort the scan.
func runStream(cfg *config.Config, reg *registry.Registry, regPath string, opts git.ScanOptions) error {
	if opts.DisableScan {
		fmt.Println("🔥 Backing up known repositories (filesystem scan disabled)...")
	} else {
		fmt.Println("🔥 Scanning and backing up repositories...")
	}
	fmt.Println()

	// Start SSH check in the background — we'll show the result at the end.
	sshChan := make(chan *auth.SSHStatus, 1)
	sshErrChan := make(chan error, 1)
	go func() {
		status, err := auth.GetSSHStatus()
		if err != nil {
			sshErrChan <- err
		} else {
			sshChan <- status
		}
	}()

	// Cancellable context so we can abort the scan after backups finish.
	ctx, cancelScan := context.WithCancel(context.Background())
	defer cancelScan()
	opts.Ctx = ctx

	// Buffer folder-progress so the walk never blocks on a slow consumer.
	folderProgress := make(chan string, 32)
	opts.FolderProgress = folderProgress

	scanChan := make(chan git.Repository, opts.Workers)
	repoChan := make(chan git.Repository, opts.Workers)

	// totalFound is updated by the upsert goroutine and read (with atomic load)
	// by the progress printer. Using int64 for atomic ops.
	var totalFound int64
	var scanErr error
	scanDone := make(chan struct{})

	// Goroutine 1: scan → scanChan (closed when scan finishes or ctx cancelled)
	go func() {
		defer close(scanDone)
		scanErr = git.ScanRepositoriesStream(opts, scanChan)
	}()

	// Drain folder-progress in the background (TUI uses it; CLI discards it).
	go func() {
		for range folderProgress {
		}
	}()

	// Goroutine 2: upsert + filter → repoChan (closed when scanChan drains)
	now := time.Now()
	defaultMode := git.ParseMode(cfg.Global.DefaultMode)
	go func() {
		defer close(repoChan)
		for repo := range scanChan {
			repo, include := upsertRepoIntoRegistry(reg, repo, now, defaultMode)
			atomic.AddInt64(&totalFound, 1)
			if include {
				repo.Selected = true
				repoChan <- repo
			}
		}
		saveRegistry(reg, regPath)
	}()

	// Setup logger, planner, runner
	logger, err := executor.NewLogger(executor.DefaultLogDir())
	if err != nil {
		return fmt.Errorf("failed to setup logger: %w", err)
	}
	defer logger.Close()

	planner := executor.NewPlanner(cfg)
	runner := executor.NewRunner(cfg)
	defer runner.Close()

	// Progress display goroutine
	go func() {
		for progress := range runner.ProgressChan() {
			total := int(atomic.LoadInt64(&totalFound))
			totalStr := "?"
			if total > 0 {
				totalStr = fmt.Sprintf("%d", total)
			}
			fmt.Printf("[%d/%s] %s: %s (%s)\n",
				progress.CurrentRepo, totalStr,
				progress.RepoName, progress.Action, progress.Status)
			if progress.Error != nil {
				fmt.Printf("  ❌ Error: %s\n", safety.SanitizeText(progress.Error.Error()))
			}
		}
	}()

	// ExecuteStream blocks until repoChan is closed (scan + upsert both done).
	// totalFound is updated atomically by the upsert goroutine as repos arrive.
	result, execErr := runner.ExecuteStream(repoChan, planner, false, &totalFound)

	// If the scan is still running after all backups are done, prompt the user.
	select {
	case <-scanDone:
		// Scan already finished — nothing to do.
	default:
		// Scan is still walking the tree.
		fmt.Println()
		fmt.Println("✅ All backups complete. Scan still running.")
		fmt.Println("   Press Enter to wait for scan to finish, or Ctrl+C to stop scanning.")

		// Cancel on either Ctrl+C or Enter.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		inputDone := make(chan struct{})
		go func() {
			defer close(inputDone)
			r := bufio.NewReader(os.Stdin)
			_, _ = r.ReadString('\n')
		}()

		select {
		case <-sigCh:
			fmt.Println("\nAborting scan...")
			cancelScan()
		case <-inputDone:
			// User pressed Enter — let scan finish normally; just wait below.
		case <-scanDone:
			// Scan finished on its own while we were waiting.
		}
		signal.Stop(sigCh)
		<-scanDone // always wait for the goroutine to exit cleanly
	}

	// Check scan error (non-fatal: report and continue to show results)
	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "warning: scan error: %s\n", safety.SanitizeText(scanErr.Error()))
	}

	logger.LogResult(result)

	// Show SSH status
	select {
	case sshStatus := <-sshChan:
		fmt.Printf("\n✓ SSH: %d keys available", len(sshStatus.AvailableKeys))
		if sshStatus.Agent.Running {
			fmt.Printf(" (%d loaded in agent)", len(sshStatus.Agent.Keys))
		}
		fmt.Println()
	case sshErr := <-sshErrChan:
		fmt.Fprintf(os.Stderr, "warning: SSH check failed: %s\n", safety.SanitizeText(sshErr.Error()))
	}

	printResult(result, logger.LogPath())

	if execErr != nil {
		return fmt.Errorf("execution failed: %w", execErr)
	}
	if result.Failed > 0 {
		fmt.Println("\n⚠️  Some repositories failed to push. Check the log for details.")
		return fmt.Errorf("some repositories failed")
	}
	return nil
}

// upsertRepoIntoRegistry adds or updates the registry entry for repo and
// returns the (possibly mode-updated) repo and whether it should be backed up
// (false only for StatusIgnored entries).
func upsertRepoIntoRegistry(reg *registry.Registry, repo git.Repository, now time.Time, defaultMode git.RepoMode) (git.Repository, bool) {
	absPath, err := filepath.Abs(repo.Path)
	if err != nil {
		// Can't resolve path — include repo to be safe (never silently drop backups).
		return repo, true
	}
	var modeStr string
	var ignored bool
	found := reg.UpdateByPath(absPath, func(e *registry.RegistryEntry) {
		modeStr = e.Mode
		e.LastSeen = now
		if e.Status == registry.StatusIgnored {
			ignored = true
			return
		}
		e.Status = registry.StatusActive
	})
	if found {
		if modeStr != "" {
			repo.Mode = git.ParseMode(modeStr)
		} else {
			repo.Mode = defaultMode
		}
		return repo, !ignored
	}
	// New discovery — register it immediately (opt-out model).
	repo.Mode = defaultMode
	reg.Upsert(registry.RegistryEntry{
		Path:     absPath,
		Name:     repo.Name,
		Status:   registry.StatusActive,
		Mode:     repo.Mode.String(),
		AddedAt:  now,
		LastSeen: now,
	})
	return repo, true
}

// saveRegistry persists the registry and logs a warning on failure.
func saveRegistry(reg *registry.Registry, regPath string) {
	if regPath == "" {
		return
	}
	if err := registry.Save(reg, regPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save registry: %v\n", err)
	}
}

// printResult prints the final summary after a run.
func printResult(result *executor.ExecutionResult, logPath string) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔥 Git Fire Complete!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n")
	fmt.Printf("✓ Successfully pushed: %d repos\n", result.Success)
	fmt.Printf("❌ Failed: %d repos\n", result.Failed)
	fmt.Printf("⊘ Skipped: %d repos\n", result.Skipped)
	fmt.Printf("⏱  Duration: %s\n", result.Duration)
	fmt.Printf("\n")
	fmt.Printf("📝 Log file: %s\n", logPath)
}

func handleInit() error {
	configPath := config.DefaultConfigPath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		if !forceInit {
			// In non-interactive environments (CI, piped stdin) fmt.Scanln would
			// hang or read garbage. Detect a non-terminal and fail fast so the
			// caller knows to use --force.
			if stat, err := os.Stdin.Stat(); err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
				return fmt.Errorf("config already exists at %s; use --force to overwrite non-interactively", configPath)
			}
			fmt.Printf("Configuration file already exists: %s\n", configPath)
			fmt.Print("Overwrite? [y/N]: ")
			var response string
			fmt.Scanln(&response)

			if response != "y" && response != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Write example config
	if err := config.WriteExampleConfig(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✓ Configuration file created: %s\n", configPath)
	fmt.Println("\nEdit this file to customize git-fire behavior.")

	return nil
}

func handleStatus() error {
	// Show SSH status
	sshStatus, err := auth.GetSSHStatus()
	if err != nil {
		return fmt.Errorf("failed to get SSH status: %w", err)
	}

	fmt.Println(sshStatus.Summary())

	// Show repositories
	cfg, err := config.LoadWithOptions(config.LoadOptions{ConfigFile: configFile})
	if err != nil {
		return fmt.Errorf("failed to load config: %s", safety.SanitizeText(err.Error()))
	}
	if scanPath != "." {
		cfg.Global.ScanPath = scanPath
	}
	if noScan {
		cfg.Global.DisableScan = true
	}

	opts := git.DefaultScanOptions()
	opts.RootPath = cfg.Global.ScanPath
	opts.DisableScan = cfg.Global.DisableScan

	reg := &registry.Registry{}
	if p, err := registry.DefaultRegistryPath(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: registry disabled: %v\n", err)
	} else if loaded, err := registry.Load(p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring unreadable registry %s: %v\n", p, err)
	} else {
		reg = loaded
	}
	opts.KnownPaths = buildKnownPaths(reg, cfg.Global.RescanSubmodules)

	if cfg.Global.DisableScan {
		if noScan {
			fmt.Println("⚠️  Scanning Disabled (this run only)")
		} else {
			fmt.Println("⚠️  Scanning Disabled")
		}
		fmt.Println()
	}

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		return fmt.Errorf("repository scan failed: %w", err)
	}

	fmt.Printf("\nRepositories found: %d\n", len(repos))
	dirtyCount := 0
	for _, repo := range repos {
		if repo.IsDirty {
			dirtyCount++
		}
	}
	fmt.Printf("Dirty repositories: %d\n", dirtyCount)

	return nil
}
