package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/git-fire/git-fire/internal/safety"
)

// CommitOptions configures auto-commit behavior
type CommitOptions struct {
	Message           string // Commit message
	AddAll            bool   // Run git add -A; must be set explicitly (default: false)
	UseDualBranch     bool   // Use staged/unstaged dual branch strategy (default: true)
	ReturnToOriginal  bool   // Reset to original state after creating branches (default: true)
}

// AutoCommitResult contains information about branches created
type AutoCommitResult struct {
	StagedBranch string // Empty if no staged changes
	FullBranch   string // Empty if no unstaged changes
	BothCreated  bool   // True if both branches were created
}

// Worktree represents a git worktree
type Worktree struct {
	Path   string // Absolute path to worktree
	Branch string // Current branch in this worktree
	Head   string // Current HEAD SHA
	IsMain bool   // True if this is the main worktree
}

// AutoCommitDirty commits all uncommitted changes in a repo
// Returns nil if repo is already clean
func AutoCommitDirty(repoPath string, opts CommitOptions) error {
	// Check if repo is dirty first — clean repos are a no-op regardless of HEAD state
	isDirty, err := IsDirty(repoPath)
	if err != nil {
		return fmt.Errorf("failed to check repo status: %w", err)
	}

	if !isDirty {
		return nil
	}

	// Refuse to commit in detached HEAD — the commit would be unreachable
	if _, err := GetCurrentBranch(repoPath); err != nil {
		return fmt.Errorf("cannot auto-commit: %w", err)
	}

	// Add all changes (respects .gitignore)
	if opts.AddAll {
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return commandError("git add", err, output)
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
		return commandError("git commit", err, output)
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
		return false, "", "", commandError("git fetch", err, output)
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

	// Compare ancestry, not just tip inequality:
	// - merge-base == remoteSHA => local is only ahead (no divergence)
	// - merge-base == localSHA  => local is behind (no divergence)
	// - otherwise               => true divergence (conflict)
	mergeBaseSHA, hasMergeBase, err := getMergeBaseSHA(repoPath, branch, remoteBranch)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get merge-base: %w", err)
	}
	if !hasMergeBase {
		// No common ancestor means true divergence for conflict-handling purposes.
		return true, localSHA, remoteSHA, nil
	}
	hasConflict := mergeBaseSHA != remoteSHA && mergeBaseSHA != localSHA

	return hasConflict, localSHA, remoteSHA, nil
}

// DetectConflictAfterFetch checks whether the local branch has diverged from the
// remote tracking branch using refs already present in the repo (no fetch).
// Call this only after refs for the remote are up to date (for example after
// DetectConflict or a prior fetch).
func DetectConflictAfterFetch(repoPath, branch, remote string) (bool, string, string, error) {
	localSHA, err := getCommitSHA(repoPath, branch)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get local SHA: %w", err)
	}

	remoteBranch := fmt.Sprintf("%s/%s", remote, branch)
	remoteSHA, err := getCommitSHA(repoPath, remoteBranch)
	if err != nil {
		return false, localSHA, "", nil
	}

	mergeBaseSHA, hasMergeBase, err := getMergeBaseSHA(repoPath, branch, remoteBranch)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get merge-base: %w", err)
	}
	if !hasMergeBase {
		return true, localSHA, remoteSHA, nil
	}
	hasConflict := mergeBaseSHA != remoteSHA && mergeBaseSHA != localSHA

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

// GetCommitSHA returns the SHA for a ref in the repository.
func GetCommitSHA(repoPath, ref string) (string, error) {
	return getCommitSHA(repoPath, ref)
}

func getMergeBaseSHA(repoPath, leftRef, rightRef string) (string, bool, error) {
	cmd := exec.Command("git", "merge-base", leftRef, rightRef)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", false, nil
		}
		return "", false, commandError("git merge-base", err, output)
	}
	return strings.TrimSpace(string(output)), true, nil
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
		return "", commandError("git branch "+branchName, err, output)
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
		return commandError("git push", err, stderr.Bytes())
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
		return commandError("git push --all", err, stderr.Bytes())
	}

	return nil
}

// PushKnownBranches pushes only branches that exist on the remote
func PushKnownBranches(repoPath, remote string) error {
	// Fetch to update remote refs
	cmd := exec.Command("git", "fetch", remote)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git fetch", err, output)
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

	// Push each local branch that exists on remote, collecting errors so all
	// branches are attempted even if one fails.
	var errs []error
	for _, localBranch := range localBranches {
		exists := false
		for _, remoteBranch := range remoteBranches {
			if remoteBranch == localBranch {
				exists = true
				break
			}
		}

		if exists {
			if err := PushBranch(repoPath, remote, localBranch); err != nil {
				errs = append(errs, fmt.Errorf("branch %s: %w", localBranch, err))
			}
			continue
		}

		fmt.Fprintf(os.Stderr, "warning: branch '%s' has no remote tracking ref — not backed up\n", localBranch)
	}

	return errors.Join(errs...)
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

// HasStagedChanges checks if there are staged changes
func HasStagedChanges(repoPath string) (bool, error) {
	cmd := exec.Command("git", "diff", "--cached", "--quiet")
	cmd.Dir = repoPath

	err := cmd.Run()
	if err != nil {
		// Exit code 1 means there are differences (staged changes exist)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return true, nil
		}
		return false, fmt.Errorf("git diff --cached --quiet failed: %w", err)
	}

	// Exit code 0 means no differences (no staged changes)
	return false, nil
}

// HasUnstagedChanges checks if there are unstaged changes (including untracked files)
func HasUnstagedChanges(repoPath string) (bool, error) {
	// Check for modified files
	cmd := exec.Command("git", "diff", "--quiet")
	cmd.Dir = repoPath

	err := cmd.Run()
	hasModified := false
	if err != nil {
		// Exit code 1 means there are differences
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			hasModified = true
		} else {
			return false, fmt.Errorf("git diff --quiet failed: %w", err)
		}
	}

	// Check for untracked files
	cmd = exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git ls-files failed: %w", err)
	}

	hasUntracked := len(strings.TrimSpace(string(output))) > 0

	return hasModified || hasUntracked, nil
}

// ListWorktrees returns all worktrees for a repository
func ListWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var worktrees []Worktree
	var current Worktree
	isFirst := true

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			// Empty line separates worktrees
			if current.Path != "" {
				current.IsMain = isFirst
				worktrees = append(worktrees, current)
				current = Worktree{}
				isFirst = false
			}
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "worktree":
			current.Path = value
		case "HEAD":
			current.Head = value
		case "branch":
			// Format: refs/heads/main
			branch := strings.TrimPrefix(value, "refs/heads/")
			current.Branch = branch
		}
	}

	// Add last worktree if exists
	if current.Path != "" {
		current.IsMain = isFirst
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// AutoCommitDirtyWithStrategy commits changes using the staged/unstaged dual branch strategy
// Returns information about branches created and any error
func AutoCommitDirtyWithStrategy(repoPath string, opts CommitOptions) (result *AutoCommitResult, retErr error) {
	result = &AutoCommitResult{}

	// Set defaults
	if !opts.UseDualBranch {
		opts.UseDualBranch = true
	}
	if !opts.ReturnToOriginal {
		opts.ReturnToOriginal = true
	}

	// Get current branch
	currentBranch, err := GetCurrentBranch(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Capture original HEAD once and use it for all cleanup/reset paths.
	// Repos with no commits (unborn HEAD) are valid and handled explicitly.
	originalHeadSHA, hasOriginalHead, err := getOptionalHeadSHA(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get original HEAD SHA: %w", err)
	}

	// Check for staged and unstaged changes
	hasStaged, err := HasStagedChanges(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check staged changes: %w", err)
	}

	hasUnstaged, err := HasUnstagedChanges(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check unstaged changes: %w", err)
	}

	// If nothing to commit, return early
	if !hasStaged && !hasUnstaged {
		return result, nil
	}

	originalIndexTree, err := captureIndexTree(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to capture original index state: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")
	commitsCreated := 0
	successRestoreAttempted := false

	// Failure path cleanup: always reset to the original commit to avoid orphan
	// backup commits, even when multiple commits were created.
	defer func() {
		if retErr == nil || commitsCreated == 0 || successRestoreAttempted {
			return
		}
		if cleanupErr := restoreOriginalState(repoPath, hasOriginalHead, originalHeadSHA, originalIndexTree); cleanupErr != nil {
			retErr = fmt.Errorf("%w; failed cleanup restore of original staged/unstaged state: %v", retErr, cleanupErr)
		}
	}()

	// Scenario 1: Only staged changes
	if hasStaged && !hasUnstaged {
		// Commit staged changes
		message := opts.Message
		if message == "" {
			message = fmt.Sprintf("git-fire staged backup - %s", time.Now().Format("2006-01-02 15:04:05"))
		}

		if err := commitChanges(repoPath, message, false); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to commit staged changes: %w", err))
		}
		commitsCreated++

		// Get SHA and create branch
		sha, err := getCommitSHA(repoPath, "HEAD")
		if err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to get commit SHA: %w", err))
		}
		shortSHA := sha
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		branchName := fmt.Sprintf("git-fire-staged-%s-%s-%s", currentBranch, timestamp, shortSHA)
		if err := createBranch(repoPath, branchName); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to create staged branch: %w", err))
		}

		result.StagedBranch = branchName
	}

	// Scenario 2: Only unstaged changes
	if !hasStaged && hasUnstaged {
		// Add and commit all changes
		message := opts.Message
		if message == "" {
			message = fmt.Sprintf("git-fire full backup - %s", time.Now().Format("2006-01-02 15:04:05"))
		}

		if err := commitChanges(repoPath, message, true); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to commit unstaged changes: %w", err))
		}
		commitsCreated++

		// Get SHA and create branch
		sha, err := getCommitSHA(repoPath, "HEAD")
		if err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to get commit SHA: %w", err))
		}
		shortSHA := sha
		if len(shortSHA) > 7 {
			shortSHA = shortSHA[:7]
		}

		branchName := fmt.Sprintf("git-fire-full-%s-%s-%s", currentBranch, timestamp, shortSHA)
		if err := createBranch(repoPath, branchName); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to create full branch: %w", err))
		}

		result.FullBranch = branchName
	}

	// Scenario 3: Both staged and unstaged changes
	if hasStaged && hasUnstaged {
		// Step 1: Commit staged changes
		message1 := opts.Message
		if message1 == "" {
			message1 = fmt.Sprintf("git-fire staged backup - %s", time.Now().Format("2006-01-02 15:04:05"))
		}

		if err := commitChanges(repoPath, message1, false); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to commit staged changes: %w", err))
		}
		commitsCreated++

		// Create staged branch
		sha1, err := getCommitSHA(repoPath, "HEAD")
		if err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to get staged commit SHA: %w", err))
		}
		shortSHA1 := sha1
		if len(shortSHA1) > 7 {
			shortSHA1 = shortSHA1[:7]
		}

		stagedBranchName := fmt.Sprintf("git-fire-staged-%s-%s-%s", currentBranch, timestamp, shortSHA1)
		if err := createBranch(repoPath, stagedBranchName); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to create staged branch: %w", err))
		}
		result.StagedBranch = stagedBranchName

		// Step 2: Add and commit unstaged changes (on top of staged)
		message2 := opts.Message
		if message2 == "" {
			message2 = fmt.Sprintf("git-fire full backup - %s", time.Now().Format("2006-01-02 15:04:05"))
		}

		if err := commitChanges(repoPath, message2, true); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to commit unstaged changes: %w", err))
		}
		commitsCreated++

		// Create full branch
		sha2, err := getCommitSHA(repoPath, "HEAD")
		if err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to get full commit SHA: %w", err))
		}
		shortSHA2 := sha2
		if len(shortSHA2) > 7 {
			shortSHA2 = shortSHA2[:7]
		}

		fullBranchName := fmt.Sprintf("git-fire-full-%s-%s-%s", currentBranch, timestamp, shortSHA2)
		if err := createBranch(repoPath, fullBranchName); err != nil {
			return returnResultOnError(result, fmt.Errorf("failed to create full branch: %w", err))
		}
		result.FullBranch = fullBranchName
		result.BothCreated = true
	}

	// Success path restore: preserve original staged/unstaged shape.
	if opts.ReturnToOriginal && commitsCreated > 0 {
		successRestoreAttempted = true
		if err := restoreOriginalState(repoPath, hasOriginalHead, originalHeadSHA, originalIndexTree); err != nil {
			return returnResultOnError(result, err)
		}
	}

	return result, nil
}

func resetMixedToCommit(repoPath, sha string) error {
	cmd := exec.Command("git", "reset", "--mixed", sha)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git reset --mixed", err, output)
	}
	return nil
}

func resetToUnborn(repoPath string) error {
	branch, err := GetCurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("resolve current branch for unborn reset: %w", err)
	}
	ref := fmt.Sprintf("refs/heads/%s", branch)
	cmd := exec.Command("git", "update-ref", "-d", ref)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git update-ref -d "+ref, err, output)
	}
	return nil
}

func clearIndexForUnborn(repoPath string) error {
	cmd := exec.Command("git", "read-tree", "--empty")
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git read-tree --empty", err, output)
	}
	return nil
}

func captureIndexTree(repoPath string) (string, error) {
	cmd := exec.Command("git", "write-tree")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", commandError("git write-tree", err, output)
	}
	tree := strings.TrimSpace(string(output))
	if tree == "" {
		return "", fmt.Errorf("git write-tree returned empty tree SHA")
	}
	return tree, nil
}

func restoreIndexTree(repoPath, treeSHA string) error {
	cmd := exec.Command("git", "read-tree", treeSHA)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git read-tree "+treeSHA, err, output)
	}
	return nil
}

func restoreOriginalState(repoPath string, hasOriginalHead bool, originalHeadSHA, originalIndexTree string) error {
	if hasOriginalHead {
		if err := resetMixedToCommit(repoPath, originalHeadSHA); err != nil {
			return err
		}
	} else {
		if err := resetToUnborn(repoPath); err != nil {
			return err
		}
		if err := clearIndexForUnborn(repoPath); err != nil {
			return err
		}
	}
	return restoreIndexTree(repoPath, originalIndexTree)
}

func returnResultOnError(result *AutoCommitResult, err error) (*AutoCommitResult, error) {
	if result != nil && (result.StagedBranch != "" || result.FullBranch != "" || result.BothCreated) {
		return result, err
	}
	return nil, err
}

func getOptionalHeadSHA(repoPath string) (string, bool, error) {
	cmd := exec.Command("git", "rev-parse", "-q", "--verify", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", false, nil
		}
		return "", false, commandError("git rev-parse -q --verify HEAD", err, output)
	}

	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", false, fmt.Errorf("git rev-parse -q --verify HEAD returned empty output")
	}
	return sha, true, nil
}

// commitChanges commits changes with optional git add -A
func commitChanges(repoPath, message string, addAll bool) error {
	if addAll {
		cmd := exec.Command("git", "add", "-A")
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return commandError("git add", err, output)
		}
	}

	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git commit", err, output)
	}

	return nil
}

// GetUncommittedFiles returns the relative paths of all files that would be
// staged by git add -A — modified, added, deleted, and untracked files that
// are not excluded by .gitignore.
//
// Uses --porcelain -z (NUL-delimited) to avoid the quoting that git applies
// to filenames containing spaces or special characters in plain porcelain output.
func GetUncommittedFiles(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain", "-z")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []string
	// -z output: entries are NUL-terminated; renames produce two NUL-separated tokens
	entries := strings.Split(string(output), "\x00")
	i := 0
	for i < len(entries) {
		entry := entries[i]
		i++
		if len(entry) < 4 {
			continue
		}
		xy := entry[:2]
		path := entry[3:] // skip "XY " prefix
		// Rename/copy: git status -z gives "XY new_path\0old_path\0"
		// entry[3:] already has the new (destination) path; skip the old path token.
		if (xy[0] == 'R' || xy[0] == 'C') && i < len(entries) {
			i++ // consume old path token — path is already set to the new path
		}
		if path != "" {
			files = append(files, path)
		}
	}
	return files, nil
}

// createBranch creates a new branch at the current HEAD
func createBranch(repoPath, branchName string) error {
	cmd := exec.Command("git", "branch", branchName)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return commandError("git branch "+branchName, err, output)
	}
	return nil
}

func commandError(action string, err error, output []byte) error {
	out := strings.TrimSpace(string(output))
	if out == "" {
		return fmt.Errorf("%s failed: %w", action, err)
	}
	return fmt.Errorf("%s failed: %w (stderr: %s)", action, err, safety.SanitizeText(out))
}
