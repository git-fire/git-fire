BINARY := git-fire
# Directory containing this Makefile (module root) — build works the same from any shell cwd.
ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
REPO_BIN := $(ROOT)$(BINARY)
# Single global install location (overwrites on each install).
USER_BIN := $(abspath $(HOME)/.local/bin)
INSTALL_BIN := $(USER_BIN)/$(BINARY)
# Version: use git tag if available, otherwise "dev"
VERSION ?= $(shell git -C "$(ROOT)" describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X github.com/TBRX103/git-fire/cmd.Version=$(VERSION)
LDFLAGS_RELEASE := $(LDFLAGS) -s -w

.PHONY: all build run test test-race lint clean install help

all: build

## build: compile binary next to this Makefile (./git-fire in repo root)
build:
	cd "$(ROOT)" && go build -ldflags "$(LDFLAGS)" -o "$(REPO_BIN)" .

## run: build and run with optional ARGS (e.g. make run ARGS="--dry-run")
run: build
	"$(REPO_BIN)" $(ARGS)

## test: run all tests
test:
	cd "$(ROOT)" && go test -count=1 ./...

## test-race: run tests with race detector
test-race:
	cd "$(ROOT)" && go test -race -count=1 ./...

## lint: vet the code
lint:
	cd "$(ROOT)" && go vet ./...

## clean: remove the repo-local built binary
clean:
	rm -f "$(REPO_BIN)"

## install: build and copy to ~/.local/bin (overwrites). Invoke from anywhere: make -C /path/to/git-fire install
install:
	@mkdir -p "$(USER_BIN)"
	cd "$(ROOT)" && go build -ldflags "$(LDFLAGS_RELEASE)" -o "$(INSTALL_BIN)" .
	@chmod 755 "$(INSTALL_BIN)"
	@echo ""
	@echo "Installed: $(INSTALL_BIN)"
	@echo "This shell:  export PATH=\"$$HOME/.local/bin:$$PATH\" && hash -r"
	@echo "   (zsh: use rehash instead of hash -r if needed)"
	@echo "Permanent: add the export line to ~/.zshrc or ~/.bashrc"
	@echo ""

## help: show this help
help:
	@grep -E '^##' Makefile | sed 's/## /  /'
