package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// pkgMu serialises Load/Save calls within a single process. The cross-process
// case is handled by the lock file written in acquireLock.
var pkgMu sync.Mutex

// acquireLock creates an exclusive per-file lock using O_CREATE|O_EXCL so that
// only one git-fire instance modifies the registry at a time. It spins for up
// to lockTimeout before returning an error. The caller must invoke the returned
// release function (even on error) to clean up.
const lockTimeout = 5 * time.Second
const lockPollInterval = 50 * time.Millisecond

func acquireLock(registryPath string) (release func(), err error) {
	lockPath := registryPath + ".lock"

	// Ensure the directory exists before attempting to create the lock file.
	// Without this, os.OpenFile returns ENOENT on first run, which would spin
	// for the full lockTimeout before giving up.
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return func() {}, fmt.Errorf("creating registry lock directory: %w", err)
	}

	deadline := time.Now().Add(lockTimeout)

	for {
		f, createErr := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if createErr == nil {
			// Write PID so the lock is identifiable in debugging.
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(lockPath) }, nil
		}

		// Only EEXIST means another process holds the lock — anything else
		// (EACCES, ENOENT after MkdirAll, etc.) is a hard failure.
		if !os.IsExist(createErr) {
			return func() {}, fmt.Errorf("acquiring registry lock: %w", createErr)
		}

		if time.Now().After(deadline) {
			return func() {}, fmt.Errorf("timed out waiting for registry lock %s", lockPath)
		}

		time.Sleep(lockPollInterval)
	}
}
