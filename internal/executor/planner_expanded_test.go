package executor

import (
	"testing"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	testutil "github.com/git-fire/git-testkit"
)

func TestBuildPlan_DefaultMode(t *testing.T) {
	// Default mode calls git.GetCurrentBranch, so we need a real repo.
	scenario := testutil.NewScenario(t)
	remote := scenario.CreateBareRepo("remote")
	repo := scenario.CreateRepo("repo1").
		WithRemote("origin", remote).
		AddFile("README.md", "hello\n").
		Commit("initial")
	repo.Push("origin", repo.GetDefaultBranch())
	repoPath := repo.Path()

	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     repoPath,
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushCurrentBranch,
			Remotes: []git.Remote{
				{Name: "origin", URL: remote.Path()},
			},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	if len(plan.Repos) != 1 {
		t.Fatalf("Expected 1 repo in plan, got %d", len(plan.Repos))
	}

	repoPlan := plan.Repos[0]

	foundPushBranch := false
	for _, action := range repoPlan.Actions {
		if action.Type == ActionPushBranch {
			foundPushBranch = true
			if action.Branch == "" {
				t.Error("Push branch action should have branch set")
			}
		}
	}

	if !foundPushBranch {
		t.Error("Expected push-branch action for default mode")
	}
}

func TestBuildPlan_WithConflict(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo1.git"},
			},
			Branches: []string{"main"},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Verify conflict stats
	if plan.Conflicts > 0 {
		if plan.FireBranches == 0 {
			t.Error("If conflicts detected, should have fire branches")
		}
	}
}

func TestBuildPlan_MultipleRemotes(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo1.git"},
				{Name: "backup", URL: "git@gitlab.com:user/repo1.git"},
				{Name: "upstream", URL: "git@github.com:org/repo1.git"},
			},
			Branches: []string{"main"},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	if len(plan.Repos) != 1 {
		t.Fatalf("Expected 1 repo in plan, got %d", len(plan.Repos))
	}

	// Should use first remote for actions
	repoPlan := plan.Repos[0]
	if len(repoPlan.Actions) == 0 {
		t.Fatal("Expected actions to be created")
	}

	// Check that actions use one of the remotes
	for _, action := range repoPlan.Actions {
		if action.Remote != "" && action.Remote != "origin" {
			// It's okay if it uses a different remote
			t.Logf("Action using remote: %s", action.Remote)
		}
	}
}

func TestBuildPlan_OnlySelectedRepos(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo1.git"},
			},
		},
		{
			Path:     "/home/user/repo2",
			Name:     "repo2",
			Selected: false, // Not selected
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo2.git"},
			},
		},
		{
			Path:     "/home/user/repo3",
			Name:     "repo3",
			Selected: true,
			Mode:     git.ModePushAll,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo3.git"},
			},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Should only include selected repos
	if plan.TotalRepos != 2 {
		t.Errorf("Expected 2 repos (only selected), got %d", plan.TotalRepos)
	}

	// Verify correct repos are in plan
	repoNames := make(map[string]bool)
	for _, repoPlan := range plan.Repos {
		repoNames[repoPlan.Repo.Name] = true
	}

	if !repoNames["repo1"] {
		t.Error("Expected repo1 to be in plan")
	}

	if repoNames["repo2"] {
		t.Error("repo2 should not be in plan (not selected)")
	}

	if !repoNames["repo3"] {
		t.Error("Expected repo3 to be in plan")
	}
}

func TestPlanSummary_WithConflicts(t *testing.T) {
	plan := &PushPlan{
		TotalRepos:   2,
		DirtyRepos:   1,
		Conflicts:    2,
		FireBranches: 2,
		DryRun:       false,
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: "/home/user/repo1",
					Name: "repo1",
				},
				HasConflict: true,
				FireBranch:  "git-fire-backup-main-20260213",
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit"},
					{Type: ActionPushAll, Description: "Push all branches"},
				},
			},
		},
	}

	summary := plan.Summary()

	if !contains(summary, "Conflicts detected: 2") {
		t.Error("Summary should mention conflicts")
	}

	if !contains(summary, "Fire branches to create: 2") {
		t.Error("Summary should mention fire branches")
	}

	if !contains(summary, "git-fire-backup-main-20260213") {
		t.Error("Summary should include fire branch name")
	}
}

func TestPlanSummary_EmptyPlan(t *testing.T) {
	plan := &PushPlan{
		Repos: []RepoPlan{},
	}

	summary := plan.Summary()

	if !contains(summary, "No repositories") {
		t.Error("Summary should indicate no repositories")
	}
}

func TestPlanSummary_SkippedRepos(t *testing.T) {
	plan := &PushPlan{
		TotalRepos: 1,
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Name: "skipped-repo",
					Path: "/home/user/skipped",
				},
				Skip:       true,
				SkipReason: "No remotes configured",
			},
		},
	}

	summary := plan.Summary()

	if !contains(summary, "SKIP") {
		t.Error("Summary should mention skipped repos")
	}

	if !contains(summary, "No remotes configured") {
		t.Error("Summary should include skip reason")
	}
}

func TestValidatePlan_InvalidRepos(t *testing.T) {
	tests := []struct {
		name    string
		plan    *PushPlan
		wantErr bool
	}{
		{
			name: "repo with no remotes and not skipped",
			plan: &PushPlan{
				Repos: []RepoPlan{
					{
						Repo: git.Repository{
							Path:    "/repo1",
							Remotes: []git.Remote{}, // No remotes
						},
						Skip: false,
						Actions: []Action{
							{Type: ActionPushBranch},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "skipped repo with no remotes is okay",
			plan: &PushPlan{
				Repos: []RepoPlan{
					{
						Repo: git.Repository{
							Path:    "/repo1",
							Remotes: []git.Remote{}, // No remotes
						},
						Skip: true,
						Actions: []Action{
							{Type: ActionSkip},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plan.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionType_String(t *testing.T) {
	tests := []struct {
		actionType ActionType
		want       string
	}{
		{ActionAutoCommit, "auto-commit"},
		{ActionPushBranch, "push-branch"},
		{ActionPushAll, "push-all"},
		{ActionPushKnown, "push-known"},
		{ActionCreateFireBranch, "create-fire-branch"},
		{ActionSkip, "skip"},
		{ActionType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.actionType.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProgressStatus_String(t *testing.T) {
	tests := []struct {
		status ProgressStatus
		want   string
	}{
		{StatusStarting, "starting"},
		{StatusInProgress, "in-progress"},
		{StatusSuccess, "success"},
		{StatusFailed, "failed"},
		{StatusSkipped, "skipped"},
		{ProgressStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildRepoPlan_ConflictStrategyNewBranch(t *testing.T) {
	_, repo, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "new-branch"
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildRepoPlan(git.Repository{
		Path:     repo.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
	})
	if err != nil {
		t.Fatalf("BuildRepoPlan() error = %v", err)
	}
	if !plan.HasConflict {
		t.Fatal("Expected conflict to be detected")
	}
	if len(plan.Actions) == 0 || plan.Actions[0].Type != ActionCreateFireBranch {
		t.Fatalf("Expected first action to be create-fire-branch, got %#v", plan.Actions)
	}
	foundPlaceholderPush := false
	for _, action := range plan.Actions {
		if action.Type == ActionPushBranch && action.Branch == fireBranchPlaceholder {
			foundPlaceholderPush = true
			break
		}
	}
	if !foundPlaceholderPush {
		t.Fatalf("Expected conflicting remote push to use placeholder branch, got %#v", plan.Actions)
	}
	if plan.FireBranch != fireBranchPlaceholder {
		t.Fatalf("Expected fire branch placeholder to be set, got %q", plan.FireBranch)
	}
}

func TestBuildRepoPlan_ConflictStrategyAbort(t *testing.T) {
	_, repo, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "abort"
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildRepoPlan(git.Repository{
		Path:     repo.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
	})
	if err != nil {
		t.Fatalf("BuildRepoPlan() error = %v", err)
	}
	if !plan.HasConflict {
		t.Fatal("Expected conflict to be detected")
	}
	if !plan.Skip {
		t.Fatal("Expected plan to skip repo when strategy is abort")
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Type != ActionSkip {
		t.Fatalf("Expected exactly one skip action for abort strategy, got %#v", plan.Actions)
	}
	if plan.FireBranch != "" {
		t.Fatalf("Expected FireBranch cleared on abort, got %q", plan.FireBranch)
	}
}

func TestBuildPlan_ConflictStrategyAbort_Stats(t *testing.T) {
	_, repo, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "abort"
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     repo.Path(),
			Name:     "local",
			Selected: true,
			Mode:     git.ModePushCurrentBranch,
			Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.Conflicts != 1 {
		t.Fatalf("Expected 1 conflict in plan stats, got %d", plan.Conflicts)
	}
	if plan.FireBranches != 0 {
		t.Fatalf("Expected 0 fire branches for abort strategy, got %d", plan.FireBranches)
	}
	if len(plan.Repos) != 1 || !plan.Repos[0].Skip {
		t.Fatalf("Expected one skipped repo plan for abort strategy, got %#v", plan.Repos)
	}
}

func TestBuildRepoPlan_ConflictStrategyAbort(t *testing.T) {
	_, repo, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "abort"
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildRepoPlan(git.Repository{
		Path:     repo.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
	})
	if err != nil {
		t.Fatalf("BuildRepoPlan() error = %v", err)
	}
	if !plan.HasConflict {
		t.Fatal("Expected conflict to be detected")
	}
	if plan.Skip {
		t.Fatal("abort strategy should not mark whole repo skipped when only the diverged remote is skipped")
	}
	var sawSkip bool
	for _, a := range plan.Actions {
		if a.Type == ActionSkip {
			sawSkip = true
			break
		}
	}
	if !sawSkip {
		t.Fatalf("expected a skip action for diverged remote, got %#v", plan.Actions)
	}
	for _, a := range plan.Actions {
		if a.Type == ActionCreateFireBranch {
			t.Fatalf("abort strategy should not create fire branch, got %#v", plan.Actions)
		}
	}
}

func TestBuildRepoPlan_SkipConflictDetection_NoFetch(t *testing.T) {
	_, repo, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "new-branch"
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildRepoPlanWithOptions(git.Repository{
		Path:     repo.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
	}, RepoPlanOptions{DetectConflicts: false})
	if err != nil {
		t.Fatalf("BuildRepoPlanWithOptions() error = %v", err)
	}
	if plan.HasConflict {
		t.Fatal("With DetectConflicts=false, planner should not run fetch/conflict detection")
	}
	var sawPush bool
	for _, a := range plan.Actions {
		if a.Type == ActionPushBranch && a.Branch != fireBranchPlaceholder {
			sawPush = true
		}
	}
	if !sawPush {
		t.Fatalf("expected a normal push-branch action, got %#v", plan.Actions)
	}
}

func TestBuildPlan_DryRunSkipsConflictDetection(t *testing.T) {
	_, local, remote := testutil.CreateConflictScenario(t)

	cfg := config.DefaultConfig()
	cfg.Global.ConflictStrategy = "new-branch"
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildPlan([]git.Repository{{
		Path:     local.Path(),
		Name:     "local",
		Selected: true,
		Mode:     git.ModePushCurrentBranch,
		Remotes:  []git.Remote{{Name: "origin", URL: remote.Path()}},
		IsDirty:  false,
	}}, true)
	if err != nil {
		t.Fatalf("BuildPlan dry-run: %v", err)
	}
	if plan.Conflicts > 0 {
		t.Errorf("dry-run plan should not run conflict detection (no fetch); Conflicts=%d", plan.Conflicts)
	}
	if len(plan.Repos) != 1 {
		t.Fatalf("expected 1 repo plan, got %d: %+v", len(plan.Repos), plan)
	}
	if plan.Repos[0].HasConflict {
		t.Errorf("unexpected conflict in dry-run plan: %+v", plan.Repos[0])
	}
}

func TestBuildPlan_RepoOverrideMode(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Repos = []config.RepoOverride{
		{PathPattern: "/override/repo/*", Mode: "leave-untouched"},
	}
	planner := NewPlanner(&cfg)

	repo := git.Repository{
		Path:     "/override/repo/myapp",
		Name:     "myapp",
		Selected: true,
		Mode:     git.ModePushAll,
		Remotes: []git.Remote{
			{Name: "origin", URL: "git@github.com:x/y.git"},
		},
	}
	plan, err := planner.BuildPlan([]git.Repository{repo}, false)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Repos) != 1 {
		t.Fatalf("want 1 repo plan, got %d", len(plan.Repos))
	}
	if !plan.Repos[0].Skip || plan.Repos[0].SkipReason == "" {
		t.Fatalf("override should force leave-untouched / skip, got Skip=%v reason=%q", plan.Repos[0].Skip, plan.Repos[0].SkipReason)
	}
}

func TestBuildPlan_RepoOverrideSkipAutoCommit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Global.AutoCommitDirty = true
	cfg.Repos = []config.RepoOverride{
		{PathPattern: "/no/commit/*", Mode: "push-all", SkipAutoCommit: true},
	}
	planner := NewPlanner(&cfg)

	plan, err := planner.BuildPlan([]git.Repository{{
		Path:     "/no/commit/r",
		Name:     "r",
		Selected: true,
		Mode:     git.ModePushAll,
		IsDirty:  true,
		Remotes:  []git.Remote{{Name: "origin", URL: "git@example.com/r.git"}},
	}}, false)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Repos) != 1 {
		t.Fatalf("expected 1 repo plan, got %d", len(plan.Repos))
	}
	if plan.Repos[0].Skip {
		t.Fatalf("repo should not be skipped in this test; reason=%q", plan.Repos[0].SkipReason)
	}
	for _, a := range plan.Repos[0].Actions {
		if a.Type == ActionAutoCommit {
			t.Fatal("SkipAutoCommit override should omit auto-commit action")
		}
	}
}
