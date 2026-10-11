# Agent integration sketch (MCP / skill wrapper)

**Status:** Design only — no MCP server ships in this repository yet.

This note describes how a future **Model Context Protocol (MCP) server** or **Cursor skill** could wrap the `git-fire` CLI without duplicating git logic.

## Goals

- Give coding agents a **small, safe tool surface** (preview → confirm → checkpoint).
- Reuse the existing binary and [`--json` output](JSON_OUTPUT.md); do not reimplement push/plan logic in TypeScript/Python.
- Stay consistent with [**git-rain**](https://github.com/git-fire/git-rain) conventions (naming, env vars, JSON schema version bumps).

## Proposed MCP tools

| Tool | Maps to CLI | Default for agents |
|------|-------------|-------------------|
| `git_fire_status` | `git-fire --status --json --path <root>` | Read-only |
| `git_fire_plan` | `git-fire --dry-run --json --non-interactive --path <root>` | Read-only |
| `git_fire_checkpoint` | `git-fire --json --non-interactive --path <root>` | **Destructive** (commits + pushes) |
| `git_fire_checkpoint_committed_only` | add `--skip-auto-commit` | Push committed refs only |
| `git_fire_repos_list` | `git-fire repos list` | Read-only (text today; JSON optional later) |
| `git_fire_repos_ignore` | `git-fire repos ignore <path>` | Mutates local registry only |

Parameters:

- `path` (string, default `.`) — scan root; agents should pass the workspace or monorepo parent explicitly.
- `config` (optional string) — `--config` path when project-local config is intentional.
- `no_scan` (bool) — registry-only run (`--no-scan`).

## Safety contract

The wrapper MUST:

1. **Never** call `git-fire --fire` from an agent (TUI requires a human terminal).
2. Set `GIT_FIRE_NON_INTERACTIVE=1` or pass `--non-interactive` on every invocation.
3. Treat `git_fire_checkpoint*` as **requires human approval** unless the user has enabled an explicit auto-backup policy.
4. Surface `dry_run` and `result.failed` from JSON; exit code `1` on any failure.
5. Not pass `--force-unlock-registry` unless the user explicitly confirms no other instance is running.

Document for end users:

- Live runs may **auto-commit dirty repos** and **push** to configured remotes (no force-push in normal flow).
- Auth is via the user’s existing git/SSH setup; git-harness sets `GIT_TERMINAL_PROMPT=0` so git does not hang on password prompts.

## Implementation sketch

```
Agent (Cursor / Claude Code)
    → MCP tool handler
        → exec: git-fire [--flags]
        → parse stdout JSON (schema_version)
        → return structured content to model + optional log_path link
```

- **Working directory:** run with `cmd.Dir` = user workspace when `--path .` is used.
- **Timeouts:** plan tool ~2–5 minutes; checkpoint tool longer (network pushes).
- **Logs:** include `log_path` from run summary in tool metadata for debugging.
- **Errors:** stderr warnings + non-zero exit; map to MCP `isError: true`.

## Skill alternative

A thin **skill** (`SKILL.md`) can document the same commands for hosts without MCP:

1. `git-fire --dry-run --json --non-interactive --path .`
2. Ask user to approve if `plan.dirty_repos > 0` or pushes are planned.
3. `git-fire --json --non-interactive --path .` (or `--skip-auto-commit`).

Link to [agentic-flows.md](agentic-flows.md) for stop-hook examples.

## Follow-ups (out of scope here)

- JSON output for `git-fire repos list`
- `git_fire_repos_scan` wrapper
- Paired **git-rain** MCP tools for restore/sync after checkpoint
