package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPickRandomMessage_Empty(t *testing.T) {
	if got := pickRandomMessage(nil); got != "" {
		t.Fatalf("expected empty string for nil slice, got %q", got)
	}
	if got := pickRandomMessage([]string{}); got != "" {
		t.Fatalf("expected empty string for empty slice, got %q", got)
	}
}

func TestPickRandomMessage_ReturnsProvidedValue(t *testing.T) {
	options := []string{"alpha", "beta", "gamma"}
	for range 50 {
		got := pickRandomMessage(options)
		if got == "" {
			t.Fatal("expected non-empty random message")
		}
		found := false
		for _, option := range options {
			if got == option {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("got message %q not present in options %v", got, options)
		}
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
