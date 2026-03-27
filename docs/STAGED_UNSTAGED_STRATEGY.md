# Staged vs Unstaged Branch Strategy

## Problem

When a repository has both staged and unstaged changes, the current `git add -A` approach loses the semantic distinction between:
- **Staged changes**: What the user explicitly prepared for commit (intentional)
- **Unstaged changes**: Work-in-progress that might not compile/run

## Solution: Dual Branch Creation

Create **two separate branches** to preserve both states:

### Branch 1: Staged Only
```
git-fire-staged-{branch}-{timestamp}-{sha}
```
- Contains: Only the changes that were staged
- Purpose: Safe, intentional changes that likely build/run
- Created by: `git commit` (without adding anything)

### Branch 2: Full Backup
```
git-fire-full-{branch}-{timestamp}-{sha}
```
- Contains: Staged + unstaged changes
- Purpose: Complete snapshot, nothing lost
- Created by: `git add -A && git commit`
- **Based on Branch 1** (inherits the staged commit)

## Scenarios

### Scenario 1: Only Staged Changes
```bash
# State: Files in staging area, no unstaged changes
git diff --cached --quiet  # exit 1 (has staged)
git diff --quiet           # exit 0 (no unstaged)

# Action: Create one branch
git commit -m "git-fire staged backup"
git branch git-fire-staged-main-20260213-143052-a1b2c3d
```
**Result:** 1 branch created

### Scenario 2: Only Unstaged Changes
```bash
# State: Modified files, nothing staged
git diff --cached --quiet  # exit 0 (no staged)
git diff --quiet           # exit 1 (has unstaged)

# Action: Create one full branch
git add -A
git commit -m "git-fire emergency backup"
git branch git-fire-full-main-20260213-143052-a1b2c3d
```
**Result:** 1 branch created

### Scenario 3: Both Staged and Unstaged Changes ⭐
```bash
# State: Some files staged, other files unstaged
git diff --cached --quiet  # exit 1 (has staged)
git diff --quiet           # exit 1 (has unstaged)

# Action: Create TWO branches
# Step 1: Commit staged changes
git commit -m "git-fire staged backup - 2026-02-13 14:30:52"
SHA1=$(git rev-parse HEAD)
git branch git-fire-staged-main-20260213-143052-${SHA1:0:7}

# Step 2: Add unstaged and commit (on top of staged)
git add -A
git commit -m "git-fire full backup - 2026-02-13 14:30:52"
SHA2=$(git rev-parse HEAD)
git branch git-fire-full-main-20260213-143052-${SHA2:0:7}

# Step 3: Return to original branch
git reset --soft HEAD~2  # Keep changes but undo commits
```
**Result:** 2 branches created (full branch is child of staged branch)

### Scenario 4: Clean Repository
```bash
# State: No changes at all
git diff --cached --quiet  # exit 0
git diff --quiet           # exit 0

# Action: Nothing to commit
```
**Result:** 0 branches created

## Implementation

### New Function Signature
```go
// AutoCommitResult contains information about branches created
type AutoCommitResult struct {
    StagedBranch string // Empty if no staged changes
    FullBranch   string // Empty if no unstaged changes
    BothCreated  bool   // True if both branches were created
}

// AutoCommitDirtyWithStrategy commits changes using the staged/unstaged strategy
// Returns branch names created and any error
func AutoCommitDirtyWithStrategy(repoPath string, opts CommitOptions) (*AutoCommitResult, error)
```

### Detection Functions
```go
// HasStagedChanges checks if there are staged changes
func HasStagedChanges(repoPath string) (bool, error) {
    // git diff --cached --quiet
    // Returns true if exit code != 0
}

// HasUnstagedChanges checks if there are unstaged changes
func HasUnstagedChanges(repoPath string) (bool, error) {
    // git diff --quiet
    // Returns true if exit code != 0
}
```

### Branch Naming
```
git-fire-staged-{branch}-{timestamp}-{sha}
git-fire-full-{branch}-{timestamp}-{sha}
```

**Format:**
- `{branch}`: Current branch name (e.g., "main", "feature/auth")
- `{timestamp}`: YYYYMMdd-HHmmss (e.g., "20260213-143052")
- `{sha}`: First 7 chars of commit SHA

**Examples:**
```
git-fire-staged-main-20260213-143052-a1b2c3d
git-fire-full-main-20260213-143052-e4f5a6b
```

## Worktree Support

Git worktrees are separate working directories attached to the same repository.

### Detection
```bash
git worktree list --porcelain
```

**Output:**
```
worktree /home/user/project
HEAD a1b2c3d4e5f6
branch refs/heads/main

worktree /home/user/project-feature
HEAD f1e2d3c4b5a6
branch refs/heads/feature/auth
```

### Strategy
Each worktree is **independent** for git-fire purposes:
- Scan each worktree separately
- Each can be in different states (staged/unstaged/clean)
- Create branches independently per worktree

### Implementation
```go
// Worktree represents a git worktree
type Worktree struct {
    Path   string // Absolute path to worktree
    Branch string // Current branch in this worktree
    Head   string // Current HEAD SHA
}

// ListWorktrees returns all worktrees for a repository
func ListWorktrees(repoPath string) ([]Worktree, error) {
    // Parse output of: git worktree list --porcelain
}
```

### Scanning Flow
```
1. Detect main repo directory (.git present)
2. List all worktrees
3. For each worktree:
   - Check staged/unstaged status
   - Create appropriate branches
   - Push branches
```

## Benefits

### 1. Semantic Preservation
Staged changes likely represent:
- Code that compiles
- Tests that pass
- Intentional, reviewed changes

Unstaged changes might be:
- Debugging console.log statements
- Experimental code
- Half-finished refactoring

**The staged branch gives you a safe fallback.**

### 2. Recovery Flexibility
After the emergency, you have options:
```bash
# Option 1: Use the staged version (likely builds)
git checkout git-fire-staged-main-20260213-143052-a1b2c3d

# Option 2: Use the full version (everything)
git checkout git-fire-full-main-20260213-143052-e4f5a6b

# Option 3: Cherry-pick specific changes
git cherry-pick git-fire-staged-main-20260213-143052-a1b2c3d
```

### 3. No Data Loss
- Staged work is preserved ✓
- Unstaged work is preserved ✓
- Original branch remains untouched ✓
- Relationship between branches is clear (full is child of staged)

## Edge Cases

### Untracked Files
Untracked files are included in the "full" backup:
```bash
git add -A  # Adds untracked files too
```

### Merge Conflicts
If a merge is in progress:
```bash
git status --porcelain | grep '^UU'  # Detect conflict
```
**Strategy:** Abort merge, commit as-is, or skip auto-commit

### Detached HEAD
```bash
git symbolic-ref HEAD  # Fails if detached
```
**Strategy:** Use HEAD SHA instead of branch name in fire branch name

### Empty Commits
If staged changes are empty (e.g., only whitespace):
```bash
git commit --allow-empty
```
**Strategy:** Allow empty commits or skip branch creation

## Testing Plan

### Unit Tests
```go
func TestAutoCommitWithStrategy_OnlyStaged(t *testing.T)
func TestAutoCommitWithStrategy_OnlyUnstaged(t *testing.T)
func TestAutoCommitWithStrategy_Both(t *testing.T)
func TestAutoCommitWithStrategy_Clean(t *testing.T)
func TestListWorktrees(t *testing.T)
func TestAutoCommitWorktree(t *testing.T)
```

### Integration Tests
- Test with real repos
- Verify branch relationships (full is child of staged)
- Verify no data loss
- Test worktree scenarios

## Migration

### Backward Compatibility
Old behavior (single branch):
```go
func AutoCommitDirty(repoPath string, opts CommitOptions) error
```

New behavior (dual branch):
```go
func AutoCommitDirtyWithStrategy(repoPath string, opts CommitOptions) (*AutoCommitResult, error)
```

**Strategy:**
1. Keep old function for now (deprecated)
2. Use new function in main flow
3. Add config option: `use_dual_branch_strategy = true` (default)

### User Communication
Update docs to explain:
- Why two branches are better
- How to recover from each
- When to use which branch

## Summary

This dual-branch strategy is **significantly better** than the original approach:

| Aspect | Old (Single Branch) | New (Dual Branch) |
|--------|---------------------|-------------------|
| Staged changes | ❌ Lost in mix | ✅ Preserved separately |
| Unstaged changes | ✅ Saved | ✅ Saved |
| Build confidence | ❓ Unknown | ✅ Staged likely builds |
| Recovery options | 1 option | 2-3 options |
| Data loss | None | None |
| Semantic meaning | ❌ Lost | ✅ Preserved |

**This is a game-changer for emergency backups!**
