// Package safety detects secrets in repo output and redacts sensitive text from logs and errors.
package safety

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretPattern defines a pattern for detecting secrets
type SecretPattern struct {
	Name        string         // Human-readable name
	Pattern     *regexp.Regexp // Regex pattern
	Description string         // What it detects
}

// SuspiciousFile represents a file that might contain secrets
type SuspiciousFile struct {
	Path        string   // Relative path
	Reason      string   // Why it's suspicious
	Patterns    []string // Matched patterns
	LineNumbers []int    // Line numbers with matches
}

// SecretScanner scans for potential secrets in files
type SecretScanner struct {
	patterns           []SecretPattern
	suspiciousNames    []string
	suspiciousExts     []string
	suspiciousKeywords []string
}

// NewSecretScanner creates a scanner with default patterns
func NewSecretScanner() *SecretScanner {
	return &SecretScanner{
		patterns:           defaultPatterns(),
		suspiciousNames:    defaultSuspiciousNames(),
		suspiciousExts:     defaultSuspiciousExtensions(),
		suspiciousKeywords: defaultSuspiciousKeywords(),
	}
}

// defaultPatterns returns common secret patterns
func defaultPatterns() []SecretPattern {
	return []SecretPattern{
		{
			Name:        "AWS Access Key",
			Pattern:     regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			Description: "AWS Access Key ID",
		},
		{
			Name:        "AWS Secret Key",
			Pattern:     regexp.MustCompile(`(?i)aws(.{0,20})?['\"][0-9a-zA-Z/+]{40}['\"]`),
			Description: "AWS Secret Access Key",
		},
		{
			Name:        "Generic API Key",
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key|apikey)(.{0,20})?['\"][0-9a-zA-Z]{32,}['\"]`),
			Description: "Generic API Key",
		},
		{
			Name:        "Generic Secret",
			Pattern:     regexp.MustCompile(`(?i)(secret|password|passwd|pwd)(.{0,20})?['\"][^'\"]{8,}['\"]`),
			Description: "Generic Secret/Password",
		},
		{
			Name:        "GitHub Token",
			Pattern:     regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`),
			Description: "GitHub Personal Access Token",
		},
		{
			Name:        "GitHub OAuth",
			Pattern:     regexp.MustCompile(`gho_[0-9a-zA-Z]{36}`),
			Description: "GitHub OAuth Token",
		},
		{
			Name:        "GitLab Personal Access Token",
			Pattern:     regexp.MustCompile(`(?i)glpat-[0-9a-zA-Z_\-]{20,}`),
			Description: "GitLab personal access token",
		},
		{
			Name:        "Private Key Header",
			Pattern:     regexp.MustCompile(`-----BEGIN (RSA|DSA|EC|OPENSSH) PRIVATE KEY-----`),
			Description: "Private Key",
		},
		{
			Name:        "Database URL",
			Pattern:     regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^:]+:[^@]+@`),
			Description: "Database connection string with credentials",
		},
		{
			Name:        "Slack Token",
			Pattern:     regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z]{10,48}`),
			Description: "Slack Token",
		},
		{
			Name:        "Bearer Token",
			Pattern:     regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9\-._~+/]+=*`),
			Description: "Bearer Token",
		},
	}
}

// defaultSuspiciousNames returns filenames that often contain secrets
func defaultSuspiciousNames() []string {
	return []string{
		".env",
		".env.local",
		".env.production",
		".env.development",
		"credentials.json",
		"credentials.yml",
		"credentials.yaml",
		"secrets.json",
		"secrets.yml",
		"secrets.yaml",
		"config/secrets.yml",
		"config/credentials.yml",
		".netrc",
		".npmrc",
		".pypirc",
		"id_rsa",
		"id_dsa",
		"id_ecdsa",
		"id_ed25519",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"*.jks",
		"*.keystore",
	}
}

// defaultSuspiciousExtensions returns file extensions that may contain secrets
func defaultSuspiciousExtensions() []string {
	return []string{
		".pem",
		".key",
		".p12",
		".pfx",
		".jks",
		".keystore",
		".crt",
		".cer",
		".der",
	}
}

// defaultSuspiciousKeywords returns keywords that suggest secrets
func defaultSuspiciousKeywords() []string {
	return []string{
		"password",
		"passwd",
		"secret",
		"api_key",
		"apikey",
		"api-key",
		"private_key",
		"access_token",
		"auth_token",
		"session_token",
	}
}

// ScanFiles scans a list of files for potential secrets
func (s *SecretScanner) ScanFiles(repoPath string, files []string) ([]SuspiciousFile, error) {
	var suspicious []SuspiciousFile

	for _, file := range files {
		fullPath := filepath.Join(repoPath, file)

		// Check by filename/extension
		if s.isSuspiciousFilename(file) {
			suspicious = append(suspicious, SuspiciousFile{
				Path:   file,
				Reason: "Filename matches common secret file pattern",
			})
			continue
		}

		// Scan file contents
		matches, err := s.scanFileContent(fullPath)
		if err != nil {
			// Can't read file - skip (might be binary)
			continue
		}

		if len(matches) > 0 {
			suspicious = append(suspicious, matches...)
		}
	}

	return suspicious, nil
}

// isSuspiciousFilename checks if a filename suggests it might contain secrets
func (s *SecretScanner) isSuspiciousFilename(filename string) bool {
	base := filepath.Base(filename)
	ext := filepath.Ext(filename)

	// Check exact name matches
	for _, suspicious := range s.suspiciousNames {
		// Handle wildcards
		if strings.Contains(suspicious, "*") {
			matched, _ := filepath.Match(suspicious, base)
			if matched {
				return true
			}
		} else if base == suspicious || filename == suspicious {
			return true
		}
	}

	// Check extension matches
	for _, suspiciousExt := range s.suspiciousExts {
		if ext == suspiciousExt {
			return true
		}
	}

	return false
}

// scanFileContent scans file content for secret patterns
func (s *SecretScanner) scanFileContent(path string) ([]SuspiciousFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []SuspiciousFile
	scanner := bufio.NewScanner(file)
	lineNum := 0
	matchedPatterns := make(map[string][]int) // pattern name -> line numbers

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check against patterns
		for _, pattern := range s.patterns {
			if pattern.Pattern.MatchString(line) {
				matchedPatterns[pattern.Name] = append(matchedPatterns[pattern.Name], lineNum)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Create result if patterns matched
	if len(matchedPatterns) > 0 {
		var patterns []string
		var lineNumbers []int

		for patternName, lines := range matchedPatterns {
			patterns = append(patterns, patternName)
			lineNumbers = append(lineNumbers, lines...)
		}

		results = append(results, SuspiciousFile{
			Path:        path,
			Reason:      "Content matches secret patterns",
			Patterns:    patterns,
			LineNumbers: lineNumbers,
		})
	}

	return results, nil
}

// FormatWarning formats a warning message for suspicious files
func FormatWarning(files []SuspiciousFile) string {
	if len(files) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n⚠️  WARNING: Potential secrets detected!\n\n")
	sb.WriteString("The following files may contain sensitive information:\n\n")

	for _, file := range files {
		fmt.Fprintf(&sb, "  ❌ %s\n", file.Path)
		fmt.Fprintf(&sb, "     Reason: %s\n", file.Reason)

		if len(file.Patterns) > 0 {
			sb.WriteString("     Patterns matched: ")
			sb.WriteString(strings.Join(file.Patterns, ", "))
			sb.WriteString("\n")
		}

		if len(file.LineNumbers) > 0 {
			fmt.Fprintf(&sb, "     Lines: %v\n", file.LineNumbers)
		}

		sb.WriteString("\n")
	}

	sb.WriteString("RECOMMENDATIONS:\n")
	sb.WriteString("  • Add sensitive files to .gitignore\n")
	sb.WriteString("  • Use environment variables for secrets\n")
	sb.WriteString("  • Run with --dry-run to preview commits\n")
	sb.WriteString("  • Use --skip-auto-commit to avoid committing these files\n\n")

	return sb.String()
}

// SecurityNotice returns the security notice to display on first run
func SecurityNotice() string {
	return `
⚠️  SECURITY WARNING

Git-fire will auto-commit ALL uncommitted files in emergency mode.

BEFORE running in panic mode:
  ✓ Use .env files for secrets (add to .gitignore)
  ✓ Add sensitive files to .gitignore or .git/info/exclude
  ✓ Never commit: .env, credentials.json, *.pem, *.key
  ✓ Use environment variables for sensitive configuration

ALWAYS run --dry-run first to preview what will be committed!
`
}

// RecommendedGitignorePatterns returns patterns to add to .gitignore
func RecommendedGitignorePatterns() []string {
	return []string{
		"# Secrets (recommended by git-fire)",
		".env",
		".env.*",
		"!.env.example",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"credentials.json",
		"secrets.yaml",
		"secrets.yml",
		"config/secrets.yml",
		"config/credentials.yml",
		".netrc",
		"id_rsa",
		"id_dsa",
		"id_ecdsa",
		"id_ed25519",
	}
}
