package flavor

import "testing"

func TestPickRandomString_Empty(t *testing.T) {
	if got := PickRandomString(nil); got != "" {
		t.Fatalf("expected empty string for nil slice, got %q", got)
	}
	if got := PickRandomString([]string{}); got != "" {
		t.Fatalf("expected empty string for empty slice, got %q", got)
	}
}

func TestPickRandomString_ReturnsProvidedValue(t *testing.T) {
	options := []string{"alpha", "beta", "gamma"}
	for range 50 {
		got := PickRandomString(options)
		if got == "" {
			t.Fatal("expected non-empty string")
		}
		found := false
		for _, option := range options {
			if got == option {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("got %q not present in options %v", got, options)
		}
	}
}
