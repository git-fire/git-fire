package ui

import (
	"math/rand"
	"sync"
	"time"
)

var startupFireQuotes = []string{
	"Fire walk with me.",
	"A light in the dark provides hope.",
	"We didn't start the fire, but we can back it up.",
	"This is fine. Everything is version controlled.",
	"Even the smallest spark can light the way home.",
	"May your commits be hot and your rollbacks rare.",
	"May the forge be with you.",
	"From ember to branch, every change deserves shelter.",
}

var (
	quoteRNG   = rand.New(rand.NewSource(time.Now().UnixNano()))
	quoteRNGMu sync.Mutex
)

func randomStartupFireQuote() string {
	if len(startupFireQuotes) == 0 {
		return ""
	}
	quoteRNGMu.Lock()
	idx := quoteRNG.Intn(len(startupFireQuotes))
	quoteRNGMu.Unlock()
	return startupFireQuotes[idx]
}
