package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/TBRX103/git-fire/internal/auth"
	"github.com/TBRX103/git-fire/internal/config"
	"github.com/TBRX103/git-fire/internal/executor"
	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/registry"
	"github.com/TBRX103/git-fire/internal/safety"
	"github.com/TBRX103/git-fire/internal/ui"
)

var (
	// Flags
	dryRun       bool
	fireDrill    bool
	fireMode     bool
	scanPath     string
	skipCommit   bool
	initConfig   bool
	backupTo     string
	showStatus   bool
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
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	rootCmd.Flags().BoolVar(&fireDrill, "fire-drill", false, "Alias for --dry-run")
	rootCmd.Flags().BoolVar(&fireMode, "fire", false, "Fire mode: TUI repo selector, skips confirmation prompt")
	rootCmd.Flags().StringVar(&scanPath, "path", ".", "Path to scan for repositories")
	rootCmd.Flags().BoolVar(&skipCommit, "skip-auto-commit", false, "Skip auto-committing dirty repos")
	rootCmd.Flags().BoolVar(&initConfig, "init", false, "Generate example configuration file")
	rootCmd.Flags().StringVar(&backupTo, "backup-to", "", "Backup to specified remote URL")
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

	// Load configuration
	cfg := config.LoadOrDefault()

	// Override config with flags
	if skipCommit {
		cfg.Global.AutoCommitDirty = false
	}
	if scanPath != "." {
		cfg.Global.ScanPath = scanPath
	}

	// Fire drill is same as dry run
	if fireDrill {
		dryRun = true
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
	registryUpdated := false
	for i, entry := range reg.Repos {
		if entry.Status == registry.StatusIgnored {
			continue
		}
		if _, statErr := os.Stat(entry.Path); statErr != nil {
			if os.IsNotExist(statErr) && reg.Repos[i].Status != registry.StatusMissing {
				reg.Repos[i].Status = registry.StatusMissing
				registryUpdated = true
			}
			continue
		}
		if entry.Status == registry.StatusMissing {
			reg.Repos[i].Status = registry.StatusActive
			registryUpdated = true
		}
	}

	// Build KnownPaths for the scanner: active entries only.
	knownPaths := buildKnownPaths(reg, cfg.Global.RescanSubmodules)

	// Background scanning - start immediately
	fmt.Println("🔥 Git Fire - Emergency Backup Tool")
	fmt.Println()
	fmt.Printf("⏳ Loading %d known repositories and scanning for new ones...\n", len(knownPaths))
	fmt.Println()

	var (
		repos     []git.Repository
		sshStatus *auth.SSHStatus
		scanErr   error
		sshErr    error
		wg        sync.WaitGroup
	)

	// Start background scans
	wg.Add(2)

	// Scan repositories in background
	go func() {
		defer wg.Done()
		opts := git.DefaultScanOptions()
		opts.RootPath = cfg.Global.ScanPath
		opts.Exclude = cfg.Global.ScanExclude
		opts.Workers = cfg.Global.ScanWorkers
		opts.KnownPaths = knownPaths

		repos, scanErr = git.ScanRepositories(opts)
	}()

	// Check SSH status in background
	go func() {
		defer wg.Done()
		sshStatus, sshErr = auth.GetSSHStatus()
	}()

	// Wait for both scans to complete
	wg.Wait()

	// Check for errors
	if scanErr != nil {
		return fmt.Errorf("repository scan failed: %w", scanErr)
	}
	if sshErr != nil {
		return fmt.Errorf("SSH status check failed: %w", sshErr)
	}

	// Upsert all discovered repos into the registry, preserving per-repo
	// mode from the registry when available.
	now := time.Now()
	for i, repo := range repos {
		absPath, absErr := filepath.Abs(repo.Path)
		if absErr != nil {
			continue
		}
		existing := reg.FindByPath(absPath)
		if existing != nil {
			// Restore persisted mode onto the in-memory repo
			if existing.Mode != "" {
				repos[i].Mode = git.ParseMode(existing.Mode)
			}
			existing.LastSeen = now
			if existing.Status != registry.StatusIgnored {
				existing.Status = registry.StatusActive
			}
		} else {
			// New discovery — add to registry
			reg.Upsert(registry.RegistryEntry{
				Path:     absPath,
				Name:     repo.Name,
				Status:   registry.StatusActive,
				Mode:     repo.Mode.String(),
				AddedAt:  now,
				LastSeen: now,
			})
			registryUpdated = true
		}
	}

	// Persist registry if anything changed
	if regPath != "" && (registryUpdated || len(repos) > 0) {
		_ = registry.Save(reg, regPath) // best-effort; don't fail the run
	}

	// Exclude ignored repos from backup
	activeRepos := make([]git.Repository, 0, len(repos))
	for _, repo := range repos {
		absPath, err := filepath.Abs(repo.Path)
		if err != nil {
			// Can't resolve path; include repo to be safe (don't silently drop backups)
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

	// Show scan results
	fmt.Printf("✓ Found %d repositories\n", len(repos))
	fmt.Printf("✓ SSH Status: %d keys available", len(sshStatus.AvailableKeys))
	if sshStatus.Agent.Running {
		fmt.Printf(" (%d loaded in agent)", len(sshStatus.Agent.Keys))
	}
	fmt.Println()
	fmt.Println()

	// If no repos found, exit
	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	// Repo selection: TUI when --fire is set, otherwise auto-select all
	if fireMode {
		// Seed defaults so the TUI can show current state and the user can override
		for i := range repos {
			repos[i].Selected = true
		}
		selected, err := ui.RunRepoSelector(repos, reg, regPath)
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
		repos = selected
	} else {
		for i := range repos {
			repos[i].Selected = true
		}
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

	// Build execution plan
	planner := executor.NewPlanner(cfg)
	plan, err := planner.BuildPlan(repos, dryRun)
	if err != nil {
		return fmt.Errorf("failed to build plan: %w", err)
	}

	// Validate plan
	if err := plan.Validate(); err != nil {
		return fmt.Errorf("invalid plan: %w", err)
	}

	// Show plan summary
	fmt.Println(plan.Summary())
	fmt.Println()

	// In dry-run mode, just show plan and exit
	if dryRun {
		fmt.Println("🔥 Fire Drill Complete - No changes were made")
		return nil
	}

	// Confirm execution (skip in fire mode — no time to type y in an emergency)
	if !fireMode {
		fmt.Print("Proceed with pushing? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
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

	// Start progress monitoring in background
	go func() {
		for progress := range runner.ProgressChan() {
			fmt.Printf("[%d/%d] %s: %s (%s)\n",
				progress.CurrentRepo, progress.TotalRepos,
				progress.RepoName, progress.Action, progress.Status)

			if progress.Error != nil {
				fmt.Printf("  ❌ Error: %v\n", progress.Error)
			}
		}
	}()

	result, err := runner.Execute(plan)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// Log results
	logger.LogResult(result)

	// Show results
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🔥 Git Fire Complete!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Printf("\n")
	fmt.Printf("✓ Successfully pushed: %d repos\n", result.Success)
	fmt.Printf("❌ Failed: %d repos\n", result.Failed)
	fmt.Printf("⊘ Skipped: %d repos\n", result.Skipped)
	fmt.Printf("⏱  Duration: %s\n", result.Duration)
	fmt.Printf("\n")
	fmt.Printf("📝 Log file: %s\n", logger.LogPath())

	if result.Failed > 0 {
		fmt.Println("\n⚠️  Some repositories failed to push. Check the log for details.")
		return fmt.Errorf("some repositories failed")
	}

	return nil
}

func handleInit() error {
	configPath := config.DefaultConfigPath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Configuration file already exists: %s\n", configPath)
		fmt.Print("Overwrite? [y/N]: ")
		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
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
	// Merge registry known paths so --status counts match normal runs.
	cfg := config.LoadOrDefault()

	opts := git.DefaultScanOptions()
	opts.RootPath = cfg.Global.ScanPath

	reg := &registry.Registry{}
	if p, err := registry.DefaultRegistryPath(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: registry disabled: %v\n", err)
	} else if loaded, err := registry.Load(p); err != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring unreadable registry %s: %v\n", p, err)
	} else {
		reg = loaded
	}
	opts.KnownPaths = buildKnownPaths(reg, cfg.Global.RescanSubmodules)

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
