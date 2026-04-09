package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
	"github.com/git-fire/git-fire/internal/safety"
)

var (
	rainPath   string
	rainNoScan bool
	rainRisky  bool
)

var rainCmd = &cobra.Command{
	Use:     "rain",
	Aliases: []string{"hydrate", "hydrant"},
	Short:   "Experimental reverse sync from remotes",
	Long: `Rain ("hydrate"/"hydrant" aliases) updates local branches from their
upstream remote-tracking refs.

Safety-first defaults:
  - never rewrites local-only commits
  - fast-forwards current branch only when merge can be done safely
  - updates non-checked-out branch refs directly

Risky mode (config: global.rain_risky_mode, flag: --risky) allows destructive
realignment of local-only commits after creating git-fire-rain-backup-* refs.`,
	RunE: runRain,
}

func init() {
	rainCmd.Flags().StringVar(&rainPath, "path", "", "Path to scan for repositories (default: config global.scan_path)")
	rainCmd.Flags().BoolVar(&rainNoScan, "no-scan", false, "Skip filesystem scan; hydrate only known registry repos")
	rainCmd.Flags().BoolVar(&rainRisky, "risky", false, "Allow destructive local branch realignment after creating backup refs")
	rootCmd.AddCommand(rainCmd)
}

func runRain(_ *cobra.Command, _ []string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH: please install git before using git-fire")
	}

	cfg, err := config.LoadWithOptions(config.LoadOptions{ConfigFile: configFile})
	if err != nil {
		return fmt.Errorf("failed to load config: %s", safety.SanitizeText(err.Error()))
	}
	if rainPath != "" {
		cfg.Global.ScanPath = rainPath
	}
	if rainNoScan {
		cfg.Global.DisableScan = true
	}
	riskyMode := cfg.Global.RainRiskyMode || rainRisky

	reg := &registry.Registry{}
	regPath := ""
	if p, pathErr := registry.DefaultRegistryPath(); pathErr != nil {
		fmt.Fprintf(os.Stderr, "warning: registry disabled: %v\n", pathErr)
	} else if unlockErr := maybeOfferRegistryUnlock(p); unlockErr != nil {
		return unlockErr
	} else if loaded, loadErr := registry.Load(p); loadErr != nil {
		fmt.Fprintf(os.Stderr, "warning: ignoring unreadable registry %s: %v\n", p, loadErr)
	} else {
		regPath = p
		reg = loaded
	}

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

	opts := git.DefaultScanOptions()
	opts.RootPath = cfg.Global.ScanPath
	opts.Exclude = cfg.Global.ScanExclude
	opts.MaxDepth = cfg.Global.ScanDepth
	opts.Workers = cfg.Global.ScanWorkers
	opts.KnownPaths = buildKnownPaths(reg, cfg.Global.RescanSubmodules)
	opts.DisableScan = cfg.Global.DisableScan

	if opts.DisableScan {
		fmt.Println("⚠️  Rain scanning disabled: hydrating known registry repositories only")
	}
	repos, err := git.ScanRepositories(opts)
	if err != nil {
		return fmt.Errorf("repository scan failed: %w", err)
	}

	now := time.Now()
	defaultMode := git.ParseMode(cfg.Global.DefaultMode)
	for i, repo := range repos {
		repos[i], _ = upsertRepoIntoRegistry(reg, repo, now, defaultMode)
	}
	saveRegistry(reg, regPath)

	active := make([]git.Repository, 0, len(repos))
	for _, repo := range repos {
		absPath, absErr := filepath.Abs(repo.Path)
		if absErr != nil {
			active = append(active, repo)
			continue
		}
		entry := reg.FindByPath(absPath)
		if entry != nil && entry.Status == registry.StatusIgnored {
			continue
		}
		active = append(active, repo)
	}
	if len(active) == 0 {
		fmt.Println("No git repositories found.")
		return nil
	}

	fmt.Println("🌧️  Git Fire Rain (experimental)")
	if riskyMode {
		fmt.Println("⚠️  Risky mode enabled: local-only commits may be realigned after backup branch creation")
	} else {
		fmt.Println("✓ Safe mode: local-only commits are preserved")
	}
	fmt.Println()

	totalUpdated := 0
	totalSkipped := 0
	totalFailed := 0
	repoFailures := 0

	for _, repo := range active {
		fmt.Printf("Repo: %s\n", repo.Name)
		res, rainErr := git.RainRepository(repo.Path, git.RainOptions{RiskyMode: riskyMode})
		if rainErr != nil {
			repoFailures++
			fmt.Printf("  ❌ failed: %s\n\n", safety.SanitizeText(rainErr.Error()))
			continue
		}
		if len(res.Branches) == 0 {
			fmt.Println("  ⊘ no local branches found")
			fmt.Println()
			continue
		}

		for _, br := range res.Branches {
			symbol := "⊘"
			switch br.Outcome {
			case git.RainOutcomeUpdated:
				symbol = "✓"
			case git.RainOutcomeUpdatedRisky:
				symbol = "⚠️"
			case git.RainOutcomeFailed:
				symbol = "❌"
			}
			line := fmt.Sprintf("  %s %s", symbol, br.Branch)
			if br.Upstream != "" {
				line += " <- " + br.Upstream
			}
			line += ": " + br.Outcome
			if br.Message != "" {
				line += " (" + safety.SanitizeText(strings.TrimSpace(br.Message)) + ")"
			}
			if br.BackupBranch != "" {
				line += " backup=" + br.BackupBranch
			}
			fmt.Println(line)
		}
		fmt.Println()

		totalUpdated += res.Updated
		totalSkipped += res.Skipped
		totalFailed += res.Failed
		if res.Failed > 0 {
			repoFailures++
		}
	}

	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("🌧️  Rain complete")
	fmt.Printf("Updated branches: %d\n", totalUpdated)
	fmt.Printf("Skipped branches: %d\n", totalSkipped)
	fmt.Printf("Failed branches: %d\n", totalFailed)

	if repoFailures > 0 || totalFailed > 0 {
		return fmt.Errorf("rain completed with failures")
	}
	return nil
}
