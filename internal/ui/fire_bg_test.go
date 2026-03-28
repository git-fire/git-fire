package ui

import (
	"strings"
	"testing"
)

func TestNewFireBackground(t *testing.T) {
	fb := NewFireBackground(40, 5)

	if fb.Width != 40 {
		t.Errorf("Width = %d, want 40", fb.Width)
	}
	if fb.Height != 5 {
		t.Errorf("Height = %d, want 5", fb.Height)
	}
	if fb.Frame != 0 {
		t.Errorf("Frame = %d, want 0", fb.Frame)
	}
	// Reset() is called during New, so particles should be seeded.
	if len(fb.Particles) == 0 {
		t.Error("expected particles after NewFireBackground, got none")
	}
}

func TestFireBackground_Reset(t *testing.T) {
	fb := NewFireBackground(20, 4)
	fb.Reset()

	want := fb.Width * 2
	if len(fb.Particles) != want {
		t.Errorf("Reset() produced %d particles, want %d (Width*2)", len(fb.Particles), want)
	}
}

func TestFireBackground_Update_IncrementsFrame(t *testing.T) {
	fb := NewFireBackground(20, 4)
	fb.Frame = 0
	fb.Update()

	if fb.Frame != 1 {
		t.Errorf("Frame after one Update() = %d, want 1", fb.Frame)
	}
}

func TestFireBackground_Update_MaintainsParticleCount(t *testing.T) {
	fb := NewFireBackground(20, 4)
	want := fb.Width * 2

	// Run several frames; particle count should stay at or above target.
	for i := 0; i < 20; i++ {
		fb.Update()
	}

	if len(fb.Particles) < want {
		t.Errorf("after 20 updates, particle count = %d, want >= %d", len(fb.Particles), want)
	}
}

func TestFireBackground_Render_LineCount(t *testing.T) {
	height := 5
	fb := NewFireBackground(30, height)
	output := fb.Render()

	// Render produces Height rows joined by newlines: Height-1 newlines total.
	lines := strings.Split(output, "\n")
	if len(lines) != height {
		t.Errorf("Render() produced %d lines, want %d", len(lines), height)
	}
}

func TestFireBackground_Render_NonEmpty(t *testing.T) {
	fb := NewFireBackground(10, 3)
	output := fb.Render()
	if output == "" {
		t.Error("Render() returned empty string")
	}
}

func TestFireBackground_BoundaryDimensions_NoPanic(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		height int
	}{
		{name: "zero_zero", width: 0, height: 0},
		{name: "zero_height", width: 20, height: 0},
		{name: "zero_width", width: 0, height: 4},
		{name: "negative_width", width: -1, height: 4},
		{name: "negative_height", width: 20, height: -1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic for width=%d height=%d: %v", tc.width, tc.height, r)
				}
			}()
			fb := NewFireBackground(tc.width, tc.height)
			fb.Update()
			_ = fb.Render()
		})
	}
}

func TestRenderFireWave_NonEmpty(t *testing.T) {
	output := RenderFireWave(40, 0)
	if output == "" {
		t.Error("RenderFireWave() returned empty string")
	}
}

func TestRenderFireWave_DifferentFrames(t *testing.T) {
	// Different frame values should produce different output (the wave shifts).
	a := RenderFireWave(40, 0)
	b := RenderFireWave(40, 10)
	if a == b {
		t.Error("RenderFireWave() frame 0 and frame 10 produced identical output, expected variation")
	}
}

func TestRenderFireWave_ZeroWidth(t *testing.T) {
	// Should not panic.
	output := RenderFireWave(0, 0)
	if output != "" {
		t.Errorf("RenderFireWave(0, 0) = %q, want empty string", output)
	}
}
