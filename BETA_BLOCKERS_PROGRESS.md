# Beta Blockers — Full Findings & Execution Plan

**Branch:** `beta-readiness-audit`
**Source:** `main` @ `2c935f8`
**Date:** 2026-03-31
**Purpose:** Preserve all audit context, agent findings, and scan results so we can systematically action every blocker via feature branches.

---

## How This Document Works

Each section below captures raw findings from a specific audit workstream. The [BETA_READINESS_REPORT.md](BETA_READINESS_REPORT.md) is the executive summary with prioritized backlog. This document is the full evidence trail.

The execution workflow:
1. You make decisions on the 4 product questions (in the report)
2. We break work into sub-tasks grouped by phase
3. Each phase gets a feature branch off this audit branch
4. Feature branches merge back to this branch, then to `main` when ready

---

## Part 1: Documentation vs Reality — Full Discrepancy Register

### CRITICAL (4 items)

**D-01 | flag-mismatch | Spec lists 18+ CLI flags that don't exist**
- Docs: `GIT_FIRE_SPEC.md` CLI section documents `--token`, `--platform`, `--prefix`, `--suffix`, `--remote-name`, `--cleanup-after`, `--no-auto-create`, `-c/--config`, `--full-scan`, `--reindex`, `--auth-check`, `--ssh-passphrase`, `--ssh-passphrase-rsa`, `--ssh-passphrase-ed25519`, `--ssh-passphrase-ecdsa`, `--quiet`, `-v/--verbose`
- Reality: `cmd/root.go` init() registers only: `--dry-run`, `--fire-drill`, `--fire`, `--path`, `--skip-auto-commit`, `--no-scan`, `--init`, `--force`, `--backup-to`, `--status`
- Impact: Users get "unknown flag" errors for every spec-documented flag
- Resolution: Rewrite spec CLI section from actual flags

**D-02 | behavior-mismatch | Spec describes fire confirmation prompt; code has none**
- Docs: `GIT_FIRE_SPEC.md` Primary Flow describes "Is the building on fire?" prompt with YES/NO/FIRE DRILL, 10-second countdown
- Reality: `cmd/root.go` runGitFire routes directly to streaming/batch/TUI with no prompt
- Impact: Safety gate described in spec doesn't exist
- Resolution: **NEEDS PRODUCT DECISION** — add prompt or rewrite spec

**D-03 | config-mismatch | Spec config schema has 30+ fields; actual has ~15**
- Docs: `GIT_FIRE_SPEC.md` config section documents `[backup]` (14+ fields), `[auth]`, `[logging]` with log_dir/level/retention/format, `operation_timeout`, `retry_attempts`, etc.
- Reality: `internal/config/types.go` has much simpler schema. No `[logging]`, no retry config, no operation timeout
- Impact: Users crafting config from spec find most fields silently ignored
- Resolution: Rewrite spec config from `types.go` + `defaults.go`

**D-04 | feature-availability | `--backup-to` described as full feature; is silent no-op**
- Docs: `GIT_FIRE_SPEC.md` "Backup to New Remote Mode" describes full workflow
- Reality: Flag registered at `cmd/root.go:76` but variable never read
- Impact: User thinks backup went to safe remote; it didn't
- Resolution: **NEEDS PRODUCT DECISION** — implement, remove, or error

### HIGH (12 items)

**D-05 | behavior-mismatch | "Pushes in parallel" claim; pushes are sequential**
- Docs: README line 19, spec line 62
- Reality: `executor/runner.go` Execute() iterates repos sequentially. CLAUDE.md correctly notes sequential push
- Resolution: **NEEDS PRODUCT DECISION** — doc fix or implement parallel push

**D-06 | architecture-claim | Spec file tree lists nonexistent UI files**
- Docs: `GIT_FIRE_SPEC.md` lists `prompt.go`, `scanning.go`, `pushing.go`, `report.go`, `styles.go`, `models.go`
- Reality: `internal/ui/` has `repo_selector.go`, `repo_selector_lite.go`, `fire_bg.go`, `config_view.go`, `path_display.go`, `selector_helpers.go`, `ignored_entries.go`
- Resolution: Resolvable now — update spec tree

**D-07 | architecture-claim | Spec says auth in `internal/git/auth.go`**
- Reality: Auth is in separate `internal/auth/` package
- Resolution: Resolvable now

**D-08 | config-mismatch | Spec says `conflict_strategy` accepts `"skip"`; code accepts `"abort"`**
- Docs: `GIT_FIRE_SPEC.md` line 173
- Reality: `internal/config/loader.go` Validate() accepts `"new-branch"` | `"abort"`
- Impact: `conflict_strategy = "skip"` fails validation
- Resolution: Resolvable now — update spec to say `"abort"`

**D-09 | config-mismatch | Spec documents 3 modes; code has 4 (missing `push-current-branch`)**
- Reality: `internal/git/types.go` and `config/loader.go` support `push-current-branch`
- Impact: Users can't discover/select a valid mode
- Resolution: Resolvable now — document all 4 modes

**D-10 | behavior-mismatch | Spec says logs at `~/.config/git-fire/logs/`**
- Reality: `executor/logger.go` DefaultLogDir() returns `~/.cache/git-fire/logs/`
- Resolution: Resolvable now

**D-11 | flag-mismatch | Spec shows positional PATH argument `git-fire ~/projects`**
- Reality: Must use `--path` flag. Positional args are silently ignored
- Resolution: Resolvable now — update spec

**D-12 | config-mismatch | Spec lists env vars that don't exist**
- Docs: `GIT_FIRE_SSH_PASSPHRASE_RSA`, `GIT_FIRE_GITHUB_TOKEN`, etc.
- Reality: Only `GIT_FIRE_API_TOKEN` and `GIT_FIRE_SSH_PASSPHRASE` exist
- Resolution: Resolvable now

**D-13 | behavior-mismatch | "Hybrid Strategy" with background indexing/cache JSON**
- Reality: Simple `filepath.Walk` + registry. No background indexing, no cache file
- Resolution: Resolvable now — update spec

**D-14 | architecture-claim | Spec defines elaborate Go structs not matching reality**
- Docs: `Repository` with `Config`, `Remote` with `IsValid`/`AuthError`, `Branch` with `HasDiverged`
- Reality: Much simpler types in `internal/git/types.go`
- Resolution: Resolvable now

**D-15 | feature-availability | Plugins presented as available; they're dead code**
- Docs: PLUGINS.md Getting Started tells users to configure and run plugins
- Reality: `cmd/root.go` never calls plugin loader. All plugin code is dead
- Resolution: **NEEDS PRODUCT DECISION** — add "Coming Soon" or wire in

**D-16 | behavior-mismatch | Auto-commit spec describes single commit; reality is dual-branch**
- Docs: Spec says `git add -A && git commit -m "emergency backup"`
- Reality: `AutoCommitDirtyWithStrategy` creates `git-fire-staged-*` and `git-fire-full-*` branches
- Resolution: Resolvable now — update spec to describe dual-branch strategy

### MEDIUM (13 items)

| ID | Category | Summary | Resolution |
|----|----------|---------|------------|
| D-17 | config | Spec says Go 1.21+; reality is 1.24.2 | Resolvable now |
| D-18 | config | Spec dependency versions stale (bubbletea v0.25 vs v1.3.10, etc.) | Resolvable now |
| D-19 | behavior | CLAUDE.md says rate limit is 2; reality is 5 (3 for GitHub) | Resolvable now |
| D-20 | flag | `--fire-drill` undocumented in README/spec | Resolvable now |
| D-21 | flag | `--no-scan` undocumented in README/spec | Resolvable now |
| D-22 | flag | `--force` (for `--init`) undocumented | Resolvable now |
| D-23 | feature | `git-fire repos` subcommand undocumented in main docs | Resolvable now |
| D-24 | behavior | Spec claims distinct exit codes (1/2/3); all failures use exit 1 | Resolvable now |
| D-25 | behavior | Spec claims retry policy (3 retries); no retry logic exists | Resolvable now |
| D-26 | config | Spec says git 2.0+; code requires 2.22+ (`--show-current`) | Resolvable now |
| D-27 | config | Per-repo config uses `remote` key; spec says `remote_url` | Resolvable now |
| D-28 | behavior | Branch timestamp format differs from spec | Resolvable now |
| D-29 | config | Config search order differs from spec; `GIT_FIRE_CONFIG` env var doesn't exist | Resolvable now |

### LOW (6 items)

| ID | Summary |
|----|---------|
| D-30 | `parallel_push_workers` config documented but doesn't exist |
| D-31 | Spec lists `crypto/ssh` dependency; code uses `os/exec` + ssh-add |
| D-32 | Spec scan timeout "default 5 minutes" doesn't exist |
| D-33 | Config search includes `/etc/git-fire/` and `./config.toml` — undocumented |
| D-34 | Plugin trigger `when = "on-success"` in agentic-flows.md is dead code |
| D-35 | Spec shows `repos-cache.json`; no such file exists |

---

## Part 2: PR Feedback Verification — Full Thread-Level Audit

### Summary Table

| PR | Title | Threads | Proven | Deferred | Partial | Confidence |
|----|-------|---------|--------|----------|---------|------------|
| #4 | Persistent repo registry | 16 | 14 | 2 | 0 | VERY HIGH |
| #5 | Reverse specs | 11 | 10 | 1 | 0 | VERY HIGH |
| #7 | Streaming pipeline | 3 | 2 | 0 | 1 (dup) | VERY HIGH |
| #8 | Agentic docs | 10 | 8 | 0 | 2 (FP) | HIGH |
| #11 | Security audit | 7 | 6 | 0 | 1 | HIGH |
| #19 | Terminal resize | 4 | 3 | 1 | 0 | HIGH |

**Totals: 51 threads, 43 proven addressed, 4 intentionally deferred, 3 false positives, 1 partially addressed**

### Remaining Real Gaps from PRs

1. **PR #11 — `TestPassphrase` API** (`internal/auth/ssh.go`): Still returns `bool` instead of `(bool, error)`. Safe default prevents data loss. LOW RISK.

2. **PR #19 — Short-terminal layout collapse** (`internal/ui/repo_selector.go`): Very short terminals (<25 rows) can overflow. Nonessential sections don't collapse. LOW RISK for emergency use.

3. **PRs #1, #2 — Unreviewed CodeRabbit feedback**: Bot posted substantive comments (command injection in release.yml, TUI selection flags) that were never formally reviewed. MEDIUM RISK — warrants manual review.

### Intentional Deferrals (Accepted by Both Parties)

| PR | Item | Reason |
|----|------|--------|
| #4 | Save concurrent safety (full flock) | PID-based temp name sufficient for one-shot emergency tool |
| #4 | Registry write errors from Bubble Tea Update | Can't propagate errors from tea.Update — matches CLAUDE.md convention |
| #5 | Table-driven test refactor (style nitpick) | Separate named tests are idiomatic Go |
| #19 | Short-terminal section collapse | Feature addition, not bug fix — deferred |

### Repeatable Audit Script

The PR audit was performed using this `gh api graphql` query which can be rerun before any release tag:

```bash
gh api graphql -f query='query($owner:String!,$name:String!){
  repository(owner:$owner,name:$name){
    pullRequests(states:MERGED,first:100,orderBy:{field:UPDATED_AT,direction:DESC}){
      nodes{
        number title url mergedAt reviewDecision
        reviewThreads(first:100){nodes{isResolved isOutdated comments(first:1){totalCount}}}
        reviews(first:100){totalCount nodes{state author{login}}}
      }
    }
  }
}' -F owner='TBRX103' -F name='git-fire'
```

Any PR with `reviewDecision: CHANGES_REQUESTED` and unresolved non-outdated threads needs manual trace.

---

## Part 3: Code Risk Findings — Full Catalog

### P0 — Must Fix Before Beta (3 items)

**F-01 | data-safety | Auto-commit partial failure leaves orphan commits**
- File: `internal/git/operations.go`, `AutoCommitDirtyWithStrategy` (lines 417-584)
- Problem: Any error between first commit and final `reset --soft HEAD~N` leaves unwanted commits on user's branch. Failure points: `getCommitSHA` after commit 1, `createBranch` after commit 1, `commitChanges` for commit 2, `reset --soft` itself.
- Fix: Capture original HEAD SHA at function start. Use `git reset --soft <original-sha>` in all error paths. Add defer-based cleanup.
- Test: Repo with 1 commit + staged+unstaged changes, sabotage `createBranch`, assert HEAD restored.

**F-02 | correctness | `conflict_strategy = "abort"` has no effect**
- File: `internal/executor/planner.go:139` (only checks `"new-branch"`)
- Problem: `"abort"` falls through to regular push; diverged repos get opaque non-fast-forward error instead of clean skip.
- Fix: Add `else if strategy == "abort"` that marks repo as Skip with descriptive reason.
- Test: Configure abort, create diverged repo, assert plan has Skip=true.

**F-03 | ux | `--backup-to` is silent no-op**
- File: `cmd/root.go:42,76`
- Problem: Flag accepted and ignored. User believes backup went to alternate remote.
- Fix: Remove flag or emit clear error. See Decision 1.
- Test: `git-fire --backup-to url` returns error or works.

### P1 — Should Fix Before Beta (8 items)

**F-04 | concurrency | `FindByPath` returns pointer, caller mutates without lock**
- File: `internal/ui/selector_helpers.go:21-23`, `internal/registry/store.go:143-154`
- Problem: In streaming mode, upsert goroutine can trigger slice reallocation while TUI holds a pointer from FindByPath. Dangling pointer.
- Fix: Use `UpdateByPath` with callback instead of FindByPath + direct mutation.

**F-05 | ux | `--fire --dry-run` silently ignores dry-run, pushes for real**
- File: `cmd/root.go:173-179`
- Problem: `--fire` check comes before `--dry-run`; TUI path hardcodes `dryRun=false`.
- Fix: Make mutually exclusive or honor both.

**F-06 | security | `SaveConfig` from TUI writes env-var secrets to disk**
- File: `internal/config/loader.go:57-62` (env→struct), `loader.go:206-226` (marshal all)
- Problem: `GIT_FIRE_API_TOKEN` and `GIT_FIRE_SSH_PASSPHRASE` from env vars end up in config.toml.
- Fix: Zero out secrets before marshal or use `toml:"-"` tags.

**F-07 | correctness | `DetectConflict` runs `git fetch` during dry-run**
- File: `internal/executor/planner.go:140-141`, `internal/git/operations.go:94-99`
- Problem: Dry-run has network side effects. Also fails if network is down.
- Fix: Skip fetch in dry-run mode.

**F-08 | ux | `ModePushCurrentBranch` not in TUI mode cycling**
- File: `internal/ui/repo_selector.go:431-438`
- Problem: Pressing 'm' on a repo with default mode is a no-op — mode not in the cycle.
- Fix: Add `push-current-branch` to the rotation.

**F-09 | correctness | Per-repo config overrides (`[[repos]]`) are never applied**
- File: `internal/config/loader.go:149` (FindRepoOverride defined, never called)
- Problem: Users who configure per-repo overrides get no effect.
- Fix: Wire FindRepoOverride into planner's BuildRepoPlan.

**F-10 | security | `safety.GetUncommittedFiles` stub — maintenance trap**
- File: `internal/safety/secrets.go:273-277`
- Problem: Stub returns empty slice. Any code accidentally importing it gets no secret scanning.
- Fix: Delete the stub.

**F-11 | data-safety | `reset --soft HEAD~N` fails on repos with <N commits**
- File: `internal/git/operations.go:576-582`
- Problem: Brand-new repos (1 commit) with both staged+unstaged changes → `HEAD~2` fails.
- Fix: Use original SHA instead of `HEAD~N` (same fix as F-01).

### P2/P3 — Fix After Beta (9 items)

| ID | Sev | Title |
|----|-----|-------|
| F-12 | P2 | Repos failing `analyzeRepository` silently dropped from backup |
| F-13 | P2 | `reset --soft` destroys user's staged/unstaged distinction |
| F-14 | P2 | Progress channel can silently drop events |
| F-15 | P2 | No tests for auto-commit partial failure cleanup |
| F-16 | P2 | No tests for `conflict_strategy = "abort"` |
| F-17 | P2 | No test for `--fire --dry-run` interaction |
| F-18 | P2 | Plugin `expandVars` substitutes untrusted repo names into shell args |
| F-19 | P3 | `SaveConfig` bloats config file with all defaults |
| F-20 | P3 | Rate limiter bypassed for unknown remote names |

---

## Part 4: Worktree Followup Bugs (from PR #20 manual testing)

Source: `.claude/worktrees/cheerful-marinating-crescent/docs/followup-scan-ux-bugs.md`

### Bug WK-1: TUI scan-status shows total repos, not new-only

- File: `internal/ui/repo_selector.go:240` — `scanNewCount` increments for every streamed repo
- Problem: "✅ Scan Complete (X new repos found)" shows total, not genuinely new
- Fix: Add `IsNew bool` to streaming message, maintain separate counters, show "3 new, 12 known"

### Bug WK-2: Default mode silently blocks during scan with no feedback

- File: `cmd/root.go:523-601`
- Problem: `ExecuteStream` blocks until scan completes. "Scan still running" prompt is unreachable code.
- Fix Option A (quick): Add periodic ticker in progress goroutine printing "⏳ Scanning... (N repos found)"
- Fix Option B (correct): Decouple backup completion from scan completion so runner can signal idle

---

## Part 5: Existing Validation Docs — Additional Gap Indicators

Source: `docs/REQUIREMENTS_VALIDATION.md`, `docs/VALIDATION_PROGRESS.md`, `docs/UAT_BUGS.md`

These docs were created during earlier validation passes. Several claims are now stale or contradict our code review findings.

### Stale "Complete" Claims in REQUIREMENTS_VALIDATION.md

| Claimed Complete | Our Finding | Actual Status |
|-----------------|-------------|---------------|
| "Per-repo overrides ✅ Complete" (Section 4) | F-09: `FindRepoOverride` is defined but never called | **NOT WORKING** — overrides silently ignored |
| "Configurable conflict strategy ✅ Complete" (Section 3) | F-02: `"abort"` falls through to regular push | **PARTIALLY BROKEN** — `"new-branch"` works, `"abort"` doesn't |
| "Environment variable overrides ✅ Complete" (Section 5) | D-12: Only 2 env vars exist, not the 5+ documented | **OVERSTATED** — works for existing vars only |
| "Plugin system functional" (Acceptance Criteria) | D-15/F-09: Plugin code is dead — never called from CLI | **NOT FUNCTIONAL** at runtime |

### Uncompleted Validation Tasks (from VALIDATION_PROGRESS.md)

The earlier validation session completed 4 of 10 planned tasks. Remaining:

| Task | Status | Relevance to Beta |
|------|--------|-------------------|
| #5: CMD module tests (0% coverage) | NOT DONE | HIGH — CLI is the user entry point |
| #6: E2E test suite | NOT DONE | HIGH — no workflow-level validation |
| #7: Plugin documentation | NOT DONE | MEDIUM — plugins not wired anyway |
| #8: UI components (prompt, report) | NOT DONE | **DECISION NEEDED** — spec says these are MVP blockers, but current TUI works without them |
| #9: Rate limiting & config | PARTIALLY DONE | Rate limiter exists but config management gaps remain |
| #10: Real-world testing | NOT DONE | HIGH — git-fire has never been tested on real user repos |

### UAT Bugs — Resolution Verification

`docs/UAT_BUGS.md` lists 5 bugs found during manual testing (2026-03-29). All marked as fixed. Our audit confirms:

- Bug 1 (dual-branch dead code): **FIXED** — `AutoCommitDirtyWithStrategy` is now on live path
- Bug 2 (no conflict recovery branch): **FIXED** — planner evaluates `conflict_strategy = "new-branch"`
- Bug 3 (push-known drops branches): **FIXED** — warnings emitted for local-only branches
- Bug 4 (DefaultMode dead code): **FIXED** — applied via registry upsert
- Bug 5 (Cobra usage spam): **FIXED** — `SilenceUsage = true`

However, Bug 2's fix only covers `"new-branch"` strategy — the `"abort"` path was never implemented (our F-02).

### Test Coverage Reality Check

The VALIDATION_PROGRESS.md claims ~78% overall coverage after its session. Current state should be verified with a fresh `go test -cover ./...` run during implementation. Key gaps that likely persist:

- `cmd/` at 0% (no tests were added since the validation session)
- `internal/ui/` at 0% (intentionally deferred per CLAUDE.md)
- No E2E tests exist anywhere

---

## Part 6: Additional Paranoia Items

1. **No end-to-end test exists.** No test runs full CLI pipeline (scan→plan→commit→push) against real repos.
2. **No Windows CI testing.** Release builds Windows binaries but path handling / `exec.Command` quoting untested.
3. **SSH agent edge cases.** No handling for: agent forwarding over tmux/screen, GPG-based SSH keys, FIDO2/hardware keys (hang on touch prompt).
4. **Large repo performance.** No benchmarks. 50+ repos with large working trees could hit memory pressure.
5. **GoReleaser untested.** PR #18 added publishing infra but no actual release has been done. Recommend dry-run release.
6. **No crash telemetry.** If beta users hit P0-1 (orphan commits), no way to know unless they report.
7. **HTTPS token support missing.** `docs/REQUIREMENTS_VALIDATION.md` Section 6 flags this. Users behind corporate proxies or without SSH may be unable to push.
8. **Preferred remotes ordering not implemented.** Spec and REQUIREMENTS_VALIDATION both flag this — could matter for users with multiple remotes.
9. **CMD module at 0% test coverage.** The CLI entry point is completely untested — flag parsing, routing, error handling all uncovered.
10. **Real-world testing never happened.** Task #10 from the validation plan was never executed. git-fire has not been run against real user repositories outside of controlled test fixtures.

---

## Execution Plan

### Phase 1: Beta Blockers (P0 code + CRITICAL docs) — ~1 day
- Feature branch: `fix/beta-blockers-p0`
- F-01 + F-11: Auto-commit cleanup with original SHA
- F-02: Implement `conflict_strategy = "abort"` in planner
- F-03: Handle `--backup-to` per decision
- F-05: Fix `--fire --dry-run` interaction
- D-01 through D-04: Rewrite spec sections from actual code

### Phase 2: Safety & Correctness (P1 code) — ~0.5 day
- Feature branch: `fix/beta-safety-p1`
- F-04: Fix FindByPath race
- F-06: Prevent secret leak to config
- F-07: Skip fetch in dry-run
- F-08: Add ModePushCurrentBranch to TUI cycle
- F-09: Wire FindRepoOverride
- F-10: Delete GetUncommittedFiles stub

### Phase 3: UX Bugs (Worktree followups) — ~0.5 day
- Feature branch: `fix/scan-ux-bugs`
- WK-1: Fix scan-status counter
- WK-2: Add scan progress feedback (Option A)

### Phase 4: Docs Alignment (HIGH + MEDIUM + LOW) — ~0.5 day
- Feature branch: `docs/spec-reality-alignment`
- All D-05 through D-35

### Phase 5: Post-Beta (P2/P3 + stretch) — ongoing
- Feature branches as needed
- E2E test, Windows CI, GoReleaser dry-run

---

## Decision Register

| # | Question | Options | Your Call |
|---|----------|---------|-----------|
| 1 | `--backup-to` flag | A: Remove flag, B: Implement, C: Error "not implemented" | Defer full feature; treat as follow-up and make beta behavior explicit (no silent no-op). |
| 2 | Fire confirmation prompt | A: Remove from spec, B: Implement prompt | Keep immediate execution for beta; track timer/alarm style flow as follow-up. |
| 3 | Plugin system for beta | A: "Coming Soon" banner, B: Wire into CLI | Coming Soon for beta; remove runtime-available claims and prioritize by demand. |
| 4 | Push concurrency | A: Doc fix (sequential), B: Implement parallel | Run in separate experiment branch; abandon if too risky and document as WIP until ready. |
| 5 | UI "MVP blocker" screens (prompt, report) | A: Not needed for beta — spec is aspirational, B: Build prompt + report screens before beta | Not required for beta; keep tracked as post-beta follow-up. |
| 6 | `--fire --dry-run` policy | A: Mutually exclusive, B: Honor both | Mutually exclusive for beta (explicit error if both flags are set). |
