package ui

import (
	"math"
	"math/rand"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/git-fire/git-fire/internal/config"
)

var fireParticleCharsClassic = [...]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "░", "▒", "▓"}
var fireParticleCharsEmberStorm = [...]string{"·", "•", "░", "▒", "▓", "█", "▆", "▇", "*", "+"}
var fireParticleCharsTorch = [...]string{"▏", "▎", "▍", "▌", "▋", "▊", "▉", "█"}

type fireAnimationStyle int

const (
	fireAnimationClassic fireAnimationStyle = iota
	fireAnimationEmberStorm
	fireAnimationTorch
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
	Style     fireAnimationStyle
}

// NewFireBackground creates a new fire background
func NewFireBackground(width, height int) *FireBackground {
	return NewFireBackgroundWithStyle(width, height, config.UIFireAnimationStyleClassic)
}

// NewFireBackgroundWithStyle creates a fire background for a specific style.
func NewFireBackgroundWithStyle(width, height int, styleName string) *FireBackground {
	bg := &FireBackground{
		Width:     width,
		Height:    height,
		Particles: make([]FireParticle, 0),
		Frame:     0,
		Style:     parseFireAnimationStyle(styleName),
	}

	// Initialize particles
	bg.Reset()

	return bg
}

// SetStyle switches the running animation style and refreshes particles.
func (fb *FireBackground) SetStyle(styleName string) {
	if fb == nil {
		return
	}
	style := parseFireAnimationStyle(styleName)
	if fb.Style == style {
		return
	}
	fb.Style = style
	fb.Frame = 0
	fb.Reset()
}

// Reset reinitializes all particles
func (fb *FireBackground) Reset() {
	fb.Particles = make([]FireParticle, 0)

	// Create fire particles at the bottom
	particleCount := fb.targetParticleCount()
	for i := 0; i < particleCount; i++ {
		fb.spawnParticle()
	}
}

// spawnParticle creates a new fire particle at the bottom
func (fb *FireBackground) spawnParticle() {
	if fb.Width <= 0 || fb.Height <= 0 {
		return
	}
	chars := fb.styleParticleChars()
	if len(chars) == 0 {
		chars = fireParticleCharsClassic[:]
	}
	x := rand.Intn(fb.Width)
	maxAge := maxInt(2, fb.Height+rand.Intn(5))
	switch fb.Style {
	case fireAnimationEmberStorm:
		maxAge = maxInt(2, fb.Height/2+rand.Intn(maxInt(3, fb.Height+3)))
	case fireAnimationTorch:
		center := fb.Width / 2
		spread := maxInt(1, fb.Width/8)
		x = center + rand.Intn(spread*2+1) - spread
		if x < 0 {
			x = 0
		}
		if x >= fb.Width {
			x = fb.Width - 1
		}
		maxAge = maxInt(2, fb.Height+2+rand.Intn(6))
	}

	particle := FireParticle{
		X:        x,
		Y:        fb.Height - 1,
		Char:     chars[rand.Intn(len(chars))],
		ColorIdx: 0, // Start at bottom (red)
		Age:      0,
		MaxAge:   maxAge,
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

		movePeriod := 2
		driftChance := 0.22 + 0.04*math.Sin(float64(fb.Frame)*0.08)
		switch fb.Style {
		case fireAnimationEmberStorm:
			movePeriod = 1
			driftChance = 0.38 + 0.10*math.Sin(float64(fb.Frame)*0.15)
			if fb.Frame%8 == 0 {
				p.X += rand.Intn(5) - 2 // -2..2 gust
			}
		case fireAnimationTorch:
			movePeriod = 2
			driftChance = 0.10 + 0.03*math.Sin(float64(fb.Frame)*0.07)
			// Torch flames stay narrow around center.
			center := fb.Width / 2
			if rand.Float64() < 0.30 {
				if p.X < center {
					p.X++
				} else if p.X > center {
					p.X--
				}
			}
			if rand.Float64() < 0.12 {
				p.Y--
			}
		}
		if movePeriod <= 0 {
			movePeriod = 1
		}
		// Move particle up (fire rises)
		if fb.Frame%movePeriod == 0 {
			p.Y--
		}

		// Drift left/right with style-dependent variation.
		if rand.Float64() < driftChance {
			step := rand.Intn(3) - 1 // -1, 0, or 1
			if fb.Style == fireAnimationEmberStorm && rand.Float64() < 0.25 {
				step = rand.Intn(5) - 2 // -2..2 extra jitter
			}
			p.X += step
		}
		switch fb.Style {
		case fireAnimationEmberStorm:
			if rand.Float64() < 0.28 {
				chars := fb.styleParticleChars()
				if len(chars) > 0 {
					p.Char = chars[rand.Intn(len(chars))]
				}
			}
		case fireAnimationTorch:
			if rand.Float64() < 0.15 {
				chars := fb.styleParticleChars()
				if len(chars) > 0 {
					p.Char = chars[rand.Intn(len(chars))]
				}
			}
		}

		// Keep X in bounds
		if p.X < 0 {
			p.X = 0
		}
		if p.X >= fb.Width {
			p.X = fb.Width - 1
		}

		// Change color as it rises (red -> orange -> yellow)
		progress := float64(p.Age) / float64(maxInt(1, p.MaxAge))
		switch fb.Style {
		case fireAnimationEmberStorm:
			// Faster visible color shift for crackling sparks.
			progress = math.Pow(progress, 0.75)
		case fireAnimationTorch:
			// Slightly slower gradient for a steadier flame core.
			progress = math.Pow(progress, 1.25)
		}
		if progress > 1 {
			progress = 1
		}
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
		targetCount := fb.targetParticleCount()
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
	return RenderFireWaveStyled(width, frame, config.UIFireAnimationStyleClassic)
}

// RenderFireWaveStyled renders a style-specific sine-wave fire strip.
func RenderFireWaveStyled(width int, frame int, styleName string) string {
	if width <= 0 {
		return ""
	}
	var result strings.Builder
	styles := fireColorStyles()
	style := parseFireAnimationStyle(styleName)
	center := float64(maxInt(1, width-1)) / 2.0

	// Create a wave pattern
	for x := 0; x < width; x++ {
		var y float64
		switch style {
		case fireAnimationEmberStorm:
			phase := float64(frame) * 0.12
			y = 0.58*math.Sin(float64(x)*0.42+phase) + 0.42*math.Sin(float64(x)*0.17+phase*1.7)
		case fireAnimationTorch:
			phase := float64(frame) * 0.055
			y = 0.80*math.Sin(float64(x)*0.08+phase) + 0.20*math.Sin(float64(x)*0.03+phase*0.5)
		default:
			// Blend two sines for gentler, less jittery motion.
			phase := float64(frame) * 0.075
			y = 0.75*math.Sin(float64(x)*0.24+phase) + 0.25*math.Sin(float64(x)*0.11+phase*0.6)
		}

		var char string
		switch style {
		case fireAnimationEmberStorm:
			if y > 0.75 {
				char = "▁"
			} else if y > 0.45 {
				char = "▂"
			} else if y > 0.15 {
				char = "▃"
			} else if y > -0.15 {
				char = "▄"
			} else if y > -0.45 {
				char = "▅"
			} else if y > -0.75 {
				char = "▆"
			} else {
				char = "▇"
			}
		case fireAnimationTorch:
			if y > 0.50 {
				char = "▁"
			} else if y > 0.20 {
				char = "▂"
			} else if y > -0.10 {
				char = "▃"
			} else if y > -0.40 {
				char = "▄"
			} else {
				char = "▅"
			}
		default:
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
		}

		// Color based on position (gradient)
		if len(styles) == 0 {
			result.WriteString(char)
			continue
		}
		colorIdx := 0
		switch style {
		case fireAnimationEmberStorm:
			rot := (x + frame) % width
			colorIdx = int(float64(rot) / float64(width) * float64(len(styles)-1))
		case fireAnimationTorch:
			dist := math.Abs(float64(x)-center) / maxFloat64(1, center)
			colorIdx = int((1 - dist) * float64(len(styles)-1))
		default:
			colorIdx = int(float64(x) / float64(width) * float64(len(styles)-1))
		}
		if colorIdx >= len(styles) {
			colorIdx = len(styles) - 1
		}
		if colorIdx < 0 {
			colorIdx = 0
		}
		result.WriteString(styles[colorIdx].Render(char))
	}

	return result.String()
}

func parseFireAnimationStyle(styleName string) fireAnimationStyle {
	switch strings.ToLower(strings.TrimSpace(styleName)) {
	case config.UIFireAnimationStyleEmberStorm:
		return fireAnimationEmberStorm
	case config.UIFireAnimationStyleTorch:
		return fireAnimationTorch
	default:
		return fireAnimationClassic
	}
}

func (fb *FireBackground) styleParticleChars() []string {
	switch fb.Style {
	case fireAnimationEmberStorm:
		return fireParticleCharsEmberStorm[:]
	case fireAnimationTorch:
		return fireParticleCharsTorch[:]
	default:
		return fireParticleCharsClassic[:]
	}
}

func (fb *FireBackground) targetParticleCount() int {
	if fb.Width <= 0 {
		return 0
	}
	switch fb.Style {
	case fireAnimationEmberStorm:
		return fb.Width * 3
	case fireAnimationTorch:
		return maxInt(fb.Width, int(float64(fb.Width)*1.6))
	default:
		return fb.Width * 2
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
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
