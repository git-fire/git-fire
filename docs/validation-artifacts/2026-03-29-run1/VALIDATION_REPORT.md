# git-fire Validation Report — 2026-03-29 Run 1 (Remediation Complete)

**Branch:** `validation-run-20260329`
**Date:** 2026-03-29
**Strict Launch Gate Decision:** **GO ✓**

---

## Summary

| Phase | Result | Details |
|-------|--------|---------|
| Phase 1 — Static Gates | **PASS** | build ✓ lint ✓ test-race ✓ (10/10 pkgs) |
| Phase 2 — Local UAT | **PASS** | 45 PASS · 0 FAIL · 0 BUGS |
| Phase 3 — Live GitHub | **PASS** | clean · dirty · conflict runs verified on dummy repos |
| Phase 4 — Resilience / Safety | **PASS** | 10/10 checks (force-push absent, JSON logs valid, dry-run clean, ignored skip, exit codes correct) |

---

## Phase 1 — Static Gates

### make build
- **Result:** PASS
- **Evidence:** `phase1/make-build.log`
- Binary compiled to `./git-fire`, size ~8 MB

### make lint (go vet)
- **Result:** PASS
- **Evidence:** `phase1/make-lint.log`
- Zero vet issues across all packages

### make test-race
- **Result:** PASS (after remediation)
- **Evidence:** `phase1/make-test-race-remediation.log`
- Initial run failed on 2 subtests in `internal/auth`
- **Root cause:** `writeAskpassScript` writes to `~/.cache/git-fire/` using real `HOME`; test subtests did not redirect `HOME` to their `tmpDir`, causing permission failures in sandboxed CI
- **Fix:** Added `os.Setenv("HOME", tmpDir)` + `defer` restore in:
  - `TestIsKeyEncrypted/real_unencrypted_key`
  - `TestTestPassphrase` (top-level setup)
- **Post-fix:** All 10 packages pass with `-race -count=1`

```
ok  github.com/TBRX103/git-fire/cmd
ok  github.com/TBRX103/git-fire/internal/auth
ok  github.com/TBRX103/git-fire/internal/config
ok  github.com/TBRX103/git-fire/internal/executor
ok  github.com/TBRX103/git-fire/internal/git
ok  github.com/TBRX103/git-fire/internal/plugins
ok  github.com/TBRX103/git-fire/internal/registry
ok  github.com/TBRX103/git-fire/internal/safety
ok  github.com/TBRX103/git-fire/internal/testutil
ok  github.com/TBRX103/git-fire/internal/ui
```

---

## Phase 2 — Local UAT (9 scenarios, scripts/uat_test.sh)

- **Initial result:** 31 PASS · 11 FAIL · 3 BUGS
- **Final result (post-remediation):** 45 PASS · 0 FAIL · 0 BUGS
- **Evidence:** `phase2-uat-final.log`

### What failed initially and why

The UAT script was written before the dual-branch backup strategy was fully wired into the runner. After bug fixes (Bugs 1–5 from `UAT_BUGS.md`), the runner now calls `AutoCommitDirtyWithStrategy` which creates:
- `git-fire-staged-<branch>-<ts>-<sha>` — staged-only snapshot
- `git-fire-full-<branch>-<ts>-<sha>` — full working-tree snapshot

The script's old assertions checked for a single commit pushed to `main`, which was the pre-fix behavior. All 11 failures were stale assertions — the actual data was correct, just on backup branches.

Additionally, 3 hardcoded `log_bug` calls remained in the script even though the bugs were fixed; these were converted to runtime behavioral checks.

Also, `git branch --list` returns branch names with leading whitespace; the variable-capture pipe needed `| tr -d ' '` before the name was used in `git show <branch>:<file>`.

### Scenario outcomes (final)

| Scenario | Result | Notes |
|----------|--------|-------|
| S1: staged + unstaged | PASS | Both backup branches created and pushed; files verified in each |
| S1b: staged only | PASS | `git-fire-staged-*` branch pushed; file verified |
| S1c: unstaged only | PASS | `git-fire-full-*` branch pushed; file verified |
| S2: upstream conflict | PASS | Main push rejected (non-fast-forward); backup branches still pushed safely |
| S3: push-known-branches | PASS | Local-only branches not pushed; warning emitted |
| S3b: push-all mode | PASS | All local branches pushed when mode=push-all via registry |
| S4: partial conflict | PASS | One branch succeeds, one rejected; correct exit code |
| S5: bad remote | PASS | Local backup branches created; push fails gracefully; exit non-zero |
| S6: clean repo | PASS | No auto-commit; push succeeds; exit 0 |
| S7: --dry-run | PASS | No changes to remote or local; dry-run message shown |
| S8: no remote | PASS | Repo skipped cleanly; exit 0 |
| S9: --skip-auto-commit | PASS | Staged changes preserved; no commit made |

---

## Phase 3 — Live GitHub Validation

- **Evidence:** `phase3/live-clean.log`, `phase3/live-dirty.log`, `phase3/live-conflict.log`, `phase3/live-remote-branches.txt`
- **Dummy repos created:** `git-fire-uat-clean-run2`, `git-fire-uat-dirty-run2`, `git-fire-uat-conflict-run2` (prefix `git-fire-uat-*`, kept for audit)

| Scenario | Result |
|----------|--------|
| Clean repo push to GitHub | PASS |
| Dirty repo → dual-branch backup to GitHub | PASS — `git-fire-staged-*` and `git-fire-full-*` appeared on GitHub |
| Conflict → backup branches pushed, main rejected | PASS |
| No force-push observed | PASS |

---

## Phase 4 — Resilience & Safety

- **Evidence:** `phase4-resilience.log`
- **Result:** 10/10 PASS

| Check | Result |
|-------|--------|
| C1: `--force-push` flag absent | PASS |
| C2: structured JSON log created on every run | PASS |
| C2: all log lines parse as valid JSON | PASS |
| C3: security warning shown when secret-looking file present | PASS |
| C4: `--dry-run` leaves working tree unchanged | PASS |
| C4: `--dry-run` creates no local branches | PASS |
| C4: `--dry-run` creates no remote branches | PASS |
| C5: ignored repo skipped (not pushed) | PASS |
| C5: ignored repo shows skip in output | PASS |
| C6: exit code non-zero on push failure | PASS |

---

## Strict Launch Gate

All gate criteria met:

- [x] `make build` clean
- [x] `make lint` clean
- [x] `make test-race` — all packages pass, race detector clean
- [x] UAT — 0 failures, 0 open bugs
- [x] No force-push capability exists
- [x] Dry-run is completely non-destructive
- [x] Ignored repos are not backed up
- [x] Structured JSON logging on every run
- [x] Security warning shown before any commit
- [x] Exit codes correctly reflect failure

**DECISION: GO ✓**

---

## Remediation Log

| Item | Root Cause | Fix |
|------|-----------|-----|
| `TestIsKeyEncrypted/real_unencrypted_key` failing | Test used real HOME; `writeAskpassScript` could not write to `~/.cache/git-fire/` in sandbox | Set `HOME=tmpDir` in subtest |
| `TestTestPassphrase/correct_passphrase` failing | Same root cause | Set `HOME=tmpDir` at top of `TestTestPassphrase` |
| 11 UAT assertion failures | Assertions checked `main` for files; dual-branch strategy puts files on backup branches | Updated all S1/S1b/S1c/S2/S5 assertions to check `git-fire-staged-*` / `git-fire-full-*` branches |
| 3 hardcoded `log_bug` calls | Scripts written before bug fixes; no runtime gate | Converted to runtime behavioral checks; bugs confirmed resolved |
| Branch names with leading whitespace | `git branch --list` pads with spaces; used raw in `git show` | Added `| tr -d ' '` to all branch-name captures |
