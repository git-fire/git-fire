package cmd

import (
	"regexp"
	"runtime/debug"
)

var bareGitHashRe = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)

// resolvedCLIVersion picks the string shown for --version. Release binaries set
// Version via -ldflags; local builds may only have a bare hash from mis-tuned
// git describe or "dev" when built with plain go build — then we prefer the
// main module version from the build info (e.g. go install pseudo-version).
func resolvedCLIVersion(linked string) string {
	mainMod := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		mainMod = info.Main.Version
	}
	return pickCLIVersion(linked, mainMod)
}

// CLIVersion returns the same version string shown by --version (release tag,
// go install pseudo-version, or dev).
func CLIVersion() string {
	return resolvedCLIVersion(Version)
}

func pickCLIVersion(ldflags, mainMod string) string {
	if ldflags != "dev" && !isBareGitHash(ldflags) {
		return ldflags
	}
	if mainMod != "" && mainMod != "(devel)" {
		return mainMod
	}
	if ldflags != "" {
		return ldflags
	}
	return "dev"
}

func isBareGitHash(s string) bool {
	return bareGitHashRe.MatchString(s)
}
