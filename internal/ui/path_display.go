package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-fire/git-harness/git"
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
	// Require an actual parent traversal (.. segment), not a name like "..repo".
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(absPath)
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}

// TruncatePath returns a substring of path starting at the given rune offset, limited
// to maxWidth runes. hasLeft/hasRight indicate whether content is hidden on either side.
// If the whole path fits within maxWidth, it is returned unchanged.
func TruncatePath(path string, maxWidth, offset int) (visible string, hasLeft, hasRight bool) {
	if maxWidth <= 0 {
		return "", false, false
	}
	runes := []rune(path)
	total := len(runes)
	if total <= maxWidth {
		return path, false, false
	}
	maxOffset := total - maxWidth
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	return string(runes[offset : offset+maxWidth]), offset > 0, offset+maxWidth < total
}

// PathWidthFor returns the number of rune columns available for the scrollable path
// portion inside a repo list row, given the current terminal width and the repo's
// other fixed-width fields.
//
// Row layout:   "> [✓] {name} (‹{path}›)  [{mode}] {remotes}{dirty}"
// Fixed chars (non-path): box(6) + "> [✓] " (6) + " (‹" (3) + "›)  [" (5) + "] " (2) + dirty_reserve(4) = 26
func PathWidthFor(windowWidth int, repo git.Repository) int {
	remotesInfo := fmt.Sprintf("(%d remotes)", len(repo.Remotes))
	if len(repo.Remotes) == 0 {
		remotesInfo = "(no remotes!)"
	}
	overhead := 26 + len([]rune(repo.Name)) + len([]rune(repo.Mode.String())) + len([]rune(remotesInfo))
	w := windowWidth - overhead
	if w < 8 {
		w = 8
	}
	return w
}
