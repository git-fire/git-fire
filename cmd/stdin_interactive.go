package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
)

// stdinInteractiveOK is true when stdin is suitable for blocking prompts
// (ReadString, Scanln). Always false under CI / known automation env vars, and
// when GIT_FIRE_NON_INTERACTIVE is set — some runners expose a pseudo-TTY
// where ModeCharDevice and isatty are both true but no human is reading stdin.
func stdinInteractiveOK() bool {
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return false
	}
	if os.Getenv("GIT_FIRE_NON_INTERACTIVE") != "" {
		return false
	}
	if _, err := os.Stdin.Stat(); err != nil {
		return false
	}
	fd := os.Stdin.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
