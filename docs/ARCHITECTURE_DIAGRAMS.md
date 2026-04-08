# Architecture diagrams (Mermaid)

Visual reference for how `git-fire` routes work and how major subsystems interact. Source of truth for behavior remains the code and [GIT_FIRE_SPEC.md](../GIT_FIRE_SPEC.md); treat these diagrams as orientation aids.

GitHub renders Mermaid in Markdown. For local preview, use an editor with a Mermaid preview or [mermaid.live](https://mermaid.live).

## CLI routing (after config and registry load)

`cmd/root.go` picks one of three orchestration paths from flags:

```mermaid
flowchart TD
  A[Start: flags parsed, config loaded] --> B{--fire?}
  B -->|yes| C[runFireStream: streaming scan + TUI]
  B -->|no| D{--dry-run / --fire-drill?}
  D -->|yes| E[runBatch: full scan, plan, dry-run execute]
  D -->|no| F[runStream: pipeline scan to backup]
  C --> G[Post-run plugins when applicable]
  E --> G
  F --> G
```

## Internal package map

High-level layering (business logic lives under `internal/`; `cmd/` wires flags and I/O):

```mermaid
flowchart LR
  subgraph cmd
    Root[cmd/root.go]
  end
  subgraph internal
    CFG[config]
    REG[registry]
    GIT[git]
    EXE[executor]
    SAF[safety]
    PLG[plugins]
    AUT[auth]
    UI[ui]
  end
  Root --> CFG
  Root --> REG
  Root --> GIT
  Root --> EXE
  Root --> SAF
  Root --> PLG
  Root --> AUT
  Root --> UI
  EXE --> GIT
  EXE --> SAF
  GIT -->|exec.Command git| SYS[(system git)]
```

## Default live run: scan → registry → streamed backup

The default path pipelines discovery and execution so the first discovered repo can be planned and pushed before the filesystem walk finishes. `ExecuteStream` blocks until `repoChan` closes (scan plus upsert complete).

```mermaid
sequenceDiagram
  participant U as User
  participant R as cmd/root runStream
  participant S as git.ScanRepositoriesStream
  participant Reg as registry
  participant P as executor.Planner
  participant X as executor.Runner
  participant G as system git

  U->>R: git-fire
  R->>S: start scan goroutine scanChan
  R->>R: upsert goroutine: scanChan to repoChan
  loop each discovered repo
    S-->>R: Repository
    R->>Reg: upsert, filter ignored
    R-->>X: repo on repoChan
    X->>P: BuildPlan per repo
    X->>G: auto-commit / push actions
  end
  Note over R,X: repoChan closes when scan and upsert finish
  R->>R: optional wait if scan still running after backups
  R->>U: summary + log path
```

## Dry-run (`--dry-run`): batch scan, then plan

Dry-run collects the full repository list first, builds one plan, prints a summary, and runs the executor in dry-run mode (including secret checks where applicable). No remote mutations.

```mermaid
sequenceDiagram
  participant U as User
  participant R as cmd/root runBatch
  participant S as git.ScanRepositories
  participant A as auth.GetSSHStatus
  participant Reg as registry
  participant P as executor.Planner
  participant X as executor.Runner

  U->>R: git-fire --dry-run
  par scan and SSH
    R->>S: ScanRepositories
    R->>A: GetSSHStatus
  end
  S-->>R: all repos
  R->>Reg: upsert all, save
  R->>P: BuildPlan repos, dryRun true
  R->>U: plan.Summary
  R->>X: Execute plan dry-run
  X-->>U: fire drill complete
```

## TUI mode (`--fire`): progressive discovery

The scanner runs in the background; repositories stream through registry upsert into the Bubble Tea selector. After the user confirms selection, planning and execution follow the same planner/runner path as a non-streaming live run (not `ExecuteStream`).

```mermaid
sequenceDiagram
  participant U as User
  participant R as cmd/root runFireStream
  participant S as git.ScanRepositoriesStream
  participant Reg as registry
  participant T as ui.RunRepoSelectorStream
  participant P as executor.Planner
  participant X as executor.Runner

  R->>S: start scan with cancel ctx
  R->>Reg: goroutine: upsert stream to TUI channel
  R->>T: stream repos + folder progress
  U->>T: select repos, confirm
  T-->>R: selected repos
  R->>P: BuildPlan selected
  R->>X: Execute plan
  X-->>U: progress + result
```

## Registry state (conceptual)

Entries persist by absolute path. Only **ignored** repos are excluded from backup by default; missing paths can be marked when validated.

```mermaid
stateDiagram-v2
  [*] --> Active: discovered / path exists
  Active --> Ignored: user repos ignore
  Ignored --> Active: user repos unignore
  Active --> Missing: path vanished
  Missing --> Active: path restored
```

## Post-run plugins

After a successful or failed run (and not on user abort, dry-run, or no-op), enabled command plugins run by trigger: `after_push`, then success/failure-specific, then `always`. Failures are logged and do not fail the CLI run.

```mermaid
flowchart LR
  subgraph triggers
    T1[after_push]
    T2[on_success / on_failure]
    T3[always]
  end
  Run[Run finished] --> T1
  T1 --> T2
  T2 --> T3
  T3 --> Done[Exit]
```

See [PLUGINS.md](../PLUGINS.md) for configuration and extension points.
