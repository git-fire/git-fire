package usb

import "testing"

func TestManifestRoundTrip(t *testing.T) {
	root := t.TempDir()
	m, err := LoadManifest(root)
	if err != nil {
		t.Fatalf("LoadManifest(new) error: %v", err)
	}
	m.Results["/tmp/repo"] = RepoOutcome{
		RepoPath:    "/tmp/repo",
		RepoName:    "repo",
		Destination: "/tmp/target/repo.git",
		Success:     true,
	}
	if err := SaveManifest(root, m); err != nil {
		t.Fatalf("SaveManifest error: %v", err)
	}
	m2, err := LoadManifest(root)
	if err != nil {
		t.Fatalf("LoadManifest(existing) error: %v", err)
	}
	if len(m2.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m2.Results))
	}
	if !m2.Results["/tmp/repo"].Success {
		t.Fatalf("expected success outcome")
	}
}
