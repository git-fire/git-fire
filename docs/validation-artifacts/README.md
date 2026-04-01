# Validation Artifacts

This directory holds timestamped validation run outputs.

## Retention policy

- Keep: concise, human-readable run summaries (`VALIDATION_REPORT.md`)
- Ignore by default: raw logs and machine outputs (`*.log`, `*.json`, `*.txt`, and `phase*` folders)

## Why

Raw validation output is useful during execution but creates doc noise and repository bloat for OSS consumers.
The report files preserve audit value without committing transient command output.

