package plugins

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ConflictContext is passed to merge-conflict resolver plugins after divergence
// is detected between the local branch and a remote tracking branch.
type ConflictContext struct {
	RepoPath  string
	RepoName  string
	Branch    string
	Remote    string
	LocalSHA  string
	RemoteSHA string

	Timestamp time.Time
	DryRun    bool

	Logger Logger
}

// ConflictResolutionResult is returned by a conflict resolver; Resolved means
// the divergence is gone (local can fast-forward or push without the fire-branch path).
type ConflictResolutionResult struct {
	Resolved bool
}

// ConflictResolver runs when git-fire detects local/remote divergence for a remote.
type ConflictResolver interface {
	Name() string
	ResolveConflict(ctx ConflictContext) (ConflictResolutionResult, error)
}

var (
	conflictMu        sync.RWMutex
	conflictResolvers []ConflictResolver
)

// RegisterConflictResolver registers a resolver invoked during planning when a
// merge conflict (divergence) is detected. Resolvers are run in registration order.
func RegisterConflictResolver(r ConflictResolver) error {
	if r == nil {
		return fmt.Errorf("cannot register nil conflict resolver")
	}
	if r.Name() == "" {
		return fmt.Errorf("conflict resolver name cannot be empty")
	}

	conflictMu.Lock()
	defer conflictMu.Unlock()

	for _, existing := range conflictResolvers {
		if existing.Name() == r.Name() {
			return fmt.Errorf("conflict resolver %s already registered", r.Name())
		}
	}
	conflictResolvers = append(conflictResolvers, r)
	return nil
}

// ListConflictResolvers returns a copy of registered conflict resolvers.
func ListConflictResolvers() []ConflictResolver {
	conflictMu.RLock()
	defer conflictMu.RUnlock()

	out := make([]ConflictResolver, len(conflictResolvers))
	copy(out, conflictResolvers)
	return out
}

// ClearConflictResolvers removes all conflict resolvers (for tests).
func ClearConflictResolvers() {
	conflictMu.Lock()
	defer conflictMu.Unlock()
	conflictResolvers = nil
}

// ParseConflictResolvedLine interprets the first line of plugin stdout as a boolean.
// Accepted true values: true, yes, 1, resolved (case-insensitive).
// Accepted false values: false, no, 0, unresolved, fail (case-insensitive).
// Any other non-empty line is treated as false. Empty output is false.
func ParseConflictResolvedLine(stdout string) bool {
	line := strings.TrimSpace(stdout)
	if line == "" {
		return false
	}
	if idx := strings.IndexAny(line, "\r\n"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
	}
	switch strings.ToLower(line) {
	case "true", "yes", "1", "resolved":
		return true
	case "false", "no", "0", "unresolved", "fail":
		return false
	default:
		return false
	}
}

// DiscardLogger is a no-op Logger for contexts where plugin output is not shown.
type discardLogger struct{}

func (discardLogger) Info(string)    {}
func (discardLogger) Success(string) {}
func (discardLogger) Error(string, error) {}
func (discardLogger) Debug(string)   {}

// DiscardLogger satisfies Logger with no output.
var DiscardLogger Logger = discardLogger{}
