package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/executor"
	"github.com/git-fire/git-fire/internal/registry"
	"github.com/git-fire/git-fire/internal/usb"
	"github.com/git-fire/git-harness/git"
	"github.com/git-fire/git-harness/safety"
)

func runUSB(cfg *config.Config, reg *registry.Registry, regPath string, opts git.ScanOptions, targets []string) error {
	if fireMode {
		return fmt.Errorf("--fire is not yet supported with --usb")
	}

	targetCfg := make(map[string]*usb.VolumeConfig, len(targets))
	normalizedTargets := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, raw := range targets {
		abs, err := filepath.Abs(raw)
		if err != nil {
			return fmt.Errorf("invalid usb target %q: %w", raw, err)
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		vol, err := usb.EnsureVolumeConfig(abs, usb.EnsureOptions{
			DefaultStrategy: cfg.USB.Strategy,
			CreateIfMissing: usbInit || cfg.USB.CreateOnFirst,
		})
		if err != nil {
			return err
		}
		targetCfg[abs] = vol
		normalizedTargets = append(normalizedTargets, abs)
	}

	if opts.DisableScan {
		fmt.Printf("🔥 USB mode: loading %d known repositories from registry (scan disabled)\n", len(opts.KnownPaths))
	} else {
		fmt.Printf("🔥 USB mode: loading %d known repositories and scanning for new ones...\n", len(opts.KnownPaths))
	}
	fmt.Println()

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
	repos = active

	if len(repos) == 0 {
		fmt.Println("No git repositories found.")
		return errRunNoop
	}

	overrides := make(map[string]usb.RepoOverride, len(repos))
	for _, repo := range repos {
		absPath, absErr := filepath.Abs(repo.Path)
		if absErr != nil {
			continue
		}
		if entry := reg.FindByPath(absPath); entry != nil {
			overrides[repo.Path] = usb.RepoOverride{
				Strategy:   entry.USBStrategy,
				RepoPath:   entry.USBRepoPath,
				SyncPolicy: entry.USBSyncPolicy,
			}
		}
	}

	plans := usb.BuildPlans(repos, normalizedTargets, targetCfg, overrides, usb.PlanOptions{
		AutoCommit: cfg.Global.AutoCommitDirty,
	})

	fmt.Printf("✓ Found %d repositories\n", len(repos))
	fmt.Printf("✓ USB targets: %d\n", len(normalizedTargets))
	for _, t := range normalizedTargets {
		fmt.Printf("  • %s\n", t)
	}
	fmt.Println()

	if dryRun {
		fmt.Println("USB Dry Run Plan:")
		for _, plan := range plans {
			fmt.Printf("  • %s\n", plan.Repo.Name)
			for _, action := range plan.Actions {
				if action.Type != usb.ActionSync {
					continue
				}
				fmt.Printf("      -> %s\n", action.Destination)
			}
		}
		fmt.Println("\n🔥 Fire Drill Complete - No changes were made")
		return errRunNoop
	}

	releases := make([]func(), 0, len(normalizedTargets))
	defer func() {
		for _, release := range releases {
			if release != nil {
				release()
			}
		}
	}()
	manifests := make(map[string]*usb.Manifest, len(normalizedTargets))
	for _, target := range normalizedTargets {
		release, lockErr := usb.AcquireTargetLock(target, 24*time.Hour)
		if lockErr != nil {
			return lockErr
		}
		releases = append(releases, release)
		m, loadErr := usb.LoadManifest(target)
		if loadErr != nil {
			return fmt.Errorf("failed loading manifest for %s: %w", target, loadErr)
		}
		manifests[target] = m
	}

	type repoFailure struct {
		repo   string
		target string
		err    error
	}
	jobs := make(chan usb.RepoPlan, len(plans))
	failures := make([]repoFailure, 0)
	var failuresMu sync.Mutex
	var manifestMu sync.Mutex
	var wg sync.WaitGroup

	recordFailure := func(repoName, target string, err error) {
		failuresMu.Lock()
		defer failuresMu.Unlock()
		failures = append(failures, repoFailure{repo: repoName, target: target, err: err})
	}

	workerCount := cfg.USB.Workers
	if workerCount <= 0 {
		workerCount = 1
	}
	targetWorkers := cfg.USB.TargetWorkers
	if targetWorkers <= 0 {
		targetWorkers = 1
	}
	targetSem := make(chan struct{}, targetWorkers)

	worker := func() {
		defer wg.Done()
		for plan := range jobs {
			repo := plan.Repo
			fmt.Printf("➡️  %s\n", repo.Name)
			for _, action := range plan.Actions {
				switch action.Type {
				case usb.ActionAutoCommit:
					if err := executor.CheckSecrets(repo.Path, cfg.Global.BlockOnSecrets); err != nil {
						recordFailure(repo.Name, "", err)
						goto nextRepo
					}
					_, err := git.AutoCommitDirtyWithStrategy(repo.Path, git.CommitOptions{
						Message:          fmt.Sprintf("git-fire emergency backup - %s", time.Now().Format("2006-01-02 15:04:05")),
						UseDualBranch:    true,
						ReturnToOriginal: true,
					})
					if err != nil {
						recordFailure(repo.Name, "", err)
						goto nextRepo
					}
				case usb.ActionSync:
					targetSem <- struct{}{}
					m := manifests[action.TargetRoot]
					if usbResume {
						manifestMu.Lock()
						prev, ok := m.Results[repo.Path]
						manifestMu.Unlock()
						if ok && prev.Success && prev.Destination == action.Destination {
							<-targetSem
							continue
						}
					}
					reposRoot := usb.TargetReposRoot(action.TargetRoot, targetCfg[action.TargetRoot])
					if err := os.MkdirAll(reposRoot, 0o700); err != nil {
						recordFailure(repo.Name, action.TargetRoot, err)
						recordManifestOutcome(&manifestMu, m, repo, action.Destination, err)
						<-targetSem
						continue
					}
					var syncErr error
					switch action.Strategy {
					case usb.StrategyMirror:
						syncErr = usb.SyncMirrorRepo(repo.Path, action.Destination)
					case usb.StrategyClone:
						syncErr = usb.SyncCloneRepo(repo.Path, action.Destination)
					default:
						syncErr = fmt.Errorf("unsupported usb strategy %q", action.Strategy)
					}
					if syncErr != nil {
						recordFailure(repo.Name, action.TargetRoot, syncErr)
						recordManifestOutcome(&manifestMu, m, repo, action.Destination, syncErr)
						<-targetSem
						continue
					}
					if usbVerify {
						if verifyErr := verifyUSBDestination(action.Destination, action.Strategy); verifyErr != nil {
							recordFailure(repo.Name, action.TargetRoot, verifyErr)
							recordManifestOutcome(&manifestMu, m, repo, action.Destination, verifyErr)
							<-targetSem
							continue
						}
					}
					recordManifestOutcome(&manifestMu, m, repo, action.Destination, nil)
					<-targetSem
				}
			}
		nextRepo:
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	for _, plan := range plans {
		jobs <- plan
	}
	close(jobs)
	wg.Wait()

	for _, target := range normalizedTargets {
		if err := usb.SaveManifest(target, manifests[target]); err != nil {
			recordFailure("manifest", target, err)
		}
		if shouldPruneUSBTarget(target, plans, cfg.USB.SyncPolicy) {
			_ = pruneUSBTarget(target, usb.TargetReposRoot(target, targetCfg[target]), plans)
		}
	}

	saveRegistry(reg, regPath)

	if len(failures) > 0 {
		fmt.Println("\n⚠️  USB mode completed with failures:")
		for _, f := range failures {
			if f.target == "" {
				fmt.Printf("  • %s: %s\n", f.repo, safety.SanitizeText(f.err.Error()))
				continue
			}
			fmt.Printf("  • %s -> %s: %s\n", f.repo, f.target, safety.SanitizeText(f.err.Error()))
		}
		return fmt.Errorf("some repositories failed in usb mode")
	}

	fmt.Printf("\n✓ USB mode complete. Mirrored %d repositories to %d target(s).\n", len(repos), len(normalizedTargets))
	return nil
}

func resolveUSBTargets(cfg *config.Config, flagTargets []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	appendTarget := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}

	for _, target := range flagTargets {
		appendTarget(target)
	}
	for _, target := range cfg.USB.Targets {
		if !target.Enabled {
			continue
		}
		appendTarget(target.Path)
	}
	return out
}

func recordManifestOutcome(mu *sync.Mutex, m *usb.Manifest, repo git.Repository, destination string, err error) {
	mu.Lock()
	defer mu.Unlock()
	if m.Results == nil {
		m.Results = map[string]usb.RepoOutcome{}
	}
	outcome := usb.RepoOutcome{
		RepoPath:    repo.Path,
		RepoName:    repo.Name,
		Destination: destination,
		Success:     err == nil,
		UpdatedAt:   time.Now().UTC(),
	}
	if err != nil {
		outcome.Error = safety.SanitizeText(err.Error())
	}
	m.Results[repo.Path] = outcome
}

func pruneUSBTarget(targetRoot, reposRoot string, plans []usb.RepoPlan) error {
	want := make(map[string]struct{}, len(plans))
	for _, plan := range plans {
		for _, action := range plan.Actions {
			if action.Type != usb.ActionSync || action.TargetRoot != targetRoot {
				continue
			}
			want[action.Destination] = struct{}{}
		}
	}

	entries, err := os.ReadDir(reposRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		full := filepath.Join(reposRoot, entry.Name())
		if !strings.HasSuffix(entry.Name(), ".git") {
			if _, statErr := os.Stat(filepath.Join(full, ".git")); statErr != nil {
				continue
			}
		}
		if _, ok := want[full]; ok {
			continue
		}
		_ = os.RemoveAll(full)
	}
	return nil
}

func shouldPruneUSBTarget(targetRoot string, plans []usb.RepoPlan, defaultPolicy string) bool {
	for _, plan := range plans {
		for _, action := range plan.Actions {
			if action.Type != usb.ActionSync || action.TargetRoot != targetRoot {
				continue
			}
			policy := strings.TrimSpace(action.SyncPolicy)
			if policy == "" {
				policy = defaultPolicy
			}
			if policy == "prune" {
				return true
			}
		}
	}
	return false
}

func verifyUSBDestination(destination, strategy string) error {
	switch strategy {
	case usb.StrategyMirror:
		if _, err := os.Stat(filepath.Join(destination, "HEAD")); err != nil {
			return fmt.Errorf("verify failed for mirror destination %s: %w", destination, err)
		}
	case usb.StrategyClone:
		if _, err := os.Stat(filepath.Join(destination, ".git")); err != nil {
			return fmt.Errorf("verify failed for clone destination %s: %w", destination, err)
		}
	default:
		return fmt.Errorf("verify failed: unsupported strategy %s", strategy)
	}
	return nil
}
