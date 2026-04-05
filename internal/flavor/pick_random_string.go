package flavor

import (
	"math/rand"
	"sync"
	"time"
)

var (
	choiceRNG   = rand.New(rand.NewSource(time.Now().UnixNano()))
	choiceRNGMu sync.Mutex
)

// PickRandomString returns a random element from s, or "" if s is nil or empty.
func PickRandomString(s []string) string {
	if len(s) == 0 {
		return ""
	}
	choiceRNGMu.Lock()
	idx := choiceRNG.Intn(len(s))
	choiceRNGMu.Unlock()
	return s[idx]
}
