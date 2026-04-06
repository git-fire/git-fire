# WinGet Release Runbook

This runbook documents the maintainer workflow for publishing `git-fire` to WinGet.

## Scope

- Package identifier: `git-fire.git-fire`
- Updater workflow: `.github/workflows/winget.yml`
- Automation engine: `vedantmgoyal9/winget-releaser`

## One-time setup

1. Fork `microsoft/winget-pkgs` under the account that owns the token.
2. Create a classic GitHub PAT with `public_repo` scope.
3. Add that PAT as repository secret `WINGET_PAT` in `git-fire/git-fire`.
4. Ensure at least one `git-fire.git-fire` version already exists in `microsoft/winget-pkgs` (bootstrap is manual).

## Normal release flow

1. Publish a stable GitHub release tag (for example `v0.2.0`).
2. `Publish WinGet Manifest` workflow runs for `release.published`.
3. The workflow opens/updates a PR against `microsoft/winget-pkgs`.
4. Merge proceeds after WinGet maintainers review and approve.

## Manual recovery flow

Use this when the release event did not trigger or you need a replay:

1. Open GitHub Actions in `git-fire/git-fire`.
2. Run `.github/workflows/winget.yml` via `workflow_dispatch`.
3. Set `release_tag` (example: `v0.2.0`).
4. Confirm the run completes and opens/updates the expected WinGet PR.

## Verification checklist

- Workflow run is green.
- WinGet PR exists for `git-fire.git-fire` at the target version.
- Installer URL matches the GitHub release asset.
- SHA256 in manifest matches release checksums.

## Common failures and fixes

- `WINGET_PAT` missing/invalid:
  - Replace with a classic PAT that has `public_repo`.
- Fork not found / permission denied:
  - Confirm fork exists and PAT owner has access.
- Asset pattern mismatch:
  - Ensure release contains assets matching `_windows_(amd64|arm64|386)\.zip$`.
