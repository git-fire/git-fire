package ui

import (
	"path/filepath"
	"testing"

	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/registry"
)

func TestIgnoredRegistryEntries_SortedAndFiltered(t *testing.T) {
	reg := &registry.Registry{
		Repos: []registry.RegistryEntry{
			{Path: "/z/ignored", Status: registry.StatusIgnored},
			{Path: "/a/active", Status: registry.StatusActive},
			{Path: "/m/ignored", Status: registry.StatusIgnored},
		},
	}
	got := IgnoredRegistryEntries(reg)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Path != "/m/ignored" || got[1].Path != "/z/ignored" {
		t.Errorf("order = %#v, want /m then /z", []string{got[0].Path, got[1].Path})
	}
}

func TestIgnoredRegistryEntries_NilRegistry(t *testing.T) {
	if got := IgnoredRegistryEntries(nil); got != nil {
		t.Errorf("want nil, got %#v", got)
	}
}

func TestRepoPathInRepos(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "repo")
	repos := []git.Repository{{Path: p}}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	if !repoPathInRepos(repos, abs) {
		t.Error("expected match for same path")
	}
	if repoPathInRepos(repos, filepath.Join(tmp, "other")) {
		t.Error("unexpected match")
	}
	if repoPathInRepos(nil, abs) {
		t.Error("nil repos should not match")
	}
}
