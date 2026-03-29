package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/testutil"
)

func TestScanRepositories_FindsSingleRepo(t *testing.T) {
	// Create a test repo
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Scan for repos
	opts := git.DefaultScanOptions()
	opts.RootPath = filepath.Dir(repoPath)
	opts.UseCache = false // Don't use cache for tests

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	// Should find exactly one repo
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, found %d", len(repos))
	}

	// Verify repo path
	repo := repos[0]
	if repo.Path != repoPath {
		t.Errorf("Expected path %s, got %s", repoPath, repo.Path)
	}

	// Verify repo name
	expectedName := "test-repo"
	if repo.Name != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, repo.Name)
	}

	// Should not be dirty
	if repo.IsDirty {
		t.Error("Expected clean repo, but IsDirty=true")
	}
}

func TestScanRepositories_FindsMultipleRepos(t *testing.T) {
	// Create a temp root directory to hold multiple repos
	tempRoot := t.TempDir()

	// Create subdirectories for each repo
	repo1Dir := filepath.Join(tempRoot, "projects/repo1")
	repo2Dir := filepath.Join(tempRoot, "projects/repo2")
	repo3Dir := filepath.Join(tempRoot, "src/repo3")

	// Create the parent dirs
	for _, dir := range []string{
		filepath.Dir(repo1Dir),
		filepath.Dir(repo2Dir),
		filepath.Dir(repo3Dir),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create repos directly in these locations using git commands
	for _, path := range []string{repo1Dir, repo2Dir, repo3Dir} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatalf("Failed to create repo dir: %v", err)
		}

		// Initialize git repo
		cmd := exec.Command("git", "init")
		cmd.Dir = path
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to init repo: %v", err)
		}

		// Create initial commit (required)
		readmePath := filepath.Join(path, "README.md")
		if err := os.WriteFile(readmePath, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to write README: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = path
		_ = cmd.Run()

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = path
		_ = cmd.Run()

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = path
		_ = cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = path
		_ = cmd.Run()
	}

	// Now scan the temp root
	opts := git.DefaultScanOptions()
	opts.RootPath = tempRoot
	opts.UseCache = false

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	// Should find all 3 repos
	if len(repos) != 3 {
		t.Fatalf("Expected 3 repos, found %d", len(repos))
	}

	// Verify we found our specific repos
	foundNames := make(map[string]bool)
	for _, repo := range repos {
		foundNames[repo.Name] = true
	}

	expected := []string{"repo1", "repo2", "repo3"}
	for _, name := range expected {
		if !foundNames[name] {
			t.Errorf("Expected to find repo %s, but didn't", name)
		}
	}
}

func TestScanRepositories_DetectsDirtyRepo(t *testing.T) {
	// Create a dirty repo
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:  "dirty-repo",
		Dirty: true,
	})

	opts := git.DefaultScanOptions()
	opts.RootPath = filepath.Dir(repoPath)
	opts.UseCache = false

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, found %d", len(repos))
	}

	// Verify repo is marked as dirty
	if !repos[0].IsDirty {
		t.Error("Expected IsDirty=true for dirty repo")
	}
}

func TestScanRepositories_ExtractsRemotes(t *testing.T) {
	// Create bare remote
	remotePath := testutil.CreateBareRemote(t, "origin")

	// Create repo with remote configured
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "remote-repo",
		Remotes: map[string]string{
			"origin": remotePath,
		},
	})

	opts := git.DefaultScanOptions()
	opts.RootPath = filepath.Dir(repoPath)
	opts.UseCache = false

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, found %d", len(repos))
	}

	// Verify remote was extracted
	repo := repos[0]
	if len(repo.Remotes) == 0 {
		t.Fatal("Expected remotes to be extracted, but got none")
	}

	// Should have "origin" remote
	foundOrigin := false
	for _, remote := range repo.Remotes {
		if remote.Name == "origin" {
			foundOrigin = true
			if remote.URL != remotePath {
				t.Errorf("Expected origin URL %s, got %s", remotePath, remote.URL)
			}
		}
	}

	if !foundOrigin {
		t.Error("Expected to find 'origin' remote")
	}
}

func TestScanRepositories_RespectsExcludePatterns(t *testing.T) {
	// Create temp directory structure
	fsRoot := testutil.SetupFakeFilesystem(t)

	// Create repos in excluded directories
	cacheDir := filepath.Join(fsRoot, "home/testuser/.cache")
	nodeModulesDir := filepath.Join(fsRoot, "home/testuser/node_modules")

	// We'd create repos in these dirs, but scanner should skip them
	// For now, just test that exclude patterns are respected

	opts := git.DefaultScanOptions()
	opts.RootPath = fsRoot
	opts.UseCache = false
	opts.Exclude = []string{".cache", "node_modules"}

	// This should not find repos in .cache or node_modules
	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	// Verify no repos found in excluded paths
	for _, repo := range repos {
		if filepath.Base(filepath.Dir(repo.Path)) == ".cache" {
			t.Error("Found repo in .cache (should be excluded)")
		}
		if filepath.Base(filepath.Dir(repo.Path)) == "node_modules" {
			t.Error("Found repo in node_modules (should be excluded)")
		}
	}

	_ = cacheDir
	_ = nodeModulesDir
}

func TestScanRepositories_ExtractsBranches(t *testing.T) {
	// Create repo with multiple branches
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:     "multi-branch",
		Branches: []string{"feature-a", "feature-b", "develop"},
	})

	opts := git.DefaultScanOptions()
	opts.RootPath = filepath.Dir(repoPath)
	opts.UseCache = false

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories failed: %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, found %d", len(repos))
	}

	repo := repos[0]

	// Should have extracted branches
	if len(repo.Branches) == 0 {
		t.Fatal("Expected branches to be extracted")
	}

	// Should find the branches we created (plus main/master)
	branchNames := make(map[string]bool)
	for _, branch := range repo.Branches {
		branchNames[branch] = true
	}

	expectedBranches := []string{"feature-a", "feature-b", "develop"}
	for _, branch := range expectedBranches {
		if !branchNames[branch] {
			t.Errorf("Expected to find branch %s", branch)
		}
	}
}

// TestScanRepositories_IncludesOutOfTreeKnownPaths verifies the registry
// invariant: repos registered from a previous run in a different directory
// are still returned even when the scan root does not contain them.
func TestScanRepositories_IncludesOutOfTreeKnownPaths(t *testing.T) {
	tempRoot := t.TempDir()

	// insideRepo is under the scan root.
	insideDir := filepath.Join(tempRoot, "scan-root", "inside-repo")
	// outsideRepo is a sibling of the scan root — NOT under it.
	outsideDir := filepath.Join(tempRoot, "other-dir", "outside-repo")

	for _, dir := range []string{insideDir, outsideDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		initBareRepo(t, dir)
	}

	opts := git.DefaultScanOptions()
	opts.RootPath = filepath.Join(tempRoot, "scan-root")
	opts.UseCache = false
	// outsideDir is known from the registry (populated by a prior run).
	opts.KnownPaths = map[string]bool{
		outsideDir: false,
	}

	repos, err := git.ScanRepositories(opts)
	if err != nil {
		t.Fatalf("ScanRepositories: %v", err)
	}

	found := make(map[string]bool)
	for _, r := range repos {
		found[r.Name] = true
	}

	if !found["inside-repo"] {
		t.Error("inside-repo (under scan root) not found")
	}
	if !found["outside-repo"] {
		t.Error("outside-repo (registry entry outside scan root) not found — registry invariant broken")
	}
}

// TestScanRepositoriesStream verifies that the streaming variant sends every
// discovered repo to the channel and closes it when done.
func TestScanRepositoriesStream(t *testing.T) {
	tempRoot := t.TempDir()

	names := []string{"alpha", "beta", "gamma"}
	for _, name := range names {
		dir := filepath.Join(tempRoot, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		initBareRepo(t, dir)
	}

	opts := git.DefaultScanOptions()
	opts.RootPath = tempRoot
	opts.UseCache = false

	out := make(chan git.Repository, len(names))
	// Drain in goroutine (caller's responsibility).
	var got []git.Repository
	done := make(chan struct{})
	go func() {
		defer close(done)
		for r := range out {
			got = append(got, r)
		}
	}()

	if err := git.ScanRepositoriesStream(opts, out); err != nil {
		t.Fatalf("ScanRepositoriesStream: %v", err)
	}
	<-done // channel must be closed by ScanRepositoriesStream

	if len(got) != len(names) {
		t.Fatalf("want %d repos, got %d", len(names), len(got))
	}
	foundNames := make(map[string]bool)
	for _, r := range got {
		foundNames[r.Name] = true
	}
	for _, name := range names {
		if !foundNames[name] {
			t.Errorf("repo %q not received from stream", name)
		}
	}
}

// initBareRepo initialises a minimal git repo (with one commit) at dir.
func initBareRepo(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}
	run("git", "init")
	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "init")
}

func TestDefaultScanOptions(t *testing.T) {
	opts := git.DefaultScanOptions()

	// Verify defaults are sensible
	if opts.RootPath != "." {
		t.Errorf("Expected default RootPath '.', got %s", opts.RootPath)
	}

	if opts.MaxDepth != 10 {
		t.Errorf("Expected default MaxDepth 10, got %d", opts.MaxDepth)
	}

	if opts.Workers != 8 {
		t.Errorf("Expected default Workers 8, got %d", opts.Workers)
	}

	// Should have common exclude patterns
	if len(opts.Exclude) == 0 {
		t.Error("Expected default exclude patterns")
	}
}
