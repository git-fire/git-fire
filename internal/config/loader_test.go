package config

import (
	"os"
	"path/filepath"
	"strings"
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
	if !cfg.Global.BlockOnSecrets {
		t.Error("Expected block_on_secrets to be true")
	}

	if cfg.Global.ScanWorkers != 8 {
		t.Errorf("Expected scan_workers to be 8, got %d", cfg.Global.ScanWorkers)
	}
	if cfg.Global.PushWorkers != DefaultPushWorkers {
		t.Errorf("Expected push_workers to be %d, got %d", DefaultPushWorkers, cfg.Global.PushWorkers)
	}
	if cfg.UI.ColorProfile != UIColorProfileClassic {
		t.Errorf("Expected ui.color_profile to be %q, got %q", UIColorProfileClassic, cfg.UI.ColorProfile)
	}
	if cfg.UI.FireTickMS != DefaultUIFireTickMS {
		t.Errorf("Expected ui.fire_tick_ms to be %d, got %d", DefaultUIFireTickMS, cfg.UI.FireTickMS)
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
push_workers = 3

[backup]
platform = "gitlab"
`

	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadWithOptions(LoadOptions{ConfigFile: configPath})
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
	if cfg.Global.PushWorkers != 3 {
		t.Errorf("Expected push_workers to be 3, got %d", cfg.Global.PushWorkers)
	}

	if cfg.Backup.Platform != "gitlab" {
		t.Errorf("Expected platform to be 'gitlab', got '%s'", cfg.Backup.Platform)
	}
}

func TestLoadConfig_DoesNotImplicitlyLoadCWDConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	configPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\ndefault_mode = \"push-all\"\n"), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Global.DefaultMode != "push-known-branches" {
		t.Fatalf("expected default mode from defaults, got %s", cfg.Global.DefaultMode)
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
			name: "invalid ui color profile",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
				},
				UI: UIConfig{
					ColorProfile: "not-a-profile",
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
		{
			name: "invalid push workers fallback to default",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
					PushWorkers:      0,
				},
				UI: UIConfig{
					ColorProfile: UIColorProfileClassic,
				},
			},
			wantErr: false,
		},
		// FireTickMS: non-positive → default; otherwise clamped to Min/Max (loader Validate).
		{
			name: "invalid fire tick fallback to default",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
				},
				UI: UIConfig{
					ColorProfile: UIColorProfileClassic,
					FireTickMS:   0,
				},
			},
			wantErr: false,
		},
		{
			name: "fire tick below min clamps",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
				},
				UI: UIConfig{
					ColorProfile: UIColorProfileClassic,
					FireTickMS:   5,
				},
			},
			wantErr: false,
		},
		{
			name: "fire tick above max clamps",
			cfg: Config{
				Global: GlobalConfig{
					DefaultMode:      "push-all",
					ConflictStrategy: "new-branch",
				},
				UI: UIConfig{
					ColorProfile: UIColorProfileClassic,
					FireTickMS:   999999,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.name == "invalid push workers fallback to default" && tt.cfg.Global.PushWorkers != DefaultPushWorkers {
				t.Errorf("PushWorkers fallback = %d, want %d", tt.cfg.Global.PushWorkers, DefaultPushWorkers)
			}
			if tt.name == "invalid fire tick fallback to default" && tt.cfg.UI.FireTickMS != DefaultUIFireTickMS {
				t.Errorf("FireTickMS fallback = %d, want %d", tt.cfg.UI.FireTickMS, DefaultUIFireTickMS)
			}
			if tt.name == "fire tick below min clamps" && tt.cfg.UI.FireTickMS != MinUIFireTickMS {
				t.Errorf("FireTickMS clamp low = %d, want %d", tt.cfg.UI.FireTickMS, MinUIFireTickMS)
			}
			if tt.name == "fire tick above max clamps" && tt.cfg.UI.FireTickMS != MaxUIFireTickMS {
				t.Errorf("FireTickMS clamp high = %d, want %d", tt.cfg.UI.FireTickMS, MaxUIFireTickMS)
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

func TestDefaultConfigPath(t *testing.T) {
	path := DefaultConfigPath()
	if filepath.Base(path) != "config.toml" {
		t.Fatalf("expected filename config.toml, got %q", filepath.Base(path))
	}
	parent := filepath.Base(filepath.Dir(path))
	if parent != "git-fire" {
		t.Fatalf("expected parent directory git-fire, got %q", parent)
	}
}

func TestDefaultConfigPath_UsesUserConfigDir(t *testing.T) {
	xdgHome := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	path := DefaultConfigPath()
	want := filepath.Join(xdgHome, "git-fire", "config.toml")
	if path != want {
		t.Fatalf("expected path %q, got %q", want, path)
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

// ---- SaveConfig round-trip tests ----

// saveConfigAndReload saves cfg to a temp file and reloads it via pelletier
// (the same library SaveConfig uses to write), giving us a clean round-trip
// without going through viper so env vars don't interfere.
func saveConfigAndReload(t *testing.T, cfg *Config) Config {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	if err := SaveConfig(cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile after SaveConfig: %v", err)
	}
	var loaded Config
	if err := tomlUnmarshal(data, &loaded); err != nil {
		t.Fatalf("toml.Unmarshal after SaveConfig: %v", err)
	}
	return loaded
}

func TestSaveConfig_GlobalFieldsRoundTrip(t *testing.T) {
	original := DefaultConfig()
	original.Global.DefaultMode = "push-all"
	original.Global.DisableScan = true
	original.Global.AutoCommitDirty = false
	original.Global.ConflictStrategy = "abort"
	original.Global.PushWorkers = 7
	original.UI.FireTickMS = 150
	original.UI.ColorProfile = UIColorProfileSynthwave

	loaded := saveConfigAndReload(t, &original)

	if loaded.Global.DefaultMode != "push-all" {
		t.Errorf("DefaultMode: want push-all, got %s", loaded.Global.DefaultMode)
	}
	if !loaded.Global.DisableScan {
		t.Error("DisableScan: want true, got false")
	}
	if loaded.Global.AutoCommitDirty {
		t.Error("AutoCommitDirty: want false, got true")
	}
	if loaded.Global.ConflictStrategy != "abort" {
		t.Errorf("ConflictStrategy: want abort, got %s", loaded.Global.ConflictStrategy)
	}
	if loaded.Global.PushWorkers != 7 {
		t.Errorf("PushWorkers: want 7, got %d", loaded.Global.PushWorkers)
	}
	if loaded.UI.ColorProfile != UIColorProfileSynthwave {
		t.Errorf("UIColorProfile: want %s, got %s", UIColorProfileSynthwave, loaded.UI.ColorProfile)
	}
	if loaded.UI.FireTickMS != 150 {
		t.Errorf("UIFireTickMS: want 150, got %d", loaded.UI.FireTickMS)
	}
}

func TestSaveConfig_RepoOverridesRoundTrip(t *testing.T) {
	original := DefaultConfig()
	original.Repos = []RepoOverride{
		{PathPattern: "/home/user/myproject", Mode: "push-all"},
	}

	loaded := saveConfigAndReload(t, &original)

	if len(loaded.Repos) != 1 {
		t.Fatalf("Repos: want 1 entry, got %d", len(loaded.Repos))
	}
	if loaded.Repos[0].PathPattern != "/home/user/myproject" {
		t.Errorf("PathPattern: want /home/user/myproject, got %s", loaded.Repos[0].PathPattern)
	}
	if loaded.Repos[0].Mode != "push-all" {
		t.Errorf("Mode: want push-all, got %s", loaded.Repos[0].Mode)
	}
}

func TestSaveConfig_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.toml")

	cfg := DefaultConfig()
	if err := SaveConfig(&cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	if _, err := os.Stat(cfgPath); err != nil {
		t.Errorf("config file missing after SaveConfig: %v", err)
	}
	if _, err := os.Stat(cfgPath + ".tmp"); err == nil {
		t.Error("temp file still exists after SaveConfig")
	}
}

func TestSaveConfig_StripsSecretsWhenEnvSet(t *testing.T) {
	t.Setenv("GIT_FIRE_API_TOKEN", "secret-from-env")
	t.Setenv("GIT_FIRE_SSH_PASSPHRASE", "ssh-secret")

	cfg := DefaultConfig()
	cfg.Backup.APIToken = "should-not-appear-on-disk"
	cfg.Auth.SSHPassphrase = "also-never-persisted"

	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	if err := SaveConfig(&cfg, cfgPath); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "should-not-appear-on-disk") || strings.Contains(content, "also-never-persisted") {
		t.Errorf("SaveConfig wrote secret values to disk: %q", content)
	}
	var loaded Config
	if err := tomlUnmarshal(data, &loaded); err != nil {
		t.Fatalf("toml unmarshal: %v", err)
	}
	if loaded.Backup.APIToken != "" || loaded.Auth.SSHPassphrase != "" {
		t.Errorf("expected empty secrets in file, got api_token=%q passphrase=%q", loaded.Backup.APIToken, loaded.Auth.SSHPassphrase)
	}
}
