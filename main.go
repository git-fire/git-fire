// Package main is the entrypoint for the git-fire CLI binary.
package main

import (
	"fmt"
	"os"

	"github.com/git-fire/git-fire/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
