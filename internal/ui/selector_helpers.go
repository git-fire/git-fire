package ui

import (
	"path/filepath"

	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
)

// selectorPersistMode writes a repo's mode to the registry.
// Returns an error so callers that care can handle it; errors indicate
// either a bad path or a failed registry save.
func selectorPersistMode(reg *registry.Registry, regPath, repoPath string, mode git.RepoMode) error {
	if reg == nil || regPath == "" {
		return nil
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	if !reg.UpdateByPath(absPath, func(e *registry.RegistryEntry) {
		e.Mode = mode.String()
	}) {
		reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   filepath.Base(absPath),
			Status: registry.StatusActive,
			Mode:   mode.String(),
		})
	}
	return registry.Save(reg, regPath)
}

// selectorPersistIgnore marks a repo as ignored in the registry.
func selectorPersistIgnore(reg *registry.Registry, regPath, repoPath string) error {
	if reg == nil || regPath == "" {
		return nil
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	if !reg.SetStatus(absPath, registry.StatusIgnored) {
		reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   filepath.Base(absPath),
			Status: registry.StatusIgnored,
		})
	}
	return registry.Save(reg, regPath)
}

// selectorGetSelected returns the repos at indices where selected[i] is true.
func selectorGetSelected(repos []git.Repository, selected map[int]bool) []git.Repository {
	out := make([]git.Repository, 0)
	for i, repo := range repos {
		if selected[i] {
			out = append(out, repo)
		}
	}
	return out
}
