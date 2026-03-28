package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// DefaultRegistryPath returns the default path for the registry file:
// ~/.git-fire/repos.toml
func DefaultRegistryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return filepath.Join(home, ".git-fire", "repos.toml"), nil
}

// Load reads the registry from disk. If the file or directory does not exist
// it is created and an empty registry is returned.
func Load(path string) (*Registry, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Registry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}

	var reg Registry
	if err := toml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	return &reg, nil
}

// Save writes the registry to disk atomically (write to a temp file, then rename).
func Save(reg *Registry, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	data, err := toml.Marshal(reg)
	if err != nil {
		return fmt.Errorf("marshaling registry: %w", err)
	}

	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("writing registry: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("saving registry: %w", err)
	}
	return nil
}

// Upsert adds a new entry or updates an existing one (matched by path).
// The AddedAt field is preserved when updating an existing entry.
func (r *Registry) Upsert(entry RegistryEntry) {
	for i, e := range r.Repos {
		if e.Path == entry.Path {
			// Preserve the original AddedAt
			entry.AddedAt = e.AddedAt
			r.Repos[i] = entry
			return
		}
	}
	if entry.AddedAt.IsZero() {
		entry.AddedAt = time.Now()
	}
	r.Repos = append(r.Repos, entry)
}

// SetStatus sets the status of the entry at path. Returns false if not found.
func (r *Registry) SetStatus(path, status string) bool {
	for i, e := range r.Repos {
		if e.Path == path {
			r.Repos[i].Status = status
			if status == StatusActive {
				r.Repos[i].LastSeen = time.Now()
			}
			return true
		}
	}
	return false
}

// Remove hard-deletes an entry by path. Returns false if not found.
func (r *Registry) Remove(path string) bool {
	for i, e := range r.Repos {
		if e.Path == path {
			r.Repos = append(r.Repos[:i], r.Repos[i+1:]...)
			return true
		}
	}
	return false
}

// FindByPath returns a pointer to the entry matching path, or nil if not found.
// The pointer is into the slice — do not store it beyond the next mutation.
func (r *Registry) FindByPath(path string) *RegistryEntry {
	for i := range r.Repos {
		if r.Repos[i].Path == path {
			return &r.Repos[i]
		}
	}
	return nil
}
