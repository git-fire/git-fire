package cmd

import "os"

// applyNonInteractiveCLI applies --non-interactive / --yes before command handlers run.
func applyNonInteractiveCLI() {
	if nonInteractive {
		_ = os.Setenv("GIT_FIRE_NON_INTERACTIVE", "1")
	}
}
