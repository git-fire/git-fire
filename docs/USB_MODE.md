# USB Mode

`git-fire` USB mode backs up repositories to one or more local targets (USB devices or regular mounted folders).

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
```

## Registry overrides

`repos.toml` entries can override USB behavior per repository:

- `usb_strategy` (`git-mirror` or `git-clone`)
- `usb_repo_path` (destination path relative to target repos root)
- `usb_sync_policy` (`keep` or `prune`)

## Notes

- `--fire` + `--usb` is currently not supported.
- `sync_policy = "prune"` removes stale destination repos not present in the current planned set for that target.
- USB mode intentionally does not fetch from target remotes during normal operation.
