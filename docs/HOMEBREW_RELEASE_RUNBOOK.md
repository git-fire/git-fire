# Homebrew Release Runbook

This runbook documents the maintainer workflow for publishing `git-fire` to Homebrew.

## Scope

- Formula name: `git-fire`
- Tap repository: `git-fire/homebrew-tap`
- Automation path: `.goreleaser.stable.yaml` + `.github/workflows/release.yml`

## One-time setup

1. Create repository `git-fire/homebrew-tap` (public).
2. Ensure it has a `Formula/` directory.
3. Create a **fine-grained** GitHub PAT:
   - Resource owner: `git-fire`
   - Repository access: **only** `git-fire/homebrew-tap`
   - Permissions: **Contents: Read and write** (Metadata read is included by default)
   - GoReleaser can push formula commits with this minimal scope.
4. Add that PAT to `git-fire/git-fire` repo secrets as **`HOMEBREW_TAP_TOKEN`**.

**Classic PAT fallback (broader):** if you must use a classic token, prefer **`public_repo`** scope for this public tap—not full `repo` unless you have a specific reason.

## How automation works

- Stable release tags (`vX.Y.Z`) use `.goreleaser.stable.yaml`.
- The stable GoReleaser config updates `Formula/git-fire.rb` in `git-fire/homebrew-tap`.
- Legacy prerelease tags (`-alpha`, `-beta`, `-rc`) use `.goreleaser.yaml` and do not publish package-manager updates.
- Going forward, releases should use plain SemVer tags (`vX.Y.Z`).

## Messaging guidance

- Reserve "stable" announcements for plain SemVer tags (`vX.Y.Z`) after release verification.

## Normal release flow

1. Trigger `.github/workflows/release.yml` with a stable tag (for example `v0.2.0`), or push that tag.
2. Verify the GoReleaser job is green.
3. Verify an update commit appears in `git-fire/homebrew-tap` for `Formula/git-fire.rb`.

## Verification checklist

- `brew tap git-fire/tap`
- `brew install git-fire`
- `git-fire --version`

## Common failures and fixes

- `HOMEBREW_TAP_TOKEN` missing/invalid:
  - Add or rotate the token in repo secrets and rerun release.
- Permission denied to tap repo:
  - Confirm token owner has write access to `git-fire/homebrew-tap`.
- Formula not updated:
  - Check GoReleaser logs for Homebrew pipe errors in the release workflow.
