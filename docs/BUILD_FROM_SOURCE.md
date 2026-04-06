# Build From Source

This guide covers building `git-fire` directly from source on Linux, macOS, and Windows.

## Prerequisites

- Go 1.24.2+
- Git

Verify:

```bash
go version
git --version
```

## Clone and Build

```bash
git clone https://github.com/git-fire/git-fire.git
cd git-fire
go build ./...
```

Build a local binary in the repo root:

```bash
make build
./git-fire --version
```

## Install to User Bin

On Linux/macOS:

```bash
make install
git-fire --version
```

On Windows PowerShell:

```powershell
go build -o git-fire.exe .
git-fire.exe --version
```

If needed, move `git-fire.exe` into a directory on your user `PATH` (for example, `$env:USERPROFILE\\bin`).

## Install Without Cloning

```bash
go install github.com/git-fire/git-fire@latest
```

Or pin an explicit release tag:

```bash
go install github.com/git-fire/git-fire@v0.2.0
```
