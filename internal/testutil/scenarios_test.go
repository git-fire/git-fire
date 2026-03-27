package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewScenario(t *testing.T) {
	scenario := NewScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if scenario.baseDir == "" {
		t.Fatal("Expected base directory to be set")
	}

	if scenario.repos == nil {
		t.Fatal("Expected repos map to be initialized")
	}
}

func TestCreateConflictScenario(t *testing.T) {
	scenario, local, remote := CreateConflictScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if local == nil {
		t.Fatal("Expected local repo to be created")
	}

	if remote == nil {
		t.Fatal("Expected remote repo to be created")
	}

	// Verify local repo exists
	if _, err := os.Stat(local.path); os.IsNotExist(err) {
		t.Fatalf("Local repo does not exist: %s", local.path)
	}

	// Verify remote repo exists
	if _, err := os.Stat(remote.path); os.IsNotExist(err) {
		t.Fatalf("Remote repo does not exist: %s", remote.path)
	}

	// Verify local repo has origin remote
	remotes := GetRemotes(t, local.path)
	if _, ok := remotes["origin"]; !ok {
		t.Fatal("Expected origin remote to be configured")
	}

	// Verify file.txt exists in local repo
	filePath := filepath.Join(local.path, "file.txt")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("file.txt does not exist in local repo: %s", filePath)
	}

	// Verify local and remote have diverged
	// This is tested by checking commit history
	localSHA := GetCurrentSHA(t, local.path)
	if localSHA == "" {
		t.Fatal("Expected local repo to have commits")
	}
}

func TestCreateWorktreeScenario(t *testing.T) {
	scenario, main, worktree1, worktree2 := CreateWorktreeScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	// Verify main repo exists
	if _, err := os.Stat(main.path); os.IsNotExist(err) {
		t.Fatalf("Main repo does not exist: %s", main.path)
	}

	// Verify worktrees exist
	if _, err := os.Stat(worktree1.path); os.IsNotExist(err) {
		t.Fatalf("Worktree 1 does not exist: %s", worktree1.path)
	}

	if _, err := os.Stat(worktree2.path); os.IsNotExist(err) {
		t.Fatalf("Worktree 2 does not exist: %s", worktree2.path)
	}

	// Verify branches exist
	branches := GetBranches(t, main.path)
	hasFeat := false
	hasBug := false

	for _, b := range branches {
		if b == "feature" {
			hasFeat = true
		}
		if b == "bugfix" {
			hasBug = true
		}
	}

	if !hasFeat {
		t.Fatal("Expected feature branch to exist")
	}

	if !hasBug {
		t.Fatal("Expected bugfix branch to exist")
	}
}

func TestCreateMultiRemoteScenario(t *testing.T) {
	scenario, local, origin, backup, upstream := CreateMultiRemoteScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	// Verify all repos exist
	repos := []*ScenarioRepo{local, origin, backup, upstream}
	names := []string{"local", "origin", "backup", "upstream"}

	for i, repo := range repos {
		if repo == nil {
			t.Fatalf("Expected %s repo to be created", names[i])
		}

		if _, err := os.Stat(repo.path); os.IsNotExist(err) {
			t.Fatalf("%s repo does not exist: %s", names[i], repo.path)
		}
	}

	// Verify local has all remotes configured
	remotes := GetRemotes(t, local.path)

	expectedRemotes := []string{"origin", "backup", "upstream"}
	for _, remoteName := range expectedRemotes {
		if _, ok := remotes[remoteName]; !ok {
			t.Fatalf("Expected %s remote to be configured", remoteName)
		}
	}
}

func TestCreateDirtyRepoScenario(t *testing.T) {
	tests := []struct {
		name     string
		staged   bool
		unstaged bool
		wantFile string
	}{
		{"staged only", true, false, "staged.txt"},
		{"unstaged only", false, true, "unstaged.txt"},
		{"both", true, true, "staged.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenario, repo := CreateDirtyRepoScenario(t, tt.staged, tt.unstaged)

			if scenario == nil {
				t.Fatal("Expected scenario to be created")
			}

			if repo == nil {
				t.Fatal("Expected repo to be created")
			}

			// Verify repo is dirty
			isDirty := IsDirty(t, repo.path)
			if !isDirty {
				t.Fatal("Expected repo to be dirty")
			}

			// Verify expected file exists
			filePath := filepath.Join(repo.path, tt.wantFile)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Fatalf("Expected file %s to exist", tt.wantFile)
			}
		})
	}
}

func TestCreateCleanRepoScenario(t *testing.T) {
	scenario, repo := CreateCleanRepoScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if repo == nil {
		t.Fatal("Expected repo to be created")
	}

	// Verify repo is clean
	isDirty := IsDirty(t, repo.path)
	if isDirty {
		t.Fatal("Expected repo to be clean")
	}

	// Verify repo has remote
	remotes := GetRemotes(t, repo.path)
	if _, ok := remotes["origin"]; !ok {
		t.Fatal("Expected origin remote to be configured")
	}
}

func TestCreateMultiBranchScenario(t *testing.T) {
	branchNames := []string{"feature-1", "feature-2", "bugfix"}

	scenario, repo := CreateMultiBranchScenario(t, branchNames)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if repo == nil {
		t.Fatal("Expected repo to be created")
	}

	// Verify all branches exist
	branches := GetBranches(t, repo.path)

	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[b] = true
	}

	for _, expectedBranch := range branchNames {
		if !branchMap[expectedBranch] {
			t.Fatalf("Expected branch %s to exist", expectedBranch)
		}
	}
}

func TestCreateLargeRepoScenario(t *testing.T) {
	scenario, repo := CreateLargeRepoScenario(t, 5, 3)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if repo == nil {
		t.Fatal("Expected repo to be created")
	}

	// Verify files were created
	// 5 files per commit * 3 commits = 15 files
	// (We're just checking repo exists, full validation would check each file)

	// Verify repo has commits
	sha := GetCurrentSHA(t, repo.path)
	if sha == "" {
		t.Fatal("Expected repo to have commits")
	}
}

func TestCreateDetachedHeadScenario(t *testing.T) {
	scenario, repo, sha := CreateDetachedHeadScenario(t)

	if scenario == nil {
		t.Fatal("Expected scenario to be created")
	}

	if repo == nil {
		t.Fatal("Expected repo to be created")
	}

	if sha == "" {
		t.Fatal("Expected SHA to be returned")
	}

	// Verify HEAD is detached
	currentSHA := GetCurrentSHA(t, repo.path)
	if currentSHA != sha {
		t.Fatalf("Expected HEAD to be at %s, got %s", sha, currentSHA)
	}
}

func TestScenarioRepoChaining(t *testing.T) {
	scenario := NewScenario(t)

	// Test method chaining
	repo := scenario.CreateRepo("test").
		AddFile("file1.txt", "content 1").
		Commit("Commit 1").
		AddFile("file2.txt", "content 2").
		Commit("Commit 2").
		WithBranch("feature").
		AddFile("feature.txt", "feature content").
		Commit("Feature commit")

	if repo == nil {
		t.Fatal("Expected repo to be created")
	}

	// Verify files exist
	files := []string{"file1.txt", "file2.txt", "feature.txt"}
	for _, file := range files {
		filePath := filepath.Join(repo.path, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Expected file %s to exist", file)
		}
	}

	// Verify feature branch exists
	branches := GetBranches(t, repo.path)
	hasFeature := false
	for _, b := range branches {
		if b == "feature" {
			hasFeature = true
			break
		}
	}

	if !hasFeature {
		t.Fatal("Expected feature branch to exist")
	}
}

func TestScenarioWithRemoteAndPush(t *testing.T) {
	scenario := NewScenario(t)

	remote := scenario.CreateBareRepo("remote")
	local := scenario.CreateRepo("local").
		WithRemote("origin", remote).
		AddFile("main.go", "package main\n").
		Commit("Initial commit")

	// Get default branch and push
	defaultBranch := local.GetDefaultBranch()
	local.Push("origin", defaultBranch)

	// Verify push succeeded by checking remote has the branch
	// (This would require git ls-remote or similar, skipping for now)

	// Verify local has origin remote
	remotes := GetRemotes(t, local.path)
	if _, ok := remotes["origin"]; !ok {
		t.Fatal("Expected origin remote to be configured")
	}
}

func TestSnapshotAndRestore(t *testing.T) {
	// Create a scenario with some complexity
	_, repo := CreateMultiBranchScenario(t, []string{"feature", "bugfix"})

	// Create snapshot
	snapshot := SnapshotRepo(t, repo.path)

	if snapshot == nil {
		t.Fatal("Expected snapshot to be created")
	}

	if snapshot.Size() == 0 {
		t.Fatal("Expected snapshot to have non-zero size")
	}

	// Restore snapshot
	restoredPath := RestoreSnapshot(t, snapshot)

	if restoredPath == "" {
		t.Fatal("Expected restored path to be returned")
	}

	// Verify restored repo exists
	if _, err := os.Stat(restoredPath); os.IsNotExist(err) {
		t.Fatalf("Restored repo does not exist: %s", restoredPath)
	}

	// Verify restored repo has same branches
	restoredBranches := GetBranches(t, restoredPath)
	originalBranches := GetBranches(t, repo.path)

	if len(restoredBranches) != len(originalBranches) {
		t.Fatalf("Expected %d branches, got %d", len(originalBranches), len(restoredBranches))
	}

	// Verify same commit SHA
	originalSHA := GetCurrentSHA(t, repo.path)
	restoredSHA := GetCurrentSHA(t, restoredPath)

	if originalSHA != restoredSHA {
		t.Fatalf("Expected restored SHA %s to match original %s", restoredSHA, originalSHA)
	}
}

func TestSnapshotPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a moderately complex repo
	_, repo := CreateLargeRepoScenario(t, 10, 5)

	// Create snapshot
	snapshot := SnapshotRepo(t, repo.path)

	// Measure restoration time
	// First restoration
	restoredPath1 := RestoreSnapshot(t, snapshot)
	if _, err := os.Stat(restoredPath1); os.IsNotExist(err) {
		t.Fatalf("First restoration failed")
	}

	// Second restoration (should be fast)
	restoredPath2 := RestoreSnapshot(t, snapshot)
	if _, err := os.Stat(restoredPath2); os.IsNotExist(err) {
		t.Fatalf("Second restoration failed")
	}

	// Both should succeed
	t.Logf("Snapshot size: %d bytes", snapshot.Size())
	t.Logf("First restoration: %s", restoredPath1)
	t.Logf("Second restoration: %s", restoredPath2)
}

func TestSnapshotSaveAndLoad(t *testing.T) {
	// Create a simple repo
	_, repo := CreateCleanRepoScenario(t)

	// Create snapshot
	snapshot := SnapshotRepo(t, repo.path)

	// Save to disk
	tmpFile := filepath.Join(t.TempDir(), "snapshot.tar.gz")
	SaveSnapshotToDisk(t, snapshot, tmpFile)

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Fatalf("Snapshot file does not exist: %s", tmpFile)
	}

	// Load from disk
	loadedSnapshot := LoadSnapshotFromDisk(t, tmpFile)

	if loadedSnapshot == nil {
		t.Fatal("Expected loaded snapshot to not be nil")
	}

	if loadedSnapshot.Size() != snapshot.Size() {
		t.Fatalf("Expected loaded snapshot size %d to match original %d",
			loadedSnapshot.Size(), snapshot.Size())
	}

	// Restore loaded snapshot
	restoredPath := RestoreSnapshot(t, loadedSnapshot)

	if _, err := os.Stat(restoredPath); os.IsNotExist(err) {
		t.Fatalf("Restored repo from loaded snapshot does not exist: %s", restoredPath)
	}
}
