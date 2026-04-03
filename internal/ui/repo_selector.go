package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/git"
	"github.com/git-fire/git-fire/internal/registry"
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

	scrollHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD166")).
			Bold(true)

	viewportWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFAA00")).
				Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6600")).
			Padding(1, 2)
)

func init() {
	applyColorProfile(config.UIColorProfileClassic)
}

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
type quoteTickMsg time.Time

// pathScrollMsg drives path-marquee advancement at a fixed cadence that is
// independent of the fire animation speed.
type pathScrollMsg time.Time

// repoDiscoveredMsg is sent when a new repo arrives via the scan channel.
type repoDiscoveredMsg git.Repository

// scanProgressMsg carries the path the scanner is currently visiting.
type scanProgressMsg string

// repoChanDoneMsg is sent when the repo scan channel is closed.
type repoChanDoneMsg struct{}

// progressChanDoneMsg is sent when the folder-progress channel is closed.
type progressChanDoneMsg struct{}

type repoSelectorView int

const (
	repoViewMain repoSelectorView = iota
	repoViewIgnored
	repoViewConfig
)

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func quoteTickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return quoteTickMsg(t)
	})
}

// pathScrollCmd schedules the next path-marquee advancement tick.
// 150 ms gives a smooth ~6-7 rune/s scroll regardless of fire speed.
const pathScrollInterval = 150 * time.Millisecond

func pathScrollCmd() tea.Cmd {
	return tea.Tick(pathScrollInterval, func(t time.Time) tea.Msg {
		return pathScrollMsg(t)
	})
}

// waitForRepo listens for the next repo on scanChan. Returns repoDiscoveredMsg
// or repoChanDoneMsg when the channel closes.
func waitForRepo(ch <-chan git.Repository) tea.Cmd {
	return func() tea.Msg {
		repo, ok := <-ch
		if !ok {
			return repoChanDoneMsg{}
		}
		return repoDiscoveredMsg(repo)
	}
}

// waitForProgress listens for the next folder path on progressChan. Returns
// scanProgressMsg or progressChanDoneMsg when the channel closes.
func waitForProgress(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		path, ok := <-ch
		if !ok {
			return progressChanDoneMsg{}
		}
		return scanProgressMsg(path)
	}
}

// RepoSelectorModel is the Bubble Tea model for selecting repositories
type RepoSelectorModel struct {
	repos               []git.Repository
	cursor              int // main list position
	scrollOffset        int // first visible repo index
	ignoredCursor       int // ignored list position
	ignoredScrollOffset int // first visible ignored entry index
	view                repoSelectorView
	ignoredEntries      []registry.RegistryEntry
	selected            map[int]bool
	quitting            bool
	confirmed           bool
	frameIndex          int             // For fire animation
	fireBg              *FireBackground // Animated fire background
	spinner             spinner.Model   // Loading spinner
	windowWidth         int
	windowHeight        int
	reg                 *registry.Registry // persistent registry for write-through
	regPath             string             // path to registry file

	// Path scrolling state for the focused repo row
	pathScrollOffset int // current rune offset into the parent-dir path
	pathScrollDir    int // +1 = scrolling right, -1 = scrolling left
	pathScrollPause  int // path-scroll ticks remaining to pause at each end

	// Streaming scan state (nil channels = batch/static mode)
	scanChan            <-chan git.Repository
	progressChan        <-chan string
	scanDone            bool   // true once scanChan is closed
	progDone            bool   // true once progressChan is closed
	scanDisabled        bool   // disable_scan = true in config OR --no-scan flag
	scanDisabledRunOnly bool   // true when disabled by --no-scan flag (not persisted config)
	scanCurrentPath     string // latest folder the scanner is visiting
	scanNewCount        int    // repos discovered during this TUI session

	// Fire animation toggle (loaded from cfg.UI.ShowFireAnimation; persisted on 'f')
	showFire bool
	fireTick time.Duration

	// Startup quote banner in TUI.
	showStartupQuote     bool
	startupQuoteBehavior string
	startupQuoteInterval time.Duration
	currentStartupQuote  string
	startupQuoteVisible  bool
	quoteTickActive      bool

	// Config menu state
	cfg           *config.Config
	cfgPath       string
	configCursor  int   // selected row in config view
	configSaveErr error // last SaveConfig error; cleared on successful save
}

// NewRepoSelectorModel creates a new repo selector
func NewRepoSelectorModel(repos []git.Repository, reg *registry.Registry, regPath string) RepoSelectorModel {
	applyColorProfile(config.UIColorProfileClassic)
	// Initialize all repos as selected by default
	selected := make(map[int]bool)
	for i := range repos {
		selected[i] = repos[i].Selected
	}

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(activeProfile().boxBorder)

	// Initialize fire background
	fireBg := NewFireBackground(70, 5)

	return RepoSelectorModel{
		repos:                repos,
		cursor:               0,
		selected:             selected,
		fireBg:               fireBg,
		spinner:              s,
		windowWidth:          80,
		windowHeight:         40,
		reg:                  reg,
		regPath:              regPath,
		pathScrollDir:        1,
		showFire:             true,
		fireTick:             time.Duration(config.DefaultUIFireTickMS) * time.Millisecond,
		showStartupQuote:     true,
		startupQuoteBehavior: config.UIQuoteBehaviorRefresh,
		startupQuoteInterval: time.Duration(config.DefaultUIStartupQuoteIntervalSec) * time.Second,
		currentStartupQuote:  randomStartupFireQuote(),
		startupQuoteVisible:  true,
		quoteTickActive:      true,
	}
}

// NewRepoSelectorModelStream creates a model that populates its repo list
// progressively as repos arrive on scanChan. Use RunRepoSelectorStream as the
// entry point; do not call this directly.
func NewRepoSelectorModelStream(
	scanChan <-chan git.Repository,
	progressChan <-chan string,
	scanDisabled bool,
	scanDisabledRunOnly bool,
	cfg *config.Config,
	cfgPath string,
	reg *registry.Registry,
	regPath string,
) RepoSelectorModel {
	profileName := config.UIColorProfileClassic
	if cfg != nil && cfg.UI.ColorProfile != "" {
		profileName = cfg.UI.ColorProfile
	}
	applyColorProfile(profileName)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(activeProfile().boxBorder)

	fireBg := NewFireBackground(70, 5)

	showFire := true
	fireTickMS := config.DefaultUIFireTickMS
	showStartupQuote := true
	startupQuoteBehavior := config.UIQuoteBehaviorRefresh
	startupQuoteIntervalSec := config.DefaultUIStartupQuoteIntervalSec
	if cfg != nil {
		showFire = cfg.UI.ShowFireAnimation
		if cfg.UI.FireTickMS > 0 {
			fireTickMS = cfg.UI.FireTickMS
		}
		showStartupQuote = cfg.UI.ShowStartupQuote
		if cfg.UI.StartupQuoteBehavior != "" {
			startupQuoteBehavior = cfg.UI.StartupQuoteBehavior
		}
		if cfg.UI.StartupQuoteIntervalSec > 0 {
			startupQuoteIntervalSec = cfg.UI.StartupQuoteIntervalSec
		}
	}

	return RepoSelectorModel{
		repos:                nil,
		cursor:               0,
		selected:             make(map[int]bool),
		fireBg:               fireBg,
		spinner:              s,
		windowWidth:          80,
		windowHeight:         40,
		reg:                  reg,
		regPath:              regPath,
		scanChan:             scanChan,
		progressChan:         progressChan,
		scanDone:             scanDisabled, // if scan is disabled there's nothing to wait for
		progDone:             scanDisabled,
		scanDisabled:         scanDisabled,
		scanDisabledRunOnly:  scanDisabledRunOnly,
		cfg:                  cfg,
		cfgPath:              cfgPath,
		showFire:             showFire,
		fireTick:             time.Duration(fireTickMS) * time.Millisecond,
		showStartupQuote:     showStartupQuote,
		startupQuoteBehavior: startupQuoteBehavior,
		startupQuoteInterval: time.Duration(startupQuoteIntervalSec) * time.Second,
		currentStartupQuote:  randomStartupFireQuote(),
		startupQuoteVisible:  showStartupQuote,
		quoteTickActive:      showStartupQuote && startupQuoteIntervalSec > 0,
	}
}

func (m RepoSelectorModel) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd(m.fireTick), pathScrollCmd(), m.spinner.Tick}
	if m.showStartupQuote && m.startupQuoteInterval > 0 {
		cmds = append(cmds, quoteTickCmd(m.startupQuoteInterval))
	}
	if m.scanChan != nil && !m.scanDone {
		cmds = append(cmds, waitForRepo(m.scanChan))
	}
	if m.progressChan != nil && !m.progDone {
		cmds = append(cmds, waitForProgress(m.progressChan))
	}
	return tea.Batch(cmds...)
}

func (m RepoSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// --- Streaming scan messages ---
	case repoDiscoveredMsg:
		repo := git.Repository(msg)
		repo.Selected = true
		idx := len(m.repos)
		m.repos = append(m.repos, repo)
		m.selected[idx] = true
		m.scanNewCount++
		if m.scanChan != nil {
			cmds = append(cmds, waitForRepo(m.scanChan))
		}

	case scanProgressMsg:
		m.scanCurrentPath = string(msg)
		if m.progressChan != nil && !m.progDone {
			cmds = append(cmds, waitForProgress(m.progressChan))
		}

	case repoChanDoneMsg:
		m.scanDone = true

	case progressChanDoneMsg:
		m.progDone = true

	// --- Animation / spinner ---
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
		m.fireBg = NewFireBackground(min(msg.Width-4, 70), 5)
		m = m.withClampedPathScroll()
		// Re-clamp scroll offsets for new height
		m.scrollOffset = m.clampScroll(m.scrollOffset, m.cursor, m.repoListVisibleCount(), len(m.repos))
		m.ignoredScrollOffset = m.clampScroll(m.ignoredScrollOffset, m.ignoredCursor, m.ignoredListVisibleCount(), len(m.ignoredEntries))

	case tickMsg:
		m.frameIndex = (m.frameIndex + 1) % len(fireFrames)
		if m.fireVisible() {
			m.fireBg.Update()
		}
		return m, tickCmd(m.fireTick)

	case quoteTickMsg:
		m.quoteTickActive = false
		if m.showStartupQuote {
			switch m.startupQuoteBehavior {
			case config.UIQuoteBehaviorHide:
				m.startupQuoteVisible = false
			default:
				m.currentStartupQuote = randomStartupFireQuote()
				m.startupQuoteVisible = true
			}
			if m.startupQuoteInterval > 0 {
				cmds = append(cmds, quoteTickCmd(m.startupQuoteInterval))
				m.quoteTickActive = true
			}
		}

	case pathScrollMsg:
		m = m.advancePathScroll()
		return m, pathScrollCmd()

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	// --- Keyboard ---
	case tea.KeyMsg:
		// Config view handles its own keys first.
		if m.view == repoViewConfig {
			return m.updateConfigView(msg, cmds)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "c":
			if m.view == repoViewMain && m.cfg != nil {
				m.view = repoViewConfig
			}
			return m, tea.Batch(cmds...)

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
					m.ignoredScrollOffset = m.clampScroll(m.ignoredScrollOffset, m.ignoredCursor, m.ignoredListVisibleCount(), len(m.ignoredEntries))
				}
			} else if m.cursor > 0 {
				m.cursor--
				m = m.withResetPathScroll()
				m.scrollOffset = m.clampScroll(m.scrollOffset, m.cursor, m.repoListVisibleCount(), len(m.repos))
			}

		case "down", "j":
			if m.view == repoViewIgnored {
				if m.ignoredCursor < len(m.ignoredEntries)-1 {
					m.ignoredCursor++
					m.ignoredScrollOffset = m.clampScroll(m.ignoredScrollOffset, m.ignoredCursor, m.ignoredListVisibleCount(), len(m.ignoredEntries))
				}
			} else if m.cursor < len(m.repos)-1 {
				m.cursor++
				m = m.withResetPathScroll()
				m.scrollOffset = m.clampScroll(m.scrollOffset, m.cursor, m.repoListVisibleCount(), len(m.repos))
			}

		case "left":
			if m.view == repoViewMain && m.pathScrollOffset > 0 {
				m.pathScrollOffset--
				m.pathScrollDir = -1
				m.pathScrollPause = 0
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
					m.pathScrollDir = 1
					m.pathScrollPause = 0
				}
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
			m = m.withResetPathScroll()
			m.scrollOffset = m.clampScroll(m.scrollOffset, m.cursor, m.repoListVisibleCount(), len(m.repos))

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

		case "f":
			if m.view == repoViewMain {
				m.showFire = !m.showFire
				if m.cfg != nil {
					m.cfg.UI.ShowFireAnimation = m.showFire
					m = m.saveConfig()
				}
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

// fireHeightThreshold is the minimum terminal height required to show the fire
// animation. Below this the animation is suppressed regardless of showFire so
// the list and controls remain usable.
const fireHeightThreshold = 20

// fireSectionReserveLines returns how many terminal rows the fire strip occupies
// when drawn: the ANSI grid from FireBackground.Render (Height lines), plus two
// lines after it—the wave string and the blank line before the title—matching
// View / viewIgnoredMain assembly (Render, then "\n"+wave+"\n\n").
//
// Keep this in sync with those layouts. Over-counting (e.g. Height+4) shrinks the
// repo list for no reason; under-counting lets the list overlap the fire block.
func (m RepoSelectorModel) fireSectionReserveLines() int {
	if !m.fireVisible() {
		return 0
	}
	return m.fireBg.Height + 2
}

func (m RepoSelectorModel) quoteVisible() bool {
	return m.showStartupQuote && m.startupQuoteVisible && m.currentStartupQuote != ""
}

func (m RepoSelectorModel) renderStartupQuote() string {
	quoteStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD166")).
		Italic(true)
	return quoteStyle.Render("🔥 \"" + m.currentStartupQuote + "\"")
}

// fireVisible reports whether the fire animation section should be rendered.
func (m RepoSelectorModel) fireVisible() bool {
	return m.showFire && m.windowHeight > fireHeightThreshold
}

// advancePathScroll steps the path marquee for the currently focused repo.
// Called from pathScrollMsg which fires at a fixed interval (pathScrollInterval),
// so cadence and edge pauses are measured in real time, not fire-tick time.
func (m RepoSelectorModel) advancePathScroll() RepoSelectorModel {
	if m.view != repoViewMain || len(m.repos) == 0 || m.cursor >= len(m.repos) {
		return m
	}
	repo := m.repos[m.cursor]
	parentPath := AbbreviateUserHome(filepath.Dir(repo.Path))
	pathLen := len([]rune(parentPath))
	pWidth := PathWidthFor(m.windowWidth, repo)
	if pathLen <= pWidth {
		m.pathScrollOffset = 0
		return m
	}
	maxOffset := pathLen - pWidth
	if m.pathScrollPause > 0 {
		m.pathScrollPause--
		return m
	}
	m.pathScrollOffset += m.pathScrollDir
	if m.pathScrollOffset >= maxOffset {
		m.pathScrollOffset = maxOffset
		m.pathScrollDir = -1
		m.pathScrollPause = 5
	} else if m.pathScrollOffset <= 0 {
		m.pathScrollOffset = 0
		m.pathScrollDir = 1
		m.pathScrollPause = 5
	}
	return m
}

// withResetPathScroll returns m with path-scroll state zeroed.
// Call whenever the focused repo changes (up/down/x/…).
func (m RepoSelectorModel) withResetPathScroll() RepoSelectorModel {
	m.pathScrollOffset = 0
	m.pathScrollDir = 1
	m.pathScrollPause = 0
	return m
}

// withClampedPathScroll returns m with pathScrollOffset clamped to the valid
// range for the current focused repo and window width.
// Call whenever windowWidth changes.
func (m RepoSelectorModel) withClampedPathScroll() RepoSelectorModel {
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

// repoListVisibleCount returns how many repo lines can be shown given the current
// window height. The repo list is the flex element: it absorbs all spare vertical
// space but is always at least 1 line tall.
//
// Overhead is measured dynamically so it stays accurate when the scan-status
// panel is present (streaming mode adds ~4–5 extra lines) and when help text
// wraps in narrow terminals.
func (m RepoSelectorModel) repoListVisibleCount() int {
	innerW := m.windowWidth - 6
	if innerW < 0 {
		innerW = 0
	}

	// Placeholder newlines stand in for the fire block so lipgloss.Height matches
	// the real View without running fire shaders. Must equal fireSectionReserveLines().
	var buf strings.Builder
	if lines := m.fireSectionReserveLines(); lines > 0 {
		for i := 0; i < lines; i++ {
			buf.WriteString("\n")
		}
	}
	if m.quoteVisible() {
		buf.WriteString(m.renderStartupQuote())
		buf.WriteString("\n\n")
	}
	buf.WriteString(lipgloss.NewStyle().Bold(true).
		Foreground(activeProfile().titleFg).
		Background(activeProfile().titleBg).
		Padding(0, 2).
		Render("🔥 GIT FIRE - SELECT REPOSITORIES 🔥"))
	buf.WriteString("\n\n")
	configHint := ""
	if m.cfg != nil {
		configHint = "  c  Settings  |  "
	}
	buf.WriteString(helpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  ←/→  Scroll path when << SCROLL PATH >> shows  |  space  Toggle selection\n" +
			"  m  Change mode  |  x  Ignore  |  a  Select all  |  n  Select none  |  f  Toggle fire\n" +
			"  i  View ignored  |  " + configHint + "enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected  |  ‹›  = path scrollable",
	))
	if m.scanChan != nil || m.scanDisabled {
		buf.WriteString("\n")
		buf.WriteString(m.renderScanStatus())
	}

	overhead := lipgloss.Height(boxStyle.Width(innerW).Render(buf.String()))
	n := m.windowHeight - overhead
	if n < 1 {
		n = 1
	}
	return n
}

// renderIgnoredViewTitle returns the styled title line for the ignored view.
// Keep in sync with viewIgnoredMain.
func (m RepoSelectorModel) renderIgnoredViewTitle() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(activeProfile().titleFg).
		Background(activeProfile().titleBg).
		Padding(0, 2).
		Render("🔥 IGNORED REPOSITORIES (NOT TRACKED) 🔥")
}

// renderIgnoredViewHelp returns the help block for the ignored view (below the list).
func renderIgnoredViewHelp() string {
	return helpStyle.Render(
		"\n" +
			"These repos are excluded from backup. Restore tracking with enter or u.\n" +
			"Controls:  ↑/k, ↓/j  Navigate  |  enter / u  Track again  |  i  Back to main  |  q  Quit\n",
	)
}

// ignoredViewNonListHeight is the boxed vertical size of the ignored view without
// any list rows: fire placeholder, title, help. Matches viewIgnoredMain assembly
// so wrapping help on narrow terminals inflates overhead the same as in the real view.
func (m RepoSelectorModel) ignoredViewNonListHeight() int {
	innerW := m.windowWidth - 6
	if innerW < 0 {
		innerW = 0
	}
	var buf strings.Builder
	if lines := m.fireSectionReserveLines(); lines > 0 {
		for i := 0; i < lines; i++ {
			buf.WriteString("\n")
		}
	}
	if m.quoteVisible() {
		buf.WriteString(m.renderStartupQuote())
		buf.WriteString("\n\n")
	}
	buf.WriteString(m.renderIgnoredViewTitle())
	buf.WriteString("\n\n")
	buf.WriteString(renderIgnoredViewHelp())
	return lipgloss.Height(boxStyle.Width(innerW).Render(buf.String()))
}

// ignoredListVisibleCount mirrors repoListVisibleCount: overhead is measured with
// lipgloss so help wrapping and box padding match the rendered ignored view.
func (m RepoSelectorModel) ignoredListVisibleCount() int {
	overhead := m.ignoredViewNonListHeight()
	n := m.windowHeight - overhead
	if n < 1 {
		n = 1
	}
	return n
}

// clampScroll returns a scroll offset that keeps cursor within the rendered item
// rows, accounting for the ↑/↓ indicator lines that consume viewport rows.
// It iterates to convergence (≤3 passes) because changing the offset can
// toggle which indicators appear, which in turn changes the item row count.
func (m RepoSelectorModel) clampScroll(offset, cursor, visible, total int) int {
	for range 3 {
		indicators := 0
		if offset > 0 {
			indicators++
		}
		if total > offset+visible {
			indicators++
		}
		itemVisible := visible - indicators
		if itemVisible < 1 {
			itemVisible = 1
		}
		var next int
		if cursor < offset {
			next = cursor
		} else if cursor >= offset+itemVisible {
			next = cursor - itemVisible + 1
		} else {
			next = offset
		}
		if next == offset {
			break
		}
		offset = next
	}
	return offset
}

// contentWidth returns the usable inner width for rendered content (box border+padding = 6 cols).
func (m RepoSelectorModel) contentWidth() int {
	w := m.windowWidth - 6
	if w < 0 {
		w = 0
	}
	return w
}

func viewportWarningRows(contentWidth int, warning string) int {
	if warning == "" {
		return 1
	}
	if contentWidth < 1 {
		return 1
	}
	h := lipgloss.Height(lipgloss.NewStyle().MaxWidth(contentWidth).Render(warning))
	if h < 1 {
		return 1
	}
	return h
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

	if m.view == repoViewConfig {
		return m.viewConfig()
	}

	cw := m.contentWidth()
	fireW := min(cw, 70)

	var s strings.Builder

	// Animated fire background and wave (suppressed when hidden or terminal too short)
	if m.fireVisible() {
		s.WriteString(m.fireBg.Render())
		s.WriteString("\n")
		s.WriteString(RenderFireWave(fireW, m.frameIndex))
		s.WriteString("\n\n")
	}

	// Title with gradient
	titleText := "🔥 GIT FIRE - SELECT REPOSITORIES 🔥"
	titleGradient := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeProfile().titleFg).
		Background(activeProfile().titleBg).
		Padding(0, 2)
	s.WriteString(titleGradient.Render(titleText))
	s.WriteString("\n\n")
	if m.quoteVisible() {
		s.WriteString(m.renderStartupQuote())
		s.WriteString("\n\n")
	}

	// Repository list — flex element, scrollable
	if len(m.repos) == 0 && !m.scanDone {
		s.WriteString(unselectedStyle.Render("  Waiting for repositories..."))
		s.WriteString("\n")
	}

	visible := m.repoListVisibleCount()
	// Re-clamp in case View is called before a scroll-adjusting Update
	scrollOffset := m.clampScroll(m.scrollOffset, m.cursor, visible, len(m.repos))

	// Scroll indicators each consume 1 line; subtract them from the viewport
	// so the box never overflows.
	hasAbove := scrollOffset > 0
	hasBelow := len(m.repos) > scrollOffset+visible
	indicators := 0
	if hasAbove {
		indicators++
	}
	if hasBelow {
		indicators++
	}
	itemVisible := visible - indicators
	hadHiddenRows := hasAbove || hasBelow
	indicatorsSuppressed := false
	viewportWarning := "  ⚠ More repos exist, but ↑/↓ indicators are hidden in this terminal size (enlarge window or press f)."
	warningRows := viewportWarningRows(cw, viewportWarning)
	if itemVisible < 1 {
		// Not enough room for both items and indicators. Suppress indicators so
		// we never render more lines than the visible budget.
		hasAbove = false
		hasBelow = false
		itemVisible = visible
		// Only show the warning when we can reserve enough viewport rows for it.
		if hadHiddenRows && visible-warningRows >= 1 {
			indicatorsSuppressed = true
			itemVisible = visible - warningRows
		}
		if itemVisible < 1 {
			itemVisible = 1
		}
	}
	end := scrollOffset + itemVisible
	if end > len(m.repos) {
		end = len(m.repos)
	}

	// Fixed parts of a repo line (before the path): "> [✓]  [mode] (N remotes) 💥"
	// "> " (2) + "[✓] " (4) + "  [" (3) + mode + "] " (2) + remotes + " 💥" (4) ≈ 15 + mode + remotes
	// Reserve ~35 cols for the non-path parts; remainder goes to the path.
	const nonPathCols = 35
	maxPathCols := cw - nonPathCols
	if maxPathCols < 0 {
		maxPathCols = 0
	}

	if hasAbove {
		s.WriteString(unselectedStyle.Render(fmt.Sprintf("  ↑ %d more", scrollOffset)))
		s.WriteString("\n")
	}

	for i := scrollOffset; i < end; i++ {
		repo := m.repos[i]
		cur := " "
		if m.cursor == i {
			cur = ">"
		}

		checked := "[ ]"
		style := unselectedStyle
		if m.selected[i] {
			checked = "[✓]"
			style = selectedStyle
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
			scrollHint = "  " + scrollHintStyle.Render("<< SCROLL PATH >>")
		}

		line := fmt.Sprintf("%s %s %s (%s%s%s)  [%s] %s%s%s",
			cur,
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

	if hasBelow {
		below := len(m.repos) - end
		s.WriteString(unselectedStyle.Render(fmt.Sprintf("  ↓ %d more", below)))
		s.WriteString("\n")
	}
	if indicatorsSuppressed {
		s.WriteString(viewportWarningStyle.Render(viewportWarning))
		s.WriteString("\n")
	}

	// Help text
	configHint := ""
	if m.cfg != nil {
		configHint = "  c  Settings  |  "
	}
	help := helpStyle.Render(
		"\n" +
			"Controls:\n" +
			"  ↑/k, ↓/j  Navigate  |  ←/→  Scroll path when << SCROLL PATH >> shows  |  space  Toggle selection\n" +
			"  m  Change mode  |  x  Ignore  |  a  Select all  |  n  Select none  |  f  Toggle fire\n" +
			"  i  View ignored  |  " + configHint + "enter  Confirm  |  q  Quit\n\n" +
			"Icons:\n" +
			"  💥 = Has uncommitted changes (will auto-commit before push)\n" +
			"  [✓] = Selected  |  [ ] = Not selected  |  ‹›  = path scrollable",
	)
	s.WriteString(help)

	// Scan-status panel (only in streaming mode)
	if m.scanChan != nil || m.scanDisabled {
		s.WriteString("\n")
		s.WriteString(m.renderScanStatus())
	}

	// Wrap in a box sized to the terminal width
	innerW := m.windowWidth - 6 // border(2) + padding(4)
	if innerW < 0 {
		innerW = 0
	}
	return boxStyle.Width(innerW).Render(s.String())
}

// renderScanStatus produces the scan-status line shown at the bottom of the
// main repo selector view when running in streaming mode.
func (m RepoSelectorModel) renderScanStatus() string {
	scanStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeProfile().scanBorder).
		Padding(0, 1)

	switch {
	case m.scanDisabled:
		var label string
		if m.scanDisabledRunOnly {
			label = "⚠️  Scanning Disabled (this run only)"
		} else {
			label = "⚠️  Scanning Disabled"
		}
		return scanStyle.Render(lipgloss.NewStyle().Foreground(activeProfile().scanWarn).Render(label))

	case m.scanDone:
		msg := fmt.Sprintf("✅ Scan Complete  (%d new repos found)", m.scanNewCount)
		return scanStyle.Render(lipgloss.NewStyle().Foreground(activeProfile().scanDone).Render(msg))

	default:
		folder := m.scanCurrentPath
		if folder == "" {
			folder = "..."
		}
		// Truncate long paths to keep the panel narrow.
		maxLen := 50
		if len(folder) > maxLen {
			folder = "..." + folder[len(folder)-maxLen+3:]
		}
		line1 := fmt.Sprintf("🔍 Scanning: %s", folder)
		line2 := fmt.Sprintf("   New repos found this session: %d", m.scanNewCount)
		return scanStyle.Render(line1 + "\n" + line2)
	}
}

func (m RepoSelectorModel) viewIgnoredMain() string {
	cw := m.contentWidth()
	fireW := min(cw, 70)

	var s strings.Builder
	if m.fireVisible() {
		s.WriteString(m.fireBg.Render())
		s.WriteString("\n")
		s.WriteString(RenderFireWave(fireW, m.frameIndex))
		s.WriteString("\n\n")
	}

	s.WriteString(m.renderIgnoredViewTitle())
	s.WriteString("\n\n")
	if m.quoteVisible() {
		s.WriteString(m.renderStartupQuote())
		s.WriteString("\n\n")
	}

	if len(m.ignoredEntries) == 0 {
		s.WriteString(unselectedStyle.Render("No ignored repositories."))
		s.WriteString("\n")
	} else {
		visible := m.ignoredListVisibleCount()
		scrollOffset := m.clampScroll(m.ignoredScrollOffset, m.ignoredCursor, visible, len(m.ignoredEntries))

		hasAbove := scrollOffset > 0
		hasBelow := len(m.ignoredEntries) > scrollOffset+visible
		indicators := 0
		if hasAbove {
			indicators++
		}
		if hasBelow {
			indicators++
		}

		maxPathCols := cw - 4
		if maxPathCols < 0 {
			maxPathCols = 0
		}

		itemVisible := visible - indicators
		hadHiddenRows := hasAbove || hasBelow
		indicatorsSuppressed := false
		viewportWarning := "  ⚠ More ignored repos exist, but ↑/↓ indicators are hidden in this terminal size."
		warningRows := viewportWarningRows(cw, viewportWarning)
		if itemVisible < 1 {
			hasAbove = false
			hasBelow = false
			itemVisible = visible
			if hadHiddenRows && visible-warningRows >= 1 {
				indicatorsSuppressed = true
				itemVisible = visible - warningRows
			}
			if itemVisible < 1 {
				itemVisible = 1
			}
		}
		end := scrollOffset + itemVisible
		if end > len(m.ignoredEntries) {
			end = len(m.ignoredEntries)
		}

		if hasAbove {
			s.WriteString(unselectedStyle.Render(fmt.Sprintf("  ↑ %d more", scrollOffset)))
			s.WriteString("\n")
		}

		for i := scrollOffset; i < end; i++ {
			e := m.ignoredEntries[i]
			cur := " "
			if m.ignoredCursor == i {
				cur = ">"
			}
			displayPath := AbbreviateUserHome(e.Path)
			if maxPathCols == 0 {
				displayPath = ""
			} else if len([]rune(displayPath)) > maxPathCols {
				displayPath = string([]rune(displayPath)[:maxPathCols-1]) + "…"
			}
			s.WriteString(fmt.Sprintf("%s %s\n", cur, displayPath))
		}

		if hasBelow {
			below := len(m.ignoredEntries) - end
			s.WriteString(unselectedStyle.Render(fmt.Sprintf("  ↓ %d more", below)))
			s.WriteString("\n")
		}
		if indicatorsSuppressed {
			s.WriteString(viewportWarningStyle.Render(viewportWarning))
			s.WriteString("\n")
		}
	}

	s.WriteString(renderIgnoredViewHelp())

	innerW := m.windowWidth - 6
	if innerW < 0 {
		innerW = 0
	}
	return boxStyle.Width(innerW).Render(s.String())
}

// persistMode writes the repo's current mode to the registry synchronously.
// Errors are silently ignored — this is best-effort during an emergency.
func (m RepoSelectorModel) persistMode(repoPath string, mode git.RepoMode) {
	_ = selectorPersistMode(m.reg, m.regPath, repoPath, mode)
}

// persistIgnore marks the repo as ignored in the registry synchronously.
func (m RepoSelectorModel) persistIgnore(repoPath string) {
	_ = selectorPersistIgnore(m.reg, m.regPath, repoPath)
}

// GetSelectedRepos returns the selected repositories
func (m RepoSelectorModel) GetSelectedRepos() []git.Repository {
	return selectorGetSelected(m.repos, m.selected)
}

// RunRepoSelector runs the interactive repo selector and returns selected repos.
// reg and regPath are used for write-through persistence of mode changes and
// ignored repos; pass nil/empty to disable persistence.
func RunRepoSelector(repos []git.Repository, reg *registry.Registry, regPath string) ([]git.Repository, error) {
	model := NewRepoSelectorModel(repos, reg, regPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

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

// RunRepoSelectorStream runs the interactive repo selector in streaming mode:
// repos are added to the list as they arrive on scanChan, and a scan-status
// panel shows live scanning progress via progressChan.
//
// scanDisabled should be true when --no-scan / disable_scan is set so the
// panel shows the appropriate "Scanning Disabled" indicator instead of
// progress. scanDisabledRunOnly should be true only when --no-scan was passed
// (run-time), so the label differs from persisted disable_scan in config.
// cfg and cfgPath enable the in-TUI config menu (pass nil/empty to disable).
func RunRepoSelectorStream(
	scanChan <-chan git.Repository,
	progressChan <-chan string,
	scanDisabled bool,
	scanDisabledRunOnly bool,
	cfg *config.Config,
	cfgPath string,
	reg *registry.Registry,
	regPath string,
) ([]git.Repository, error) {
	model := NewRepoSelectorModelStream(scanChan, progressChan, scanDisabled, scanDisabledRunOnly, cfg, cfgPath, reg, regPath)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	m := finalModel.(RepoSelectorModel)
	if !m.confirmed {
		return nil, ErrCancelled
	}

	return m.GetSelectedRepos(), nil
}
