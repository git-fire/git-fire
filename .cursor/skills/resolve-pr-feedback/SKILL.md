---
name: resolve-pr-feedback
description: >-
  Resolves GitHub PR feedback: fresh thread scan; remote verified before “done”;
  comment on every actionable and deferred item; user-input items stay in chat
  until agreed, then post. Use for resolve PR feedback, address review,
  CodeRabbit, or clearing review comments.
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

**Failure mode this prevents:** claiming a fix or replying while **the branch still points at an older commit**.

## Rule two: comment on every actionable and deferred item

For **each** review thread (or each distinct actionable bullet), leave **visibility on GitHub**—after the work matches **Rule one**:

| Outcome | GitHub |
|--------|--------|
| **Addressed** | Thread reply: what changed, **remote commit SHA**, file/behavior pointer. |
| **Deferred** (won’t do now, out of scope, blocked) | Thread reply: **deferred**, short reason, what would unblock or “won’t fix” rationale. |

Do **not** leave actionable or deferred threads **without a reply** once you’ve decided the outcome—unless **Rule three** applies.

**Chat:** Still summarize the checklist for the user; GitHub comments are for reviewers/bots/history.

## Rule three: user-input items — agree in chat first, **then** comment

If a thread needs a **product/design/call** the user must make (multiple valid approaches, risk trade-off, “should we…?”):

1. **Do not** post a final position on GitHub until you and the user **agree** in chat on what to do (or that you’ll defer / won’t fix).
2. In chat: restate options, recommend if asked, get explicit alignment (“we’ll do A” / “defer” / “reply won’t fix because…”).
3. **After** agreement: implement if needed, **push**, then post the **single** agreed thread reply (or deferral) on GitHub.

**Failure mode this prevents:** committing the repo to a direction on the PR **before** the maintainer decided—then having to walk it back publicly.

If you truly cannot reach the user: post a **neutral** note only if necessary (“Following up in thread — need maintainer input on X”)—not a fake resolution.

## Rule four: no silent omissions (chat + GitHub)

If anything is **not** done, the user sees **why** in chat. Rules two–three govern **when** GitHub gets a comment.

| Situation | Say explicitly (chat) | GitHub (after Rule one) |
|-----------|------------------------|---------------------------|
| Can’t push | Why; local vs remote | Paste-ready text; or wait until push |
| Can’t post | Paste-ready + `gh` commands | — |
| Needs user input | Options; wait for agreement (Rule three) | **After** agreement per Rule three |

Skipping GitHub replies requires **code-only** opt-in, or “could not post” + paste-ready text.

## Rule five: own the workflow, don’t blame the bot

If automated review says “not fixed”:

1. Check **Rule one** first (remote SHA, push, right branch).
2. Then wrong **diff base** — cite **remote** commit and links.
3. Do **not** lead with “their script is broken”—lead with **whether the fix is on PR head**.

## Non-negotiables

1. Checklist from **live** PR data + context.
2. Tests before commit.
3. **Push** + verify **origin** before “done” (Rule one).
4. **Per-thread** replies for **addressed** and **deferred** (Rule two); **user-input** threads follow Rule three.

## Steps (order matters)

1. `git fetch` + sync with `origin` (stash/commit/pull safely).
2. Full PR scan → written checklist (tag each item: actionable / deferred / **needs user input**).
3. For **needs user input**: discuss in chat → agree → then implement + push + reply.
4. For **actionable**: implement + test → commit → push → verify **origin** → **thread reply**.
5. For **deferred**: **thread reply** (reason) after decision—still after push if other commits land same session.
6. Summarize in chat: shipped SHA, deferred threads, agreed positions.

## UI / layout reviews

Fixed **line counts / overhead** without **`windowWidth`**: suspect; match **`repoListVisibleCount`**-style lipgloss measurement.

## Why this skill exists

Partial workflow (code only, no push, no per-thread visibility) looks “done” in chat and **isn’t**—and **commenting before alignment** on maintainer decisions makes the PR noisy or wrong.
