// Package buildinfo exposes version metadata injected at link time.
package buildinfo

import "strings"

var (
	version = "dev"
	commit  = "unknown"
)

// Info is the current build identity.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

// Current returns the linked build identity with dev fallbacks.
func Current() Info {
	return Info{Version: fallback(version, "dev"), Commit: fallback(commit, "unknown")}
}

func fallback(value string, def string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return def
}
