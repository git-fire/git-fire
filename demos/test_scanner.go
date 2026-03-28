//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/TBRX103/git-fire/internal/git"
)

func main() {
	fmt.Println("🔥 Git Fire - Repository Scanner Test (Non-Interactive)\n")

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
		fmt.Println("  go run ./demos/test_scanner.go ~/projects")
		os.Exit(0)
	}

	// Display all found repos
	fmt.Println("📋 Discovered Repositories:\n")
	for i, repo := range repos {
		fmt.Printf("%d. %s\n", i+1, repo.Name)
		fmt.Printf("   Path: %s\n", repo.Path)
		fmt.Printf("   Mode: %s\n", repo.Mode.String())
		fmt.Printf("   Selected: %v\n", repo.Selected)

		if repo.IsDirty {
			fmt.Printf("   Status: 💥 HAS UNCOMMITTED CHANGES\n")
		} else {
			fmt.Printf("   Status: ✓ Clean\n")
		}

		if len(repo.Remotes) > 0 {
			fmt.Printf("   Remotes:\n")
			for _, r := range repo.Remotes {
				fmt.Printf("     - %s: %s\n", r.Name, r.URL)
			}
		} else {
			fmt.Printf("   Remotes: (none)\n")
		}

		if len(repo.Branches) > 0 {
			fmt.Printf("   Branches: %v\n", repo.Branches)
		}

		if !repo.LastModified.IsZero() {
			fmt.Printf("   Last commit: %s\n", repo.LastModified.Format("2006-01-02 15:04:05"))
		}

		fmt.Println()
	}

	fmt.Printf("Total: %d repositories found\n", len(repos))
}
