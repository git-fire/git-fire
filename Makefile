BINARY := git-fire
MODULE := github.com/TBRX103/git-fire

.PHONY: build run test lint clean install help

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

## lint: vet the code
lint:
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
