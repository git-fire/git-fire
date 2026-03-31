package ui

import (
	"fmt"
	"strings"

	"github.com/TBRX103/git-fire/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	{label: "Show fire animation", kind: configRowBool},
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
		if cfg.UI.ShowFireAnimation {
			return "true"
		}
		return "false"
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
		case 4:
			cfg.UI.ShowFireAnimation = !cfg.UI.ShowFireAnimation
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
		}
	}
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
		m = m.saveConfig()

	case "left", "h":
		applyConfigChange(m.configCursor, m.cfg, -1)
		m = m.saveConfig()
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

	s.WriteString(m.fireBg.Render())
	s.WriteString("\n")
	s.WriteString(RenderFireWave(min(m.windowWidth-4, 70), m.frameIndex))
	s.WriteString("\n\n")

	titleGradient := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ff4500")).
		Background(lipgloss.Color("#1a1a1a")).
		Padding(0, 2)
	s.WriteString(titleGradient.Render("🔥 GIT FIRE — SETTINGS"))
	s.WriteString("\n\n")

	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF99")).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))

	for i, row := range configRows {
		cur := " "
		if m.configCursor == i {
			cur = ">"
		}

		val := configRowValue(i, m.cfg)

		hintStr := ""
		if m.configCursor == i {
			if row.kind == configRowBool {
				hintStr = dimStyle.Render("  space to toggle")
			} else {
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
				"Controls:  ↑/k, ↓/j  Navigate  |  space/→  Next value  |  ←  Prev value  |  c/Esc  Back  |  q  Quit",
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
				"Controls:  ↑/k, ↓/j  Navigate  |  space/→  Next value  |  ←  Prev value  |  c/Esc  Back  |  q  Quit",
		))
	}

	return boxStyle.Render(s.String())
}
