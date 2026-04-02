# KTLO Parallel Execution Playbook

This is the exact operating procedure to execute a large KTLO backlog (for example: 50 markdown-documented addressables) while keeping quality high and progress measurable.

## Goal

- Process 50 items in 5 parallel batches of 10.
- Keep each batch isolated in its own branch/worktree.
- Open 5 PRs per wave, review/merge with explicit gates.

## Non-Negotiable Rules

- Keep `beta-readiness-audit` as the control branch.
- Never mix batch scopes in one branch.
- Every item must have a status and owner before execution starts.
- Every PR must include what changed, what was tested, and what remains.
- If a batch gets unstable, pause that batch only; do not block other batches.

## Canonical Files

- Planning and backlog: `BETA_BLOCKERS_PROGRESS.md`
- Decision and phase strategy: `BETA_EXECUTION_STRATEGY.md`
- Deferred/future tracking: `ROADMAP.md`
- This operating guide: `KTLO_PARALLEL_EXECUTION_PLAYBOOK.md`

## Step 1: Normalize the 50 Items

- Build one normalized list with:
  - Stable item ID (`KTLO-001` ... `KTLO-050`)
  - Source file/path where requirement came from
  - Severity (`P0`, `P1`, `P2`, `P3`)
  - Domain (`git`, `executor`, `config`, `ui`, `docs`, `tests`)
  - Acceptance criteria (1-3 bullets)
  - Dependencies (`none` or item IDs)

- If an item cannot be tested clearly, rewrite it before batching.

## Step 2: Create 5 Batches of 10 (Dependency-Aware)

- Batch by dependency and risk, not by file order.
- Recommended batch policy:
  - Batch A/B: highest-risk correctness and safety first.
  - Batch C: medium-risk behavior and CLI UX.
  - Batch D: docs/spec alignment and low-risk cleanup.
  - Batch E: tests/hardening/follow-ups that do not block core flows.

- Hard rule:
  - No item in a batch may depend on an item in a later batch.
  - If dependencies cross batches, move items to preserve topological order.

## Step 3: Create Execution Branches and Worktrees

- Base all batches from latest `beta-readiness-audit`.
- Branch naming:
  - `fix/ktlo-batch-01`
  - `fix/ktlo-batch-02`
  - `fix/ktlo-batch-03`
  - `fix/ktlo-batch-04`
  - `fix/ktlo-batch-05`

- Use separate worktrees for true parallelism (one worktree per batch).

## Step 4: Agent Assignment Model (Recommended)

- Use one coordinator thread (this branch context) for:
  - prioritization
  - dependency resolution
  - merge order decisions
  - release readiness checks

- Use one execution agent/session per batch for:
  - implementation
  - local validation
  - PR prep

- Do not let one execution agent touch multiple batches.

## Step 5: Per-Batch Definition of Done

Each batch is done only when all are true:

- All 10 batch items are implemented or explicitly descoped with rationale.
- Acceptance criteria checked for each item.
- Tests added/updated for changed behavior.
- Validation commands pass:
  - `make test-race`
  - `make lint`
  - targeted package tests for changed areas
- PR body includes:
  - Item IDs completed
  - Risk notes
  - Test evidence
  - Follow-ups (if any)

## Step 6: PR Wave and Merge Policy

- Open up to 5 PRs in parallel (one per batch).
- Merge order:
  1. Low-dependency safety/correctness batches first
  2. Then medium-risk behavior
  3. Docs and polish last

- If a PR fails CI repeatedly:
  - Time-box fix attempts
  - Split the PR by sub-scope
  - Re-open as smaller PR(s)

## Step 7: Rebase and Conflict Management

- Rebase each in-flight batch branch at least once daily against control branch.
- Resolve conflicts immediately after rebase, then rerun validation gate.
- Never merge with unresolved dependency drift.

## Step 8: Tracking Cadence

- Update tracking docs at least twice per day:
  - `BETA_BLOCKERS_PROGRESS.md`: status by item/batch
  - `ROADMAP.md`: any deferred/new follow-up item

- Use status labels:
  - `todo`
  - `in-progress`
  - `blocked`
  - `in-review`
  - `done`
  - `deferred`

## Step 9: Release Gate (Before Go-Live)

- Confirm no unresolved P0 items.
- Confirm all decisions are reflected in code/docs.
- Confirm no critical docs-vs-code mismatch remains.
- Confirm deferred items are captured in `ROADMAP.md`.
- Run final integrated validation pass from control branch.

## Operator Checklist (Run Every Wave)

- Refresh base branch and pull latest.
- Recompute dependency-safe batching.
- Spin up/update batch branches and worktrees.
- Launch one execution agent per batch.
- Enforce per-batch DoD and validation gate.
- Open PR wave and triage CI/review quickly.
- Merge in dependency order.
- Update blockers + roadmap immediately after merges.

## Practical Advice for Context Windows

- Keep one long-lived coordinator context for planning/governance.
- Use fresh execution contexts for each batch to avoid context bloat.
- When a batch completes, summarize outcomes back into coordinator docs.
