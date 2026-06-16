package ui

import (
	"path/filepath"
	"testing"

	"github.com/git-fire/git-harness/git"
	"github.com/git-fire/git-fire/internal/registry"
)

func TestSelectorPersistMode_UpdateByPath(t *testing.T) {
	tmp := t.TempDir()
	regPath := filepath.Join(tmp, "repos.toml")
	reg := &registry.Registry{}
	reg.Upsert(registry.RegistryEntry{
		Path:   "/repos/a",
		Name:   "a",
		Status: registry.StatusActive,
		Mode:   "push-known-branches",
	})

	if err := selectorPersistMode(reg, regPath, "/repos/a", git.ModePushAll); err != nil {
		t.Fatalf("selectorPersistMode: %v", err)
	}
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	e := loaded.FindByPath("/repos/a")
	if e == nil || e.Mode != git.ModePushAll.String() {
		t.Fatalf("mode after persist = %v, want push-all", e)
	}
}

func TestSelectorPersistMode_UpsertWhenMissing(t *testing.T) {
	tmp := t.TempDir()
	regPath := filepath.Join(tmp, "repos.toml")
	reg := &registry.Registry{}

	repoDir := filepath.Join(tmp, "newrepo")
	if err := selectorPersistMode(reg, regPath, repoDir, git.ModePushCurrentBranch); err != nil {
		t.Fatalf("selectorPersistMode: %v", err)
	}
	abs, _ := filepath.Abs(repoDir)
	loaded, err := registry.Load(regPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	e := loaded.FindByPath(abs)
	if e == nil || e.Mode != git.ModePushCurrentBranch.String() {
		t.Fatalf("upserted entry mode = %v, want push-current-branch", e)
	}
}
