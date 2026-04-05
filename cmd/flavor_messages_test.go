package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/git-fire/git-fire/internal/config"
)

func TestFlavorQuotesEnabled(t *testing.T) {
	if !FlavorQuotesEnabled(nil) {
		t.Fatal("nil cfg should default to enabled")
	}
	cfg := config.DefaultConfig()
	cfg.UI.ShowStartupQuote = false
	if FlavorQuotesEnabled(&cfg) {
		t.Fatal("ShowStartupQuote false should disable flavor quotes")
	}
}

func TestPrintStartupFireQuote_PrintsFirePrefix(t *testing.T) {
	output := captureStdoutFlavor(t, printStartupFireQuote)
	if !strings.Contains(output, "🔥 ") {
		t.Fatalf("expected startup output to include fire prefix, got %q", output)
	}
}

func TestPrintExtinguishWaterMessage_PrintsWaterPrefix(t *testing.T) {
	output := captureStdoutFlavor(t, printExtinguishWaterMessage)
	if !strings.Contains(output, "💧 ") {
		t.Fatalf("expected water output to include water prefix, got %q", output)
	}
}

func TestPrintFailedRunEmberMessage_PrintsFirePrefix(t *testing.T) {
	output := captureStdoutFlavor(t, printFailedRunEmberMessage)
	if !strings.Contains(output, "🔥 ") {
		t.Fatalf("expected failed-run output to include fire prefix, got %q", output)
	}
}

func captureStdoutFlavor(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close write pipe: %v", err)
	}
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to capture stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close read pipe: %v", err)
	}

	return buf.String()
}
