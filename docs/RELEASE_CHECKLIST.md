# Release Checklist

Use this checklist for every tagged release.

## Channel guidance (what to announce)

- Announcing as `alpha` is valid when tags are prerelease (for example `v0.1.0-alpha.1`).
- Announcing as `beta` is valid when tags are prerelease (for example `v0.1.0-beta.1`).
- Announcing as `stable` requires a plain SemVer tag (`vX.Y.Z`) and full verification.
- First stable is usually `v0.1.0` when coming from alpha/beta; choose `v0.2.0+` only if you intentionally want to signal a larger scope jump.

## 1) Preconditions

- `main` is green in CI.
- Planned version is final (`vX.Y.Z`) or prerelease (`vX.Y.Z-rc.1`).
- Required secrets are configured:
  - `HOMEBREW_TAP_TOKEN` (stable releases)
  - `WINGET_PAT` (stable releases)
- Homebrew tap repo exists and is writable: `git-fire/homebrew-tap`.
- `microsoft/winget-pkgs` fork exists and is up to date in the account tied to `WINGET_PAT`.

## 2) Tag and Trigger

- Run `.github/workflows/release.yml` via `workflow_dispatch` with the target tag, or push a tag directly.
- Confirm the workflow detects the correct channel:
  - prerelease (`-alpha`, `-beta`, `-rc`) -> binaries only
  - stable (`vX.Y.Z`) -> binaries + Homebrew + deb/rpm assets

## 3) Verify Release Assets

In the GitHub Release page, verify:

- `checksums.txt` exists.
- Platform archives exist for Linux/macOS/Windows.
- Stable releases include `.deb` and `.rpm` artifacts.

## 4) Smoke Tests

Run at least one install per channel:

- Homebrew:
  - `brew install git-fire/tap/git-fire`
  - `git-fire --version`
- WinGet:
  - `winget install git-fire.git-fire`
  - `git-fire --version`
- Linux script:
  - `curl -fsSL https://raw.githubusercontent.com/git-fire/git-fire/main/scripts/install.sh | bash`
  - `git-fire --version`
- Linux package:
  - `sudo dpkg -i ./git-fire_<version>_amd64.deb` or `sudo dnf install ./git-fire_<version>_amd64.rpm`
  - `git-fire --version`
- Source build:
  - follow [BUILD_FROM_SOURCE.md](BUILD_FROM_SOURCE.md)

## 5) Failure Handling

- If Homebrew publish fails, keep release assets and rerun release job with corrected token/config.
- If WinGet PR fails, rerun `.github/workflows/winget.yml` with `workflow_dispatch` and target tag.
- If package metadata is wrong, publish a patch tag (`vX.Y.(Z+1)`) rather than mutating existing release assets.

## 6) Launch Posts and Distribution

- Prepare announcement copy from [LAUNCH_POSTS_PLAYBOOK.md](LAUNCH_POSTS_PLAYBOOK.md).
- Confirm any references to release channels match the actual tag class:
  - prerelease (`-alpha`, `-beta`, `-rc`) -> alpha/beta messaging
  - stable (`vX.Y.Z`) -> stable messaging
- Post in a staged wave:
  1. Primary post (`Show HN` or equivalent) with the clearest problem statement
  2. Community posts (Reddit + language communities)
  3. Professional/social channels (X/LinkedIn/Discord)
- Keep one source-of-truth update thread and link it from all follow-up posts.
- Track URLs for each post in your launch notes so comments/bugs can be triaged quickly.
