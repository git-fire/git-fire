# USB Mode (Planned)

`git-fire` USB mode is a planned feature and is not part of current beta behavior.

## Intended Scope

- Support one or more USB/mounted folder backup targets.
- Sync repository backups incrementally using git-native workflows.
- Store per-target metadata under a `.git-fire` marker/config path at the target root.

## Status

- No CLI wiring yet.
- No runtime implementation yet.
- This document exists to avoid ambiguous "coming soon" claims and to track planned scope clearly.

## Current Recommendation

For beta users who need offline redundancy now:

- use `git-fire` for remote checkpointing first
- layer a separate filesystem backup workflow for local/offline copies
