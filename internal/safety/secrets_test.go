package safety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSecretScanner(t *testing.T) {
	scanner := NewSecretScanner()

	if scanner == nil {
		t.Fatal("Expected non-nil scanner")
	}

	if len(scanner.patterns) == 0 {
		t.Error("Expected default patterns to be loaded")
	}

	if len(scanner.suspiciousNames) == 0 {
		t.Error("Expected suspicious names to be loaded")
	}
}

func TestIsSuspiciousFilename(t *testing.T) {
	scanner := NewSecretScanner()

	tests := []struct {
		filename string
		want     bool
	}{
		{".env", true},
		{".env.local", true},
		{".env.production", true},
		{"credentials.json", true},
		{"secrets.yml", true},
		{"id_rsa", true},
		{"id_rsa.pub", false}, // Public key is OK
		{"private.pem", true},
		{"cert.key", true},
		{"README.md", false},
		{"main.go", false},
		{"config/secrets.yml", true},
		{"src/secrets.yaml", true},
		{"test.pfx", true},
		{"keystore.jks", true},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := scanner.isSuspiciousFilename(tt.filename)
			if got != tt.want {
				t.Errorf("isSuspiciousFilename(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestScanFileContent_AWSKeys(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	// Create test file with AWS keys
	testFile := filepath.Join(tmpDir, "config.txt")
	content := `
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to detect AWS keys")
	}

	// Check that AWS pattern was matched
	found := false
	for _, result := range results {
		for _, pattern := range result.Patterns {
			if pattern == "AWS Access Key" {
				found = true
				break
			}
		}
	}

	if !found {
		t.Error("Expected to detect AWS Access Key pattern")
	}
}

func TestScanFileContent_GitHubToken(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "token.txt")
	// GitHub token format: ghp_ + 36 chars
	content := `
GITHUB_TOKEN=ghp_abcdefghijklmnopqrstuvwxyz1234567890
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to detect GitHub token")
	}
}

func TestScanFileContent_PrivateKey(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "key.pem")
	content := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567890abcdefghijklmnopqrstuvwxyz
-----END RSA PRIVATE KEY-----`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to detect private key")
	}
}

func TestScanFileContent_DatabaseURL(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "db.conf")
	content := `DATABASE_URL=postgres://username:password@localhost:5432/mydb`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to detect database URL with credentials")
	}
}

func TestScanFileContent_GitLabToken(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "token.txt")
	// GitLab PAT format: glpat- + 20+ alphanumeric/dash/underscore chars
	content := `
GITLAB_TOKEN=glpat-abcdefghij1234567890
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to detect GitLab personal access token")
	}
}

func TestScanFileContent_CleanFile(t *testing.T) {
	scanner := NewSecretScanner()
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "clean.txt")
	content := `
This is a clean file with no secrets.
Just some regular text content.
Nothing suspicious here.
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	results, err := scanner.scanFileContent(testFile)
	if err != nil {
		t.Fatalf("scanFileContent() error = %v", err)
	}

	if len(results) > 0 {
		t.Errorf("Expected no secrets in clean file, but found: %v", results)
	}
}

func TestScanFiles(t *testing.T) {
	t.Run("mixed_files", func(t *testing.T) {
		scanner := NewSecretScanner()
		tmpDir := t.TempDir()

		// Create test files
		files := map[string]string{
			".env":             "API_KEY=secret123",
			"config.json":      `{"key": "value"}`,
			"secrets.yml":      "password: secret",
			"README.md":        "# README",
			"credentials.json": `{"token": "abc123"}`,
		}

		for name, content := range files {
			path := filepath.Join(tmpDir, name)
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create test file %s: %v", name, err)
			}
		}

		fileList := []string{".env", "config.json", "secrets.yml", "README.md", "credentials.json"}

		results, err := scanner.ScanFiles(tmpDir, fileList)
		if err != nil {
			t.Fatalf("ScanFiles() error = %v", err)
		}

		// Should detect .env, secrets.yml, and credentials.json by filename
		if len(results) < 3 {
			t.Errorf("Expected at least 3 suspicious files, got %d", len(results))
		}

		// Check that .env was detected
		foundEnv := false
		for _, result := range results {
			if filepath.Base(result.Path) == ".env" {
				foundEnv = true
				break
			}
		}

		if !foundEnv {
			t.Error("Expected to detect .env file")
		}
	})

	t.Run("deleted_suspicious_filename", func(t *testing.T) {
		// Deleted paths still go through isSuspiciousFilename; scanFileContent is
		// never reached (os.Open fails) — assert filename-only reason.
		scanner := NewSecretScanner()
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envPath, []byte("SECRET=x\n"), 0644); err != nil {
			t.Fatalf("write .env: %v", err)
		}
		if err := os.Remove(envPath); err != nil {
			t.Fatalf("remove .env: %v", err)
		}

		results, err := scanner.ScanFiles(tmpDir, []string{".env"})
		if err != nil {
			t.Fatalf("ScanFiles() error = %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("want 1 filename-based hit, got %d (%v)", len(results), results)
		}
		if results[0].Path != ".env" {
			t.Errorf("got path %q, want .env", results[0].Path)
		}
		const wantReason = "Filename matches common secret file pattern"
		if results[0].Reason != wantReason {
			t.Errorf("want reason %q (filename path only), got %q", wantReason, results[0].Reason)
		}
	})
}

func TestFormatWarning(t *testing.T) {
	files := []SuspiciousFile{
		{
			Path:     ".env",
			Reason:   "Filename matches common secret file pattern",
			Patterns: []string{"Generic API Key"},
		},
		{
			Path:        "config.yml",
			Reason:      "Content matches secret patterns",
			Patterns:    []string{"AWS Access Key", "Generic Secret"},
			LineNumbers: []int{5, 10},
		},
	}

	warning := FormatWarning(files)

	if warning == "" {
		t.Error("Expected non-empty warning message")
	}

	// Should contain file paths
	if !strings.Contains(warning, ".env") {
		t.Error("Warning should mention .env file")
	}

	if !strings.Contains(warning, "config.yml") {
		t.Error("Warning should mention config.yml file")
	}

	// Should contain recommendations
	if !strings.Contains(warning, "RECOMMENDATIONS") {
		t.Error("Warning should include recommendations")
	}
}

func TestFormatWarning_Empty(t *testing.T) {
	warning := FormatWarning([]SuspiciousFile{})

	if warning != "" {
		t.Error("Expected empty warning for no suspicious files")
	}
}

func TestSecurityNotice(t *testing.T) {
	notice := SecurityNotice()

	if notice == "" {
		t.Error("Expected non-empty security notice")
	}

	// Should contain key warnings
	if !strings.Contains(notice, "WARNING") {
		t.Error("Notice should contain WARNING")
	}

	if !strings.Contains(notice, ".env") {
		t.Error("Notice should mention .env files")
	}

	if !strings.Contains(notice, "--dry-run") {
		t.Error("Notice should recommend --dry-run")
	}
}

func TestRecommendedGitignorePatterns(t *testing.T) {
	patterns := RecommendedGitignorePatterns()

	if len(patterns) == 0 {
		t.Error("Expected non-empty gitignore patterns")
	}

	// Should include common secret files
	foundEnv := false
	foundPem := false

	for _, pattern := range patterns {
		if strings.Contains(pattern, ".env") {
			foundEnv = true
		}
		if strings.Contains(pattern, ".pem") {
			foundPem = true
		}
	}

	if !foundEnv {
		t.Error("Patterns should include .env")
	}

	if !foundPem {
		t.Error("Patterns should include .pem files")
	}
}
