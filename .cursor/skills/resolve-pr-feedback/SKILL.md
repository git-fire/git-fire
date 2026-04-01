---
name: resolve-pr-feedback
description: >-
  Resolves GitHub PR review feedback: fresh full scan of PR comments every run,
  plus retained chat context; explicit deferrals (never silent); fixes, tests,
  and thread replies. Use for resolve PR feedback, address review, CodeRabbit,
  or clearing review comments.
---

# Resolve PR feedback (complete workflow)

## Rule zero: fresh scan **and** kept context

**Every invocation:** pull a **fresh** full view of review data (`gh` / API)—do not treat an older triage or another chat as the authoritative thread list.

**Also:** keep **conversation context** (what was already fixed, commits, disagreements). Merge the two:

- Fresh scan → current checklist of threads and bodies.
- Context → what landed in code already, so you don’t duplicate work and you can say “already fixed in …” accurately.

Session context **does not** replace the scan; the scan **updates** what still needs a reply or a code change.

If `gh` / network is unavailable, say so **immediately** and ask for a paste or offer commands—do not imply “nothing left” without data.

## Rule one: no silent omissions — deferrals must be **obvious**

If anything is **not** done, the user must see **why** in the agent’s reply. **Never** end a turn leaving open review threads **unmentioned**.

| Situation | Say explicitly |
|-----------|----------------|
| Can’t post on GitHub (no `gh`, auth, sandbox) | Why; paste-ready replies per thread; optional `gh` commands |
| Fix deferred (scope, risk, needs product call) | Which thread; reason; what would unblock |
| Thread already addressed in repo | File/commit; **still** offer a short reply line they can post to resolve the conversation |
| Thread is wrong / won’t fix | Reasoning; suggested reply so the reviewer isn’t left hanging |
| Didn’t run full scan | Say so; don’t guess the thread list |

**Unresolved GitHub conversations without a posted reply** are a failure mode unless the user chose “code only.” If you skip posting, **say you skipped** and give paste-ready text—that is **not** optional silence.

## Non-negotiables

1. **Enumerate from live data** into a written checklist (can merge with context for “done vs open”).
2. **Replies:** post via `gh` when possible; otherwise **paste-ready text + explicit note that nothing was posted to GitHub**.
3. **Pasted comment text** from the user is authoritative—merge with the scan.

## Steps

1. Fresh scan + checklist (Rule zero) + reconcile with context (Rule zero).
2. Per thread: fix, confirm, or **defer with reason** (Rule one).
3. Tests as usual.
4. GitHub: post or **state failure + paste-ready text** (Rule one).

## UI / layout reviews

Hard-coded **line counts / overhead** without `windowWidth`: suspect; compare to `repoListVisibleCount`-style lipgloss measurement.

## Why repeat the full scan

Low cost vs one missed thread. **Rule one** is why the user doesn’t pay twice in confusion: deferrals are visible, not silent.
