# Git-Fire Plugins: Status + API Reference

Command plugins defined in config.toml are now auto-loaded and executed automatically after each run. This page documents the current plugin API and planned extensions.

For quick project onboarding, see [README.md](README.md). For docs navigation, see [docs/README.md](docs/README.md). For a working example, see [examples/plugins/s3-upload.md](examples/plugins/s3-upload.md).

## Current Status

| Feature | Status |
|---------|--------|
| Plugin type definitions | ✅ Implemented |
| Internal executor | ✅ Implemented |
| CLI auto-loading from config | ✅ Shipped |
| Webhook plugins | 🔜 Planned |
| Go `.so` dynamic plugins | ❌ Removed from roadmap |

Command plugins (`[[plugins.command]]`) in config.toml are loaded and executed automatically after each run. The `when` field selects the dispatch phase: `after-push` (default), `on-success`, `on-failure`, and `always`. Post-run plugins are skipped on dry-run, user-aborted runs, and no-op runs (the run finished with no backup actions).

---

## API Reference

The rest of this document describes the current plugin config shape and planned extensions.

### Philosophy

While git-fire is built around git, emergencies don't respect tool boundaries. Sometimes you need to:
- Upload to S3 as a redundant backup
- Trigger a remote backup service
- Call a company-specific backup script
- Notify your team
- Create offline copies

The plugin system makes git-fire a **general-purpose emergency data evacuation tool**.

---

## Plugin Types (Planned API)

### 1. Command Plugins (Simplest)

Execute external commands or scripts:

```toml
[[plugins.command]]
name = "s3-backup"
command = "aws"
args = ["s3", "sync", "{repo_path}", "s3://emergency-backups/{repo_name}"]
when = "after-push"  # alias for on-success; also supports on-failure and always

[[plugins.command]]
name = "notify-team"
command = "/usr/local/bin/slack-notify.sh"
args = ["Emergency backup completed for {repo_name}"]
when = "after-push"

[[plugins.command]]
name = "create-tarball"
command = "tar"
args = ["czf", "/backups/{repo_name}-{timestamp}.tar.gz", "-C", "{repo_path}", "."]
when = "always"
```

**Variables available (seeded from configured scan root):**
- `{repo_path}` - Absolute path of the scan root
- `{repo_name}` - Basename of the scan root
- `{timestamp}` - Current timestamp (20060102-150405 format)
- `{branch}` - Branch of scan root (if it is a git repo)
- `{commit_sha}` - HEAD commit of scan root (if it is a git repo)

---

### 2. Go Plugins (Historical context) — ❌ Removed from roadmap

> **ARCHIVAL ONLY:** Dynamic Go `.so` plugin loading is no longer planned.
> Do not follow the build/setup steps in this section for production integration.
> The examples below are retained only as historical design context.

Write plugins in Go that integrate deeply:

```go
// plugins/s3_backup/main.go
package main

import (
    "github.com/git-fire/git-fire/internal/plugins"
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

**Historical build/install example (do not use in current versions):**
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

### 3. HTTP/Webhook Plugins — 🗓 Planned

> **Not yet implemented.** This describes the intended design for v0.2.

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

### 4. Remote Backup Services — 🗓 Planned

> **Not yet implemented.** Use command plugins today to achieve the same result (see examples below).

Integrate with existing backup solutions via dedicated plugin types:

```toml
# Future syntax (not yet supported — use [[plugins.command]] today)
[[plugins.restic]]
repository = "/backups/restic"
password_file = "~/.restic-password"
tags = ["git-fire", "emergency"]

[[plugins.rclone]]
remote = "backup-s3"
path = "git-fire-backups"
flags = ["--fast-list", "--transfers=32"]
```

---

## Implementation Plan (RFC)

### Phase 1: Command Plugins — ✅ Complete

**Implemented in:**
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
    When    Trigger  // on-success, on-failure, always (after-push aliases on-success)
    Timeout time.Duration
}
```

### Phase 2: Go Plugins — ❌ Removed from roadmap

This section is retained as historical design context and is not on the current roadmap:

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

### Phase 3: Webhook/HTTP Plugins — 🔜 v0.2 target

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

# Slack via curl — planned command-plugin style configuration
[[plugins.command]]
name = "slack-notify"
command = "curl"
args = ["-s", "-X", "POST", "${SLACK_WEBHOOK}", "-H", "Content-Type: application/json",
        "-d", "{\"text\": \"✅ {repo_name} backed up to S3\"}"]
when = "after-push"
```

### Example 2: Company Backup Service

```toml
# Use a command plugin with curl until webhook plugins are implemented
[[plugins.command]]
name = "company-backup"
command = "curl"
args = ["-s", "-X", "POST", "https://backup.company.com/api/git-fire",
        "-H", "X-API-Key: ${COMPANY_BACKUP_KEY}",
        "-H", "Content-Type: application/json",
        "-d", "{\"repo\": \"{repo_name}\", \"path\": \"{repo_path}\", \"emergency\": true}"]
when = "after-push"
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

Plugins run automatically after each backup run when configured in ~/.config/git-fire/config.toml. Dry-run skips the post-run plugin flow entirely: post-run plugins are not executed and nothing is printed for them, so you cannot validate plugin wiring with `--dry-run` alone.

```bash
# Preview backup plan (post-run plugins are not run on dry-run)
git-fire --dry-run

# Generate a config file to add plugins to
git-fire --init
```

> **Note on template variables:** Post-run plugins fire once per session, seeded from the configured scan root: {repo_path} is the absolute scan root path, {repo_name} is its basename, and {branch}/{commit_sha} are populated when the scan root is itself a git repo. Use {timestamp} for per-run unique paths.

> **Planned CLI flags** (not yet implemented): --list-plugins, --plugin <name>, --no-plugins, --test-plugin, --show-plugins.

---

## Security Considerations

1. **Credential Management**
   - Use environment variables, not config files
   - Support credential helpers (e.g., `pass`, `1password`)
   - Warn about plaintext secrets in config

2. **Plugin Validation**
   - Sandbox command plugins (limit file access)
   - Require explicit plugin enable in config

3. **Dry-Run Support**
   - Post-run command plugins are not invoked on `--dry-run` (the CLI never enters the post-run plugin path).
   - Command plugins that do run with `DryRun: true` in context should log intent only and not execute side effects.

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

3. Preview the backup plan (plugins are still skipped on dry-run):
   ```bash
   git-fire --dry-run
   ```

4. Run for real (post-run plugins execute after the backup flow):
   ```bash
   git-fire
   ```
