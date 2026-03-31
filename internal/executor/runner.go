package executor

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/safety"
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

	type executeJob struct {
		index int
		plan  RepoPlan
	}
	type executeOutput struct {
		index  int
		result RepoResult
	}

	jobs := make(chan executeJob, len(plan.Repos))
	outputs := make(chan executeOutput, len(plan.Repos))

	workers := r.pushWorkerCount(len(plan.Repos))
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				repoResult := r.executeRepo(job.plan, job.index+1, len(plan.Repos))
				outputs <- executeOutput{index: job.index, result: repoResult}
			}
		}()
	}

	for i, repoPlan := range plan.Repos {
		jobs <- executeJob{index: i, plan: repoPlan}
	}
	close(jobs)
	wg.Wait()
	close(outputs)

	ordered := make([]RepoResult, len(plan.Repos))
	for output := range outputs {
		ordered[output.index] = output.result
	}
	result.RepoResults = ordered
	r.aggregateResultCounts(result)

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

	actions := append([]Action(nil), repoPlan.Actions...)

	// Execute each action. Continue past failures so every remote gets a
	// best-effort push — collecting the first error to surface to the caller.
	var firstErr error
	for i := 0; i < len(actions); i++ {
		action := actions[i]
		executedAction := r.executeAction(repoPlan.Repo, action, current, total)
		result.Actions = append(result.Actions, executedAction)

		if executedAction.Error != nil {
			if firstErr == nil {
				firstErr = executedAction.Error
			}
		}

		// Track successfully pushed branches only
		if action.Type == ActionPushBranch && executedAction.Branch != "" && executedAction.Error == nil {
			result.PushedBranches = append(result.PushedBranches, executedAction.Branch)
		}

		// Dual-branch auto-commit creates backup branch names. Replace pending
		// push-branch actions to push those backup branches instead of the
		// original current branch.
		if action.Type == ActionAutoCommit && executedAction.Error == nil && executedAction.Branch != "" {
			createdBranches := strings.Split(executedAction.Branch, ",")
			var remotes []string
			for _, pending := range actions[i+1:] {
				if pending.Type == ActionPushBranch && pending.Remote != "" {
					remotes = append(remotes, pending.Remote)
				}
			}
			if len(remotes) == 0 {
				for _, remote := range repoPlan.Repo.Remotes {
					remotes = append(remotes, remote.Name)
				}
			}

			replacementPushes := make([]Action, 0, len(remotes)*len(createdBranches))
			for _, remote := range remotes {
				for _, branch := range createdBranches {
					branch = strings.TrimSpace(branch)
					if branch == "" {
						continue
					}
					replacementPushes = append(replacementPushes, Action{
						Type:        ActionPushBranch,
						Description: fmt.Sprintf("Push backup branch %s (%s)", branch, remote),
						Remote:      remote,
						Branch:      branch,
					})
				}
			}

			filteredTail := make([]Action, 0, len(actions[i+1:]))
			for _, pending := range actions[i+1:] {
				if pending.Type == ActionPushBranch {
					continue
				}
				filteredTail = append(filteredTail, pending)
			}
			actions = append(actions[:i+1], append(replacementPushes, filteredTail...)...)
		}

		// Fire-branch creation runs once; replace pending placeholder pushes so
		// each conflicting remote pushes the created backup branch.
		if action.Type == ActionCreateFireBranch && executedAction.Error == nil && executedAction.Branch != "" {
			for j := i + 1; j < len(actions); j++ {
				if actions[j].Type == ActionPushBranch && actions[j].Branch == fireBranchPlaceholder {
					actions[j].Branch = executedAction.Branch
				}
			}
		}
	}

	result.Success = firstErr == nil
	if firstErr != nil {
		result.Error = firstErr
	}
	result.Duration = time.Since(startTime)

	finalStatus := StatusSuccess
	if firstErr != nil {
		finalStatus = StatusFailed
	}
	r.sendProgress(Progress{
		CurrentRepo: current,
		TotalRepos:  total,
		RepoName:    repoPlan.Repo.Name,
		Action:      "Complete",
		Status:      finalStatus,
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
		// Scan for secrets before committing — warn on stderr but always proceed
		warnAboutSecrets(repo.Path)
		result, commitErr := git.AutoCommitDirtyWithStrategy(repo.Path, git.CommitOptions{
			Message:          fmt.Sprintf("git-fire emergency backup - %s", time.Now().Format("2006-01-02 15:04:05")),
			UseDualBranch:    true,
			ReturnToOriginal: true,
		})
		if commitErr != nil {
			err = commitErr
			break
		}
		created := make([]string, 0, 2)
		if result.StagedBranch != "" {
			created = append(created, result.StagedBranch)
		}
		if result.FullBranch != "" {
			created = append(created, result.FullBranch)
		}
		executedAction.Branch = strings.Join(created, ",")

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
		branchForBackup := action.Branch
		if branchForBackup == "" {
			var errBranch error
			branchForBackup, errBranch = git.GetCurrentBranch(repo.Path)
			if errBranch != nil {
				err = errBranch
				break
			}
		}

		localSHA, errSHA := git.GetCommitSHA(repo.Path, branchForBackup)
		if errSHA != nil {
			err = errSHA
			break
		}

		fireBranch, errBranchCreate := git.CreateFireBranch(repo.Path, branchForBackup, localSHA)
		if errBranchCreate != nil {
			err = errBranchCreate
			break
		}
		executedAction.Branch = fireBranch

		if action.Remote != "" {
			remoteURL := r.getRemoteURL(repo, action.Remote)
			r.rateLimiter.Acquire(remoteURL)
			defer r.rateLimiter.Release(remoteURL)
			err = git.PushBranch(repo.Path, action.Remote, fireBranch)
		}

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

// ExecuteStream reads repositories from repos as they arrive from the scanner,
// builds a per-repo plan, and executes each one immediately. Workers block
// when the channel is empty and stop when it is closed. total is an int64
// pointer updated atomically by the caller's scan goroutine; it is used only
// for progress display and may read as 0 while scanning is still in progress
// (shown as "?" in that case).
//
// The aggregate result is equivalent to calling Execute on a plan built from
// all repos, but backup starts as soon as the first repo is ready.
func (r *Runner) ExecuteStream(
	repos <-chan git.Repository,
	planner *Planner,
	dryRun bool,
	total *int64,
) (*ExecutionResult, error) {
	result := &ExecutionResult{
		StartTime:   time.Now(),
		RepoResults: make([]RepoResult, 0),
	}

	type streamJob struct {
		sequence int
		repoPlan RepoPlan
	}
	type streamOutput struct {
		sequence int
		result   RepoResult
	}

	jobs := make(chan streamJob, 32)
	outputs := make(chan streamOutput, 32)

	workers := r.pushWorkerCount(0)
	var workersWG sync.WaitGroup
	for w := 0; w < workers; w++ {
		workersWG.Add(1)
		go func() {
			defer workersWG.Done()
			for job := range jobs {
				tot := int(atomic.LoadInt64(total))
				if tot <= 0 {
					tot = job.sequence
				}

				var repoResult RepoResult
				if dryRun {
					repoResult = RepoResult{
						Path:    job.repoPlan.Repo.Path,
						Success: true,
						Actions: job.repoPlan.Actions,
					}
					for _, action := range job.repoPlan.Actions {
						if action.Type == ActionAutoCommit {
							warnAboutSecrets(job.repoPlan.Repo.Path)
							break
						}
					}
					r.sendProgress(Progress{
						CurrentRepo: job.sequence,
						TotalRepos:  tot,
						RepoName:    job.repoPlan.Repo.Name,
						Action:      "[DRY RUN] Would execute actions",
						Status:      StatusSuccess,
					})
				} else {
					repoResult = r.executeRepo(job.repoPlan, job.sequence, tot)
				}

				outputs <- streamOutput{sequence: job.sequence, result: repoResult}
			}
		}()
	}

	sequence := 0
	planErrors := 0
	// Keep final results in discovery order while worker completion is concurrent.
	indexedResults := make(map[int]RepoResult)
	var resultsMu sync.Mutex

	var collectorWG sync.WaitGroup
	collectorWG.Add(1)
	go func() {
		defer collectorWG.Done()
		for output := range outputs {
			// Single map used by planning-error path and worker-output path.
			resultsMu.Lock()
			indexedResults[output.sequence] = output.result
			resultsMu.Unlock()
		}
	}()

	for repo := range repos {
		if !repo.Selected {
			continue
		}
		sequence++

		repoPlan, err := planner.BuildRepoPlan(repo)
		if err != nil {
			// Log and skip repos that can't be planned rather than aborting
			// the whole run — in an emergency, back up as much as possible.
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", repo.Path, err)

			// Create a failed RepoResult for this repo
			planErrors++
			failedResult := RepoResult{
				Path:    repo.Path,
				Success: false,
				Error:   err,
				Actions: make([]Action, 0),
			}
			resultsMu.Lock()
			indexedResults[sequence] = failedResult
			resultsMu.Unlock()

			// Send progress update
			tot := int(atomic.LoadInt64(total))
			if tot <= 0 {
				tot = sequence
			}
			r.sendProgress(Progress{
				CurrentRepo: sequence,
				TotalRepos:  tot,
				RepoName:    repo.Name,
				Action:      "Failed to build plan",
				Status:      StatusFailed,
				Error:       err,
			})

			continue
		}
		repoPlan.Repo.Selected = true
		jobs <- streamJob{sequence: sequence, repoPlan: repoPlan}
	}
	close(jobs)
	workersWG.Wait()
	close(outputs)
	collectorWG.Wait()

	keys := make([]int, 0, len(indexedResults))
	for k := range indexedResults {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	result.RepoResults = make([]RepoResult, 0, len(keys))
	for _, k := range keys {
		result.RepoResults = append(result.RepoResults, indexedResults[k])
	}
	r.aggregateResultCounts(result)

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	// Return aggregated plan errors if any occurred
	var returnErr error
	if planErrors > 0 {
		returnErr = fmt.Errorf("%d repo(s) failed during plan building", planErrors)
	}

	return result, returnErr
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

		// Warn about secrets even in dry run — the whole point of dry run is to
		// surface issues before they become real commits.
		for _, action := range repoPlan.Actions {
			if action.Type == ActionAutoCommit {
				warnAboutSecrets(repoPlan.Repo.Path)
				break
			}
		}

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

func (r *Runner) pushWorkerCount(totalRepos int) int {
	workers := config.DefaultPushWorkers
	if r.config != nil && r.config.Global.PushWorkers > 0 {
		workers = r.config.Global.PushWorkers
	}
	if workers <= 0 {
		workers = config.DefaultPushWorkers
	}
	if totalRepos > 0 && workers > totalRepos {
		return totalRepos
	}
	return workers
}

func (r *Runner) aggregateResultCounts(result *ExecutionResult) {
	result.Success = 0
	result.Failed = 0
	result.Skipped = 0
	result.TotalActions = 0
	result.FailedActions = 0

	for _, repoResult := range result.RepoResults {
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

// warnAboutSecrets scans uncommitted files for secrets and prints warnings to stderr.
func warnAboutSecrets(repoPath string) {
	if uncommitted, scanErr := git.GetUncommittedFiles(repoPath); scanErr == nil && len(uncommitted) > 0 {
		scanner := safety.NewSecretScanner()
		if suspicious, scanErr := scanner.ScanFiles(repoPath, uncommitted); scanErr == nil && len(suspicious) > 0 {
			fmt.Fprint(os.Stderr, safety.FormatWarning(suspicious))
		}
	}
}
