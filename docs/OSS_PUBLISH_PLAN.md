# OSS Publish Plan

## Strategy: Single Clean Initial Commit

When the project is ready to move to the public OSS repository, the entire current commit history will be squashed into a single initial commit. This keeps the public repo clean and avoids exposing internal dev history (planning docs, AI-assisted iteration, intermediate states).

**Steps when ready to publish:**

1. Create a new public GitHub repo
2. Check out this repo at the desired state (tip of the release-ready branch)
3. Create a fresh git history:
   ```bash
   git checkout --orphan initial
   git add -A
   git commit -m "Initial commit"
   git push <new-public-remote> initial:main
   ```
4. Add install/release workflow (GoReleaser or similar) for binary distribution
5. Update install scripts once first tagged release exists

## What ships in the initial commit

- All source code and tests
- README, CONTRIBUTING, Makefile, CI workflow
- Scripts (`install.sh`, `emergency.sh`) — note: these reference GitHub Releases, so a `v0.1.0` tag must be created immediately after pushing so the install path works
- The `docs/` directory can be cleaned up or omitted before the initial commit

## What does NOT ship

- Local `.claude/` settings (already in `.gitignore`)
- The built `git-fire` binary (already in `.gitignore`)
- Any `.env` or credential files
