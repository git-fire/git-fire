package registry

import "time"

// Status values for registry entries
const (
	StatusActive  = "active"
	StatusMissing = "missing"
	StatusIgnored = "ignored"
)

// RegistryEntry represents a tracked git repository
type RegistryEntry struct {
	// Absolute filesystem path to the repository root
	Path string `toml:"path"`

	// Human-readable name (directory basename)
	Name string `toml:"name"`

	// Status: "active", "missing", or "ignored"
	Status string `toml:"status"`

	// Last-used push mode (e.g. "push-known-branches")
	Mode string `toml:"mode,omitempty"`

	// Per-repo override for submodule re-scanning.
	// nil means inherit the global rescan_submodules setting.
	RescanSubmodules *bool `toml:"rescan_submodules,omitempty"`

	// Optional per-repo USB strategy override (e.g. "git-mirror", "git-clone").
	USBStrategy string `toml:"usb_strategy,omitempty"`

	// Optional per-repo destination repo path override relative to USB repos root.
	USBRepoPath string `toml:"usb_repo_path,omitempty"`

	// USB sync policy override: "keep" or "prune".
	USBSyncPolicy string `toml:"usb_sync_policy,omitempty"`

	// When this repo was first added to the registry
	AddedAt time.Time `toml:"added_at"`

	// Last time git-fire confirmed the path exists
	LastSeen time.Time `toml:"last_seen"`
}

// Registry is the top-level structure persisted to ~/.config/git-fire/repos.toml
type Registry struct {
	Repos []RegistryEntry `toml:"repos"`
}
