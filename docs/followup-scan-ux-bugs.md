# Followup: Scan UX Bugs (Post-PR #20 Manual Testing)

Found during manual testing of PR #20 ("feat: background scanning, progressive TUI, no-scan mode, and TUI config menu").

---

## Bug 1: TUI scan-status panel shows total repos, not new-only

**Symptom:** After `--fire` scan completes, the panel reads `"✅ Scan Complete (X new repos found)"` but X is the total number of repos discovered this session, not the count of repos *not previously in the registry*.

**Root cause:** `m.scanNewCount` in `internal/ui/repo_selector.go:240` increments for every repo streamed into the TUI during the session. Repos that were already in the registry (pre-existing entries) are counted the same as genuinely new discoveries. There is no separate counter for "first time seen" vs "seen before."

**Relevant files:**
- `internal/ui/repo_selector.go:139` — `scanNewCount int` field definition
- `internal/ui/repo_selector.go:240` — increment site (increments unconditionally on every streamed repo)
- `internal/ui/repo_selector.go:591` — display: `"(%d new repos found)"` uses `scanNewCount`
- `internal/ui/repo_selector.go:605` — in-progress display: same counter

**Fix approach:**
The scanner already knows if a repo is new to the registry vs pre-existing — `upsertRepoIntoRegistry` in `cmd/root.go` returns updated repo info. The `git.Repository` struct or the streaming message needs a flag like `IsNew bool` so the TUI can maintain two counters: `scanNewCount` (truly new) and `scanKnownCount` (already registered). Update the scan-status panel to show both: e.g. `"✅ Scan Complete — 3 new, 12 known"`.

---

## Bug 2: Default mode silently blocks during scan with no feedback

**Symptom:** Running `git-fire` (no `--fire`, no `--dry-run`) backs up known repos quickly, then goes silent for an extended period with no output. The user has no indication the scan is still running. The "scan still running" prompt is never shown.

**Root cause:** Architectural issue in `cmd/root.go`:

```
scanChan → upsert goroutine → repoChan → ExecuteStream (blocks here)
                                close(repoChan) only when scanChan drains
```

`ExecuteStream` blocks until `repoChan` is closed (`cmd/root.go:569`). `repoChan` is only closed by `defer close(repoChan)` in the upsert goroutine (`cmd/root.go:527`), which only runs after `scanChan` is fully drained. So `ExecuteStream` cannot return while the scan is still in progress — the "scan still running" prompt at lines 572-601 is **unreachable** in the current implementation.

The silence the user experiences is `ExecuteStream` blocking while the scan slowly produces more repos (or finishes walking the tree with nothing new to back up).

**Fix approach (two options):**

**Option A — Live progress during wait (simpler):** Add a periodic ticker inside the progress goroutine (`cmd/root.go:551-565`) that prints something like `"⏳ Scanning... (N repos found so far)"` every 2s when no backup progress events arrive. This doesn't change the blocking model, just makes the wait visible.

**Option B — Decouple backup completion from scan completion (correct fix):** Restructure so `ExecuteStream` can return when backup work is drained even if more repos may arrive. This requires splitting repoChan into two phases or letting the runner signal "idle" without closing the channel. Then the "scan still running" prompt logic at lines 572-601 actually works as designed.

**Recommended:** Implement Option A first (low risk, quick win), then Option B as a follow-up if the UX still feels wrong.

**Relevant files:**
- `cmd/root.go:523-537` — upsert goroutine, `close(repoChan)` is deferred
- `cmd/root.go:551-565` — progress display goroutine (add ticker here for Option A)
- `cmd/root.go:567-601` — `ExecuteStream` + unreachable "scan still running" prompt
- `internal/executor/runner.go` — `ExecuteStream` signature and blocking behavior

---

## Testing checklist for fixes

- [ ] `git-fire --fire`: scan-status panel during scan shows `"X new, Y known"` counts separately
- [ ] `git-fire --fire`: after scan completes, shows correct new vs known breakdown
- [ ] `git-fire` (default): while scan is running, periodic status output appears every ~2s
- [ ] `git-fire` (default): "scan still running" prompt appears and Enter/Ctrl+C work correctly
- [ ] `make test-race` passes with no data races introduced by new counter or ticker
