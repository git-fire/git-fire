package config

import "testing"

func TestNormalizeHexColor(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "full with hash", input: "#ff6600", want: "#FF6600"},
		{name: "full without hash", input: "00aa11", want: "#00AA11"},
		{name: "short with hash", input: "#f60", want: "#FF6600"},
		{name: "short without hash", input: "abc", want: "#AABBCC"},
		{name: "trimmed", input: "  #0f0  ", want: "#00FF00"},
		{name: "bad length", input: "#abcd", wantErr: true},
		{name: "bad char", input: "#gg0000", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeHexColor(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeHexColor(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("NormalizeHexColor(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeCustomFireColors(t *testing.T) {
	got, err := NormalizeCustomFireColors([]string{"#f60", "00aa11", "  #ABCDEF "})
	if err != nil {
		t.Fatalf("NormalizeCustomFireColors() unexpected error: %v", err)
	}
	want := []string{"#FF6600", "#00AA11", "#ABCDEF"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeCustomFireColors() len=%d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeCustomFireColors()[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeCustomFireColors_Invalid(t *testing.T) {
	_, err := NormalizeCustomFireColors([]string{"#ff6600", "bad!"})
	if err == nil {
		t.Fatal("NormalizeCustomFireColors() expected error for invalid color")
	}
}
