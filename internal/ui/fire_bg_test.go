package ui

import (
	"strings"
	"testing"
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
