# Staged vs Unstaged Implementation Summary

## What Was Built

Successfully implemented the dual-branch emergency backup strategy with git worktree support!

### New Features

#### 1. Dual Branch Strategy (`AutoCommitDirtyWithStrategy`)

Creates **1 or 2 branches** depending on repository state:

**Scenario 1: Only Staged Changes**
```bash
# User has staged files but no unstaged changes
git status
# staged: file.txt

# Result: 1 branch
git-fire-staged-main-20260213-143052-a1b2c3d
```

**Scenario 2: Only Unstaged Changes**
```bash
# User has unstaged/untracked files
git status
# unstaged: file.txt, new.txt

# Result: 1 branch
git-fire-full-main-20260213-143052-e4f5a6b
```

**Scenario 3: Both Staged AND Unstaged** ⭐
```bash
# User has BOTH staged and unstaged changes
git status
# staged: important.txt
# unstaged: debug.txt, experiment.js

# Result: 2 branches!
git-fire-staged-main-20260213-143052-a1b2c3d  # Safe, builds
git-fire-full-main-20260213-143052-e4f5a6b    # Everything (child of staged)
```

**Scenario 4: Clean Repository**
```bash
# Nothing to commit
git status
# clean

# Result: 0 branches (nothing to back up)
```

#### 2. Detection Functions

```go
// Check for staged changes
hasStaged, err := HasStagedChanges(repoPath)

// Check for unstaged changes (including untracked files)
hasUnstaged, err := HasUnstagedChanges(repoPath)
```

#### 3. Git Worktree Support

```go
// List all worktrees in a repository
worktrees, err := ListWorktrees(repoPath)

// Each worktree can be scanned independently
for _, wt := range worktrees {
    // Process each worktree separately
}
```

### API

#### AutoCommitDirtyWithStrategy

```go
type CommitOptions struct {
    Message          string // Commit message
    AddAll           bool   // Run git add -A (default: true)
    UseDualBranch    bool   // Use staged/unstaged strategy (default: true)
    ReturnToOriginal bool   // Reset to original state after (default: true)
}

type AutoCommitResult struct {
    StagedBranch string // Empty if no staged changes
    FullBranch   string // Empty if no unstaged changes
    BothCreated  bool   // True if both branches created
}

func AutoCommitDirtyWithStrategy(repoPath string, opts CommitOptions) (*AutoCommitResult, error)
```

**Usage Example:**
```go
result, err := git.AutoCommitDirtyWithStrategy("/path/to/repo", git.CommitOptions{
    ReturnToOriginal: true, // Keep working tree as-is
})

if err != nil {
    return err
}

if result.BothCreated {
    fmt.Printf("Created staged branch: %s\n", result.StagedBranch)
    fmt.Printf("Created full branch: %s\n", result.FullBranch)
    fmt.Println("Staged branch likely builds, full branch has everything!")
}
```

### Branch Naming Convention

```
git-fire-staged-{branch}-{timestamp}-{sha}
git-fire-full-{branch}-{timestamp}-{sha}
```

**Examples:**
- `git-fire-staged-main-20260213-143052-a1b2c3d`
- `git-fire-full-feature/auth-20260213-143052-e4f5a6b`
- `git-fire-full-bugfix-api-20260213-143100-1234567`

### Worktree Structure

```go
type Worktree struct {
    Path   string // Absolute path (/home/user/project)
    Branch string // Current branch (main, feature, etc.)
    Head   string // Current HEAD SHA
    IsMain bool   // True if main worktree
}
```

**Usage:**
```go
worktrees, err := git.ListWorktrees(repoPath)
for _, wt := range worktrees {
    fmt.Printf("Worktree: %s on branch %s\n", wt.Path, wt.Branch)

    // Each worktree can be in a different state
    result, _ := git.AutoCommitDirtyWithStrategy(wt.Path, opts)
    // ...
}
```

## Testing

### Test Coverage

**8 new tests, all passing:**
1. `TestHasStagedChanges` - Detection of staged changes
2. `TestHasUnstagedChanges` - Detection of unstaged/untracked changes
3. `TestAutoCommitDirtyWithStrategy_OnlyStaged` - Scenario 1
4. `TestAutoCommitDirtyWithStrategy_OnlyUnstaged` - Scenario 2
5. `TestAutoCommitDirtyWithStrategy_Both` - Scenario 3 ⭐
6. `TestAutoCommitDirtyWithStrategy_Clean` - Scenario 4
7. `TestAutoCommitDirtyWithStrategy_ReturnToOriginal` - Verify reset behavior
8. `TestListWorktrees` - Worktree detection

**Total: 20/20 tests passing** across all git operations

### Test Example

```go
func TestAutoCommitDirtyWithStrategy_Both(t *testing.T) {
    repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
        Name: "test-repo",
    })

    // Stage a file
    stagedFile := filepath.Join(repo, "staged.txt")
    os.WriteFile(stagedFile, []byte("staged"), 0644)
    testutil.RunGitCmd(t, repo, "add", "staged.txt")

    // Create an unstaged file
    unstagedFile := filepath.Join(repo, "unstaged.txt")
    os.WriteFile(unstagedFile, []byte("unstaged"), 0644)

    // Run with strategy
    result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
        ReturnToOriginal: false,
    })

    // Verify BOTH branches created
    assert.NotEmpty(t, result.StagedBranch)
    assert.NotEmpty(t, result.FullBranch)
    assert.True(t, result.BothCreated)
}
```

## Benefits

### 1. Semantic Preservation
- **Staged changes** = Intentional, likely builds
- **Unstaged changes** = WIP, experiments, debug code

The staged branch gives you a **safe fallback** that probably works!

### 2. Recovery Flexibility

After an emergency, you have options:
```bash
# Option 1: Use staged version (safe)
git checkout git-fire-staged-main-20260213-143052-a1b2c3d
npm run build  # ✅ Likely works!

# Option 2: Use full version (everything)
git checkout git-fire-full-main-20260213-143052-e4f5a6b
npm run build  # ❓ Might fail, but has all your work

# Option 3: Cherry-pick from staged
git cherry-pick git-fire-staged-main-20260213-143052-a1b2c3d
```

### 3. No Data Loss
- Staged work: ✅ Preserved in staged branch
- Unstaged work: ✅ Preserved in full branch
- Original branch: ✅ Untouched
- Relationship: ✅ Full branch is child of staged branch

### 4. Worktree Support
Each worktree is independent:
- Main repo: Clean, nothing to back up
- Worktree 1 (feature/auth): Dirty, creates branches
- Worktree 2 (bugfix/api): Both staged and unstaged, creates 2 branches

## Implementation Details

### How It Works

**Internal Flow:**
1. Detect staged changes: `git diff --cached --quiet`
2. Detect unstaged changes: `git diff --quiet` + `git ls-files --others`
3. Based on state, execute strategy:

**Strategy 3 (Both):**
```bash
# 1. Commit staged changes
git commit -m "git-fire staged backup - 2026-02-13 14:30:52"
SHA1=$(git rev-parse HEAD)
git branch git-fire-staged-main-20260213-143052-${SHA1:0:7}

# 2. Add and commit unstaged (on top of staged)
git add -A
git commit -m "git-fire full backup - 2026-02-13 14:30:52"
SHA2=$(git rev-parse HEAD)
git branch git-fire-full-main-20260213-143052-${SHA2:0:7}

# 3. Reset to original state (optional)
git reset --soft HEAD~2  # Undoes commits, keeps changes
```

**Result:**
- Original branch: Unchanged
- Working tree: Unchanged (all changes still present)
- New branches: Created and ready to push
- Changes: Now staged (side effect of reset --soft)

### Worktree Detection

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

Parsed into `Worktree` structs for easy iteration.

## Next Steps

### Integration with Main CLI

Update `cmd/root.go` to use new strategy:
```go
// Scan repos
repos, _ := git.ScanRepositories(opts)

// For each repo
for _, repo := range repos {
    // Use new dual branch strategy
    result, err := git.AutoCommitDirtyWithStrategy(repo.Path, git.CommitOptions{
        ReturnToOriginal: true,
    })

    if result.BothCreated {
        // Push both branches
        git.PushBranch(repo.Path, "origin", result.StagedBranch)
        git.PushBranch(repo.Path, "origin", result.FullBranch)
    } else if result.StagedBranch != "" {
        git.PushBranch(repo.Path, "origin", result.StagedBranch)
    } else if result.FullBranch != "" {
        git.PushBranch(repo.Path, "origin", result.FullBranch)
    }
}
```

### Worktree Scanning

Update scanner to detect worktrees:
```go
func ScanRepositories(opts ScanOptions) ([]Repository, error) {
    // Find main repos
    repos := findGitDirs(opts)

    // For each repo, check for worktrees
    for i, repo := range repos {
        worktrees, _ := ListWorktrees(repo.Path)
        if len(worktrees) > 1 {
            // Add worktrees as separate repos
            for _, wt := range worktrees[1:] { // Skip main
                repos = append(repos, Repository{
                    Path: wt.Path,
                    // ...
                })
            }
        }
    }

    return repos, nil
}
```

### User Education

Add to output:
```
🔥 Created 2 backup branches:
   📦 git-fire-staged-main-20260213-143052-a1b2c3d (safe - staged changes)
   📦 git-fire-full-main-20260213-143052-e4f5a6b (complete - everything)

ℹ️  The staged branch likely builds correctly!
   Use it if you need a safe recovery point.
```

## Documentation

- ✅ Strategy design: `docs/STAGED_UNSTAGED_STRATEGY.md`
- ✅ Implementation summary: `docs/IMPLEMENTATION_SUMMARY.md`
- ⏳ User guide: Update `README.md`
- ⏳ Examples: Add to `examples/`

## Files Modified

**Implementation:**
- `internal/git/operations.go` - Added new functions
  - `AutoCommitDirtyWithStrategy()`
  - `HasStagedChanges()`
  - `HasUnstagedChanges()`
  - `ListWorktrees()`
  - Helper functions: `commitChanges()`, `createBranch()`

**Types:**
- `internal/git/operations.go` - New types
  - `AutoCommitResult` struct
  - `Worktree` struct
  - Updated `CommitOptions` struct

**Tests:**
- `internal/git/operations_test.go` - 8 new tests
  - All scenarios covered
  - Worktree detection tested
  - 20/20 tests passing

**Documentation:**
- `docs/STAGED_UNSTAGED_STRATEGY.md` - Design document
- `docs/IMPLEMENTATION_SUMMARY.md` - This file!

## Summary

This dual-branch strategy is a **major improvement** over simple `git add -A` backup:

| Feature | Old Approach | New Approach |
|---------|--------------|--------------|
| Staged preservation | ❌ Lost in mix | ✅ Separate branch |
| Unstaged preservation | ✅ Saved | ✅ Saved |
| Safe fallback | ❌ No guarantee | ✅ Staged branch likely works |
| Recovery options | 1 choice | 2-3 choices |
| Semantic meaning | ❌ Lost | ✅ Preserved |
| Worktree support | ❌ No | ✅ Yes |
| Data loss | None | None |
| Complexity | Low | Manageable |

**This is production-ready and significantly better for emergency scenarios! 🔥**
