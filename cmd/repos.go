package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "Manage the persistent repository registry",
	Long: `Manage the persistent repository registry stored at ~/.config/git-fire/repos.toml
(same directory as config.toml).

The registry tracks all git repositories that git-fire has discovered, so that
future runs load them instantly without re-scanning the filesystem.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return reposList(cmd, args)
	},
}

var reposListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked repositories",
	RunE:  reposList,
}

var reposScanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a directory for git repos and add them to the registry",
	Long: `Scan a directory for git repos and add any newly discovered ones to the
registry. Defaults to the scan_path from your configuration file if no path is
given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: reposScan,
}

var reposRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Permanently remove a repository from the registry",
	Args:  cobra.ExactArgs(1),
	RunE:  reposRemove,
}

var reposIgnoreCmd = &cobra.Command{
	Use:   "ignore <path>",
	Short: "Mark a repository as ignored (excluded from backup runs)",
	Args:  cobra.ExactArgs(1),
	RunE:  reposIgnore,
}

var reposUnignoreCmd = &cobra.Command{
	Use:   "unignore <path>",
	Short: "Un-ignore a repository (restore to active status)",
	Args:  cobra.ExactArgs(1),
	RunE:  reposUnignore,
}

func init() {
	reposCmd.AddCommand(reposListCmd)
	reposCmd.AddCommand(reposScanCmd)
	reposCmd.AddCommand(reposRemoveCmd)
	reposCmd.AddCommand(reposIgnoreCmd)
	reposCmd.AddCommand(reposUnignoreCmd)
	rootCmd.AddCommand(reposCmd)
}

// loadRegistry is a shared helper used by all repos subcommands.
func loadRegistry() (*registry.Registry, string, error) {
	regPath, err := registry.DefaultRegistryPath()
	if err != nil {
		return nil, "", fmt.Errorf("registry path: %w", err)
	}
	reg, err := registry.Load(regPath)
	if err != nil {
		return nil, "", fmt.Errorf("loading registry: %w", err)
	}
	return reg, regPath, nil
}

func reposList(_ *cobra.Command, _ []string) error {
	reg, _, err := loadRegistry()
	if err != nil {
		return err
	}

	if len(reg.Repos) == 0 {
		fmt.Println("No repositories tracked yet. Run 'git-fire repos scan' to discover repos.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tNAME\tMODE\tLAST SEEN\tPATH")
	fmt.Fprintln(w, "------\t----\t----\t---------\t----")

	for _, e := range reg.Repos {
		status := statusLabel(e.Status)
		mode := e.Mode
		if mode == "" {
			mode = "—"
		}
		lastSeen := "never"
		if !e.LastSeen.IsZero() {
			lastSeen = e.LastSeen.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", status, e.Name, mode, lastSeen, e.Path)
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: %d repositories\n", len(reg.Repos))
	return nil
}

func reposScan(_ *cobra.Command, args []string) error {
	cfg := config.LoadOrDefault()

	scanRoot := cfg.Global.ScanPath
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	fmt.Printf("Scanning %s for git repositories...\n", absRoot)

	reg, regPath, err := loadRegistry()
	if err != nil {
		return err
	}

	opts := git.DefaultScanOptions()
	opts.RootPath = absRoot
	opts.Exclude = cfg.Global.ScanExclude
	opts.Workers = cfg.Global.ScanWorkers
	// Pass known paths so the scanner skips re-walking them
	opts.KnownPaths = buildKnownPaths(reg, cfg.Global.RescanSubmodules)

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	now := time.Now()
	added := 0
	for _, repo := range repos {
		absPath, absErr := filepath.Abs(repo.Path)
		if absErr != nil {
			continue
		}
		if entry := reg.FindByPath(absPath); entry != nil {
			if entry.Status != registry.StatusIgnored {
				entry.Status = registry.StatusActive
				entry.Name = repo.Name
				entry.LastSeen = now
				if entry.Mode == "" {
					entry.Mode = repo.Mode.String()
				}
			}
		} else {
			reg.Upsert(registry.RegistryEntry{
				Path:     absPath,
				Name:     repo.Name,
				Status:   registry.StatusActive,
				Mode:     repo.Mode.String(),
				AddedAt:  now,
				LastSeen: now,
			})
			added++
			fmt.Printf("  + %s (%s)\n", repo.Name, absPath)
		}
	}

	if err := registry.Save(reg, regPath); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	if added == 0 {
		fmt.Println("No new repositories found.")
	} else {
		fmt.Printf("\nAdded %d new repository(ies) to the registry.\n", added)
	}
	return nil
}

func reposRemove(_ *cobra.Command, args []string) error {
	reg, regPath, err := loadRegistry()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !reg.Remove(absPath) {
		return fmt.Errorf("repository not found in registry: %s", absPath)
	}

	if err := registry.Save(reg, regPath); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Removed %s from registry.\n", absPath)
	return nil
}

func reposIgnore(_ *cobra.Command, args []string) error {
	return setRepoStatus(args[0], registry.StatusIgnored, "ignored")
}

func reposUnignore(_ *cobra.Command, args []string) error {
	return setRepoStatus(args[0], registry.StatusActive, "active")
}

func setRepoStatus(rawPath, status, label string) error {
	reg, regPath, err := loadRegistry()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(rawPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if !reg.SetStatus(absPath, status) {
		return fmt.Errorf("repository not found in registry: %s", absPath)
	}

	if err := registry.Save(reg, regPath); err != nil {
		return fmt.Errorf("saving registry: %w", err)
	}

	fmt.Printf("Repository marked as %s: %s\n", label, absPath)
	return nil
}

// buildKnownPaths constructs the KnownPaths map for the scanner. Active,
// missing, and empty-status entries are included so repos persist across
// runs from different working directories; ignored entries are excluded.
// Paths that no longer exist are skipped at analysis time (see scanner).
func buildKnownPaths(reg *registry.Registry, globalRescan bool) map[string]bool {
	m := make(map[string]bool, len(reg.Repos))
	for _, e := range reg.Repos {
		if e.Status == registry.StatusIgnored {
			continue
		}
		if e.Status != registry.StatusActive && e.Status != registry.StatusMissing && e.Status != "" {
			continue
		}
		abs, err := filepath.Abs(e.Path)
		if err != nil {
			continue
		}
		rescan := globalRescan
		if e.RescanSubmodules != nil {
			rescan = *e.RescanSubmodules
		}
		m[abs] = rescan
	}
	return m
}

// statusLabel returns a short coloured-ish label for a registry status.
func statusLabel(s string) string {
	switch s {
	case registry.StatusActive, "":
		return "active "
	case registry.StatusMissing:
		return "MISSING"
	case registry.StatusIgnored:
		return "ignored"
	default:
		return s
	}
}
