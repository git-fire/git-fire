package git

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// RainOptions controls how rain/hydrate updates local branches from remotes.
type RainOptions struct {
	// RiskyMode allows destructive realignment of local-only commits after
	// creating a backup branch.
	RiskyMode bool
}

const (
	RainOutcomeUpdated                = "updated"
	RainOutcomeUpdatedRisky           = "updated-risky"
	RainOutcomeUpToDate               = "up-to-date"
	RainOutcomeSkippedNoUpstream      = "skipped-no-upstream"
	RainOutcomeSkippedUpstreamMissing = "skipped-upstream-missing"
	RainOutcomeSkippedCheckedOut      = "skipped-checked-out"
	RainOutcomeSkippedLocalAhead      = "skipped-local-ahead"
	RainOutcomeSkippedDiverged        = "skipped-diverged"
	RainOutcomeSkippedUnsafeMerge     = "skipped-unsafe-merge"
	RainOutcomeSkippedUnsafeDirty     = "skipped-unsafe-dirty"
	RainOutcomeFailed                 = "failed"
)

// RainBranchResult reports the outcome for one local branch.
type RainBranchResult struct {
	Branch       string
	Upstream     string
	Current      bool
	Outcome      string
	Message      string
	BackupBranch string
}

// RainResult reports per-repo rain/hydrate outcomes.
type RainResult struct {
	RepoPath string
	Branches []RainBranchResult
	Updated  int
	Skipped  int
	Failed   int
}

type localBranchTracking struct {
	Branch   string
	Upstream string
	Current  bool
}

// RainRepository updates local branches toward their upstream refs while
// preserving worktree/index safety by default.
func RainRepository(repoPath string, opts RainOptions) (RainResult, error) {
	result := RainResult{
		RepoPath: repoPath,
		Branches: make([]RainBranchResult, 0),
	}

	branches, err := listLocalBranchesWithUpstream(repoPath)
	if err != nil {
		return result, err
	}
	if len(branches) == 0 {
		return result, nil
	}

	// Refresh remote-tracking refs before we compare branch ancestry.
	hasRemote, err := repoHasAnyRemote(repoPath)
	if err != nil {
		return result, err
	}
	if hasRemote {
		cmd := exec.Command("git", "fetch", "--all", "--prune")
		cmd.Dir = repoPath
		if output, fetchErr := cmd.CombinedOutput(); fetchErr != nil {
			return result, commandError("git fetch --all --prune", fetchErr, output)
		}
	}

	checkedOut, err := checkedOutBranchesByWorktree(repoPath)
	if err != nil {
		return result, err
	}

	hasStaged, err := HasStagedChanges(repoPath)
	if err != nil {
		return result, fmt.Errorf("check staged changes: %w", err)
	}
	hasUnstaged, err := HasUnstagedChanges(repoPath)
	if err != nil {
		return result, fmt.Errorf("check unstaged changes: %w", err)
	}
	currentDirty := hasStaged || hasUnstaged

	for _, branch := range branches {
		entry := RainBranchResult{
			Branch:   branch.Branch,
			Upstream: branch.Upstream,
			Current:  branch.Current,
		}

		record := func(outcome, message, backupBranch string) {
			entry.Outcome = outcome
			entry.Message = message
			entry.BackupBranch = backupBranch
			result.Branches = append(result.Branches, entry)
			switch outcome {
			case RainOutcomeUpdated, RainOutcomeUpdatedRisky:
				result.Updated++
			case RainOutcomeFailed:
				result.Failed++
			default:
				result.Skipped++
			}
		}

		if branch.Upstream == "" {
			record(RainOutcomeSkippedNoUpstream, "branch has no upstream", "")
			continue
		}

		localSHA, localErr := getCommitSHA(repoPath, branch.Branch)
		if localErr != nil {
			record(RainOutcomeFailed, fmt.Sprintf("read local SHA: %v", localErr), "")
			continue
		}

		upstreamSHA, upstreamErr := getCommitSHA(repoPath, branch.Upstream)
		if upstreamErr != nil {
			record(RainOutcomeSkippedUpstreamMissing, "upstream ref missing after fetch", "")
			continue
		}

		localAhead, remoteAhead, divergeErr := branchDivergenceCounts(repoPath, branch.Branch, branch.Upstream)
		if divergeErr != nil {
			record(RainOutcomeFailed, fmt.Sprintf("compute divergence: %v", divergeErr), "")
			continue
		}

		if localAhead == 0 && remoteAhead == 0 {
			record(RainOutcomeUpToDate, "already up-to-date", "")
			continue
		}

		checkedOutElsewhere := !branch.Current && checkedOut[branch.Branch]

		if localAhead == 0 && remoteAhead > 0 {
			if branch.Current {
				mergeCmd := exec.Command("git", "merge", "--ff-only", branch.Upstream)
				mergeCmd.Dir = repoPath
				if output, mergeErr := mergeCmd.CombinedOutput(); mergeErr != nil {
					record(RainOutcomeSkippedUnsafeMerge, fmt.Sprintf("cannot fast-forward current branch safely: %s", strings.TrimSpace(string(output))), "")
					continue
				}
				record(RainOutcomeUpdated, "fast-forwarded current branch", "")
				continue
			}

			if checkedOutElsewhere {
				record(RainOutcomeSkippedCheckedOut, "branch is checked out in another worktree", "")
				continue
			}

			if updateErr := updateBranchRef(repoPath, branch.Branch, localSHA, upstreamSHA); updateErr != nil {
				record(RainOutcomeFailed, fmt.Sprintf("fast-forward ref update failed: %v", updateErr), "")
				continue
			}
			record(RainOutcomeUpdated, "fast-forwarded local ref", "")
			continue
		}

		if !opts.RiskyMode {
			if localAhead > 0 && remoteAhead == 0 {
				record(RainOutcomeSkippedLocalAhead, "local branch is ahead of upstream; risky mode disabled", "")
				continue
			}
			record(RainOutcomeSkippedDiverged, "branch diverged from upstream; risky mode disabled", "")
			continue
		}

		if branch.Current {
			if currentDirty {
				record(RainOutcomeSkippedUnsafeDirty, "current branch has staged or unstaged changes", "")
				continue
			}
			backupBranch, backupErr := createRainBackupBranch(repoPath, branch.Branch, localSHA)
			if backupErr != nil {
				record(RainOutcomeFailed, fmt.Sprintf("create backup branch failed: %v", backupErr), "")
				continue
			}
			resetCmd := exec.Command("git", "reset", "--hard", branch.Upstream)
			resetCmd.Dir = repoPath
			if output, resetErr := resetCmd.CombinedOutput(); resetErr != nil {
				record(RainOutcomeFailed, fmt.Sprintf("hard reset failed: %s", strings.TrimSpace(string(output))), backupBranch)
				continue
			}
			record(RainOutcomeUpdatedRisky, "realigned current branch to upstream via hard reset", backupBranch)
			continue
		}

		if checkedOutElsewhere {
			record(RainOutcomeSkippedCheckedOut, "branch is checked out in another worktree", "")
			continue
		}
		backupBranch, backupErr := createRainBackupBranch(repoPath, branch.Branch, localSHA)
		if backupErr != nil {
			record(RainOutcomeFailed, fmt.Sprintf("create backup branch failed: %v", backupErr), "")
			continue
		}
		if updateErr := updateBranchRef(repoPath, branch.Branch, localSHA, upstreamSHA); updateErr != nil {
			record(RainOutcomeFailed, fmt.Sprintf("destructive ref update failed: %v", updateErr), backupBranch)
			continue
		}
		record(RainOutcomeUpdatedRisky, "realigned local ref to upstream", backupBranch)
	}

	return result, nil
}

func repoHasAnyRemote(repoPath string) (bool, error) {
	cmd := exec.Command("git", "remote")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, commandError("git remote", err, out)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func listLocalBranchesWithUpstream(repoPath string) ([]localBranchTracking, error) {
	currentBranch, currentErr := GetCurrentBranch(repoPath)
	if currentErr != nil {
		// Detached HEAD is acceptable for this operation; no branch is "current".
		currentBranch = ""
	}

	cmd := exec.Command("git", "for-each-ref", "refs/heads", "--format=%(refname:short)%09%(upstream:short)")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, commandError("git for-each-ref", err, out)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	branches := make([]localBranchTracking, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		branch := strings.TrimSpace(parts[0])
		upstream := ""
		if len(parts) == 2 {
			upstream = strings.TrimSpace(parts[1])
		}
		branches = append(branches, localBranchTracking{
			Branch:   branch,
			Upstream: upstream,
			Current:  branch != "" && branch == currentBranch,
		})
	}
	return branches, nil
}

func checkedOutBranchesByWorktree(repoPath string) (map[string]bool, error) {
	worktrees, err := ListWorktrees(repoPath)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(worktrees))
	for _, wt := range worktrees {
		if wt.Branch != "" {
			out[wt.Branch] = true
		}
	}
	return out, nil
}

func branchDivergenceCounts(repoPath, localRef, upstreamRef string) (int, int, error) {
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", localRef+"..."+upstreamRef)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, commandError("git rev-list --left-right --count", err, out)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", strings.TrimSpace(string(out)))
	}
	localAhead, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse local-ahead count: %w", err)
	}
	remoteAhead, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, 0, fmt.Errorf("parse remote-ahead count: %w", err)
	}
	return localAhead, remoteAhead, nil
}

func updateBranchRef(repoPath, branch, oldSHA, newSHA string) error {
	cmd := exec.Command("git", "update-ref", "refs/heads/"+branch, newSHA, oldSHA)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return commandError("git update-ref", err, out)
	}
	return nil
}

func createRainBackupBranch(repoPath, branch, sha string) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	shortSHA := sha
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}
	branchSafe := strings.ReplaceAll(branch, "/", "-")
	backupBranch := fmt.Sprintf("git-fire-rain-backup-%s-%s-%s", branchSafe, timestamp, shortSHA)
	cmd := exec.Command("git", "branch", backupBranch, sha)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", commandError("git branch "+backupBranch, err, out)
	}
	return backupBranch, nil
}
