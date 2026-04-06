package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLI_InitHonorsConfigFlag builds the real binary and verifies --init --config
// writes the example file to the requested path (regression for INIT-001).
func TestCLI_InitHonorsConfigFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess build in -short")
	}
	repoRoot := moduleRootDir(t)
	origHome := os.Getenv("HOME")
	if origHome == "" {
		var err error
		origHome, err = os.UserHomeDir()
		if err != nil {
			t.Fatalf("HOME / UserHomeDir: %v", err)
		}
	}
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	binDir := t.TempDir()
	binName := "git-fire"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(binDir, binName)

	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = repoRoot
	// Restore real HOME so the module cache is not written under tmpHome (read-only
	// cache entries break t.TempDir cleanup).
	build.Env = append(os.Environ(), "HOME="+origHome)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	customCfg := filepath.Join(tmpHome, "nested", "project-fire.toml")
	runInit := exec.Command(binPath, "--init", "--force", "--config", customCfg)
	runInit.Env = append(os.Environ(), "HOME="+tmpHome)
	if out, err := runInit.CombinedOutput(); err != nil {
		t.Fatalf("git-fire --init --force --config: %v\n%s", err, out)
	}

	data, err := os.ReadFile(customCfg)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "[global]") {
		preview := string(data)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		t.Fatalf("config missing [global], got: %q", preview)
	}

	// Default user config must not appear when only custom --config was used.
	defaultPath := filepath.Join(tmpHome, ".config", "git-fire", "config.toml")
	if _, err := os.Stat(defaultPath); err == nil {
		t.Fatalf("unexpected default config at %s (should only use --config path)", defaultPath)
	}
}

func moduleRootDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// This file lives in cmd/; module root is parent.
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}
