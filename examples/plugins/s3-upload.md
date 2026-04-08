# S3 Upload Plugin Example

Upload your repositories to Amazon S3 as a backup strategy.

## Prerequisites

- AWS CLI installed (`aws`)
- AWS credentials configured (`aws configure`)
- S3 bucket created

## Configuration

Add to `~/.config/git-fire/config.toml`:

```toml
[[plugins.command]]
name = "s3-upload"
command = "aws"
args = [
    "s3", "sync",
    "{repo_path}",
    "s3://my-emergency-backups/{repo_name}-{timestamp}/",
    "--exclude", ".git/*",
    "--exclude", "node_modules/*",
    "--exclude", ".venv/*"
]
when = "after-push"
timeout = "10m"

[plugins.command.env]
AWS_PROFILE = "default"
```

## Usage

```bash
# Dry run to preview git-fire actions (plugins are not executed)
git-fire --dry-run

# Execute backup — plugins run automatically after push completes
git-fire
```

## What It Does

1. After git push completes successfully
2. Syncs your scan root directory (all repos) to S3 as a timestamped backup
3. Creates path: s3://bucket/projects-20260407-150405/
4. Times out after 10 minutes if taking too long

## Variables Available

Post-run plugins fire once per session, seeded from the configured scan root:

- `{repo_path}` - Absolute path of the scan root (e.g. /home/user/projects)
- `{repo_name}` - Basename of the scan root (e.g. projects)
- `{branch}` - Branch of the scan root (if it is itself a git repo)
- `{commit_sha}` - HEAD commit of the scan root (if it is itself a git repo)
- `{timestamp}` - Current timestamp (20060102-150405 format)
- `{date}` - Current date (2006-01-02)
- `{time}` - Current time (15:04:05)

## Cost Estimate

S3 Standard storage:
- 1 GB repo = $0.023/month
- 10 GB repo = $0.23/month
- Negligible for emergency backups!

## Alternative: S3 Glacier

For cheaper long-term storage:

```toml
[[plugins.command]]
name = "s3-glacier"
command = "aws"
args = [
    "s3", "sync",
    "{repo_path}",
    "s3://my-backups/{repo_name}-{timestamp}/",
    "--storage-class", "GLACIER_IR"
]
```

Instant retrieval Glacier: ~$0.004/GB/month (5x cheaper!)

## Troubleshooting

**"aws: command not found"**
```bash
# Install AWS CLI
pip install awscli
# OR
brew install awscli
```

**"Access Denied"**
```bash
# Configure credentials
aws configure
```

**Slow uploads**
```bash
# Add to args for faster uploads:
"--only-show-errors",
"--no-progress"
```

## See Also

- [Plugin architecture](../../PLUGINS.md)
- [Documentation index](../../docs/README.md)
