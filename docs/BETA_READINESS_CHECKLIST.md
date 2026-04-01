# Beta Readiness Checklist (`path_to_beta` -> `main`)

Use this checklist as the merge gate for promoting beta-ready changes from `path_to_beta` into `main`.

## Must-Fix (block merge until resolved)

- [ ] CI parity gates pass on the PR branch and in GitHub Actions.
- [ ] No open **critical/high** defects with reproducible evidence in backup correctness, conflict handling, or safety guardrails.
- [ ] No unresolved regressions in core flows: default run, `--dry-run`, `--fire`, and `--status`.
- [ ] No docs/spec contradictions that would cause unsafe operator behavior.

## Should-Fix (strongly preferred for merge)

- [ ] Close high-confidence medium-severity bugs found during beta validation.
- [ ] Add tests for newly fixed edge cases in `internal/git`, `internal/executor`, `internal/safety`, or `internal/config`.
- [ ] Reduce known release friction tracked in-repo (example: re-enable `golangci-lint` after v2 config migration in CI).

## Defer (acceptable to ship with caveats)

- Planned roadmap work already documented as not yet wired (for example plugin CLI auto-loading and `--backup-to` in docs).
- UX/polish improvements that do not affect data safety or execution correctness.
- Larger refactors that are not required to preserve current safety and reliability guarantees.

## Current CI Parity Gate Commands

These match `.github/workflows/ci.yml` and should be run locally before merge:

```bash
go build ./...
go vet ./...
go test -race -count=1 ./...
```

Equivalent Make targets:

```bash
make build
make lint
make test-race
```

## Release Decision Rubric (`path_to_beta` -> `main`)

- **Go**: all Must-Fix items are complete, CI parity gates are green, and only Should-Fix/Defer items remain with explicit follow-up tracking.
- **Conditional Go**: no Must-Fix failures, CI is green, and remaining Should-Fix items have documented owner + due milestone in the next beta patch window.
- **No-Go**: any Must-Fix item is open, CI parity is failing/flaky without root-cause disposition, or there is unresolved evidence of backup/safety correctness risk.
