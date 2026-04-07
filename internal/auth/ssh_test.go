package auth

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFindSSHKeys(t *testing.T) {
	// Create temp SSH directory
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatalf("Failed to create .ssh dir: %v", err)
	}

	// Set HOME to temp dir
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create fake SSH keys
	keyFiles := []string{"id_rsa", "id_ed25519"}
	for _, keyFile := range keyFiles {
		keyPath := filepath.Join(sshDir, keyFile)
		// Create a minimal fake private key file
		content := "-----BEGIN OPENSSH PRIVATE KEY-----\nfakekey\n-----END OPENSSH PRIVATE KEY-----\n"
		if err := os.WriteFile(keyPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create key file %s: %v", keyFile, err)
		}
	}

	keys, err := FindSSHKeys()
	if err != nil {
		t.Fatalf("FindSSHKeys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	// Check key names
	foundRSA := false
	foundEd25519 := false

	for _, key := range keys {
		if key.Name == "id_rsa" {
			foundRSA = true
			if key.Type != "rsa" {
				t.Errorf("Expected type 'rsa' for id_rsa, got '%s'", key.Type)
			}
		}
		if key.Name == "id_ed25519" {
			foundEd25519 = true
			if key.Type != "ed25519" {
				t.Errorf("Expected type 'ed25519' for id_ed25519, got '%s'", key.Type)
			}
		}
	}

	if !foundRSA {
		t.Error("Expected to find id_rsa key")
	}
	if !foundEd25519 {
		t.Error("Expected to find id_ed25519 key")
	}
}

func TestFindSSHKeys_NoSSHDir(t *testing.T) {
	// Use temp dir with no .ssh directory
	tmpDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	keys, err := FindSSHKeys()
	if err != nil {
		t.Fatalf("FindSSHKeys() error = %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("Expected 0 keys when no .ssh dir exists, got %d", len(keys))
	}
}

func TestCheckSSHAgent(t *testing.T) {
	// Save original env
	originalSocket := os.Getenv("SSH_AUTH_SOCK")
	defer func() {
		if originalSocket != "" {
			os.Setenv("SSH_AUTH_SOCK", originalSocket)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}()

	t.Run("no agent running", func(t *testing.T) {
		os.Unsetenv("SSH_AUTH_SOCK")

		agent, err := CheckSSHAgent()
		if err != nil {
			t.Fatalf("CheckSSHAgent() error = %v", err)
		}

		if agent.Running {
			t.Error("Expected agent.Running to be false when SSH_AUTH_SOCK not set")
		}

		if agent.Socket != "" {
			t.Error("Expected empty socket when agent not running")
		}
	})

	// Note: Testing with actual ssh-agent is environment-dependent
	// In real env, ssh-agent might or might not be running
	// So we only test the "not running" case reliably
}

func TestCheckSSHAgent_ProbeFailureIsNonFatalAndVisible(t *testing.T) {
	tmpBin := t.TempDir()
	if err := writeFakeSshAdd(t, tmpBin, "probe_failure"); err != nil {
		t.Fatalf("write fake ssh-add: %v", err)
	}

	t.Setenv("PATH", tmpBin)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")

	agent, err := CheckSSHAgent()
	if err != nil {
		t.Fatalf("CheckSSHAgent() error = %v", err)
	}
	if !agent.Running {
		t.Fatal("Expected running=true when SSH_AUTH_SOCK is set")
	}
	if agent.Error == "" {
		t.Fatal("Expected non-fatal probe warning when ssh-add exits unexpectedly")
	}
	if agent.KeysKnown {
		t.Fatal("Expected KeysKnown=false when probe fails")
	}
}

func TestCheckSSHAgent_NoKeysExitCodeIsTreatedAsHealthy(t *testing.T) {
	tmpBin := t.TempDir()
	if err := writeFakeSshAdd(t, tmpBin, "no_keys"); err != nil {
		t.Fatalf("write fake ssh-add: %v", err)
	}

	t.Setenv("PATH", tmpBin)
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")

	agent, err := CheckSSHAgent()
	if err != nil {
		t.Fatalf("CheckSSHAgent() error = %v", err)
	}
	if !agent.Running {
		t.Fatal("Expected running=true when SSH_AUTH_SOCK is set")
	}
	if agent.Error != "" {
		t.Fatalf("Expected no warning for no-identities case, got %q", agent.Error)
	}
	if len(agent.Keys) != 0 {
		t.Fatalf("Expected no loaded keys, got %d", len(agent.Keys))
	}
	if !agent.KeysKnown {
		t.Fatal("Expected KeysKnown=true for successful no-identities probe")
	}
}

func TestCheckSSHAgent_MissingSshAddIsVisible(t *testing.T) {
	tmpBin := t.TempDir()
	t.Setenv("PATH", tmpBin) // intentionally empty of ssh-add
	t.Setenv("SSH_AUTH_SOCK", "/tmp/fake.sock")

	agent, err := CheckSSHAgent()
	if err != nil {
		t.Fatalf("CheckSSHAgent() error = %v", err)
	}
	if !agent.Running {
		t.Fatal("Expected running=true when SSH_AUTH_SOCK is set")
	}
	if !strings.Contains(agent.Error, "ssh-add not found") {
		t.Fatalf("Expected missing ssh-add warning, got %q", agent.Error)
	}
	if agent.KeysKnown {
		t.Fatal("Expected KeysKnown=false when ssh-add is missing")
	}
}

func TestFindSSHKeys_FingerprintFailureIsSurfaced(t *testing.T) {
	tmpHome := t.TempDir()
	sshDir := filepath.Join(tmpHome, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}

	keyPath := filepath.Join(sshDir, "id_ed25519")
	// Minimal placeholder key bytes; fingerprint command is expected to fail.
	if err := os.WriteFile(keyPath, []byte("not-a-real-key"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	t.Setenv("HOME", tmpHome)
	t.Setenv("USERPROFILE", tmpHome)
	t.Setenv("PATH", t.TempDir()) // prevent ssh-keygen lookup for deterministic failure

	keys, err := FindSSHKeys()
	if err != nil {
		t.Fatalf("FindSSHKeys() error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("Expected one key, got %d", len(keys))
	}
	if keys[0].Fingerprint != "" {
		t.Fatalf("Expected empty fingerprint on failure, got %q", keys[0].Fingerprint)
	}
	if keys[0].FingerprintError == "" {
		t.Fatal("Expected FingerprintError to be populated when fingerprint probe fails")
	}
}

func writeFakeSshAdd(t *testing.T, dir, mode string) error {
	t.Helper()

	name := "ssh-add"
	var content string
	if runtime.GOOS == "windows" {
		name = "ssh-add.cmd"
		if mode == "no_keys" {
			content = "@echo off\r\necho The agent has no identities.\r\nexit /b 1\r\n"
		} else {
			content = "@echo off\r\necho broken-agent 1>&2\r\nexit /b 2\r\n"
		}
	} else {
		if mode == "no_keys" {
			content = "#!/bin/sh\necho The agent has no identities.\nexit 1\n"
		} else {
			content = "#!/bin/sh\necho broken-agent >&2\nexit 2\n"
		}
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o700)
}

func TestIsKeyEncrypted(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		encrypted bool
	}{
		{
			name: "encrypted traditional key",
			content: `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-128-CBC,12345
-----END RSA PRIVATE KEY-----`,
			encrypted: true,
		},
		{
			name: "key with ENCRYPTED marker",
			content: `-----BEGIN ENCRYPTED PRIVATE KEY-----
MIIBpjBABgkqhkiG9w0BBQ0wMzAbBgkqhkiG9w
-----END ENCRYPTED PRIVATE KEY-----`,
			encrypted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := filepath.Join(tmpDir, "test_key")
			if err := os.WriteFile(keyPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("Failed to write test key: %v", err)
			}

			encrypted, err := IsKeyEncrypted(keyPath)
			if err != nil {
				t.Fatalf("IsKeyEncrypted() error = %v", err)
			}

			if encrypted != tt.encrypted {
				t.Errorf("IsKeyEncrypted() = %v, want %v", encrypted, tt.encrypted)
			}
		})
	}

	// Test with real unencrypted key (only if ssh-keygen available)
	t.Run("real unencrypted key", func(t *testing.T) {
		if _, err := exec.LookPath("ssh-keygen"); err != nil {
			t.Skip("ssh-keygen not available")
		}

		// writeAskpassScript writes under the user cache dir; point HOME at
		// tmpDir so the test is fully self-contained and works in sandboxes.
		origHome := os.Getenv("HOME")
		os.Setenv("HOME", tmpDir)
		defer os.Setenv("HOME", origHome)

		keyPath := filepath.Join(tmpDir, "real_key")
		cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "", "-q")
		if err := cmd.Run(); err != nil {
			t.Skipf("Failed to generate test key: %v", err)
		}

		encrypted, err := IsKeyEncrypted(keyPath)
		if err != nil {
			t.Fatalf("IsKeyEncrypted() error = %v", err)
		}

		if encrypted {
			t.Error("Expected unencrypted key to be detected as not encrypted")
		}
	})
}

func TestSSHStatus_Summary(t *testing.T) {
	status := &SSHStatus{
		AvailableKeys: []SSHKey{
			{
				Name:     "id_rsa",
				Type:     "rsa",
				IsLoaded: true,
			},
			{
				Name:      "id_ed25519",
				Type:      "ed25519",
				IsLoaded:  false,
				NeedsPass: true,
			},
		},
		Agent: &SSHAgent{
			Running: true,
			Socket:  "/tmp/ssh-agent.sock",
			Keys:    []SSHKey{{Name: "id_rsa"}},
		},
	}

	summary := status.Summary()

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// Check for key elements
	if !contains(summary, "ssh-agent running") {
		t.Error("Summary should mention ssh-agent running")
	}

	if !contains(summary, "id_rsa") {
		t.Error("Summary should mention id_rsa key")
	}

	if !contains(summary, "id_ed25519") {
		t.Error("Summary should mention id_ed25519 key")
	}

	if !contains(summary, "needs passphrase") {
		t.Error("Summary should mention keys needing passphrase")
	}
}

func TestGetSSHStatus(t *testing.T) {
	// This test is environment-dependent, so we do basic validation
	// In CI or minimal environments, SSH might not be configured

	status, err := GetSSHStatus()
	if err != nil {
		t.Fatalf("GetSSHStatus() error = %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	// Agent might or might not be running - both are valid
	if status.Agent == nil {
		t.Error("Expected Agent to be set")
	}

	// AvailableKeys might be empty if no SSH keys exist - that's OK
	// Just verify the field exists
	_ = status.AvailableKeys
}

// TestTestPassphrase tests passphrase validation
// This test only runs if ssh-keygen is available and skips if not
func TestTestPassphrase(t *testing.T) {
	// Check if ssh-keygen is available
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available, skipping test")
	}

	tmpDir := t.TempDir()

	// writeAskpassScript writes under the user cache dir; point HOME at
	// tmpDir so the test is fully self-contained and works in sandboxes.
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	keyPath := filepath.Join(tmpDir, "test_key")

	// Generate a test key with passphrase
	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", keyPath, "-N", "testpass", "-q")
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to generate test key: %v", err)
	}

	t.Run("correct passphrase", func(t *testing.T) {
		result := TestPassphrase(keyPath, "testpass")
		if !result {
			t.Error("Expected correct passphrase to validate")
		}
	})

	t.Run("incorrect passphrase", func(t *testing.T) {
		result := TestPassphrase(keyPath, "wrongpass")
		if result {
			t.Error("Expected incorrect passphrase to fail")
		}
	})
}

func TestWriteAskpassScript_UsesUserCacheDir(t *testing.T) {
	xdgCache := filepath.Join(t.TempDir(), "xdg-cache")
	t.Setenv("XDG_CACHE_HOME", xdgCache)

	scriptPath, cleanup, err := writeAskpassScript("secret")
	if err != nil {
		t.Fatalf("writeAskpassScript() error = %v", err)
	}
	defer cleanup()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("os.UserCacheDir() error = %v", err)
	}
	want := filepath.Join(cacheDir, "git-fire")
	if filepath.Dir(scriptPath) != want {
		t.Fatalf("expected script dir under %q, got %q", want, filepath.Dir(scriptPath))
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(strings.ToLower(scriptPath), ".cmd") {
			t.Fatalf("expected windows askpass helper to end with .cmd, got %q", scriptPath)
		}
	} else if !strings.HasSuffix(scriptPath, ".sh") {
		t.Fatalf("expected unix askpass helper to end with .sh, got %q", scriptPath)
	}
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("expected script to exist: %v", err)
	}
}

func TestEscapeForCmdSetP(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "exclamation remains literal",
			in:   "p@ss!",
			want: "p@ss!",
		},
		{
			name: "quote escaped before metacharacter",
			in:   `p"&x`,
			want: `p^"^&x`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeForCmdSetP(tt.in)
			if got != tt.want {
				t.Fatalf("escapeForCmdSetP(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsHelper(s, substr)
}

func containsHelper(s, substr string) bool {
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
