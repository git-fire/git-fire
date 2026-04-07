# PR bots: CodeRabbit vs Cursor Bugbot (TL;DR)

**Repo:** git-fire/git-fire · **Data:** GitHub PR review comments, threads grouped by `in_reply_to_id` · **Bots:** `coderabbitai[bot]` (CodeRabbit), `cursor[bot]` (Bugbot).

## Numbers

| | CodeRabbit | Bugbot |
|--|----------:|-------:|
| Threads started (first inline comment) | 194 | 74 |
| PRs with ≥1 thread | 34 | 11 |
| PRs where **both** ran | 5: #47, #51, #55, #59, #63 | — |

## Overlap

- **Low duplication:** ~**9** same `(PR, file)` pairs with both bots; ~**7** where comment lines are within ~30 lines (same hunk-ish).
- Most findings are **unique to one bot**.

## Caveat

[`.coderabbit.yaml`](../.coderabbit.yaml) has `reviews.auto_review.enabled: false` — CodeRabbit only runs when triggered. **More threads ≠ better**; compare coverage and triggers, not raw counts.

## Takeaway

Both produce **actionable, safety-style** feedback. CodeRabbit is **more verbose** (severity, suggestions, AI prompts). Bugbot is **shorter** and hits **fewer PRs** in this sample. For a fair bake-off, align **when** each runs (e.g. on ready-for-review) and track **false positives** on a small human-graded set.

*Generated 2026-04-07; full export script and JSON lived under a local worktree `.local-pr-bot-report/` (gitignored).*
