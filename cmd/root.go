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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"

	"github.com/git-fire/git-fire/internal/auth"
	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/plugins"
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

var errRunAborted = errors.New("run aborted")
var errRunNoop = errors.New("run completed with no backup actions")

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
	rootCmd.Flags().StringVar(&backupTo, "backup-to", "", "Backup to specified remote URL (not yet implemented)")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Use an explicit config file path (default: user config dir, e.g. ~/.config/git-fire/config.toml)")
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

	// --fire-drill is an alias for --dry-run.
	if fireDrill {
		dryRun = true
	}
	if fireMode && dryRun {
		return fmt.Errorf("--fire and --dry-run cannot be used together")
	}

	var cfg *config.Config
	failRun := func(err error) error {
		if err != nil {
			if FlavorQuotesEnabled(cfg) {
				printFailedRunEmberMessage()
			}
		}
		return err
	}

	fmt.Println("🔥 Git Fire - Emergency Backup Tool")

	if backupTo != "" {
		// TODO(v0.2): implement backup-to remote URL
		return failRun(fmt.Errorf("--backup-to is not yet implemented (planned for v0.2)"))
	}

	// Verify git is available before doing anything else
	if _, err := exec.LookPath("git"); err != nil {
		return failRun(fmt.Errorf("git not found in PATH: please install git before using git-fire"))
	}

	// Load configuration
	var cfgErr error
	cfg, cfgErr = config.LoadWithOptions(config.LoadOptions{ConfigFile: configFile})
	if cfgErr != nil {
		return failRun(fmt.Errorf("failed to load config: %s", safety.SanitizeText(cfgErr.Error())))
	}

	// Flavor quotes (see ui.show_startup_quote / Settings → Show flavor quotes).
	// --fire: quote is printed after the TUI exits (runFireStream) so it is visible
	// on the normal screen buffer.
	if !fireMode && FlavorQuotesEnabled(cfg) {
		printStartupFireQuote()
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

	// Load plugins from config (non-fatal: warn and continue on failure)
	if loadErr := plugins.LoadFromConfig(cfg); loadErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load plugins from config: %s\n", safety.SanitizeText(loadErr.Error()))
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
	opts.MaxDepth = cfg.Global.ScanDepth
	opts.Workers = cfg.Global.ScanWorkers
	opts.KnownPaths = knownPaths
	opts.DisableScan = cfg.Global.DisableScan

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
	//   --dry-run      → batch collect, plan summary, then dry-run execute (no git mutations; secret warnings)
	//   default        → streaming backup pipeline
	var runErr error
	if fireMode {
		runErr = runFireStream(cfg, reg, regPath, opts)
	} else if dryRun {
		runErr = runBatch(cfg, reg, regPath, opts)
	} else {
		runErr = runStream(cfg, reg, regPath, opts)
	}

	// Fire post-run plugins (non-fatal, skipped on dry-run, user abort, and no-op runs)
	if shouldRunPostRunPlugins(dryRun, runErr) {
		pluginCtx := plugins.Context{
			Timestamp: time.Now(),
			DryRun:    dryRun,
			Emergency: fireMode,
			Logger:    &cmdPluginLogger{},
		}
		enabledPlugins, enabledErr := plugins.GetEnabledPlugins(cfg)
		if enabledErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to resolve enabled plugins: %s\n", safety.SanitizeText(enabledErr.Error()))
		} else {
			runPlugins := func(trigger plugins.Trigger) {
				for _, p := range plugins.FilterPluginsByTrigger(enabledPlugins, trigger) {
					if pErr := p.Execute(pluginCtx); pErr != nil {
						fmt.Fprintf(os.Stderr, "plugin %s: %s\n", p.Name(), safety.SanitizeText(pErr.Error()))
					}
				}
			}

			// after-push is the default trigger for command plugins.
			runPlugins(plugins.TriggerAfterPush)

			if runErr != nil && !errors.Is(runErr, errRunNoop) {
				runPlugins(plugins.TriggerOnFailure)
			} else {
				runPlugins(plugins.TriggerOnSuccess)
			}

			runPlugins(plugins.TriggerAlways)
		}
	}

	if errors.Is(runErr, errRunAborted) {
		if FlavorQuotesEnabled(cfg) {
			printFailedRunEmberMessage()
		}
		return nil
	}
	if errors.Is(runErr, errRunNoop) {
		return nil
	}
	if runErr != nil {
		if FlavorQuotesEnabled(cfg) {
			printFailedRunEmberMessage()
		}
		return runErr
	}

	if FlavorQuotesEnabled(cfg) {
		printExtinguishWaterMessage()
	}
	return nil
}

// runBatch is used for --fire (TUI) and --dry-run. It collects the full repo
// list before proceeding, which is necessary for the interactive selector and
// for showing a complete plan summary. --dry-run then runs the executor dry-run
// path (e.g. secret scans) without mutating repositories.
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
		if sshStatus.Agent.KeysKnown {
			fmt.Printf(" (%d loaded in agent)", len(sshStatus.Agent.Keys))
		} else if sshStatus.Agent.Error != "" {
			fmt.Printf(" (agent key status unknown: %s)", safety.SanitizeText(sshStatus.Agent.Error))
		} else {
			fmt.Printf(" (agent key status unknown)")
		}
	}
	fmt.Println()
	fmt.Println()

	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return errRunNoop
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
		runner := executor.NewRunner(cfg)
		defer runner.Close()
		if _, err := runner.Execute(plan); err != nil {
			return fmt.Errorf("dry run failed: %w", err)
		}
		fmt.Println("🔥 Fire Drill Complete - No changes were made")
		return errRunNoop
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

	userCfgDir, _ := config.UserGitFireDir()
	selected, err := ui.RunRepoSelectorStream(
		tuiRepoChan,
		folderProgress,
		cfg.Global.DisableScan,
		noScan,
		cfg,
		filepath.Join(userCfgDir, "config.toml"),
		reg,
		regPath,
	)
	// Drain both channels BEFORE cancelling so neither the upsert goroutine
	// (tuiRepoChan) nor the scanner's walk goroutine (folderProgress) can block
	// on a send after the TUI exits. Only then cancel the scan and wait.
	go func() {
		for range tuiRepoChan {
		}
	}()
	go func() {
		for range folderProgress {
		}
	}()
	cancelScan()
	<-scanDone

	if err != nil {
		if errors.Is(err, ui.ErrCancelled) {
			fmt.Println("Aborted.")
			return errRunAborted
		}
		return fmt.Errorf("repo selection failed: %w", err)
	}
	if len(selected) == 0 {
		fmt.Println("No repositories selected.")
		return errRunNoop
	}

	if scanErr != nil {
		fmt.Fprintf(os.Stderr, "warning: scan error: %s\n", safety.SanitizeText(scanErr.Error()))
	}

	// Back on the normal screen buffer: surface the flavor line here so it is
	// visible after the alt-screen TUI (pre-run stdout is easy to miss).
	if FlavorQuotesEnabled(cfg) {
		printStartupFireQuote()
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

	// Folder progress: TUI consumes paths live; default stream prints periodic
	// updates so long walks are not silent.
	var lastFolder atomic.Pointer[string]
	if !opts.DisableScan {
		go func() {
			for p := range folderProgress {
				pp := p
				lastFolder.Store(&pp)
			}
		}()
		go func() {
			tick := time.NewTicker(2 * time.Second)
			defer tick.Stop()
			const scanPrefix = "   🔍 Scanning… "
			for {
				select {
				case <-scanDone:
					return
				case <-tick.C:
					ptr := lastFolder.Load()
					if ptr != nil && *ptr != "" {
						maxPathLen := scanProgressPathMaxLen(scanPrefix)
						fmt.Printf("%s%s\n", scanPrefix, truncateScanProgressPath(*ptr, maxPathLen))
					}
				}
			}
		}()
	} else {
		go func() {
			for range folderProgress {
			}
		}()
	}

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
			if sshStatus.Agent.KeysKnown {
				fmt.Printf(" (%d loaded in agent)", len(sshStatus.Agent.Keys))
			} else if sshStatus.Agent.Error != "" {
				fmt.Printf(" (agent key status unknown: %s)", safety.SanitizeText(sshStatus.Agent.Error))
			} else {
				fmt.Printf(" (agent key status unknown)")
			}
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
		repo.IsNewRegistryEntry = false
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
		repo.IsNewRegistryEntry = false
		if modeStr != "" {
			repo.Mode = git.ParseMode(modeStr)
		} else {
			repo.Mode = defaultMode
		}
		return repo, !ignored
	}
	// New discovery — register it immediately (opt-out model).
	repo.Mode = defaultMode
	repo.IsNewRegistryEntry = true
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

// truncateScanProgressPath shortens a filesystem path for one-line CLI output.
func truncateScanProgressPath(path string, maxLen int) string {
	if maxLen <= 0 || runewidth.StringWidth(path) <= maxLen {
		return path
	}
	ellipsis := "..."
	ellipsisWidth := runewidth.StringWidth(ellipsis)
	if maxLen <= ellipsisWidth {
		return runewidth.Truncate(path, maxLen, "")
	}
	remaining := maxLen - ellipsisWidth
	runes := []rune(path)
	start := len(runes)
	width := 0
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if width+rw > remaining {
			break
		}
		width += rw
		start = i
	}
	return ellipsis + string(runes[start:])
}

func scanProgressPathMaxLen(prefix string) int {
	const (
		fallback = 72
		minLen   = 8
	)
	cols, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || cols <= 0 {
		return fallback
	}
	dynamic := cols - runewidth.StringWidth(prefix)
	if dynamic < minLen {
		return minLen
	}
	if dynamic > fallback {
		return fallback
	}
	return dynamic
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
	if configFile != "" {
		configPath = configFile
	}

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
	opts.MaxDepth = cfg.Global.ScanDepth
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

// cmdPluginLogger satisfies plugins.Logger for post-run plugin execution.
type cmdPluginLogger struct{}

func (l *cmdPluginLogger) Info(msg string)    { fmt.Println(" ", safety.SanitizeText(msg)) }
func (l *cmdPluginLogger) Success(msg string) { fmt.Println(" ", safety.SanitizeText(msg)) }
func (l *cmdPluginLogger) Error(msg string, err error) {
	safeMsg := safety.SanitizeText(msg)
	if err == nil {
		fmt.Fprintf(os.Stderr, "  %s\n", safeMsg)
		return
	}
	fmt.Fprintf(os.Stderr, "  %s: %s\n", safeMsg, safety.SanitizeText(err.Error()))
}
func (l *cmdPluginLogger) Debug(_ string) {}

func shouldRunPostRunPlugins(isDryRun bool, runErr error) bool {
	return !isDryRun && !errors.Is(runErr, errRunAborted) && !errors.Is(runErr, errRunNoop)
}
