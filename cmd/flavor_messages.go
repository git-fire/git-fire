package cmd

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/git-fire/git-fire/internal/flavor"
)

var extinguishWaterMessages = []string{
	"Extinguishing complete. Repos are cool, calm, and pushed.",
	"Water deployed. The flames are out and your changes are safe.",
	"Fire contained. Backup branches are soaking in success.",
	"All clear. The blaze is out; your history stays alive.",
}

var failedRunEmberMessages = []string{
	"The fire inside you still burns. This run paused, but your spirit stays lit.",
	"Aborted or blocked, not defeated. The ember remains.",
	"The flames are still alive in you. Catch your breath and try again.",
	"This spark lives on. Regroup, re-run, reignite.",
}

var (
	messageRNG   = rand.New(rand.NewSource(time.Now().UnixNano()))
	messageRNGMu sync.Mutex
)

func pickRandomMessage(messages []string) string {
	if len(messages) == 0 {
		return ""
	}
	messageRNGMu.Lock()
	idx := messageRNG.Intn(len(messages))
	messageRNGMu.Unlock()
	return messages[idx]
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
	msg := pickRandomMessage(extinguishWaterMessages)
	if msg == "" {
		return
	}
	fmt.Printf("💧 %s\n", msg)
}

func printFailedRunEmberMessage() {
	msg := pickRandomMessage(failedRunEmberMessages)
	if msg == "" {
		return
	}
	fmt.Printf("🔥 %s\n", msg)
}
