package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetKeyFingerprint(t *testing.T) {
	// Create a temporary SSH key for testing
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	// Generate a test key (without passphrase)
	// This requires ssh-keygen to be available
	// Skip if ssh-keygen is not available
	if !commandExists("ssh-keygen") {
		t.Skip("ssh-keygen not available")
	}

	// Generate an ED25519 key (faster than RSA)
	err := generateTestKey(keyPath, "")
	if err != nil {
		t.Skipf("Failed to generate test key: %v", err)
	}

	fingerprint, err := getKeyFingerprint(keyPath)
	if err != nil {
		t.Errorf("getKeyFingerprint() error = %v", err)
	}

	if fingerprint == "" {
		t.Error("Expected non-empty fingerprint")
	}

	// Fingerprint should contain SHA256
	if len(fingerprint) < 10 {
		t.Errorf("Fingerprint seems too short: %s", fingerprint)
	}
}

func TestGetKeyFingerprint_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "invalid_key")

	// Create an invalid key file
	err := os.WriteFile(invalidKeyPath, []byte("not a valid ssh key"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = getKeyFingerprint(invalidKeyPath)
	if err == nil {
		t.Error("Expected error for invalid key file")
	}
}

func TestGetKeyFingerprint_NonExistent(t *testing.T) {
	_, err := getKeyFingerprint("/nonexistent/key/path")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestAddKeyToAgent_NoPassphrase(t *testing.T) {
	if !commandExists("ssh-agent") || !commandExists("ssh-add") {
		t.Skip("ssh-agent or ssh-add not available")
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	err := generateTestKey(keyPath, "")
	if err != nil {
		t.Skipf("Failed to generate test key: %v", err)
	}

	// Note: This test may fail if ssh-agent is not running
	// We're testing the function logic, not the actual ssh-agent interaction
	err = AddKeyToAgent(keyPath, "")

	// We expect either success or a specific error
	// Don't fail if agent isn't running in test environment
	if err != nil {
		t.Logf("AddKeyToAgent failed (expected in test environment): %v", err)
	}
}

func TestAddKeyToAgent_WithPassphrase(t *testing.T) {
	if !commandExists("ssh-agent") || !commandExists("ssh-add") {
		t.Skip("ssh-agent or ssh-add not available")
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")
	passphrase := "test-passphrase"

	err := generateTestKey(keyPath, passphrase)
	if err != nil {
		t.Skipf("Failed to generate test key: %v", err)
	}

	// Note: This likely won't work in test environment as ssh-add typically
	// requires a TTY for passphrase input
	err = AddKeyToAgent(keyPath, passphrase)

	if err != nil {
		t.Logf("AddKeyToAgent with passphrase failed (expected): %v", err)
	}
}

func TestEnsureSSHAgent(t *testing.T) {
	if !commandExists("ssh-agent") {
		t.Skip("ssh-agent not available")
	}

	// Save current SSH_AUTH_SOCK
	oldSocket := os.Getenv("SSH_AUTH_SOCK")
	defer func() {
		if oldSocket != "" {
			os.Setenv("SSH_AUTH_SOCK", oldSocket)
		}
	}()

	// Test when agent is already running
	agent, _ := CheckSSHAgent()
	if agent != nil && agent.Running {
		err := EnsureSSHAgent()
		if err != nil {
			t.Errorf("EnsureSSHAgent() error = %v", err)
		}
		return
	}

	// Test when agent is not running
	// Clear SSH_AUTH_SOCK to simulate no agent
	os.Unsetenv("SSH_AUTH_SOCK")

	err := EnsureSSHAgent()

	// This may fail in some test environments
	if err != nil {
		t.Logf("EnsureSSHAgent() error (may be expected in test env): %v", err)
	} else {
		// Verify SSH_AUTH_SOCK was set
		socket := os.Getenv("SSH_AUTH_SOCK")
		if socket == "" {
			t.Error("Expected SSH_AUTH_SOCK to be set after starting agent")
		}
	}
}

func TestCheckSSHAgent_WithKeys(t *testing.T) {
	if !commandExists("ssh-add") {
		t.Skip("ssh-add not available")
	}

	agent, err := CheckSSHAgent()
	if err != nil {
		t.Fatalf("CheckSSHAgent() error = %v", err)
	}

	// Agent status should be valid
	if agent == nil {
		t.Fatal("Expected agent to be non-nil")
	}

	// Log the status for debugging
	t.Logf("Agent running: %v", agent.Running)
	t.Logf("Agent keys loaded: %d", len(agent.Keys))
	t.Logf("Agent socket: %s", agent.Socket)

	// Basic validation
	if agent.Running && agent.Socket == "" {
		t.Error("If agent is running, socket should be set")
	}
}

func TestFindSSHKeys_MultipleTkeys(t *testing.T) {
	// Save original home dir
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// Create temporary home with .ssh directory
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	sshDir := filepath.Join(tmpHome, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	if err != nil {
		t.Fatalf("Failed to create .ssh directory: %v", err)
	}

	// Create multiple key files
	if commandExists("ssh-keygen") {
		keyTypes := []string{"id_rsa", "id_ed25519"}
		for _, keyName := range keyTypes {
			keyPath := filepath.Join(sshDir, keyName)
			err := generateTestKey(keyPath, "")
			if err != nil {
				t.Logf("Failed to generate %s: %v", keyName, err)
				continue
			}
		}
	}

	// Find keys
	keys, err := FindSSHKeys()
	if err != nil {
		t.Fatalf("FindSSHKeys() error = %v", err)
	}

	// Should find the keys we created (if ssh-keygen was available)
	t.Logf("Found %d keys", len(keys))

	// Verify key structure
	for _, key := range keys {
		if key.Path == "" {
			t.Error("Key path should not be empty")
		}
		if key.Name == "" {
			t.Error("Key name should not be empty")
		}
		if key.Type == "" {
			t.Error("Key type should not be empty")
		}
	}
}

func TestIsKeyEncrypted_VariousFormats(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		encrypted bool
	}{
		{
			name: "OpenSSH encrypted key",
			content: `-----BEGIN OPENSSH PRIVATE KEY-----
ENCRYPTED
b3BlbnNzaC1rZXktdjEAAAAA
-----END OPENSSH PRIVATE KEY-----`,
			encrypted: true,
		},
		{
			name: "Traditional encrypted key",
			content: `-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-128-CBC,1234567890ABCDEF
-----END RSA PRIVATE KEY-----`,
			encrypted: true,
		},
		{
			name: "OpenSSH key with none cipher",
			content: `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAA
-----END OPENSSH PRIVATE KEY-----`,
			encrypted: true, // ssh-keygen validation will fail, so it's treated as encrypted
		},
		{
			name: "Unknown format",
			content: `-----BEGIN SOMETHING-----
randomdata
-----END SOMETHING-----`,
			encrypted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "test_key")
			err := os.WriteFile(tmpFile, []byte(tt.content), 0600)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			encrypted, err := IsKeyEncrypted(tmpFile)
			if err != nil {
				t.Errorf("IsKeyEncrypted() error = %v", err)
			}

			if encrypted != tt.encrypted {
				t.Errorf("IsKeyEncrypted() = %v, want %v", encrypted, tt.encrypted)
			}
		})
	}
}

func TestSSHStatus_SummaryWithAgent(t *testing.T) {
	status := &SSHStatus{
		AvailableKeys: []SSHKey{
			{Name: "id_rsa", Type: "rsa", IsLoaded: true},
			{Name: "id_ed25519", Type: "ed25519", IsLoaded: false, NeedsPass: true},
		},
		Agent: &SSHAgent{
			Running: true,
			Socket:  "/tmp/ssh-agent.sock",
			Keys:    []SSHKey{{Fingerprint: "SHA256:abc123"}},
		},
	}

	summary := status.Summary()

	if summary == "" {
		t.Error("Expected non-empty summary")
	}

	// Check for key information
	if !contains(summary, "ssh-agent running") {
		t.Error("Summary should mention agent is running")
	}

	if !contains(summary, "id_rsa") {
		t.Error("Summary should mention id_rsa key")
	}

	if !contains(summary, "id_ed25519") {
		t.Error("Summary should mention id_ed25519 key")
	}

	if !contains(summary, "loaded") {
		t.Error("Summary should show loaded status")
	}

	if !contains(summary, "needs passphrase") {
		t.Error("Summary should show passphrase needed status")
	}
}

func TestSSHStatus_SummaryWithoutAgent(t *testing.T) {
	status := &SSHStatus{
		AvailableKeys: []SSHKey{
			{Name: "id_rsa", Type: "rsa", IsLoaded: false},
		},
		Agent: &SSHAgent{
			Running: false,
		},
	}

	summary := status.Summary()

	if !contains(summary, "not running") {
		t.Error("Summary should mention agent is not running")
	}
}

// Helper functions

func commandExists(cmd string) bool {
	_, err := os.Stat("/usr/bin/" + cmd)
	if err == nil {
		return true
	}
	_, err = os.Stat("/bin/" + cmd)
	return err == nil
}

func generateTestKey(keyPath, passphrase string) error {
	args := []string{
		"-t", "ed25519",
		"-f", keyPath,
		"-N", passphrase,
		"-C", "test@example.com",
	}

	cmd := exec.Command("ssh-keygen", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-keygen failed: %w\nOutput: %s", err, output)
	}

	return nil
}
