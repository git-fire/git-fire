package cmd

import (
	"os"
	"testing"
)

// xdgEnvKeys are cleared while HOME points at a temp dir so os.UserConfigDir
// and os.UserCacheDir resolve under that HOME. GitHub Actions sets XDG_* on
// the runner, which would otherwise bypass an isolated HOME.
var xdgEnvKeys = []string{
	"XDG_CONFIG_HOME",
	"XDG_CACHE_HOME",
	"XDG_STATE_HOME",
	"XDG_DATA_HOME",
}

// setTestHome sets HOME to tmp and temporarily unsets XDG_* vars. Restores all
// on cleanup.
func setTestHome(t *testing.T, tmp string) {
	t.Helper()
	origHome, hadHome := os.LookupEnv("HOME")
	origXDG := make(map[string]string, len(xdgEnvKeys))
	hadXDG := make(map[string]bool, len(xdgEnvKeys))
	for _, k := range xdgEnvKeys {
		v, ok := os.LookupEnv(k)
		hadXDG[k] = ok
		origXDG[k] = v
	}
	t.Cleanup(func() {
		if hadHome {
			_ = os.Setenv("HOME", origHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
		for _, k := range xdgEnvKeys {
			if hadXDG[k] {
				_ = os.Setenv(k, origXDG[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})
	_ = os.Setenv("HOME", tmp)
	for _, k := range xdgEnvKeys {
		_ = os.Unsetenv(k)
	}
}
