package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---- Load / Save ----

func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")
	reg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if len(reg.Repos) != 0 {
		t.Errorf("expected empty registry, got %d repos", len(reg.Repos))
	}
}

func TestLoad_CorruptFile_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")
	if err := os.WriteFile(path, []byte("not valid toml [[["), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for corrupt file, got nil")
	}
}

func TestSave_Load_Roundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")

	rescan := true
	orig := &Registry{
		Repos: []RegistryEntry{
			{
				Path:             "/repos/foo",
				Name:             "foo",
				Status:           StatusActive,
				Mode:             "push-known-branches",
				RescanSubmodules: &rescan,
				AddedAt:          time.Now().Truncate(time.Second),
				LastSeen:         time.Now().Truncate(time.Second),
			},
		},
	}

	if err := Save(orig, path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if len(loaded.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(loaded.Repos))
	}
	got := loaded.Repos[0]
	if got.Path != orig.Repos[0].Path {
		t.Errorf("Path: got %q, want %q", got.Path, orig.Repos[0].Path)
	}
	if got.Name != orig.Repos[0].Name {
		t.Errorf("Name: got %q, want %q", got.Name, orig.Repos[0].Name)
	}
	if got.Status != StatusActive {
		t.Errorf("Status: got %q, want %q", got.Status, StatusActive)
	}
	if got.Mode != orig.Repos[0].Mode {
		t.Errorf("Mode: got %q, want %q", got.Mode, orig.Repos[0].Mode)
	}
	if got.RescanSubmodules == nil || *got.RescanSubmodules != true {
		t.Errorf("RescanSubmodules: got %v, want true", got.RescanSubmodules)
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "repos.toml")
	reg := &Registry{}
	if err := Save(reg, path); err != nil {
		t.Fatalf("Save() should create missing directories: %v", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("registry file was not created")
	}
}

func TestSave_FilePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repos.toml")
	if err := Save(&Registry{}, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("expected permissions 0600, got %o", info.Mode().Perm())
	}
}

// ---- Upsert ----

func TestUpsert_NewEntry(t *testing.T) {
	reg := &Registry{}
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive})
	if len(reg.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(reg.Repos))
	}
}

func TestUpsert_SetsAddedAt_WhenZero(t *testing.T) {
	reg := &Registry{}
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive})
	if reg.Repos[0].AddedAt.IsZero() {
		t.Error("AddedAt should be set automatically when zero")
	}
}

func TestUpsert_UpdatesExisting(t *testing.T) {
	reg := &Registry{}
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive})
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a-renamed", Status: StatusMissing})

	if len(reg.Repos) != 1 {
		t.Fatalf("Upsert should update, not append: got %d repos", len(reg.Repos))
	}
	if reg.Repos[0].Name != "a-renamed" {
		t.Errorf("Name not updated: got %q", reg.Repos[0].Name)
	}
	if reg.Repos[0].Status != StatusMissing {
		t.Errorf("Status not updated: got %q", reg.Repos[0].Status)
	}
}

func TestUpsert_PreservesAddedAt(t *testing.T) {
	original := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", AddedAt: original}},
	}
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive})
	if !reg.Repos[0].AddedAt.Equal(original) {
		t.Errorf("AddedAt should be preserved on update: got %v, want %v", reg.Repos[0].AddedAt, original)
	}
}

func TestUpsert_PreservesRescanSubmodules_WhenNil(t *testing.T) {
	rescan := true
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", RescanSubmodules: &rescan}},
	}
	// Upsert with nil RescanSubmodules — existing override should be kept
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive, RescanSubmodules: nil})

	if reg.Repos[0].RescanSubmodules == nil || *reg.Repos[0].RescanSubmodules != true {
		t.Error("RescanSubmodules override should be preserved when caller passes nil")
	}
}

func TestUpsert_OverridesRescanSubmodules_WhenNonNil(t *testing.T) {
	old := true
	new := false
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", RescanSubmodules: &old}},
	}
	reg.Upsert(RegistryEntry{Path: "/repos/a", Name: "a", Status: StatusActive, RescanSubmodules: &new})

	if reg.Repos[0].RescanSubmodules == nil || *reg.Repos[0].RescanSubmodules != false {
		t.Error("RescanSubmodules should be updated when caller provides a non-nil value")
	}
}

// ---- SetStatus ----

func TestSetStatus_Found(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Status: StatusActive}},
	}
	if !reg.SetStatus("/repos/a", StatusIgnored) {
		t.Fatal("SetStatus() returned false for existing path")
	}
	if reg.Repos[0].Status != StatusIgnored {
		t.Errorf("Status not updated: got %q", reg.Repos[0].Status)
	}
}

func TestSetStatus_NotFound(t *testing.T) {
	reg := &Registry{}
	if reg.SetStatus("/repos/missing", StatusActive) {
		t.Error("SetStatus() should return false for unknown path")
	}
}

func TestSetStatus_ActiveUpdatesLastSeen(t *testing.T) {
	before := time.Now().Add(-time.Minute)
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Status: StatusMissing, LastSeen: before}},
	}
	reg.SetStatus("/repos/a", StatusActive)
	if !reg.Repos[0].LastSeen.After(before) {
		t.Error("LastSeen should be updated when status is set to active")
	}
}

func TestSetStatus_NonActive_DoesNotUpdateLastSeen(t *testing.T) {
	fixed := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Status: StatusActive, LastSeen: fixed}},
	}
	reg.SetStatus("/repos/a", StatusIgnored)
	if !reg.Repos[0].LastSeen.Equal(fixed) {
		t.Error("LastSeen should not change when setting non-active status")
	}
}

// ---- Remove ----

func TestRemove_Found(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{
			{Path: "/repos/a"},
			{Path: "/repos/b"},
		},
	}
	if !reg.Remove("/repos/a") {
		t.Fatal("Remove() returned false for existing path")
	}
	if len(reg.Repos) != 1 || reg.Repos[0].Path != "/repos/b" {
		t.Errorf("wrong repos after remove: %v", reg.Repos)
	}
}

func TestRemove_NotFound(t *testing.T) {
	reg := &Registry{}
	if reg.Remove("/repos/missing") {
		t.Error("Remove() should return false for unknown path")
	}
}

func TestRemove_Last(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a"}},
	}
	reg.Remove("/repos/a")
	if len(reg.Repos) != 0 {
		t.Errorf("expected empty slice after removing last entry, got %d", len(reg.Repos))
	}
}

// ---- FindByPath ----

func TestFindByPath_Found(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Name: "a"}},
	}
	e := reg.FindByPath("/repos/a")
	if e == nil {
		t.Fatal("FindByPath() returned nil for existing path")
	}
	if e.Name != "a" {
		t.Errorf("wrong entry returned: %+v", e)
	}
}

func TestFindByPath_NotFound(t *testing.T) {
	reg := &Registry{}
	if reg.FindByPath("/repos/missing") != nil {
		t.Error("FindByPath() should return nil for unknown path")
	}
}

func TestFindByPath_MutationReflectsInSlice(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Status: StatusActive}},
	}
	e := reg.FindByPath("/repos/a")
	e.Status = StatusMissing
	if reg.Repos[0].Status != StatusMissing {
		t.Error("FindByPath() should return a pointer into the slice, not a copy")
	}
}

// ---- UpdateByPath ----

func TestUpdateByPath_Found(t *testing.T) {
	reg := &Registry{
		Repos: []RegistryEntry{{Path: "/repos/a", Name: "a", Status: StatusActive}},
	}
	ok := reg.UpdateByPath("/repos/a", func(e *RegistryEntry) {
		e.Status = StatusIgnored
		e.Name = "a-updated"
	})
	if !ok {
		t.Fatal("UpdateByPath() returned false for existing path")
	}
	if reg.Repos[0].Status != StatusIgnored || reg.Repos[0].Name != "a-updated" {
		t.Fatalf("UpdateByPath() did not persist changes: %+v", reg.Repos[0])
	}
}

func TestUpdateByPath_NotFound(t *testing.T) {
	reg := &Registry{}
	called := false
	ok := reg.UpdateByPath("/repos/missing", func(e *RegistryEntry) {
		called = true
	})
	if ok {
		t.Fatal("UpdateByPath() should return false for missing path")
	}
	if called {
		t.Fatal("UpdateByPath() callback should not run for missing path")
	}
}

// ---- DefaultRegistryPath ----

func TestDefaultRegistryPath(t *testing.T) {
	path, err := DefaultRegistryPath()
	if err != nil {
		t.Fatalf("DefaultRegistryPath() error: %v", err)
	}
	if filepath.Base(path) != "repos.toml" {
		t.Errorf("expected filename repos.toml, got %q", filepath.Base(path))
	}
	parent := filepath.Dir(path)
	if filepath.Base(parent) != "git-fire" {
		t.Errorf("expected parent dir git-fire, got %q", filepath.Base(parent))
	}
}

func TestDefaultRegistryPath_UsesUserConfigDir(t *testing.T) {
	xdgHome := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	path, err := DefaultRegistryPath()
	if err != nil {
		t.Fatalf("DefaultRegistryPath() error: %v", err)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	want := filepath.Join(base, "git-fire", "repos.toml")
	if path != want {
		t.Fatalf("expected path %q, got %q", want, path)
	}
}
