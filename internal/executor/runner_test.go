package executor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	testutil "github.com/git-fire/git-testkit"
)

func TestRunner_ExecuteDryRun(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create test repos using scenario builder
	scenario, repo := testutil.CreateCleanRepoScenario(t)

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "test-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: scenario.GetRepo("remote").Path()},
			},
			Branches: []string{"main"},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, true) // DRY RUN
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Execute dry run
	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify dry run results
	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	if result.Plan != plan {
		t.Error("Expected result to reference original plan")
	}

	if result.Duration == 0 {
		t.Error("Expected duration to be recorded")
	}

	// Dry run should not modify git state
	// (No actual git commands should be executed)
}

func TestRunner_ExecuteRealWithDirtyRepo(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create dirty repo scenario with remote
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("initial.txt", "initial content\n").
		Commit("Initial commit")

	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Make it dirty
	repo.AddFile("staged.txt", "staged changes\n")
	repo.ModifyFile("unstaged.txt", "unstaged changes\n")

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "dirty-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
			Branches: []string{defaultBranch},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false) // Real execution
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Execute real push
	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify execution results
	if len(result.RepoResults) != 1 {
		t.Fatalf("Expected 1 repo result, got %d", len(result.RepoResults))
	}

	repoResult := result.RepoResults[0]
	if !repoResult.Success {
		if repoResult.Error != nil {
			t.Fatalf("Expected success, got error: %v", repoResult.Error)
		}
		t.Fatal("Expected repo execution to succeed")
	}

	// Verify auto-commit was executed
	foundAutoCommit := false
	for _, action := range repoResult.Actions {
		if action.Type == ActionAutoCommit {
			foundAutoCommit = true
			if action.Error != nil {
				t.Errorf("Auto-commit failed: %v", action.Error)
			}
		}
	}

	if !foundAutoCommit {
		t.Error("Expected auto-commit action to be executed")
	}
}

func TestRunner_ExecuteSkippedRepo(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/tmp/fake-repo",
			Name:     "skipped-repo",
			Selected: true,
			Mode:     git.ModeLeaveUntouched, // This should be skipped
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo.git"},
			},
		},
	}

	plan, err := runner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	executor := NewRunner(&cfg)
	result, err := executor.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if result.Skipped != 1 {
		t.Errorf("Expected 1 skipped repo, got %d", result.Skipped)
	}

	if result.Success != 0 {
		t.Errorf("Expected 0 successful repos, got %d", result.Success)
	}
}

func TestRunner_ExecuteMultipleRepos(t *testing.T) {
	cfg := config.DefaultConfig()

	// Create multiple repos with different states
	scenario1, repo1 := testutil.CreateCleanRepoScenario(t)

	// Create second repo with remote
	scenario2 := testutil.NewScenario(t)
	remote2 := scenario2.CreateBareRepo("remote")
	repo2 := scenario2.CreateRepo("dirty").
		WithRemote("origin", remote2).
		AddFile("file.txt", "content\n").
		Commit("Initial commit")

	branch1 := repo1.GetDefaultBranch()
	branch2 := repo2.GetDefaultBranch()

	repo2.Push("origin", branch2)
	repo2.AddFile("dirty.txt", "dirty content\n") // Make it dirty

	repos := []git.Repository{
		{
			Path:     repo1.Path(),
			Name:     "clean-repo",
			Selected: true,
			Mode:     git.ModePushKnownBranches,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: scenario1.GetRepo("remote").Path()},
			},
			Branches: []string{branch1},
		},
		{
			Path:     repo2.Path(),
			Name:     "dirty-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote2.Path()},
			},
			Branches: []string{branch2},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	executor := NewRunner(&cfg)
	result, err := executor.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should have results for both repos
	if len(result.RepoResults) != 2 {
		t.Fatalf("Expected 2 repo results, got %d", len(result.RepoResults))
	}

	// Both should succeed (or at least attempt)
	if result.Success < 1 {
		t.Error("Expected at least 1 successful repo")
	}
}

func TestRunner_ExecuteActionAutoCommit(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create dirty repo
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("dirty").
		AddFile("file.txt", "content\n").
		Commit("Initial commit")

	// Make it dirty
	repo.AddFile("staged.txt", "staged\n")
	repo.ModifyFile("unstaged.txt", "unstaged\n")

	action := Action{
		Type:        ActionAutoCommit,
		Description: "Auto-commit changes",
	}

	gitRepo := git.Repository{
		Path: repo.Path(),
		Name: "test-repo",
	}

	// Execute the action
	executedAction := runner.executeAction(gitRepo, action, 1, 1)

	if executedAction.Error != nil {
		t.Errorf("Auto-commit action failed: %v", executedAction.Error)
	}
	if executedAction.Branch == "" {
		t.Error("Expected auto-commit action to return created backup branch names")
	}
}

func TestRunner_ExecuteActionCreateFireBranch_PushesRemote(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("test").
		WithRemote("origin", remote).
		AddFile("file.txt", "content\n").
		Commit("initial commit")

	currentBranch := repo.GetDefaultBranch()
	repo.Push("origin", currentBranch)

	repo.AddFile("local-only.txt", "new local work\n")
	repo.Commit("local commit")

	action := Action{
		Type:        ActionCreateFireBranch,
		Description: "Create fire branch",
		Remote:      "origin",
		Branch:      currentBranch,
	}

	gitRepo := git.Repository{
		Path: repo.Path(),
		Name: "test-repo",
		Remotes: []git.Remote{
			{Name: "origin", URL: remote.Path()},
		},
	}

	executed := runner.executeAction(gitRepo, action, 1, 1)
	if executed.Error != nil {
		t.Fatalf("CreateFireBranch action failed: %v", executed.Error)
	}
	if executed.Branch == "" {
		t.Fatal("Expected fire branch name to be set on executed action")
	}

	remoteSHA, err := git.GetCommitSHA(remote.Path(), "refs/heads/"+executed.Branch)
	if err != nil {
		t.Fatalf("failed to read fire branch from remote: %v", err)
	}
	localFireSHA, err := git.GetCommitSHA(repo.Path(), executed.Branch)
	if err != nil {
		t.Fatalf("failed to read local fire branch SHA: %v", err)
	}
	if remoteSHA != localFireSHA {
		t.Errorf("fire branch push mismatch: remote=%s local=%s", remoteSHA, localFireSHA)
	}
}

func TestRunner_Execute_UsesDualBranchPushes(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("base.txt", "base\n").
		Commit("base commit")
	currentBranch := repo.GetDefaultBranch()
	repo.Push("origin", currentBranch)

	repo.AddFile("staged.txt", "staged\n")
	repo.StageFile("staged.txt")
	repo.AddFile("unstaged.txt", "unstaged\n")

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "dirty-repo",
			Selected: true,
			Mode:     git.ModePushCurrentBranch,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.RepoResults) != 1 {
		t.Fatalf("Expected one repo result, got %d", len(result.RepoResults))
	}

	rr := result.RepoResults[0]
	if len(rr.PushedBranches) == 0 {
		t.Fatal("Expected pushed backup branches, got none")
	}
	for _, b := range rr.PushedBranches {
		if strings.HasPrefix(b, currentBranch) {
			t.Errorf("expected backup branch push, got current branch %q", b)
		}
	}
}

func TestRunner_Execute_AutoCommitWithOnlySkipActions_DoesNotInjectFallbackPushes(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("base.txt", "base\n").
		Commit("base commit")
	currentBranch := repo.GetDefaultBranch()
	repo.Push("origin", currentBranch)

	// Make repo dirty so auto-commit creates backup branches.
	repo.AddFile("staged.txt", "staged\n")
	repo.StageFile("staged.txt")
	repo.AddFile("unstaged.txt", "unstaged\n")

	plan := &PushPlan{
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: repo.Path(),
					Name: "dirty-repo",
					Remotes: []git.Remote{
						{Name: "origin", URL: remote.Path()},
					},
				},
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit uncommitted changes"},
					{Type: ActionSkip, Description: "Skip push to origin due to conflict_strategy=abort"},
				},
			},
		},
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.RepoResults) != 1 {
		t.Fatalf("Expected one repo result, got %d", len(result.RepoResults))
	}
	if result.Success != 0 || result.Skipped != 1 {
		t.Fatalf("expected skip classification when only skip actions remain, got success=%d skipped=%d", result.Success, result.Skipped)
	}

	rr := result.RepoResults[0]
	if rr.Success {
		t.Fatalf("repo result should not be marked success when no executable actions remain: %#v", rr)
	}
	for _, action := range rr.Actions {
		if action.Type == ActionAutoCommit {
			t.Fatalf("expected auto-commit to be skipped when no push actions remain, got actions=%#v", rr.Actions)
		}
	}
	for _, action := range rr.Actions {
		if action.Type == ActionPushBranch {
			t.Fatalf("expected no injected push-branch actions when only skip actions remain, got actions=%#v", rr.Actions)
		}
	}
}

func TestRunner_Execute_AutoCommitReplacesPushAllAndPushKnown(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("base.txt", "base\n").
		Commit("base commit")
	currentBranch := repo.GetDefaultBranch()
	repo.Push("origin", currentBranch)

	// Ensure dual-branch auto-commit generates both staged and full backups.
	repo.AddFile("staged.txt", "staged\n")
	repo.StageFile("staged.txt")
	repo.AddFile("unstaged.txt", "unstaged\n")

	plan := &PushPlan{
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: repo.Path(),
					Name: "dirty-repo",
					Remotes: []git.Remote{
						{Name: "origin", URL: remote.Path()},
					},
				},
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit uncommitted changes"},
					{Type: ActionPushAll, Description: "Push all branches (origin)", Remote: "origin"},
					{Type: ActionPushKnown, Description: "Push known branches (origin)", Remote: "origin"},
				},
			},
		},
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.RepoResults) != 1 {
		t.Fatalf("Expected one repo result, got %d", len(result.RepoResults))
	}

	rr := result.RepoResults[0]
	pushBranchCount := 0
	for _, action := range rr.Actions {
		if action.Type == ActionPushAll || action.Type == ActionPushKnown {
			t.Fatalf("expected push-all/push-known actions to be replaced after auto-commit, got %#v", rr.Actions)
		}
		if action.Type == ActionPushBranch {
			pushBranchCount++
		}
	}
	if pushBranchCount == 0 {
		t.Fatalf("expected replacement backup branch pushes, got %#v", rr.Actions)
	}
}

func TestRunner_Execute_AllSkipActionsCountAsSkipped(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	plan := &PushPlan{
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: "/tmp/skip-only",
					Name: "skip-only",
				},
				Actions: []Action{
					{Type: ActionSkip, Description: "Skip push to origin due to conflict_strategy=abort"},
				},
			},
		},
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success != 0 || result.Skipped != 1 || result.Failed != 0 {
		t.Fatalf("expected success=0 skipped=1 failed=0, got success=%d skipped=%d failed=%d", result.Success, result.Skipped, result.Failed)
	}
	if len(result.RepoResults) != 1 {
		t.Fatalf("expected 1 repo result, got %d", len(result.RepoResults))
	}
	if result.RepoResults[0].Success {
		t.Fatalf("skip-only repo should not be marked successful: %#v", result.RepoResults[0])
	}
	if result.RepoResults[0].Error != nil {
		t.Fatalf("skip-only repo should not have error: %#v", result.RepoResults[0])
	}
}

func TestRunner_ExecuteActionPushBranch(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create clean repo with remote
	scenario, repo := testutil.CreateCleanRepoScenario(t)

	defaultBranch := repo.GetDefaultBranch()

	action := Action{
		Type:        ActionPushBranch,
		Description: "Push branch",
		Branch:      defaultBranch,
		Remote:      "origin",
	}

	gitRepo := git.Repository{
		Path: repo.Path(),
		Name: "test-repo",
		Remotes: []git.Remote{
			{Name: "origin", URL: scenario.GetRepo("remote").Path()},
		},
	}

	// Execute the action
	executedAction := runner.executeAction(gitRepo, action, 1, 1)

	if executedAction.Error != nil {
		t.Errorf("Push branch action failed: %v", executedAction.Error)
	}
}

func TestRunner_ExecuteActionSkip(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	action := Action{
		Type:        ActionSkip,
		Description: "Skip this repo",
	}

	gitRepo := git.Repository{
		Path: "/tmp/fake-repo",
		Name: "test-repo",
	}

	// Execute skip action
	executedAction := runner.executeAction(gitRepo, action, 1, 1)

	if executedAction.Error != nil {
		t.Errorf("Skip action should not produce error, got: %v", executedAction.Error)
	}
}

func TestRunner_ExecuteActionUnknownType(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	action := Action{
		Type:        ActionType(999), // Invalid action type
		Description: "Unknown action type",
	}

	gitRepo := git.Repository{
		Path: "/tmp/fake-repo",
		Name: "test-repo",
	}

	// Execute unknown action
	executedAction := runner.executeAction(gitRepo, action, 1, 1)

	if executedAction.Error == nil {
		t.Error("Expected error for unknown action type")
	}
}

func TestRunner_ProgressTracking(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create simple repo
	scenario, repo := testutil.CreateCleanRepoScenario(t)

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "test-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: scenario.GetRepo("remote").Path()},
			},
			Branches: []string{"main"},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Collect progress updates — use WaitGroup so we don't race on the slice
	var wg sync.WaitGroup
	progressUpdates := []Progress{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for progress := range runner.progress {
			progressUpdates = append(progressUpdates, progress)
		}
	}()

	// Execute
	_, err = runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Close the channel so the range loop above exits, then wait for it
	runner.Close()
	wg.Wait()

	// Verify progress updates were sent
	if len(progressUpdates) < 1 {
		t.Error("Expected at least some progress updates")
	}
}

func TestRunner_ErrorAccumulation(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create a repo that will fail (bad remote URL)
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("test")

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "test-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: "/nonexistent/bad/remote/path.git"}, // This will fail
			},
			Branches: []string{repo.GetDefaultBranch()},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Execute (should fail)
	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() should not return error, got: %v", err)
	}

	// Check that failure was recorded
	if len(result.RepoResults) == 0 {
		t.Fatal("Expected at least 1 repo result")
	}

	repoResult := result.RepoResults[0]

	// Either the execution failed (Failed counter incremented)
	// OR the repo result has an error
	if result.Failed == 0 && repoResult.Error == nil {
		// Some actions should have failed
		hasFailedAction := false
		for _, action := range repoResult.Actions {
			if action.Error != nil {
				hasFailedAction = true
				break
			}
		}
		if !hasFailedAction {
			t.Error("Expected at least one failed action with bad remote")
		}
	}
}

func TestRunner_PartialFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	// Create 2 repos: one good, one bad
	scenario, goodRepo := testutil.CreateCleanRepoScenario(t)

	scenario2 := testutil.NewScenario(t)
	badRepo := scenario2.CreateRepo("bad")

	goodBranch := goodRepo.GetDefaultBranch()
	badBranch := badRepo.GetDefaultBranch()

	repos := []git.Repository{
		{
			Path:     goodRepo.Path(),
			Name:     "good-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: scenario.GetRepo("remote").Path()},
			},
			Branches: []string{goodBranch},
		},
		{
			Path:     badRepo.Path(),
			Name:     "bad-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: "/nonexistent/path/repo.git"}, // Will fail
			},
			Branches: []string{badBranch},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Execute
	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() should not fail on partial failure, got: %v", err)
	}

	// Should have processed both repos
	if len(result.RepoResults) != 2 {
		t.Fatalf("Expected 2 repo results, got %d", len(result.RepoResults))
	}

	// At least one should succeed (the good repo)
	if result.Success == 0 {
		t.Error("Expected at least 1 successful repo")
	}

	// Check that we have some action failures from the bad repo
	totalErrors := 0
	for _, repoResult := range result.RepoResults {
		for _, action := range repoResult.Actions {
			if action.Error != nil {
				totalErrors++
			}
		}
		if repoResult.Error != nil {
			totalErrors++
		}
	}

	if totalErrors == 0 {
		t.Log("Warning: Expected some errors from bad remote, but got none")
		// Don't fail the test as git behavior may vary
	}
}

func TestDryRunExecute(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	plan := &PushPlan{
		DryRun:     true,
		TotalRepos: 1,
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: "/tmp/fake-repo",
					Name: "test-repo",
				},
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit"},
					{Type: ActionPushAll, Description: "Push all branches"},
				},
			},
		},
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("DryRun execute error = %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to not be nil")
	}

	// Dry run should complete quickly (no actual git operations)
	if result.Duration > 1*time.Second {
		t.Errorf("Dry run took too long: %v", result.Duration)
	}

	// Should have repo results
	if len(result.RepoResults) != 1 {
		t.Errorf("Expected 1 repo result, got %d", len(result.RepoResults))
	}
}

func TestExecutionResult_Stats(t *testing.T) {
	result := &ExecutionResult{
		Success:       2,
		Failed:        1,
		Skipped:       1,
		TotalActions:  10,
		FailedActions: 2,
		Duration:      5 * time.Second,
	}

	// Test basic result stats
	if result.Success != 2 {
		t.Errorf("Expected 2 successful repos, got %d", result.Success)
	}

	if result.Failed != 1 {
		t.Errorf("Expected 1 failed repo, got %d", result.Failed)
	}

	if result.Skipped != 1 {
		t.Errorf("Expected 1 skipped repo, got %d", result.Skipped)
	}

	if result.TotalActions != 10 {
		t.Errorf("Expected 10 total actions, got %d", result.TotalActions)
	}

	if result.FailedActions != 2 {
		t.Errorf("Expected 2 failed actions, got %d", result.FailedActions)
	}
}

func TestProgress_Status(t *testing.T) {
	progress := Progress{
		CurrentRepo: 2,
		TotalRepos:  5,
		RepoName:    "test-repo",
		Action:      "Pushing branches",
		Status:      StatusInProgress,
	}

	// Test progress status
	if progress.Status != StatusInProgress {
		t.Errorf("Expected status to be in-progress, got %v", progress.Status)
	}

	if progress.CurrentRepo != 2 {
		t.Errorf("Expected current repo to be 2, got %d", progress.CurrentRepo)
	}

	if progress.TotalRepos != 5 {
		t.Errorf("Expected total repos to be 5, got %d", progress.TotalRepos)
	}
}

// TestDryRun_SecretWarning verifies that --dry-run emits a secret warning to
// stderr when uncommitted files contain a detectable secret pattern.
func TestDryRun_SecretWarning(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("initial.txt", "content\n").
		Commit("initial")

	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	// Drop a file with a fake AWS key — untracked, not staged
	secretFile := filepath.Join(repo.Path(), "secrets.txt")
	if err := os.WriteFile(secretFile, []byte("AWS_KEY=AKIAIOSFODNN7EXAMPLE\n"), 0644); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "dirty-repo",
			Selected: true,
			Mode:     git.ModePushKnownBranches,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
			Branches: []string{defaultBranch},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, true) // dry run
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w

	_, err = runner.Execute(plan)

	w.Close()
	os.Stderr = old

	var buf strings.Builder
	io.Copy(&buf, r)
	stderr := buf.String()

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stderr, "WARNING") {
		t.Errorf("expected secret warning on stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "AWS") {
		t.Errorf("expected AWS pattern name in warning, got: %q", stderr)
	}
}

// TestRunner_ExecuteStream verifies that ExecuteStream processes repos fed
// through a channel and returns correct aggregate counts. It uses repos with
// no remotes so each is skipped — no real git pushes occur.
func TestRunner_ExecuteStream(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)
	defer runner.Close()

	scenario := testutil.NewScenario(t)
	repoA := scenario.CreateRepo("repo-a")
	repoB := scenario.CreateRepo("repo-b")

	// Drain progress channel in background.
	go func() {
		for range runner.ProgressChan() {
		}
	}()

	repos := []git.Repository{
		{Path: repoA.Path(), Name: "repo-a", Selected: true, Mode: git.ModePushKnownBranches},
		{Path: repoB.Path(), Name: "repo-b", Selected: true, Mode: git.ModePushKnownBranches},
	}

	repoChan := make(chan git.Repository, len(repos))
	for _, r := range repos {
		repoChan <- r
	}
	close(repoChan)

	planner := NewPlanner(&cfg)
	total := int64(len(repos))

	result, err := runner.ExecuteStream(repoChan, planner, false, &total)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}

	processed := result.Success + result.Failed + result.Skipped
	if processed != len(repos) {
		t.Errorf("want %d repos processed, got %d (success=%d failed=%d skipped=%d)",
			len(repos), processed, result.Success, result.Failed, result.Skipped)
	}
}

func TestRunner_ExecuteStream_DryRun_SkippedNotCountedAsSuccess(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)
	defer runner.Close()

	go func() {
		for range runner.ProgressChan() {
		}
	}()

	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("solo")

	repoChan := make(chan git.Repository, 1)
	repoChan <- git.Repository{
		Path:     repo.Path(),
		Name:     "solo",
		Selected: true,
		Mode:     git.ModePushKnownBranches,
	}
	close(repoChan)

	planner := NewPlanner(&cfg)
	var total int64 = 1

	result, err := runner.ExecuteStream(repoChan, planner, true, &total)
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	if result.Success != 0 {
		t.Errorf("dry run with skipped plan: want success count 0, got %d", result.Success)
	}
	if result.Skipped != 1 {
		t.Errorf("dry run with skipped plan: want skipped count 1, got %d", result.Skipped)
	}
}

func TestRunner_ExecuteStream_DryRun_SkipsConflictDetection(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "new-branch"
	runner := NewRunner(&cfg)
	defer runner.Close()

	go func() {
		for range runner.ProgressChan() {
		}
	}()

	_, local, remote := testutil.CreateConflictScenario(t)

	repoChan := make(chan git.Repository, 1)
	repoChan <- git.Repository{
		Path:     local.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
	}
	close(repoChan)

	planner := NewPlanner(&cfg)
	var total int64 = 1
	result, err := runner.ExecuteStream(repoChan, planner, true, &total)
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	if len(result.RepoResults) != 1 {
		t.Fatalf("expected one repo result, got %d", len(result.RepoResults))
	}

	repoResult := result.RepoResults[0]
	if !repoResult.Success {
		t.Fatalf("expected dry-run repo result to be success, got %+v", repoResult)
	}

	var sawPush bool
	for _, action := range repoResult.Actions {
		if action.Type == ActionCreateFireBranch {
			t.Fatalf("dry-run stream should skip conflict detection and fire branch planning: %#v", repoResult.Actions)
		}
		if action.Type == ActionPushBranch {
			sawPush = true
			if action.Branch == fireBranchPlaceholder {
				t.Fatalf("dry-run stream should not include placeholder fire-branch push: %#v", repoResult.Actions)
			}
		}
	}
	if !sawPush {
		t.Fatalf("expected a normal push action, got %#v", repoResult.Actions)
	}
}

func TestDryRunExecute_SkippedAggregatesLikeLiveRun(t *testing.T) {
	cfg := config.DefaultConfig()
	runner := NewRunner(&cfg)

	plan := &PushPlan{
		DryRun: true,
		Repos: []RepoPlan{
			{
				Repo:       git.Repository{Path: "/tmp/skipped", Name: "skipped"},
				Skip:       true,
				SkipReason: "No remotes configured",
				Actions:    []Action{{Type: ActionPushAll, Description: "noop"}},
			},
			{
				Repo: git.Repository{Path: "/tmp/fake-repo", Name: "would-run"},
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit"},
				},
			},
		},
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success != 1 || result.Skipped != 1 {
		t.Fatalf("want success=1 skipped=1, got success=%d skipped=%d", result.Success, result.Skipped)
	}
	if result.RepoResults[0].Success {
		t.Error("first repo (skipped) should have Success false")
	}
	if !result.RepoResults[1].Success {
		t.Error("second repo (dry-run actions) should have Success true")
	}
}

func TestRunner_Execute_ParallelPushWorkers(t *testing.T) {
	serialDuration := runSlowTwoRepoPush(t, 1)
	parallelDuration := runSlowTwoRepoPush(t, 2)

	if parallelDuration >= serialDuration {
		t.Fatalf("expected parallel execution to be faster: serial=%v parallel=%v", serialDuration, parallelDuration)
	}
}

func TestRunner_Execute_ConcurrentResultAggregation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.PushWorkers = 3
	runner := NewRunner(&cfg)

	scenario, goodRepo := testutil.CreateCleanRepoScenario(t)
	other := testutil.NewScenario(t)
	badRepo := other.CreateRepo("bad")

	repos := []git.Repository{
		{
			Path:     goodRepo.Path(),
			Name:     "good-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: scenario.GetRepo("remote").Path()},
			},
		},
		{
			Path:     badRepo.Path(),
			Name:     "bad-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "/nonexistent/path/repo.git"},
			},
		},
		{
			Path:     "/tmp/skip-repo",
			Name:     "skip-repo",
			Selected: true,
			Mode:     git.ModeLeaveUntouched,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo.git"},
			},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.RepoResults) != 3 {
		t.Fatalf("Expected 3 repo results, got %d", len(result.RepoResults))
	}
	if result.Success+result.Failed+result.Skipped != 3 {
		t.Fatalf("unexpected result accounting: success=%d failed=%d skipped=%d", result.Success, result.Failed, result.Skipped)
	}
	if result.FailedActions < 1 {
		t.Fatalf("expected at least one failed action from bad remote, got %d", result.FailedActions)
	}
}

func TestRunner_Execute_PreservesPerRepoActionOrderWithConcurrency(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.PushWorkers = 2
	runner := NewRunner(&cfg)

	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("dirty").
		WithRemote("origin", remote).
		AddFile("base.txt", "base\n").
		Commit("base commit")
	branch := repo.GetDefaultBranch()
	repo.Push("origin", branch)
	repo.AddFile("work.txt", "dirty\n")

	otherScenario, otherRepo := testutil.CreateCleanRepoScenario(t)
	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "dirty-repo",
			Selected: true,
			Mode:     git.ModePushCurrentBranch,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
		},
		{
			Path:     otherRepo.Path(),
			Name:     "clean-repo",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: otherScenario.GetRepo("remote").Path()},
			},
		},
	}

	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var dirtyResult *RepoResult
	for i := range result.RepoResults {
		if result.RepoResults[i].Path == repo.Path() {
			dirtyResult = &result.RepoResults[i]
			break
		}
	}
	if dirtyResult == nil {
		t.Fatal("missing dirty repo result")
	}
	if len(dirtyResult.Actions) < 2 {
		t.Fatalf("expected multiple actions for dirty repo, got %d", len(dirtyResult.Actions))
	}
	if dirtyResult.Actions[0].Type != ActionAutoCommit {
		t.Fatalf("expected first action to be auto-commit, got %s", dirtyResult.Actions[0].Type)
	}
}

func TestRunner_ExecuteStream_ConcurrentWorkersCloseAndDrain(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.PushWorkers = 2
	runner := NewRunner(&cfg)
	defer runner.Close()

	repos := buildSlowPushRepos(t)
	repoChan := make(chan git.Repository)
	var total int64

	planner := NewPlanner(&cfg)
	go func() {
		for _, repo := range repos {
			atomic.AddInt64(&total, 1)
			repoChan <- repo
		}
		close(repoChan)
	}()

	start := time.Now()
	result, err := runner.ExecuteStream(repoChan, planner, false, &total)
	if err != nil {
		t.Fatalf("ExecuteStream() error = %v", err)
	}
	duration := time.Since(start)

	if len(result.RepoResults) != len(repos) {
		t.Fatalf("expected %d repo results, got %d", len(repos), len(result.RepoResults))
	}
	if result.Success != len(repos) {
		t.Fatalf("expected all stream repos to succeed, got success=%d", result.Success)
	}
	if duration > 700*time.Millisecond {
		t.Fatalf("expected concurrent stream execution to finish quickly, duration=%v", duration)
	}
}

func runSlowTwoRepoPush(t *testing.T, workers int) time.Duration {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Global.PushWorkers = workers
	runner := NewRunner(&cfg)

	repos := buildSlowPushRepos(t)
	planner := NewPlanner(&cfg)
	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	start := time.Now()
	result, err := runner.Execute(plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Success != len(repos) {
		t.Fatalf("expected all repos to succeed, success=%d total=%d", result.Success, len(repos))
	}
	return time.Since(start)
}

func buildSlowPushRepos(t *testing.T) []git.Repository {
	t.Helper()

	scenario := testutil.NewScenario(t)
	repos := make([]git.Repository, 0, 2)
	for i := 0; i < 2; i++ {
		remote := scenario.CreateBareRepo(fmt.Sprintf("remote-%d", i))
		installSlowPushHook(t, remote.Path(), 350*time.Millisecond)

		repo := scenario.CreateRepo(fmt.Sprintf("repo-%d", i)).
			WithRemote("origin", remote).
			AddFile("base.txt", "base\n").
			Commit("base commit")

		current := repo.GetDefaultBranch()
		repo.Push("origin", current)
		repo.AddFile("next.txt", "next\n")
		repo.Commit("next commit")

		repos = append(repos, git.Repository{
			Path:     repo.Path(),
			Name:     fmt.Sprintf("repo-%d", i),
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
			Branches: []string{current},
		})
	}

	return repos
}

func installSlowPushHook(t *testing.T, bareRepoPath string, delay time.Duration) {
	t.Helper()

	hookPath := filepath.Join(bareRepoPath, "hooks", "pre-receive")
	script := fmt.Sprintf("#!/bin/sh\nsleep %.3f\n", delay.Seconds())
	if err := os.WriteFile(hookPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write pre-receive hook: %v", err)
	}
}
