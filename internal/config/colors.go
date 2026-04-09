package config

import (
	"fmt"
	"strings"
)

var defaultCustomFireColors = []string{
	"#FF0000",
	"#FF4500",
	"#FF6600",
	"#FF8C00",
	"#FFA500",
	"#FFB700",
	"#FFD700",
	"#FFFF00",
}

// DefaultCustomFireColors returns the default custom fire palette as uppercase
// #RRGGBB values.
func DefaultCustomFireColors() []string {
	return append([]string(nil), defaultCustomFireColors...)
}

// NormalizeHexColor converts a user-provided color string into #RRGGBB form.
// Supports "#RGB", "RGB", "#RRGGBB", and "RRGGBB".
func NormalizeHexColor(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("empty color")
	}
	if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	switch len(s) {
	case 3:
		s = strings.ToUpper(strings.Repeat(string(s[0]), 2) + strings.Repeat(string(s[1]), 2) + strings.Repeat(string(s[2]), 2))
	case 6:
		s = strings.ToUpper(s)
	default:
		return "", fmt.Errorf("must be 3 or 6 hex digits")
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'A' || r > 'F') {
			return "", fmt.Errorf("contains non-hex character %q", r)
		}
	}
	return "#" + s, nil
}

// NormalizeCustomFireColors validates and normalizes each custom fire color.
func NormalizeCustomFireColors(colors []string) ([]string, error) {
	if len(colors) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(colors))
	for i, color := range colors {
		value, err := NormalizeHexColor(color)
		if err != nil {
			return nil, fmt.Errorf("color %d (%q): %w", i+1, color, err)
		}
		normalized = append(normalized, value)
	}
	return normalized, nil
}
