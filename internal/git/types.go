package git

import "time"

// Repository represents a discovered git repository
type Repository struct {
	Path         string    // Full filesystem path
	Name         string    // Repo name (from directory)
	Remotes      []Remote  // Configured remotes
	Branches     []string  // Local branch names
	IsDirty      bool      // Has uncommitted changes
	LastModified time.Time // Last commit time
	Selected     bool      // User selected for push
	Mode         RepoMode  // Push mode for this repo
}

// Remote represents a git remote
type Remote struct {
	Name string // "origin", "upstream", etc.
	URL  string // Remote URL
}

// RepoMode defines how to handle a repo
type RepoMode int

const (
	ModeLeaveUntouched   RepoMode = iota // Skip this repo
	ModePushKnownBranches                // Push only branches that exist on remote
	ModePushAll                          // Push all branches
	ModePushCurrentBranch                // Push only current checked-out branch
)

func (m RepoMode) String() string {
	switch m {
	case ModeLeaveUntouched:
		return "leave-untouched"
	case ModePushKnownBranches:
		return "push-known-branches"
	case ModePushAll:
		return "push-all"
	case ModePushCurrentBranch:
		return "push-current-branch"
	default:
		return "unknown"
	}
}

// ParseMode converts a mode string back to a RepoMode constant.
// Unknown strings return ModeLeaveUntouched to fail closed (no accidental pushes).
func ParseMode(s string) RepoMode {
	switch s {
	case "leave-untouched":
		return ModeLeaveUntouched
	case "", "push-known-branches":
		return ModePushKnownBranches
	case "push-all":
		return ModePushAll
	case "push-current-branch":
		return ModePushCurrentBranch
	default:
		return ModeLeaveUntouched
	}
}

// ScanOptions configures repository scanning
type ScanOptions struct {
	// Root path to start scanning from
	RootPath string

	// Exclude patterns (directories to skip)
	Exclude []string

	// Max directory depth
	MaxDepth int

	// Use cached results if available
	UseCache bool

	// Cache file path
	CacheFile string

	// Cache TTL
	CacheTTL time.Duration

	// Parallel workers
	Workers int

	// KnownPaths maps absolute repo paths already in the registry to their
	// rescan_submodules flag. When rescan_submodules is false the scanner skips
	// descending into the directory (it is already known), but still analyzes
	// it for fresh metadata. When true the scanner descends normally so that
	// new submodules inside the repo can be discovered.
	KnownPaths map[string]bool
}

// DefaultScanOptions returns sensible defaults
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		RootPath: ".",
		Exclude: []string{
			".cache",
			"node_modules",
			".venv",
			"venv",
			"vendor",
			"dist",
			"build",
			"target",
		},
		MaxDepth:  10,
		UseCache:  true,
		CacheFile: "",
		CacheTTL:  24 * time.Hour,
		Workers:   8,
	}
}
