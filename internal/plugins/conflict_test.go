package plugins

import "testing"

func TestParseConflictResolvedLine(t *testing.T) {
	tests := []struct {
		stdout string
		want   bool
	}{
		{"", false},
		{"true", true},
		{"TRUE", true},
		{"yes", true},
		{"1", true},
		{"resolved", true},
		{"false", false},
		{"no", false},
		{"0", false},
		{"unresolved", false},
		{"maybe", false},
		{"true\nextra", true},
		{"  true  ", true},
	}
	for _, tt := range tests {
		if got := ParseConflictResolvedLine(tt.stdout); got != tt.want {
			t.Errorf("ParseConflictResolvedLine(%q) = %v, want %v", tt.stdout, got, tt.want)
		}
	}
}
