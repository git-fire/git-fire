# Git-Fire for Security and Operations Workflows

This guide covers legitimate dual-use workflows where fast repository checkpointing matters: red team teardown, purple team exercise sync, and incident response state preservation.

`git-fire` is not a replacement for incident evidence tooling. It is a practical multi-repo checkpoint command with safety controls.

---

## Why This Matters in Security/Ops

Security and ops work often changes many repos under time pressure:

- IaC updates
- detection rule changes
- automation script edits
- exercise notes and response playbooks

The risk is not only "did we push?" but "did we push safely and can we audit what happened?"

---

## Core Properties That Help

- **Parallel multi-repo execution:** checkpoint multiple repos quickly.
- **Dry-run mode:** inspect intended actions before any push.
- **Secret detection warnings:** catches likely credential leaks before push.
- **Structured JSON logs:** session audit trail in `~/.cache/git-fire/logs/`.
- **Conflict-safe behavior:** no force-push in normal flows; backup branches when needed.

---

## Workflow 1: Red Team Session Teardown

Goal: checkpoint tool and notes repos before environment teardown.

```bash
# 1) Preview what will happen
git-fire --dry-run --path ~/engagement

# 2) Run checkpoint
git-fire --path ~/engagement

# 3) Review any errors from latest structured log
LATEST=$(ls -t ~/.cache/git-fire/logs/git-fire-*.log | head -1)
cat "$LATEST" | jq 'select(.level == "error")'
```

Practical notes:
- Use separate repos for tooling vs reporting artifacts where possible.
- Keep secret material out of tracked files; heed warnings and review before push.

---

## Workflow 2: Purple Team Exercise Sync

Goal: synchronize scenario scripts, detections, and notes before debrief.

```bash
# Optional status snapshot before sync
git-fire --status --path ~/purple-team

# Safe preview
git-fire --dry-run --path ~/purple-team

# Real checkpoint
git-fire --path ~/purple-team
```

Practical notes:
- Dry-run helps avoid last-minute accidental pushes.
- Keep the generated log file as part of exercise artifacts.

---

## Workflow 3: Incident Response State Preservation

Goal: preserve repo state before broad corrective changes.

```bash
# Before making sweeping edits
git-fire --dry-run --path ~/response
git-fire --path ~/response

# Continue response actions after checkpoint is complete
```

Practical notes:
- This is useful before mass config changes, rollback scripts, or infra remediation.
- Use logs as an audit helper for what was pushed and when.

---

## Operational Guardrails

- Run `--dry-run` first in high-risk environments.
- Review secret warnings before accepting auto-commit output.
- Treat logs as audit support, not a compliance system by themselves.
- Keep independent backups and repository access controls in place.

---

## Current Beta Caveats

- Command plugins defined under `[[plugins.command]]` in `config.toml` are loaded and executed automatically after each non-dry run. Post-run plugins fire once per session and use scan-root template context.
- `--backup-to` is exposed but not implemented yet (`v0.2` target).
- `git-fire` is a checkpoint accelerator, not an evidence collection framework.
