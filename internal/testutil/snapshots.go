package testutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Snapshot represents a saved state of a git repository
type Snapshot struct {
	name    string
	tarball []byte // Compressed repository state in memory
}

// SnapshotRepo creates an in-memory snapshot of a repository
// This allows fast restoration of expensive test setups
func SnapshotRepo(t *testing.T, repoPath string) *Snapshot {
	t.Helper()

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	// Walk the repository directory and add all files to tarball
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Set relative path in tarball
		relPath, err := filepath.Rel(repoPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Write file content (if regular file)
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file %s to tar: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Close writers
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Failed to close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	return &Snapshot{
		name:    filepath.Base(repoPath),
		tarball: buf.Bytes(),
	}
}

// RestoreSnapshot restores a snapshot to a new temporary directory
// Returns the path to the restored repository
func RestoreSnapshot(t *testing.T, snapshot *Snapshot) string {
	t.Helper()

	// Create temp directory for restoration
	tmpDir := t.TempDir()
	restorePath := filepath.Join(tmpDir, snapshot.name)

	if err := os.MkdirAll(restorePath, 0755); err != nil {
		t.Fatalf("Failed to create restore directory: %v", err)
	}

	// Create readers
	gzipReader, err := gzip.NewReader(bytes.NewReader(snapshot.tarball))
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	// Extract files from tarball
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			t.Fatalf("Failed to read tar header: %v", err)
		}

		// Construct full path
		targetPath := filepath.Join(restorePath, header.Name)

		// Handle different file types
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				t.Fatalf("Failed to create directory %s: %v", targetPath, err)
			}

		case tar.TypeReg:
			// Create parent directory if needed
			dir := filepath.Dir(targetPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatalf("Failed to create parent directory for %s: %v", targetPath, err)
			}

			// Create and write file
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				t.Fatalf("Failed to create file %s: %v", targetPath, err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				t.Fatalf("Failed to write file %s: %v", targetPath, err)
			}
			file.Close()

		default:
			t.Logf("Skipping unsupported file type %v for %s", header.Typeflag, header.Name)
		}
	}

	return restorePath
}

// SnapshotSize returns the size of the snapshot in bytes
func (s *Snapshot) Size() int {
	return len(s.tarball)
}

// SnapshotName returns the name of the snapshot
func (s *Snapshot) Name() string {
	return s.name
}

// SaveSnapshotToDisk saves a snapshot to a file (for debugging or caching)
func SaveSnapshotToDisk(t *testing.T, snapshot *Snapshot, filepath string) {
	t.Helper()

	if err := os.WriteFile(filepath, snapshot.tarball, 0644); err != nil {
		t.Fatalf("Failed to save snapshot to disk: %v", err)
	}
}

// LoadSnapshotFromDisk loads a snapshot from a file
func LoadSnapshotFromDisk(t *testing.T, filepath string) *Snapshot {
	t.Helper()

	data, err := os.ReadFile(filepath)
	if err != nil {
		t.Fatalf("Failed to load snapshot from disk: %v", err)
	}

	return &Snapshot{
		name:    filepath,
		tarball: data,
	}
}

// Example usage in tests:
//
// Expensive setup (run once):
//   repo := CreateLargeRepoScenario(t, 100, 50)
//   snapshot := SnapshotRepo(t, repo.Path())
//
// Fast restoration (run many times):
//   repoPath := RestoreSnapshot(t, snapshot)
//   // Run test on repoPath (10-100x faster than recreating)
