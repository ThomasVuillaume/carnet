package main

import (
	"bytes"
	"errors"
	"runtime/debug"
	"strings"
	"testing"
)

// fullRevision is the commit this file was written against. Tests slice it
// rather than hard-coding two spellings of the same SHA.
const fullRevision = "952755a5a2a535ea7f78722b82f9925338ee2e23"

func buildInfo(version, revision string) *debug.BuildInfo {
	info := &debug.BuildInfo{Main: debug.Module{Version: version}}
	if revision != "" {
		info.Settings = []debug.BuildSetting{{Key: "vcs.revision", Value: revision}}
	}

	return info
}

// The table walks the four shapes debug.ReadBuildInfo produces, plus the two
// ways a revision can fail to be a usable SHA. What it discriminates is when
// the abbreviated revision is appended: a tag says nothing about the commit it
// points at, a pseudo-version already spells it out.
func TestFormatVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info *debug.BuildInfo
		want string
	}{
		{
			name: "no build metadata at all",
			info: nil,
			want: "carnet unknown",
		},
		{
			name: "tagged build gains the revision",
			info: buildInfo("v1.2.3", fullRevision),
			want: "carnet v1.2.3 (952755a)",
		},
		{
			name: "pseudo-version already embeds the revision",
			info: buildInfo("v0.0.0-20260821162324-952755a5a2a5", fullRevision),
			want: "carnet v0.0.0-20260821162324-952755a5a2a5",
		},
		{
			name: "dirty tree keeps the suffix the toolchain wrote",
			info: buildInfo("v0.0.0-20260821162324-952755a5a2a5+dirty", fullRevision),
			want: "carnet v0.0.0-20260821162324-952755a5a2a5+dirty",
		},
		{
			name: "test binary has no vcs metadata",
			info: buildInfo("(devel)", ""),
			want: "carnet (devel)",
		},
		{
			name: "revision shorter than the cut is printed whole",
			info: buildInfo("v1.2.3", "abc"),
			want: "carnet v1.2.3 (abc)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := formatVersion(tc.info); got != tc.want {
				t.Errorf("formatVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

// Absence and empty value are one case for the caller, so the table states that
// they collapse to the same "" rather than to distinguishable results.
func TestBuildSetting(t *testing.T) {
	t.Parallel()

	full := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: fullRevision},
		{Key: "vcs.modified", Value: "true"},
		{Key: "GOARCH", Value: ""},
	}}

	tests := []struct {
		name string
		info *debug.BuildInfo
		key  string
		want string
	}{
		{name: "nil info", info: nil, key: "vcs.revision", want: ""},
		{name: "first key of the slice", info: full, key: "vcs.revision", want: fullRevision},
		{name: "later key of the slice", info: full, key: "vcs.modified", want: "true"},
		{name: "key present but empty", info: full, key: "GOARCH", want: ""},
		{name: "key absent", info: full, key: "vcs.time", want: ""},
		{name: "no settings at all", info: &debug.BuildInfo{}, key: "vcs.revision", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := buildSetting(tc.info, tc.key); got != tc.want {
				t.Errorf("buildSetting(%q) = %q, want %q", tc.key, got, tc.want)
			}
		})
	}
}

var errWrite = errors.New("writer refused")

// failingWriter stands in for a closed pipe: carnet version | head -0.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

// This test binary reports "(devel)" with no vcs.* key, so the exact line is
// not assertable here — only its shape, and that the write error travels.
func TestPrintVersion(t *testing.T) {
	t.Parallel()

	t.Run("writes one terminated line to the writer", func(t *testing.T) {
		t.Parallel()

		var buf bytes.Buffer
		if err := printVersion(&buf); err != nil {
			t.Fatalf("printVersion() error = %v", err)
		}

		got := buf.String()
		if !strings.HasPrefix(got, binaryName+" ") {
			t.Errorf("printVersion() wrote %q, want it to start with %q", got, binaryName+" ")
		}

		if !strings.HasSuffix(got, "\n") || strings.Count(got, "\n") != 1 {
			t.Errorf("printVersion() wrote %q, want exactly one trailing newline", got)
		}
	})

	t.Run("wraps the writer error", func(t *testing.T) {
		t.Parallel()

		err := printVersion(failingWriter{})
		if !errors.Is(err, errWrite) {
			t.Errorf("printVersion() error = %v, want it to wrap %v", err, errWrite)
		}
	})
}
