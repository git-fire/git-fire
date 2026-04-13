package git

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	testutil "github.com/git-fire/git-testkit"
)

func TestAutoCommitDirty(t *testing.T) {
	tests := []struct {
		name    string
		dirty   bool
		wantErr bool
	}{
		{
			name:    "commits dirty repo",
			dirty:   true,
			wantErr: false,
		},
		{
			name:    "handles clean repo",
			dirty:   false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
				Name:  "test-repo",
				Dirty: tt.dirty,
			})

			err := AutoCommitDirty(repo, CommitOptions{
				AddAll:  true,
				Message: "Emergency backup",
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("AutoCommitDirty() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify repo is clean after commit
			if tt.dirty && !tt.wantErr {
				isDirty := testutil.IsDirty(t, repo)
				if isDirty {
					t.Error("Repo should be clean after auto-commit")
				}
			}
		})
	}
}

func TestDetectConflict(t *testing.T) {
	// Create bare remote
	remoteRepo := testutil.CreateBareRemote(t, "origin")

	// Create local repo and push to remote
	localRepo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "local",
		Remotes: map[string]string{
			"origin": remoteRepo,
		},
		Files: map[string]string{
			"file.txt": "initial content",
		},
	})

	// Get the current branch name (could be main or master)
	currentBranch, err := GetCurrentBranch(localRepo)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Push to remote
	if err := PushBranch(localRepo, "origin", currentBranch); err != nil {
		t.Fatalf("Failed to push initial branch: %v", err)
	}

	tests := []struct {
		name         string
		setup        func(*testing.T, string) // Setup function receives localRepo path
		wantConflict bool
		wantErr      bool
	}{
		{
			name: "no conflict - branches match",
			setup: func(t *testing.T, repo string) {
				// Do nothing - branches are already in sync
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name: "no conflict - local ahead only",
			setup: func(t *testing.T, repo string) {
				// Add local commit
				newFile := filepath.Join(repo, "new.txt")
				if err := os.WriteFile(newFile, []byte("new content"), 0644); err != nil {
					t.Fatal(err)
				}
				testutil.RunGitCmd(t, repo, "add", "new.txt")
				testutil.RunGitCmd(t, repo, "commit", "-m", "Local commit")
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name: "conflict - true divergence",
			setup: func(t *testing.T, repo string) {
				// Local commit (ahead locally)
				localFile := filepath.Join(repo, "local-only.txt")
				if err := os.WriteFile(localFile, []byte("local content"), 0644); err != nil {
					t.Fatal(err)
				}
				testutil.RunGitCmd(t, repo, "add", "local-only.txt")
				testutil.RunGitCmd(t, repo, "commit", "-m", "Local diverging commit")

				// Independent remote commit from another clone (ahead remotely)
				cloneBase := t.TempDir()
				peerDir := filepath.Join(cloneBase, "peer")
				testutil.RunGitCmd(t, cloneBase, "clone", remoteRepo, peerDir)
				testutil.RunGitCmd(t, peerDir, "config", "user.email", "test@example.com")
				testutil.RunGitCmd(t, peerDir, "config", "user.name", "Test User")

				peerFile := filepath.Join(peerDir, "remote-only.txt")
				if err := os.WriteFile(peerFile, []byte("remote content"), 0644); err != nil {
					t.Fatal(err)
				}
				testutil.RunGitCmd(t, peerDir, "add", "remote-only.txt")
				testutil.RunGitCmd(t, peerDir, "commit", "-m", "Remote diverging commit")
				testutil.RunGitCmd(t, peerDir, "push", "origin", currentBranch)
			},
			wantConflict: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset repo state
			testutil.RunGitCmd(t, localRepo, "reset", "--hard", "origin/"+currentBranch)

			if tt.setup != nil {
				tt.setup(t, localRepo)
			}

			hasConflict, localSHA, remoteSHA, err := DetectConflict(localRepo, currentBranch, "origin")

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectConflict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if hasConflict != tt.wantConflict {
				t.Errorf("DetectConflict() conflict = %v, want %v (local=%s, remote=%s)",
					hasConflict, tt.wantConflict, localSHA, remoteSHA)
			}

			if !tt.wantErr {
				if localSHA == "" {
					t.Error("Expected localSHA to be set")
				}
			}
		})
	}
}

func TestAutoCommitDirtyWithStrategy_FailureCleansUpUnbornHead(t *testing.T) {
	repo := t.TempDir()
	testutil.RunGitCmd(t, repo, "init")
	testutil.RunGitCmd(t, repo, "config", "user.email", "test@example.com")
	testutil.RunGitCmd(t, repo, "config", "user.name", "Test User")

	// Prepare staged + unstaged changes with no initial commit.
	stagedFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(stagedFile, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	unstagedFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(unstagedFile, []byte("unstaged"), 0644); err != nil {
		t.Fatal(err)
	}

	// Fail only the second commit ("full backup").
	hooksDir := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	hookPath := filepath.Join(hooksDir, "commit-msg")
	hook := `#!/bin/sh
msg_file="$1"
if grep -q "full backup" "$msg_file"; then
  exit 1
fi
exit 0
`
	if err := os.WriteFile(hookPath, []byte(hook), 0755); err != nil {
		t.Fatalf("write commit-msg hook: %v", err)
	}
	testutil.RunGitCmd(t, repo, "config", "core.hooksPath", hooksDir)

	_, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: false,
	})
	if err == nil {
		t.Fatal("expected auto-commit strategy to fail on second commit")
	}

	// Ensure we returned to unborn HEAD state.
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	cmd.Dir = repo
	if output, cmdErr := cmd.CombinedOutput(); cmdErr == nil {
		t.Fatalf("expected unborn HEAD after cleanup, but rev-parse HEAD succeeded: %s", strings.TrimSpace(string(output)))
	}

	assertUncommittedFilesContain(t, repo, "staged.txt", "unstaged.txt")
	assertStatusEntries(t, repo, "A  staged.txt", "?? unstaged.txt")
}

func TestCreateFireBranch(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
		Files: map[string]string{
			"file.txt": "content",
		},
	})

	tests := []struct {
		name           string
		originalBranch string
		wantPrefix     string
		wantErr        bool
	}{
		{
			name:           "creates fire branch from main",
			originalBranch: "main",
			wantPrefix:     "git-fire-backup-main-",
			wantErr:        false,
		},
		{
			name:           "creates fire branch from feature",
			originalBranch: "feature",
			wantPrefix:     "git-fire-backup-feature-",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get current commit SHA
			localSHA := testutil.GetCurrentSHA(t, repo)

			branchName, err := CreateFireBranch(repo, tt.originalBranch, localSHA)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFireBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !strings.HasPrefix(branchName, tt.wantPrefix) {
					t.Errorf("CreateFireBranch() = %v, want prefix %v", branchName, tt.wantPrefix)
				}

				// Verify branch exists
				branches := testutil.GetBranches(t, repo)
				found := false
				for _, b := range branches {
					if b == branchName {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Fire branch %s not found in repo", branchName)
				}
			}
		})
	}
}

func TestPushBranch(t *testing.T) {
	// Create bare remote
	remoteRepo := testutil.CreateBareRemote(t, "origin")

	// Create local repo
	localRepo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "local",
		Remotes: map[string]string{
			"origin": remoteRepo,
		},
		Files: map[string]string{
			"file.txt": "content",
		},
	})

	// Get current branch name
	currentBranch, err := GetCurrentBranch(localRepo)
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "pushes current branch",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PushBranch(localRepo, "origin", currentBranch)

			if (err != nil) != tt.wantErr {
				t.Errorf("PushBranch() error = %v, wantErr %v", err, tt.wantErr)
			}

			// TODO: Verify branch exists on remote
		})
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
		Branches: []string{
			"feature",
		},
	})

	tests := []struct {
		name       string
		setup      func()
		wantBranch string
		wantErr    bool
	}{
		{
			name:       "gets current branch",
			setup:      func() {},
			wantBranch: "main", // or "master" depending on git version
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			branch, err := GetCurrentBranch(repo)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentBranch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && branch == "" {
				t.Error("GetCurrentBranch() returned empty branch")
			}
		})
	}
}

func TestHasStagedChanges(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Initially no staged changes
	hasStaged, err := HasStagedChanges(repo)
	if err != nil {
		t.Fatalf("HasStagedChanges() error = %v", err)
	}
	if hasStaged {
		t.Error("Expected no staged changes initially")
	}

	// Stage a file
	testFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(testFile, []byte("staged content"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	// Now should have staged changes
	hasStaged, err = HasStagedChanges(repo)
	if err != nil {
		t.Fatalf("HasStagedChanges() error = %v", err)
	}
	if !hasStaged {
		t.Error("Expected staged changes after git add")
	}
}

func TestHasUnstagedChanges(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
		Files: map[string]string{
			"test.txt": "initial",
		},
	})

	// Initially no unstaged changes
	hasUnstaged, err := HasUnstagedChanges(repo)
	if err != nil {
		t.Fatalf("HasUnstagedChanges() error = %v", err)
	}
	if hasUnstaged {
		t.Error("Expected no unstaged changes initially")
	}

	// Create an unstaged file
	testFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(testFile, []byte("unstaged content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now should have unstaged changes
	hasUnstaged, err = HasUnstagedChanges(repo)
	if err != nil {
		t.Fatalf("HasUnstagedChanges() error = %v", err)
	}
	if !hasUnstaged {
		t.Error("Expected unstaged changes after creating file")
	}

	// Modify an existing tracked file
	existingFile := filepath.Join(repo, "test.txt")
	if err := os.WriteFile(existingFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	hasUnstaged, err = HasUnstagedChanges(repo)
	if err != nil {
		t.Fatalf("HasUnstagedChanges() error = %v", err)
	}
	if !hasUnstaged {
		t.Error("Expected unstaged changes after modifying file")
	}
}

func TestAutoCommitDirtyWithStrategy_OnlyStaged(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Stage a file
	testFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(testFile, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	// Run with strategy
	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: false, // Keep the commit for verification
	})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}

	// Should create only staged branch
	if result.StagedBranch == "" {
		t.Error("Expected staged branch to be created")
	}
	if result.FullBranch != "" {
		t.Errorf("Expected no full branch, got %s", result.FullBranch)
	}
	if result.BothCreated {
		t.Error("Expected BothCreated to be false")
	}

	// Verify branch exists
	branches := testutil.GetBranches(t, repo)
	found := false
	for _, b := range branches {
		if b == result.StagedBranch {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Branch %s not found in repo", result.StagedBranch)
	}
}

func TestAutoCommitDirtyWithStrategy_OnlyUnstaged(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Create an unstaged file
	testFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(testFile, []byte("unstaged"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with strategy
	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: false,
	})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}

	// Should create only full branch
	if result.StagedBranch != "" {
		t.Errorf("Expected no staged branch, got %s", result.StagedBranch)
	}
	if result.FullBranch == "" {
		t.Error("Expected full branch to be created")
	}
	if result.BothCreated {
		t.Error("Expected BothCreated to be false")
	}

	// Verify branch exists
	branches := testutil.GetBranches(t, repo)
	found := false
	for _, b := range branches {
		if b == result.FullBranch {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Branch %s not found in repo", result.FullBranch)
	}
}

func TestAutoCommitDirtyWithStrategy_Both(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Stage a file
	stagedFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(stagedFile, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	// Create an unstaged file
	unstagedFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(unstagedFile, []byte("unstaged"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with strategy
	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: false,
	})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}

	// Should create BOTH branches
	if result.StagedBranch == "" {
		t.Error("Expected staged branch to be created")
	}
	if result.FullBranch == "" {
		t.Error("Expected full branch to be created")
	}
	if !result.BothCreated {
		t.Error("Expected BothCreated to be true")
	}

	// Verify both branches exist
	branches := testutil.GetBranches(t, repo)
	foundStaged := false
	foundFull := false
	for _, b := range branches {
		if b == result.StagedBranch {
			foundStaged = true
		}
		if b == result.FullBranch {
			foundFull = true
		}
	}
	if !foundStaged {
		t.Errorf("Staged branch %s not found", result.StagedBranch)
	}
	if !foundFull {
		t.Errorf("Full branch %s not found", result.FullBranch)
	}
}

func TestAutoCommitDirtyWithStrategy_Clean(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Repo is clean, no changes
	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}

	// Should create no branches
	if result.StagedBranch != "" {
		t.Errorf("Expected no staged branch, got %s", result.StagedBranch)
	}
	if result.FullBranch != "" {
		t.Errorf("Expected no full branch, got %s", result.FullBranch)
	}
	if result.BothCreated {
		t.Error("Expected BothCreated to be false")
	}
}

func TestAutoCommitDirtyWithStrategy_ReturnToOriginal(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Get original HEAD
	originalSHA := testutil.GetCurrentSHA(t, repo)

	// Stage and create unstaged changes
	stagedFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(stagedFile, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	unstagedFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(unstagedFile, []byte("unstaged"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with ReturnToOriginal = true (default)
	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: true,
	})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}

	// Should have created branches
	if result.StagedBranch == "" || result.FullBranch == "" {
		t.Error("Expected both branches to be created")
	}

	// HEAD should be back to original
	currentSHA := testutil.GetCurrentSHA(t, repo)
	if currentSHA != originalSHA {
		t.Errorf("Expected HEAD to return to %s, got %s", originalSHA, currentSHA)
	}

	assertUncommittedFilesContain(t, repo, "staged.txt", "unstaged.txt")
	assertStatusEntries(t, repo, "A  staged.txt", "?? unstaged.txt")
}

func TestAutoCommitDirtyWithStrategy_ReturnToOriginal_PreservesIndexState(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})
	filePath := filepath.Join(repo, "partial.txt")
	if err := os.WriteFile(filePath, []byte("base\n"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "partial.txt")
	testutil.RunGitCmd(t, repo, "commit", "-m", "add partial file")

	// Worktree content differs from staged content.
	if err := os.WriteFile(filePath, []byte("worktree\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Directly write a blob into index to emulate a partial-stage style state.
	writeBlob := exec.Command("git", "hash-object", "-w", "--stdin")
	writeBlob.Dir = repo
	writeBlob.Stdin = strings.NewReader("staged\n")
	blobOut, err := writeBlob.CombinedOutput()
	if err != nil {
		t.Fatalf("git hash-object -w --stdin failed: %v (%s)", err, strings.TrimSpace(string(blobOut)))
	}
	blobSHA := strings.TrimSpace(string(blobOut))
	testutil.RunGitCmd(t, repo, "update-index", "--cacheinfo", "100644", blobSHA, "partial.txt")

	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: true,
	})
	if err != nil {
		t.Fatalf("AutoCommitDirtyWithStrategy() error = %v", err)
	}
	if result.StagedBranch == "" {
		t.Fatal("expected staged branch to be created")
	}

	// Index content should remain exactly as originally staged.
	indexShow := exec.Command("git", "show", ":partial.txt")
	indexShow.Dir = repo
	indexOut, err := indexShow.CombinedOutput()
	if err != nil {
		t.Fatalf("git show :partial.txt failed: %v (%s)", err, strings.TrimSpace(string(indexOut)))
	}
	if string(indexOut) != "staged\n" {
		t.Fatalf("expected index content to remain staged variant, got %q", string(indexOut))
	}

	// Worktree content should remain untouched.
	wtBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read worktree file: %v", err)
	}
	if string(wtBytes) != "worktree\n" {
		t.Fatalf("expected worktree content to remain worktree variant, got %q", string(wtBytes))
	}
}

func TestAutoCommitDirtyWithStrategy_FailureCleansUpToOriginalHead(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	originalSHA := testutil.GetCurrentSHA(t, repo)

	// Prepare staged + unstaged changes so the strategy performs two commits.
	stagedFile := filepath.Join(repo, "staged.txt")
	if err := os.WriteFile(stagedFile, []byte("staged"), 0644); err != nil {
		t.Fatal(err)
	}
	testutil.RunGitCmd(t, repo, "add", "staged.txt")

	unstagedFile := filepath.Join(repo, "unstaged.txt")
	if err := os.WriteFile(unstagedFile, []byte("unstaged"), 0644); err != nil {
		t.Fatal(err)
	}

	// Fail only the second commit ("full backup") via commit-msg hook.
	hooksDir := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("mkdir hooks dir: %v", err)
	}
	hookPath := filepath.Join(hooksDir, "commit-msg")
	hook := `#!/bin/sh
msg_file="$1"
if grep -q "full backup" "$msg_file"; then
  exit 1
fi
exit 0
`
	if err := os.WriteFile(hookPath, []byte(hook), 0755); err != nil {
		t.Fatalf("write commit-msg hook: %v", err)
	}
	testutil.RunGitCmd(t, repo, "config", "core.hooksPath", hooksDir)

	result, err := AutoCommitDirtyWithStrategy(repo, CommitOptions{
		ReturnToOriginal: false, // failure cleanup must still reset to original
	})
	if err == nil {
		t.Fatal("expected auto-commit strategy to fail on second commit")
	}
	if !strings.Contains(err.Error(), "failed to commit unstaged changes") {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil || result.StagedBranch == "" {
		t.Fatalf("expected partial result with staged backup branch on failure, got %+v", result)
	}

	currentSHA := testutil.GetCurrentSHA(t, repo)
	if currentSHA != originalSHA {
		t.Fatalf("expected cleanup reset to original HEAD %s, got %s", originalSHA, currentSHA)
	}

	assertUncommittedFilesContain(t, repo, "staged.txt", "unstaged.txt")
	assertStatusEntries(t, repo, "A  staged.txt", "?? unstaged.txt")
}

func assertUncommittedFilesContain(t *testing.T, repo string, expected ...string) {
	t.Helper()

	files, err := GetUncommittedFiles(repo)
	if err != nil {
		t.Fatalf("GetUncommittedFiles() error: %v", err)
	}

	fileSet := make(map[string]struct{}, len(files))
	for _, f := range files {
		fileSet[f] = struct{}{}
	}

	for _, want := range expected {
		if _, ok := fileSet[want]; !ok {
			t.Fatalf("expected %q in uncommitted files, got %v", want, files)
		}
	}
}

func assertStatusEntries(t *testing.T, repo string, expected ...string) {
	t.Helper()

	cmd := exec.Command("git", "status", "--porcelain=v1")
	cmd.Dir = repo
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git status --porcelain=v1 failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	set := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		set[line] = struct{}{}
	}

	for _, want := range expected {
		if _, ok := set[want]; !ok {
			t.Fatalf("expected status entry %q, got %v", want, lines)
		}
	}
}

func TestListWorktrees(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// List worktrees - should have just the main one
	worktrees, err := ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("Expected 1 worktree, got %d", len(worktrees))
	}

	if !worktrees[0].IsMain {
		t.Error("Expected first worktree to be main")
	}

	if worktrees[0].Path == "" {
		t.Error("Expected worktree path to be set")
	}

	// Create an additional worktree
	worktreePath := filepath.Join(t.TempDir(), "worktree2")
	testutil.RunGitCmd(t, repo, "worktree", "add", worktreePath, "-b", "feature")

	// List again
	worktrees, err = ListWorktrees(repo)
	if err != nil {
		t.Fatalf("ListWorktrees() error = %v", err)
	}

	if len(worktrees) != 2 {
		t.Fatalf("Expected 2 worktrees, got %d", len(worktrees))
	}

	// Verify second worktree
	var featureWorktree *Worktree
	for i := range worktrees {
		if worktrees[i].Branch == "feature" {
			featureWorktree = &worktrees[i]
			break
		}
	}

	if featureWorktree == nil {
		t.Error("Expected to find feature worktree")
	} else {
		if featureWorktree.IsMain {
			t.Error("Expected feature worktree to not be main")
		}
		if featureWorktree.Path != worktreePath {
			t.Errorf("Expected worktree path %s, got %s", worktreePath, featureWorktree.Path)
		}
	}
}

// TestAutoCommitDirty_DetachedHead validates that AutoCommitDirty fails safely
// when the repo is in detached HEAD state. The original qw3rtman/git-fire (bash)
// was known to behave incorrectly in this scenario.
func TestAutoCommitDirty_DetachedHead(t *testing.T) {
	_, repo, _ := testutil.CreateDetachedHeadScenario(t)

	// Add an untracked file so the repo is dirty
	dirtyFile := filepath.Join(repo.Path(), "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("change\n"), 0644); err != nil {
		t.Fatalf("failed to create dirty file: %v", err)
	}

	err := AutoCommitDirty(repo.Path(), CommitOptions{AddAll: true})
	if err == nil {
		t.Fatal("expected an error for detached HEAD, got nil")
	}
	// git commit fails with "not a branch" / "HEAD detached" messaging
	errMsg := err.Error()
	if !strings.Contains(errMsg, "detached") &&
		!strings.Contains(errMsg, "not a branch") &&
		!strings.Contains(errMsg, "HEAD") {
		t.Errorf("unexpected error message for detached HEAD: %v", err)
	}
}

// TestGetCurrentBranch_DetachedHead validates that GetCurrentBranch returns an
// explicit error (not an empty string) when HEAD is detached.
func TestGetCurrentBranch_DetachedHead(t *testing.T) {
	_, repo, _ := testutil.CreateDetachedHeadScenario(t)

	branch, err := GetCurrentBranch(repo.Path())
	if err == nil {
		t.Fatalf("expected error for detached HEAD, got branch=%q", branch)
	}
	if !strings.Contains(err.Error(), "detached HEAD") && !strings.Contains(err.Error(), "not on any branch") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGetUncommittedFiles(t *testing.T) {
	t.Run("mixed_working_tree", func(t *testing.T) {
		scenario := testutil.NewScenario(t)
		repo := scenario.CreateRepo("test")

		// Committed file that will be modified in working tree ( M status)
		repo.AddFile("tracked.txt", "original").Commit("add tracked")
		repo.ModifyFile("tracked.txt", "modified")

		// Staged new file (A  status) — added after the commit above
		repo.AddFile("staged.txt", "staged content")

		// Untracked file (written directly, not staged)
		if err := os.WriteFile(filepath.Join(repo.Path(), "untracked.txt"), []byte("hello\n"), 0644); err != nil {
			t.Fatalf("failed to write untracked file: %v", err)
		}

		files, err := GetUncommittedFiles(repo.Path())
		if err != nil {
			t.Fatalf("GetUncommittedFiles() error = %v", err)
		}

		fileSet := make(map[string]bool, len(files))
		for _, f := range files {
			fileSet[f] = true
		}

		if !fileSet["staged.txt"] {
			t.Errorf("expected staged.txt in results, got %v", files)
		}
		if !fileSet["tracked.txt"] {
			t.Errorf("expected tracked.txt in results, got %v", files)
		}
		if !fileSet["untracked.txt"] {
			t.Errorf("expected untracked.txt in results, got %v", files)
		}
	})

	t.Run("deleted_tracked_file", func(t *testing.T) {
		scenario := testutil.NewScenario(t)
		repo := scenario.CreateRepo("test")
		repo.AddFile("remove-me.txt", "content\n").Commit("add file")
		if err := os.Remove(filepath.Join(repo.Path(), "remove-me.txt")); err != nil {
			t.Fatalf("remove file: %v", err)
		}

		files, err := GetUncommittedFiles(repo.Path())
		if err != nil {
			t.Fatalf("GetUncommittedFiles() error = %v", err)
		}
		found := false
		for _, f := range files {
			if f == "remove-me.txt" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected deleted tracked file remove-me.txt in results, got %v", files)
		}
	})
}

func TestGetUncommittedFiles_SpacesInPath(t *testing.T) {
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("test")

	repo.AddFile("normal.txt", "content").Commit("initial")

	// File with spaces in the name — old line-based parsers would split on the space
	if err := os.WriteFile(filepath.Join(repo.Path(), "file with spaces.txt"), []byte("spaced\n"), 0644); err != nil {
		t.Fatalf("failed to write spaced file: %v", err)
	}

	files, err := GetUncommittedFiles(repo.Path())
	if err != nil {
		t.Fatalf("GetUncommittedFiles() error = %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	if !fileSet["file with spaces.txt"] {
		t.Errorf("expected 'file with spaces.txt' in results, got %v", files)
	}
}

func TestGetUncommittedFiles_Rename(t *testing.T) {
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("test")

	// Commit a file, then rename it so it shows as R in git status
	repo.AddFile("old-name.txt", "content").StageFile("old-name.txt").Commit("add file")

	// Rename via git mv so it's staged as a rename
	testutil.RunGitCmd(t, repo.Path(), "mv", "old-name.txt", "new-name.txt")

	files, err := GetUncommittedFiles(repo.Path())
	if err != nil {
		t.Fatalf("GetUncommittedFiles() error = %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	// Should return the new (destination) path, not the old one
	if !fileSet["new-name.txt"] {
		t.Errorf("expected new-name.txt (rename destination) in results, got %v", files)
	}
	if fileSet["old-name.txt"] {
		t.Errorf("old-name.txt (rename source) should not be in results, got %v", files)
	}
}

func TestGetUncommittedFiles_Copy(t *testing.T) {
	scenario := testutil.NewScenario(t)
	repo := scenario.CreateRepo("test")

	// Commit a file with enough content for Git's similarity detection
	content := "copy detection content: this file will be copied to a new path\n"
	repo.AddFile("old-name.txt", content).StageFile("old-name.txt").Commit("add file")

	// Enable copy detection so git status reports C instead of A for similar new files
	testutil.RunGitCmd(t, repo.Path(), "config", "status.renames", "copies")

	// Stage a new file with identical content — Git should detect it as a copy
	repo.AddFile("new-name.txt", content).StageFile("new-name.txt")

	files, err := GetUncommittedFiles(repo.Path())
	if err != nil {
		t.Fatalf("GetUncommittedFiles() error = %v", err)
	}

	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	// Should return the copy destination (new path), not the source
	if !fileSet["new-name.txt"] {
		t.Errorf("expected new-name.txt (copy destination) in results, got %v", files)
	}
	if fileSet["old-name.txt"] {
		t.Errorf("old-name.txt (copy source) should not be in results, got %v", files)
	}
}

func TestGetUncommittedFiles_Clean(t *testing.T) {
	_, repo := testutil.CreateCleanRepoScenario(t)

	files, err := GetUncommittedFiles(repo.Path())
	if err != nil {
		t.Fatalf("GetUncommittedFiles() error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no files for clean repo, got %v", files)
	}
}

func TestPushAllBranches(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "test-repo",
		Remotes: map[string]string{"origin": remote},
	})

	localSHA := testutil.GetCurrentSHA(t, repo)

	if err := PushAllBranches(repo, "origin"); err != nil {
		t.Errorf("PushAllBranches() error = %v", err)
	}

	// Verify the remote actually has the commit we pushed
	branch, err := GetCurrentBranch(repo)
	if err != nil {
		t.Fatalf("GetCurrentBranch: %v", err)
	}
	remoteSHA := getBareBranchSHA(t, remote, branch)
	if remoteSHA != localSHA {
		t.Errorf("remote SHA = %s, want %s — branch was not actually pushed", remoteSHA, localSHA)
	}
}

func TestPushAllBranches_InvalidRemote(t *testing.T) {
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{Name: "test-repo"})

	if err := PushAllBranches(repo, "nonexistent"); err == nil {
		t.Error("expected error pushing to nonexistent remote")
	}
}

func TestPushKnownBranches(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "test-repo",
		Remotes: map[string]string{"origin": remote},
	})

	// Push once to establish the remote branch
	if err := PushAllBranches(repo, "origin"); err != nil {
		t.Fatalf("initial push: %v", err)
	}

	// Add a new commit so we can verify PushKnownBranches actually moves the remote ref
	testutil.RunGitCmd(t, repo, "commit", "--allow-empty", "-m", "second commit")
	localSHA := testutil.GetCurrentSHA(t, repo)

	if err := PushKnownBranches(repo, "origin"); err != nil {
		t.Errorf("PushKnownBranches() error = %v", err)
	}

	// Confirm the remote ref was updated to the new commit
	remoteSHA := testutil.GetCurrentSHA(t, remote)
	if remoteSHA != localSHA {
		t.Errorf("remote SHA = %s, want %s — branch was not pushed", remoteSHA, localSHA)
	}
}

func TestPushKnownBranches_NoRemoteBranches(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "test-repo",
		Remotes: map[string]string{"origin": remote},
	})

	// No branches pushed yet — PushKnownBranches should succeed (nothing to do)
	if err := PushKnownBranches(repo, "origin"); err != nil {
		t.Errorf("PushKnownBranches() with no remote branches: error = %v", err)
	}
}

func TestPushKnownBranches_WarnsForLocalOnlyBranch(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:    "test-repo",
		Remotes: map[string]string{"origin": remote},
	})

	// Establish the default branch on remote.
	currentBranch, err := GetCurrentBranch(repo)
	if err != nil {
		t.Fatalf("GetCurrentBranch() error = %v", err)
	}
	if err := PushBranch(repo, "origin", currentBranch); err != nil {
		t.Fatalf("initial push failed: %v", err)
	}

	// Create local-only branch that does not exist on remote.
	testutil.RunGitCmd(t, repo, "checkout", "-b", "feature-local-only")
	testutil.RunGitCmd(t, repo, "commit", "--allow-empty", "-m", "feature work")
	testutil.RunGitCmd(t, repo, "checkout", "-")

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	callErr := PushKnownBranches(repo, "origin")

	w.Close()
	os.Stderr = origStderr

	if callErr != nil {
		t.Fatalf("PushKnownBranches() error = %v", callErr)
	}

	var stderr bytes.Buffer
	if _, err := stderr.ReadFrom(r); err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(stderr.String(), "feature-local-only") {
		t.Fatalf("expected warning for local-only branch, got: %q", stderr.String())
	}
}

func TestPushKnownBranches_ContinuesOnError(t *testing.T) {
	remote := testutil.CreateBareRemote(t, "origin")
	repo := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:     "test-repo",
		Remotes:  map[string]string{"origin": remote},
		Branches: []string{"accept-branch", "reject-branch"},
	})

	// Push all branches to establish them on the remote
	if err := PushAllBranches(repo, "origin"); err != nil {
		t.Fatalf("initial push: %v", err)
	}

	// accept-branch: add a new commit (fast-forward friendly)
	testutil.RunGitCmd(t, repo, "checkout", "accept-branch")
	testutil.RunGitCmd(t, repo, "commit", "--allow-empty", "-m", "extra commit on accept-branch")
	acceptLocalSHA := testutil.GetCurrentSHA(t, repo)

	// reject-branch: push an extra commit to the remote, then rewind local and
	// add a different commit — local and remote now have diverged histories.
	testutil.RunGitCmd(t, repo, "checkout", "reject-branch")
	testutil.RunGitCmd(t, repo, "commit", "--allow-empty", "-m", "commit pushed to remote")
	testutil.RunGitCmd(t, repo, "push", "origin", "reject-branch") // advance remote
	testutil.RunGitCmd(t, repo, "reset", "--hard", "HEAD^")        // rewind local
	testutil.RunGitCmd(t, repo, "commit", "--allow-empty", "-m", "diverged local commit")

	// Return to the default branch before calling PushKnownBranches
	testutil.RunGitCmd(t, repo, "checkout", "-")

	err := PushKnownBranches(repo, "origin")
	if err == nil {
		t.Error("expected error: reject-branch should fail with non-fast-forward rejection")
	}

	// accept-branch must have been pushed despite reject-branch failing
	acceptRemoteSHA := getBareBranchSHA(t, remote, "accept-branch")
	if acceptRemoteSHA != acceptLocalSHA {
		t.Errorf("accept-branch remote SHA = %s, want %s — branch was not pushed after sibling failure",
			acceptRemoteSHA, acceptLocalSHA)
	}
}

// getBareBranchSHA returns the SHA of a branch in a bare remote repository.
func getBareBranchSHA(t *testing.T, bareRemote, branch string) string {
	t.Helper()
	sha, err := getCommitSHA(bareRemote, "refs/heads/"+branch)
	if err != nil {
		t.Fatalf("get remote branch SHA %s: %v", branch, err)
	}
	return sha
}
