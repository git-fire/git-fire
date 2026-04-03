package flavor

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
	startupQuoteRNG   = rand.New(rand.NewSource(time.Now().UnixNano()))
	startupQuoteRNGMu sync.Mutex
)

func RandomStartupFireQuote() string {
	if len(startupFireQuotes) == 0 {
		return ""
	}
	startupQuoteRNGMu.Lock()
	idx := startupQuoteRNG.Intn(len(startupFireQuotes))
	startupQuoteRNGMu.Unlock()
	return startupFireQuotes[idx]
}

func StartupFireQuotes() []string {
	return append([]string(nil), startupFireQuotes...)
}
