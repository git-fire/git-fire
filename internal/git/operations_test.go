package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TBRX103/git-fire/internal/testutil"
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
		setup        func(string) // Setup function receives localRepo path
		wantConflict bool
		wantErr      bool
	}{
		{
			name: "no conflict - branches match",
			setup: func(repo string) {
				// Do nothing - branches are already in sync
			},
			wantConflict: false,
			wantErr:      false,
		},
		{
			name: "conflict - local ahead",
			setup: func(repo string) {
				// Add local commit
				newFile := filepath.Join(repo, "new.txt")
				os.WriteFile(newFile, []byte("new content"), 0644)
				testutil.RunGitCmd(t, repo, "add", "new.txt")
				testutil.RunGitCmd(t, repo, "commit", "-m", "Local commit")
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
				tt.setup(localRepo)
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
