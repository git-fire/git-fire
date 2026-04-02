package ui

import "testing"

func TestRandomStartupFireQuote_ReturnsKnownQuote(t *testing.T) {
	for range 25 {
		got := randomStartupFireQuote()
		if got == "" {
			t.Fatal("expected non-empty quote")
		}
		found := false
		for _, q := range startupFireQuotes {
			if got == q {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("quote %q not found in startupFireQuotes", got)
		}
	}
}
