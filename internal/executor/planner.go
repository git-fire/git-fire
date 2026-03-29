package executor

import (
	"fmt"
	"time"

	"github.com/TBRX103/git-fire/internal/config"
	"github.com/TBRX103/git-fire/internal/git"
)

const fireBranchPlaceholder = "__git_fire_created_branch__"

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

	// Step 2+3: Determine push strategy and add an action for every remote.
	// In an emergency every configured remote is a backup destination.
	//
	// The default mode pushes only the currently checked-out branch. We resolve
	// it from git here rather than using Branches[0], which is in discovery
	// order and may not match. The call is guarded so ModePushKnownBranches /
	// ModePushAll repos (which don't need it) can skip the git invocation.
	var currentBranch string
	if repo.Mode != git.ModePushKnownBranches && repo.Mode != git.ModePushAll {
		var err error
		currentBranch, err = git.GetCurrentBranch(repo.Path)
		if err != nil {
			// Detached HEAD or damaged repo — can't safely target a single branch.
			repoPlan.Skip = true
			repoPlan.SkipReason = fmt.Sprintf("cannot determine current branch: %v", err)
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionSkip,
				Description: repoPlan.SkipReason,
			})
			return repoPlan, nil
		}
	}

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

		case git.ModePushCurrentBranch:
			if p.config.Global.ConflictStrategy == "new-branch" {
				hasConflict, _, _, conflictErr := git.DetectConflict(repo.Path, currentBranch, remote.Name)
				if conflictErr != nil {
					return repoPlan, fmt.Errorf("failed to detect conflict for %s (%s): %w", repo.Name, remote.Name, conflictErr)
				}
				if hasConflict {
					repoPlan.HasConflict = true
					if !repoPlanHasFireCreateAction(repoPlan.Actions) {
						repoPlan.Actions = append(repoPlan.Actions, Action{
							Type:        ActionCreateFireBranch,
							Description: fmt.Sprintf("Create fire backup branch for %s", currentBranch),
							Branch:      currentBranch,
						})
					}
					repoPlan.Actions = append(repoPlan.Actions, Action{
						Type:        ActionPushBranch,
						Description: fmt.Sprintf("Push fire backup branch for %s (%s)", currentBranch, remote.Name),
						Remote:      remote.Name,
						Branch:      fireBranchPlaceholder,
					})
					continue
				}
			}

			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionPushBranch,
				Description: fmt.Sprintf("Push branch %s (%s)", currentBranch, remote.Name),
				Remote:      remote.Name,
				Branch:      currentBranch,
			})

		default:
			repoPlan.Skip = true
			repoPlan.SkipReason = fmt.Sprintf("unsupported mode: %s", repo.Mode.String())
			repoPlan.Actions = append(repoPlan.Actions, Action{
				Type:        ActionSkip,
				Description: repoPlan.SkipReason,
			})
			return repoPlan, nil
		}
	}

	return repoPlan, nil
}

func repoPlanHasFireCreateAction(actions []Action) bool {
	for _, action := range actions {
		if action.Type == ActionCreateFireBranch {
			return true
		}
	}
	return false
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
