package executor

import (
	"net/url"
	"sync"
	"time"
)

// HostLimiter provides per-host rate limiting for push operations
// This prevents overwhelming a single git host (e.g., GitHub) with too many concurrent pushes
type HostLimiter struct {
	limits   map[string]chan struct{} // host -> semaphore channel
	global   chan struct{}
	delays   map[string]time.Duration // host -> delay between operations
	lastPush map[string]time.Time     // host -> timestamp of last push
	mu       sync.Mutex
	config   RateLimitConfig
}

// RateLimitConfig configures rate limiting behavior
type RateLimitConfig struct {
	// Maximum concurrent pushes globally
	GlobalLimit int

	// Maximum concurrent pushes per host (default: 5)
	PerHostLimit int

	// Delay between pushes to the same host (default: 0)
	PerHostDelay time.Duration

	// Custom limits for specific hosts
	HostLimits map[string]int // host pattern -> limit
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalLimit:  10, // Max 10 concurrent pushes total
		PerHostLimit: 5,  // Max 5 concurrent pushes per host
		PerHostDelay: 0,  // No delay by default
		HostLimits: map[string]int{
			"github.com": 3,                 // Be conservative with GitHub
			"gitlab.com": 5,                 // GitLab can handle more
			"localhost":  10,                // Local git servers can handle many
			"192.168.":   10,                // Local network, no limits
			"10.":        10,                // Private network
			"172.16.":    10,                // Private network
		},
	}
}

// NewHostLimiter creates a new host-based rate limiter
func NewHostLimiter(config RateLimitConfig) *HostLimiter {
	globalLimit := config.GlobalLimit
	if globalLimit <= 0 {
		globalLimit = 1
	}
	return &HostLimiter{
		limits:   make(map[string]chan struct{}),
		global:   make(chan struct{}, globalLimit),
		delays:   make(map[string]time.Duration),
		lastPush: make(map[string]time.Time),
		config:   config,
	}
}

// Acquire acquires a rate limit slot for the given remote URL
// Blocks until a slot is available
func (h *HostLimiter) Acquire(remoteURL string) {
	host := h.extractHost(remoteURL)
	if host == "" {
		return // No rate limiting for unparseable URLs
	}

	h.mu.Lock()

	// Get or create semaphore for this host
	sem := h.getSemaphoreForHost(host)

	// Get delay for this host
	delay := h.getDelayForHost(host)

	// Check if we need to wait due to delay
	if delay > 0 {
		lastPush, exists := h.lastPush[host]
		if exists {
			elapsed := time.Since(lastPush)
			if elapsed < delay {
				waitTime := delay - elapsed
				h.mu.Unlock()
				time.Sleep(waitTime)
				h.mu.Lock()
			}
		}
	}

	h.mu.Unlock()

	// Acquire global semaphore first.
	h.global <- struct{}{}
	// Acquire semaphore (blocks if at limit)
	sem <- struct{}{}
}

// Release releases a rate limit slot for the given remote URL
func (h *HostLimiter) Release(remoteURL string) {
	host := h.extractHost(remoteURL)
	if host == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Update last push timestamp
	h.lastPush[host] = time.Now()

	// Release semaphore
	if sem, exists := h.limits[host]; exists {
		<-sem
	}
	<-h.global
}

// getSemaphoreForHost gets or creates a semaphore channel for a host
// Must be called with lock held
func (h *HostLimiter) getSemaphoreForHost(host string) chan struct{} {
	if sem, exists := h.limits[host]; exists {
		return sem
	}

	// Determine limit for this host
	limit := h.getLimitForHost(host)

	// Create new semaphore
	sem := make(chan struct{}, limit)
	h.limits[host] = sem

	return sem
}

// getLimitForHost determines the concurrency limit for a host
// Must be called with lock held
func (h *HostLimiter) getLimitForHost(host string) int {
	// Check for exact match first
	if limit, exists := h.config.HostLimits[host]; exists {
		return limit
	}

	// Check for prefix matches (e.g., "192.168." matches "192.168.1.1")
	for pattern, limit := range h.config.HostLimits {
		if len(pattern) > 0 && len(host) > 0 {
			// Check if host starts with pattern
			if len(host) >= len(pattern) && host[:len(pattern)] == pattern {
				return limit
			}
		}
	}

	// Use default per-host limit
	return h.config.PerHostLimit
}

// getDelayForHost determines the delay for a host
func (h *HostLimiter) getDelayForHost(host string) time.Duration {
	if delay, exists := h.delays[host]; exists {
		return delay
	}

	// Use default delay
	return h.config.PerHostDelay
}

// extractHost extracts the hostname from a git remote URL
func (h *HostLimiter) extractHost(remoteURL string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if len(remoteURL) > 0 && remoteURL[0] != 'h' {
		// Likely SSH format
		parts := parseSSHURL(remoteURL)
		if len(parts) >= 2 {
			return parts[1]
		}
	}

	// Handle HTTP(S) URLs
	u, err := url.Parse(remoteURL)
	if err == nil && u.Host != "" {
		return u.Host
	}

	// Fallback: try to extract from SSH format manually
	// Format: [user@]host:path
	if atIndex := indexOf(remoteURL, '@'); atIndex >= 0 {
		rest := remoteURL[atIndex+1:]
		if colonIndex := indexOf(rest, ':'); colonIndex >= 0 {
			return rest[:colonIndex]
		}
	}

	// Could not determine host
	return ""
}

// parseSSHURL parses SSH-style git URLs
// Format: git@github.com:user/repo.git
// Returns: [user, host, path]
func parseSSHURL(sshURL string) []string {
	var parts []string

	// Split on @
	atIndex := indexOf(sshURL, '@')
	if atIndex < 0 {
		return parts
	}

	user := sshURL[:atIndex]
	rest := sshURL[atIndex+1:]

	// Split on :
	colonIndex := indexOf(rest, ':')
	if colonIndex < 0 {
		return parts
	}

	host := rest[:colonIndex]
	path := rest[colonIndex+1:]

	return []string{user, host, path}
}

// indexOf finds the first index of char in s
func indexOf(s string, char byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == char {
			return i
		}
	}
	return -1
}

// Stats returns statistics about current rate limiting state
func (h *HostLimiter) Stats() map[string]RateLimitStats {
	h.mu.Lock()
	defer h.mu.Unlock()

	stats := make(map[string]RateLimitStats)

	for host, sem := range h.limits {
		stats[host] = RateLimitStats{
			Host:          host,
			Limit:         cap(sem),
			CurrentActive: len(sem),
			LastPush:      h.lastPush[host],
		}
	}

	return stats
}

// RateLimitStats provides statistics for a host
type RateLimitStats struct {
	Host          string
	Limit         int
	CurrentActive int
	LastPush      time.Time
}
