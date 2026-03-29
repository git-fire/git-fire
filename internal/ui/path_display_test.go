package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAbbreviateUserHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows os.UserHomeDir prefers USERPROFILE

	sub := filepath.Join(home, "projects", "git-fire")
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	// Directory name starts with ".." — must not be treated as outside $HOME.
	dotDotName := filepath.Join(home, "..repo")
	if err := os.MkdirAll(dotDotName, 0o700); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "other", "repo")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	outsideAbs, err := filepath.Abs(outside)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string
	}{
		{home, "~"},
		{sub, "~/projects/git-fire"},
		{dotDotName, "~/..repo"},
		{outside, filepath.ToSlash(outsideAbs)},
	}

	for _, tt := range tests {
		got := AbbreviateUserHome(tt.path)
		if got != tt.want {
			t.Errorf("AbbreviateUserHome(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
