# git-fire Beta Readiness Report

**Branch:** `main` @ `2c935f8`
**Date:** 2026-03-31
**Auditor:** Automated (claude-4.6-opus) + CodeRabbit PR review data

---

## Executive Summary

**Verdict: NOT YET READY for beta — 3 P0 code issues + 4 CRITICAL doc mismatches must be resolved first.**

The codebase is structurally sound: all tests pass with `-race`, `go vet` is clean, and the core scan-commit-push pipeline works. However, this audit uncovered **3 must-fix code bugs** (data safety / silent no-ops), **4 critical documentation mismatches** that will mislead users on first contact, and **8 should-fix code issues** including a security concern (secrets leaking to disk) and a concurrency race.

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

README says "pushes in parallel." Code pushes sequentially (intentionally, per CLAUDE.md, to avoid SSH contention).

- **Option A:** Update docs to say "scans in parallel, pushes sequentially" (doc fix)
- **Option B:** Implement parallel push workers with SSH contention handling (feature work)
- **Chosen direction:** Run as isolated experiment branch; abandon if too risky and document as WIP until complete.

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

**Risk:** DATA LOSS / REPO MUTATION
**File:** `internal/git/operations.go`, `AutoCommitDirtyWithStrategy` (lines 417-584)

If any step between the first commit and the final `reset --soft HEAD~N` fails (branch creation, second commit, SHA retrieval), the user's current branch is left with 1-2 unwanted commits that are hard to discover. The backup branches may or may not exist.

**Fix direction:** Capture original HEAD SHA at function start. Use `git reset --soft <original-sha>` instead of `HEAD~N` in all cleanup paths. Add a `defer` cleanup that triggers on error after any commit is made.

### P0-2: `conflict_strategy = "abort"` is accepted but has no effect

**Risk:** CORRECTNESS — user expects safety gate, gets raw push failure instead
**File:** `internal/executor/planner.go:139` (only checks `"new-branch"`)

The planner falls through to regular `ActionPushBranch` when strategy is `"abort"`, meaning diverged repos attempt a regular push that fails with an opaque non-fast-forward error instead of being cleanly skipped.

**Fix direction:** Add `else if strategy == "abort"` branch that marks the repo as `Skip` with reason "conflict detected (strategy: abort)".

### P0-3: `--backup-to` flag is a silent no-op

**Risk:** UX SAFETY — user believes backup went to safe location; it didn't
**File:** `cmd/root.go:42,76`

See Decision 1 above. At minimum, the flag must either work or error clearly.

---

## CRITICAL Doc Mismatches

These cause users to fail on first contact with the tool.

| ID | What docs claim | What code does | Files | Resolution |
|----|----------------|----------------|-------|------------|
| D-01 | Spec lists 18+ CLI flags (`--token`, `--platform`, `--prefix`, etc.) | Only 10 flags exist | `GIT_FIRE_SPEC.md` vs `cmd/root.go` init() | Rewrite spec CLI section |
| D-02 | Spec describes fire confirmation prompt with countdown | No prompt exists; immediate execution | `GIT_FIRE_SPEC.md` vs `cmd/root.go` runGitFire | Decision 2 |
| D-03 | Spec config schema has 30+ fields across `[backup]`, `[auth]`, `[logging]` | Actual schema is ~15 fields, no `[logging]` section | `GIT_FIRE_SPEC.md` vs `internal/config/types.go` | Rewrite from actual types |
| D-04 | `--backup-to` described as full feature | Flag exists but is never read | `GIT_FIRE_SPEC.md` + `cmd/root.go` | Decision 1 |

---

## P1 — Should Fix Before Beta

| ID | Category | Title | File | Fix Direction |
|----|----------|-------|------|---------------|
| P1-1 | concurrency | `FindByPath` returns pointer, caller mutates without lock | `internal/ui/selector_helpers.go:21-23` | Use `UpdateByPath` with callback instead |
| P1-2 | ux | `--fire --dry-run` silently ignores dry-run, executes real pushes | `cmd/root.go:173-179` | Make flags mutually exclusive or honor both |
| P1-3 | security | `SaveConfig` from TUI writes env-var secrets to disk in plaintext | `internal/config/loader.go:206-226` | Zero out secrets before marshal or use `toml:"-"` |
| P1-4 | correctness | `DetectConflict` runs `git fetch` during dry-run (network side effect) | `internal/executor/planner.go:140-141` | Skip fetch in dry-run mode |
| P1-5 | ux | `ModePushCurrentBranch` not in TUI mode cycling — 'm' key is no-op | `internal/ui/repo_selector.go:431-438` | Add to cycle |
| P1-6 | correctness | Per-repo config overrides (`[[repos]]`) are never applied | `internal/config/loader.go:149` (defined, never called) | Wire into planner |
| P1-7 | security | `safety.GetUncommittedFiles` stub always returns empty — maintenance trap | `internal/safety/secrets.go:273-277` | Delete the stub |
| P1-8 | data-safety | `reset --soft HEAD~N` fails on repos with <N commits | `internal/git/operations.go:576-582` | Use original SHA instead of `HEAD~N` |

---

## HIGH Doc Mismatches

| ID | What docs claim | What code does | Fix |
|----|----------------|----------------|-----|
| D-05 | "pushes in parallel" | Pushes sequentially | Decision 4 |
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

| ID | Sev | Title |
|----|-----|-------|
| P2-1 | P2 | Repos that fail `analyzeRepository` are silently dropped from backup |
| P2-2 | P2 | `reset --soft` destroys user's original staged/unstaged distinction |
| P2-3 | P2 | Progress channel can silently drop events |
| P2-4 | P2 | No tests for auto-commit partial failure cleanup |
| P2-5 | P2 | No tests for `conflict_strategy = "abort"` |
| P2-6 | P2 | No test for `--fire --dry-run` interaction |
| P2-7 | P2 | Plugin `expandVars` substitutes untrusted repo names into shell args |
| P2-8 | P3 | `SaveConfig` bloats config file with all defaults |
| P2-9 | P3 | Rate limiter bypassed for unknown remote names |

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

### Phase 1: Beta Blockers (estimate: 1 day)

1. **P0-1:** Fix auto-commit partial failure cleanup (use original SHA)
2. **P0-3:** Remove/error `--backup-to` flag (per your decision)
3. **P0-2:** Implement `conflict_strategy = "abort"` in planner
4. **P1-2:** Fix `--fire --dry-run` interaction
5. **P1-8:** Fix `HEAD~N` reset on shallow repos (same fix as P0-1)
6. **P1-3:** Prevent secrets leaking to config file

### Phase 2: Safety & Correctness (estimate: 0.5 day)

7. **P1-1:** Fix `FindByPath` race condition
8. **P1-4:** Skip `git fetch` in dry-run mode
9. **P1-5:** Add `ModePushCurrentBranch` to TUI mode cycle
10. **P1-6:** Wire up `FindRepoOverride` in planner
11. **P1-7:** Delete `safety.GetUncommittedFiles` stub

### Phase 3: Docs Alignment (estimate: 0.5 day)

12. Rewrite `GIT_FIRE_SPEC.md` CLI flags section from actual code
13. Rewrite spec config schema from `types.go` + `defaults.go`
14. Update spec's primary flow to match 3 actual execution paths
15. Fix all HIGH doc mismatches (D-05 through D-15)
16. Fix all MEDIUM/LOW doc mismatches

### Phase 4: Post-Beta (ongoing)

17. All P2/P3 items
18. Manual review of PR #1 and #2 bot feedback
19. Implement features from decisions (if chosen)

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
