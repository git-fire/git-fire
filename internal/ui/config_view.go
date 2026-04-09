package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/git-fire/git-fire/internal/config"
)

// configRow describes one editable row in the config menu.
type configRow struct {
	label   string
	kind    configRowKind
	options []string // for enum rows; unused for bool rows
}

type configRowKind int

const (
	configRowBool configRowKind = iota
	configRowEnum
	configRowComingSoon
)

var configRows = []configRow{
	{label: "Default mode", kind: configRowEnum, options: []string{
		"push-known-branches",
		"push-all",
		"leave-untouched",
		"push-current-branch",
	}},
	{label: "Auto-commit dirty repos", kind: configRowBool},
	{label: "Conflict strategy", kind: configRowEnum, options: []string{
		"new-branch",
		"abort",
	}},
	{label: "Disable scan", kind: configRowBool},
	{label: "Push workers", kind: configRowEnum, options: []string{
		"1",
		"2",
		"4",
		"8",
		"16",
	}},
	{label: "Show fire animation", kind: configRowBool},
	{label: "Show flavor quotes", kind: configRowBool},
	{label: "Flavor quote behavior", kind: configRowEnum, options: []string{
		config.UIQuoteBehaviorRefresh,
		config.UIQuoteBehaviorHide,
	}},
	{label: "Flavor quote interval (s)", kind: configRowEnum, options: []string{
		"5",
		"10",
		"15",
		"30",
	}},
	{label: "Fire speed (ms)", kind: configRowEnum, options: []string{
		"120",
		"150",
		"180",
		"220",
		"280",
		"340",
	}},
	{label: "Color profile", kind: configRowEnum, options: config.UIColorProfiles()},
	{label: "Custom hex palette", kind: configRowComingSoon},
}

// configRowValue returns the current string representation of row i for cfg.
func configRowValue(i int, cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	switch i {
	case 0:
		return cfg.Global.DefaultMode
	case 1:
		if cfg.Global.AutoCommitDirty {
			return "true"
		}
		return "false"
	case 2:
		return cfg.Global.ConflictStrategy
	case 3:
		if cfg.Global.DisableScan {
			return "true"
		}
		return "false"
	case 4:
		return strconv.Itoa(cfg.Global.PushWorkers)
	case 5:
		if cfg.UI.ShowFireAnimation {
			return "true"
		}
		return "false"
	case 6:
		if cfg.UI.ShowStartupQuote {
			return "true"
		}
		return "false"
	case 7:
		return cfg.UI.StartupQuoteBehavior
	case 8:
		return strconv.Itoa(cfg.UI.StartupQuoteIntervalSec)
	case 9:
		if cfg.UI.FireTickMS <= 0 {
			return strconv.Itoa(config.DefaultUIFireTickMS)
		}
		return strconv.Itoa(cfg.UI.FireTickMS)
	case 10:
		return cfg.UI.ColorProfile
	case 11:
		return palettePreviewString(activeFireColors)
	}
	return ""
}

// applyConfigChange mutates cfg for row i in the given direction (+1 = next, -1 = prev for enums;
// toggled for bools).
func applyConfigChange(i int, cfg *config.Config, dir int) {
	if cfg == nil {
		return
	}
	row := configRows[i]
	switch row.kind {
	case configRowBool:
		switch i {
		case 1:
			cfg.Global.AutoCommitDirty = !cfg.Global.AutoCommitDirty
		case 3:
			cfg.Global.DisableScan = !cfg.Global.DisableScan
		case 5:
			cfg.UI.ShowFireAnimation = !cfg.UI.ShowFireAnimation
		case 6:
			cfg.UI.ShowStartupQuote = !cfg.UI.ShowStartupQuote
		}
	case configRowEnum:
		opts := row.options
		cur := configRowValue(i, cfg)
		idx := 0
		for j, o := range opts {
			if o == cur {
				idx = j
				break
			}
		}
		idx = (idx + dir + len(opts)) % len(opts)
		switch i {
		case 0:
			cfg.Global.DefaultMode = opts[idx]
		case 2:
			cfg.Global.ConflictStrategy = opts[idx]
		case 4:
			workers, err := strconv.Atoi(opts[idx])
			if err == nil && workers > 0 {
				cfg.Global.PushWorkers = workers
			}
		case 7:
			cfg.UI.StartupQuoteBehavior = opts[idx]
		case 8:
			sec, err := strconv.Atoi(opts[idx])
			if err == nil && sec > 0 {
				cfg.UI.StartupQuoteIntervalSec = sec
			}
		case 9:
			applyFireTickChange(cfg, opts, dir)
		case 10:
			cfg.UI.ColorProfile = opts[idx]
		}
	case configRowComingSoon:
		// Reserved for future custom hex palette editing.
	}
}

func palettePreviewString(palette []lipgloss.Color) string {
	if len(palette) == 0 {
		return "coming soon"
	}
	preview := make([]string, 0, min(4, len(palette)))
	for i := 0; i < len(palette) && i < 4; i++ {
		preview = append(preview, string(palette[i]))
	}
	return strings.Join(preview, " ")
}

// updateConfigView handles key input while the config view is active.
func (m RepoSelectorModel) updateConfigView(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "c", "esc":
		m.view = repoViewMain
		return m, tea.Batch(cmds...)

	case "up", "k":
		if m.configCursor > 0 {
			m.configCursor--
		}

	case "down", "j":
		if m.configCursor < len(configRows)-1 {
			m.configCursor++
		}

	case " ", "right", "l":
		applyConfigChange(m.configCursor, m.cfg, +1)
		if m.cfg != nil {
			applyColorProfile(m.cfg.UI.ColorProfile)
		}
		m = m.saveConfig()
		m, cmds = m.syncRuntimeFromConfig(cmds)

	case "left", "h":
		applyConfigChange(m.configCursor, m.cfg, -1)
		if m.cfg != nil {
			applyColorProfile(m.cfg.UI.ColorProfile)
		}
		m = m.saveConfig()
		m, cmds = m.syncRuntimeFromConfig(cmds)
	}

	return m, tea.Batch(cmds...)
}

// updateConfigViewMouse handles scroll-wheel movement on the settings screen.
func (m RepoSelectorModel) updateConfigViewMouse(ev tea.MouseEvent, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	if ev.Action != tea.MouseActionPress || !ev.IsWheel() {
		return m, tea.Batch(cmds...)
	}
	d := mouseWheelVerticalDelta(ev)
	if d == 0 {
		return m, tea.Batch(cmds...)
	}
	if d < 0 {
		if m.configCursor > 0 {
			m.configCursor--
		}
	} else if m.configCursor < len(configRows)-1 {
		m.configCursor++
	}
	return m, tea.Batch(cmds...)
}

// saveConfig writes the current config to disk and records success or failure on the model.
func (m RepoSelectorModel) saveConfig() RepoSelectorModel {
	if m.cfg == nil || m.cfgPath == "" {
		return m
	}
	if err := config.SaveConfig(m.cfg, m.cfgPath); err != nil {
		m.configSaveErr = err
	} else {
		m.configSaveErr = nil
	}
	return m
}

// viewConfig renders the settings screen.
func (m RepoSelectorModel) viewConfig() string {
	var s strings.Builder

	if m.fireVisible() {
		s.WriteString(m.fireBg.Render())
		s.WriteString("\n")
		s.WriteString(RenderFireWave(min(m.windowWidth-4, 70), m.frameIndex))
		s.WriteString("\n\n")
	}

	titleGradient := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeProfile().titleFg).
		Background(activeProfile().titleBg).
		Padding(0, 2)
	s.WriteString(titleGradient.Render("🔥 GIT FIRE — SETTINGS"))
	s.WriteString("\n\n")

	cursorStyle := lipgloss.NewStyle().Foreground(activeProfile().configCursor).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(activeProfile().configLabel)
	valueStyle := lipgloss.NewStyle().Foreground(activeProfile().configValue).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(activeProfile().configDim)

	for i, row := range configRows {
		cur := " "
		if m.configCursor == i {
			cur = ">"
		}

		val := configRowValue(i, m.cfg)

		hintStr := ""
		if m.configCursor == i {
			switch row.kind {
			case configRowBool:
				hintStr = dimStyle.Render("  space to toggle")
			case configRowComingSoon:
				hintStr = dimStyle.Render("  coming soon")
			default:
				hintStr = dimStyle.Render("  ←/→ to change")
			}
		}

		line := fmt.Sprintf("%s  %-30s %s%s",
			cursorStyle.Render(cur),
			labelStyle.Render(row.label+":"),
			valueStyle.Render(val),
			hintStr,
		)
		s.WriteString(line)
		s.WriteString("\n")
	}

	s.WriteString("\n")
	if m.configSaveErr != nil {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6666"))
		s.WriteString(errStyle.Render("⚠️  Save failed: " + m.configSaveErr.Error()))
		s.WriteString("\n")
		s.WriteString(helpStyle.Render(
			"In-memory settings updated; fix the error above to persist to disk.\n" +
				"Custom hex palette editing is coming soon.\n" +
				"Controls:  ↑/k, ↓/j / mouse wheel  Navigate  |  space/→  Next value  |  ←  Prev value  |  c/Esc  Back  |  q  Quit",
		))
	} else {
		cfgPathStr := m.cfgPath
		if cfgPathStr == "" {
			cfgPathStr = "(config path unknown — changes not saved)"
		} else {
			cfgPathStr = AbbreviateUserHome(cfgPathStr)
		}
		s.WriteString(helpStyle.Render(
			"Changes saved immediately to " + cfgPathStr + "\n" +
				"Custom hex palette editing is coming soon.\n" +
				"Controls:  ↑/k, ↓/j / mouse wheel  Navigate  |  space/→  Next value  |  ←  Prev value  |  c/Esc  Back  |  q  Quit",
		))
	}

	return boxStyle.Render(s.String())
}

func applyFireTickChange(cfg *config.Config, options []string, dir int) {
	if cfg == nil || len(options) == 0 || dir == 0 {
		return
	}
	cur := cfg.UI.FireTickMS
	if cur <= 0 {
		cur = config.DefaultUIFireTickMS
	}

	// Support manual overrides from config.toml by moving to the next/prev preset
	// relative to the current numeric value, even if it is not exactly a preset.
	if dir > 0 {
		for _, opt := range options {
			v, err := strconv.Atoi(opt)
			if err == nil && v > cur {
				cfg.UI.FireTickMS = v
				return
			}
		}
		// Wrap to first preset.
		if v, err := strconv.Atoi(options[0]); err == nil {
			cfg.UI.FireTickMS = v
		}
		return
	}

	for i := len(options) - 1; i >= 0; i-- {
		v, err := strconv.Atoi(options[i])
		if err == nil && v < cur {
			cfg.UI.FireTickMS = v
			return
		}
	}
	// Wrap to last preset.
	if v, err := strconv.Atoi(options[len(options)-1]); err == nil {
		cfg.UI.FireTickMS = v
	}
}

func (m RepoSelectorModel) syncRuntimeFromConfig(cmds []tea.Cmd) (RepoSelectorModel, []tea.Cmd) {
	if m.cfg == nil {
		return m, cmds
	}
	wasShowingStartupQuote := m.showStartupQuote
	m.showFire = m.cfg.UI.ShowFireAnimation
	m.fireTick = time.Duration(m.cfg.UI.FireTickMS) * time.Millisecond
	m.showStartupQuote = m.cfg.UI.ShowStartupQuote
	m.startupQuoteBehavior = m.cfg.UI.StartupQuoteBehavior
	m.startupQuoteInterval = time.Duration(m.cfg.UI.StartupQuoteIntervalSec) * time.Second
	if m.showStartupQuote {
		if m.currentStartupQuote == "" {
			m.currentStartupQuote = randomStartupFireQuote()
		}
		// Re-show only when the feature is toggled on; avoid reviving hidden quotes
		// on unrelated settings changes.
		if !wasShowingStartupQuote {
			m.startupQuoteVisible = true
		}
		if m.startupQuoteInterval > 0 && !m.quoteTickActive {
			cmds = append(cmds, quoteTickCmd(m.startupQuoteInterval))
			m.quoteTickActive = true
		}
	} else {
		m.startupQuoteVisible = false
		m.quoteTickActive = false
	}
	if m.startupQuoteInterval <= 0 {
		m.quoteTickActive = false
	}
	return m, cmds
}
