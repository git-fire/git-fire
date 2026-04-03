package ui

import (
	"testing"

	"github.com/git-fire/git-fire/internal/flavor"
)

func TestRandomStartupFireQuote_ReturnsKnownQuote(t *testing.T) {
	for range 25 {
		got := randomStartupFireQuote()
		if got == "" {
			t.Fatal("expected non-empty quote")
		}
		found := false
		for _, q := range flavor.StartupFireQuotes() {
			if got == q {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("quote %q not found in startup flavor quotes", got)
		}
	}
}
