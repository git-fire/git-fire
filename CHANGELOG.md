# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- **Plugin auto-loading:** command plugins defined under `[[plugins.command]]` in `config.toml` are now loaded and executed automatically after each run. Plugins fire on `on-success`, `on-failure`, or `always` triggers based on run outcome. Dry-run and user-aborted runs skip plugin execution.

### Fixed

- **`--init` + `--config`:** `git-fire --init` now writes the example config to the path given by `--config` when set, instead of always using the default user config path.

## [0.1.0-alpha] — 2026-04-01

Alpha release: multi-repo scan, checkpoint/push flows, registry, TUI selector (`--fire`), and safety rails. See README for known limitations (`--backup-to`, USB destination mode).

[Unreleased]: https://github.com/git-fire/git-fire/compare/v0.1.0-alpha...HEAD
[0.1.0-alpha]: https://github.com/git-fire/git-fire/releases/tag/v0.1.0-alpha
