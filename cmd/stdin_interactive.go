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
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	return isatty.IsTerminal(os.Stdin.Fd())
}
