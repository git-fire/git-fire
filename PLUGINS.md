# Git-Fire Plugin Architecture

Git-fire is designed to be extensible beyond just git operations. This document describes how to add external tools, remote services, and custom backup strategies.

## Philosophy

While git-fire is built around git, **emergencies don't respect tool boundaries**. Sometimes you need to:
- Upload to S3 as a redundant backup
- Trigger a remote backup service
- Call a company-specific backup script
- Notify your team
- Create offline copies

The plugin system makes git-fire a **general-purpose emergency data evacuation tool**.

---

## Plugin Types

### 1. Command Plugins (Simplest)

Execute external commands or scripts:

```toml
[[plugins.command]]
name = "s3-backup"
command = "aws"
args = ["s3", "sync", "{repo_path}", "s3://emergency-backups/{repo_name}"]
when = "after-push"  # before-push, after-push, on-failure

[[plugins.command]]
name = "notify-team"
command = "/usr/local/bin/slack-notify.sh"
args = ["Emergency backup completed for {repo_name}"]
when = "after-push"

[[plugins.command]]
name = "create-tarball"
command = "tar"
args = ["czf", "/backups/{repo_name}-{timestamp}.tar.gz", "-C", "{repo_path}", "."]
when = "before-push"
```

**Variables available:**
- `{repo_path}` - Full path to repository
- `{repo_name}` - Repository directory name
- `{timestamp}` - ISO8601 timestamp
- `{branch}` - Current branch name
- `{commit_sha}` - Latest commit SHA

---

### 2. Go Plugins (Most Powerful)

Write plugins in Go that integrate deeply:

```go
// plugins/s3_backup/main.go
package main

import (
    "github.com/TBRX103/git-fire/internal/plugins"
)

type S3BackupPlugin struct {
    Bucket    string
    Region    string
    Compress  bool
}

func (p *S3BackupPlugin) Name() string {
    return "s3-backup"
}

func (p *S3BackupPlugin) Execute(ctx plugins.Context) error {
    // Create tarball if needed
    archivePath := ""
    if p.Compress {
        archivePath = createTarball(ctx.RepoPath)
        defer os.Remove(archivePath)
    }

    // Upload to S3
    sess := session.NewSession(&aws.Config{
        Region: aws.String(p.Region),
    })

    uploader := s3manager.NewUploader(sess)

    file, err := os.Open(archivePath)
    if err != nil {
        return err
    }
    defer file.Close()

    key := fmt.Sprintf("%s-%s.tar.gz",
        ctx.RepoName,
        time.Now().Format("20060102-150405"))

    _, err = uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(p.Bucket),
        Key:    aws.String(key),
        Body:   file,
    })

    if err == nil {
        ctx.Logger.Success("Uploaded to S3",
            fmt.Sprintf("s3://%s/%s", p.Bucket, key))
    }

    return err
}

func (p *S3BackupPlugin) Validate() error {
    if p.Bucket == "" {
        return fmt.Errorf("bucket is required")
    }
    return nil
}

// Plugin registration
func init() {
    plugins.Register(&S3BackupPlugin{})
}
```

**Build and install:**
```bash
cd plugins/s3_backup
go build -buildmode=plugin -o ~/.config/git-fire/plugins/s3_backup.so

# Load in config
[plugins]
load = ["s3_backup.so"]

[plugins.s3-backup]
bucket = "my-emergency-backups"
region = "us-east-1"
compress = true
```

---

### 3. HTTP/Webhook Plugins

Call remote services:

```toml
[[plugins.webhook]]
name = "company-backup-service"
url = "https://backup.company.com/api/v1/emergency"
method = "POST"
headers = { "Authorization" = "Bearer ${BACKUP_TOKEN}" }
body = '''
{
  "repo_name": "{repo_name}",
  "repo_path": "{repo_path}",
  "timestamp": "{timestamp}",
  "branches": "{branches}",
  "commit_sha": "{commit_sha}",
  "urgency": "high"
}
'''
when = "after-push"
timeout = "30s"

[[plugins.webhook]]
name = "slack-notification"
url = "${SLACK_WEBHOOK_URL}"
method = "POST"
body = '''
{
  "text": "🔥 Emergency backup: {repo_name}",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "*Repository:* {repo_name}\n*Branch:* {branch}\n*Status:* Backed up successfully"
      }
    }
  ]
}
'''
```

---

### 4. Remote Backup Services

Integrate with existing backup solutions:

```toml
[[plugins.restic]]
repository = "/backups/restic"
password_file = "~/.restic-password"
tags = ["git-fire", "emergency"]

[[plugins.rclone]]
remote = "backup-s3"
path = "git-fire-backups"
flags = ["--fast-list", "--transfers=32"]

[[plugins.borgbackup]]
repository = "ssh://backup-server/~/backups"
passphrase_command = "pass show backup/borg"
```

---

## Implementation Plan

### Phase 1: Command Plugins (Week 1)

**Files to create:**
```
internal/plugins/
├── types.go        # Plugin interfaces
├── command.go      # Command executor plugin
├── registry.go     # Plugin registration
└── loader.go       # Config → Plugin loader
```

**Core types:**
```go
type Plugin interface {
    Name() string
    Execute(Context) error
    Validate() error
}

type Context struct {
    RepoPath   string
    RepoName   string
    Branch     string
    CommitSHA  string
    Timestamp  string
    DryRun     bool
    Logger     Logger
    Config     map[string]interface{}
}

type CommandPlugin struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
    When    Trigger  // before-push, after-push, on-failure
    Timeout time.Duration
}
```

### Phase 2: Go Plugins (Week 2)

Add plugin loading via Go's `plugin` package:

```go
// Load .so files from ~/.config/git-fire/plugins/
func LoadGoPlugins() error {
    pluginDir := filepath.Join(os.UserHomeDir(), ".config/git-fire/plugins")

    files, _ := filepath.Glob(filepath.Join(pluginDir, "*.so"))

    for _, file := range files {
        p, err := plugin.Open(file)
        if err != nil {
            continue
        }

        symbol, err := p.Lookup("Plugin")
        if err != nil {
            continue
        }

        plugin := symbol.(Plugin)
        Register(plugin)
    }
}
```

### Phase 3: Webhook/HTTP Plugins (Week 3)

REST API integration:

```go
type WebhookPlugin struct {
    URL     string
    Method  string
    Headers map[string]string
    Body    string
    Timeout time.Duration
}

func (p *WebhookPlugin) Execute(ctx Context) error {
    // Template substitution
    body := expandVars(p.Body, ctx)

    req, _ := http.NewRequest(p.Method, p.URL, strings.NewReader(body))

    for k, v := range p.Headers {
        req.Header.Set(k, expandVars(v, ctx))
    }

    client := &http.Client{Timeout: p.Timeout}
    resp, err := client.Do(req)

    // Handle response...
    return err
}
```

---

## Real-World Examples

### Example 1: S3 + Slack Notification

```toml
[plugins]
enabled = ["s3-backup", "slack-notify"]

[[plugins.command]]
name = "s3-backup"
command = "aws"
args = ["s3", "sync", "{repo_path}", "s3://emergency/{repo_name}-{timestamp}/"]
when = "after-push"

[[plugins.webhook]]
name = "slack-notify"
url = "${SLACK_WEBHOOK}"
body = '{"text": "✅ {repo_name} backed up to S3"}'
when = "after-push"
```

### Example 2: Company Backup Service

```toml
[[plugins.webhook]]
name = "company-backup"
url = "https://backup.company.com/api/git-fire"
method = "POST"
headers = { "X-API-Key" = "${COMPANY_BACKUP_KEY}" }
body = '''
{
  "repo": "{repo_name}",
  "path": "{repo_path}",
  "emergency": true,
  "contact": "user@company.com"
}
'''
timeout = "60s"
```

### Example 3: Multi-Strategy Backup

```toml
# Belt and suspenders approach
[plugins]
enabled = ["git-push", "s3-backup", "rsync-nas", "usb-backup"]

# Normal git push
[[plugins.git-push]]
# (built-in, always enabled)

# Cloud backup
[[plugins.command]]
name = "s3-backup"
command = "rclone"
args = ["sync", "{repo_path}", "s3:emergency/{repo_name}"]

# Local NAS
[[plugins.command]]
name = "rsync-nas"
command = "rsync"
args = ["-av", "{repo_path}/", "nas.local:/backups/git-fire/{repo_name}/"]

# USB drive if mounted
[[plugins.command]]
name = "usb-backup"
command = "sh"
args = ["-c", "test -d /media/usb && cp -r {repo_path} /media/usb/backups/"]
on_failure = "ignore"  # Don't fail if USB not mounted
```

---

## CLI Integration

```bash
# List available plugins
git-fire --list-plugins

# Run with specific plugins
git-fire --plugin s3-backup --plugin slack-notify

# Disable plugins for this run
git-fire --no-plugins

# Test plugin configuration
git-fire --test-plugin s3-backup --dry-run

# Show plugin execution plan
git-fire --dry-run --show-plugins
```

---

## Security Considerations

1. **Credential Management**
   - Use environment variables, not config files
   - Support credential helpers (e.g., `pass`, `1password`)
   - Warn about plaintext secrets in config

2. **Plugin Validation**
   - Verify plugin signatures (for Go plugins)
   - Sandbox command plugins (limit file access)
   - Require explicit plugin enable in config

3. **Dry-Run Support**
   - All plugins MUST respect `DryRun` flag
   - Show what would be executed
   - Validate credentials without executing

---

## Future Ideas

### Plugin Marketplace
- `git-fire plugin install s3-backup`
- Community-contributed plugins
- Plugin ratings and reviews

### Plugin Templates
```bash
git-fire plugin create my-backup-plugin --template=command
# Generates plugin scaffold
```

### Plugin Composition
```toml
# Run plugins in parallel
[plugins.parallel]
plugins = ["s3-backup", "glacier-backup", "nas-backup"]

# Run plugins in sequence
[plugins.sequence]
plugins = ["create-tarball", "encrypt-tarball", "upload-tarball"]
```

---

## Getting Started

**To add your first plugin:**

1. Create config file:
   ```bash
   git-fire --init
   ```

2. Add a simple command plugin:
   ```toml
   [[plugins.command]]
   name = "local-backup"
   command = "cp"
   args = ["-r", "{repo_path}", "/backups/{repo_name}-{timestamp}"]
   when = "after-push"
   ```

3. Test it:
   ```bash
   git-fire --dry-run --plugin local-backup
   ```

4. Run for real:
   ```bash
   git-fire
   ```

**Your backup strategy, your tools, your rules.** 🔥
