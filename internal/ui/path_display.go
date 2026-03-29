package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// AbbreviateUserHome formats an absolute path for display: paths under the
// current user's home directory use a ~/ prefix with forward slashes; all
// other paths are shown as an absolute path (also slash-normalized for display).
func AbbreviateUserHome(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	absPath = filepath.Clean(absPath)

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	homeAbs, err := filepath.Abs(home)
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	homeAbs = filepath.Clean(homeAbs)

	rel, err := filepath.Rel(homeAbs, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(absPath)
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}
