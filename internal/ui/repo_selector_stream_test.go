package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/git-fire/git-fire/internal/git"
)

func TestRepoSelectorModelStream_scanCountsByRegistryNew(t *testing.T) {
	ch := make(chan git.Repository)
	m := NewRepoSelectorModelStream(ch, nil, false, false, nil, "", nil, "", "")

	var model tea.Model = m
	model, _ = model.Update(repoDiscoveredMsg(git.Repository{
		Path:                 "/a",
		Name:                 "a",
		IsNewRegistryEntry:   true,
	}))
	model, _ = model.Update(repoDiscoveredMsg(git.Repository{
		Path:                 "/b",
		Name:                 "b",
		IsNewRegistryEntry:   false,
	}))
	rm := model.(RepoSelectorModel)
	if rm.scanNewRegistryCount != 1 || rm.scanKnownRegistryCount != 1 {
		t.Fatalf("new=%d known=%d, want new=1 known=1",
			rm.scanNewRegistryCount, rm.scanKnownRegistryCount)
	}
}
