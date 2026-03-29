package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/registry"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Simple styles for lite mode
var (
	liteBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6600")).
			Padding(1, 2)

	liteTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			MarginBottom(1)

	liteSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true)

	liteUnselectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	liteHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)
)

// RepoSelectorLiteModel is the simple, non-animated version
type RepoSelectorLiteModel struct {
	repos          []git.Repository
	cursor         int
	ignoredCursor  int
	view           repoSelectorView
	ignoredEntries []registry.RegistryEntry
	selected       map[int]bool
	quitting       bool
	confirmed      bool
	reg            *registry.Registry
	regPath        string
	lastErr        error
}

// NewRepoSelectorLiteModel creates a new lite repo selector
func NewRepoSelectorLiteModel(repos []git.Repository, reg *registry.Registry, regPath string) RepoSelectorLiteModel {
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	return RepoSelectorLiteModel{
		repos:    repos,
		cursor:   0,
		selected: selected,
		reg:      reg,
		regPath:  regPath,
	}
}

func (m RepoSelectorLiteModel) Init() tea.Cmd {
	return nil
}

func (m RepoSelectorLiteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "i":
			m.lastErr = nil
			if m.view == repoViewIgnored {
				m.view = repoViewMain
			} else {
				m.view = repoViewIgnored
				m.ignoredEntries = IgnoredRegistryEntries(m.reg)
				m.ignoredCursor = clampSelectorCursor(m.ignoredCursor, len(m.ignoredEntries))
			}
			return m, nil

		case "enter":
			if m.view == repoViewIgnored {
				m = m.restoreIgnoredAtCursorLite()
				return m, nil
			}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit

		case "u":
			if m.view == repoViewIgnored {
				m = m.restoreIgnoredAtCursorLite()
				return m, nil
			}

		case "up", "k":
			if m.view == repoViewIgnored {
				if m.ignoredCursor > 0 {
					m.ignoredCursor--
				}
			} else if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.view == repoViewIgnored {
				if m.ignoredCursor < len(m.ignoredEntries)-1 {
					m.ignoredCursor++
				}
			} else if m.cursor < len(m.repos)-1 {
				m.cursor++
			}

		case " ":
			if m.view == repoViewIgnored || len(m.repos) == 0 {
				return m, nil
			}
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.repos[m.cursor].Selected = m.selected[m.cursor]

		case "m":
			if m.view == repoViewIgnored || len(m.repos) == 0 {
				return m, nil
			}
			repo := &m.repos[m.cursor]
			switch repo.Mode {
			case git.ModeLeaveUntouched:
				repo.Mode = git.ModePushKnownBranches
			case git.ModePushKnownBranches:
				repo.Mode = git.ModePushAll
			case git.ModePushAll:
				repo.Mode = git.ModeLeaveUntouched
			}
			m.lastErr = m.persistMode(repo.Path, repo.Mode)

		case "x":
			if m.view == repoViewIgnored || len(m.repos) == 0 {
				return m, nil
			}
			repo := m.repos[m.cursor]
			m.lastErr = m.persistIgnore(repo.Path)
			m.repos = append(m.repos[:m.cursor], m.repos[m.cursor+1:]...)
			newSelected := make(map[int]bool)
			for i := range m.repos {
				oldIdx := i
				if i >= m.cursor {
					oldIdx = i + 1
				}
				newSelected[i] = m.selected[oldIdx]
			}
			m.selected = newSelected
			if m.cursor >= len(m.repos) && m.cursor > 0 {
				m.cursor--
			}

		case "a":
			if m.view == repoViewIgnored {
				return m, nil
			}
			for i := range m.repos {
				m.selected[i] = true
				m.repos[i].Selected = true
			}

		case "n":
			if m.view == repoViewIgnored {
				return m, nil
			}
			for i := range m.repos {
				m.selected[i] = false
				m.repos[i].Selected = false
			}
		}
	}

	return m, nil
}

func (m RepoSelectorLiteModel) View() string {
	if m.quitting {
		if m.confirmed {
			selectedCount := 0
			for _, sel := range m.selected {
				if sel {
					selectedCount++
				}
			}
			return fmt.Sprintf("\n✅ Selected %d repositories for backup\n\n", selectedCount)
		}
		return "\n❌ Cancelled\n\n"
	}

	if m.view == repoViewIgnored {
		return m.viewIgnoredLite()
	}

	var s strings.Builder

	// Simple title with emoji
	s.WriteString(liteTitleStyle.Render("🔥 GIT FIRE - SELECT REPOSITORIES"))
	s.WriteString("\n\n")

	// Repository list
	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := "[ ]"
		style := liteUnselectedStyle
		if m.selected[i] {
			checked = "[✓]"
			style = liteSelectedStyle
		}

		dirtyIndicator := ""
		if repo.IsDirty {
			dirtyIndicator = "💥"
		}

		remotesInfo := fmt.Sprintf("(%d remotes)", len(repo.Remotes))
		if len(repo.Remotes) == 0 {
			remotesInfo = "(no remotes!)"
		}

		displayPath := AbbreviateUserHome(repo.Path)
		line := fmt.Sprintf("%s %s %s  [%s] %s %s",
			cursor,
			checked,
			style.Render(displayPath),
			repo.Mode.String(),
			remotesInfo,
			dirtyIndicator,
		)

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Help text
	help := liteHelpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  space  Toggle selection  |  m  Change mode  |  x  Ignore repo\n" +
			"  a  Select all  |  n  Select none  |  i  View ignored  |  enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected\n\n" +
			"💡 Tip: Run with --fire flag for animated fire background!",
	)
	s.WriteString(help)

	if m.lastErr != nil {
		s.WriteString(fmt.Sprintf("\n\n⚠️  registry save failed: %v", m.lastErr))
	}

	return liteBoxStyle.Render(s.String())
}

func (m RepoSelectorLiteModel) viewIgnoredLite() string {
	var s strings.Builder
	s.WriteString(liteTitleStyle.Render("🔥 IGNORED REPOSITORIES (NOT TRACKED)"))
	s.WriteString("\n\n")
	if len(m.ignoredEntries) == 0 {
		s.WriteString(liteUnselectedStyle.Render("No ignored repositories."))
		s.WriteString("\n")
	} else {
		for i, e := range m.ignoredEntries {
			cursor := " "
			if m.ignoredCursor == i {
				cursor = ">"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cursor, AbbreviateUserHome(e.Path)))
		}
	}
	help := liteHelpStyle.Render(
		"\n" +
			"Excluded from backup. Restore with enter or u.\n" +
			"↑/k ↓/j  Navigate  |  enter / u  Track again  |  i  Back  |  q  Quit\n",
	)
	s.WriteString(help)
	if m.lastErr != nil {
		s.WriteString(fmt.Sprintf("\n\n⚠️  %v", m.lastErr))
	}
	return liteBoxStyle.Render(s.String())
}

func (m RepoSelectorLiteModel) restoreIgnoredAtCursorLite() RepoSelectorLiteModel {
	m.lastErr = nil
	if m.reg == nil || m.regPath == "" || len(m.ignoredEntries) == 0 {
		return m
	}
	if m.ignoredCursor < 0 || m.ignoredCursor >= len(m.ignoredEntries) {
		return m
	}
	entry := m.ignoredEntries[m.ignoredCursor]
	absPath, err := filepath.Abs(entry.Path)
	if err != nil {
		m.lastErr = err
		return m
	}
	if !m.reg.SetStatus(entry.Path, registry.StatusActive) && !m.reg.SetStatus(absPath, registry.StatusActive) {
		name := entry.Name
		if name == "" {
			name = filepath.Base(absPath)
		}
		m.reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   name,
			Status: registry.StatusActive,
			Mode:   entry.Mode,
		})
	}
	if err := registry.Save(m.reg, m.regPath); err != nil {
		m.lastErr = err
		return m
	}
	if repo, aerr := git.AnalyzeRepository(absPath); aerr == nil {
		if entry.Mode != "" {
			repo.Mode = git.ParseMode(entry.Mode)
		}
		repo.Selected = true
		if !repoPathInRepos(m.repos, absPath) {
			idx := len(m.repos)
			m.repos = append(m.repos, repo)
			m.selected[idx] = true
		}
	}
	m.ignoredEntries = IgnoredRegistryEntries(m.reg)
	m.ignoredCursor = clampSelectorCursor(m.ignoredCursor, len(m.ignoredEntries))
	return m
}

func (m RepoSelectorLiteModel) persistMode(repoPath string, mode git.RepoMode) error {
	if m.reg == nil || m.regPath == "" {
		return nil
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	entry := m.reg.FindByPath(absPath)
	if entry != nil {
		entry.Mode = mode.String()
	} else {
		m.reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   filepath.Base(absPath),
			Status: registry.StatusActive,
			Mode:   mode.String(),
		})
	}
	return registry.Save(m.reg, m.regPath)
}

func (m RepoSelectorLiteModel) persistIgnore(repoPath string) error {
	if m.reg == nil || m.regPath == "" {
		return nil
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	if !m.reg.SetStatus(absPath, registry.StatusIgnored) {
		m.reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   filepath.Base(absPath),
			Status: registry.StatusIgnored,
		})
	}
	return registry.Save(m.reg, m.regPath)
}

// GetSelectedRepos returns the selected repositories
func (m RepoSelectorLiteModel) GetSelectedRepos() []git.Repository {
	selected := make([]git.Repository, 0)
	for i, repo := range m.repos {
		if m.selected[i] {
			selected = append(selected, repo)
		}
	}
	return selected
}

// RunRepoSelectorLite runs the lite (non-animated) interactive repo selector.
func RunRepoSelectorLite(repos []git.Repository, reg *registry.Registry, regPath string) ([]git.Repository, error) {
	model := NewRepoSelectorLiteModel(repos, reg, regPath)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(RepoSelectorLiteModel)
	if !m.confirmed {
		return nil, ErrCancelled
	}
	return m.GetSelectedRepos(), nil
}
