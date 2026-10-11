package cmd

import (
	"os"
	"testing"
)

func TestApplyNonInteractiveCLI_setsEnv(t *testing.T) {
	t.Setenv("GIT_FIRE_NON_INTERACTIVE", "")
	nonInteractive = true
	applyNonInteractiveCLI()
	if os.Getenv("GIT_FIRE_NON_INTERACTIVE") == "" {
		t.Fatal("expected GIT_FIRE_NON_INTERACTIVE to be set")
	}
	nonInteractive = false
}
