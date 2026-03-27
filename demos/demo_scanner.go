//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/ui"
)

func main() {
	fmt.Println("🔥 Git Fire - Repository Scanner Demo\n")

	// Get scan path from args or use current directory
	scanPath := "."
	if len(os.Args) > 1 {
		scanPath = os.Args[1]
	}

	fmt.Printf("Scanning for repositories in: %s\n", scanPath)
	fmt.Println("This may take a moment...\n")

	// Scan for repositories
	opts := git.DefaultScanOptions()
	opts.RootPath = scanPath

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning repositories: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Found %d repositories\n\n", len(repos))

	if len(repos) == 0 {
		fmt.Println("No repositories found. Try specifying a different path:")
		fmt.Println("  go run ./demos/demo_scanner.go ~/projects")
		os.Exit(0)
	}

	// Show interactive selector
	selected, err := ui.RunRepoSelector(repos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Println("\n📋 Selected Repositories:\n")
	for _, repo := range selected {
		dirtyMarker := ""
		if repo.IsDirty {
			dirtyMarker = " (has uncommitted changes)"
		}

		remotesInfo := ""
		if len(repo.Remotes) > 0 {
			remoteNames := make([]string, len(repo.Remotes))
			for i, r := range repo.Remotes {
				remoteNames[i] = r.Name
			}
			remotesInfo = fmt.Sprintf(" - remotes: %v", remoteNames)
		}

		fmt.Printf("  • %s [%s]%s%s\n",
			repo.Name,
			repo.Mode.String(),
			dirtyMarker,
			remotesInfo,
		)
	}

	fmt.Printf("\nTotal: %d repositories selected\n", len(selected))
}
