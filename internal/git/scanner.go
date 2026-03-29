package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ScanRepositoriesStream finds all git repositories and sends each one to out
// as soon as it is analyzed. out is closed when scanning is complete (or on
// early error). The caller must drain out; a goroutine that drains out should
// be running before calling this function.
//
// Registry entries in opts.KnownPaths are included regardless of whether they
// fall under opts.RootPath — the registry is global.
func ScanRepositoriesStream(opts ScanOptions, out chan<- Repository) error {
	semaphore := make(chan struct{}, opts.Workers)
	var wg sync.WaitGroup

	absRoot, err := filepath.Abs(opts.RootPath)
	if err != nil {
		close(out)
		return fmt.Errorf("resolving scan path: %w", err)
	}

	spawnAnalysis := func(repoPath string) {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(p string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			repo, err := analyzeRepository(p)
			if err != nil {
				return
			}
			out <- repo
		}(repoPath)
	}

	// Pre-analyze all known paths immediately so they appear in results even
	// if the walk skips them. This also handles nested known repos correctly
	// (a parent skipped by the walker does not prevent inner repos from being
	// analyzed, since they were already queued here).
	// Known paths outside the scan root are included too — the registry is
	// global and should not be filtered by the current working directory.
	for knownPath := range opts.KnownPaths {
		spawnAnalysis(knownPath)
	}

	// Walk the directory tree to discover new (unknown) repos.
	walkErr := filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		absPath, absErr := filepath.Abs(path)
		if absErr != nil {
			return nil
		}

		// If this directory is a known repo root:
		//   rescan=false → skip the entire subtree (already queued above)
		//   rescan=true  → continue walking so new submodules inside are found
		if rescan, isKnown := opts.KnownPaths[absPath]; isKnown && !rescan {
			return filepath.SkipDir
		}

		// Check if this is a .git directory (signals a repo root).
		if info.Name() == ".git" {
			repoPath := filepath.Dir(absPath)
			// Only queue if not already handled via KnownPaths pre-analysis.
			if _, alreadyKnown := opts.KnownPaths[repoPath]; !alreadyKnown {
				spawnAnalysis(repoPath)
			}
			return filepath.SkipDir
		}

		// Check exclude patterns
		for _, exclude := range opts.Exclude {
			if info.Name() == exclude {
				return filepath.SkipDir
			}
		}

		// Check depth limit
		depth := strings.Count(strings.TrimPrefix(path, absRoot), string(os.PathSeparator))
		if depth > opts.MaxDepth {
			return filepath.SkipDir
		}

		return nil
	})

	// Wait for all workers, then signal end-of-stream.
	wg.Wait()
	close(out)

	if walkErr != nil {
		return fmt.Errorf("error scanning repositories: %w", walkErr)
	}
	return nil
}

// ScanRepositories finds all git repositories under the given root path and
// returns them as a slice. It is a convenience wrapper around
// ScanRepositoriesStream for callers that need the full list before proceeding
// (e.g. the TUI repo selector).
func ScanRepositories(opts ScanOptions) ([]Repository, error) {
	out := make(chan Repository, opts.Workers)
	var repos []Repository

	// Drain the channel in a goroutine so workers never block on sends.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for repo := range out {
			repos = append(repos, repo)
		}
	}()

	err := ScanRepositoriesStream(opts, out)
	<-done // wait for draining goroutine to finish
	return repos, err
}

// analyzeRepository extracts metadata from a git repository
func analyzeRepository(repoPath string) (Repository, error) {
	repo := Repository{
		Path:     repoPath,
		Name:     filepath.Base(repoPath),
		Selected: true,                  // Default: selected
		Mode:     ModePushKnownBranches, // Default: push known branches
	}

	// Extract remotes
	remotes, err := getRemotes(repoPath)
	if err == nil {
		repo.Remotes = remotes
	}

	// Extract branches
	branches, err := getBranches(repoPath)
	if err == nil {
		repo.Branches = branches
	}

	// Check if dirty
	dirty, err := isDirty(repoPath)
	if err == nil {
		repo.IsDirty = dirty
	}

	// Get last modified time
	lastModified, err := getLastCommitTime(repoPath)
	if err == nil {
		repo.LastModified = lastModified
	}

	return repo, nil
}

// getRemotes extracts configured remotes from a git repository
func getRemotes(repoPath string) ([]Remote, error) {
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	remotes := make([]Remote, 0)
	seen := make(map[string]bool)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "origin	git@github.com:user/repo.git (fetch)"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		url := parts[1]

		// Skip duplicates (fetch/push lines)
		if seen[name] {
			continue
		}
		seen[name] = true

		remotes = append(remotes, Remote{
			Name: name,
			URL:  url,
		})
	}

	return remotes, nil
}

// getBranches extracts local branch names from a git repository
func getBranches(repoPath string) ([]string, error) {
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	branches := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, nil
}

// isDirty checks if a repository has uncommitted changes
func isDirty(repoPath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// If output is not empty, repo is dirty
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// getLastCommitTime gets the timestamp of the last commit
func getLastCommitTime(repoPath string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}

	timestamp := strings.TrimSpace(string(output))
	if timestamp == "" {
		return time.Time{}, fmt.Errorf("no commits")
	}

	// Parse unix timestamp
	var unixTime int64
	_, err = fmt.Sscanf(timestamp, "%d", &unixTime)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(unixTime, 0), nil
}
