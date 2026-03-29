# Git-Fire for Agentic Coding

AI coding agents edit code at high speed across multiple repositories. A session that lasts an hour can touch dozens of files across several repos — and most agents don't commit. When the agent crashes, the laptop dies, or you close the terminal, that work is gone.

Git-fire solves this for human emergencies. It solves the same problem for agentic ones.

---

## Why Agents Need Git-Fire

### The core problem

Agents like Claude Code work by editing files directly. They don't commit after every change — they make a series of edits, run tests, iterate, and eventually stop. If the session ends unexpectedly:

- Uncommitted changes vanish if the filesystem isn't durable
- No remote backup means no recovery path
- Multi-repo work is especially fragile — it's hard to even know which repos were touched

### What git-fire provides

- **Auto-commit and push** all dirty repos in one command
- **Parallel execution** — 8+ workers scan and push simultaneously
- **Persistent registry** — remembers which repos exist; no re-scanning from scratch
- **Structured JSON logs** — machine-parseable execution results
- **Plugin hooks** — run arbitrary commands before/after backup completes
- **Dry-run mode** — agents can preview without side effects
- **Exit codes** — success/failure signaling for automation

---

## Current Integration Points

### 1. Claude Code Hooks (Available Today)

Claude Code supports [hooks](https://docs.anthropic.com/en/docs/claude-code/hooks) — shell commands that run in response to events. Git-fire fits naturally as a `Stop` hook, running after every agent session ends.

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "git-fire --path . --skip-auto-commit >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

This pushes any commits the agent made (without auto-committing new dirty state, since the agent may have left things intentionally mid-edit). Use `--dry-run` first to verify:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "git-fire --dry-run --path . 2>&1 | tee ~/.cache/git-fire/last-session-preview.txt"
          }
        ]
      }
    ]
  }
}
```

For aggressive safety — auto-commit everything the agent touched:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "git-fire --path . >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

> **Note:** Combine with secret detection awareness. Run `git-fire --dry-run` regularly to audit what an agent would commit.

### 2. Pre-Operation Backup (Available Today)

Before an agent starts a risky refactor, back up first:

```bash
# In a script that launches an agent session
git-fire --path ~/projects --dry-run  # verify what would be backed up
git-fire --path ~/projects             # actually back up
agent-cli start --task "refactor auth module"
```

This gives a clean restore point before the agent makes changes.

### 3. Plugin Hooks for Agent Notifications (Available Today)

Use command plugins to notify orchestration systems when a backup completes:

```toml
# ~/.config/git-fire/config.toml

[[plugins.command]]
name = "notify-orchestrator"
command = "curl"
args = [
  "-s", "-X", "POST",
  "http://localhost:8080/agent/backup-complete",
  "-H", "Content-Type: application/json",
  "-d", "{\"repo\": \"{repo_name}\", \"sha\": \"{commit_sha}\", \"branch\": \"{branch}\"}"
]
when = "after-push"
```

Or write the result to a file that an agent can read:

```toml
[[plugins.command]]
name = "write-backup-manifest"
command = "sh"
args = [
  "-c",
  "echo '{\"repo\":\"{repo_name}\",\"sha\":\"{commit_sha}\",\"branch\":\"{branch}\",\"time\":\"{timestamp}\"}' >> ~/.cache/git-fire/session-manifest.ndjson"
]
when = "on-success"
```

### 4. Registry as Agent State (Available Today)

Agents that work across many repos can pre-populate the registry:

```bash
# Before starting a long agent session, register all relevant repos
git-fire repos scan ~/projects

# Check what's registered
git-fire repos list

# Ignore repos the agent shouldn't touch
git-fire repos ignore ~/projects/vendor-lib
```

The registry persists at `~/.git-fire/repos.toml` — agents or orchestration scripts can read it directly. For writes or updates, prefer `git-fire repos scan`, `ignore`, `unignore`, or `remove` so validation and invariants are preserved.

### 5. Environment Variable Configuration (Available Today)

No config file needed for agent environments:

```bash
GIT_FIRE_SCAN_PATH=~/projects \
GIT_FIRE_DEFAULT_MODE=push-known-branches \
GIT_FIRE_AUTO_COMMIT_DIRTY=true \
git-fire
```

### 6. JSON Logs for Agent Consumption (Available Today)

Every git-fire run appends structured JSON to `~/.cache/git-fire/logs/`. Each line is a log entry:

```json
{"timestamp":"2026-03-28T10:23:45Z","level":"success","repo":"/home/user/projects/api","action":"push-branch","description":"Pushed main to origin","duration":"1.2s"}
{"timestamp":"2026-03-28T10:23:46Z","level":"error","repo":"/home/user/projects/auth","action":"push-branch","error":"rejected: non-fast-forward"}
```

Agents can tail the log file and parse results without screen-scraping:

```bash
# Get the most recent log file
LATEST=$(ls -t ~/.cache/git-fire/logs/*.json | head -1)
# Parse failures
cat "$LATEST" | jq 'select(.level == "error")'
```

---

## Future Work: Agentic Enhancements

The following items are not yet implemented. They represent the gap between git-fire's current capabilities and what fully-integrated agentic workflows would need.

### P0 — Machine-Readable Output (`--output=json`)

**Problem:** Current output is human-readable terminal text. Agents that invoke git-fire as a subprocess can't reliably parse results.

**Proposed interface:**

```bash
git-fire --output=json
```

Returns a single JSON object on stdout after completion:

```json
{
  "success": true,
  "duration": "3.4s",
  "repos": [
    {
      "path": "/home/user/projects/api",
      "name": "api",
      "status": "success",
      "actions": ["auto-commit", "push-branch"],
      "branch": "main",
      "commit_sha": "abc1234",
      "pushed_to": ["origin"]
    },
    {
      "path": "/home/user/projects/auth",
      "name": "auth",
      "status": "failed",
      "error": "rejected: non-fast-forward",
      "fire_branch": "git-fire-backup-main-20260328-def5678"
    }
  ],
  "summary": {
    "total": 5,
    "success": 4,
    "failed": 1,
    "skipped": 0
  }
}
```

Also applies to subcommands:

```bash
git-fire repos list --output=json
git-fire --status --output=json
```

**Why this matters for agents:** Agents need structured output to make decisions — retry a failed repo, report status to a user, or trigger follow-up actions.

---

### P0 — Plan Output (`git-fire plan --output=json`)

**Problem:** Agents need to inspect the execution plan before committing to it, without running the full dry-run through the TUI.

**Proposed interface:**

```bash
git-fire plan --output=json
```

Returns the full execution plan as JSON:

```json
{
  "dry_run": true,
  "repos": [
    {
      "path": "/home/user/projects/api",
      "name": "api",
      "dirty": true,
      "actions": ["auto-commit", "push-branch"],
      "branch": "main",
      "has_conflict": false,
      "secret_warnings": []
    }
  ]
}
```

**Why this matters for agents:** Agents can evaluate the plan, filter repos, override modes, or warn users about secret detections — all before any side effects occur.

---

### P1 — MCP Server Mode (`git-fire mcp`)

**Problem:** Agents using the Model Context Protocol can't call git-fire as a tool; they can only invoke it as a subprocess.

**Proposed interface:**

```bash
git-fire mcp  # starts an MCP server on stdio
```

Exposes tools:

| Tool | Description |
|------|-------------|
| `backup_repos` | Run git-fire for specified repos or scan path |
| `plan_backup` | Return execution plan without running it |
| `get_status` | Return SSH and registry status |
| `scan_repos` | Discover and register repos at a path |
| `list_repos` | Return registry contents |
| `ignore_repo` | Mark a repo as ignored |

**Example tool call from an agent:**

```json
{
  "tool": "backup_repos",
  "params": {
    "path": "~/projects",
    "dry_run": false,
    "skip_auto_commit": false
  }
}
```

**Why this matters for agents:** MCP is the standard protocol for AI tool use. An MCP-native git-fire would work with Claude Code, Cursor, Windsurf, and any other MCP-compatible agent without subprocess overhead or output parsing.

---

### P1 — Session Tagging (`--session-id`)

**Problem:** When multiple agent sessions run concurrently or sequentially, log entries aren't tied to the session that triggered the backup.

**Proposed interface:**

```bash
git-fire --session-id "claude-session-$(date +%s)"
```

Adds `session_id` to all JSON log entries and the output JSON:

```json
{"timestamp":"...","session_id":"claude-session-1743152400","level":"success",...}
```

**Why this matters for agents:** Orchestration systems can correlate backup events with specific agent sessions, useful for audit trails and recovery workflows.

---

### P1 — NDJSON Progress Streaming

**Problem:** Long-running backups (many repos, slow SSH) give no feedback to agents. The agent blocks on a subprocess with no progress signal.

**Proposed interface:**

```bash
git-fire --output=ndjson
```

Emits one JSON object per line to stdout as work progresses:

```
{"event":"start","total_repos":12,"timestamp":"..."}
{"event":"repo_start","repo":"api","actions":["auto-commit","push-branch"]}
{"event":"repo_success","repo":"api","duration":"1.2s","commit_sha":"abc1234"}
{"event":"repo_start","repo":"auth","actions":["push-branch"]}
{"event":"repo_failed","repo":"auth","error":"rejected: non-fast-forward"}
{"event":"complete","success":11,"failed":1,"duration":"8.3s"}
```

**Why this matters for agents:** Real-time progress lets agents surface status to users during long operations instead of blocking silently.

---

### P1 — Webhook Plugin Implementation

**Problem:** Webhook plugins are specified in PLUGINS.md and `PluginTypeWebhook` exists in the type system, but the implementation is a stub.

**Required work:** Implement `internal/plugins/webhook.go` with:
- HTTP POST/GET with configurable method, headers, body
- Template variable expansion in URL, headers, and body (same vars as command plugins)
- Configurable timeout and retry logic
- `DryRun` support (log what would be sent, don't send)

**Why this matters for agents:** HTTP callbacks are the standard integration point for orchestration systems. Without webhook plugins, agents must poll or use command plugins with `curl`, which is fragile.

**Config example (already supported by config types, needs implementation):**

```toml
[[plugins.webhook]]
name = "agent-callback"
url = "http://localhost:9000/hooks/git-fire"
method = "POST"
headers = { "X-Session-ID" = "${AGENT_SESSION_ID}" }
body = '{"repo":"{repo_name}","sha":"{commit_sha}","status":"backed-up"}'
when = "after-push"
timeout = "10s"
```

---

### P2 — Repo Targeting via Stdin or Flag

**Problem:** Agents often know exactly which repos they've touched. They shouldn't need to scan the entire `--path` to back up just the relevant repos.

**Proposed interface:**

```bash
# Explicit list via flag
git-fire --repos /home/user/projects/api,/home/user/projects/auth

# Or via stdin (NDJSON repo paths)
echo '{"path":"/home/user/projects/api"}' | git-fire --repos-from-stdin
```

**Why this matters for agents:** An agent that modified files in `/home/user/projects/api` can back up precisely that repo without triggering a full recursive scan.

---

### P2 — Go Library Mode

**Problem:** Agents written in Go (or using Go-based frameworks) can't embed git-fire as a library; they must shell out to the binary.

**Proposed work:** Export a stable public API from a `gitfire` package:

```go
import "github.com/TBRX103/git-fire/pkg/gitfire"

result, err := gitfire.Backup(gitfire.Options{
    Path:           "~/projects",
    DryRun:         false,
    SkipAutoCommit: false,
    SessionID:      "agent-session-123",
})

for _, repo := range result.Repos {
    if repo.Status == gitfire.StatusFailed {
        log.Printf("backup failed for %s: %s", repo.Name, repo.Error)
    }
}
```

**Why this matters for agents:** In-process calls are faster and avoid the output-parsing problem entirely. Agent frameworks built in Go can treat git-fire as a dependency.

---

### P3 — Pre-Tool Hook Mode

**Problem:** Claude Code's `PreToolUse` hook can block or warn before an agent uses a tool. A git-fire integration could ensure a backup exists *before* a destructive operation.

**Proposed Claude Code hook configuration:**

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "git-fire --path . --skip-auto-commit --output=json >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

This would back up the current state before every Bash tool call, ensuring the agent always has a recovery point before executing shell commands.

**Why this matters for agents:** The biggest risk in agentic coding is a bash command that wipes a file or corrupts state. A pre-tool backup provides a rollback point.

---

### P3 — Restore/Replay from Logs

**Problem:** Git-fire logs every action, but there's no tooling to reverse them. If an auto-commit introduced a secret or the wrong files, users have to manually `git reset` and `git push --force`.

**Proposed interface:**

```bash
# List recent sessions
git-fire log --list

# Inspect a session
git-fire log --session 2026-03-28T10-23-45

# Undo a session (reset commits, optionally force-push)
git-fire log --undo 2026-03-28T10-23-45 --dry-run
```

**Why this matters for agents:** Agents make mistakes. A fast undo path for git-fire-created commits reduces the risk of using git-fire in automated contexts.

---

## Recommended Agent Integration Pattern

For teams using AI coding agents heavily, the recommended setup is:

```toml
# ~/.config/git-fire/config.toml
[global]
auto_commit_dirty = true
default_mode = "push-known-branches"
scan_path = "~/projects"

[plugins]
enabled = ["session-log"]

[[plugins.command]]
name = "session-log"
command = "sh"
args = ["-c", "echo '{\"repo\":\"{repo_name}\",\"sha\":\"{commit_sha}\",\"branch\":\"{branch}\",\"time\":\"{timestamp}\"}' >> ~/.cache/git-fire/agent-sessions.ndjson"]
when = "on-success"
```

And in Claude Code settings:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "git-fire --path ~/projects >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

This gives you:
- Auto-backup at the end of every agent session
- A persistent machine-readable log of what each session committed
- Recovery path via standard git tools (log entries include commit SHAs)
