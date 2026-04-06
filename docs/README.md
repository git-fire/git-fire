# Documentation Index

This directory is the docs hub for OSS-facing material.

Command forms used throughout docs: `git-fire` and `git fire` are equivalent when `git-fire` is on your PATH.

## Start Here

- Project entrypoint: [../README.md](../README.md)
- Project snapshot and architecture reference: [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)
- Behavior spec: [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- Contributor workflow: [../CONTRIBUTING.md](../CONTRIBUTING.md)

## Guides

- Agent workflows and automation patterns: [agentic-flows.md](agentic-flows.md)
- Security and operations checkpoint workflows: [security-ops.md](security-ops.md)
- Persistent repository registry internals: [REGISTRY.md](REGISTRY.md)
- Plugin examples: [../examples/plugins/s3-upload.md](../examples/plugins/s3-upload.md)

## Reference

- Plugin architecture and supported types: [../PLUGINS.md](../PLUGINS.md)
- Requirements validation matrix: [REQUIREMENTS_VALIDATION.md](REQUIREMENTS_VALIDATION.md)

## Documentation changelog

### 2026-03-31 — Spec / guide reality alignment (Phase 4)

- Replaced root `git-.md` with **[GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)** as the implementation-aligned behavior spec (legacy aspirational content removed).
- Documented actual CLI flags, env vars (including: no `GIT_FIRE_CONFIG`; use `--config`), config keys, registry and log paths, runtime modes, and explicit **deferred** items (`--backup-to`, webhook load, plugin auto-run on default CLI, etc.).
- Added **D-05–D-35** coverage table mapping themes to spec sections.
- Corrected historical errors in [REQUIREMENTS_VALIDATION.md](REQUIREMENTS_VALIDATION.md) (quick scan paths, rate limiting, `--config`, plugin docs, env vars).
- [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md): noted `git-fire repos` subcommand.

## Active vs Historical Validation Docs

- Current behavior/source-of-truth:
  - [../README.md](../README.md)
  - [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)
  - [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- Historical validation snapshot and planning context:
  - [REQUIREMENTS_VALIDATION.md](REQUIREMENTS_VALIDATION.md)
  - [VALIDATION_PROGRESS.md](VALIDATION_PROGRESS.md)
  - [FINAL_VALIDATION_PLAN.md](FINAL_VALIDATION_PLAN.md)
  - [UAT_BUGS.md](UAT_BUGS.md)

