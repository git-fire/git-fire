package usb

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const LockFileName = ".git-fire.lock"

func AcquireTargetLock(targetRoot string, staleAfter time.Duration) (func(), error) {
	lockPath := filepath.Join(targetRoot, LockFileName)
	now := time.Now().UTC()
	content := fmt.Sprintf("%d\n%s\n", os.Getpid(), now.Format(time.RFC3339))
	if err := writeLockFileExclusive(lockPath, []byte(content)); err == nil {
		return func() { _ = os.Remove(lockPath) }, nil
	}

	existing, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to acquire usb lock at %s", lockPath)
	}
	lines := strings.Split(strings.TrimSpace(string(existing)), "\n")
	if len(lines) > 1 {
		if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(lines[1])); err == nil {
			if staleAfter > 0 && time.Since(ts) > staleAfter {
				_ = os.Remove(lockPath)
				if err := writeLockFileExclusive(lockPath, []byte(content)); err == nil {
					return func() { _ = os.Remove(lockPath) }, nil
				}
			}
		}
	}
	pid := "unknown"
	if len(lines) > 0 {
		if _, err := strconv.Atoi(strings.TrimSpace(lines[0])); err == nil {
			pid = strings.TrimSpace(lines[0])
		}
	}
	return nil, fmt.Errorf("usb target is locked by pid %s (%s)", pid, lockPath)
}

func writeLockFileExclusive(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}
