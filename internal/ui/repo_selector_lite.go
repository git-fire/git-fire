package ui

import (
	"fmt"
	"strings"

	"github.com/TBRX103/git-fire/internal/git"
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
	repos     []git.Repository
	cursor    int
	selected  map[int]bool
	quitting  bool
	confirmed bool
}

// NewRepoSelectorLiteModel creates a new lite repo selector
func NewRepoSelectorLiteModel(repos []git.Repository) RepoSelectorLiteModel {
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	return RepoSelectorLiteModel{
		repos:    repos,
		cursor:   0,
		selected: selected,
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

		case "enter":
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.repos)-1 {
				m.cursor++
			}

		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.repos[m.cursor].Selected = m.selected[m.cursor]

		case "m":
			repo := &m.repos[m.cursor]
			switch repo.Mode {
			case git.ModeLeaveUntouched:
				repo.Mode = git.ModePushKnownBranches
			case git.ModePushKnownBranches:
				repo.Mode = git.ModePushAll
			case git.ModePushAll:
				repo.Mode = git.ModeLeaveUntouched
			}

		case "a":
			for i := range m.repos {
				m.selected[i] = true
				m.repos[i].Selected = true
			}

		case "n":
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

		line := fmt.Sprintf("%s %s %s  [%s] %s %s",
			cursor,
			checked,
			style.Render(repo.Name),
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
			"  ↑/k, ↓/j  Navigate  |  space  Toggle selection  |  m  Change mode\n" +
			"  a  Select all  |  n  Select none  |  enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected\n\n" +
			"💡 Tip: Run with --fire flag for animated fire background!",
	)
	s.WriteString(help)

	return liteBoxStyle.Render(s.String())
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
