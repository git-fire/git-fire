# Git-Fire for Agentic Workflows

Agent sessions can leave dirty state across multiple repositories. `git-fire` gives you a single end-of-session checkpoint command with dry-run preview, conflict-safe push behavior, and structured logs.

Related docs:
- project quickstart: [../README.md](../README.md)
- behavior spec: [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- security and operations guide: [security-ops.md](security-ops.md)
- plugin reference: [../PLUGINS.md](../PLUGINS.md)

---

## Recommended Patterns

### 1) Session stop hook (safe mode)

Back up already-created commits without auto-committing unfinished edits.

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "mkdir -p ~/.cache/git-fire && git-fire --path . --skip-auto-commit >> ~/.cache/git-fire/agent-stop.log 2>&1 || true"
          }
        ]
      }
    ]
  }
}
```

### 2) Pre-task checkpoint

```bash
git-fire --dry-run --path ~/projects
git-fire --path ~/projects
```

Use this before risky refactors to create a recovery point.

### 3) Plugin notifications

Use command plugins to notify orchestration systems after backup completes.
See `PLUGINS.md` for plugin types and payload guidance.

## Operational guardrails

- Prefer `--dry-run` before enabling automatic execution.
- Keep `--skip-auto-commit` on stop hooks unless fully intentional.
- Ensure secret scanning and `.gitignore` are set correctly in target repos.
- Treat branch-push failures as non-ignorable and inspect logs promptly.

## Where to go next

- Quickstart and CLI usage: `README.md`
- Behavior reference: `GIT_FIRE_SPEC.md`
- Plugin details: `PLUGINS.md`
- Validation status: `docs/REQUIREMENTS_VALIDATION.md`
