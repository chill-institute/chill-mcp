package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestCurrentVersionPrecedence(t *testing.T) {
	tests := []struct {
		name    string
		linked  string
		module  string
		hasInfo bool
		vcs     bool
		want    string
	}{
		{name: "linked version wins", linked: "1.2.3", module: "v9.9.9", hasInfo: true, want: "1.2.3"},
		{name: "module version from go install", module: "v1.4.0", hasInfo: true, want: "1.4.0"},
		{name: "local build", module: "(devel)", hasInfo: true, want: "dev"},
		{name: "local build in a git checkout", module: "v1.4.1-0.20260101000000-abcdef123456+dirty", hasInfo: true, vcs: true, want: "dev"},
		{name: "no build info", want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origVersion, origRead := version, readBuildInfo
			t.Cleanup(func() { version, readBuildInfo = origVersion, origRead })
			if tt.linked != "" {
				version = tt.linked
			}
			readBuildInfo = func() (*debug.BuildInfo, bool) {
				if !tt.hasInfo {
					return nil, false
				}
				info := &debug.BuildInfo{Main: debug.Module{Version: tt.module}}
				if tt.vcs {
					info.Settings = []debug.BuildSetting{{Key: "vcs", Value: "git"}}
				}
				return info, true
			}
			if got := Current().Version; got != tt.want {
				t.Fatalf("Current().Version = %q, want %q", got, tt.want)
			}
		})
	}
}
