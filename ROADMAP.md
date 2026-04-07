# Git-Fire Roadmap (Current)

This roadmap tracks current priorities for beta stabilization and near-term follow-up.
It intentionally avoids stale week-by-week timelines and historical metrics.

For historical planning context, see docs listed under "Historical / archive" in [docs/README.md](docs/README.md).

## Current Beta Priorities

### P0 - Trust and release readiness

- [x] Keep README and docs aligned with shipped behavior on `main` (docs overhaul pass).
- [x] Complete docs clarity pass for first-run safety (`--dry-run`, `--path`, trust notes).
- [x] Add/maintain clear security reporting policy (`SECURITY.md`).

### P0 - CI baseline quality

- [ ] Finish `golangci-lint` v2 migration.
- [ ] Re-enable lint job in `.github/workflows/ci.yml`.
- [ ] Keep `go build`, `go vet`, and `go test -race -count=1 ./...` green on PRs.

### P1 - Feature completion

- [x] Plugin CLI auto-loading from config: command plugins defined in `config.toml` are now loaded and fired automatically after each run.
- [ ] `--backup-to` implementation.
- [ ] Webhook plugin runtime path wiring.

### P1 - Operational confidence

- [ ] Expand integration tests around large multi-repo and divergence scenarios.
- [ ] Improve install guidance with stronger checksum verification docs.
- [ ] Continue hardening log/error sanitization in edge cases.

## Near-Term (Beta Track)

- Machine-readable output (`--output=json`, `--output=ndjson`) for agent workflows.
- Planning command (`git-fire plan`) for no-side-effect execution previews.
- Repo targeting flags (`--repos`, `--repos-from-stdin`) for orchestrated workflows.
- Per-repo branch targeting controls (explicit include/ignore branch lists per repository override).
- Config surface expansion pass: expose/document all currently supported repo/global options with examples.

## Longer-Term (Post-Beta)

- MCP server mode.
- Restore/replay tooling from structured logs.
- Additional backup destinations and redundancy layers (including planned USB mode).

## Ideas / Future Exploration

These are not committed to any release — early-stage thinking worth tracking.

- **Arbitrary file capture into portable git repo:** extend git-fire beyond existing git repos to capture arbitrary files and directories (non-repo paths) by initializing a temporary git repo and committing them. Target use case: red team operators and security practitioners who need to stash tooling, notes, findings, or other artifacts into a git-backed portable archive at the end of an engagement — whether those artifacts live in a git repo or not. Natural extension of the USB mode direction.

- **Document and artifact stashing:** companion to the above — a lightweight `git-fire stash <path>` style command that captures any file or directory into a named, portable git repo on a configured destination (remote, USB, or local archive path). Non-destructive, auditable, and consistent with git-fire's safety-first design.

## Notes

- This file is the active roadmap summary.
- Historical deep planning artifacts should stay out of user-facing status claims unless refreshed.

