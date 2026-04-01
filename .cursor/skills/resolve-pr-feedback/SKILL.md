---
name: resolve-pr-feedback
description: >-
  Resolves GitHub PR review feedback: fresh thread scan every run; remote branch
  verified before claiming done; commit+push with replies; explicit deferrals
  only. Use for resolve PR feedback, address review, CodeRabbit, or clearing
  review comments.
---

# Resolve PR feedback (complete workflow)

## Rule zero: fresh scan **and** kept context

**Every invocation:** fetch a **fresh** full view of review data (`gh` / API). Do not treat an older triage or another chat as the thread list.

**Also:** use **conversation context** (what you already changed, commits). Merge: fresh scan → checklist; context → avoid duplicate work.

If `gh` / network fails, say so **immediately**—do not imply “nothing left.”

## Rule one: remote is the source of truth for “fixed”

Reviewers and bots read **`origin` / PR head**, not your working tree.

1. After implementing feedback: **`git fetch`**, reconcile with **`origin/<branch>`** (pull/rebase/ff as appropriate), then **commit** and **`git push`**.
2. **Before** telling the user or posting “addressed” on GitHub: verify the fix exists **on the remote** (e.g. `git show origin/<branch>:path/to/file | rg …` or compare `HEAD` to `origin/<branch>` — they must match after push).
3. If you **cannot** push (permissions, user must push): say so **explicitly** and label the state **“local only, not on PR yet.”** Do not describe the fix as done for the review.

**Failure mode this prevents:** claiming a fix, replying on threads, or arguing with bot output while **the branch still points at an older commit**—wasting everyone’s time.

## Rule two: no silent omissions — deferrals must be obvious

If anything is **not** done, the user sees **why**. Never end a turn with open threads **unmentioned**.

| Situation | Say explicitly |
|-----------|----------------|
| Can’t push / commit | Why; what’s local vs remote |
| Can’t post on GitHub | Paste-ready replies; `gh` commands |
| Fix deferred | Thread; reason; unblock criteria |
| Already in code | **Commit SHA on remote** + file; still offer a one-line reply for the thread |

Skipping GitHub replies requires **code-only** opt-in from the user, or explicit “could not post” + paste-ready text.

## Rule three: own the workflow, don’t blame the bot

If automated review says “not fixed”:

1. Check **Rule one** first (remote SHA, push, right branch).
2. Then **wrong commit** (e.g. bot cited `cafe770`, head is `fda42a8`) — cite **remote** commit and file links.
3. Do **not** lead with “their script is broken” as the main story—lead with **whether the fix is on the PR head** and **what commit proves it**.

## Non-negotiables

1. Checklist from **live** PR data + reconcile with context.
2. Tests (`go test -race -count=1 ./...` or project default) before commit.
3. **Push** before “done” (Rule one).
4. Replies on threads **after** remote matches, or note **local-only** + paste-ready text (Rule two).

## Steps (order matters)

1. `git fetch` + sync local branch with `origin` (stash/commit/pull as needed—**never lose uncommitted work** without user OK).
2. Full PR comment/thread scan → written checklist.
3. Implement + test per item (or defer with reason).
4. **Commit** → **push** → **verify on `origin`** (Rule one).
5. Post thread replies (`gh api …` or paste-ready) with **correct commit SHA / links** (Rule two).
6. Summarize: what shipped, what’s deferred, **remote commit** for fixes.

## UI / layout reviews

Fixed **line counts / overhead** without measuring at **`windowWidth`**: suspect; match **`repoListVisibleCount`**-style lipgloss measurement.

## Why this skill exists

**Partial workflow** (code only, no push, no replies) looks “done” in chat and **isn’t** done on GitHub—**that** is the failure mode, not a flaky comment.
