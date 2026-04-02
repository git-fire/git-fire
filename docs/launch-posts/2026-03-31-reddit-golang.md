# r/golang launch post

## Title
Built `git-fire` in Go: multi-repo Git checkpoint CLI with bounded concurrency (alpha)

## Body (<400 words)
I shipped an alpha called `git-fire`: an open-source Go CLI for staging/committing/pushing across many repos in one run.

Repo: https://github.com/git-fire/git-fire

I used the original Bash `qw3rtman/git-fire` for years. Since it's abandoned, I rebuilt the concept in Go with two goals: keep UX simple and make internals easier to test and extend.

Implementation choices:

- **Cobra + Viper** for command surface + config/env layering
- **Bubble Tea + Lipgloss** for optional TUI and status view
- **native `git` exec vs `go-git`** for behavior parity with real-world Git installs/config/auth
- **bounded worker concurrency** for large repo sets with controlled parallelism
- **persistent repo registry** so it becomes set-once/run-repeatedly
- **repeatable checkpoint workflow** for end-of-session hygiene
- **`--status` snapshot** for per-repo glance in one screen
- **extensible via plugins (v0.2)**
- **secret detection warnings** before push
- **250+ tests**

This also fits agentic tooling: dry-run + structured exit behavior + JSON logs make it automation-friendly for "checkpoint what the agent changed" flows.

For team use: it already works well for multi-repo operational hygiene; policy controls and richer audit features are roadmap, not marketed as shipped. Current state is alpha/MVP.

If your build is literally on fire, this is your hail mary - checkpoint everything and get out.

I'd value technical critique on:
1. command/API shape
2. concurrency and partial-failure semantics
3. exec-based Git integration tradeoffs
4. JSON/logging contract for automation

Recommended post time (ET): Wednesday, 1:00 PM ET

Likely critical comment:
"Shelling out to git is lazy. Why not `go-git`?"

Suggested honest response:
I chose native `git` deliberately for compatibility and fewer edge-case surprises. Same behavior users already trust, plus guardrails a script usually lacks: conflict-safe backup branches, secret warnings, structured logs, and test coverage across flows.
