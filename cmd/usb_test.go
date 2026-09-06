package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/registry"
	"github.com/git-fire/git-fire/internal/usb"
	"github.com/git-fire/git-harness/git"
	testutil "github.com/git-fire/git-testkit"
)

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
		{Actions: []usb.Action{{Type: usb.ActionSync, TargetRoot: targetRoot, Destination: keepDest, SyncPolicy: "keep"}}},
		{Actions: []usb.Action{{Type: usb.ActionSync, TargetRoot: targetRoot, Destination: pruneDest, SyncPolicy: "prune"}}},
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
		{Actions: []usb.Action{{Type: usb.ActionSync, TargetRoot: targetRoot, Destination: keepCloneDest, SyncPolicy: "prune"}}},
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
	output := captureStdout(t, func() {
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
		t.Fatalf("did not expect .git suffix for override destination, got %q", output)
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
	dup := targetRoot + string(os.PathSeparator)
	if err := runUSB(&cfg, reg, "", opts, []string{targetRoot, dup}); err != nil {
		t.Fatalf("expected duplicate normalized targets to be handled, got: %v", err)
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
