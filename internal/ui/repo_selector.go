package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TBRX103/git-fire/internal/git"
	"github.com/TBRX103/git-fire/internal/registry"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned by RunRepoSelector when the user cancels the TUI.
var ErrCancelled = errors.New("cancelled")

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

type repoSelectorView int

const (
	repoViewMain repoSelectorView = iota
	repoViewIgnored
)

func tickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// RepoSelectorModel is the Bubble Tea model for selecting repositories
type RepoSelectorModel struct {
	repos          []git.Repository
	cursor         int // main list position
	ignoredCursor  int // ignored list position
	view           repoSelectorView
	ignoredEntries []registry.RegistryEntry
	selected       map[int]bool
	quitting       bool
	confirmed      bool
	frameIndex     int             // For fire animation
	fireBg         *FireBackground // Animated fire background
	spinner        spinner.Model   // Loading spinner
	windowWidth    int
	windowHeight   int
	reg            *registry.Registry // persistent registry for write-through
	regPath        string             // path to registry file
}

// NewRepoSelectorModel creates a new repo selector
func NewRepoSelectorModel(repos []git.Repository, reg *registry.Registry, regPath string) RepoSelectorModel {
	// Initialize all repos as selected by default
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600"))

	// Initialize fire background
	fireBg := NewFireBackground(70, 5)

	return RepoSelectorModel{
		repos:        repos,
		cursor:       0,
		selected:     selected,
		fireBg:       fireBg,
		spinner:      s,
		windowWidth:  80,
		windowHeight: 40,
		reg:          reg,
		regPath:      regPath,
	}
}

func (m RepoSelectorModel) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.spinner.Tick)
}

func (m RepoSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		// Resize fire background
		m.fireBg = NewFireBackground(min(msg.Width-4, 70), 5)

	case tickMsg:
		// Advance animation frame
		m.frameIndex = (m.frameIndex + 1) % len(fireFrames)
		// Update fire background
		m.fireBg.Update()
		return m, tickCmd()

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "i":
			if m.view == repoViewIgnored {
				m.view = repoViewMain
			} else {
				m.view = repoViewIgnored
				m.ignoredEntries = IgnoredRegistryEntries(m.reg)
				m.ignoredCursor = clampSelectorCursor(m.ignoredCursor, len(m.ignoredEntries))
			}
			return m, tea.Batch(cmds...)

		case "enter":
			if m.view == repoViewIgnored {
				m = m.restoreIgnoredAtCursor()
				return m, tea.Batch(cmds...)
			}
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit

		case "u":
			if m.view == repoViewIgnored {
				m = m.restoreIgnoredAtCursor()
				return m, tea.Batch(cmds...)
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
				break
			}
			m.selected[m.cursor] = !m.selected[m.cursor]
			m.repos[m.cursor].Selected = m.selected[m.cursor]

		case "m":
			if m.view == repoViewIgnored || len(m.repos) == 0 {
				break
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
			m.persistMode(repo.Path, repo.Mode)

		case "x":
			if m.view == repoViewIgnored || len(m.repos) == 0 {
				break
			}
			repo := m.repos[m.cursor]
			m.persistIgnore(repo.Path)
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
				break
			}
			for i := range m.repos {
				m.selected[i] = true
				m.repos[i].Selected = true
			}

		case "n":
			if m.view == repoViewIgnored {
				break
			}
			for i := range m.repos {
				m.selected[i] = false
				m.repos[i].Selected = false
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampSelectorCursor(cursor, n int) int {
	if n <= 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= n {
		return n - 1
	}
	return cursor
}

func (m RepoSelectorModel) restoreIgnoredAtCursor() RepoSelectorModel {
	if m.reg == nil || m.regPath == "" || len(m.ignoredEntries) == 0 {
		return m
	}
	if m.ignoredCursor < 0 || m.ignoredCursor >= len(m.ignoredEntries) {
		return m
	}
	entry := m.ignoredEntries[m.ignoredCursor]
	absPath, err := filepath.Abs(entry.Path)
	if err != nil {
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
	_ = registry.Save(m.reg, m.regPath)

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

	if m.view == repoViewIgnored {
		return m.viewIgnoredMain()
	}

	var s strings.Builder

	// Animated fire background at top
	s.WriteString(m.fireBg.Render())
	s.WriteString("\n")

	// Animated fire wave separator
	s.WriteString(RenderFireWave(min(m.windowWidth-4, 70), m.frameIndex))
	s.WriteString("\n\n")

	// Title with gradient
	titleText := "🔥 GIT FIRE - SELECT REPOSITORIES 🔥"
	titleGradient := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff4500")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 2)
	s.WriteString(titleGradient.Render(titleText))
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
	help := helpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  space  Toggle selection  |  m  Change mode  |  x  Ignore repo\n" +
			"  a  Select all  |  n  Select none  |  i  View ignored  |  enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected",
	)
	s.WriteString(help)

	// Wrap everything in a box
	content := s.String()
	return boxStyle.Render(content)
}

func (m RepoSelectorModel) viewIgnoredMain() string {
	var s strings.Builder
	s.WriteString(m.fireBg.Render())
	s.WriteString("\n")
	s.WriteString(RenderFireWave(min(m.windowWidth-4, 70), m.frameIndex))
	s.WriteString("\n\n")

	titleGradient := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff4500")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 2)
	s.WriteString(titleGradient.Render("🔥 IGNORED REPOSITORIES (NOT TRACKED) 🔥"))
	s.WriteString("\n\n")

	if len(m.ignoredEntries) == 0 {
		s.WriteString(unselectedStyle.Render("No ignored repositories."))
		s.WriteString("\n")
	} else {
		for i, e := range m.ignoredEntries {
			cursor := " "
			if m.ignoredCursor == i {
				cursor = ">"
			}
			line := fmt.Sprintf("%s %s", cursor, AbbreviateUserHome(e.Path))
			s.WriteString(line)
			s.WriteString("\n")
		}
	}

	help := helpStyle.Render(
		"\n" +
			"These repos are excluded from backup. Restore tracking with enter or u.\n" +
			"Controls:  ↑/k, ↓/j  Navigate  |  enter / u  Track again  |  i  Back to main  |  q  Quit\n",
	)
	s.WriteString(help)
	return boxStyle.Render(s.String())
}

// persistMode writes the repo's current mode to the registry synchronously.
// Errors are silently ignored — this is best-effort during an emergency.
func (m RepoSelectorModel) persistMode(repoPath string, mode git.RepoMode) {
	if m.reg == nil || m.regPath == "" {
		return
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return
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
	_ = registry.Save(m.reg, m.regPath)
}

// persistIgnore marks the repo as ignored in the registry synchronously.
func (m RepoSelectorModel) persistIgnore(repoPath string) {
	if m.reg == nil || m.regPath == "" {
		return
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return
	}
	if !m.reg.SetStatus(absPath, registry.StatusIgnored) {
		m.reg.Upsert(registry.RegistryEntry{
			Path:   absPath,
			Name:   filepath.Base(absPath),
			Status: registry.StatusIgnored,
		})
	}
	_ = registry.Save(m.reg, m.regPath)
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

// RunRepoSelector runs the interactive repo selector and returns selected repos.
// reg and regPath are used for write-through persistence of mode changes and
// ignored repos; pass nil/empty to disable persistence.
func RunRepoSelector(repos []git.Repository, reg *registry.Registry, regPath string) ([]git.Repository, error) {
	model := NewRepoSelectorModel(repos, reg, regPath)
	p := tea.NewProgram(model)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	// Type assert back to our model
	m := finalModel.(RepoSelectorModel)

	if !m.confirmed {
		return nil, ErrCancelled
	}

	return m.GetSelectedRepos(), nil
}
