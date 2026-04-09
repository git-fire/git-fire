package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testutil "github.com/git-fire/git-testkit"
)

func TestRainRepository_FastForwardsCurrentBranchPreservingUnstaged(t *testing.T) {
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-current").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	cloneBase := t.TempDir()
	peerDir := filepath.Join(cloneBase, "peer")
	testutil.RunGitCmd(t, cloneBase, "clone", remote.Path(), peerDir)
	testutil.RunGitCmd(t, peerDir, "config", "user.email", "test@example.com")
	testutil.RunGitCmd(t, peerDir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(peerDir, "remote-only.txt"), []byte("from remote\n"), 0o644); err != nil {
		t.Fatalf("write peer file: %v", err)
	}
	testutil.RunGitCmd(t, peerDir, "add", "remote-only.txt")
	testutil.RunGitCmd(t, peerDir, "commit", "-m", "remote advance")
	testutil.RunGitCmd(t, peerDir, "push", "origin", defaultBranch)

	// Local unstaged work on an unrelated file should be preserved by ff merge.
	if err := os.WriteFile(filepath.Join(repo.Path(), "wip.txt"), []byte("local wip\n"), 0o644); err != nil {
		t.Fatalf("write local wip: %v", err)
	}

	result, err := RainRepository(repo.Path(), RainOptions{})
	if err != nil {
		t.Fatalf("RainRepository() error = %v", err)
	}
	if result.Updated == 0 {
		t.Fatalf("expected at least one updated branch, got %+v", result)
	}
	if _, statErr := os.Stat(filepath.Join(repo.Path(), "remote-only.txt")); statErr != nil {
		t.Fatal("expected remote commit to be fast-forwarded into current branch")
	}
	if _, statErr := os.Stat(filepath.Join(repo.Path(), "wip.txt")); statErr != nil {
		t.Fatal("expected unstaged local file to be preserved")
	}
}

func TestRainRepository_SafeModeSkipsLocalAhead(t *testing.T) {
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-safe-ahead").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)
	repo.AddFile("local-only.txt", "ahead\n").Commit("local ahead")
	localSHABefore := testutil.GetCurrentSHA(t, repo.Path())

	result, err := RainRepository(repo.Path(), RainOptions{RiskyMode: false})
	if err != nil {
		t.Fatalf("RainRepository() error = %v", err)
	}
	localSHANow := testutil.GetCurrentSHA(t, repo.Path())
	if localSHANow != localSHABefore {
		t.Fatalf("safe mode should not rewrite local-ahead branch (before=%s after=%s)", localSHABefore, localSHANow)
	}
	if !containsRainOutcome(result, RainOutcomeSkippedLocalAhead) {
		t.Fatalf("expected skipped-local-ahead outcome, got %+v", result.Branches)
	}
}

func TestRainRepository_RiskyModeResetsCurrentBranchWhenClean(t *testing.T) {
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-risky-reset").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)
	remoteSHA := testutil.GetCurrentSHA(t, repo.Path())

	repo.AddFile("local-only.txt", "ahead\n").Commit("local ahead")
	localAheadSHA := testutil.GetCurrentSHA(t, repo.Path())
	if localAheadSHA == remoteSHA {
		t.Fatal("test setup error: local ahead SHA should differ from remote")
	}

	result, err := RainRepository(repo.Path(), RainOptions{RiskyMode: true})
	if err != nil {
		t.Fatalf("RainRepository() error = %v", err)
	}
	currentSHA := testutil.GetCurrentSHA(t, repo.Path())
	if currentSHA != remoteSHA {
		t.Fatalf("expected risky mode reset current branch to upstream SHA=%s, got %s", remoteSHA, currentSHA)
	}
	if !containsRainOutcome(result, RainOutcomeUpdatedRisky) {
		t.Fatalf("expected updated-risky outcome, got %+v", result.Branches)
	}
	if !hasRainBackupBranch(t, repo.Path()) {
		t.Fatal("expected risky mode to create a git-fire-rain-backup-* branch")
	}
}

func TestRainRepository_RiskyModeSkipsDirtyCurrentBranch(t *testing.T) {
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-risky-dirty").
		WithRemote("origin", remote).
		AddFile("tracked.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	repo.AddFile("local-only.txt", "ahead\n").Commit("local ahead")
	localAheadSHA := testutil.GetCurrentSHA(t, repo.Path())

	// Dirty the current branch (staged + unstaged) to force risky-mode safety skip.
	if err := os.WriteFile(filepath.Join(repo.Path(), "staged.txt"), []byte("staged\n"), 0o644); err != nil {
		t.Fatalf("write staged: %v", err)
	}
	testutil.RunGitCmd(t, repo.Path(), "add", "staged.txt")
	if err := os.WriteFile(filepath.Join(repo.Path(), "unstaged.txt"), []byte("unstaged\n"), 0o644); err != nil {
		t.Fatalf("write unstaged: %v", err)
	}

	result, err := RainRepository(repo.Path(), RainOptions{RiskyMode: true})
	if err != nil {
		t.Fatalf("RainRepository() error = %v", err)
	}
	currentSHA := testutil.GetCurrentSHA(t, repo.Path())
	if currentSHA != localAheadSHA {
		t.Fatalf("dirty current branch should not be rewritten (before=%s after=%s)", localAheadSHA, currentSHA)
	}
	if !containsRainOutcome(result, RainOutcomeSkippedUnsafeDirty) {
		t.Fatalf("expected skipped-unsafe-dirty outcome, got %+v", result.Branches)
	}
}

func TestRainRepository_UpdatesNonCurrentBranchWhileCurrentDirty(t *testing.T) {
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("rain-noncurrent").
		WithRemote("origin", remote).
		AddFile("main.txt", "v1\n").
		Commit("init")
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Create and push a feature branch.
	testutil.RunGitCmd(t, repo.Path(), "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(repo.Path(), "feature.txt"), []byte("feature v1\n"), 0o644); err != nil {
		t.Fatalf("write feature file: %v", err)
	}
	testutil.RunGitCmd(t, repo.Path(), "add", "feature.txt")
	testutil.RunGitCmd(t, repo.Path(), "commit", "-m", "feature init")
	repo.Push("origin", "feature")
	featureBefore := testutil.GetCurrentSHA(t, repo.Path())

	// Advance remote feature branch from a peer clone.
	cloneBase := t.TempDir()
	peerDir := filepath.Join(cloneBase, "peer")
	testutil.RunGitCmd(t, cloneBase, "clone", remote.Path(), peerDir)
	testutil.RunGitCmd(t, peerDir, "config", "user.email", "test@example.com")
	testutil.RunGitCmd(t, peerDir, "config", "user.name", "Test User")
	testutil.RunGitCmd(t, peerDir, "checkout", "feature")
	if err := os.WriteFile(filepath.Join(peerDir, "feature.txt"), []byte("feature v2\n"), 0o644); err != nil {
		t.Fatalf("write peer feature update: %v", err)
	}
	testutil.RunGitCmd(t, peerDir, "add", "feature.txt")
	testutil.RunGitCmd(t, peerDir, "commit", "-m", "feature remote advance")
	testutil.RunGitCmd(t, peerDir, "push", "origin", "feature")

	// Return to main/default branch and make it dirty.
	testutil.RunGitCmd(t, repo.Path(), "checkout", defaultBranch)
	if err := os.WriteFile(filepath.Join(repo.Path(), "wip.txt"), []byte("do not lose me\n"), 0o644); err != nil {
		t.Fatalf("write local wip: %v", err)
	}

	result, err := RainRepository(repo.Path(), RainOptions{})
	if err != nil {
		t.Fatalf("RainRepository() error = %v", err)
	}

	// Feature branch should be updated even though current branch is dirty.
	testutil.RunGitCmd(t, repo.Path(), "checkout", "feature")
	featureAfter := testutil.GetCurrentSHA(t, repo.Path())
	if featureAfter == featureBefore {
		t.Fatalf("expected feature branch to update from remote while current branch is dirty")
	}
	testutil.RunGitCmd(t, repo.Path(), "checkout", defaultBranch)
	if _, statErr := os.Stat(filepath.Join(repo.Path(), "wip.txt")); statErr != nil {
		t.Fatal("expected dirty current-branch worktree file to remain after rain")
	}
	if !containsRainOutcome(result, RainOutcomeUpdated) {
		t.Fatalf("expected at least one updated outcome, got %+v", result.Branches)
	}
}

func containsRainOutcome(result RainResult, outcome string) bool {
	for _, br := range result.Branches {
		if br.Outcome == outcome {
			return true
		}
	}
	return false
}

func hasRainBackupBranch(t *testing.T, repoPath string) bool {
	t.Helper()
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git branch listing failed: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "git-fire-rain-backup-") {
			return true
		}
	}
	return false
}
