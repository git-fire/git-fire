package executor

import (
	"fmt"
	"time"

	"github.com/TBRX103/git-fire/internal/config"
	"github.com/TBRX103/git-fire/internal/git"
)

// Planner creates execution plans
type Planner struct {
	config *config.Config
}

// NewPlanner creates a new planner
func NewPlanner(cfg *config.Config) *Planner {
	return &Planner{
		config: cfg,
	}
}

// BuildPlan creates a push plan from scanned repositories
func (p *Planner) BuildPlan(repos []git.Repository, dryRun bool) (*PushPlan, error) {
	plan := &PushPlan{
		Repos:     make([]RepoPlan, 0, len(repos)),
		DryRun:    dryRun,
		CreatedAt: time.Now(),
	}

	for _, repo := range repos {
		if !repo.Selected {
			continue // Skip unselected repos
		}

		repoPlan, err := p.BuildRepoPlan(repo)
		if err != nil {
			return nil, fmt.Errorf("failed to plan repo %s: %w", repo.Path, err)
		}

		plan.Repos = append(plan.Repos, repoPlan)

		// Update stats
		plan.TotalRepos++
		if repo.IsDirty {
			plan.DirtyRepos++
		}
		if repoPlan.HasConflict {
			plan.Conflicts++
		}
		if repoPlan.FireBranch != "" {
			plan.FireBranches++
		}
	}

	return plan, nil
}

// BuildRepoPlan creates a plan for a single repository. It is exported so the
// streaming executor can plan each repo as it arrives from the scanner.
func (p *Planner) BuildRepoPlan(repo git.Repository) (RepoPlan, error) {
	repoPlan := RepoPlan{
		Repo:    repo,
		Actions: []Action{},
	}

	// Check if repo should be skipped
	if repo.Mode == git.ModeLeaveUntouched {
		repoPlan.Skip = true
		repoPlan.SkipReason = "Mode set to leave-untouched"
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        ActionSkip,
			Description: "Skipping (leave-untouched)",
		})
		return repoPlan, nil
	}

	// Check if repo has remotes
	if len(repo.Remotes) == 0 {
		repoPlan.Skip = true
		repoPlan.SkipReason = "No remotes configured"
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        ActionSkip,
			Description: "Skipping (no remotes)",
		})
		return repoPlan, nil
	}

	// Step 1: Auto-commit if dirty
	if repo.IsDirty && p.config.Global.AutoCommitDirty {
		repoPlan.Actions = append(repoPlan.Actions, Action{
			Type:        ActionAutoCommit,
			Description: "Auto-commit uncommitted changes",
		})
	}

	// Get current branch (we'll simulate this - in real code would query git)
	// For now, assume "main" or first branch
	currentBranch := "main"
	if len(repo.Branches) > 0 {
		currentBranch = repo.Branches[0]
	}

	// Step 2+3: Determine push strategy and add an action for every remote.
	// In an emergency every configured remote is a backup destination.
	for _, remote := range repo.Remotes {
		switch repo.Mode {
		case git.ModePushKnownBranches:
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionPushKnown,
				Description: fmt.Sprintf("Push branches that exist on remote (%s)", remote.Name),
				Remote:      remote.Name,
			})

		case git.ModePushAll:
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionPushAll,
				Description: fmt.Sprintf("Push all branches (%s)", remote.Name),
				Remote:      remote.Name,
			})

		default:
			// Default to pushing current branch
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionPushBranch,
				Description: fmt.Sprintf("Push branch %s (%s)", currentBranch, remote.Name),
				Remote:      remote.Name,
				Branch:      currentBranch,
			})
		}
	}

	return repoPlan, nil
}

// Summary returns a human-readable summary of the plan
func (p *PushPlan) Summary() string {
	if len(p.Repos) == 0 {
		return "No repositories selected for push."
	}

	summary := fmt.Sprintf("Push Plan:\n")
	summary += fmt.Sprintf("  Total repositories: %d\n", p.TotalRepos)
	summary += fmt.Sprintf("  Dirty repositories: %d\n", p.DirtyRepos)

	if p.Conflicts > 0 {
		summary += fmt.Sprintf("  Conflicts detected: %d\n", p.Conflicts)
		summary += fmt.Sprintf("  Fire branches to create: %d\n", p.FireBranches)
	}

	if p.DryRun {
		summary += "\n⚠️  DRY RUN - No changes will be made\n"
	}

	summary += "\nRepositories:\n"
	for i, repo := range p.Repos {
		summary += fmt.Sprintf("\n%d. %s\n", i+1, repo.Repo.Name)
		summary += fmt.Sprintf("   Path: %s\n", repo.Repo.Path)

		if repo.Skip {
			summary += fmt.Sprintf("   Status: SKIP (%s)\n", repo.SkipReason)
			continue
		}

		summary += "   Actions:\n"
		for _, action := range repo.Actions {
			summary += fmt.Sprintf("     - %s: %s\n", action.Type, action.Description)
		}

		if repo.HasConflict {
			summary += fmt.Sprintf("   ⚠️  Conflict: Will create fire branch: %s\n", repo.FireBranch)
		}
	}

	return summary
}

// ValidatePlan checks if a plan is valid and safe to execute
func (p *PushPlan) Validate() error {
	if len(p.Repos) == 0 {
		return fmt.Errorf("no repositories in plan")
	}

	for _, repo := range p.Repos {
		if repo.Skip {
			continue
		}

		if len(repo.Actions) == 0 {
			return fmt.Errorf("repo %s has no actions", repo.Repo.Path)
		}

		// Check that repo has remotes
		if len(repo.Repo.Remotes) == 0 && !repo.Skip {
			return fmt.Errorf("repo %s has no remotes configured", repo.Repo.Path)
		}
	}

	return nil
}
