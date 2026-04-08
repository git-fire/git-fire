package usb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const ManifestFileName = "git-fire-usb-manifest.json"

type Manifest struct {
	SchemaVersion int                    `json:"schema_version"`
	GeneratedAt   time.Time              `json:"generated_at"`
	TargetRoot    string                 `json:"target_root"`
	Results       map[string]RepoOutcome `json:"results"`
}

type RepoOutcome struct {
	RepoPath    string    `json:"repo_path"`
	RepoName    string    `json:"repo_name"`
	Destination string    `json:"destination"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ManifestPath(targetRoot string) string {
	return filepath.Join(targetRoot, ManifestFileName)
}

func LoadManifest(targetRoot string) (*Manifest, error) {
	p := ManifestPath(targetRoot)
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &Manifest{
			SchemaVersion: DefaultSchemaVersion,
			GeneratedAt:   time.Now().UTC(),
			TargetRoot:    targetRoot,
			Results:       map[string]RepoOutcome{},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Results == nil {
		m.Results = map[string]RepoOutcome{}
	}
	return &m, nil
}

func SaveManifest(targetRoot string, m *Manifest) error {
	if m == nil {
		return nil
	}
	m.GeneratedAt = time.Now().UTC()
	m.TargetRoot = targetRoot
	if m.SchemaVersion <= 0 {
		m.SchemaVersion = DefaultSchemaVersion
	}
	if m.Results == nil {
		m.Results = map[string]RepoOutcome{}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ManifestPath(targetRoot), b, 0o600)
}
