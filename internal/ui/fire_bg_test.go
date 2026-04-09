package ui

import (
	"strings"
	"testing"

	"github.com/git-fire/git-fire/internal/config"
)

// ---- NewFireBackground ----

func TestNewFireBackground_Normal(t *testing.T) {
	fb := NewFireBackground(80, 24)
	if fb.Width != 80 {
		t.Errorf("Width: got %d, want 80", fb.Width)
	}
	if fb.Height != 24 {
		t.Errorf("Height: got %d, want 24", fb.Height)
	}
	if fb.Particles == nil {
		t.Error("Particles should be initialized")
	}
}

func TestNewFireBackground_ZeroWidth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewFireBackground(0, 10) panicked: %v", r)
		}
	}()
	fb := NewFireBackground(0, 10)
	if fb == nil {
		t.Error("expected non-nil FireBackground")
	}
}

func TestNewFireBackground_ZeroHeight_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewFireBackground(10, 0) panicked: %v", r)
		}
	}()
	fb := NewFireBackground(10, 0)
	if fb == nil {
		t.Error("expected non-nil FireBackground")
	}
}

func TestNewFireBackground_ZeroBoth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewFireBackground(0, 0) panicked: %v", r)
		}
	}()
	NewFireBackground(0, 0)
}

func TestNewFireBackground_Negative_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewFireBackground(-5, -5) panicked: %v", r)
		}
	}()
	NewFireBackground(-5, -5)
}

// ---- Reset ----

func TestFireBackground_Reset_ParticleCount(t *testing.T) {
	fb := NewFireBackground(20, 4)
	fb.Reset()

	want := fb.Width * 2
	if len(fb.Particles) != want {
		t.Errorf("Reset() produced %d particles, want %d (Width*2)", len(fb.Particles), want)
	}
}

// ---- Update ----

func TestUpdate_Normal_DoesNotPanic(t *testing.T) {
	fb := NewFireBackground(80, 24)
	for i := 0; i < 10; i++ {
		fb.Update()
	}
}

func TestUpdate_ZeroWidth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Update() on zero-width FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(0, 10)
	fb.Update()
}

func TestUpdate_ZeroHeight_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Update() on zero-height FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(10, 0)
	fb.Update()
}

func TestUpdate_ZeroBoth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Update() on 0x0 FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(0, 0)
	fb.Update()
}

func TestUpdate_IncreasesFrame(t *testing.T) {
	fb := NewFireBackground(20, 10)
	before := fb.Frame
	fb.Update()
	if fb.Frame != before+1 {
		t.Errorf("Frame should increment by 1: got %d, want %d", fb.Frame, before+1)
	}
}

func TestUpdate_MaintainsParticleCount(t *testing.T) {
	fb := NewFireBackground(20, 4)
	want := fb.Width * 2

	for i := 0; i < 20; i++ {
		fb.Update()
	}

	if len(fb.Particles) < want {
		t.Errorf("after 20 updates, particle count = %d, want >= %d", len(fb.Particles), want)
	}
}

// ---- Render ----

func TestRender_Normal_ReturnsString(t *testing.T) {
	fb := NewFireBackground(20, 5)
	out := fb.Render()
	if out == "" {
		t.Error("Render() returned empty string for non-zero dimensions")
	}
	lines := strings.Split(out, "\n")
	if len(lines) != 5 {
		t.Errorf("expected %d lines, got %d", 5, len(lines))
	}
}

func TestRender_ZeroWidth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Render() on zero-width FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(0, 10)
	fb.Render()
}

func TestRender_ZeroHeight_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Render() on zero-height FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(10, 0)
	fb.Render()
}

func TestRender_ZeroBoth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Render() on 0x0 FireBackground panicked: %v", r)
		}
	}()
	fb := NewFireBackground(0, 0)
	fb.Render()
}

func TestRender_AfterUpdates_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Render() after updates panicked: %v", r)
		}
	}()
	fb := NewFireBackground(40, 10)
	for i := 0; i < 20; i++ {
		fb.Update()
	}
	fb.Render()
}

// ---- RenderFireWave ----

func TestRenderFireWave_Normal(t *testing.T) {
	out := RenderFireWave(80, 0)
	if out == "" {
		t.Error("RenderFireWave() returned empty string for width=80")
	}
}

func TestRenderFireWave_ZeroWidth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderFireWave(0, 0) panicked: %v", r)
		}
	}()
	RenderFireWave(0, 0)
}

func TestRenderFireWave_Width1_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderFireWave(1, 0) panicked: %v", r)
		}
	}()
	RenderFireWave(1, 0)
}

func TestRenderFireWave_NegativeWidth_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderFireWave(-1, 0) panicked: %v", r)
		}
	}()
	RenderFireWave(-1, 0)
}

func TestRenderFireWave_DifferentFrames(t *testing.T) {
	a := RenderFireWave(40, 0)
	b := RenderFireWave(40, 10)
	if a == b {
		t.Error("RenderFireWave() frame 0 and frame 10 produced identical output, expected variation")
	}
}

func TestRenderFireWaveStyled_DifferentStyles(t *testing.T) {
	classic := RenderFireWaveStyled(50, 5, config.UIFireAnimationStyleClassic)
	ember := RenderFireWaveStyled(50, 5, config.UIFireAnimationStyleEmberStorm)
	torch := RenderFireWaveStyled(50, 5, config.UIFireAnimationStyleTorch)
	if classic == ember {
		t.Error("classic and ember-storm waves should differ")
	}
	if classic == torch {
		t.Error("classic and torch waves should differ")
	}
	if ember == torch {
		t.Error("ember-storm and torch waves should differ")
	}
}

func TestRenderFireWaveStyled_UnknownStyleFallsBack(t *testing.T) {
	fallback := RenderFireWaveStyled(40, 9, "not-a-style")
	classic := RenderFireWaveStyled(40, 9, config.UIFireAnimationStyleClassic)
	if fallback != classic {
		t.Error("unknown fire style should fall back to classic wave rendering")
	}
}

func TestFireBackground_SetStyle_ResetsAndChangesStyle(t *testing.T) {
	fb := NewFireBackgroundWithStyle(30, 6, config.UIFireAnimationStyleClassic)
	if fb.Style != fireAnimationClassic {
		t.Fatalf("initial style = %v, want classic", fb.Style)
	}
	fb.Update()
	if fb.Frame == 0 {
		t.Fatal("expected frame to advance before style change")
	}
	fb.SetStyle(config.UIFireAnimationStyleTorch)
	if fb.Style != fireAnimationTorch {
		t.Fatalf("style after SetStyle = %v, want torch", fb.Style)
	}
	if fb.Frame != 0 {
		t.Errorf("frame should reset on style change, got %d", fb.Frame)
	}
}

func TestNewFireBackgroundWithStyle_UnknownStyleFallsBack(t *testing.T) {
	fb := NewFireBackgroundWithStyle(20, 4, "unknown-style")
	if fb.Style != fireAnimationClassic {
		t.Fatalf("unknown style should fallback to classic, got %v", fb.Style)
	}
}

func TestRenderFireWave_UsesActiveColorProfile(t *testing.T) {
	prevProfile := activeProfileName
	defer applyColorProfile(prevProfile)

	applyColorProfile(config.UIColorProfileClassic)
	classicFirst := activeFireColors[0]

	applyColorProfile(config.UIColorProfileSynthwave)
	synthwaveFirst := activeFireColors[0]

	if classicFirst == synthwaveFirst {
		t.Error("expected different active fire colors between classic and synthwave profiles")
	}
}
