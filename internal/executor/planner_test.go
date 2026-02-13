package executor

import (
	"testing"

	"github.com/TBRX103/git-fire/internal/config"
	"github.com/TBRX103/git-fire/internal/git"
)

func TestBuildPlan(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushKnownBranches,
			IsDirty:  true,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo1.git"},
			},
			Branches: []string{"main", "develop"},
		},
		{
			Path:     "/home/user/repo2",
			Name:     "repo2",
			Selected: true,
			Mode:     git.ModePushAll,
			IsDirty:  false,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo2.git"},
			},
			Branches: []string{"main"},
		},
		{
			Path:     "/home/user/repo3",
			Name:     "repo3",
			Selected: false, // Not selected
			Mode:     git.ModePushKnownBranches,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo3.git"},
			},
		},
	}

	plan, err := planner.BuildPlan(repos, false)
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}

	// Should only include selected repos (repo1 and repo2)
	if plan.TotalRepos != 2 {
		t.Errorf("Expected 2 repos in plan, got %d", plan.TotalRepos)
	}

	if plan.DirtyRepos != 1 {
		t.Errorf("Expected 1 dirty repo, got %d", plan.DirtyRepos)
	}

	// Check repo1 plan
	repo1Plan := plan.Repos[0]
	if repo1Plan.Repo.Name != "repo1" {
		t.Errorf("Expected first repo to be repo1, got %s", repo1Plan.Repo.Name)
	}

	// Should have auto-commit action (because dirty)
	foundAutoCommit := false
	for _, action := range repo1Plan.Actions {
		if action.Type == ActionAutoCommit {
			foundAutoCommit = true
		}
	}
	if !foundAutoCommit {
		t.Error("Expected auto-commit action for dirty repo")
	}

	// Should have push-known action
	foundPushKnown := false
	for _, action := range repo1Plan.Actions {
		if action.Type == ActionPushKnown {
			foundPushKnown = true
		}
	}
	if !foundPushKnown {
		t.Error("Expected push-known action for repo1")
	}

	// Check repo2 plan
	repo2Plan := plan.Repos[1]
	if repo2Plan.Repo.Name != "repo2" {
		t.Errorf("Expected second repo to be repo2, got %s", repo2Plan.Repo.Name)
	}

	// Should NOT have auto-commit (not dirty)
	foundAutoCommit = false
	for _, action := range repo2Plan.Actions {
		if action.Type == ActionAutoCommit {
			foundAutoCommit = true
		}
	}
	if foundAutoCommit {
		t.Error("Should not have auto-commit action for clean repo")
	}

	// Should have push-all action
	foundPushAll := false
	for _, action := range repo2Plan.Actions {
		if action.Type == ActionPushAll {
			foundPushAll = true
		}
	}
	if !foundPushAll {
		t.Error("Expected push-all action for repo2")
	}
}

func TestBuildPlan_SkipNoRemotes(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModePushKnownBranches,
			Remotes:  []git.Remote{}, // No remotes
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
	if !repoPlan.Skip {
		t.Error("Expected repo to be skipped when no remotes")
	}

	if repoPlan.SkipReason == "" {
		t.Error("Expected skip reason to be set")
	}
}

func TestBuildPlan_LeaveUntouched(t *testing.T) {
	cfg := config.DefaultConfig()
	planner := NewPlanner(&cfg)

	repos := []git.Repository{
		{
			Path:     "/home/user/repo1",
			Name:     "repo1",
			Selected: true,
			Mode:     git.ModeLeaveUntouched,
			Remotes: []git.Remote{
				{Name: "origin", URL: "git@github.com:user/repo1.git"},
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
	if !repoPlan.Skip {
		t.Error("Expected repo to be skipped when mode is leave-untouched")
	}
}

func TestPlanSummary(t *testing.T) {
	plan := &PushPlan{
		TotalRepos: 2,
		DirtyRepos: 1,
		Conflicts:  0,
		DryRun:     true,
		Repos: []RepoPlan{
			{
				Repo: git.Repository{
					Path: "/home/user/repo1",
					Name: "repo1",
				},
				Actions: []Action{
					{Type: ActionAutoCommit, Description: "Auto-commit"},
					{Type: ActionPushKnown, Description: "Push known branches"},
				},
			},
		},
	}

	summary := plan.Summary()

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// Should contain key info
	if !contains(summary, "Total repositories: 2") {
		t.Error("Summary should contain total repos")
	}

	if !contains(summary, "DRY RUN") {
		t.Error("Summary should mention dry run")
	}

	if !contains(summary, "repo1") {
		t.Error("Summary should contain repo name")
	}
}

func TestValidatePlan(t *testing.T) {
	tests := []struct {
		name    string
		plan    *PushPlan
		wantErr bool
	}{
		{
			name: "valid plan",
			plan: &PushPlan{
				Repos: []RepoPlan{
					{
						Repo: git.Repository{
							Path: "/repo1",
							Remotes: []git.Remote{
								{Name: "origin", URL: "git@github.com:user/repo.git"},
							},
						},
						Actions: []Action{
							{Type: ActionPushBranch},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty plan",
			plan: &PushPlan{
				Repos: []RepoPlan{},
			},
			wantErr: true,
		},
		{
			name: "repo with no actions",
			plan: &PushPlan{
				Repos: []RepoPlan{
					{
						Repo: git.Repository{
							Path: "/repo1",
							Remotes: []git.Remote{
								{Name: "origin", URL: "git@github.com:user/repo.git"},
							},
						},
						Actions: []Action{},
					},
				},
			},
			wantErr: true,
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

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
