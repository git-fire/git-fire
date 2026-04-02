# r/programming launch post

## Title
`git-fire`: one-command multi-repo Git checkpointing (alpha, OSS)

## Body (<300 words)
I kept hitting the same problem: too many local repos, too many in-progress changes, and no low-friction way to checkpoint everything safely.

So I built `git-fire` (open-source Go CLI):
https://github.com/git-fire/git-fire

Tagline is intentionally blunt:
"In case of fire: 1) `git-fire` 2) leave building."

It's a spiritual successor to `qw3rtman/git-fire`, rebuilt for current workflows.

What makes it useful:

- streamed scan -> backup pipeline (doesn't wait for full discovery)
- bounded parallel execution for larger repo sets
- persistent repo registry (set once, reuse)
- dry-run planning + one-screen `--status`
- secret detection guardrails before push
- structured JSON logs for audit/automation
- 250+ tests on core packages

It's also handy for AI coding sessions: one command to checkpoint everything your agent touched across multiple repos.

If your build is literally on fire, this is your hail mary - checkpoint everything and get out.

Status is alpha/MVP: useful now, still hardening.

No tagged release yet (first release coming soon), so install via:
`go install github.com/git-fire/git-fire@main`
MIT licensed.

Recommended post time (ET): Monday, 9:30 AM ET

Likely critical comment:
"This creates noisy commits."

Suggested honest response:
It can if used carelessly. The goal is safety checkpoints, not replacing clean PR hygiene. You can control cadence, use dry-run first, and keep your later history tidy with normal squashing/rebase workflows.
