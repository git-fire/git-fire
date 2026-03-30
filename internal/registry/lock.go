package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// pkgMu serialises all registry operations within a single process — both
// file I/O (Load/Save) and in-memory mutations (Upsert, SetStatus, etc.).
// The cross-process case is handled by the lock file written in acquireLock.
var pkgMu sync.RWMutex

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

		// If the owning process is gone (killed, OOM) the lock is stale.
		// Break it immediately and retry rather than spinning for lockTimeout.
		if staleLock(lockPath) {
			fmt.Fprintf(os.Stderr, "⚠️  WARNING: removing stale registry lock (owner process is gone): %s\n", lockPath)
			os.Remove(lockPath)
			continue
		}

		if time.Now().After(deadline) {
			return func() {}, fmt.Errorf("timed out waiting for registry lock %s", lockPath)
		}

		time.Sleep(lockPollInterval)
	}
}

// staleLock reports whether lockPath was written by a process that no longer
// exists. Returns false on any parse or system error (safe default: assume live).
// Stale-lock detection works on Unix; on Windows Signal(0) returns a different
// error code that will not match ESRCH, so the function safely returns false.
func staleLock(lockPath string) bool {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false
	}
	var pid int
	if _, err := fmt.Sscan(string(data), &pid); err != nil || pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	// Signal(0) probes liveness without delivering a real signal.
	// ESRCH ("no such process") confirms the owner is gone.
	err = proc.Signal(syscall.Signal(0))
	return errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone)
}
