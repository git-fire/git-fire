# WinGet Release Runbook

This runbook documents the maintainer workflow for publishing `git-fire` to WinGet.

## Scope

- Package identifier: `git-fire.git-fire`
- Updater workflow: `.github/workflows/winget.yml`
- Automation engine: `vedantmgoyal9/winget-releaser` (workflow pins a fixed commit SHA for `v2`)
- Channel note: WinGet publishing is for SemVer stable tags (`vX.Y.Z`).

## One-time setup

1. Fork `microsoft/winget-pkgs` under the account that owns the token.
2. Create a classic GitHub PAT with `public_repo` scope. (Fine-grained PATs are not supported by WinGet Releaser; see [vedantmgoyal9/winget-releaser#172](https://github.com/vedantmgoyal9/winget-releaser/issues/172).)
3. Add that PAT as repository secret `WINGET_PAT` in `git-fire/git-fire`.
4. If this repository is owned by a **GitHub Organization** but the fork and PAT live on a **personal** GitHub account, add a repository or organization **Actions variable** `WINGET_FORK_USER` set to that personal username. Otherwise WinGet Releaser defaults `fork-user` to the org name and `komac update --submit` fails with `does not have the correct permissions to execute CreateRef`.
5. Ensure at least one `git-fire.git-fire` version already exists in `microsoft/winget-pkgs` (bootstrap is manual).

## Normal release flow

1. Publish a stable GitHub release tag (for example `v0.2.0`).
2. Run `.github/workflows/winget.yml` via `workflow_dispatch`.
3. Set `release_tag` to the stable tag (example: `v0.2.0`).
4. The workflow opens/updates a PR against `microsoft/winget-pkgs`.
5. Merge proceeds after WinGet maintainers review and approve.

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
- `CreateRef` / permission denied (often names the PAT owner, e.g. `user does not have the correct permissions to execute CreateRef`):
  - For an org-owned upstream repo, set Actions variable `WINGET_FORK_USER` to the GitHub username that owns `microsoft/winget-pkgs` fork and `WINGET_PAT`, then rerun the workflow.
- Fork not found / permission denied:
  - Confirm fork exists and PAT owner has access.
- Asset pattern mismatch:
  - Ensure release contains assets matching `_windows_(amd64|arm64|386)\.zip$`.
