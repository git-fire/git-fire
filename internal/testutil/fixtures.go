package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// CreateTestRepo creates a real git repository in a temporary directory
// with optional configuration for testing different scenarios
type RepoOptions struct {
	// Name of the repo (used for directory name)
	Name string

	// Add uncommitted files (makes repo "dirty")
	Dirty bool

	// Files to create and commit
	Files map[string]string

	// Remotes to configure (name -> URL)
	Remotes map[string]string

	// Branches to create
	Branches []string

	// Initial commit message
	InitialCommit string
}

// CreateTestRepo creates a test git repository
func CreateTestRepo(t *testing.T, opts RepoOptions) string {
	t.Helper()

	// Create temp directory
	tmpDir := t.TempDir()
	repoPath := filepath.Join(tmpDir, opts.Name)

	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("Failed to create repo directory: %v", err)
	}

	// Initialize git repo
	runGit(t, repoPath, "init")
	runGit(t, repoPath, "config", "user.email", "test@example.com")
	runGit(t, repoPath, "config", "user.name", "Test User")

	// Create initial commit (required for most operations)
	initialFile := filepath.Join(repoPath, "README.md")
	commitMsg := opts.InitialCommit
	if commitMsg == "" {
		commitMsg = "Initial commit"
	}

	if err := os.WriteFile(initialFile, []byte("# Test Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	runGit(t, repoPath, "add", "README.md")
	runGit(t, repoPath, "commit", "-m", commitMsg)

	// Create additional files if specified
	for filename, content := range opts.Files {
		filePath := filepath.Join(repoPath, filename)

		// Create parent directories if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", filename, err)
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", filename, err)
		}
		runGit(t, repoPath, "add", filename)
		runGit(t, repoPath, "commit", "-m", "Add "+filename)
	}

	// Add remotes
	for name, url := range opts.Remotes {
		runGit(t, repoPath, "remote", "add", name, url)
	}

	// Create branches
	for _, branch := range opts.Branches {
		runGit(t, repoPath, "checkout", "-b", branch)
	}

	// Return to main/master branch
	if len(opts.Branches) > 0 {
		// Try main first, fallback to master
		if err := exec.Command("git", "-C", repoPath, "checkout", "main").Run(); err != nil {
			runGit(t, repoPath, "checkout", "master")
		}
	}

	// Make repo dirty if requested
	if opts.Dirty {
		dirtyFile := filepath.Join(repoPath, "uncommitted.txt")
		if err := os.WriteFile(dirtyFile, []byte("uncommitted changes\n"), 0644); err != nil {
			t.Fatalf("Failed to create dirty file: %v", err)
		}
	}

	return repoPath
}

// CreateBareRemote creates a bare git repository to use as a remote
func CreateBareRemote(t *testing.T, name string) string {
	t.Helper()

	tmpDir := t.TempDir()
	remotePath := filepath.Join(tmpDir, name+".git")

	if err := os.MkdirAll(remotePath, 0755); err != nil {
		t.Fatalf("Failed to create bare repo directory: %v", err)
	}

	runGit(t, remotePath, "init", "--bare")

	return remotePath
}

// SetupFakeFilesystem creates a fake filesystem structure for scanning tests
func SetupFakeFilesystem(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"home/testuser/projects",
		"home/testuser/src",
		"home/testuser/.cache",
		"home/testuser/node_modules",
		"root/sys",
		"root/proc",
	}

	for _, dir := range dirs {
		path := filepath.Join(tmpDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	return tmpDir
}

// runGit is a helper to run git commands in a specific directory
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Git command failed: git %v\nOutput: %s\nError: %v", args, output, err)
	}
}

// IsDirty checks if a git repo has uncommitted changes
func IsDirty(t *testing.T, repoPath string) bool {
	t.Helper()

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to check git status: %v", err)
	}

	return len(output) > 0
}

// GetRemotes returns the configured remotes for a repo
func GetRemotes(t *testing.T, repoPath string) map[string]string {
	t.Helper()

	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get remotes: %v", err)
	}

	// Parse output into map
	// Format: "origin	/path/to/remote (fetch)"
	//         "origin	/path/to/remote (push)"
	remotes := make(map[string]string)

	lines := string(output)
	if lines == "" {
		return remotes
	}

	// Simple parsing - just extract remote names
	// Full parsing not needed for tests
	for _, line := range splitLines(lines) {
		if line == "" {
			continue
		}
		// Just check if "origin" appears in the line
		// Good enough for test validation
		if len(line) > 0 {
			// Extract first word (remote name)
			parts := splitWhitespace(line)
			if len(parts) >= 2 {
				name := parts[0]
				url := parts[1]
				remotes[name] = url
			}
		}
	}

	return remotes
}

// Helper: split by newlines
func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// Helper: split by whitespace/tabs
func splitWhitespace(s string) []string {
	var parts []string
	current := ""
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// RunGitCmd runs a git command and fails the test if it errors
// Exported version of runGit for use in other test packages
func RunGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	runGit(t, dir, args...)
}

// GetCurrentSHA returns the current commit SHA
func GetCurrentSHA(t *testing.T, repoPath string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current SHA: %v", err)
	}

	sha := string(output)
	// Trim newline
	if len(sha) > 0 && sha[len(sha)-1] == '\n' {
		sha = sha[:len(sha)-1]
	}

	return sha
}

// GetBranches returns all branches in the repo
func GetBranches(t *testing.T, repoPath string) []string {
	t.Helper()

	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get branches: %v", err)
	}

	branches := splitLines(string(output))

	// Filter out empty lines
	var result []string
	for _, b := range branches {
		if b != "" {
			result = append(result, b)
		}
	}

	return result
}
