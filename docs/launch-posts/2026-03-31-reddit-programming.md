# r/programming launch post

## Title
`git-fire`: checkpoint all your repos fast when your day goes sideways

## Body (<300 words)
I kept hitting the same problem: too many local repos, too many in-progress changes, and no low-friction way to checkpoint everything quickly.

So I built `git-fire` (open source, Go):
https://github.com/git-fire/git-fire

Tagline is intentionally blunt:
"In case of fire: 1) `git-fire` 2) leave building."

It's a spiritual successor to the old `qw3rtman/git-fire`, rebuilt for current workflows.

What makes it useful:

- handles large repo sets with parallel execution + bounded workers
- persistent repo registry (set once, reuse forever)
- repeatable end-of-session checkpoint pattern
- dry-run support to verify first
- `--status` gives a quick per-repo glance in one screen
- extensible via plugins (v0.2)
- secret detection warnings before push

It's also handy for AI coding sessions: one command to checkpoint everything your agent touched across multiple repos.

If your build is literally on fire, this is your hail mary - checkpoint everything and get out.

Status is alpha/MVP. Useful now, not pretending to be finished.

Install via Homebrew, Scoop, or `go install`. MIT licensed.

Recommended post time (ET): Monday, 9:30 AM ET

Likely critical comment:
"This creates noisy commits."

Suggested honest response:
It can if used carelessly. The goal is safety checkpoints, not replacing clean PR hygiene. You can control cadence, use dry-run first, and keep your later history tidy with normal squashing/rebase workflows.
