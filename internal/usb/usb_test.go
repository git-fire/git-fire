package usb

import (
	"os"
	"path/filepath"
	"testing"

	testutil "github.com/git-fire/git-testkit"
)

func TestEnsureVolumeConfig_CreateAndRead(t *testing.T) {
	root := t.TempDir()

	cfg, err := EnsureVolumeConfig(root, EnsureOptions{
		DefaultStrategy: StrategyMirror,
		CreateIfMissing: true,
	})
	if err != nil {
		t.Fatalf("EnsureVolumeConfig(create) error: %v", err)
	}
	if cfg.LayoutDir != DefaultRepoLayoutDir {
		t.Fatalf("unexpected layout dir: %s", cfg.LayoutDir)
	}
	if cfg.Strategy != StrategyMirror {
		t.Fatalf("unexpected strategy: %s", cfg.Strategy)
	}

	cfg2, err := EnsureVolumeConfig(root, EnsureOptions{
		DefaultStrategy: StrategyClone,
		CreateIfMissing: false,
	})
	if err != nil {
		t.Fatalf("EnsureVolumeConfig(read) error: %v", err)
	}
	if cfg2.Strategy != StrategyMirror {
		t.Fatalf("expected existing strategy to persist, got: %s", cfg2.Strategy)
	}
}

func TestEnsureVolumeConfig_RequireMarker(t *testing.T) {
	root := t.TempDir()
	if _, err := EnsureVolumeConfig(root, EnsureOptions{CreateIfMissing: false}); err == nil {
		t.Fatal("expected missing marker error")
	}
}

func TestStableRepoName_Deterministic(t *testing.T) {
	repoPath := "/tmp/example/repo"
	nameA := StableRepoName(repoPath, "Repo Name")
	nameB := StableRepoName(repoPath, "Repo Name")
	if nameA != nameB {
		t.Fatalf("expected deterministic name, got %s vs %s", nameA, nameB)
	}
	if nameA == "Repo Name" {
		t.Fatalf("expected sanitized+hashed name, got %s", nameA)
	}
}

func TestSyncMirrorRepo(t *testing.T) {
	source := testutil.CreateTestRepo(t, testutil.RepoOptions{
		Name: "source",
		Files: map[string]string{
			"README.md": "hello",
		},
	})
	destRoot := t.TempDir()
	destBare := filepath.Join(destRoot, "mirror.git")

	if err := SyncMirrorRepo(source, destBare); err != nil {
		t.Fatalf("SyncMirrorRepo() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destBare, "HEAD")); err != nil {
		t.Fatalf("expected bare repo HEAD, got err: %v", err)
	}

	// Update source and ensure second sync succeeds.
	newFile := filepath.Join(source, "more.txt")
	if err := os.WriteFile(newFile, []byte("more"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	testutil.RunGitCmd(t, source, "add", "more.txt")
	testutil.RunGitCmd(t, source, "commit", "-m", "more")

	if err := SyncMirrorRepo(source, destBare); err != nil {
		t.Fatalf("SyncMirrorRepo(second) error: %v", err)
	}
}
