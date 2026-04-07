package auth

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SSHKey represents an SSH key found on the system
type SSHKey struct {
	Path        string // Full path to key file
	Name        string // Key name (e.g., "id_rsa")
	Type        string // Key type: rsa, dsa, ecdsa, ed25519
	IsLoaded    bool   // Is it loaded in ssh-agent?
	NeedsPass   bool   // Does it require a passphrase?
	Fingerprint string // SSH key fingerprint
	// FingerprintError captures non-fatal fingerprint probe errors (e.g. missing
	// ssh-keygen). We keep key discovery functional while surfacing ambiguity.
	FingerprintError string
}

// SSHAgent represents the SSH agent status
type SSHAgent struct {
	Running bool   // Is ssh-agent running?
	Socket  string // SSH_AUTH_SOCK path
	// Keys loaded in agent. Inspect only when KeysKnown is true.
	Keys []SSHKey
	// KeysKnown is true only when ssh-add probing completed successfully.
	// When false, loaded-key status is unknown and callers should not infer
	// "no keys loaded" from an empty Keys slice.
	KeysKnown bool
	// Error records non-fatal status probe failures when SSH_AUTH_SOCK is set
	// but ssh-add cannot be executed or parsed reliably.
	Error string
}

// FindSSHKeys discovers SSH keys in ~/.ssh/
func FindSSHKeys() ([]SSHKey, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	sshDir := filepath.Join(home, ".ssh")
	if _, err := os.Stat(sshDir); os.IsNotExist(err) {
		return []SSHKey{}, nil // No .ssh directory
	}

	// Look for common SSH key files
	keyNames := []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ed25519"}
	var keys []SSHKey

	for _, name := range keyNames {
		keyPath := filepath.Join(sshDir, name)
		if _, err := os.Stat(keyPath); err == nil {
			// Key file exists
			keyType := strings.TrimPrefix(name, "id_")

			key := SSHKey{
				Path: keyPath,
				Name: name,
				Type: keyType,
			}

			// Fingerprint probe failures are non-fatal for key discovery, but they
			// reduce confidence in loaded-key matching, so expose the failure.
			fingerprint, fpErr := getKeyFingerprint(keyPath)
			if fpErr != nil {
				key.FingerprintError = fpErr.Error()
			} else {
				key.Fingerprint = fingerprint
			}

			keys = append(keys, key)
		}
	}

	return keys, nil
}

// getKeyFingerprint gets the fingerprint of an SSH key
func getKeyFingerprint(keyPath string) (string, error) {
	cmd := exec.Command("ssh-keygen", "-l", "-f", keyPath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Output format: "2048 SHA256:abc123... user@host (RSA)"
	// We want the SHA256 part
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}

	return strings.TrimSpace(string(output)), nil
}

// CheckSSHAgent checks if ssh-agent is running and what keys are loaded
func CheckSSHAgent() (*SSHAgent, error) {
	agent := &SSHAgent{
		Keys: []SSHKey{},
	}

	// Check if SSH_AUTH_SOCK is set
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		agent.Running = false
		return agent, nil
	}

	agent.Socket = socket
	agent.Running = true

	sshAddPath, pathErr := exec.LookPath("ssh-add")
	if pathErr != nil {
		agent.Error = "ssh-add not found on PATH"
		return agent, nil
	}

	// List keys in agent
	cmd := exec.Command(sshAddPath, "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// ssh-add returns exit code 1 when agent is reachable but no keys loaded.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			lowerOutput := strings.ToLower(strings.TrimSpace(string(output)))
			if lowerOutput == "" ||
				strings.Contains(lowerOutput, "no identities") ||
				strings.Contains(lowerOutput, "the agent has no identities") {
				agent.KeysKnown = true
				return agent, nil
			}
		}

		agent.Error = fmt.Sprintf("ssh-add -l failed: %v", err)
		return agent, nil
	}

	// Parse output
	agent.KeysKnown = true
	// Format: "2048 SHA256:abc123... user@host (RSA)"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			fingerprint := parts[1]
			keyType := ""
			if len(parts) >= 4 {
				// Extract type from "(RSA)" format
				typeStr := parts[len(parts)-1]
				keyType = strings.Trim(typeStr, "()")
			}

			agent.Keys = append(agent.Keys, SSHKey{
				Fingerprint: fingerprint,
				Type:        strings.ToLower(keyType),
				IsLoaded:    true,
			})
		}
	}

	return agent, nil
}

// AddKeyToAgent adds an SSH key to ssh-agent with optional passphrase
func AddKeyToAgent(keyPath, passphrase string) error {
	// If passphrase is provided, use expect/stdin method
	if passphrase != "" {
		return addKeyWithPassphrase(keyPath, passphrase)
	}

	// No passphrase - try direct add
	cmd := exec.Command("ssh-add", keyPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh-add failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// addKeyWithPassphrase adds a key using stdin for passphrase
func addKeyWithPassphrase(keyPath, passphrase string) error {
	// Use expect or similar tool if available, otherwise manual method
	// For now, use a simple approach with SSH_ASKPASS
	// This is a simplified version - production might need expect or similar

	cmd := exec.Command("ssh-add", keyPath)

	// Try to pipe passphrase via stdin
	// Note: This might not work on all systems, as ssh-add often reads from TTY
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ssh-add: %w", err)
	}

	// Write passphrase
	if _, err := stdin.Write([]byte(passphrase + "\n")); err != nil {
		return fmt.Errorf("failed to write passphrase: %w", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ssh-add failed: %w", err)
	}

	return nil
}

// writeAskpassScript creates a temporary SSH_ASKPASS helper script in the
// user cache dir (git-fire subdirectory) that outputs passphrase when invoked
// by ssh-keygen.
// Using an app-owned directory avoids noexec tmpfs mounts on hardened hosts.
// The caller must invoke the returned cleanup function when done.
func writeAskpassScript(passphrase string) (name string, cleanup func(), err error) {
	base, err := os.UserCacheDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", func() {}, err
		}
		if home == "" {
			return "", func() {}, fmt.Errorf("user home directory is empty")
		}
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "git-fire")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", func() {}, err
	}
	pattern := "gf-askpass-*.sh"
	if runtime.GOOS == "windows" {
		pattern = "gf-askpass-*.cmd"
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", func() {}, err
	}
	name = f.Name()
	cleanup = func() { os.Remove(name) }
	// Emit a platform-native askpass helper. Windows OpenSSH expects an
	// executable command file (.cmd), while Unix uses a shell script.
	var script string
	if runtime.GOOS == "windows" {
		escaped := escapeForCmdSetP(passphrase)
		script = fmt.Sprintf("@echo off\r\n<nul set /p =%s\r\n", escaped)
	} else {
		// Use printf to avoid a trailing newline that could mismatch the passphrase.
		// Single-quote the passphrase; escape any embedded single quotes.
		escaped := strings.ReplaceAll(passphrase, "'", `'\''`)
		script = fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\n", escaped)
	}
	if _, err := f.WriteString(script); err != nil {
		_ = f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(name, 0o700); err != nil {
			cleanup()
			return "", func() {}, err
		}
	}
	return name, cleanup, nil
}

func escapeForCmdSetP(s string) string {
	replacer := strings.NewReplacer(
		"^", "^^",
		`"`, `^"`,
		"&", "^&",
		"|", "^|",
		"<", "^<",
		">", "^>",
		"(", "^(",
		")", "^)",
		"%", "%%",
	)
	return replacer.Replace(s)
}

// TestPassphrase tests if a passphrase is correct for a key.
// Returns true if the passphrase unlocks the key.
// Uses SSH_ASKPASS to avoid exposing the passphrase in process listings.
func TestPassphrase(keyPath, passphrase string) bool {
	askpassName, cleanup, err := writeAskpassScript(passphrase)
	if err != nil {
		return false
	}
	defer cleanup()

	cmd := exec.Command("ssh-keygen", "-y", "-f", keyPath)
	// SSH_ASKPASS_REQUIRE=force tells OpenSSH ≥8.4 to use SSH_ASKPASS even
	// without a TTY. DISPLAY must be set (any non-empty value) for older versions.
	cmd.Env = append(os.Environ(),
		"SSH_ASKPASS="+askpassName,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=dummy",
	)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// IsKeyEncrypted checks if an SSH key is encrypted (requires passphrase)
func IsKeyEncrypted(keyPath string) (bool, error) {
	content, err := os.ReadFile(keyPath)
	if err != nil {
		return false, fmt.Errorf("failed to read key file: %w", err)
	}

	contentStr := string(content)

	// Check for explicit encryption markers
	if strings.Contains(contentStr, "ENCRYPTED") {
		return true, nil
	}

	// Check for Proc-Type: 4,ENCRYPTED (traditional format)
	if strings.Contains(contentStr, "Proc-Type: 4,ENCRYPTED") {
		return true, nil
	}

	// For OpenSSH keys, check cipher field
	if strings.Contains(contentStr, "BEGIN OPENSSH PRIVATE KEY") {
		// Check if the key has "none" cipher (unencrypted)
		// This is a heuristic - the actual format is binary after the header
		if strings.Contains(contentStr, "none") {
			return false, nil
		}

		// Try to extract public key with an empty passphrase via SSH_ASKPASS
		// rather than the deprecated -P "" flag (OpenSSH ≥ 8.x).
		askpassName, cleanup, askErr := writeAskpassScript("")
		if askErr != nil {
			return true, nil // safe default
		}
		defer cleanup()
		cmd := exec.Command("ssh-keygen", "-y", "-f", keyPath)
		cmd.Env = append(os.Environ(),
			"SSH_ASKPASS="+askpassName,
			"SSH_ASKPASS_REQUIRE=force",
			"DISPLAY=dummy",
		)
		output, runErr := cmd.CombinedOutput()
		if runErr == nil && len(output) > 0 {
			return false, nil
		}
		// Failed → encrypted (or invalid key; default to encrypted for safety).
		return true, nil
	}

	// Default to not encrypted for unknown formats
	return false, nil
}

// EnsureSSHAgent starts ssh-agent if it's not running
func EnsureSSHAgent() error {
	agent, err := CheckSSHAgent()
	if err != nil {
		return err
	}

	if agent.Running {
		return nil // Already running
	}

	// Try to start ssh-agent
	cmd := exec.Command("ssh-agent", "-s")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start ssh-agent: %w", err)
	}

	// Parse output and set environment variables
	// Output format:
	// SSH_AUTH_SOCK=/tmp/ssh-XXX/agent.1234; export SSH_AUTH_SOCK;
	// SSH_AGENT_PID=1234; export SSH_AGENT_PID;

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "SSH_AUTH_SOCK=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				socket := strings.TrimSuffix(parts[1], "; export SSH_AUTH_SOCK;")
				socket = strings.TrimSpace(socket)
				os.Setenv("SSH_AUTH_SOCK", socket)
			}
		}
		if strings.Contains(line, "SSH_AGENT_PID=") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				pid := strings.TrimSuffix(parts[1], "; export SSH_AGENT_PID;")
				pid = strings.TrimSpace(pid)
				os.Setenv("SSH_AGENT_PID", pid)
			}
		}
	}

	return nil
}

// GetSSHStatus returns a summary of SSH configuration
func GetSSHStatus() (*SSHStatus, error) {
	status := &SSHStatus{}

	// Find SSH keys
	keys, err := FindSSHKeys()
	if err != nil {
		return nil, err
	}
	status.AvailableKeys = keys

	// Check ssh-agent
	agent, err := CheckSSHAgent()
	if err != nil {
		return nil, err
	}
	status.Agent = agent

	// Mark which keys are loaded
	if agent.KeysKnown {
		for i := range status.AvailableKeys {
			key := &status.AvailableKeys[i]

			// Check if this key is loaded in agent
			for _, agentKey := range agent.Keys {
				if agentKey.Fingerprint == key.Fingerprint {
					key.IsLoaded = true
					break
				}
			}

			// Check if key needs passphrase
			if !key.IsLoaded {
				encrypted, _ := IsKeyEncrypted(key.Path)
				key.NeedsPass = encrypted
			}
		}
	}

	return status, nil
}

// SSHStatus represents the complete SSH configuration status
type SSHStatus struct {
	AvailableKeys []SSHKey
	Agent         *SSHAgent
}

// Summary returns a human-readable summary
func (s *SSHStatus) Summary() string {
	var sb strings.Builder

	sb.WriteString("SSH Configuration:\n")

	if s.Agent.Running {
		sb.WriteString(fmt.Sprintf("  ✓ ssh-agent running (socket: %s)\n", s.Agent.Socket))
		if s.Agent.KeysKnown {
			sb.WriteString(fmt.Sprintf("  ✓ %d key(s) loaded in agent\n", len(s.Agent.Keys)))
		} else {
			sb.WriteString("  ? key inventory unavailable (agent probe did not complete)\n")
		}
		if s.Agent.Error != "" {
			sb.WriteString(fmt.Sprintf("  ⚠ agent status warning: %s\n", s.Agent.Error))
		}
	} else {
		sb.WriteString("  ✗ ssh-agent not running\n")
	}

	sb.WriteString(fmt.Sprintf("\nAvailable SSH keys: %d\n", len(s.AvailableKeys)))
	for _, key := range s.AvailableKeys {
		status := "?"
		if s.Agent.KeysKnown {
			status = "✗"
		}
		if key.IsLoaded {
			status = "✓"
		}

		sb.WriteString(fmt.Sprintf("  %s %s (%s)", status, key.Name, key.Type))

		if key.IsLoaded {
			sb.WriteString(" [loaded]")
		} else if key.NeedsPass {
			sb.WriteString(" [needs passphrase]")
		}
		if key.FingerprintError != "" {
			sb.WriteString(" [fingerprint unavailable]")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}
