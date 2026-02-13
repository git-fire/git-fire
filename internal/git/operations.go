package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommitOptions configures auto-commit behavior
type CommitOptions struct {
	Message string // Commit message
	AddAll  bool   // Run git add -A (default: true)
}

// AutoCommitDirty commits all uncommitted changes in a repo
// Returns nil if repo is already clean
func AutoCommitDirty(repoPath string, opts CommitOptions) error {
	// Check if repo is dirty
	isDirty, err := IsDirty(repoPath)
	if err != nil {
		return fmt.Errorf("failed to check repo status: %w", err)
	}

	if !isDirty {
		// Repo is clean, nothing to do
		return nil
	}

	// Add all changes (respects .gitignore)
	addAll := opts.AddAll
	if !addAll {
		addAll = true // Default to adding all
	}

	if addAll {
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git add failed: %w\nOutput: %s", err, output)
		}
	}

	// Commit with message
	message := opts.Message
	if message == "" {
		message = fmt.Sprintf("git-fire emergency backup - %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// IsDirty checks if a repo has uncommitted changes
func IsDirty(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}

	return len(output) > 0, nil
}

// DetectConflict checks if local and remote branches have diverged
// Returns: hasConflict, localSHA, remoteSHA, error
func DetectConflict(repoPath, branch, remote string) (bool, string, string, error) {
	// Fetch from remote to get latest refs
	cmd := exec.Command("git", "fetch", remote)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return false, "", "", fmt.Errorf("git fetch failed: %w\nOutput: %s", err, output)
	}

	// Get local SHA
	localSHA, err := getCommitSHA(repoPath, branch)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get local SHA: %w", err)
	}

	// Get remote SHA
	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	remoteSHA, err := getCommitSHA(repoPath, remoteBranch)
	if err != nil {
		// Remote branch might not exist yet
		return false, localSHA, "", nil
	}

	// Check if they differ
	hasConflict := localSHA != remoteSHA

	return hasConflict, localSHA, remoteSHA, nil
}

// getCommitSHA returns the SHA of a commit ref
func getCommitSHA(repoPath, ref string) (string, error) {
	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed for %s: %w", ref, err)
	}

	sha := strings.TrimSpace(string(output))
	return sha, nil
}

// CreateFireBranch creates a new fire backup branch
// Returns the new branch name
func CreateFireBranch(repoPath, originalBranch, localSHA string) (string, error) {
	// Generate unique branch name
	// Format: git-fire-backup-{branch}-{timestamp}-{short-sha}
	timestamp := time.Now().Format("20060102-150405")
	shortSHA := localSHA
	if len(shortSHA) > 7 {
		shortSHA = shortSHA[:7]
	}

	branchName := fmt.Sprintf("git-fire-backup-%s-%s-%s", originalBranch, timestamp, shortSHA)

	// Create branch
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create fire branch: %w\nOutput: %s", err, output)
	}

	return branchName, nil
}

// PushBranch pushes a specific branch to a remote
func PushBranch(repoPath, remote, branch string) error {
	cmd := exec.Command("git", "push", remote, branch)
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push failed: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}

// PushAllBranches pushes all branches to a remote
func PushAllBranches(repoPath, remote string) error {
	cmd := exec.Command("git", "push", remote, "--all")
	cmd.Dir = repoPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git push --all failed: %w\nStderr: %s", err, stderr.String())
	}

	return nil
}

// PushKnownBranches pushes only branches that exist on the remote
func PushKnownBranches(repoPath, remote string) error {
	// Fetch to update remote refs
	cmd := exec.Command("git", "fetch", remote)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %w\nOutput: %s", err, output)
	}

	// Get list of remote branches
	remoteBranches, err := getRemoteBranches(repoPath, remote)
	if err != nil {
		return fmt.Errorf("failed to get remote branches: %w", err)
	}

	// Get local branches
	localBranches, err := getLocalBranches(repoPath)
	if err != nil {
		return fmt.Errorf("failed to get local branches: %w", err)
	}

	// Push each local branch that exists on remote
	for _, localBranch := range localBranches {
		// Check if this branch exists on remote
		exists := false
		for _, remoteBranch := range remoteBranches {
			if remoteBranch == localBranch {
				exists = true
				break
			}
		}

		if exists {
			if err := PushBranch(repoPath, remote, localBranch); err != nil {
				return fmt.Errorf("failed to push branch %s: %w", localBranch, err)
			}
		}
	}

	return nil
}

// getRemoteBranches returns list of branches on a remote
func getRemoteBranches(repoPath, remote string) ([]string, error) {
	cmd := exec.Command("git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch -r failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string

	prefix := remote + "/"
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Filter to only this remote and strip remote prefix
		if strings.HasPrefix(line, prefix) {
			branch := strings.TrimPrefix(line, prefix)
			// Skip HEAD
			if branch != "HEAD" && !strings.Contains(branch, "->") {
				branches = append(branches, branch)
			}
		}
	}

	return branches, nil
}

// getLocalBranches returns list of local branches
func getLocalBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git branch failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// GetCurrentBranch returns the currently checked out branch
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current failed: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("not on any branch (detached HEAD?)")
	}

	return branch, nil
}
