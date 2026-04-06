# WinGet Release Runbook

This runbook documents the human process for publishing `git-fire` updates to WinGet.

## Scope

- Package identifier: `git-fire.git-fire`
- Updater workflow: `.github/workflows/winget.yml`
- Automation engine: `vedantmgoyal9/winget-releaser`

## One-time setup

1. Fork `microsoft/winget-pkgs` under the `git-fire` GitHub account.
2. Create a classic GitHub PAT with `public_repo` scope.
3. Add that PAT as repository secret `WINGET_TOKEN` in `git-fire`.
4. Ensure `git-fire.git-fire` already exists in `microsoft/winget-pkgs` (initial bootstrap PR is manual).

## Normal release flow

1. Publish a GitHub Release from a tag like `v0.1.2-alpha`.
2. The `Publish to WinGet` workflow runs automatically on `release.published`.
3. The workflow submits an update PR via the `git-fire/winget-pkgs` fork.
4. A PR is opened against `microsoft/winget-pkgs` for the new version.

## Manual recovery flow

Use this if the release event failed or you need to replay a publish.

1. Open `Actions` in `git-fire`.
2. Run `Publish to WinGet` with `workflow_dispatch`.
3. Set `release_tag` (example: `v0.1.1-alpha`).
4. Confirm the run completes and opens/updates the expected PR.

## Verify a successful publish

After a release, confirm all of the following:

- `Publish to WinGet` workflow run is green.
- A WinGet PR exists for `git-fire.git-fire` with the new version.
- Installer URLs point to the release assets for that tag.
- SHA256 values match release checksums.

## Common failures and fixes

- `WINGET_TOKEN` missing or invalid:
  - Regenerate a classic PAT with `public_repo` and update the repository secret.
- Fork not found / permission issues:
  - Confirm `git-fire/winget-pkgs` exists and is accessible to the token owner.
- No baseline package:
  - Submit the first `git-fire.git-fire` manifest manually to `microsoft/winget-pkgs`.
- Assets not matched:
  - Ensure Windows artifacts follow `git-fire_<version>_windows_<arch>.zip`.
  - Current workflow regex: `_windows_(386|amd64|arm64)\.zip$`.

## Operational notes

- WinGet merge requires moderator approval in `microsoft/winget-pkgs`.
- Do not edit generated manifest content manually in the automation path unless required by reviewer feedback.
