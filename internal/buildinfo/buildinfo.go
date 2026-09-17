// Package buildinfo exposes version metadata injected at link time.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "unknown"
)

// Info is the current build identity.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Current returns the linked build identity. Without ldflags it uses the
// module version recorded by go install, so a released binary never reports
// "dev" and never selects the dev chilly profile.
func Current() Info {
	v := fallback(version, "")
	if v == "" {
		v = moduleVersion()
	}
	return Info{Version: fallback(v, "dev"), Commit: fallback(commit, "unknown")}
}

func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return ""
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}

func fallback(value string, def string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return def
}
