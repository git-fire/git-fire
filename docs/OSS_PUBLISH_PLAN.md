# OSS Publish Plan (Historical + Current State)

This file previously documented a pre-public migration strategy. `git-fire` is now already public at `github.com/git-fire/git-fire`, so the orphan-history bootstrap instructions are no longer the active workflow.

## Current publish workflow

Use the release automation and runbooks that reflect the live repository:

- Release checklist: [RELEASE_CHECKLIST.md](RELEASE_CHECKLIST.md)
- Homebrew maintainer flow: [HOMEBREW_RELEASE_RUNBOOK.md](HOMEBREW_RELEASE_RUNBOOK.md)
- WinGet maintainer flow: [WINGET_RELEASE_RUNBOOK.md](WINGET_RELEASE_RUNBOOK.md)
- Release workflow definition: [../.github/workflows/release.yml](../.github/workflows/release.yml)

Release flow summary:

1. Tag a version (`vX.Y.Z` or prerelease suffix like `-alpha`, `-beta`, `-rc`).
2. Run the release workflow (or push the tag).
3. Verify release assets (`checksums.txt`, platform archives, stable `.deb`/`.rpm`).
4. Run smoke installs (Homebrew, WinGet, Linux script/package, source build as needed).

## Historical note

The old "single clean initial commit" approach is preserved only as archival context from pre-public planning and should not be used for current releases.
