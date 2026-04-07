# Documentation Index

This directory is the docs hub for OSS-facing material.

Command forms used throughout docs: `git-fire` and `git fire` are equivalent when `git-fire` is on your PATH.

## README Map

- Product overview, install paths, and first-run commands: [../README.md](../README.md)
- Build from source (Linux/macOS/Windows): [BUILD_FROM_SOURCE.md](BUILD_FROM_SOURCE.md)
- Release process and channel checks: [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md)
- Maintainer package-manager runbooks:
  - [HOMEBREW_RELEASE_RUNBOOK.md](HOMEBREW_RELEASE_RUNBOOK.md)
  - [WINGET_RELEASE_RUNBOOK.md](WINGET_RELEASE_RUNBOOK.md)

## Start Here

- Project entrypoint: [../README.md](../README.md)
- Project snapshot and architecture reference: [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)
- Behavior spec (semantics and edge cases; README + code win if drift): [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- Contributor workflow: [../CONTRIBUTING.md](../CONTRIBUTING.md)

## Guides

- Agent workflows and automation patterns: [agentic-flows.md](agentic-flows.md)
- Security and operations checkpoint workflows: [security-ops.md](security-ops.md)
- Persistent repository registry internals: [REGISTRY.md](REGISTRY.md)
- Manual smoke fixture setup for OSS testers: [MANUAL_SMOKE_FIXTURES.md](MANUAL_SMOKE_FIXTURES.md)
- Build/install from source by platform: [BUILD_FROM_SOURCE.md](BUILD_FROM_SOURCE.md)
- Launch copy and channel playbook: [LAUNCH_POSTS_PLAYBOOK.md](LAUNCH_POSTS_PLAYBOOK.md)
- Planned USB mode scope and non-claims: [USB_MODE.md](USB_MODE.md)
- Homebrew release runbook (maintainers): [HOMEBREW_RELEASE_RUNBOOK.md](HOMEBREW_RELEASE_RUNBOOK.md)
- WinGet release runbook (maintainers): [WINGET_RELEASE_RUNBOOK.md](WINGET_RELEASE_RUNBOOK.md)
- Tagged release checklist (maintainers): [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md)
- Plugin examples: [../examples/plugins/s3-upload.md](../examples/plugins/s3-upload.md)

## Reference

- Plugin architecture and supported types: [../PLUGINS.md](../PLUGINS.md)
- Requirements validation matrix: [REQUIREMENTS_VALIDATION.md](REQUIREMENTS_VALIDATION.md)
- Security reporting policy: [../SECURITY.md](../SECURITY.md)

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
  - Beta-branch-only readiness planning docs were removed after integration; use Git history/PR #69 for historical traceability.

