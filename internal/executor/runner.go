package executor

import (
	"fmt"
	"time"

	"github.com/TBRX103/git-fire/internal/config"
	"github.com/TBRX103/git-fire/internal/git"
)

// Runner executes push plans
type Runner struct {
	config      *config.Config
	progress    chan Progress
	rateLimiter *HostLimiter
}

// NewRunner creates a new runner
func NewRunner(cfg *config.Config) *Runner {
	// Create rate limiter with default config
	rateLimitConfig := DefaultRateLimitConfig()

	return &Runner{
		config:      cfg,
		progress:    make(chan Progress, 10),
		rateLimiter: NewHostLimiter(rateLimitConfig),
	}
}

// Execute runs the push plan
func (r *Runner) Execute(plan *PushPlan) (*ExecutionResult, error) {
	if plan.DryRun {
		return r.dryRunExecute(plan)
	}

	result := &ExecutionResult{
		Plan:        plan,
		StartTime:   time.Now(),
		RepoResults: make([]RepoResult, 0, len(plan.Repos)),
	}

	for i, repoPlan := range plan.Repos {
		repoResult := r.executeRepo(repoPlan, i+1, len(plan.Repos))
		result.RepoResults = append(result.RepoResults, repoResult)

		if repoResult.Success {
			result.Success++
		} else if repoResult.Error != nil {
			result.Failed++
		} else {
			result.Skipped++
		}

		result.TotalActions += len(repoResult.Actions)
		for _, action := range repoResult.Actions {
			if action.Error != nil {
				result.FailedActions++
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// executeRepo executes actions for a single repository
func (r *Runner) executeRepo(repoPlan RepoPlan, current, total int) RepoResult {
	startTime := time.Now()

	result := RepoResult{
		Path:           repoPlan.Repo.Path,
		Success:        false,
		Actions:        make([]Action, 0),
		PushedBranches: make([]string, 0),
	}

	// Send progress update
	r.sendProgress(Progress{
		CurrentRepo: current,
		TotalRepos:  total,
		RepoName:    repoPlan.Repo.Name,
		Action:      "Starting",
		Status:      StatusStarting,
	})

	// Skip if marked
	if repoPlan.Skip {
		r.sendProgress(Progress{
			CurrentRepo: current,
			TotalRepos:  total,
			RepoName:    repoPlan.Repo.Name,
			Action:      repoPlan.SkipReason,
			Status:      StatusSkipped,
		})
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute each action
	for _, action := range repoPlan.Actions {
		executedAction := r.executeAction(repoPlan.Repo, action, current, total)
		result.Actions = append(result.Actions, executedAction)

		if executedAction.Error != nil {
			result.Error = executedAction.Error
			result.Success = false
			result.Duration = time.Since(startTime)
			return result
		}

		// Track pushed branches
		if action.Type == ActionPushBranch && executedAction.Branch != "" {
			result.PushedBranches = append(result.PushedBranches, executedAction.Branch)
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	r.sendProgress(Progress{
		CurrentRepo: current,
		TotalRepos:  total,
		RepoName:    repoPlan.Repo.Name,
		Action:      "Complete",
		Status:      StatusSuccess,
	})

	return result
}

// executeAction executes a single action
func (r *Runner) executeAction(repo git.Repository, action Action, current, total int) Action {
	executedAction := action

	r.sendProgress(Progress{
		CurrentRepo: current,
		TotalRepos:  total,
		RepoName:    repo.Name,
		Action:      action.Description,
		Status:      StatusInProgress,
	})

	var err error

	switch action.Type {
	case ActionAutoCommit:
		err = git.AutoCommitDirty(repo.Path, git.CommitOptions{
			Message: fmt.Sprintf("git-fire emergency backup - %s", time.Now().Format("2006-01-02 15:04:05")),
		})

	case ActionPushBranch:
		// Apply rate limiting for push operations
		remoteURL := r.getRemoteURL(repo, action.Remote)
		r.rateLimiter.Acquire(remoteURL)
		defer r.rateLimiter.Release(remoteURL)

		err = git.PushBranch(repo.Path, action.Remote, action.Branch)

	case ActionPushAll:
		// Apply rate limiting for push operations
		remoteURL := r.getRemoteURL(repo, action.Remote)
		r.rateLimiter.Acquire(remoteURL)
		defer r.rateLimiter.Release(remoteURL)

		err = git.PushAllBranches(repo.Path, action.Remote)

	case ActionPushKnown:
		// Apply rate limiting for push operations
		remoteURL := r.getRemoteURL(repo, action.Remote)
		r.rateLimiter.Acquire(remoteURL)
		defer r.rateLimiter.Release(remoteURL)

		err = git.PushKnownBranches(repo.Path, action.Remote)

	case ActionCreateFireBranch:
		// Get current branch and SHA
		currentBranch, errBranch := git.GetCurrentBranch(repo.Path)
		if errBranch != nil {
			err = errBranch
			break
		}

		// This would get the SHA, but we'd need to implement it
		// For now, create fire branch without SHA in name
		_, err = git.CreateFireBranch(repo.Path, currentBranch, "unknown")

	case ActionSkip:
		// Nothing to do
		return executedAction

	default:
		err = fmt.Errorf("unknown action type: %s", action.Type)
	}

	executedAction.Error = err

	if err != nil {
		r.sendProgress(Progress{
			CurrentRepo: current,
			TotalRepos:  total,
			RepoName:    repo.Name,
			Action:      action.Description,
			Status:      StatusFailed,
			Error:       err,
		})
	}

	return executedAction
}

// dryRunExecute simulates execution without making changes
func (r *Runner) dryRunExecute(plan *PushPlan) (*ExecutionResult, error) {
	result := &ExecutionResult{
		Plan:        plan,
		StartTime:   time.Now(),
		RepoResults: make([]RepoResult, 0, len(plan.Repos)),
	}

	for i, repoPlan := range plan.Repos {
		repoResult := RepoResult{
			Path:    repoPlan.Repo.Path,
			Success: true,
			Actions: repoPlan.Actions,
		}

		if repoPlan.Skip {
			result.Skipped++
		} else {
			result.Success++
		}

		result.TotalActions += len(repoPlan.Actions)
		result.RepoResults = append(result.RepoResults, repoResult)

		r.sendProgress(Progress{
			CurrentRepo: i + 1,
			TotalRepos:  len(plan.Repos),
			RepoName:    repoPlan.Repo.Name,
			Action:      "[DRY RUN] Would execute actions",
			Status:      StatusSuccess,
		})
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// sendProgress sends a progress update (non-blocking)
func (r *Runner) sendProgress(p Progress) {
	select {
	case r.progress <- p:
	default:
		// Channel full, skip this update
	}
}

// Progress returns the progress channel
func (r *Runner) ProgressChan() <-chan Progress {
	return r.progress
}

// Close closes the progress channel
func (r *Runner) Close() {
	close(r.progress)
}


// getRemoteURL gets the URL for a remote name from the repository
func (r *Runner) getRemoteURL(repo git.Repository, remoteName string) string {
	// Find the remote in the repo's remote list
	for _, remote := range repo.Remotes {
		if remote.Name == remoteName {
			return remote.URL
		}
	}

	// Fallback: return empty string if remote not found
	return ""
}
