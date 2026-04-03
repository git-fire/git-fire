# git-fire Beta Readiness Report

**Audit origin:** `main` @ `2c935f8` · **Date:** 2026-03-31  
**Auditor:** Automated (claude-4.6-opus) + CodeRabbit PR review data

**Integration refresh:** `path_to_beta` @ **`bbb3837`** · **2026-04-03**  
*(PR #47 merged into `path_to_beta`; beta docs below are annotated against this tip.)*

---

## Executive Summary

**Verdict (2026-04-03):** **P0 code blockers called out in this report are addressed on `path_to_beta`** (`bbb3837`): auto-commit cleanup / original-HEAD restoration (PR #47), `conflict_strategy=abort` with conflict detection, and `--backup-to` rejecting with a clear error. **CI posture:** `go vet` + `go test -race` green on that tip.

**Still before calling “beta shipped to `main`”:** merge **[PR #55](https://github.com/git-fire/git-fire/pull/55)** (`path_to_beta` → `main`), then run your release smoke (optional: E2E CLI test, GoReleaser dry-run, Windows spot-check — see [ROADMAP.md](ROADMAP.md)).

**Doc/spec gap:** `GIT_FIRE_SPEC.md` was heavily realigned in **#49**; remaining **D-06–D-35** style items in `BETA_BLOCKERS_PROGRESS.md` are **backlog triage** (verify line-by-line vs spec), not automatic release blockers unless you tighten policy.

The original audit below is **preserved** for traceability; treat **P0/P1 tables** as historical unless a row has a **Status** note.

Merged PR feedback is in good shape: across 6 high-risk PRs with CodeRabbit reviews, **43 of 51 threads were proven addressed**, 4 were intentionally deferred with reviewer agreement, 3 were false positives/duplicates, and 1 partial fix remains (low risk). Two early PRs (#1, #2) had unreviewed bot feedback that warrants a quick follow-up.

---

## Table of Contents

1. [Decisions Needed From You](#decisions-needed-from-you)
2. [P0 — Must Fix Before Beta](#p0--must-fix-before-beta)
3. [CRITICAL Doc Mismatches](#critical-doc-mismatches)
4. [P1 — Should Fix Before Beta](#p1--should-fix-before-beta)
5. [HIGH Doc Mismatches](#high-doc-mismatches)
6. [P2/P3 — Fix After Beta Launch](#p2p3--fix-after-beta-launch)
7. [PR Feedback Verification Summary](#pr-feedback-verification-summary)
8. [Suggested Execution Order](#suggested-execution-order)
9. [Recommended Execution Model](#recommended-execution-model)

---

## Decisions (Resolved on `beta-readiness-audit`)

These items were resolved during the decision pass and are now tracked as implementation constraints in `BETA_BLOCKERS_PROGRESS.md`, `ROADMAP.md`, and `BETA_EXECUTION_STRATEGY.md`.

### Decision 1: `--backup-to` flag — remove or implement?

The `--backup-to <url>` flag is registered in the CLI but entirely unimplemented. It silently does nothing. The spec describes a full "backup to new remote" workflow.

- **Option A:** Remove the flag now, move spec section to "Planned / Phase 2" (15 min fix)
- **Option B:** Implement the feature before beta (multi-day effort)
- **Option C:** Keep the flag but emit an error "not yet implemented" (30 min fix)
- **Chosen direction:** Defer full feature to follow-up and make beta behavior explicit (no silent no-op).

### Decision 2: Fire confirmation prompt — spec vs reality

The spec describes an interactive "Is the building on fire?" prompt with countdown timer. The code has no such prompt — default mode executes immediately.

- **Option A:** Remove the prompt from the spec — immediate execution is the intended UX
- **Option B:** Add a confirmation prompt before non-dry-run execution (feature work)
- **Chosen direction:** Keep immediate execution for beta; track timer/alarm UX as follow-up work.

### Decision 3: Plugin system — how to present?

Plugin code exists (`internal/plugins/`) but is never wired into the CLI. Docs partially present plugins as available.

- **Option A:** Add a "Coming Soon" banner to PLUGINS.md and agentic-flows.md (doc fix only)
- **Option B:** Wire plugins into the CLI for beta (requires testing, ~1 day)
- **Chosen direction:** Present as Coming Soon for beta; remove runtime-available claims and prioritize by demand.

### Decision 4: Push concurrency — doc fix or feature?

**Update (post-audit):** Scanning uses a parallel worker pool; push execution uses `global.push_workers` (default 4) plus host/global rate limiting in the executor. README “parallel pushes” wording should match this model (see spec and `internal/executor/runner.go`).

- **Option A:** Update docs to describe scan parallelism + bounded push workers and per-host limits (doc fix)
- **Option B:** Further experiments (e.g. higher fan-out) stay on isolated branches per original decision
- **Chosen direction:** Document shipped behavior; optional higher-risk concurrency experiments remain non-blocking.

### Decision 5: UI prompt/report screens in beta scope?

Additional prompt/report UI screens are tracked in blockers docs as a follow-up decision.

- **Option A:** Not required for beta
- **Option B:** Required before beta
- **Chosen direction:** Not required for beta; keep tracked as post-beta follow-up.

### Decision 6: `--fire --dry-run` behavior policy

This interaction required an explicit product call for implementation.

- **Option A:** Mutually exclusive flags
- **Option B:** Honor both in interactive dry mode
- **Chosen direction:** Mutually exclusive for beta (explicit error when both are set).

---

## P0 — Must Fix Before Beta

### P0-1: `AutoCommitDirtyWithStrategy` partial failure leaves orphan commits on user's branch

**Status (2026-04-03):** **Addressed on `path_to_beta`** — original-HEAD capture, index-aware cleanup, partial metadata on failure; see **PR #47** / `internal/git/operations.go` + tests.

**Risk:** DATA LOSS / REPO MUTATION
**File:** `internal/git/operations.go`, `AutoCommitDirtyWithStrategy` (lines 417-584)

If any step between the first commit and the final `reset --soft HEAD~N` fails (branch creation, second commit, SHA retrieval), the user's current branch is left with 1-2 unwanted commits that are hard to discover. The backup branches may or may not exist.

**Fix direction:** Capture original HEAD SHA at function start. Use `git reset --soft <original-sha>` instead of `HEAD~N` in all cleanup paths. Add a `defer` cleanup that triggers on error after any commit is made.

### P0-2: `conflict_strategy = "abort"` is accepted but has no effect

**Status (2026-04-03):** **Addressed on `path_to_beta`** — `DetectConflict` runs for `abort`; diverged remotes get **`ActionSkip`** per remote (multi-remote safe). `TestBuildRepoPlan_ConflictStrategyAbort` in `planner_expanded_test.go`.

**Risk:** CORRECTNESS — user expects safety gate, gets raw push failure instead
**File:** `internal/executor/planner.go:139` (only checks `"new-branch"`)

The planner falls through to regular `ActionPushBranch` when strategy is `"abort"`, meaning diverged repos attempt a regular push that fails with an opaque non-fast-forward error instead of being cleanly skipped.

**Fix direction:** Add `else if strategy == "abort"` branch that marks the repo as `Skip` with reason "conflict detected (strategy: abort)".

### P0-3: `--backup-to` flag is a silent no-op

**Status (2026-04-03):** **Addressed** — `runGitFire` returns an error if `--backup-to` is set (`cmd/root.go`); not silent.

**Risk:** UX SAFETY — user believes backup went to safe location; it didn't
**File:** `cmd/root.go:42,76`

See Decision 1 above. At minimum, the flag must either work or error clearly.

---

## CRITICAL Doc Mismatches

These caused users to fail on first contact at audit time. **Re-check `GIT_FIRE_SPEC.md` after #49** — many rows may already match code; treat the table as a verification list.

| ID | What docs claimed | Status (2026-04-03) | Notes |
|----|-------------------|----------------------|-------|
| D-01 | Spec lists fictitious CLI flags | **Verify** | #49 realigned spec to shipped flags — spot-check vs `cmd/root.go` |
| D-02 | Fire confirmation prompt in spec | **Open / doc** | Decision 2: align spec with immediate execution |
| D-03 | Spec config wider than `types.go` | **Verify** | #49 — confirm remaining drift vs `internal/config/types.go` |
| D-04 | `--backup-to` as full feature | **Code OK** | Runtime errors if flag set; spec should say “not implemented” explicitly |

---

## P1 — Should Fix Before Beta

| ID | Category | Title | Status (path_to_beta) |
|----|----------|-------|------------------------|
| P1-1 | concurrency | `FindByPath` mutation without lock | **Done (#48)** — registry `UpdateByPath` |
| P1-2 | ux | `--fire --dry-run` | **Done** — mutually exclusive; `TestRunGitFire_FireAndDryRunMutuallyExclusive` |
| P1-3 | security | `SaveConfig` + env secrets | **Done (#48)** — `sanitizeSecretsForSave` |
| P1-4 | correctness | `DetectConflict` in dry-run | **Done (#48)** — `BuildPlan` passes `DetectConflicts: !dryRun` |
| P1-5 | ux | TUI `push-current-branch` in cycle | **Done (#48)** |
| P1-6 | correctness | Per-repo overrides in planner | **Done (#48)** — `resolveRepoMode` / `effectiveAutoCommitDirty` |
| P1-7 | security | `GetUncommittedFiles` stub | **Done (#48)** — removed |
| P1-8 | data-safety | `HEAD~N` reset | **Done** — same line as P0-1 / PR #47 |

*Original file/line references in the audit snapshot may be stale after refactors.*

---

## HIGH Doc Mismatches

**D-05 note:** `BETA_BLOCKERS_PROGRESS.md` Part 1 still describes “sequential push”; executor uses **worker pool + host limiter** — reconcile that file when editing docs.

| ID | What docs claim | What code does | Fix |
|----|----------------|----------------|-----|
| D-05 | "pushes in parallel" (historical README claim) | Scan parallel; pushes via worker pool + host/global limits | Align README/spec with `global.push_workers` + limiter (Decision 4) |
| D-06 | Spec file tree lists `prompt.go`, `scanning.go`, etc. | UI files are `repo_selector.go`, `fire_bg.go`, etc. | Update spec tree |
| D-07 | `internal/git/auth.go` for auth | Auth is in `internal/auth/` | Update spec |
| D-08 | `conflict_strategy` accepts `"skip"` | Accepts `"abort"` | Update spec |
| D-09 | 3 modes documented | 4 modes exist (missing `push-current-branch`) | Document all 4 |
| D-10 | Log location `~/.config/git-fire/logs/` | Actually `~/.cache/git-fire/logs/` | Update spec |
| D-11 | Positional PATH argument `git-fire ~/projects` | Must use `--path` flag | Update spec |
| D-12 | Env vars `GIT_FIRE_SSH_PASSPHRASE_RSA`, `GIT_FIRE_GITHUB_TOKEN`, etc. | Only `GIT_FIRE_API_TOKEN` and `GIT_FIRE_SSH_PASSPHRASE` exist | Update spec |
| D-13 | "Hybrid Strategy" with background indexing and cache JSON | Simple `filepath.Walk` + registry | Update spec |
| D-14 | Elaborate Go structs with `IsValid`, `HasDiverged`, etc. | Much simpler structs | Update spec |
| D-15 | Plugins presented as available in Getting Started | Plugins are dead code | Decision 3 |

---

## P2/P3 — Fix After Beta Launch

| ID | Sev | Title | Notes (2026-04-03) |
|----|-----|-------|---------------------|
| P2-1 | P2 | Repos that fail `analyzeRepository` are silently dropped from backup | Open |
| P2-2 | P2 | `reset --soft` destroys user's original staged/unstaged distinction | Open (dual-branch tradeoff) |
| P2-3 | P2 | Progress channel can silently drop events | Open |
| P2-4 | P2 | No tests for auto-commit partial failure cleanup | Improved — see `internal/git/operations_test.go` after PR #47 |
| P2-5 | P2 | No tests for `conflict_strategy = "abort"` | Done — `TestBuildRepoPlan_ConflictStrategyAbort` |
| P2-6 | P2 | No test for `--fire --dry-run` interaction | Done — `TestRunGitFire_FireAndDryRunMutuallyExclusive` |
| P2-7 | P2 | Plugin `expandVars` substitutes untrusted repo names into shell args | Open |
| P2-8 | P3 | `SaveConfig` bloats config file with all defaults | Open |
| P2-9 | P3 | Rate limiter bypassed for unknown remote names | Open |

### MEDIUM/LOW Doc Items (15 total, all resolvable now)

- Spec says Go 1.21+, reality is 1.24.2
- Spec dependency versions are stale (bubbletea v0.25 vs v1.3.10, etc.)
- CLAUDE.md says rate limit is 2, reality is 5 (3 for GitHub)
- Spec describes single-commit auto-commit, reality is dual-branch strategy
- `--fire-drill` and `--no-scan` and `--force` flags undocumented in README/spec
- `git-fire repos` subcommand undocumented in main docs
- Spec claims distinct exit codes (1/2/3); code uses exit 1 for all
- Spec claims retry policy (3 retries); no retry logic exists
- Spec says git 2.0+; code requires git 2.22+ (`--show-current`)
- Per-repo config uses `remote` key, spec says `remote_url`
- Branch timestamp format differs from spec
- Config search order differs from spec (spec says `GIT_FIRE_CONFIG` env var)
- `parallel_push_workers` config key documented but doesn't exist

---

## PR Feedback Verification Summary

### High-Risk PRs (had CHANGES_REQUESTED or unresolved threads)

| PR | Title | Threads | Addressed | Deferred | Partial | Confidence |
|----|-------|---------|-----------|----------|---------|------------|
| #4 | Persistent repo registry | 16 | 14 | 2 | 0 | VERY HIGH |
| #5 | Reverse specs | 11 | 10 | 1 | 0 | VERY HIGH |
| #7 | Streaming pipeline | 3 | 2 | 0 | 1 (dup) | VERY HIGH |
| #8 | Agentic docs | 10 | 8 | 0 | 2 (false pos) | HIGH |
| #11 | Security audit | 7 | 6 | 0 | 1 | HIGH |
| #19 | Terminal resize | 4 | 3 | 1 | 0 | HIGH |

**Totals: 51 threads, 43 proven addressed, 4 intentionally deferred, 3 false positives/duplicates, 1 partially addressed.**

### Remaining Gaps from PRs

1. **PR #11 — `TestPassphrase` API** still returns `bool` instead of `(bool, error)`. Low risk: safe default prevents data loss.
2. **PR #19 — Short-terminal collapse** deferred. Low risk: emergency users unlikely on 10-row terminals.
3. **PRs #1, #2 — Unreviewed bot feedback.** CodeRabbit posted substantive comments (command injection in release.yml, TUI selection flags) that were never formally reviewed. **Recommend a quick manual review of these two PRs.**

### Is Full PR-by-PR Trace Overkill?

**Partially.** The automated metadata pass (thread resolution status + review decisions from GitHub API) covers 90% of the verification. The deep commit-level traces were necessary only for the 6 PRs with unresolved/CHANGES_REQUESTED states. For future releases, the `gh api graphql` query used here can be scripted as a pre-release gate check (5-minute automated run).

---

## Suggested Execution Order

### Phase 1–2: Beta blockers + P1 *(completed on `path_to_beta` @ bbb3837)*

Items 1–11 from the original plan are **landed** via **#46–#49**, **PR #47**, and follow-up commits on `path_to_beta`. Use git history if you need exact PR ↔ finding mapping.

### Phase 3: Docs alignment *(ongoing / verify)*

1. Walk **D-06–D-35** in `BETA_BLOCKERS_PROGRESS.md` against current `GIT_FIRE_SPEC.md`, README, and CLAUDE — strike or update each row.
2. Confirm **D-02** (prompt) and any remaining plugin copy match **Decisions 2–3**.

### Phase 4: Ship + post-beta

1. **Merge PR #55** to `main` when ready.
2. Remaining **P2/P3** rows (analyze drop, plugin `expandVars`, etc.) — backlog.
3. Manual review **PR #1 / #2** bot threads if still relevant.
4. Optional: E2E smoke, GoReleaser dry-run, Windows pass (**ROADMAP.md**).

---

## Recommended Execution Model

**Use a high-reasoning flagship model (like the one generating this report) for:**
- P0 fixes (data safety, correctness-critical logic)
- Cross-file refactors (planner + runner + config wiring)
- Final review and verification of all changes

**Use a fast model for:**
- Mechanical doc alignment (spec rewriting, flag list updates)
- Deleting dead code (stub removal, flag removal)
- Simple one-file fixes (add mode to TUI cycle, `toml:"-"` tags)

**Practical default:** Have the flagship model handle Phase 1 and Phase 2 in sequence, then hand Phase 3 doc work to a fast model. Use flagship for final verification pass.

---

## Additional Paranoia Items

Things to be paranoid about for a safety-critical emergency backup tool:

1. **No end-to-end test exists.** There is no test that runs the full CLI pipeline (scan -> plan -> commit -> push) against real git repos. All testing is unit/integration at the package level. Consider adding one E2E smoke test.

2. **No telemetry or crash reporting.** If beta users hit P0-1 (orphan commits), you won't know unless they report it. Consider adding structured error logging that's easy for users to share.

3. **Windows path handling.** The release workflow builds Windows binaries, but `filepath.Walk` behavior, path separators, and `exec.Command("git", ...)` quoting may differ. No Windows CI testing exists.

4. **SSH agent forwarding edge cases.** The auth detection code checks for `SSH_AUTH_SOCK` and runs `ssh-add -l`, but doesn't handle: agent forwarding over tmux/screen, GPG-based SSH keys, or FIDO2/hardware keys that require touch confirmation (which would hang during emergency backup).

5. **Large repo performance.** No benchmarks exist. A user with 50+ repos containing large working trees could hit filesystem scan timeouts or memory pressure during parallel analysis.

6. **GoReleaser + Homebrew tap setup.** PR #18 added publishing infrastructure but it hasn't been tested with an actual release. Consider a dry-run release to a test tag before the real beta.
