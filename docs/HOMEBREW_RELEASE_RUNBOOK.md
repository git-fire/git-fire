# Homebrew Release Runbook

This runbook documents how to publish and maintain `git-fire` via Homebrew.

## Scope

- Formula name: `git-fire`
- Tap repository: `git-fire/homebrew-tap`
- Automation path: `.goreleaser.yaml` + `.github/workflows/release.yml`

## There is no Homebrew account signup

Homebrew itself does not require an account.
Publishing uses GitHub repositories and tokens.

You need:
- A GitHub account/org that owns `git-fire/git-fire` and `git-fire/homebrew-tap`.
- Homebrew installed locally for verification.

## One-time setup

1. Create tap repo `git-fire/homebrew-tap` on GitHub.
2. Add a `Formula/` directory in the tap repo.
3. Create a classic GitHub PAT with `repo` scope for the account that can push to `git-fire/homebrew-tap`.
4. Add the PAT to `git-fire` repo secrets as `HOMEBREW_TAP_TOKEN`.

## How automation works

- `.goreleaser.yaml` defines a `brews` pipe that writes Formula updates to `git-fire/homebrew-tap`.
- `release.yml` auto-detects whether `HOMEBREW_TAP_TOKEN` exists:
  - If present: full release including Homebrew publish.
  - If missing: release still succeeds and runs with `--skip=homebrew`.

## Release flow

1. Create/publish a release tag (for example `v0.1.2-alpha`) using existing release workflow.
2. GoReleaser builds artifacts, creates GitHub release assets, and updates the Homebrew formula.
3. Verify commit appears in `git-fire/homebrew-tap`.

## Human verification checklist

- Confirm release workflow is green.
- Confirm `git-fire/homebrew-tap` has updated `Formula/git-fire.rb`.
- On macOS/Linux:
  - `brew tap git-fire/tap`
  - `brew install git-fire`
  - `git-fire --version`

## Common failures and fixes

- Missing `HOMEBREW_TAP_TOKEN`:
  - Homebrew publish is skipped by design; add the secret and rerun release.
- Permission denied to tap repo:
  - Ensure PAT owner has write access to `git-fire/homebrew-tap`.
- Formula did not update:
  - Check release workflow logs for GoReleaser Homebrew pipe output.
