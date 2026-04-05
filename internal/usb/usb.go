package usb

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	MarkerFileName       = ".git-fire"
	DefaultSchemaVersion = 1
	DefaultRepoLayoutDir = "repos"
	StrategyMirror       = "git-mirror"
	StrategyClone        = "git-clone"
	DefaultWorkers       = 1
)

type VolumeConfig struct {
	SchemaVersion int       `toml:"schema_version"`
	LayoutDir     string    `toml:"layout_dir"`
	Strategy      string    `toml:"strategy"`
	CreatedAt     time.Time `toml:"created_at"`
}

type EnsureOptions struct {
	DefaultStrategy string
	CreateIfMissing bool
}

func EnsureVolumeConfig(root string, opts EnsureOptions) (*VolumeConfig, error) {
	root = filepath.Clean(root)
	if root == "" || root == "." {
		return nil, fmt.Errorf("usb target root cannot be empty")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("usb target not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("usb target is not a directory: %s", root)
	}

	markerPath := filepath.Join(root, MarkerFileName)
	cfg, loadErr := loadVolumeConfig(markerPath)
	if loadErr == nil {
		return normalizeConfig(cfg), nil
	}
	if !os.IsNotExist(loadErr) {
		return nil, fmt.Errorf("failed reading %s: %w", markerPath, loadErr)
	}
	if !opts.CreateIfMissing {
		return nil, fmt.Errorf("missing usb marker %s (use --usb-init to create it)", markerPath)
	}

	cfg = &VolumeConfig{
		SchemaVersion: DefaultSchemaVersion,
		LayoutDir:     DefaultRepoLayoutDir,
		Strategy:      normalizeStrategy(opts.DefaultStrategy),
		CreatedAt:     time.Now().UTC(),
	}
	if err := writeVolumeConfig(markerPath, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func TargetReposRoot(targetRoot string, cfg *VolumeConfig) string {
	layoutDir := DefaultRepoLayoutDir
	if cfg != nil && strings.TrimSpace(cfg.LayoutDir) != "" {
		layoutDir = strings.TrimSpace(cfg.LayoutDir)
	}
	return filepath.Join(targetRoot, layoutDir)
}

func StableRepoName(repoPath, repoName string) string {
	base := strings.TrimSpace(repoName)
	if base == "" {
		base = filepath.Base(repoPath)
	}
	base = sanitize(base)
	sum := sha1.Sum([]byte(filepath.Clean(repoPath)))
	short := hex.EncodeToString(sum[:])[:8]
	return fmt.Sprintf("%s-%s", base, short)
}

func SyncMirrorRepo(sourceRepoPath, destinationBarePath string) error {
	if err := os.MkdirAll(filepath.Dir(destinationBarePath), 0o755); err != nil {
		return fmt.Errorf("failed creating destination parent: %w", err)
	}
	if _, err := os.Stat(destinationBarePath); os.IsNotExist(err) {
		if err := runGit("", "init", "--bare", destinationBarePath); err != nil {
			return fmt.Errorf("failed to initialize bare destination: %w", err)
		}
	}
	fileURL, err := fileURLFromPath(destinationBarePath)
	if err != nil {
		return err
	}
	if err := runGit(sourceRepoPath, "push", "--mirror", fileURL); err != nil {
		return fmt.Errorf("failed mirror push: %w", err)
	}
	return nil
}

func SyncCloneRepo(sourceRepoPath, destinationPath string) error {
	if _, err := os.Stat(destinationPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return fmt.Errorf("failed creating clone destination parent: %w", err)
		}
		if err := runGit("", "clone", sourceRepoPath, destinationPath); err != nil {
			return fmt.Errorf("failed initial clone: %w", err)
		}
		return nil
	}

	const syncRemote = "git-fire-source"
	if err := runGit(destinationPath, "remote", "add", syncRemote, sourceRepoPath); err != nil {
		_ = runGit(destinationPath, "remote", "set-url", syncRemote, sourceRepoPath)
	}
	if err := runGit(destinationPath, "fetch", syncRemote, "--prune", "--tags"); err != nil {
		return fmt.Errorf("failed fetch from source: %w", err)
	}

	currentBranch, err := currentBranch(sourceRepoPath)
	if err != nil {
		return fmt.Errorf("failed to detect source branch: %w", err)
	}
	remoteRef := fmt.Sprintf("%s/%s", syncRemote, currentBranch)
	if err := runGit(destinationPath, "checkout", "-B", currentBranch, remoteRef); err != nil {
		return fmt.Errorf("failed checkout synced branch: %w", err)
	}
	if err := runGit(destinationPath, "reset", "--hard", remoteRef); err != nil {
		return fmt.Errorf("failed hard reset to source branch: %w", err)
	}
	return nil
}

func normalizeConfig(cfg *VolumeConfig) *VolumeConfig {
	if cfg == nil {
		return &VolumeConfig{
			SchemaVersion: DefaultSchemaVersion,
			LayoutDir:     DefaultRepoLayoutDir,
			Strategy:      StrategyMirror,
		}
	}
	if cfg.SchemaVersion <= 0 {
		cfg.SchemaVersion = DefaultSchemaVersion
	}
	if strings.TrimSpace(cfg.LayoutDir) == "" {
		cfg.LayoutDir = DefaultRepoLayoutDir
	}
	cfg.Strategy = normalizeStrategy(cfg.Strategy)
	return cfg
}

func normalizeStrategy(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case StrategyClone:
		return StrategyClone
	case StrategyMirror:
		return StrategyMirror
	default:
		return StrategyMirror
	}
}

func loadVolumeConfig(path string) (*VolumeConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg VolumeConfig
	if err := toml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func writeVolumeConfig(path string, cfg *VolumeConfig) error {
	b, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed encoding marker config: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("failed writing marker config: %w", err)
	}
	return nil
}

func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w (%s)", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func currentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "", fmt.Errorf("detached HEAD")
	}
	return branch, nil
}

func fileURLFromPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve destination path: %w", err)
	}
	u := &url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}
	return u.String(), nil
}

func sanitize(value string) string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "repo"
	}
	return out
}
