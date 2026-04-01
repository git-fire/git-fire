package executor

import (
	"sync"
	"testing"
	"time"
)

func TestNewHostLimiter(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter := NewHostLimiter(config)

	if limiter == nil {
		t.Fatal("Expected limiter to be created")
	}

	if limiter.config.GlobalLimit != 10 {
		t.Errorf("Expected global limit 10, got %d", limiter.config.GlobalLimit)
	}

	if limiter.config.PerHostLimit != 5 {
		t.Errorf("Expected per-host limit 5, got %d", limiter.config.PerHostLimit)
	}
}

func TestHostLimiter_ExtractHost(t *testing.T) {
	limiter := NewHostLimiter(DefaultRateLimitConfig())

	tests := []struct {
		name      string
		remoteURL string
		wantHost  string
	}{
		{
			name:      "SSH format github",
			remoteURL: "git@github.com:user/repo.git",
			wantHost:  "github.com",
		},
		{
			name:      "SSH format gitlab",
			remoteURL: "git@gitlab.com:user/repo.git",
			wantHost:  "gitlab.com",
		},
		{
			name:      "HTTPS github",
			remoteURL: "https://github.com/user/repo.git",
			wantHost:  "github.com",
		},
		{
			name:      "HTTP gitlab",
			remoteURL: "http://gitlab.com/user/repo.git",
			wantHost:  "gitlab.com",
		},
		{
			name:      "SSH with custom port",
			remoteURL: "ssh://git@github.com:22/user/repo.git",
			wantHost:  "github.com", // Port is parsed separately by url.Parse
		},
		{
			name:      "Local path",
			remoteURL: "/tmp/repo.git",
			wantHost:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := limiter.extractHost(tt.remoteURL)
			if got != tt.wantHost {
				t.Errorf("extractHost(%q) = %q, want %q", tt.remoteURL, got, tt.wantHost)
			}
		})
	}
}

func TestHostLimiter_GetLimitForHost(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter := NewHostLimiter(config)

	tests := []struct {
		name      string
		host      string
		wantLimit int
	}{
		{
			name:      "GitHub specific limit",
			host:      "github.com",
			wantLimit: 3,
		},
		{
			name:      "GitLab specific limit",
			host:      "gitlab.com",
			wantLimit: 5,
		},
		{
			name:      "Localhost limit",
			host:      "localhost",
			wantLimit: 10,
		},
		{
			name:      "Private network 192.168",
			host:      "192.168.1.1",
			wantLimit: 10,
		},
		{
			name:      "Private network 10.x",
			host:      "10.0.0.1",
			wantLimit: 10,
		},
		{
			name:      "Unknown host uses default",
			host:      "example.com",
			wantLimit: 5, // Default per-host limit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter.mu.Lock()
			got := limiter.getLimitForHost(tt.host)
			limiter.mu.Unlock()

			if got != tt.wantLimit {
				t.Errorf("getLimitForHost(%q) = %d, want %d", tt.host, got, tt.wantLimit)
			}
		})
	}
}

func TestHostLimiter_AcquireRelease(t *testing.T) {
	config := RateLimitConfig{
		GlobalLimit:  10,
		PerHostLimit: 2,
		PerHostDelay: 0,
		HostLimits:   make(map[string]int),
	}

	limiter := NewHostLimiter(config)

	remoteURL := "git@github.com:user/repo.git"

	// Acquire first slot
	limiter.Acquire(remoteURL)

	// Acquire second slot
	limiter.Acquire(remoteURL)

	// Check stats
	stats := limiter.Stats()
	if stats["github.com"].CurrentActive != 2 {
		t.Errorf("Expected 2 active slots, got %d", stats["github.com"].CurrentActive)
	}

	// Release one slot
	limiter.Release(remoteURL)

	// Check stats again
	stats = limiter.Stats()
	if stats["github.com"].CurrentActive != 1 {
		t.Errorf("Expected 1 active slot after release, got %d", stats["github.com"].CurrentActive)
	}

	// Release second slot
	limiter.Release(remoteURL)

	// Check stats
	stats = limiter.Stats()
	if stats["github.com"].CurrentActive != 0 {
		t.Errorf("Expected 0 active slots after both releases, got %d", stats["github.com"].CurrentActive)
	}
}

func TestHostLimiter_ConcurrencyLimit(t *testing.T) {
	config := RateLimitConfig{
		GlobalLimit:  10,
		PerHostLimit: 2, // Only allow 2 concurrent operations
		PerHostDelay: 0,
		HostLimits:   make(map[string]int),
	}

	limiter := NewHostLimiter(config)
	remoteURL := "git@github.com:user/repo.git"

	var wg sync.WaitGroup
	var mu sync.Mutex
	maxConcurrent := 0
	currentConcurrent := 0

	// Launch 5 goroutines that try to acquire
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			limiter.Acquire(remoteURL)

			// Track concurrent operations
			mu.Lock()
			currentConcurrent++
			if currentConcurrent > maxConcurrent {
				maxConcurrent = currentConcurrent
			}
			mu.Unlock()

			// Simulate work
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			currentConcurrent--
			mu.Unlock()

			limiter.Release(remoteURL)
		}()
	}

	wg.Wait()

	// Max concurrent should not exceed limit
	if maxConcurrent > 2 {
		t.Errorf("Max concurrent operations %d exceeded limit of 2", maxConcurrent)
	}

	t.Logf("Max concurrent operations: %d (limit: 2)", maxConcurrent)
}

func TestHostLimiter_MultipleHosts(t *testing.T) {
	config := RateLimitConfig{
		GlobalLimit:  10,
		PerHostLimit: 2,
		PerHostDelay: 0,
		HostLimits:   make(map[string]int),
	}

	limiter := NewHostLimiter(config)

	githubURL := "git@github.com:user/repo.git"
	gitlabURL := "git@gitlab.com:user/repo.git"

	// Acquire slots for both hosts
	limiter.Acquire(githubURL)
	limiter.Acquire(gitlabURL)

	stats := limiter.Stats()

	// Should have separate limits for each host
	if len(stats) != 2 {
		t.Errorf("Expected stats for 2 hosts, got %d", len(stats))
	}

	if stats["github.com"].CurrentActive != 1 {
		t.Errorf("Expected 1 active slot for github.com, got %d", stats["github.com"].CurrentActive)
	}

	if stats["gitlab.com"].CurrentActive != 1 {
		t.Errorf("Expected 1 active slot for gitlab.com, got %d", stats["gitlab.com"].CurrentActive)
	}

	limiter.Release(githubURL)
	limiter.Release(gitlabURL)
}

func TestHostLimiter_GlobalLimitAcrossHosts(t *testing.T) {
	config := RateLimitConfig{
		GlobalLimit:  1,
		PerHostLimit: 10,
		PerHostDelay: 0,
		HostLimits:   make(map[string]int),
	}
	limiter := NewHostLimiter(config)

	first := make(chan struct{})
	done := make(chan struct{})
	go func() {
		limiter.Acquire("git@github.com:user/repo.git")
		close(first)
		time.Sleep(50 * time.Millisecond)
		limiter.Release("git@github.com:user/repo.git")
		close(done)
	}()
	<-first

	start := time.Now()
	limiter.Acquire("git@gitlab.com:user/repo.git")
	elapsed := time.Since(start)
	limiter.Release("git@gitlab.com:user/repo.git")
	<-done

	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected global limiter to block second acquire, blocked only %v", elapsed)
	}
}

func TestHostLimiter_WithDelay(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping delay test in short mode")
	}

	config := RateLimitConfig{
		GlobalLimit:  10,
		PerHostLimit: 5,
		PerHostDelay: 100 * time.Millisecond, // 100ms delay between operations
		HostLimits:   make(map[string]int),
	}

	limiter := NewHostLimiter(config)
	remoteURL := "git@github.com:user/repo.git"

	// First acquire - should be immediate
	start := time.Now()
	limiter.Acquire(remoteURL)
	limiter.Release(remoteURL)
	firstDuration := time.Since(start)

	// Second acquire - should wait for delay
	start = time.Now()
	limiter.Acquire(remoteURL)
	limiter.Release(remoteURL)
	secondDuration := time.Since(start)

	// Second operation should have taken at least the delay time
	if secondDuration < config.PerHostDelay {
		t.Errorf("Expected delay of at least %v, got %v", config.PerHostDelay, secondDuration)
	}

	t.Logf("First operation: %v, Second operation: %v (delay: %v)",
		firstDuration, secondDuration, config.PerHostDelay)
}

func TestHostLimiter_Stats(t *testing.T) {
	config := DefaultRateLimitConfig()
	limiter := NewHostLimiter(config)

	// Initially, no stats
	stats := limiter.Stats()
	if len(stats) != 0 {
		t.Errorf("Expected 0 stats initially, got %d", len(stats))
	}

	// Acquire some slots
	limiter.Acquire("git@github.com:user/repo1.git")
	limiter.Acquire("git@gitlab.com:user/repo2.git")

	stats = limiter.Stats()

	// Should have stats for 2 hosts
	if len(stats) != 2 {
		t.Errorf("Expected 2 hosts in stats, got %d", len(stats))
	}

	// Check individual stats
	if githubStats, ok := stats["github.com"]; ok {
		if githubStats.Host != "github.com" {
			t.Errorf("Expected host 'github.com', got %q", githubStats.Host)
		}

		if githubStats.Limit != 3 { // GitHub has custom limit of 3
			t.Errorf("Expected limit 3 for GitHub, got %d", githubStats.Limit)
		}

		if githubStats.CurrentActive != 1 {
			t.Errorf("Expected 1 active operation, got %d", githubStats.CurrentActive)
		}
	} else {
		t.Error("Expected stats for github.com")
	}

	limiter.Release("git@github.com:user/repo1.git")
	limiter.Release("git@gitlab.com:user/repo2.git")
}

func TestParseSSHURL(t *testing.T) {
	tests := []struct {
		name     string
		sshURL   string
		wantUser string
		wantHost string
		wantPath string
	}{
		{
			name:     "Standard GitHub URL",
			sshURL:   "git@github.com:user/repo.git",
			wantUser: "git",
			wantHost: "github.com",
			wantPath: "user/repo.git",
		},
		{
			name:     "GitLab URL",
			sshURL:   "git@gitlab.com:group/project.git",
			wantUser: "git",
			wantHost: "gitlab.com",
			wantPath: "group/project.git",
		},
		{
			name:     "Custom user",
			sshURL:   "myuser@example.com:path/to/repo.git",
			wantUser: "myuser",
			wantHost: "example.com",
			wantPath: "path/to/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := parseSSHURL(tt.sshURL)

			if len(parts) != 3 {
				t.Fatalf("Expected 3 parts, got %d: %v", len(parts), parts)
			}

			if parts[0] != tt.wantUser {
				t.Errorf("User = %q, want %q", parts[0], tt.wantUser)
			}

			if parts[1] != tt.wantHost {
				t.Errorf("Host = %q, want %q", parts[1], tt.wantHost)
			}

			if parts[2] != tt.wantPath {
				t.Errorf("Path = %q, want %q", parts[2], tt.wantPath)
			}
		})
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.GlobalLimit <= 0 {
		t.Error("Global limit should be positive")
	}

	if config.PerHostLimit <= 0 {
		t.Error("Per-host limit should be positive")
	}

	// Check specific host limits
	expectedLimits := map[string]int{
		"github.com": 3,
		"gitlab.com": 5,
	}

	for host, expectedLimit := range expectedLimits {
		if limit, ok := config.HostLimits[host]; !ok {
			t.Errorf("Missing limit for %s", host)
		} else if limit != expectedLimit {
			t.Errorf("Limit for %s = %d, want %d", host, limit, expectedLimit)
		}
	}
}
