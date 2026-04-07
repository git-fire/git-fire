# CodeRabbit — maintainer feedback report (git-fire)

**Audience:** CodeRabbit product / success / engineering  
**Project:** [git-fire/git-fire](https://github.com/git-fire/git-fire) — Go CLI, safety- and git-ops–heavy  
**Date:** 2026-04-08  
**Companion:** [PR_BOT_REVIEW_TLDR.md](PR_BOT_REVIEW_TLDR.md) (quant snapshot)

This report mixes **observed PR comment data** with **maintainer workflow experience**. It is not a controlled A/B test.

---

## Executive summary

On git-fire, **both** CodeRabbit and Cursor Bugbot produce **useful, often high-signal** inline review feedback on correctness, safety, and tests. **Thread volume and PR coverage are not comparable** without normalizing triggers: CodeRabbit is configured with **`auto_review: false`** in this repo, while Bugbot has been effectively **“always on” for whatever we throw at it”** from the maintainer’s perspective.

The **largest practical gap for CodeRabbit** here is **operational**: hitting **review limits when shipping many small PRs** (common in multi-agent / iterative flows). That pushed us toward **fewer, larger feature branches**, which is the opposite of how we’d like to work and may **hurt** review quality (bigger diffs, more context, more noise).

**Concrete ask:** explore **higher limits or metering tuned for small PRs**, and **first-class “small change” review mode** (narrow diff, aggressive de-noising, fast path) so maintainers don’t have to batch work to stay under caps.

---

## What we measured vs what we did not

| Measured (approx.) | Not measured (honest) |
|--------------------|-------------------------|
| Inline review **threads** per bot (GitHub API, thread roots via `in_reply_to_id`) | Ground-truth **defects** in merged code |
| **Overlap** of both bots on same PR / file / nearby lines | **False negative** rate (“missed bugs”) |
| Subjective **quality** from spot-checking representative threads | Statistical **false positive** rate |
| Maintainer **workflow friction** (limits, branch sizing) | Identical **trigger parity** (CodeRabbit often manual/triggered here) |

We **cannot** fairly claim “Bugbot caught X% more bugs” or “CodeRabbit missed Y” without a labeled defect set. The scores below are **judgment + ergonomics**, not recall.

---

## Scoring rubric (1–5)

| Dimension | Meaning |
|-----------|--------|
| **Feedback quality** | Actionable, correct-on-diff, appropriate severity, not nitpick-heavy |
| **Signal / noise** | Useful findings vs style/doc churn (given our `.coderabbit.yaml` anti-nitpick instructions) |
| **Operational fit** | How often we can use the tool without changing how we branch or batch PRs |
| **Coverage in practice** | How much of our real PR stream gets reviewed **given limits + triggers** |
| **Composability** | Works well with **small, frequent PRs** and agentic iteration |

---

## Scores (git-fire maintainer view)

Scale: **1** poor · **3** acceptable · **5** excellent.

| Dimension | CodeRabbit | Cursor Bugbot | Notes |
|-----------|:----------:|:-------------:|-------|
| Feedback quality | **4** | **4** | Both surface real safety/correctness issues; CodeRabbit often richer (severity, suggested patches, agent prompts). |
| Signal / noise | **3.5** | **4** | CodeRabbit is good after config, still occasionally verbose; Bugbot tends to shorter, issue-shaped comments. |
| Operational fit | **2.5** | **4.5** | **Rate limits on PR volume** made us batch work → larger PRs → worse iteration. Bugbot did not impose that friction in our usage. |
| Coverage in practice | **3**\* | **4**\* | \*CodeRabbit **auto_review off** + limits skew this; Bugbot reviewed more of our stream **without** us reshaping workflow. |
| Composability (small PRs) | **2.5** | **4.5** | Small PRs are ideal for review, but **limits push us away** from that model; Bugbot **“takes whatever you throw at it.”** |

**Overall (weighted toward maintainer reality):** CodeRabbit **~3.3 / 5**, Bugbot **~4.2 / 5** for *this repo and workflow*. If rate limits and triggers were aligned with small-PR cadence, CodeRabbit’s **quality** score could plausibly **match or exceed** Bugbot while keeping structured output.

---

## What CodeRabbit does well (please keep)

- **Structured reviews** (severity, repro context, committable suggestions) help triage.
- **Path-based instructions** (e.g. `internal/executor/**`, `internal/safety/**`) steer toward **risk**, not bikeshedding — matches an emergency backup tool.
- **Low duplicate findings vs Bugbot** on the same hunks (~7 proximate overlaps in our export): bots are **not** mostly re-stating each other.

---

## Improvement opportunities (product + positioning)

### 1. Rate limits vs small, frequent PRs (high impact)

**Observed pain:** We **frequently hit CodeRabbit’s cap on PRs per time window** when using **small, multi-step / multi-agent flows**. That forced **longer-lived feature branches** and **bigger PRs** to amortize reviews — which increases diff size, review fatigue, and the chance of noisy or conflicting feedback.

**Hypothesis for CodeRabbit:** **Smaller PRs** often mean **simpler diffs** and **better** automated review outcomes — if the product is tuned for them. Today, **pricing/metering** can **invert** that incentive.

**Suggestions (non-prescriptive):**

- **Tier or burst allowances** for OSS / small repos or for **small diffs** (e.g. lines changed below N, or file count).
- **Per-PR “micro review”** mode: cheaper/faster, fewer comments, optimized for **agent-sized** changes.
- Clear dashboard surfacing **remaining quota** and **soft guidance** (“3 small PRs left this window”) so teams don’t discover limits mid-flow.

### 2. “Small change” excellence

If CodeRabbit explicitly optimized for **narrow deltas** (single concern, few files), it could **outcompete** generic bots on **signal/noise** for the way many AI-assisted workflows ship code.

### 3. Trigger transparency

When **`auto_review` is off**, comparisons to always-on tools are **unfair** in benchmarks. Documenting **recommended defaults** for OSS safety-critical repos (without drowning maintainers) would help set expectations.

---

## Cursor Bugbot — brief counterpart notes (for context only)

- **Strength:** Low friction; **no** analogous pressure to merge small PRs into mega-PRs.
- **Tradeoff:** Fewer structured artifacts; less configurability than CodeRabbit’s instruction stack in our experience.

---

## Appendix — quantitative snapshot (see TL;DR)

From a one-time export of inline review threads (see [PR_BOT_REVIEW_TLDR.md](PR_BOT_REVIEW_TLDR.md)):

- CodeRabbit (`coderabbitai[bot]`): **194** threads, **34** PRs with activity.  
- Bugbot (`cursor[bot]`): **74** threads, **11** PRs.  
- **5** PRs had **both**; **low hunk-level duplication**.

---

*Prepared by the git-fire maintainer for a CodeRabbit conversation; scores are subjective and workflow-dependent.*
