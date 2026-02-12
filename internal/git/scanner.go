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

// ScanRepositories finds all git repositories under the given root path
func ScanRepositories(opts ScanOptions) ([]Repository, error) {
	// TODO: Implement caching if opts.UseCache

	repos := make([]Repository, 0)
	reposMutex := &sync.Mutex{}

	// Use a channel to limit concurrent workers
	semaphore := make(chan struct{}, opts.Workers)
	var wg sync.WaitGroup

	// Walk the directory tree
	err := filepath.Walk(opts.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			return nil
		}

		// Check if this is a .git directory
		if info.IsDir() && info.Name() == ".git" {
			// Found a git repo!
			repoPath := filepath.Dir(path)

			// Spawn goroutine to process this repo
			wg.Add(1)
			semaphore <- struct{}{} // Acquire

			go func(path string) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release

				repo, err := analyzeRepository(path)
				if err != nil {
					// Skip repos we can't analyze
					return
				}

				reposMutex.Lock()
				repos = append(repos, repo)
				reposMutex.Unlock()
			}(repoPath)

			// Don't descend into .git directory
			return filepath.SkipDir
		}

		// Check exclude patterns
		if info.IsDir() {
			for _, exclude := range opts.Exclude {
				if info.Name() == exclude {
					return filepath.SkipDir
				}
			}
		}

		// Check depth limit
		depth := strings.Count(strings.TrimPrefix(path, opts.RootPath), string(os.PathSeparator))
		if depth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
		}

		return nil
	})

	// Wait for all workers to finish
	wg.Wait()

	if err != nil {
		return nil, fmt.Errorf("error scanning repositories: %w", err)
	}

	return repos, nil
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
