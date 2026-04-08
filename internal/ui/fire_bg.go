package ui

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var fireParticleChars = [...]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "░", "▒", "▓"}

// FireParticle represents a single fire particle
type FireParticle struct {
	X        int
	Y        int
	Char     string
	ColorIdx int
	Age      int
	MaxAge   int
}

// FireBackground manages animated fire background
type FireBackground struct {
	Width     int
	Height    int
	Particles []FireParticle
	Frame     int
}

// NewFireBackground creates a new fire background
func NewFireBackground(width, height int) *FireBackground {
	bg := &FireBackground{
		Width:     width,
		Height:    height,
		Particles: make([]FireParticle, 0),
		Frame:     0,
	}

	// Initialize particles
	bg.Reset()

	return bg
}

// Reset reinitializes all particles
func (fb *FireBackground) Reset() {
	fb.Particles = make([]FireParticle, 0)

	// Create fire particles at the bottom
	particleCount := fb.Width * 2
	for i := 0; i < particleCount; i++ {
		fb.spawnParticle()
	}
}

// spawnParticle creates a new fire particle at the bottom
func (fb *FireBackground) spawnParticle() {
	if fb.Width <= 0 || fb.Height <= 0 {
		return
	}

	particle := FireParticle{
		X:        rand.Intn(fb.Width),
		Y:        fb.Height - 1,
		Char:     fireParticleChars[rand.Intn(len(fireParticleChars))],
		ColorIdx: 0, // Start at bottom (red)
		Age:      0,
		MaxAge:   fb.Height + rand.Intn(5),
	}

	fb.Particles = append(fb.Particles, particle)
}

// Update advances the animation by one frame
func (fb *FireBackground) Update() {
	fb.Frame++

	// Update each particle
	for i := range fb.Particles {
		p := &fb.Particles[i]
		p.Age++

		// Move particle up (fire rises)
		if fb.Frame%2 == 0 { // Slow down movement
			p.Y--
		}

		// Drift left/right with gentle variation to reduce visual jitter.
		driftChance := 0.22 + 0.04*math.Sin(float64(fb.Frame)*0.08)
		if rand.Float64() < driftChance {
			p.X += rand.Intn(3) - 1 // -1, 0, or 1
		}

		// Keep X in bounds
		if p.X < 0 {
			p.X = 0
		}
		if p.X >= fb.Width {
			p.X = fb.Width - 1
		}

		// Change color as it rises (red -> orange -> yellow)
		progress := float64(p.Age) / float64(p.MaxAge)
		paletteLen := len(activeFireColors)
		if paletteLen == 0 {
			paletteLen = 1
		}
		p.ColorIdx = int(progress * float64(paletteLen-1))
		if p.ColorIdx >= paletteLen {
			p.ColorIdx = paletteLen - 1
		}
	}

	// Remove dead particles in-place (off screen or too old).
	alive := fb.Particles[:0]
	for _, p := range fb.Particles {
		if p.Y >= 0 && p.Age < p.MaxAge {
			alive = append(alive, p)
		}
	}
	fb.Particles = alive

	// Spawn new particles to maintain count
	if fb.Width > 0 && fb.Height > 0 {
		targetCount := fb.Width * 2
		for len(fb.Particles) < targetCount {
			fb.spawnParticle()
		}
	}
}

// Render returns the fire background as a string
func (fb *FireBackground) Render() string {
	if fb.Width <= 0 || fb.Height <= 0 {
		return ""
	}
	// Flat grid avoids per-row allocations in the hot path.
	cellCount := fb.Width * fb.Height
	cells := make([]string, cellCount)
	for i := range cells {
		cells[i] = " "
	}
	styles := fireColorStyles()

	// Place particles on grid
	for _, p := range fb.Particles {
		if p.Y >= 0 && p.Y < fb.Height && p.X >= 0 && p.X < fb.Width {
			safeIdx := p.ColorIdx % len(styles)
			if safeIdx < 0 {
				safeIdx += len(styles)
			}
			cellIdx := p.Y*fb.Width + p.X
			cells[cellIdx] = styles[safeIdx].Render(p.Char)
		}
	}

	// Convert grid to string
	var result strings.Builder
	result.Grow(cellCount*2 + fb.Height)
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			result.WriteString(cells[y*fb.Width+x])
		}
		if y < fb.Height-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// RenderFireWave renders a sine-wave fire strip for the given terminal width and animation frame.
func RenderFireWave(width int, frame int) string {
	var result strings.Builder
	styles := fireColorStyles()

	// Create a wave pattern
	for x := 0; x < width; x++ {
		// Blend two sines for gentler, less jittery motion.
		phase := float64(frame) * 0.075
		y := 0.75*math.Sin(float64(x)*0.24+phase) + 0.25*math.Sin(float64(x)*0.11+phase*0.6)

		// Map to fire characters
		var char string
		if y > 0.65 {
			char = "▁"
		} else if y > 0.25 {
			char = "▂"
		} else if y > 0 {
			char = "▃"
		} else if y > -0.25 {
			char = "▄"
		} else if y > -0.65 {
			char = "▅"
		} else {
			char = "▆"
		}

		// Color based on position (gradient)
		if len(styles) == 0 {
			result.WriteString(char)
			continue
		}
		colorIdx := int(float64(x) / float64(width) * float64(len(styles)-1))
		if colorIdx >= len(styles) {
			colorIdx = len(styles) - 1
		}
		result.WriteString(styles[colorIdx].Render(char))
	}

	return result.String()
}

func fireColorStyles() []lipgloss.Style {
	if len(activeFireColors) == 0 {
		return []lipgloss.Style{
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6600")),
		}
	}
	styles := make([]lipgloss.Style, len(activeFireColors))
	for i, color := range activeFireColors {
		styles[i] = lipgloss.NewStyle().Foreground(color)
	}
	return styles
}
