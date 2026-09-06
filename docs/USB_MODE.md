# USB Mode

`git-fire` USB mode backs up repositories to one or more local targets (USB devices or regular mounted folders).

**Status: Beta (implemented).** CLI and runtime are wired on current `main`. Human dogfood on a physical USB stick is still recommended before calling the feature generally available.

## Current scope (MVP)

- Repeatable target flag: `--usb <path>` (can be supplied multiple times).
- Target marker/config: `<target>/.git-fire`.
- Strategies:
  - `git-mirror` (default): bare mirror repo (`*.git`) per source repo.
  - `git-clone`: checked-out clone sync mode.
- Optional marker bootstrap: `--usb-init` (or `usb.create_on_first_use = true`).
- Per-target run lock: `<target>/.git-fire.lock`.
- Per-target run manifest: `<target>/git-fire-usb-manifest.json`.
- Resume support: `--usb-resume-last-run` skips repo-target pairs marked successful in the previous manifest.
- Verify support: `--usb-verify` checks destination shape after sync.
- Worker overrides: `--usb-workers`, plus config `usb.workers` / `usb.target_workers`.

## Volume contract (on-disk layout)

Kept in `git-fire` for now (shared harness extraction waits until git-rain restore needs it):

```
<target>/
  .git-fire                 # TOML volume marker (schema_version, layout_dir, strategy, created_at)
  .git-fire.lock            # exclusive run lock (pid + timestamp)
  git-fire-usb-manifest.json
  repos/                    # default layout_dir
    <stable-name>.git       # git-mirror destinations (bare)
    <stable-name>/          # git-clone destinations (worktree + .git)
```

Stable destination names are derived from the source repo path so renames of the parent directory do not collide silently. Registry overrides may replace the relative destination path.

## Config

Add to `~/.config/git-fire/config.toml`:

```toml
[usb]
strategy = "git-mirror"      # git-mirror | git-clone
workers = 1                  # repos processed concurrently
target_workers = 1           # concurrent target sync operations
create_on_first_use = false
sync_policy = "keep"         # keep | prune

[[usb.targets]]
name = "travel-stick"
path = "/media/user/TRAVEL"
enabled = true
```

You can also pass targets via CLI:

```bash
git-fire --usb "/media/user/TRAVEL" --usb "/mnt/backup-folder"
git-fire --usb "/media/user/TRAVEL" --usb-init --dry-run
git-fire --usb "/media/user/TRAVEL" --usb-strategy git-clone --usb-verify
```

## Registry overrides

`repos.toml` entries can override USB behavior per repository:

- `usb_strategy` (`git-mirror` or `git-clone`)
- `usb_repo_path` (destination path relative to target repos root)
- `usb_sync_policy` (`keep` or `prune`)

When an existing registry entry is upserted, these overrides are preserved if the incoming update leaves them empty.

## Notes

- `--fire` + `--usb` is currently not supported (USB runs use batch scan planning, not the TUI stream).
- `sync_policy = "prune"` removes stale destination repos not present in the current planned set for that target.
- USB mode intentionally does not fetch from target remotes during normal operation.
- Integration tests use only local temp dirs / bare repos / `file://` URLs — never GitHub remotes.
