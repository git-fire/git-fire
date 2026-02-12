package testutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TBRX103/git-fire/internal/testutil"
)

func TestCreateTestRepo(t *testing.T) {
	// Test creating a basic clean repo
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "test-repo",
	})

	// Verify .git directory exists
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatalf("Expected .git directory to exist at %s", gitDir)
	}

	// Verify repo is clean (not dirty)
	if testutil.IsDirty(t, repoPath) {
		t.Fatal("Expected repo to be clean, but it has uncommitted changes")
	}
}

func TestCreateTestRepo_Dirty(t *testing.T) {
	// Test creating a dirty repo
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name:  "dirty-repo",
		Dirty: true,
	})

	// Verify repo is dirty
	if !testutil.IsDirty(t, repoPath) {
		t.Fatal("Expected repo to be dirty, but it's clean")
	}
}

func TestCreateTestRepo_WithFiles(t *testing.T) {
	// Test creating a repo with custom files
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "files-repo",
		Files: map[string]string{
			"test.txt":       "test content",
			"src/main.go":    "package main",
			"config/app.yml": "port: 8080",
		},
	})

	// Verify files exist
	files := []string{"test.txt", "src/main.go", "config/app.yml"}
	for _, file := range files {
		filePath := filepath.Join(repoPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Fatalf("Expected file to exist: %s", filePath)
		}
	}
}

func TestCreateTestRepo_WithRemotes(t *testing.T) {
	// Create a bare remote first
	remotePath := testutil.CreateBareRemote(t, "origin")

	// Create repo with remote configured
	repoPath := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "remote-repo",
		Remotes: map[string]string{
			"origin": remotePath,
		},
	})

	// Verify remote is configured
	remotes := testutil.GetRemotes(t, repoPath)
	if _, exists := remotes["origin"]; !exists {
		t.Fatal("Expected 'origin' remote to be configured")
	}
}

func TestCreateBareRemote(t *testing.T) {
	remotePath := testutil.CreateBareRemote(t, "test-remote")

	// Verify it's a bare repo
	gitDir := filepath.Join(remotePath, "config")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatalf("Expected bare repo config to exist at %s", gitDir)
	}
}

func TestSetupFakeFilesystem(t *testing.T) {
	fsRoot := testutil.SetupFakeFilesystem(t)

	// Verify expected directories exist
	dirs := []string{
		"home/testuser/projects",
		"home/testuser/.cache",
		"root/sys",
	}

	for _, dir := range dirs {
		dirPath := filepath.Join(fsRoot, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Fatalf("Expected directory to exist: %s", dirPath)
		}
	}
}
