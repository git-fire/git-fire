# Git-Fire Requirements Validation Matrix

> Historical validation snapshot: this file reflects a point-in-time assessment from 2026-03-29 and is not the canonical source for current runtime behavior on `main`.
>
> The percentages, blocker counts, and timeline statements below are historical and may not match the current implementation state.
>
> For current behavior and docs source-of-truth, use:
> - [../README.md](../README.md)
> - [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)
> - [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md) (written spec and validation target; prefer README and shipped code when they disagree during beta)

**Last Updated:** 2026-03-29 (UAT backlog + spec/doc realignment pass)
**Validation Scope:** MVP Feature Completeness vs. GIT_FIRE_SPEC.md
**Purpose:** Identify gaps between specification and implementation before 1.0 release

Status guidance:
- This file is preserved as historical validation context.
- Historical planning notes are preserved for context, and are not canonical status:
  - `docs/VALIDATION_PROGRESS.md`
  - `docs/FINAL_VALIDATION_PLAN.md`
  - `docs/UAT_BUGS.md`
- For navigation, see [docs/README.md](README.md).

**UAT alignment (2026-03-29):** Items tracked in `docs/UAT_BUGS.md` (dual-branch on live path, conflict `new-branch` in planner/runner, push-known warnings, `default_mode` via registry upsert, `SilenceUsage`) were verified in code and tests. This matrix’s evidence rows below were refreshed where they had drifted; summary counts were not re-tallied.

---

## Summary

| Status | Count | Percentage |
|--------|-------|------------|
| ✅ **Fully Implemented & Tested** | 28 | 56% |
| 🟡 **Implemented, Needs Tests** | 8 | 16% |
| 🔴 **Partially Implemented** | 6 | 12% |
| 🔲 **Not Implemented (MVP Blocker)** | 3 | 6% |
| 🔵 **Not Implemented (Phase 2)** | 5 | 10% |
| **Total Requirements** | **50** | **100%** |

**MVP Readiness:** **72%** (28 + 8 out of 50 requirements)
**Blockers:** 3 critical UI components missing

---

## 1. Primary Flow & UI Components

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| **Prompt Screen** | 🔲 Missing | N/A | No `internal/ui/prompt.go` | **BLOCKER**: Build prompt with countdown |
| ASCII fire animation | 🟡 Partial | `internal/ui/fire_bg.go` | Has fire ASCII but not integrated | Wire into prompt screen |
| Repository scanning | ✅ Complete | `internal/git/scanner.go` | Tests pass, coverage 71.9% | ✓ None |
| Parallel scanning | ✅ Complete | `scanner.go:37` | Goroutine pool implementation | ✓ None |
| Scan excludes | ✅ Complete | `config/types.go:44` | `ScanExclude []string` | ✓ None |
| Quick scan paths | 🔲 Not implemented | N/A | Legacy spec only; scanner walks `scan_path` + registry known paths | Historical matrix error (2026-03-29); see [GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md) |
| Dry-run analysis | ✅ Complete | `executor/planner.go` | `GeneratePlan()` tested | ✓ None |
| **Fire drill mode** | 🟡 Partial | `cmd/root.go:83` | `--dry-run` flag exists | Missing full UI report |
| Execute mode | ✅ Complete | `executor/runner.go` | `ExecutePlan()` implemented | ✓ None |
| **Completion report** | 🔲 Missing | N/A | No `internal/ui/report.go` | **BLOCKER**: Build report screen |
| Progress tracking | 🟡 Partial | `executor/runner.go` | Basic console output only | Missing Bubble Tea UI |

**Completion: 5/11 ✅ (45%)**

---

## 2. Auto-Commit Logic

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Auto-commit dirty repos | ✅ Complete | `git/operations.go:36` | `AutoCommitDirty()` | ✓ Tested |
| `git add -A` respects .gitignore | ✅ Verified | `operations.go:50` | Uses git command | ✓ Implicit |
| **Dual-branch strategy** | ✅ Complete | `git/operations.go` + `executor/runner.go` | `AutoCommitDirtyWithStrategy()`; `ActionAutoCommit` pushes `git-fire-staged-*` / `git-fire-full-*` | ✓ Tested |
| Staged-only branch | ✅ Complete | `git/operations.go` | Creates branch from staged | ✓ Tested |
| Full branch (staged + unstaged) | ✅ Complete | `git/operations.go` | Full state branch | ✓ Tested |
| Restore original state | ✅ Complete | `git/operations.go` | `ReturnToOriginal` / soft reset | ✓ Tested |
| Timestamp in commit message | ✅ Complete | `operations.go:15` | ISO8601 timestamp | ✓ Tested |
| Skip auto-commit if clean | ✅ Complete | `operations.go:43` | Early return on clean | ✓ Tested |

**Completion: 8/8 ✅ (100%)**

---

## 3. Conflict Handling

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Detect conflicts | ✅ Complete | `git/operations.go` | `DetectConflict()`; planner calls it per remote in branch-push mode | ✓ Tested |
| Create fire branches | ✅ Complete | `git/operations.go` + `executor/runner.go` | `CreateFireBranch()`; `ActionCreateFireBranch` creates and pushes `git-fire-backup-*` | ✓ Tested |
| Never force push | ✅ Enforced | Design principle | No `--force` in code | ✓ Verified |
| Configurable conflict strategy | ✅ Complete | `config/types.go` + `executor/planner.go` | `ConflictStrategy`; `new-branch` schedules fire backup instead of direct branch push | ✓ Tested |
| Fire branch naming | ✅ Complete | `git/operations.go` | Fixed `git-fire-backup-{branch}-{timestamp}-{shortSHA}` (not a user-editable template today) | ✓ Tested |

**Completion: 5/5 ✅ (100%)**

---

## 4. Push Modes

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Leave untouched mode | ✅ Complete | `executor/planner.go:45` | `ModeLeaveUntouched` | ✓ Tested |
| Push known branches mode | ✅ Complete | `executor/planner.go:48` | `ModePushKnownBranches` | ✓ Tested |
| Push all branches mode | ✅ Complete | `executor/planner.go:51` | `ModePushAll` | ✓ Tested |
| Per-repo overrides | ✅ Complete | `config/loader.go:67` | `FindRepoOverride()` | ✓ Tested |
| Push to all remotes | ✅ Complete | `executor/planner.go` | One action per entry in `repo.Remotes` (origin, backup, etc.) | ✓ Verified |
| Preferred remotes order | 🔲 Missing | N/A | No implementation | Implement config.PreferredRemotes |

**Completion: 4/6 ✅ (67%)**

---

## 5. Configuration System

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| TOML config parsing | ✅ Complete | `config/loader.go` | Viper integration | ✓ Tested |
| Config file locations | ✅ Complete | `loader.go:29` | `~/.config/git-fire/` | ✓ Tested |
| Environment variable overrides | ✅ Complete | `loader.go` | Viper `GIT_FIRE_*` for nested keys; `GIT_FIRE_API_TOKEN`, `GIT_FIRE_SSH_PASSPHRASE` | No `GIT_FIRE_CONFIG` binding — use `--config` |
| Zero-config defaults | ✅ Complete | `config/defaults.go` | Safe defaults | ✓ Tested |
| Per-repo overrides | ✅ Complete | `config/types.go:86` | `[[repos]]` array | ✓ Tested |
| Path matching | ✅ Complete | `loader.go:72` | Exact path match | ✓ Tested |
| Remote URL matching | ✅ Complete | `loader.go:78` | Substring match | ✓ Tested |
| `--init` flag | ✅ Complete | `cmd/root.go:62` | Generates config | ✓ Tested |
| Config validation | ✅ Complete | `loader.go:41` | Error checking | ✓ Tested |

**Completion: 9/9 ✅ (100%)**

---

## 6. SSH & Authentication

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| SSH key detection | ✅ Complete | `auth/ssh.go:23` | `DetectSSHKeys()` | Coverage 49.3% |
| ssh-agent integration | ✅ Complete | `ssh.go:67` | `CheckSSHAgent()` | Add more tests |
| Passphrase handling | ✅ Complete | `ssh.go:89` | `TestPassphrase()` | Add edge cases |
| Multi-key support | ✅ Complete | `ssh.go:28` | Scans common key types | Add tests |
| Passphrase retry logic | 🟡 Needs tests | `ssh.go:102` | Implementation exists | Write failure scenarios |
| SSH status display | 🟡 Partial | `cmd/root.go:119` | Basic status only | Missing interactive fix UI |
| HTTPS token support | 🔲 Missing | N/A | No git HTTPS credential integration for push | `GIT_FIRE_API_TOKEN` only feeds `backup.api_token` (backup mode not active) |

**Completion: 4/7 ✅ (57%)**

---

## 7. Safety & Security

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Secret detection | ✅ Complete | `safety/secrets.go:15` | Pattern matching | Coverage 92.3% ✓ |
| .gitignore respect | ✅ Implicit | `git add -A` behavior | Git respects it | ✓ Verified |
| Security warnings | ✅ Complete | `safety/secrets.go:94` | `SecurityNotice()` | ✓ Tested |
| Log file sanitization | ✅ Complete | `executor/logger.go:45` | Redacts secrets | ✓ Tested |

**Completion: 4/4 ✅ (100%)**

---

## 8. Plugin System

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Command plugins | ✅ Complete | `plugins/command.go` | Shell execution | Coverage 89.1% ✓ |
| Plugin config loading | ✅ Complete | `plugins/loader.go` | TOML parsing | ✓ Tested |
| Trigger system | ✅ Complete | `plugins/types.go:18` | before/after/on-success | ✓ Tested |
| Variable substitution | ✅ Complete | `command.go:82` | Template expansion | ✓ Tested |
| Plugin execution order | ✅ Complete | `command.go:45` | Sequential execution | ✓ Tested |
| Error handling | ✅ Complete | `command.go:67` | Continues on failure | ✓ Tested |
| Go plugins (native) | 🔵 Phase 2 | N/A | Future enhancement | Post-1.0 |
| Webhook plugins | 🔵 Deferred | `plugins/loader.go` | Config types exist; loader TODO | Not loaded in default path |
| **Plugin documentation** | ✅ Complete | `PLUGINS.md` | Status + planned API | Optional deeper PLUGIN_GUIDE still open |

**Completion: 6/9 ✅ (67%)**

---

## 9. Backup Mode (Push to New Remote)

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| `--backup-to` flag | 🟡 Exists | `cmd/root.go:56` | Flag defined | Not implemented |
| GitHub API integration | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |
| GitLab API integration | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |
| Gitea API integration | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |
| Repo name templates | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |
| Manifest generation | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |
| Auto-create repos | 🔵 Phase 2 | N/A | Not implemented | Post-MVP |

**Completion: 0/7 ✅ (0%)** - **Entire feature deferred to Phase 2**

---

## 10. Worktree Support

| Requirement | Status | Location | Evidence | Action Needed |
|-------------|--------|----------|----------|---------------|
| Detect worktrees | ✅ Complete | `git/operations.go:181` | `ListWorktrees()` | ✓ Tested |
| Process independently | 🟡 Design | Scanner finds them | Need executor parallelization | **Task 9**: Worktree threading |
| Worktree-specific commits | 🔴 Partial | Needs testing | Not E2E tested | Add E2E test scenario |

**Completion: 1/3 ✅ (33%)**

---

## 11. CLI Flags (Spec Compliance)

| Spec Flag | Implemented | Location | Notes |
|-----------|-------------|----------|-------|
| `--backup-to URL` | 🟡 Partial | `cmd/root.go:56` | Flag exists, no impl |
| `--token TOKEN` | ❌ No | N/A | Not implemented |
| `--platform TYPE` | ❌ No | N/A | Not implemented |
| `--prefix PREFIX` | ❌ No | N/A | Not implemented |
| `--suffix SUFFIX` | ❌ No | N/A | Not implemented |
| `--remote-name NAME` | ❌ No | N/A | Not implemented |
| `--cleanup-after` | ❌ No | N/A | Not implemented |
| `--no-auto-create` | ❌ No | N/A | Not implemented |
| `--config FILE` | ✅ Yes | `cmd/root.go` | Long form only (no `-c`) |
| `--dry-run` | ✅ Yes | `cmd/root.go:50` | Working |
| `--fire-drill` | ✅ Yes | `cmd/root.go:51` | Alias for --dry-run |
| `--full-scan` | ❌ No | N/A | No flag; walk uses `scan_path` / `--path` and `scan_depth` |
| `--no-scan` | ✅ Yes | `cmd/root.go` | Registry-only walk skip for this run |
| `--force` | ✅ Yes | `cmd/root.go` | With `--init`, overwrite config |
| `--reindex` | ❌ No | N/A | No cache impl |
| `--init` | ✅ Yes | `cmd/root.go:55` | Working |
| `--auth-check` | ❌ No | N/A | Not implemented |
| `--status` | ✅ Yes | `cmd/root.go:57` | Working |
| `--ssh-passphrase` | ❌ No | N/A | Not implemented |
| `--quiet` | ❌ No | N/A | Not implemented |
| `-v, --verbose` | ❌ No | N/A | Not implemented |
| `--path` | ✅ Yes | `cmd/root.go:53` | Working |
| `--skip-auto-commit` | ✅ Yes | `cmd/root.go:54` | Working |
| `--fire` | ✅ Yes | `cmd/root.go:52` | Fancy UI mode |

**Flag Completion: 10/24 ✅ (42%)** (counts exclude obsolete `-c` short form)

---

## 12. UI Screens (Spec Compliance)

| Screen | Status | Location | Evidence | Priority |
|--------|--------|----------|----------|----------|
| **Prompt screen** | 🔲 Missing | N/A | No implementation | **HIGH** - MVP blocker |
| Countdown timer | 🔲 Missing | N/A | No timer logic | **HIGH** |
| Fire animation | ✅ Partial | `ui/fire_bg.go` | Used in `--fire` TUI; not a separate “prompt” screen | **MEDIUM** |
| SSH status display | 🟡 Partial | `cmd/root.go:119` | Text-only status | **MEDIUM** |
| Interactive passphrase fix | 🔲 Missing | N/A | No UI | **LOW** |
| **Scanning progress** | 🔲 Missing | N/A | Console output only | **MEDIUM** |
| Spinner animation | 🔲 Missing | N/A | No Bubble Tea model | **LOW** |
| **Dry-run report** | 🔲 Missing | N/A | Console output only | **HIGH** - MVP blocker |
| **Push progress** | 🔲 Missing | N/A | Console output only | **MEDIUM** |
| Real-time branch progress | 🔲 Missing | N/A | No implementation | **LOW** |
| **Completion report** | 🔲 Missing | N/A | No implementation | **HIGH** - MVP blocker |
| Error screen | 🔲 Missing | N/A | No implementation | **MEDIUM** |

**Screen Completion: 0/12 ✅ (0%)**

---

## 13. Test Coverage Status

| Module | Coverage | Status | Priority to Improve |
|--------|----------|--------|---------------------|
| **safety** | 92.3% | ✅ Excellent | ✓ Sufficient |
| **plugins** | 89.1% | ✅ Excellent | ✓ Sufficient |
| **config** | 86.3% | ✅ Good | ✓ Sufficient |
| **git** | 71.9% | ✅ Good | 🟡 Add edge cases |
| **testutil** | 69.2% | ✅ Good | 🟡 Add scenarios |
| **auth** | 49.3% | 🔴 Weak | **HIGH** - Improve to 70%+ |
| **executor** | 31.7% | 🔴 Weak | **HIGH** - Improve to 70%+ |
| **cmd** | 0.0% | 🔴 None | **HIGH** - Add to 50%+ |
| **ui** | 0.0% | 🔴 None | 🟡 After components built |

**Overall Coverage:** ~65% (weighted average)
**Target for 1.0:** 75%+

---

## Critical Gaps (MVP Blockers)

### 1. Missing UI Components (HIGH PRIORITY)

**Impact:** Core user experience incomplete

**Gaps:**
- ❌ No interactive prompt screen (spec requirement #1)
- ❌ No countdown timer with fire animation
- ❌ No dry-run report (fire drill mode incomplete)
- ❌ No completion report screen
- ❌ No real-time progress tracking UI

**Action:** **Task #8** - Build UI components

### 2. Test Coverage Gaps (HIGH PRIORITY)

**Impact:** Confidence in reliability for production use

**Weak modules:**
- 🔴 executor: 31.7% → target 70%+ (**Task #3**)
- 🔴 auth: 49.3% → target 70%+ (**Task #4**)
- 🔴 cmd: 0% → target 50%+ (**Task #5**)

**Action:** Tasks #3, #4, #5 - Improve coverage

### 3. Missing E2E Tests (MEDIUM PRIORITY)

**Impact:** No validation of complete workflows

**Gaps:**
- ❌ No multi-repo emergency backup scenario
- ❌ No conflict resolution workflow test
- ❌ No plugin execution integration test
- ❌ No worktree handling E2E test

**Action:** **Task #6** - Build E2E test suite

### 4. Incomplete Parallelization (MEDIUM PRIORITY)

**Impact:** Performance and rate limit issues

**Gaps:**
- ❌ No per-host rate limiting (GitHub may block 30+ concurrent pushes)
- ❌ No worktree parallelization (each worktree should be independent thread)
- ❌ No configurable concurrency limits

**Action:** **Task #9** - Implement rate limiting and threading

---

## Non-Critical Gaps (Phase 2)

### 1. Backup Mode (Entire Feature)

**Status:** 🔵 Deferred to Phase 2 (not MVP blocker)

All backup mode requirements (GitHub/GitLab/Gitea API, manifest generation, auto-create repos) are planned for post-1.0.

### 2. Advanced CLI Flags

**Status:** Many spec flags not implemented (32% complete)

**Missing but nice-to-have:**
- `--config` flag (uses env var instead - acceptable)
- `--full-scan` / `--reindex` (no caching yet - acceptable)
- `--auth-check` (can use `--status` - acceptable)
- `--ssh-passphrase` (can use env vars - acceptable)
- `--quiet` / `--verbose` (use defaults - acceptable)

**Impact:** Low - workarounds exist

### 3. Plugin Documentation

**Status:** No comprehensive guide

**Impact:** Medium - users can't easily create plugins

**Action:** **Task #7** - Write PLUGIN_GUIDE.md

---

## Verification Checklist

### Pre-1.0 Release Requirements

**Must complete before 1.0:**
- [ ] Task #1: Requirements validation (this document) ✅ IN PROGRESS
- [ ] Task #3: Executor test coverage to 70%+
- [ ] Task #4: Auth test coverage to 70%+
- [ ] Task #5: CMD test coverage to 50%+
- [ ] Task #8: Build UI components (prompt, report)
- [ ] Task #9: Implement rate limiting & parallelization
- [ ] Task #6: E2E test suite (5+ scenarios)
- [ ] Task #10: Real-world testing (git-fire on itself)

**Should complete before 1.0:**
- [ ] Task #2: Enhance testutil with scenarios
- [ ] Task #7: Plugin documentation guide
- [ ] Overall test coverage: 75%+
- [ ] All MVP UI screens functional

**Can defer to Phase 2:**
- ⏸ Backup mode (GitHub/GitLab API integration)
- ⏸ Advanced CLI flags (--full-scan, --reindex, etc.)
- ⏸ Webhook/Go plugins
- ⏸ Windows support
- ⏸ Web dashboard

---

## Acceptance Criteria for 1.0

### Functional Requirements

- ✅ Core operations work (scan, commit, detect conflicts, push)
- 🔲 Fire drill mode shows detailed report (BLOCKER)
- 🔲 Completion report displays stats (BLOCKER)
- ✅ Dual-branch strategy (staged/unstaged)
- 🟡 Plugin system (internals + docs; default CLI auto-load deferred per PLUGINS.md)
- ✅ Config system with overrides
- 🟡 SSH auth with passphrase support (needs UI polish)
- ✅ Secret detection and warnings
- ✅ Per-host rate limiting (`internal/executor/ratelimit.go`)
- 🔴 Worktree parallelization (PARTIAL)

### Deferred / out-of-scope for 1.0

- ⏸ Interactive prompt with countdown (deferred — not in current CLI; see spec)

### Testing Requirements

- 🔴 Executor coverage: 31.7% → **70%+**
- 🔴 Auth coverage: 49.3% → **70%+**
- 🔴 CMD coverage: 0% → **50%+**
- 🔲 E2E test suite: **5+ scenarios**
- ✅ Safety coverage: 92.3% (sufficient)
- ✅ Plugins coverage: 89.1% (sufficient)
- ✅ Config coverage: 86.3% (sufficient)

### Documentation Requirements

- ✅ README.md (exists, may need updates)
- ✅ GIT_FIRE_SPEC.md (complete)
- ✅ PLUGINS.md (architecture documented)
- 🔲 docs/PLUGIN_GUIDE.md (comprehensive guide)
- ✅ docs/REQUIREMENTS_VALIDATION.md (this document)
- 🔲 docs/TESTING_PLAN.md (after Task #10)
- 🔲 Plugin examples (5+ working examples)
- 🔲 Example configs (3+ configurations)

### Quality Requirements

- 🔴 Zero high-priority bugs
- 🔲 All MVP UI screens functional
- ✅ Safe defaults (works without config)
- ✅ No force-push behavior
- ✅ Proper error handling
- ✅ Rate limiting for concurrent pushes (host + global)
- 🔲 Git-fire successfully backs up itself
- 🔲 Multi-repo (10+) stress test passes

---

## Implementation Timeline

### Week 1: Foundation & Testing
- **Day 1-2:** Task #1 - Requirements validation ✅ DONE (this doc)
- **Day 3-4:** Task #2 - Enhance testutil with scenarios
- **Day 5:** Task #3.1 - Improve executor tests

### Week 2: Coverage & E2E
- **Day 6:** Task #3.2 - Auth tests + Task #5 - CMD tests
- **Day 7-8:** Task #6 - E2E test suite
- **Day 9-10:** Task #7 - Plugin documentation

### Week 3: Core Features
- **Day 11-12:** Task #9 - Rate limiting & config management
- **Day 13-14:** Task #8 - UI components (prompt, report)

### Week 4: Validation
- **Day 15-16:** Task #10 - Real-world testing
- **Day 17:** Bug fixes, polish, 1.0 readiness review

---

## Recommendation: Path to 1.0

### Critical Path (Must-Do)

1. **Build UI components** (Task #8) - 3 MVP blockers
2. **Improve test coverage** (Tasks #3, #4, #5) - Confidence
3. **Implement rate limiting** (Task #9) - Production safety
4. **E2E testing** (Task #6) - Validation
5. **Real-world testing** (Task #10) - Final check

### Quality Improvements (Should-Do)

1. **Enhance testutil** (Task #2) - Developer experience
2. **Plugin docs** (Task #7) - User enablement

### Deferred to Phase 2 (Won't-Do for 1.0)

1. Backup mode (GitHub/GitLab API)
2. Advanced flags (--full-scan, --reindex, etc.)
3. Webhook/Go plugins
4. Windows support

---

## Conclusion

**Git-fire is 72% feature-complete for MVP**, with strong foundations in core operations, config system, and plugin architecture. The main gaps are:

1. **UI components** - 3 critical screens missing
2. **Test coverage** - 3 weak modules need improvement
3. **Rate limiting** - New requirement for production safety

With focused effort on **Tasks #3, #4, #5, #6, #8, #9**, git-fire can reach production-ready 1.0 status within **3-4 weeks**.

**Recommendation:** Proceed with validation plan as outlined. All MVP blockers are well-defined and addressable.

---

**Next Steps:**
1. Complete Task #1 ✅ DONE
2. Begin Task #2 (testutil enhancement)
3. Proceed with critical path (Tasks #3-#9)
4. Final validation (Task #10)
5. Tag v1.0.0 when acceptance criteria met

---

## Hypothetical features (out of validation scope)

The following themes are captured for **roadmap discussion only** in [`GIT_FIRE_SPEC.md`](../GIT_FIRE_SPEC.md) under **“Hypothetical product directions (not specifications)”**:

- General **non-emergency / everyday** mode (calmer UX than the fire narrative).
- **Lazy uploads** as explicit product framing for multi-repo commit-and-push.
- **Automated git pushes** (user-driven scheduling/hooks vs. future first-class automation).
- **Standalone open-source** git integration test helpers (spin-out from `internal/testutil`).

They are **not** rows in this validation matrix until promoted into a numbered phase or requirement in `GIT_FIRE_SPEC.md`.
