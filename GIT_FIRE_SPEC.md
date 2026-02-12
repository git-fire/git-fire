# Git Fire - Emergency Git Repository Backup CLI

## Project Overview

**Purpose:** A panic-mode backup tool that discovers all git repositories on your system and safely pushes them to remote locations with intelligent conflict handling.

**Language:** Go 1.21+
**Invocation:** `git-fire` binary (enables `git fire` syntax when in PATH)

### Core Dependencies
```go
// go.mod excerpt
github.com/go-git/go-git/v5 v5.11.0
github.com/charmbracelet/bubbletea v0.25.0
github.com/charmbracelet/lipgloss v0.10.0
github.com/spf13/cobra v1.8.0
github.com/spf13/viper v1.18.0
```

---

## Behavioral Specification

### Primary Flow

```
User runs: git-fire
    ↓
[PROMPT SCREEN] "Is the building on fire?"
    ├─→ YES → [SCAN] → [DRY RUN] → [PUSH] → [REPORT]
    ├─→ NO → [EXIT]
    └─→ FIRE DRILL → [SCAN] → [DRY RUN REPORT] → [EXIT]
```

#### Step 1: Prompt Screen (5-second timeout)
- Display interactive prompt with countdown
- Accept user input: YES / NO / FIRE DRILL
- Timeout after 5 seconds defaults to NO
- Keyboard: ↑↓ to navigate, Enter to select, Ctrl+C to abort

#### Step 2: Repository Scanning
- Scan filesystem for `.git` directories (configurable root path)
- Respect `scan_exclude` patterns (e.g., `.cache`, `node_modules`)
- Extract remotes from `.git/config` for each repo
- Collect branch info (local, remote, tracking status)
- Run in parallel (goroutine pool) with progress indicator
- Timeout: configurable, default 5 minutes

#### Step 3: Dry-Run Analysis
- For each discovered repo, determine action based on config
- Detect branch conflicts (local != remote tip)
- Generate new branch names for conflicts using template
- Calculate total branches to push
- Estimate push time (rough: branches * avg_time_per_push)
- **No git operations executed at this stage**

#### Step 4A: Fire Drill Mode (DRY RUN REPORT)
- Display comprehensive report of what WOULD happen
- Show repos, branches, conflicts, new branches
- **DO NOT EXECUTE any git operations**
- Exit after user reviews report

#### Step 4B: Panic Mode (EXECUTE)
- Validate auth credentials (SSH keys, tokens)
- If auth invalid, abort with clear error
- For each repo, execute based on configured mode:
  - **Leave Untouched:** Skip entirely
  - **Push Known Branches:** Push only branches that exist on remote
  - **Push All Branches:** Push all local branches, create new on remote
- On conflict: Create new branch per template, push instead
- Handle single repo failures gracefully (log, continue)
- Show real-time progress per repo and branch

#### Step 5: Completion Report
- Summary: repos pushed, branches pushed, conflicts resolved
- Failures list (if any)
- Log file location
- Total time elapsed

---

## Configuration Specification

### File Location
- Primary: `~/.config/git-fire/config.toml`
- Fallback: `~/.git-fire/config.toml`
- Env override: `GIT_FIRE_CONFIG=/path/to/config.toml`

### Default Config (auto-generated on first run)

```toml
[global]
# Mode for unconfigured repos: "leave-untouched" | "push-known-branches" | "push-all"
default_mode = "push-known-branches"

# Conflict handling: "new-branch" (create backup) | "skip" (don't push)
conflict_strategy = "new-branch"

# Template for new branch names on conflict
# Variables: {branch} (original name), {timestamp} (unix), {hash} (commit SHA prefix)
branch_name_template = "git-fire-backup-{branch}-{timestamp}-{hash}"

# Auto-create repos on GitHub (requires GitHub token in auth)
auto_create_repos = false

# Consider repo unmaintained if last commit older than N days
unmaintained_threshold_days = 90

# Filesystem scan settings
scan_exclude = [
  ".cache",
  "node_modules",
  ".venv",
  "venv",
  ".virtualenv",
  "vendor",
  "dist",
  "build",
  "target"
]

# Max directory depth to scan (prevent scanning entire filesystem)
scan_max_depth = 10

# Default scan root (relative paths expand from home)
scan_root = "~"

# Auth settings
[auth]
# SSH key paths (if left empty, uses default ~/.ssh/id_rsa, ~/.ssh/id_ed25519, etc.)
ssh_keys = []

# GitHub token for HTTPS + auto-create (use env var GIT_FIRE_GITHUB_TOKEN instead)
github_token = ""

# Timeout for each git operation (seconds)
operation_timeout = 30

# Parallel goroutines for scanning and pushing
parallel_scan_workers = 8
parallel_push_workers = 4

# Logging
[logging]
log_dir = "~/.config/git-fire/logs"
log_level = "info"  # "debug" | "info" | "warn" | "error"

# Per-repository overrides (matched by path or remote URL)
[[repos]]
path = "/home/user/projects/important-project"
mode = "push-all"
conflict_strategy = "new-branch"

[[repos]]
remote_url = "github.com:user/critical-repo"
mode = "push-all"

[[repos]]
path = "/home/user/archive/old-project"
mode = "leave-untouched"

[[repos]]
path = "/home/user/work/client-repo"
mode = "push-known-branches"
conflict_strategy = "skip"  # Don't create new branches for this one
```

### Config Loading Rules
1. Try to load from `GIT_FIRE_CONFIG` env var
2. Try `~/.config/git-fire/config.toml`
3. Try `~/.git-fire/config.toml`
4. If none exist, create default at `~/.config/git-fire/config.toml` and exit with message
5. User must configure, then re-run

### Per-Repo Matching Logic
- Match by exact `path` first (highest priority)
- Then match by `remote_url` (substring match on origin URL)
- Fall back to `global.default_mode`

---

## Core Data Structures

```go
// Repository represents a discovered git repo
type Repository struct {
    Path          string        // Full filesystem path
    Remotes       []Remote      // All configured remotes
    Branches      []Branch      // All local + remote branches
    IsDirty       bool          // Has uncommitted changes
    Config        RepoConfig    // Resolved config for this repo
    LastError     error         // If scanning failed
}

// Remote represents a git remote
type Remote struct {
    Name       string          // "origin", "upstream", etc.
    URL        string          // Remote URL
    IsValid    bool            // Auth works
    AuthError  string          // If IsValid == false
}

// Branch represents a local or remote branch
type Branch struct {
    Name            string      // Branch name
    IsLocal         bool        // Exists locally
    IsRemote        bool        // Exists on origin
    LocalSHA        string      // Local commit hash
    RemoteSHA       string      // Remote commit hash
    HasDiverged     bool        // Local != remote
    TrackingBranch  string      // "origin/main" etc, empty if untracked
}

// RepoConfig is the resolved config for a single repo
type RepoConfig struct {
    Mode               Mode           // leave-untouched | push-known-branches | push-all
    ConflictStrategy   ConflictStrat  // new-branch | skip
    BranchNameTemplate string
    AutoCreate         bool
}

// PushPlan describes what will happen to a repo
type PushPlan struct {
    Repo              *Repository
    Action            Action              // skip, push, create
    BranchesToPush    []BranchPushPlan
    ConflictsDetected []ConflictInfo
    EstimatedTime     time.Duration
}

// BranchPushPlan describes action for one branch
type BranchPushPlan struct {
    BranchName        string
    Action            Action              // push, new-branch, skip
    NewBranchName     string              // If action == new-branch
    Remote            string              // "origin"
}

// ConflictInfo describes a detected conflict
type ConflictInfo struct {
    BranchName    string
    LocalSHA      string
    RemoteSHA     string
    NewBranchName string  // What we'll create
}

// Config is the full configuration
type Config struct {
    Global      GlobalConfig
    Auth        AuthConfig
    Logging     LoggingConfig
    Repos       []RepoOverride
}
```

---

## File Organization

```
git-fire/
├── cmd/
│   └── root.go              # Cobra root command, entry point
├── internal/
│   ├── config/
│   │   ├── loader.go        # LoadConfig, ValidateConfig
│   │   ├── defaults.go      # Default settings
│   │   └── types.go         # Config structs
│   ├── git/
│   │   ├── scanner.go       # ScanRepositories, parallel scanning
│   │   ├── operations.go    # Push, conflict detection, branch ops
│   │   ├── auth.go          # Validate remotes, auth setup
│   │   └── types.go         # Repository, Branch, Remote structs
│   ├── executor/
│   │   ├── planner.go       # Build push plans (dry-run logic)
│   │   ├── runner.go        # Execute push plans (panic mode)
│   │   ├── logger.go        # Structured logging
│   │   └── types.go         # PushPlan, etc.
│   └── ui/
│       ├── prompt.go        # "Is building on fire?" screen
│       ├── scanning.go      # Scanning progress screen
│       ├── pushing.go       # Real-time push progress screen
│       ├── report.go        # Fire drill or completion report
│       ├── styles.go        # Lipgloss styling
│       └── models.go        # Bubble Tea models
├── main.go
├── go.mod
├── go.sum
└── README.md
```

---

## CLI Interface

### Command: `git-fire [FLAGS]`

```
FLAGS:
  -c, --config FILE          Path to config file
                             (default: ~/.config/git-fire/config.toml)
  
  -p, --path DIR             Only scan this directory
                             (default: home directory)
  
  --dry-run                  Run fire drill without interactive prompt
  
  --auth-check               Validate auth credentials and exit
  
  -v, --verbose              Enable debug logging
  
  -h, --help                 Show help
  
  --version                  Show version
```

### Examples

```bash
# Interactive mode (normal usage)
$ git-fire

# Non-interactive dry-run (useful in scripts)
$ git-fire --dry-run

# Scan only a specific directory
$ git-fire --path /home/user/projects

# Validate auth before trusting the tool
$ git-fire --auth-check

# With custom config
$ git-fire --config ~/my-fire-config.toml
```

---

## User Interface Screens

### Screen 1: Prompt (Initial)

```
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                  🔥 GIT FIRE - PANIC MODE 🔥                  ║
║                                                                ║
║          Is the building on fire? You have 5 seconds...        ║
║                                                                ║
║                      ► YES, PUSH EVERYTHING                    ║
║                        NO, CANCEL                              ║
║                        FIRE DRILL (DRY RUN)                    ║
║                                                                ║
║  Timer: [████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░] 5s         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

**Controls:**
- ↑↓ arrow keys to navigate
- Enter to select
- Ctrl+C to abort
- Auto-selects NO after 5 seconds

### Screen 2: Scanning Repos

```
╔════════════════════════════════════════════════════════════════╗
║  🔍 SCANNING DISK FOR GIT REPOSITORIES...                      ║
║                                                                ║
║  ⠋ Scanning: /home/user/projects                              ║
║                                                                ║
║  Found: 12 repositories                                        ║
║  Current: /home/user/projects/myproject/.git                  ║
║                                                                ║
║  Progress: [████████████░░░░░░░░░░░░░░░░░░░░░░░] 45%          ║
║  Elapsed: 3s | Est. remaining: 4s                              ║
║                                                                ║
║  Press Ctrl+C to cancel                                        ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Real-time spinner animation
- Progress bar updates as repos found
- Shows current directory being scanned
- Estimate time remaining

### Screen 3A: Fire Drill Report (Dry-Run, No Execute)

```
╔════════════════════════════════════════════════════════════════╗
║  🔥 FIRE DRILL REPORT - DRY RUN MODE                          ║
║     (NOTHING WILL BE PUSHED - THIS IS JUST A PREVIEW)         ║
║                                                                ║
║  📊 SUMMARY                                                    ║
║  ─────────────────────────────────────────────────────────────║
║  Repositories found:        15                                 ║
║  Repositories to push:      12                                 ║
║  Total branches:            34                                 ║
║  Detected conflicts:        2                                  ║
║  New branches to create:    2                                  ║
║  Estimated push time:       ~2m 15s                            ║
║                                                                ║
║  📍 REPOS & BRANCHES TO PUSH                                   ║
║  ─────────────────────────────────────────────────────────────║
║  [PUSH-ALL] /home/user/projects/myproject                     ║
║    └─ main (new)                                               ║
║    └─ feature-x (new)                                          ║
║    └─ dev (new)                                                ║
║                                                                ║
║  [PUSH-KNOWN] /home/user/projects/important-repo              ║
║    └─ main (exists remote)                                     ║
║    └─ hotfix (exists remote, CONFLICT)                        ║
║       → Creating: git-fire-backup-hotfix-1706-abc123          ║
║                                                                ║
║  [SKIP] /home/user/archive/old-project                        ║
║    (configured: leave-untouched)                              ║
║                                                                ║
║  ⚠️  CONFLICTS DETECTED (2)                                    ║
║  ─────────────────────────────────────────────────────────────║
║  myproject / feature-x                                        ║
║    Local:  a1b2c3d (3 commits ahead of remote)                ║
║    Remote: x9y8z7w                                             ║
║    Action: Creating git-fire-backup-feature-x-1706-a1b2      ║
║                                                                ║
║  important-repo / hotfix                                      ║
║    Local:  e4f5g6h (2 commits ahead of remote)                ║
║    Remote: m7n8o9p                                             ║
║    Action: Creating git-fire-backup-hotfix-1706-e4f5         ║
║                                                                ║
║  [SPACE] Page down | [B] Page up | [ENTER] Exit               ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Scrollable report (if long)
- Shows exactly what would be pushed
- Clear indication: "NOTHING WILL BE PUSHED"
- Shows conflict resolution strategy

### Screen 3B: Pushing (Panic Mode Active)

```
╔════════════════════════════════════════════════════════════════╗
║  📤 PUSHING REPOSITORIES...                                    ║
║                                                                ║
║  ✓ myproject                                                   ║
║    ├─ main               [██████████████████░░░░░░] 100% ✓    ║
║    ├─ feature-x          [██████████████████░░░░░░] 100% ✓    ║
║    └─ dev                [████████░░░░░░░░░░░░░░░░] 35%       ║
║                                                                ║
║  ⠙ important-repo                                              ║
║    ├─ main               [██████████░░░░░░░░░░░░░░] 50%       ║
║    └─ hotfix             [░░░░░░░░░░░░░░░░░░░░░░░░] 0%       ║
║       ⚠️ CONFLICT DETECTED                                     ║
║       Creating: git-fire-backup-hotfix-1706-e4f5...          ║
║       Pushing new branch...                                    ║
║                                                                ║
║  ⏳ archive-project       [████░░░░░░░░░░░░░░░░░░░░] 20%      ║
║                                                                ║
║  OVERALL: [██████████████░░░░░░░░░░░░░░░░░░░░░░] 45%          ║
║                                                                ║
║  Complete: 1/3 repos | Branches: 8/34 | Conflicts: 1/2       ║
║  Elapsed: 1m 23s | Est. remaining: 2m 15s                     ║
║                                                                ║
║  Press Ctrl+C to abort (may leave partial state)              ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Real-time progress per branch
- Checkmarks for completed
- Spinner for in-progress
- Conflict detection triggers new branch creation inline
- Overall progress bar
- Remaining time estimate

### Screen 4: Completion Report

```
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                    ✅ PUSH COMPLETE ✅                         ║
║                                                                ║
║  12 repositories pushed successfully                          ║
║  34 branches pushed                                            ║
║  2 branches created on remote (conflicts resolved)            ║
║  0 failures                                                    ║
║                                                                ║
║  Time elapsed: 2m 34s                                          ║
║                                                                ║
║  📋 Details logged to:                                         ║
║     ~/.config/git-fire/logs/git-fire-2024-02-06-143025.log   ║
║                                                                ║
║  You can now safely escape the building! 🏃‍♂️                   ║
║                                                                ║
║                    Press ENTER to exit                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Shows summary of push results
- Lists log file for debugging
- Clean exit message

### Screen 5: Error (Auth Failed Example)

```
╔════════════════════════════════════════════════════════════════╗
║                                                                ║
║                    ❌ ERROR - CANNOT PROCEED ❌                ║
║                                                                ║
║  Authentication failed for the following remotes:              ║
║                                                                ║
║  SSH auth issue:                                               ║
║  • origin (github.com:user/repo1) - SSH key not found        ║
║  • upstream (github.com:org/repo2) - Permission denied        ║
║                                                                ║
║  HTTPS auth issue:                                             ║
║  • origin (github.com/user/repo3) - Invalid token            ║
║                                                                ║
║  💡 Fix: Ensure SSH keys are loaded or GitHub token is set   ║
║     $ ssh-add ~/.ssh/id_rsa                                   ║
║     $ export GIT_FIRE_GITHUB_TOKEN=ghp_xxxxx                 ║
║                                                                ║
║  For more help, see: ~/.config/git-fire/logs/               ║
║                                                                ║
║                    Press ENTER to exit                         ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

---

## Algorithm Details

### Scanning Algorithm (Parallel)

```
1. Create work queue of directories to scan
2. Spawn N goroutines from pool
3. Each goroutine:
   a. Pop directory from queue
   b. Walk directory
   c. Check depth < max_depth
   d. If .git/ found, record path
   e. Add subdirs to queue (respecting excludes)
   f. Collect remotes from .git/config
   g. Report progress
4. Aggregate results
5. Return []Repository
```

### Conflict Detection

```
For each local branch B in repo R:
  1. Check if B exists on origin
  2. If exists:
     a. Fetch latest from origin (or skip if recent)
     b. Get local commit SHA: L
     c. Get remote commit SHA: R
     d. If L != R:
        - Check if L is ancestor of R (fast-forward)
        - If not: CONFLICT
        - Mark HasDiverged = true
  3. If doesn't exist:
     - Mark IsRemote = false
     - Action depends on config mode
```

### Push Execution

```
For each repo in push plan:
  1. Validate auth for all remotes
  2. For each branch in plan:
     a. If action == SKIP: continue
     b. If action == PUSH:
        - git push origin branch
        - If fails (conflict): create new branch instead
     c. If action == NEW_BRANCH:
        - git branch <new-name> <base>
        - git push origin <new-name>
     d. Log result
     e. Update progress
  3. Aggregate per-repo result
  4. Continue to next repo (don't fail hard on one repo error)
```

---

## Logging

### Log File Format
- Location: `~/.config/git-fire/logs/git-fire-YYYY-MM-DD-HHMMSS.log`
- Format: JSON lines (one log entry per line)
- Level: Configurable (debug, info, warn, error)

### Sample Log Entry
```json
{"timestamp":"2024-02-06T14:30:25.123Z","level":"info","component":"executor","message":"Starting push phase","repo":"/home/user/projects/myproject","branches_to_push":3}
{"timestamp":"2024-02-06T14:30:26.456Z","level":"info","component":"git","message":"Branch pushed","repo":"/home/user/projects/myproject","branch":"main","remote":"origin","duration_ms":523}
{"timestamp":"2024-02-06T14:30:27.789Z","level":"warn","component":"git","message":"Conflict detected","repo":"/home/user/projects/myproject","branch":"feature-x","local_sha":"a1b2c3d","remote_sha":"x9y8z7w","new_branch":"git-fire-backup-feature-x-1706-a1b2"}
{"timestamp":"2024-02-06T14:30:28.012Z","level":"info","component":"executor","message":"Panic mode complete","total_repos":12,"total_branches_pushed":34,"conflicts_resolved":2,"duration_seconds":154,"failures":0}
```

### Log Levels
- **DEBUG:** Detailed operation info, git commands, internal state
- **INFO:** Major operations (repos found, branches pushed, conflicts)
- **WARN:** Non-fatal issues (skipped repos, slow operations)
- **ERROR:** Failures (auth issues, push failed, etc.)

---

## Error Handling Strategy

### Critical Errors (Halt Execution)
- Auth validation fails → Stop before pushing, show error, exit code 1
- Config invalid → Show validation errors, exit code 2
- Filesystem scan timeout → Show error, exit code 3

### Non-Critical Errors (Log & Continue)
- Single repo push fails → Log error, continue to next repo, note in summary
- Single branch push fails → Log error, try next branch
- Network timeout on one push → Retry N times, then skip, log

### Error Recovery
- Retry policy: 3 retries with exponential backoff (1s, 2s, 4s)
- If fatal: Show user-friendly message + point to logs
- Always save logs, even on partial failure

---

## Git Operations Implementation Notes

### Using `go-git`

Key functions to implement:
```go
// Scanner
func ScanRepositories(scanPath string, excludes []string, maxDepth int) ([]Repository, error)
func GetRemotes(repo *git.Repository) ([]Remote, error)
func GetBranches(repo *git.Repository) ([]Branch, error)

// Operations
func DetectConflict(repo *git.Repository, branchName string) (bool, error)
func PushBranch(repo *git.Repository, remoteName, branchName string) error
func CreateBranchAndPush(repo *git.Repository, newBranchName, baseBranch, remoteName string) error

// Auth
func ValidateRemoteAuth(remote *git.Remote) (bool, error)
func SetupSSHAuth() (transport.AuthMethod, error)
func SetupHTTPSAuth(token string) (transport.AuthMethod, error)
```

**Important:** `go-git` is pure Go but slower than `git` CLI for large repos. Consider shelling out to `git` binary for push operations if performance is critical.

---

## Testing Requirements (MVP)

### Unit Tests
- Config loading and validation
- Branch conflict detection logic
- Push plan generation

### Integration Tests
- Scanning with local test repos
- Pushing to local bare repositories
- Conflict creation and resolution

### Manual Testing Checklist
- [ ] Fire drill shows accurate dry-run
- [ ] Prompt timeout works
- [ ] Scanning on large directories completes
- [ ] Push with conflicts creates backup branches correctly
- [ ] Auth failures caught early
- [ ] Log files created with correct content
- [ ] Cross-platform binary works (Linux, macOS, Windows)

---

## MVP Deliverables

### Must-Have (Phase 1)
1. Config file parsing (TOML) with validation
2. Parallel disk scanning for `.git` directories
3. Remote and branch discovery
4. Interactive prompt screen with timeout
5. Fire drill dry-run mode with detailed report
6. Conflict detection
7. Panic mode push execution with real-time UI
8. New branch creation on conflicts
9. Structured logging to file
10. Completion report
11. Cross-platform binary (macOS, Linux)

### Nice-to-Have (Phase 2)
- Windows support
- GitHub auto-repo creation
- Dropbox / S3 backend support
- Web dashboard for history
- Scheduled automatic backups
- Slack/email notifications
- Configuration UI wizard

---

## Project Structure & Entry Point

### main.go
```go
package main

import "git-fire/cmd"

func main() {
    cmd.Execute()
}
```

### cmd/root.go
Cobra root command that:
- Loads config
- Parses flags
- Determines mode (interactive / --dry-run / --auth-check)
- Hands off to executor

---

## Dependencies Summary

```
go-git/go-git/v5       - Git operations (pure Go)
charmbracelet/bubbletea - TUI framework
charmbracelet/lipgloss  - Terminal styling
spf13/cobra             - CLI framework
spf13/viper             - Config loading
fatih/color             - Colored output
```

**Note:** Avoid `os/exec` calls to system `git` in MVP. If needed for performance later, can be added post-MVP.

---

## Success Criteria

1. **Functional:** User can run `git-fire`, see prompt, run fire drill, execute push
2. **Safe:** No data loss, conflicts handled gracefully, logs comprehensive
3. **Performant:** Scans 100+ repos in < 1 minute, pushes with reasonable speed
4. **Portable:** Single binary, no dependencies beyond Go runtime
5. **User-Friendly:** Clear UI, helpful error messages, intuitive controls
