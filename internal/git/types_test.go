package git

import "testing"

func TestRepoMode_String(t *testing.T) {
	tests := []struct {
		mode RepoMode
		want string
	}{
		{ModeLeaveUntouched, "leave-untouched"},
		{ModePushKnownBranches, "push-known-branches"},
		{ModePushAll, "push-all"},
		{RepoMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input string
		want  RepoMode
	}{
		{"leave-untouched", ModeLeaveUntouched},
		{"push-known-branches", ModePushKnownBranches},
		{"push-all", ModePushAll},
		// Empty string defaults to push-known-branches (safe default)
		{"", ModePushKnownBranches},
		// Unknown strings fail closed (no accidental pushes)
		{"unknown-garbage", ModeLeaveUntouched},
		{"PUSH-ALL", ModeLeaveUntouched},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseMode(tt.input); got != tt.want {
				t.Errorf("ParseMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMode_RoundTrip(t *testing.T) {
	modes := []RepoMode{ModeLeaveUntouched, ModePushKnownBranches, ModePushAll}
	for _, m := range modes {
		t.Run(m.String(), func(t *testing.T) {
			if got := ParseMode(m.String()); got != m {
				t.Errorf("ParseMode(String()) round-trip failed: got %v, want %v", got, m)
			}
		})
	}
}
