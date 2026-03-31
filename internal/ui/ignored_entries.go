package ui

import (
	"path/filepath"
	"sort"

	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
)

// IgnoredRegistryEntries returns registry entries with status ignored, sorted by path.
func IgnoredRegistryEntries(reg *registry.Registry) []registry.RegistryEntry {
	if reg == nil {
		return nil
	}
	out := make([]registry.RegistryEntry, 0)
	for _, e := range reg.Repos {
		if e.Status == registry.StatusIgnored {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Path < out[j].Path
	})
	return out
}

func repoPathInRepos(repos []git.Repository, absPath string) bool {
	if absPath == "" {
		return false
	}
	for _, r := range repos {
		ra, err := filepath.Abs(r.Path)
		if err != nil {
			continue
		}
		if ra == absPath {
			return true
		}
	}
	return false
}
