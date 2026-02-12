package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/TBRX103/git-fire/internal/git"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6600")).
			Background(lipgloss.Color("#1A1A1A")).
			Padding(0, 2).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6600")).
			Padding(1, 2)
)

// ASCII fire frames for animation
var fireFrames = []string{
	`     (  )   (   )  )
      ) (   )  (  (
      ( )  (    ) )`,

	`    (  )  (   )  )
     )  (  )  (  (
     (  )  (   ) )`,

	`    )  (  )  (  )
     (  )  (  ) (
     )  (  )  ( )`,
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// RepoSelectorModel is the Bubble Tea model for selecting repositories
type RepoSelectorModel struct {
	repos      []git.Repository
	cursor     int
	selected   map[int]bool
	quitting   bool
	confirmed  bool
	frameIndex int // For fire animation
}

// NewRepoSelectorModel creates a new repo selector
func NewRepoSelectorModel(repos []git.Repository) RepoSelectorModel {
	// Initialize all repos as selected by default
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	return RepoSelectorModel{
		repos:    repos,
		cursor:   0,
		selected: selected,
	}
}

func (m RepoSelectorModel) Init() tea.Cmd {
	return tickCmd()
}

func (m RepoSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Advance animation frame
		m.frameIndex = (m.frameIndex + 1) % len(fireFrames)
		return m, tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			// Confirm selections
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
			// Toggle selection
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.repos[m.cursor].Selected = m.selected[m.cursor]

		case "m":
			// Cycle through modes
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
			// Select all
			for i := range m.repos {
				m.selected[i] = true
				m.repos[i].Selected = true
			}

		case "n":
			// Select none
			for i := range m.repos {
				m.selected[i] = false
				m.repos[i].Selected = false
			}
		}
	}

	return m, nil
}

func (m RepoSelectorModel) View() string {
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

	// Animated fire at top
	fireStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))
	s.WriteString(fireStyle.Render(fireFrames[m.frameIndex]))
	s.WriteString("\n\n")

	// Title
	s.WriteString(titleStyle.Render("                🔥 GIT FIRE - SELECT REPOSITORIES 🔥                "))
	s.WriteString("\n\n")

	// Repository list
	for i, repo := range m.repos {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := "[ ]"
		style := unselectedStyle
		if m.selected[i] {
			checked = "[✓]"
			style = selectedStyle
		}

		// Repo line: cursor, checkbox, name, mode, status
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
	help := helpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  space  Toggle selection  |  m  Change mode\n" +
			"  a  Select all  |  n  Select none  |  enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected",
	)
	s.WriteString(help)

	// Wrap everything in a box
	content := s.String()
	return boxStyle.Render(content)
}

// GetSelectedRepos returns the selected repositories
func (m RepoSelectorModel) GetSelectedRepos() []git.Repository {
	selected := make([]git.Repository, 0)
	for i, repo := range m.repos {
		if m.selected[i] {
			selected = append(selected, repo)
		}
	}
	return selected
}

// RunRepoSelector runs the interactive repo selector and returns selected repos
func RunRepoSelector(repos []git.Repository) ([]git.Repository, error) {
	model := NewRepoSelectorModel(repos)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	// Type assert back to our model
	m := finalModel.(RepoSelectorModel)

	if !m.confirmed {
		return nil, fmt.Errorf("cancelled")
	}

	return m.GetSelectedRepos(), nil
}
