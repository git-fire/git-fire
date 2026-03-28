BINARY := git-fire
MODULE := github.com/TBRX103/git-fire

.PHONY: all build run test test-race lint lint-fix vet clean install help

all: build

## build: compile the binary to ./git-fire
build:
	go build -o $(BINARY) .

## run: build and run with optional ARGS (e.g. make run ARGS="--dry-run")
run: build
	./$(BINARY) $(ARGS)

## test: run all tests
test:
	go test -count=1 ./...

## test-race: run tests with race detector
test-race:
	go test -race -count=1 ./...

## lint: run golangci-lint (install: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## lint-fix: run golangci-lint and auto-fix what it can
lint-fix:
	golangci-lint run --fix ./...

## vet: run go vet only (faster than full lint)
vet:
	go vet ./...

## clean: remove the built binary
clean:
	rm -f $(BINARY)

## install: install the binary to $GOPATH/bin (makes it available as 'git-fire' anywhere)
install:
	go install .

## help: show this help
help:
	@grep -E '^##' Makefile | sed 's/## /  /'
