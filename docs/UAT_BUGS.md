# UAT Bug Backlog

Found during MVP UAT (`scripts/uat_test.sh`) — 2026-03-29.
All bugs confirmed by running the compiled binary against real git repos with local bare remotes.

---

## [HIGH] Bug 1 — Dual-branch commit strategy is dead code

**Files:** `internal/executor/runner.go:162`, `internal/git/operations.go:408`

`AutoCommitDirtyWithStrategy()` (creates separate `git-fire-staged-*` and `git-fire-full-*`
branches for staged vs unstaged changes) is fully implemented but **never called**. The runner
calls `AutoCommitDirty()` instead — plain `git add -A && git commit`. Staged and unstaged
changes are merged into one commit; the distinction is lost.

**Expected:** Staged changes → `git-fire-staged-<branch>-<ts>-<sha>` branch. Then unstaged
added → `git-fire-full-<branch>-<ts>-<sha>` branch. Both pushed.

**Fix:** In `runner.go:162`, call `git.AutoCommitDirtyWithStrategy()`. After it returns,
replace the single `ActionPushBranch` with push actions for each created branch
(`result.StagedBranch`, `result.FullBranch`).

---

## [HIGH] Bug 2 — Upstream conflict creates no recovery branch

**Files:** `internal/executor/runner.go`, `internal/executor/planner.go`,
`internal/git/operations.go:92` (DetectConflict), `internal/git/operations.go:134` (CreateFireBranch)

When a push is rejected (non-fast-forward), git-fire creates the local commit but then:
- Does NOT detect the conflict pre-push
- Does NOT create a `git-fire-backup-*` recovery branch
- Does NOT attempt to push the recovery branch

`conflict_strategy = "new-branch"` is validated in config but **never read** at runtime.
`DetectConflict()` and `CreateFireBranch()` have zero callers in the execution path.

**Impact:** In the most critical emergency scenario (diverged branch), the backup commit
exists only locally and cannot reach the remote. This is the highest priority fix before launch.

**Fix (two-phase approach):**
1. Pre-push: Call `DetectConflict()` from the planner; if conflict, add
   `ActionCreateFireBranch` before `ActionPushBranch`.
2. Or post-push: In the runner's push error handler, check for `non-fast-forward` in stderr,
   then call `CreateFireBranch()` and push the new branch.

---

## [MEDIUM] Bug 3 — push-known-branches silently drops never-pushed branches

**File:** `internal/git/operations.go:188` (PushKnownBranches)

Branches that have never been pushed to any remote are **silently skipped** with no warning.
Exit code is 0. Users with local-only branches (e.g. a week of work on `feature-a` never
pushed) will get a "success" result while that work is completely unprotected.

**Fix:** In `PushKnownBranches` (or the runner post-execution), compare local branches to
remote tracking refs and emit a warning for each untracked local branch:
`warning: branch 'feature-a' has no remote tracking ref — not backed up`

---

## [LOW] Bug 4 — `DefaultMode` config field is dead code

**File:** `internal/git/scanner.go:152`, `internal/config/defaults.go`

Scanner hardcodes `Mode: ModePushKnownBranches` on every repo. `cfg.Global.DefaultMode` is
loaded and validated but never applied to repos. Users cannot change the default push mode
via config or env var.

**Fix:** Apply `cfg.Global.DefaultMode` (converted via `git.ParseMode`) to repos in
`upsertRepoIntoRegistry()` when the registry has no existing mode override for that repo.
Remove the hardcoded `ModePushKnownBranches` from `scanner.go:152` (or keep it as the
scanner's zero-value and let the registry/config layer override it).

---

## [LOW] Bug 5 — Cobra prints full flag usage on every error exit

**File:** `cmd/root.go` (`init()` function)

On any error exit (push failure, partial failure), Cobra prints the full flag/command
reference after the error message. The error word "Error:" appears twice and the
actual failure is buried under ~20 lines of help text — bad UX in an emergency.

**Fix:** Add `rootCmd.SilenceUsage = true` in `cmd/root.go init()`.
