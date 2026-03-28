package plugins

import (
	"fmt"
	"sync"
	"testing"
)

// mockPlugin is a minimal Plugin implementation for registry tests.
type mockPlugin struct {
	name        string
	validateErr error
}

func (m *mockPlugin) Name() string              { return m.name }
func (m *mockPlugin) Type() PluginType          { return PluginTypeCommand }
func (m *mockPlugin) Validate() error           { return m.validateErr }
func (m *mockPlugin) Execute(_ Context) error   { return nil }
func (m *mockPlugin) Cleanup() error            { return nil }

// newRegistry returns an empty registry for test isolation.
func newTestRegistry() *Registry {
	return &Registry{plugins: make(map[string]Plugin)}
}

// mustRegister registers a plugin or fails the test immediately.
func mustRegister(t *testing.T, r *Registry, p Plugin) {
	t.Helper()
	if err := r.Register(p); err != nil {
		t.Fatalf("setup: Register(%q) failed: %v", p.Name(), err)
	}
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name    string
		plugin  Plugin
		wantErr bool
	}{
		{
			name:    "valid plugin",
			plugin:  &mockPlugin{name: "my-plugin"},
			wantErr: false,
		},
		{
			name:    "nil plugin",
			plugin:  nil,
			wantErr: true,
		},
		{
			name:    "empty name",
			plugin:  &mockPlugin{name: ""},
			wantErr: true,
		},
		{
			name:    "validation failure",
			plugin:  &mockPlugin{name: "bad-plugin", validateErr: fmt.Errorf("bad config")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRegistry()
			err := r.Register(tt.plugin)
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := newTestRegistry()
	p := &mockPlugin{name: "dup"}

	if err := r.Register(p); err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}
	if err := r.Register(p); err == nil {
		t.Error("expected error registering duplicate plugin name")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := newTestRegistry()
	p := &mockPlugin{name: "target"}
	mustRegister(t, r, p)

	got, err := r.Get("target")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if got.Name() != "target" {
		t.Errorf("Get() returned plugin with name %q, want %q", got.Name(), "target")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := newTestRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("Get() should return an error for missing plugin")
	}
}

func TestRegistry_List(t *testing.T) {
	r := newTestRegistry()
	mustRegister(t, r, &mockPlugin{name: "alpha"})
	mustRegister(t, r, &mockPlugin{name: "beta"})
	mustRegister(t, r, &mockPlugin{name: "gamma"})

	plugins := r.List()
	if len(plugins) != 3 {
		t.Errorf("List() returned %d plugins, want 3", len(plugins))
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	r := newTestRegistry()
	plugins := r.List()
	if len(plugins) != 0 {
		t.Errorf("List() on empty registry returned %d plugins, want 0", len(plugins))
	}
}

func TestRegistry_Clear(t *testing.T) {
	r := newTestRegistry()
	mustRegister(t, r, &mockPlugin{name: "a"})
	mustRegister(t, r, &mockPlugin{name: "b"})
	r.Clear()

	if r.Exists("a") || r.Exists("b") {
		t.Error("Clear() did not remove all plugins")
	}
	if len(r.List()) != 0 {
		t.Errorf("Clear() left %d plugins, want 0", len(r.List()))
	}
}

func TestRegistry_Clear_AllowsReRegistration(t *testing.T) {
	r := newTestRegistry()
	mustRegister(t, r, &mockPlugin{name: "reused"})
	r.Clear()

	if err := r.Register(&mockPlugin{name: "reused"}); err != nil {
		t.Errorf("Register after Clear() failed: %v", err)
	}
}

func TestRegistry_Exists(t *testing.T) {
	r := newTestRegistry()
	mustRegister(t, r, &mockPlugin{name: "present"})

	if !r.Exists("present") {
		t.Error("Exists() returned false for registered plugin")
	}
	if r.Exists("absent") {
		t.Error("Exists() returned true for unregistered plugin")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	// Smoke test: concurrent reads and writes should not race.
	// Run with: go test -race ./internal/plugins/...
	r := newTestRegistry()
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = r.Register(&mockPlugin{name: fmt.Sprintf("plugin-%d", i)})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			r.List()
		}
	}()

	wg.Wait()
}
