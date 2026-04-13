#!/usr/bin/env bash
# Local parity with CI: build, vet, race tests, plugin contract.
# Optional: golangci-lint and goreleaser if installed (same as workflows).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "==> go build ./..."
go build ./...

echo "==> go vet ./..."
go vet ./...

echo "==> go test -race -count=1 ./..."
go test -race -count=1 ./...

echo "==> plugin contract tests"
go test -race -count=1 ./cmd/... ./internal/plugins/... \
  -run 'TestRunGitFire_OnFailurePluginErrorKeepsRunError|TestRunGitFire_OnSuccessPluginFailRunFailsRun|TestRunGitFire_DryRun_SkipsPostRunPlugins|TestParseTrigger|TestFilterPluginsByTrigger_AfterPushAliasesOnSuccess|TestLoadFromConfig_WiresFailRun|TestLoadFromConfig_FailRunDefaultFalse'

if command -v golangci-lint >/dev/null 2>&1; then
  echo "==> golangci-lint run"
  golangci-lint run --timeout=5m
else
  echo "==> golangci-lint: skipped (not on PATH; CI lint job still runs it)"
fi

if command -v goreleaser >/dev/null 2>&1; then
  echo "==> goreleaser check (.goreleaser.yaml)"
  goreleaser check --config .goreleaser.yaml
  echo "==> goreleaser check (.goreleaser.stable.yaml)"
  goreleaser check --config .goreleaser.stable.yaml
else
  echo "==> goreleaser: skipped (not on PATH; release-validate workflow still runs it)"
fi

echo "==> validate: OK"
