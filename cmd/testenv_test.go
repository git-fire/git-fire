package cmd

import (
	"path/filepath"
	"testing"
)

// setTestHome isolates all user-dir env vars so UserConfigDir/UserCacheDir stay
// under a temp directory across platforms.
func setTestHome(t *testing.T, tmp string) {
	t.Helper()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, ".local", "state"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
	t.Setenv("APPDATA", filepath.Join(tmp, ".config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmp, ".cache"))
}
