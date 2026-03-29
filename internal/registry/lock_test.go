package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAcquireLock_Basic verifies that acquireLock creates the lock file, returns
// a working release function, and that release removes the lock file.
func TestAcquireLock_Basic(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "repos.toml")
	lockPath := registryPath + ".lock"

	release, err := acquireLock(registryPath)
	if err != nil {
		t.Fatalf("acquireLock() unexpected error: %v", err)
	}

	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("lock file should exist after acquireLock")
	}

	release()

	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("lock file should be removed after release")
	}
}

// TestAcquireLock_CreatesDirectory verifies that acquireLock creates the
// registry directory if it does not exist.
func TestAcquireLock_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	registryPath := filepath.Join(dir, "repos.toml")

	release, err := acquireLock(registryPath)
	if err != nil {
		t.Fatalf("acquireLock() unexpected error on missing directory: %v", err)
	}
	defer release()

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("acquireLock should create the registry directory")
	}
}

// TestAcquireLock_ReleaseAllowsReacquire verifies that after calling release,
// the same lock can be acquired again.
func TestAcquireLock_ReleaseAllowsReacquire(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "repos.toml")

	release, err := acquireLock(registryPath)
	if err != nil {
		t.Fatalf("first acquireLock() error: %v", err)
	}
	release()

	release2, err := acquireLock(registryPath)
	if err != nil {
		t.Fatalf("second acquireLock() after release error: %v", err)
	}
	defer release2()
}

// TestAcquireLock_FailsFast_OnDirectoryError verifies that when the directory
// cannot be created (e.g. a regular file exists where a directory is required),
// acquireLock returns an error immediately without spinning.
func TestAcquireLock_FailsFast_OnDirectoryError(t *testing.T) {
	// Place a regular file where the directory should be, so MkdirAll fails.
	tmp := t.TempDir()
	blockingFile := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	registryPath := filepath.Join(blockingFile, "sub", "repos.toml")

	start := time.Now()
	release, err := acquireLock(registryPath)
	elapsed := time.Since(start)
	defer release()

	if err == nil {
		t.Fatal("acquireLock() should return an error when directory cannot be created")
	}
	// Must not spin for lockTimeout — should fail in well under a second.
	if elapsed > time.Second {
		t.Errorf("acquireLock() should fail fast on directory error, took %v", elapsed)
	}
}

// TestAcquireLock_FailsFast_OnPermissionError verifies that a non-EEXIST error
// from OpenFile (e.g. EACCES on the lock directory) causes immediate failure
// rather than spinning until lockTimeout.
func TestAcquireLock_FailsFast_OnPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission test is not meaningful as root")
	}

	dir := t.TempDir()
	// Make the directory non-writable so OpenFile fails with EACCES.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) }) // restore so t.TempDir cleanup works

	registryPath := filepath.Join(dir, "repos.toml")

	start := time.Now()
	release, err := acquireLock(registryPath)
	elapsed := time.Since(start)
	defer release()

	if err == nil {
		t.Fatal("acquireLock() should return an error on EACCES")
	}
	if elapsed > time.Second {
		t.Errorf("acquireLock() should fail fast on permission error, took %v", elapsed)
	}
}

// TestAcquireLock_Timeout verifies that when a lock file is already present
// (simulating another process holding the lock), acquireLock eventually returns
// a timeout error rather than blocking indefinitely. Skipped in short mode.
func TestAcquireLock_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode (-short)")
	}

	registryPath := filepath.Join(t.TempDir(), "repos.toml")
	lockPath := registryPath + ".lock"

	// Simulate a live process holding the lock by writing the current PID.
	// Using a dead PID would trigger the stale-lock cleanup and let acquireLock
	// succeed immediately, defeating the purpose of this test.
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		t.Fatal(err)
	}
	lockContent := fmt.Sprintf("%d\n", os.Getpid())
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o600); err != nil {
		t.Fatal(err)
	}

	release, err := acquireLock(registryPath)
	defer release()

	if err == nil {
		t.Fatal("acquireLock() should return an error when lock is held for longer than lockTimeout")
	}
}
