package cmd

import (
	"fmt"

	"github.com/git-fire/git-fire/internal/config"
	"github.com/git-fire/git-fire/internal/flavor"
)

// FlavorQuotesEnabled reports whether flavor text (TUI quote banner and CLI
// motivation lines) should appear. Nil cfg defaults to true so messages still
// show when config has not been loaded yet.
func FlavorQuotesEnabled(cfg *config.Config) bool {
	if cfg == nil {
		return true
	}
	return cfg.UI.ShowStartupQuote
}

var extinguishWaterMessages = []string{
	"Extinguishing complete. Repos are cool, calm, and pushed.",
	"Water deployed. The flames are out and your changes are safe.",
	"Fire contained. Backup branches are soaking in success.",
	"All clear. The blaze is out; your history stays alive.",
	"Containment holds; the blaze is out and the remotes are humming.",
	"Cool and clear. No crossing the streams; just clean mirrors.",
	"Experiment complete. Moisture deployed; nothing left incendiary.",
	"Mission accomplished: cooldown engaged, remotes refreshed.",
}

var failedRunEmberMessages = []string{
	"The fire inside you still burns. This run paused, but your spirit stays lit.",
	"Aborted or blocked, not defeated. The ember remains.",
	"The flames are still alive in you. Catch your breath and try again.",
	"This spark lives on. Regroup, re-run, reignite.",
	"Bonfire low, not out; rest, then push again.",
	"The run paused; your resolve didn't. Breathe and retry.",
	"Checkpoint missed, not deleted; respawn when you're ready.",
	"Another attempt earns respect; light the next run when you're ready.",
}

func printStartupFireQuote() {
	quote := flavor.RandomStartupFireQuote()
	if quote == "" {
		return
	}
	fmt.Printf("🔥 %s\n", quote)
	fmt.Println()
}

func printExtinguishWaterMessage() {
	msg := flavor.PickRandomString(extinguishWaterMessages)
	if msg == "" {
		return
	}
	fmt.Printf("💧 %s\n", msg)
}

func printFailedRunEmberMessage() {
	msg := flavor.PickRandomString(failedRunEmberMessages)
	if msg == "" {
		return
	}
	fmt.Printf("🔥 %s\n", msg)
}
