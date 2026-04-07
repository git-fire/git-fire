# Agent Work Plan — Post-Launch Polish

## Status: Ready to execute
## Branch: chore/post-launch-polish

---

## DONE (merged in PR #71)

- [x] Plugin auto-loading wired in `cmd/root.go` via `plugins.LoadFromConfig(cfg)`
- [x] Always-trigger double-fire bug fixed in post-run dispatch loop
- [x] `CHANGELOG.md` and `ROADMAP.md` updated to reflect plugin auto-loading

---

## DONE (this branch)

- [x] GitLab token detection added to `internal/safety/secrets.go` `defaultPatterns()`
- [x] golangci-lint CI skip documented with structured comment in `.github/workflows/ci.yml`
- [x] `README.md`: git-fire.com mention added after badge block
- [x] `README.md`: WinGet short form (`winget install git-fire`) added alongside explicit ID
- [x] `README.md`: git-testkit v0.2.0 mention added to Contributing section

---

## TODO — Code

### 1. cmd package coverage (stretch goal)
- **Current:** ~50%
- **Target:** 60%+
- **Files:** `cmd/root_test.go` or equivalent test files in `cmd/`
- **Add smoke tests for:**
  - `--dry-run` flag path: confirm no git mutations occur
  - `--status` flag path: smoke test that it returns without error on a clean env
  - Plugin loading error path: confirm non-fatal warning behavior (stderr, not exit 1)
- **Constraint:** Do not mock the git binary — use `internal/testutil` helpers or `git-testkit` fixtures

### 2. golangci-lint v2 migration (P0 from ROADMAP)
- **File:** `.golangci.yml` (create or update)
- **Task:** Migrate config to v2 format so the lint job in `ci.yml` can be re-enabled
- **Verify:** Uncomment the lint job and confirm `golangci/golangci-lint-action@v7` passes

---

## TODO — Docs/README

### 3. Security Notes update
- **File:** `README.md`
- **Consider:** Clarify that `BlockOnSecrets` defaults to `true` (blocking auto-commit when secrets detected)
- **Relevant code:** `internal/config/defaults.go:34`, `internal/safety/secrets.go`

---

## Verification Checklist

```bash
go build ./...          # must be clean
go test -race -count=1 ./...  # all green
go vet ./...            # clean
```

Manual smoke:
```bash
git-fire --dry-run --path /tmp   # no mutations
git-fire --status                # clean output
```

New secret pattern smoke (add to `internal/safety/secrets_test.go`):
```go
// GitLab token should be detected
input := "token: glpat-abcdefghij1234567890"
// assert: scanner.Scan(input) returns findings
```

---

## Notes for Executing Agent

- Launch posts (`docs/launch-posts/`) are excluded from git via `.git/info/exclude` — edit locally, do not commit
- Do not modify `cmd/root.go` or any executor/git internals — stable post-PR-71
- `SecretPattern` struct has no `Severity` field — do not add one
- `go vet` already runs in the `test` CI job — lint task is separate from vet
