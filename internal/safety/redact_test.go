package safety

import "testing"

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		notWant  string
		wantPart string
	}{
		{
			name:     "url credentials",
			in:       "https://user:pass@github.com/org/repo.git",
			notWant:  "pass",
			wantPart: "[REDACTED]",
		},
		{
			name:     "key value",
			in:       "API_KEY=supersecret",
			notWant:  "supersecret",
			wantPart: "API_KEY=",
		},
		{
			name:     "github token",
			in:       "ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			notWant:  "ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			wantPart: "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeText(tt.in)
			if tt.notWant != "" && got == tt.in {
				t.Fatalf("expected sanitization to modify input")
			}
			if tt.notWant != "" && contains(got, tt.notWant) {
				t.Fatalf("sanitized output still contains secret %q", tt.notWant)
			}
			if tt.wantPart != "" && !contains(got, tt.wantPart) {
				t.Fatalf("sanitized output missing expected marker %q: %s", tt.wantPart, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
