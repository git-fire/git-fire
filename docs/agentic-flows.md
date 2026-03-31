# Git-Fire for Agentic Workflows

Agent sessions can leave dirty state across multiple repositories. `git-fire` gives you a single end-of-session checkpoint command with dry-run preview, conflict-safe push behavior, and structured logs.

Related docs:
- project quickstart: [../README.md](../README.md)
- behavior spec: [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- security and operations guide: [security-ops.md](security-ops.md)

---

## Fastest Practical Setup: Stop Hook

The strongest concrete use case today is running `git-fire` when an agent session ends.

### Claude Code stop hook

Add this to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "mkdir -p ~/.cache/git-fire && git-fire --path ~/projects >> ~/.cache/git-fire/claude-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

Safer preview-first variant:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "mkdir -p ~/.cache/git-fire && git-fire --dry-run --path ~/projects > ~/.cache/git-fire/claude-stop-preview.txt 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

### Cursor end-session task

Use a task to standardize the same checkpoint step in Cursor-managed workflows.

Add to `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "git-fire: checkpoint workspace",
      "type": "shell",
      "command": "mkdir -p ~/.cache/git-fire && git-fire --path ~/projects >> ~/.cache/git-fire/cursor-stop.log 2>&1",
      "problemMatcher": []
    },
    {
      "label": "git-fire: dry-run checkpoint",
      "type": "shell",
      "command": "mkdir -p ~/.cache/git-fire && git-fire --dry-run --path ~/projects > ~/.cache/git-fire/cursor-stop-preview.txt 2>&1",
      "problemMatcher": []
    }
  ]
}
```

Run the dry-run task first, then run the real checkpoint task at the end of each agent-heavy session.

---

## Why Git-Fire Instead of a Shell Script

A shell loop can push multiple repos. It usually misses the hard parts.

- **Repo discovery + persistence:** `git-fire` keeps a registry at `~/.config/git-fire/repos.toml`.
- **Safety defaults:** no force-push in normal flows; conflict backup branches are created when needed.
- **Dry-run planning:** preview the run before changing anything.
- **Secret warnings:** highlights likely secret patterns before push.
- **Structured logs:** JSON lines under `~/.cache/git-fire/logs/` for audit and automation.
- **One tool behavior:** same command for humans, hooks, and CI-style wrappers.

---

## Recommended Agent Workflow

1. Run `git-fire --dry-run --path ~/projects`.
2. Review what will be committed and pushed.
3. Run `git-fire --path ~/projects` at session end.
4. Parse the newest log file in `~/.cache/git-fire/logs/` if you need machine-readable post-run status.

Example log parsing:

```bash
LATEST=$(ls -t ~/.cache/git-fire/logs/git-fire-*.log | head -1)
cat "$LATEST" | jq 'select(.level == "error")'
```

---

## Current Limits (Alpha)

- Plugin internals exist, but plugin auto-loading in the default CLI path is not wired yet (`v0.2` target).
- `--backup-to` is exposed but not implemented yet (`v0.2` target).
- Keep an independent backup strategy; this tool is a checkpoint layer, not your only recovery mechanism.
