# Machine-readable CLI output (`--json`)

`git-fire` supports a stable JSON envelope on **stdout** for agent and automation use. Human-oriented progress lines go to **stderr** when `--json` is set.

Schema version: **`schema_version`: 1** (bump only on breaking JSON changes).

## Commands

| Invocation | stdout |
|------------|--------|
| `git-fire --status --json` | [Status document](#status-document) |
| `git-fire --dry-run --json` | [Run summary document](#run-summary-document) |
| `git-fire --json` (live backup) | [Run summary document](#run-summary-document) at end of run |

Always combine automation with **`--non-interactive`** (or `GIT_FIRE_NON_INTERACTIVE=1`) so registry-lock and config prompts never block.

## Status document

```json
{
  "schema_version": 1,
  "cli_version": "v0.2.0",
  "ssh": {
    "agent_running": true,
    "agent_socket": "/tmp/ssh-xxx/agent.123",
    "agent_keys_known": true,
    "agent_keys_loaded": 1,
    "available_keys": [
      {
        "name": "id_ed25519",
        "type": "ed25519",
        "path": "/home/user/.ssh/id_ed25519",
        "loaded_in_agent": true,
        "needs_passphrase": false
      }
    ]
  },
  "scan": {
    "scan_path": "/home/user/projects",
    "disable_scan": false,
    "no_scan_this_run": false,
    "repos_found": 3,
    "dirty_repos": 1,
    "repositories": [
      {
        "path": "/home/user/projects/app",
        "name": "app",
        "is_dirty": true
      }
    ]
  }
}
```

## Run summary document

Emitted after `--dry-run` planning/execution or after a live backup completes.

```json
{
  "schema_version": 1,
  "cli_version": "v0.2.0",
  "dry_run": true,
  "plan": {
    "total_repos": 2,
    "dirty_repos": 1,
    "conflicts": 0,
    "fire_branches": 0
  },
  "result": {
    "success": 2,
    "failed": 0,
    "skipped": 0,
    "duration": "1.2s",
    "total_actions": 4,
    "failed_actions": 0,
    "repo_results": [
      {
        "path": "/path/to/repo",
        "success": true,
        "duration": "500ms",
        "actions": [
          {
            "type": "auto-commit",
            "description": "..."
          }
        ]
      }
    ]
  },
  "log_path": "/home/user/.cache/git-fire/logs/git-fire-20260101-120000.log"
}
```

`log_path` is empty for dry-run when no session log file is created.

## Related

- Agent playbook: [AGENTS.md](../AGENTS.md)
- Sibling restore CLI: [git-rain](https://github.com/git-fire/git-rain) (pull/sync counterpart)
- Future MCP/skill wrapper: [agent-integration.md](agent-integration.md)
