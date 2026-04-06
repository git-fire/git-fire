# Homebrew Release Runbook

This runbook documents the maintainer workflow for publishing `git-fire` to Homebrew.

## Scope

- Formula name: `git-fire`
- Tap repository: `git-fire/homebrew-tap`
- Automation path: `.goreleaser.stable.yaml` + `.github/workflows/release.yml`

## One-time setup

1. Create repository `git-fire/homebrew-tap` (public).
2. Ensure it has a `Formula/` directory.
3. Create a classic GitHub PAT with `repo` scope for an account that can push to `git-fire/homebrew-tap`.
4. Add that PAT to `git-fire/git-fire` repo secrets as `HOMEBREW_TAP_TOKEN`.

## How automation works

- Stable release tags (`vX.Y.Z`) use `.goreleaser.stable.yaml`.
- The stable GoReleaser config updates `Formula/git-fire.rb` in `git-fire/homebrew-tap`.
- Prerelease tags (`-alpha`, `-beta`, `-rc`) use `.goreleaser.yaml` and do not publish package-manager updates.

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
