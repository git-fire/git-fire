package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Global.DefaultMode != "push-known-branches" {
		t.Errorf("Expected default_mode to be 'push-known-branches', got '%s'", cfg.Global.DefaultMode)
	}

	if cfg.Global.ConflictStrategy != "new-branch" {
		t.Errorf("Expected conflict_strategy to be 'new-branch', got '%s'", cfg.Global.ConflictStrategy)
	}

	if !cfg.Global.AutoCommitDirty {
		t.Error("Expected auto_commit_dirty to be true")
	}

	if cfg.Global.ScanWorkers != 8 {
		t.Errorf("Expected scan_workers to be 8, got %d", cfg.Global.ScanWorkers)
	}
}

func TestLoadConfig_NoFile(t *testing.T) {
	// Test in temp directory with no config file
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	cfg := LoadOrDefault()
	if cfg == nil {
		t.Fatal("Expected default config when no file exists")
	}

	// Should have defaults
	if cfg.Global.DefaultMode != "push-known-branches" {
		t.Errorf("Expected default mode, got %s", cfg.Global.DefaultMode)
	}
}

func TestLoadConfig_WithFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	// Create config file
	configContent := `
[global]
default_mode = "push-all"
auto_commit_dirty = false
scan_workers = 16

[backup]
platform = "gitlab"
`

	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Global.DefaultMode != "push-all" {
		t.Errorf("Expected default_mode to be 'push-all', got '%s'", cfg.Global.DefaultMode)
	}

	if cfg.Global.AutoCommitDirty {
		t.Error("Expected auto_commit_dirty to be false")
	}

	if cfg.Global.ScanWorkers != 16 {
		t.Errorf("Expected scan_workers to be 16, got %d", cfg.Global.ScanWorkers)
	}

	if cfg.Backup.Platform != "gitlab" {
		t.Errorf("Expected platform to be 'gitlab', got '%s'", cfg.Backup.Platform)
	}
}

func TestLoadConfig_EnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	os.Chdir(tmpDir)

	// Set environment variables
	os.Setenv("GIT_FIRE_API_TOKEN", "test-token-123")
	os.Setenv("GIT_FIRE_SSH_PASSPHRASE", "test-passphrase")
	defer func() {
		os.Unsetenv("GIT_FIRE_API_TOKEN")
		os.Unsetenv("GIT_FIRE_SSH_PASSPHRASE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.Backup.APIToken != "test-token-123" {
		t.Errorf("Expected API token from env var, got '%s'", cfg.Backup.APIToken)
	}

	if cfg.Auth.SSHPassphrase != "test-passphrase" {
		t.Errorf("Expected SSH passphrase from env var, got '%s'", cfg.Auth.SSHPassphrase)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid mode",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "invalid-mode",
					ConflictStrategy: "new-branch",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid conflict strategy",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "invalid-strategy",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid platform",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
				},
				Backup: BackupConfig{
					Platform: "invalid-platform",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFindRepoOverride(t *testing.T) {
	cfg := Config{
		Repos: []RepoOverride{
			{
				PathPattern: "/home/user/critical/*",
				Mode:        "push-all",
			},
			{
				RemotePattern: "github.com/company",
				Mode:          "push-known-branches",
			},
		},
	}

	tests := []struct {
		name      string
		repoPath  string
		remoteURL string
		wantMode  string
		wantNil   bool
	}{
		{
			name:      "matches path pattern",
			repoPath:  "/home/user/critical/project",
			remoteURL: "",
			wantMode:  "push-all",
			wantNil:   false,
		},
		{
			name:      "matches remote pattern",
			repoPath:  "/home/user/other",
			remoteURL: "git@github.com/company/repo.git",
			wantMode:  "push-known-branches",
			wantNil:   false,
		},
		{
			name:      "no match",
			repoPath:  "/home/user/other",
			remoteURL: "git@gitlab.com/user/repo.git",
			wantMode:  "",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := cfg.FindRepoOverride(tt.repoPath, tt.remoteURL)
			if tt.wantNil {
				if override != nil {
					t.Error("Expected nil override")
				}
			} else {
				if override == nil {
					t.Fatal("Expected non-nil override")
				}
				if override.Mode != tt.wantMode {
					t.Errorf("Expected mode %s, got %s", tt.wantMode, override.Mode)
				}
			}
		})
	}
}

func TestWriteExampleConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	err := WriteExampleConfig(configPath)
	if err != nil {
		t.Fatalf("WriteExampleConfig() error = %v", err)
	}

	// Check file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Check content
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	contentStr := string(content)
	if len(contentStr) == 0 {
		t.Error("Config file is empty")
	}

	// Should contain sections
	if !contains(contentStr, "[global]") {
		t.Error("Config should contain [global] section")
	}
	if !contains(contentStr, "[backup]") {
		t.Error("Config should contain [backup] section")
	}
	if !contains(contentStr, "[auth]") {
		t.Error("Config should contain [auth] section")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"24h", 24 * time.Hour, false},
		{"1h30m", 90 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDuration(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && result != tt.expected {
				t.Errorf("ParseDuration(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
