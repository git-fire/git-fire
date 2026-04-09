package ui

import tea "github.com/charmbracelet/bubbletea"

// horizontalWheelPathSteps is how many runes to move the path marquee per
// horizontal wheel notch (trackpads often emit one event per small scroll).
const horizontalWheelPathSteps = 3

func mouseWheelVerticalDelta(ev tea.MouseEvent) int {
	switch ev.Button {
	case tea.MouseButtonWheelUp:
		return -1
	case tea.MouseButtonWheelDown:
		return +1
	default:
		return 0
	}
}

func mouseWheelHorizontalDelta(ev tea.MouseEvent) int {
	switch ev.Button {
	case tea.MouseButtonWheelLeft:
		return -1
	case tea.MouseButtonWheelRight:
		return +1
	default:
		return 0
	}
}
