package safety

import (
	"strings"
	"testing"
)

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
		{
			name:     "url without credentials",
			in:       "https://github.com/org/repo.git",
			wantPart: "https://github.com/org/repo.git",
		},
		{
			name:     "gitlab pat",
			in:       "glpat-abcdefghijklmnopqrstuvwxyz",
			notWant:  "glpat-abcdefghijklmnopqrstuvwxyz",
			wantPart: "[REDACTED]",
		},
		{
			name:     "near-miss AKIA 15 chars",
			in:       "AKIA1234567890123",
			wantPart: "AKIA1234567890123",
		},
		{
			name:     "no secrets passthrough",
			in:       "hello world",
			wantPart: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeText(tt.in)
			if tt.notWant != "" && got == tt.in {
				t.Fatalf("expected sanitization to modify input")
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Fatalf("sanitized output still contains secret %q", tt.notWant)
			}
			if tt.wantPart != "" && !strings.Contains(got, tt.wantPart) {
				t.Fatalf("sanitized output missing expected marker %q: %s", tt.wantPart, got)
			}
		})
	}
}
