package ui

import tea "github.com/charmbracelet/bubbletea"

// horizontalWheelPathSteps is how many runes to move the path marquee per
// horizontal wheel notch (trackpads often emit one event per small scroll).
const horizontalWheelPathSteps = 3

// liteNavPageStep is the PageUp/PageDown jump size for RepoSelectorLiteModel,
// which has no viewport-based row count (full list is always rendered).
func liteNavPageStep(total int) int {
	if total <= 1 {
		return 1
	}
	step := 10
	if step > total-1 {
		step = total - 1
	}
	if step < 1 {
		step = 1
	}
	return step
}

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
