package cmd

import "testing"

func TestPickCLIVersion(t *testing.T) {
	tests := []struct {
		name     string
		ldflags  string
		mainMod  string
		want     string
	}{
		{name: "release tag from ldflags", ldflags: "v1.2.3", mainMod: "v0.0.0", want: "v1.2.3"},
		{name: "describe style from ldflags", ldflags: "v0.2.1-3-gabc", mainMod: "", want: "v0.2.1-3-gabc"},
		{name: "dev uses main module", ldflags: "dev", mainMod: "v0.2.2-0.20260101120000-abcdef123456", want: "v0.2.2-0.20260101120000-abcdef123456"},
		{name: "bare hash ldflags uses main module", ldflags: "98e0d4ff6d06", mainMod: "v0.2.2-0.20260101120000-abcdef123456", want: "v0.2.2-0.20260101120000-abcdef123456"},
		{name: "bare hash uppercase", ldflags: "98E0D4F", mainMod: "v1.0.0", want: "v1.0.0"},
		{name: "dev and devel stays dev", ldflags: "dev", mainMod: "(devel)", want: "dev"},
		{name: "bare hash no main mod", ldflags: "abc1234", mainMod: "", want: "abc1234"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickCLIVersion(tt.ldflags, tt.mainMod); got != tt.want {
				t.Fatalf("pickCLIVersion(%q, %q) = %q, want %q", tt.ldflags, tt.mainMod, got, tt.want)
			}
		})
	}
}

func TestIsBareGitHash(t *testing.T) {
	if !isBareGitHash("98e0d4f") {
		t.Fatal("expected short hash")
	}
	if !isBareGitHash("98e0d4ff6d0612345678901234567890abcdef") {
		t.Fatal("expected 40-char hash")
	}
	if isBareGitHash("v0.2.1") {
		t.Fatal("tag is not bare hash")
	}
	if isBareGitHash("v0.2.1-1-g98e0d4f") {
		t.Fatal("describe output is not bare hash")
	}
	if isBareGitHash("dev") {
		t.Fatal("dev is not bare hash")
	}
}
