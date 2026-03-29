package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Scenario represents a complex test scenario with multiple repos and states
type Scenario struct {
	t       *testing.T
	baseDir string
	repos   map[string]*ScenarioRepo
}

// ScenarioRepo represents a repository in a scenario
type ScenarioRepo struct {
	path      string
	name      string
	remotes   map[string]string
	t         *testing.T
}

// NewScenario creates a new test scenario
func NewScenario(t *testing.T) *Scenario {
	t.Helper()

	return &Scenario{
		t:       t,
		baseDir: t.TempDir(),
		repos:   make(map[string]*ScenarioRepo),
	}
}

// CreateRepo creates a basic repository in the scenario
func (s *Scenario) CreateRepo(name string) *ScenarioRepo {
	s.t.Helper()

	repoPath := CreateTestRepo(s.t, RepoOptions{
		Name: name,
	})

	repo := &ScenarioRepo{
		path:    repoPath,
		name:    name,
		remotes: make(map[string]string),
		t:       s.t,
	}

	s.repos[name] = repo
	return repo
}

// CreateBareRepo creates a bare repository (typically used as remote)
func (s *Scenario) CreateBareRepo(name string) *ScenarioRepo {
	s.t.Helper()

	remotePath := CreateBareRemote(s.t, name)

	repo := &ScenarioRepo{
		path:    remotePath,
		name:    name,
		remotes: make(map[string]string),
		t:       s.t,
	}

	s.repos[name] = repo
	return repo
}

// GetRepo retrieves a repo by name from the scenario
func (s *Scenario) GetRepo(name string) *ScenarioRepo {
	return s.repos[name]
}

// WithRemote adds a remote to the repository
func (r *ScenarioRepo) WithRemote(remoteName string, remote *ScenarioRepo) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "remote", "add", remoteName, remote.path)
	r.remotes[remoteName] = remote.path

	return r
}

// WithBranch creates and checks out a new branch
func (r *ScenarioRepo) WithBranch(branchName string) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "checkout", "-b", branchName)

	return r
}

// AddFile creates and stages a file
func (r *ScenarioRepo) AddFile(filename, content string) *ScenarioRepo {
	r.t.Helper()

	filePath := filepath.Join(r.path, filename)

	// Create parent directories if needed
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		r.t.Fatalf("Failed to create directory for %s: %v", filename, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		r.t.Fatalf("Failed to write file %s: %v", filename, err)
	}

	RunGitCmd(r.t, r.path, "add", filename)

	return r
}

// ModifyFile modifies an existing file (or creates if doesn't exist)
func (r *ScenarioRepo) ModifyFile(filename, content string) *ScenarioRepo {
	r.t.Helper()

	filePath := filepath.Join(r.path, filename)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		r.t.Fatalf("Failed to write file %s: %v", filename, err)
	}

	return r
}

// StageFile stages a file for commit
func (r *ScenarioRepo) StageFile(filename string) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "add", filename)

	return r
}

// Commit creates a commit with the given message
func (r *ScenarioRepo) Commit(message string) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "commit", "-m", message)

	return r
}

// Push pushes to the specified remote and branch
func (r *ScenarioRepo) Push(remoteName, branchName string) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "push", remoteName, branchName)

	return r
}

// Checkout switches to an existing branch
func (r *ScenarioRepo) Checkout(branchName string) *ScenarioRepo {
	r.t.Helper()

	RunGitCmd(r.t, r.path, "checkout", branchName)

	return r
}

// Path returns the filesystem path to the repository
func (r *ScenarioRepo) Path() string {
	return r.path
}

// AddWorktree adds a worktree to the repository
func (r *ScenarioRepo) AddWorktree(branch, path string) *ScenarioRepo {
	r.t.Helper()

	worktreePath := filepath.Join(filepath.Dir(r.path), path)

	RunGitCmd(r.t, r.path, "worktree", "add", worktreePath, branch)

	// Create a new ScenarioRepo for the worktree
	worktreeRepo := &ScenarioRepo{
		path:    worktreePath,
		name:    path,
		remotes: r.remotes, // Share remotes with main repo
		t:       r.t,
	}

	return worktreeRepo
}

// GetDefaultBranch returns the default branch name for a repo (main or master)
func (r *ScenarioRepo) GetDefaultBranch() string {
	r.t.Helper()

	branches := GetBranches(r.t, r.path)
	for _, b := range branches {
		if b == "main" || b == "master" {
			return b
		}
	}
	// Fallback to main if neither exists
	return "main"
}

// CreateConflictScenario creates a scenario where local and remote have diverged
func CreateConflictScenario(t *testing.T) (*Scenario, *ScenarioRepo, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	// Create bare remote
	remote := scenario.CreateBareRepo("remote")

	// Create local repo with initial commit
	local := scenario.CreateRepo("local").
		WithRemote("origin", remote).
		AddFile("file.txt", "version A\n").
		Commit("Commit A")

	// Get the default branch name
	defaultBranch := local.GetDefaultBranch()

	local.Push("origin", defaultBranch)

	// Create a second clone to simulate remote changes
	tempClone := scenario.CreateRepo("temp-clone").
		WithRemote("origin", remote)

	RunGitCmd(t, tempClone.path, "fetch", "origin", defaultBranch)
	RunGitCmd(t, tempClone.path, "reset", "--hard", "FETCH_HEAD")

	// Remote adds commit B
	tempClone.
		ModifyFile("file.txt", "version B\n").
		StageFile("file.txt").
		Commit("Commit B").
		Push("origin", defaultBranch)

	// Local adds commit C (diverges from remote)
	local.
		ModifyFile("file.txt", "version C\n").
		StageFile("file.txt").
		Commit("Commit C")

	// Now local and remote have diverged
	return scenario, local, remote
}

// CreateWorktreeScenario creates a scenario with multiple worktrees
func CreateWorktreeScenario(t *testing.T) (*Scenario, *ScenarioRepo, *ScenarioRepo, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	// Create main repo with branches
	main := scenario.CreateRepo("main")

	// Get default branch
	defaultBranch := main.GetDefaultBranch()

	// Create feature branch
	main.WithBranch("feature").
		AddFile("feature.go", "// Feature code\n").
		Commit("Add feature")

	// Create bugfix branch
	main.Checkout(defaultBranch).
		WithBranch("bugfix").
		AddFile("bugfix.go", "// Bugfix code\n").
		Commit("Add bugfix")

	// Back to default branch
	main.Checkout(defaultBranch)

	// Create worktrees
	worktree1 := main.AddWorktree("feature", "worktree-feature")
	worktree2 := main.AddWorktree("bugfix", "worktree-bugfix")

	// Make each worktree dirty in different ways
	worktree1.AddFile("wt1-file.go", "worktree 1 changes\n") // Staged only

	worktree2.ModifyFile("wt2-file.go", "worktree 2 changes\n") // Unstaged only (not added)

	return scenario, main, worktree1, worktree2
}

// CreateMultiRemoteScenario creates a repo with multiple remotes
func CreateMultiRemoteScenario(t *testing.T) (*Scenario, *ScenarioRepo, *ScenarioRepo, *ScenarioRepo, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	// Create multiple bare remotes
	origin := scenario.CreateBareRepo("origin")
	backup := scenario.CreateBareRepo("backup")
	upstream := scenario.CreateBareRepo("upstream")

	// Create local repo connected to all remotes
	local := scenario.CreateRepo("local").
		WithRemote("origin", origin).
		WithRemote("backup", backup).
		WithRemote("upstream", upstream).
		AddFile("main.go", "package main\n").
		Commit("Initial commit")

	// Push to origin using default branch
	defaultBranch := local.GetDefaultBranch()
	local.Push("origin", defaultBranch)

	return scenario, local, origin, backup, upstream
}

// CreateDirtyRepoScenario creates a repo with both staged and unstaged changes
func CreateDirtyRepoScenario(t *testing.T, staged, unstaged bool) (*Scenario, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	repo := scenario.CreateRepo("dirty")

	if staged {
		repo.AddFile("staged.txt", "staged changes\n")
	}

	if unstaged {
		repo.ModifyFile("unstaged.txt", "unstaged changes\n")
		// Don't stage it
	}

	return scenario, repo
}

// CreateCleanRepoScenario creates a clean repo with no uncommitted changes
func CreateCleanRepoScenario(t *testing.T) (*Scenario, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	remote := scenario.CreateBareRepo("remote")

	repo := scenario.CreateRepo("clean").
		WithRemote("origin", remote).
		AddFile("main.go", "package main\n").
		Commit("Initial commit")

	// Push using default branch
	defaultBranch := repo.GetDefaultBranch()
	repo.Push("origin", defaultBranch)

	return scenario, repo
}

// CreateMultiBranchScenario creates a repo with multiple branches
func CreateMultiBranchScenario(t *testing.T, branchNames []string) (*Scenario, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	remote := scenario.CreateBareRepo("remote")

	repo := scenario.CreateRepo("multi-branch").
		WithRemote("origin", remote).
		AddFile("main.go", "package main\n").
		Commit("Initial commit")

	// Get default branch
	defaultBranch := repo.GetDefaultBranch()

	repo.Push("origin", defaultBranch)

	// Create additional branches
	for i, branchName := range branchNames {
		repo.WithBranch(branchName).
			AddFile(fmt.Sprintf("branch-%d.go", i), fmt.Sprintf("// Branch %s\n", branchName)).
			Commit(fmt.Sprintf("Add %s", branchName))
	}

	// Return to default branch
	repo.Checkout(defaultBranch)

	return scenario, repo
}

// CreateLargeRepoScenario creates a repo with many files and commits
func CreateLargeRepoScenario(t *testing.T, numFiles, numCommits int) (*Scenario, *ScenarioRepo) {
	t.Helper()

	scenario := NewScenario(t)

	repo := scenario.CreateRepo("large")

	// Create files and commits
	for i := 0; i < numCommits; i++ {
		for j := 0; j < numFiles; j++ {
			filename := fmt.Sprintf("file-%d-%d.txt", i, j)
			content := fmt.Sprintf("Content for commit %d, file %d\n", i, j)
			repo.AddFile(filename, content)
		}
		repo.Commit(fmt.Sprintf("Commit %d", i))
	}

	return scenario, repo
}

// CreateDetachedHeadScenario creates a repo in detached HEAD state
func CreateDetachedHeadScenario(t *testing.T) (*Scenario, *ScenarioRepo, string) {
	t.Helper()

	scenario := NewScenario(t)

	repo := scenario.CreateRepo("detached").
		AddFile("file1.txt", "content 1\n").
		Commit("Commit 1").
		AddFile("file2.txt", "content 2\n").
		Commit("Commit 2")

	// Get SHA of first commit
	sha := GetCurrentSHA(t, repo.path)

	// Detach HEAD by checking out the commit SHA
	RunGitCmd(t, repo.path, "checkout", sha)

	return scenario, repo, sha
}
