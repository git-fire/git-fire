package ui

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	chars := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "░", "▒", "▓"}

	particle := FireParticle{
		X:        rand.Intn(fb.Width),
		Y:        fb.Height - 1,
		Char:     chars[rand.Intn(len(chars))],
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

		// Drift left/right randomly
		if rand.Float32() < 0.3 {
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

	// Remove dead particles (off screen or too old)
	newParticles := make([]FireParticle, 0, len(fb.Particles))
	for _, p := range fb.Particles {
		if p.Y >= 0 && p.Age < p.MaxAge {
			newParticles = append(newParticles, p)
		}
	}
	fb.Particles = newParticles

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
	// Create 2D grid
	grid := make([][]string, fb.Height)
	for i := range grid {
		grid[i] = make([]string, fb.Width)
		for j := range grid[i] {
			grid[i][j] = " "
		}
	}

	// Place particles on grid
	for _, p := range fb.Particles {
		if p.Y >= 0 && p.Y < fb.Height && p.X >= 0 && p.X < fb.Width {
			// Style the character with fire color
			color := lipgloss.Color("#FF6600")
			if len(activeFireColors) > 0 {
				color = activeFireColors[p.ColorIdx]
			}
			style := lipgloss.NewStyle().Foreground(color)
			grid[p.Y][p.X] = style.Render(p.Char)
		}
	}

	// Convert grid to string
	var result strings.Builder
	for y := 0; y < fb.Height; y++ {
		for x := 0; x < fb.Width; x++ {
			result.WriteString(grid[y][x])
		}
		if y < fb.Height-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// RenderWave creates a sine wave fire effect at the top
func RenderFireWave(width int, frame int) string {
	var result strings.Builder

	// Create a wave pattern
	for x := 0; x < width; x++ {
		// Sine wave calculation
		offset := float64(frame) * 0.1
		y := math.Sin(float64(x)*0.3 + offset)

		// Map to fire characters
		var char string
		if y > 0.7 {
			char = "▁"
		} else if y > 0.3 {
			char = "▂"
		} else if y > 0 {
			char = "▃"
		} else if y > -0.3 {
			char = "▄"
		} else if y > -0.7 {
			char = "▅"
		} else {
			char = "▆"
		}

		// Color based on position (gradient)
		if len(activeFireColors) == 0 {
			result.WriteString(char)
			continue
		}
		colorIdx := int(float64(x) / float64(width) * float64(len(activeFireColors)-1))
		if colorIdx >= len(activeFireColors) {
			colorIdx = len(activeFireColors) - 1
		}

		style := lipgloss.NewStyle().Foreground(activeFireColors[colorIdx])
		result.WriteString(style.Render(char))
	}

	return result.String()
}
