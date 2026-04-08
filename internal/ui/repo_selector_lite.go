package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
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

	liteScrollHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFD166")).
				Bold(true)
)

// RepoSelectorLiteModel is the simple, non-animated version
type RepoSelectorLiteModel struct {
	repos            []git.Repository
	cursor           int
	ignoredCursor    int
	view             repoSelectorView
	ignoredEntries   []registry.RegistryEntry
	selected         map[int]bool
	quitting         bool
	confirmed        bool
	reg              *registry.Registry
	regPath          string
	lastErr          error
	windowWidth      int
	pathScrollOffset int // manual path scroll offset for the focused repo row
}

// NewRepoSelectorLiteModel creates a new lite repo selector
func NewRepoSelectorLiteModel(repos []git.Repository, reg *registry.Registry, regPath string) RepoSelectorLiteModel {
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	return RepoSelectorLiteModel{
		repos:       repos,
		cursor:      0,
		selected:    selected,
		reg:         reg,
		regPath:     regPath,
		windowWidth: 80,
	}
}

func (m RepoSelectorLiteModel) Init() tea.Cmd {
	return nil
}

func (m RepoSelectorLiteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m = m.withClampedPathScroll()
		return m, nil

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
				m = m.withResetPathScroll()
			}

		case "down", "j":
			if m.view == repoViewIgnored {
				if m.ignoredCursor < len(m.ignoredEntries)-1 {
					m.ignoredCursor++
				}
			} else if m.cursor < len(m.repos)-1 {
				m.cursor++
				m = m.withResetPathScroll()
			}

		case "left":
			if m.view == repoViewMain && m.pathScrollOffset > 0 {
				m.pathScrollOffset--
			}

		case "right":
			if m.view == repoViewMain && len(m.repos) > 0 && m.cursor < len(m.repos) {
				repo := m.repos[m.cursor]
				parentPath := AbbreviateUserHome(filepath.Dir(repo.Path))
				pathLen := len([]rune(parentPath))
				pWidth := PathWidthFor(m.windowWidth, repo)
				maxOffset := pathLen - pWidth
				if maxOffset > 0 && m.pathScrollOffset < maxOffset {
					m.pathScrollOffset++
				}
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
				repo.Mode = git.ModePushCurrentBranch
			case git.ModePushCurrentBranch:
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
			m = m.withResetPathScroll()

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
			dirtyIndicator = " 💥"
		}

		remotesInfo := fmt.Sprintf("(%d remotes)", len(repo.Remotes))
		if len(repo.Remotes) == 0 {
			remotesInfo = "(no remotes!)"
		}

		parentPath := AbbreviateUserHome(filepath.Dir(repo.Path))
		pWidth := PathWidthFor(m.windowWidth, repo)
		scrollOff := 0
		if m.cursor == i {
			scrollOff = m.pathScrollOffset
		}
		visible, hasLeft, hasRight := TruncatePath(parentPath, pWidth, scrollOff)
		leftInd, rightInd := " ", " "
		if hasLeft {
			leftInd = "‹"
		}
		if hasRight {
			rightInd = "›"
		}

		scrollHint := ""
		if m.cursor == i && (hasLeft || hasRight) {
			scrollHint = "  " + liteScrollHintStyle.Render("<< SCROLL PATH >>")
		}

		line := fmt.Sprintf("%s %s %s (%s%s%s)  [%s] %s%s%s",
			cursor,
			checked,
			style.Render(repo.Name),
			leftInd, visible, rightInd,
			repo.Mode.String(),
			remotesInfo,
			dirtyIndicator,
			scrollHint,
		)

		s.WriteString(line)
		s.WriteString("\n")
	}

	// Help text
	help := liteHelpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  ←/→  Scroll path when << SCROLL PATH >> shows  |  space  Toggle selection\n" +
			"  m  Change mode  |  x  Ignore  |  a  Select all  |  n  Select none\n" +
			"  i  View ignored  |  enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected  |  ‹›  = path scrollable\n\n" +
			"💡 Tip: Run with --fire flag for animated fire background!",
	)
	s.WriteString(help)

	if m.lastErr != nil {
		fmt.Fprintf(&s, "\n\n⚠️  registry save failed: %v", m.lastErr)
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
			fmt.Fprintf(&s, "%s %s\n", cursor, AbbreviateUserHome(e.Path))
		}
	}
	help := liteHelpStyle.Render(
		"\n" +
			"Excluded from backup. Restore with enter or u.\n" +
			"↑/k ↓/j  Navigate  |  enter / u  Track again  |  i  Back  |  q  Quit\n",
	)
	s.WriteString(help)
	if m.lastErr != nil {
		fmt.Fprintf(&s, "\n\n⚠️  %v", m.lastErr)
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

// withResetPathScroll returns m with path-scroll offset zeroed.
// Call whenever the focused repo changes (up/down/x/…).
func (m RepoSelectorLiteModel) withResetPathScroll() RepoSelectorLiteModel {
	m.pathScrollOffset = 0
	return m
}

// withClampedPathScroll returns m with pathScrollOffset clamped to the valid
// range for the current focused repo and window width.
// Call whenever windowWidth changes.
func (m RepoSelectorLiteModel) withClampedPathScroll() RepoSelectorLiteModel {
	if len(m.repos) == 0 || m.cursor >= len(m.repos) {
		m.pathScrollOffset = 0
		return m
	}
	repo := m.repos[m.cursor]
	parentPath := AbbreviateUserHome(filepath.Dir(repo.Path))
	pathLen := len([]rune(parentPath))
	pWidth := PathWidthFor(m.windowWidth, repo)
	maxOffset := pathLen - pWidth
	if maxOffset <= 0 {
		m.pathScrollOffset = 0
	} else if m.pathScrollOffset > maxOffset {
		m.pathScrollOffset = maxOffset
	}
	return m
}

func (m RepoSelectorLiteModel) persistMode(repoPath string, mode git.RepoMode) error {
	return selectorPersistMode(m.reg, m.regPath, repoPath, mode)
}

func (m RepoSelectorLiteModel) persistIgnore(repoPath string) error {
	return selectorPersistIgnore(m.reg, m.regPath, repoPath)
}

// GetSelectedRepos returns the selected repositories
func (m RepoSelectorLiteModel) GetSelectedRepos() []git.Repository {
	return selectorGetSelected(m.repos, m.selected)
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
