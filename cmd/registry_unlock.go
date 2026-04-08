package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/git-fire/git-fire/internal/registry"
	"github.com/git-fire/git-fire/internal/safety"
)

// maybeOfferRegistryUnlock handles a leftover repos.toml.lock before registry I/O.
// When forceUnlockRegistry is true, an existing lock file is removed after a warning.
func maybeOfferRegistryUnlock(regPath string) error {
	if regPath == "" {
		return nil
	}

	if forceUnlockRegistry {
		lockPath := registry.LockPath(regPath)
		if _, err := os.Stat(lockPath); err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("registry lock: %w", err)
		}
		info, _ := registry.ReadLockFile(regPath)
		fmt.Fprintf(os.Stderr, "warning: removing registry lock file (--force-unlock-registry): %s\n", lockPath)
		if info != nil {
			if info.OwnerAppearsAlive && info.PID > 0 {
				fmt.Fprintf(os.Stderr, "warning: lock listed PID %d, which still appears to be running; another git-fire may be active.\n", info.PID)
				fmt.Fprintf(os.Stderr, "warning: removing the lock can corrupt the registry if that process is still using it.\n")
			} else if info.PID > 0 {
				fmt.Fprintf(os.Stderr, "warning: lock listed PID %d; only remove this if no other git-fire is running.\n", info.PID)
			}
		}
		if err := registry.RemoveLockFile(regPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing registry lock: %w", err)
		}
		return nil
	}

	info, err := registry.ReadLockFile(regPath)
	if err != nil {
		return fmt.Errorf("registry lock: %w", err)
	}
	if info == nil {
		return nil
	}

	if !info.OwnerAppearsAlive {
		fmt.Fprintf(os.Stderr, "warning: removing stale registry lock (owner process is gone): %s\n", info.LockPath)
		if err := registry.RemoveLockFile(regPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing stale registry lock: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nRegistry lock file is present:\n  %s\n", info.LockPath)
	fmt.Fprintf(os.Stderr, "This usually means another git-fire is running, or a previous run exited uncleanly (e.g. Ctrl+C).\n")
	if info.PID > 0 {
		fmt.Fprintf(os.Stderr, "Lock owner PID: %d (still appears to be running).\n", info.PID)
	}
	fmt.Fprintf(os.Stderr, "\nRemoving the lock while another instance is active can corrupt your repo registry.\n")
	fmt.Fprintf(os.Stderr, "If you are sure no other git-fire is running, you can remove the lock and continue.\n\n")

	if stat, err := os.Stdin.Stat(); err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return fmt.Errorf("registry is locked; pass --force-unlock-registry to remove %s non-interactively (only if no other git-fire is running)", info.LockPath)
	}

	fmt.Print("Remove lock and continue? [y/N]: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading confirmation: %w", err)
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		if err := registry.RemoveLockFile(regPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing registry lock: %w", err)
		}
		fmt.Fprintln(os.Stderr, "Lock removed.")
		return nil
	default:
		return fmt.Errorf("aborted: registry lock still present at %s", safety.SanitizeText(info.LockPath))
	}
}
