# Persistent Repository Registry

`git-fire` maintains a registry at `~/.config/git-fire/repos.toml` (next to `config.toml`) that accumulates every git repo discovered across runs. Known repos are loaded instantly at startup; the filesystem walker only descends into directories not already in the registry.

Related docs:
- quickstart and CLI usage: [../README.md](../README.md)
- full behavior spec: [../GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md)
- docs index: [README.md](README.md)

## Architecture Diagrams

### 1. Startup Registry Flow (`runGitFire`)

```mermaid
sequenceDiagram
    participant Main as cmd/root.go
    participant Reg as registry.Store
    participant FS as Filesystem
    participant Scanner as git.ScanRepositories

    Main->>Reg: DefaultRegistryPath()
    Reg-->>Main: ~/.config/git-fire/repos.toml

    Main->>Reg: Load(path)
    Reg->>FS: MkdirAll + ReadFile
    FS-->>Reg: raw TOML (or ENOENT)
    Reg-->>Main: *Registry (empty on first run)

    loop each RegistryEntry (not ignored)
        Main->>FS: os.Stat(entry.Path)
        alt path exists
            Main->>Reg: SetStatus(path, StatusActive)
        else missing
            Main->>Reg: SetStatus(path, StatusMissing)
        end
    end

    Main->>Scanner: ScanRepositories(opts{KnownPaths: activeMap})
    Scanner-->>Main: []Repository (new + known)

    loop each discovered repo
        alt already in registry
            Main->>Reg: Upsert (update LastSeen, restore Mode)
        else new
            Main->>Reg: Upsert (StatusActive, AddedAt=now)
        end
    end

    Main->>Reg: Save(reg, path)   [best-effort]
    Main->>Main: filter out StatusIgnored repos
```

### 2. TUI Write-Through (mode `m` / ignore `x`)

```mermaid
sequenceDiagram
    participant User
    participant TUI as RepoSelectorModel (Bubble Tea)
    participant Reg as registry.Registry (in-memory)
    participant Disk as repos.toml

    User->>TUI: press "m"
    TUI->>TUI: cycle repo.Mode
    TUI->>Reg: FindByPath(absPath) → Upsert(entry{Mode:newMode})
    TUI->>Disk: Save(reg, regPath)  [best-effort, error→lastErr]

    User->>TUI: press "x"
    TUI->>Reg: SetStatus(absPath, StatusIgnored)
    alt entry not found
        TUI->>Reg: Upsert(entry{Status:ignored})
    end
    TUI->>Disk: Save(reg, regPath)  [best-effort]
    TUI->>TUI: remove repo from visible list, rebuild selected map
```

### 3. `git-fire repos` CLI Subcommands

```mermaid
sequenceDiagram
    participant CLI as cmd/repos.go
    participant Reg as registry.Store
    participant Scanner as git.ScanRepositories

    %% list
    CLI->>Reg: loadRegistry() → Load(path)
    Reg-->>CLI: *Registry
    CLI->>CLI: print tabular summary

    %% scan
    CLI->>Reg: loadRegistry()
    CLI->>CLI: buildKnownPaths(reg, globalRescan)
    CLI->>Scanner: ScanRepositories(opts{KnownPaths})
    Scanner-->>CLI: []Repository
    loop new repos only
        CLI->>Reg: Upsert(StatusActive, AddedAt=now)
    end
    CLI->>Reg: Save(reg, path)

    %% remove / ignore / unignore
    CLI->>Reg: loadRegistry()
    CLI->>Reg: Remove(absPath) OR SetStatus(absPath, status)
    CLI->>Reg: Save(reg, path)
```

### 4. Registry Store — Data Operations

```mermaid
flowchart TD
    A[Load] -->|file missing| B[empty Registry]
    A -->|file exists| C["TOML unmarshal → *Registry"]

    D[Upsert] -->|path exists| E["update fields<br/>preserve AddedAt<br/>preserve RescanSubmodules if nil<br/>preserve usb_* overrides when unset"]
    D -->|path missing| F["append new entry<br/>set AddedAt=now if zero"]

    G[SetStatus] -->|found| H["update Status<br/>set LastSeen=now if active"]
    G -->|not found| I[return false]

    J[Remove] -->|found| K[splice from Repos slice]
    J -->|not found| L[return false]

    M[Save] --> N[MkdirAll 0700]
    N --> O[TOML marshal]
    O --> P[write PID-temp file 0600]
    P --> Q[os.Rename onto target - atomic]
    Q -->|rename fails| R[remove temp, return error]
```

## USB-related Registry Fields

Each `repos.toml` entry may include optional USB overrides:

- `usb_strategy`: per-repo strategy override (`git-mirror` or `git-clone`)
- `usb_repo_path`: destination path override relative to target repos root
- `usb_sync_policy`: per-repo sync policy (`keep` or `prune`)

When an existing repo entry is upserted, these overrides are preserved if the incoming update does not specify them.
