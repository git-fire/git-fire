package ui

import (
	"github.com/git-fire/git-fire/internal/config"
	"github.com/charmbracelet/lipgloss"
)

type colorProfile struct {
	fire         []lipgloss.Color
	titleFg      lipgloss.Color
	titleBg      lipgloss.Color
	selected     lipgloss.Color
	unselected   lipgloss.Color
	help         lipgloss.Color
	scrollHint   lipgloss.Color
	viewportWarn lipgloss.Color
	boxBorder    lipgloss.Color
	scanBorder   lipgloss.Color
	scanWarn     lipgloss.Color
	scanDone     lipgloss.Color
	configCursor lipgloss.Color
	configLabel  lipgloss.Color
	configValue  lipgloss.Color
	configDim    lipgloss.Color
}

var profileMap = map[string]colorProfile{
	config.UIColorProfileClassic: {
		fire: []lipgloss.Color{
			lipgloss.Color("#ff0000"),
			lipgloss.Color("#ff4500"),
			lipgloss.Color("#ff6600"),
			lipgloss.Color("#ff8c00"),
			lipgloss.Color("#ffa500"),
			lipgloss.Color("#ffb700"),
			lipgloss.Color("#ffd700"),
			lipgloss.Color("#ffff00"),
		},
		titleFg:      lipgloss.Color("#FF6600"),
		titleBg:      lipgloss.Color("#1A1A1A"),
		selected:     lipgloss.Color("#00FF00"),
		unselected:   lipgloss.Color("#888888"),
		help:         lipgloss.Color("#666666"),
		scrollHint:   lipgloss.Color("#FFD166"),
		viewportWarn: lipgloss.Color("#FFAA00"),
		boxBorder:    lipgloss.Color("#FF6600"),
		scanBorder:   lipgloss.Color("#555555"),
		scanWarn:     lipgloss.Color("#FFAA00"),
		scanDone:     lipgloss.Color("#00CC66"),
		configCursor: lipgloss.Color("#FF6600"),
		configLabel:  lipgloss.Color("#CCCCCC"),
		configValue:  lipgloss.Color("#00FF99"),
		configDim:    lipgloss.Color("#666666"),
	},
	config.UIColorProfileSynthwave: {
		fire: []lipgloss.Color{
			lipgloss.Color("#2E1065"),
			lipgloss.Color("#5B21B6"),
			lipgloss.Color("#7C3AED"),
			lipgloss.Color("#A21CAF"),
			lipgloss.Color("#DB2777"),
			lipgloss.Color("#F43F5E"),
			lipgloss.Color("#FB7185"),
			lipgloss.Color("#FDE047"),
		},
		titleFg:      lipgloss.Color("#F472B6"),
		titleBg:      lipgloss.Color("#130A2A"),
		selected:     lipgloss.Color("#22D3EE"),
		unselected:   lipgloss.Color("#A78BFA"),
		help:         lipgloss.Color("#8B5CF6"),
		scrollHint:   lipgloss.Color("#FDE047"),
		viewportWarn: lipgloss.Color("#FB7185"),
		boxBorder:    lipgloss.Color("#C026D3"),
		scanBorder:   lipgloss.Color("#7E22CE"),
		scanWarn:     lipgloss.Color("#FB7185"),
		scanDone:     lipgloss.Color("#22D3EE"),
		configCursor: lipgloss.Color("#F472B6"),
		configLabel:  lipgloss.Color("#E9D5FF"),
		configValue:  lipgloss.Color("#67E8F9"),
		configDim:    lipgloss.Color("#8B5CF6"),
	},
	config.UIColorProfileForest: {
		fire: []lipgloss.Color{
			lipgloss.Color("#0B3D20"),
			lipgloss.Color("#14532D"),
			lipgloss.Color("#166534"),
			lipgloss.Color("#15803D"),
			lipgloss.Color("#16A34A"),
			lipgloss.Color("#22C55E"),
			lipgloss.Color("#86EFAC"),
			lipgloss.Color("#ECFCCB"),
		},
		titleFg:      lipgloss.Color("#22C55E"),
		titleBg:      lipgloss.Color("#0A1A12"),
		selected:     lipgloss.Color("#A3E635"),
		unselected:   lipgloss.Color("#86A88F"),
		help:         lipgloss.Color("#6B8F71"),
		scrollHint:   lipgloss.Color("#BEF264"),
		viewportWarn: lipgloss.Color("#F59E0B"),
		boxBorder:    lipgloss.Color("#16A34A"),
		scanBorder:   lipgloss.Color("#3F6E4F"),
		scanWarn:     lipgloss.Color("#F59E0B"),
		scanDone:     lipgloss.Color("#4ADE80"),
		configCursor: lipgloss.Color("#22C55E"),
		configLabel:  lipgloss.Color("#D1FAE5"),
		configValue:  lipgloss.Color("#A3E635"),
		configDim:    lipgloss.Color("#6B8F71"),
	},
	config.UIColorProfileArctic: {
		fire: []lipgloss.Color{
			lipgloss.Color("#0B1D2A"),
			lipgloss.Color("#0F3B57"),
			lipgloss.Color("#155E75"),
			lipgloss.Color("#0891B2"),
			lipgloss.Color("#06B6D4"),
			lipgloss.Color("#22D3EE"),
			lipgloss.Color("#67E8F9"),
			lipgloss.Color("#CCFBF1"),
		},
		titleFg:      lipgloss.Color("#22D3EE"),
		titleBg:      lipgloss.Color("#0A1520"),
		selected:     lipgloss.Color("#7DD3FC"),
		unselected:   lipgloss.Color("#7C9DB5"),
		help:         lipgloss.Color("#5F8199"),
		scrollHint:   lipgloss.Color("#93C5FD"),
		viewportWarn: lipgloss.Color("#F59E0B"),
		boxBorder:    lipgloss.Color("#06B6D4"),
		scanBorder:   lipgloss.Color("#3F6A82"),
		scanWarn:     lipgloss.Color("#F59E0B"),
		scanDone:     lipgloss.Color("#2DD4BF"),
		configCursor: lipgloss.Color("#22D3EE"),
		configLabel:  lipgloss.Color("#E0F2FE"),
		configValue:  lipgloss.Color("#7DD3FC"),
		configDim:    lipgloss.Color("#5F8199"),
	},
}

var activeFireColors = profileMap[config.UIColorProfileClassic].fire
var activeProfileName = config.UIColorProfileClassic

func applyColorProfile(profile string) string {
	p, ok := profileMap[profile]
	if !ok {
		profile = config.UIColorProfileClassic
		p = profileMap[profile]
	}
	activeProfileName = profile
	activeFireColors = p.fire

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(p.titleFg).Background(p.titleBg).Padding(0, 2).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Foreground(p.selected).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(p.unselected)
	helpStyle = lipgloss.NewStyle().Foreground(p.help).MarginTop(1)
	scrollHintStyle = lipgloss.NewStyle().Foreground(p.scrollHint).Bold(true)
	viewportWarningStyle = lipgloss.NewStyle().Foreground(p.viewportWarn).Bold(true)
	boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.boxBorder).Padding(1, 2)

	liteBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.boxBorder).Padding(1, 2)
	liteTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(p.titleFg).MarginBottom(1)
	liteSelectedStyle = lipgloss.NewStyle().Foreground(p.selected).Bold(true)
	liteUnselectedStyle = lipgloss.NewStyle().Foreground(p.unselected)
	liteHelpStyle = lipgloss.NewStyle().Foreground(p.help).MarginTop(1)
	liteScrollHintStyle = lipgloss.NewStyle().Foreground(p.scrollHint).Bold(true)

	return profile
}

func activeProfile() colorProfile {
	if p, ok := profileMap[activeProfileName]; ok {
		return p
	}
	return colorProfile{}
}
