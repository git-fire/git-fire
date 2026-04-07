# Launch Posts Playbook

Use this playbook to run an announcement wave without drifting message or over-promising.

## Positioning (keep this consistent)

- Core promise: one command to checkpoint many local git repos safely.
- Primary value: preserve local-only work before context switches, risky refactors, travel, or outages.
- Trust posture: beta software with clear constraints and safety-first defaults.

## Required message blocks

Every post should include:

1. Problem in one sentence.
2. What `git-fire` does in one sentence.
3. Fast proof path (`--dry-run`, then real run).
4. Current status (beta today; stable once `vX.Y.Z` is published) that matches the release tag.
5. Link to repo and install docs.

## Canonical 30-second demo

```bash
git-fire --dry-run --path ~/projects
git-fire
```

Optional:

```bash
git-fire --fire
```

## Suggested launch wave

### Wave 1: Primary technical audience (high intent)

- Hacker News (`Show HN`)
- Reddit:
  - `r/golang`
  - `r/programming`
  - `r/devops`

### Wave 2: Broader developer distribution

- X / Twitter thread
- LinkedIn post
- GitHub discussion/repo announcement
- Discord communities you actively participate in

### Wave 3: Long-tail discovery

- Dev.to write-up
- Indie Hackers build log
- Lobsters (if post fits community style)
- Product Hunt (when stable release/packaging confidence is strong)

Note: Digg is optional and likely low-return for dev tooling compared with HN/Reddit/Dev.to.

## Example headline variants

- "Show HN: git-fire - one command to checkpoint all dirty git repos"
- "Built a CLI to back up local-only git work across many repos in one run"
- "Emergency multi-repo git backup: dry-run first, push safely second"

## Reusable post template

```md
I built `git-fire`, a CLI for one-command multi-repo git checkpoints.

Problem: when I juggle many repos, manual push loops fail at the worst times.

What it does: discovers repos, optionally auto-commits dirty work, and pushes backups with safety rails.

Quick start:
1. `git-fire --dry-run --path ~/projects`
2. `git-fire`

Status: <beta|stable> (tag: <vX.Y.Z>)

Repo + docs: https://github.com/git-fire/git-fire
Would love brutal feedback on UX, edge cases, and failure modes.
```

## Launch day tracking checklist

- Save final copy in `docs/launch-posts/` before publishing.
- Keep all links to the same release/tag.
- Capture post URLs and comments in a single triage note.
- Respond quickly to first-wave questions (first 2-4 hours matter most).
- Convert repeated questions into README/doc updates within 24h.
