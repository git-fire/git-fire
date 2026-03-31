# Beta Execution Strategy (All Outstanding Work)

This document defines the practical strategy to complete all beta readiness work without losing momentum, while preserving safety for a repo-mutation tool.

## Objectives

- Ship beta with no known P0 safety/correctness defects.
- Keep product-scope decisions explicit and traceable.
- Separate high-risk experiments from beta-critical fixes.
- Ensure every deferred item is tracked in a durable roadmap.

## Source of Truth

- Decision log and findings: `BETA_BLOCKERS_PROGRESS.md`
- Release-readiness summary: `BETA_READINESS_REPORT.md`
- Deferred/future backlog: `ROADMAP.md`

## Operating Model

- Use `beta-readiness-audit` as the coordination branch.
- Use short-lived feature branches per phase.
- Merge phases in order, with verification gates between phases.
- Keep experiments isolated so they can be abandoned safely.

## Branching Strategy

- **Coordination branch:** `beta-readiness-audit`
- **Phase 1 (P0 blockers):** `fix/beta-blockers-p0`
- **Phase 2 (P1 safety/correctness):** `fix/beta-safety-p1`
- **Phase 3 (scan UX followups):** `fix/scan-ux-bugs`
- **Phase 4 (docs reality alignment):** `docs/spec-reality-alignment`
- **Parallel push experiment:** `feat/parallel-push-experiment` (isolated, disposable)

## Execution Phases

## Phase 1: Beta Blockers (must complete before beta)

- Fix P0 safety/correctness items first.
- Apply resolved decisions in implementation:
  - `--backup-to` deferred feature with explicit beta behavior.
  - `--fire --dry-run` mutually exclusive.
  - No new pre-run prompt for beta path.
- Align CRITICAL docs with actual behavior.

**Gate to exit Phase 1**
- All Phase 1 items merged.
- Tests for touched areas pass.
- `make test-race` and `make lint` pass.

## Phase 2: Safety and Correctness (P1)

- Implement P1 code fixes in small reviewable commits.
- Add focused tests for each corrected behavior.
- Keep UI-only behavior changes minimal and deterministic.

**Gate to exit Phase 2**
- P1 fixes merged with passing validation.
- No new regressions in dry-run and fire flows.

## Phase 3: UX Followups

- Implement low-risk scan UX fixes.
- Defer architectural redesign variants to post-beta unless clearly needed.

**Gate to exit Phase 3**
- UX issues fixed without affecting core backup logic.

## Phase 4: Documentation Alignment

- Resolve all high/medium/low mismatches after code behavior is stable.
- Ensure docs describe current shipped behavior only.
- Keep future ideas clearly marked as roadmap/follow-up.

**Gate to exit Phase 4**
- No known docs-vs-code contradictions in core commands/config.

## Parallel Experiment Lane (Non-blocking)

- Run parallel push investigation only in `feat/parallel-push-experiment`.
- Time-box the effort.
- If complexity/risk is high, close experiment and keep sequential push docs.
- If successful, promote as a separate scoped change with focused tests.

## Work Management Rules

- Every work item must map to one of:
  - Phase task in `BETA_BLOCKERS_PROGRESS.md`
  - Deferred item in `ROADMAP.md`
  - Experiment in dedicated branch
- No orphan TODOs in chat only.
- Any descoped item must be explicitly recorded with rationale.

## Quality and Safety Gates

- For each phase:
  - Run targeted package tests for changed areas.
  - Run `make test-race`.
  - Run `make lint`.
- Require clear rollback path for risky git-operation changes.
- Prefer behavior-preserving refactors with tests over broad rewrites.

## Ownership and Delegation Pattern

- Delegate by phase or tightly scoped vertical slices.
- Keep each delegated branch focused on one cluster of related issues.
- Require each delegate to return:
  - What changed
  - What was tested
  - What remains and why

## Definition of Beta-Ready

- No unresolved P0 issues.
- Agreed decisions implemented or explicitly deferred with tracking.
- Critical docs aligned to actual CLI/config behavior.
- Known non-beta items captured in `ROADMAP.md`.
- Branch is stable, reproducible, and merge-ready.
