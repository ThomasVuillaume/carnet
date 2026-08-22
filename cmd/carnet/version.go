package main

import (
	"fmt"
	"io"
	"runtime/debug"
	"strings"
)

// binaryName leads the version line, as git and go do with theirs. A bare
// version string would leave a piped log without a subject.
const binaryName = "carnet"

// shortRevisionLen is the git SHA prefix printed. Seven characters is what git
// itself abbreviates to, and what a reader will paste back into git show.
const shortRevisionLen = 7

// printVersion writes the build version to w.
//
// The impure half of the pair. debug.ReadBuildInfo reads the metadata blob the
// linker wrote into this binary: it takes no argument and cannot be varied from
// a test. Everything worth testing lives in formatVersion.
func printVersion(w io.Writer) error {
	// info is nil exactly when ok is false, so the bool carries nothing the
	// pointer does not. Dropping it keeps formatVersion to one parameter.
	info, _ := debug.ReadBuildInfo()

	if _, err := fmt.Fprintln(w, formatVersion(info)); err != nil {
		return fmt.Errorf("writing version: %w", err)
	}

	return nil
}

// buildSetting returns the value the toolchain recorded for key, or "" when the
// key is absent. The keys worth reading here: vcs.revision, vcs.time,
// vcs.modified.
//
// Settings is a slice and is walked as one. Copying it into a map to index it
// would make any multi-key output depend on map iteration order.
func buildSetting(info *debug.BuildInfo, key string) string {
	if info == nil {
		return ""
	}

	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}

	return ""
}

// shortRevision abbreviates a git SHA. A revision shorter than the cut is
// returned whole rather than padded: it is not a SHA, and inventing characters
// would be worse than printing what is there.
func shortRevision(rev string) string {
	if len(rev) < shortRevisionLen {
		return rev
	}

	return rev[:shortRevisionLen]
}

// formatVersion renders the single line printed by "carnet version".
//
// Four shapes reach this function, all measured on this repository:
//
//   - go build, untagged: Main.Version is the pseudo-version the toolchain
//     synthesises from the commit — v0.0.0-20260821162324-952755a5a2a5+dirty.
//     The +dirty suffix appears whenever the tree has uncommitted changes.
//   - go build, tagged: Main.Version is the tag, v1.2.3.
//   - go run, go test, or -buildvcs=false: Main.Version is "(devel)" and
//     Settings holds no vcs.* key at all, so the revision fallback finds
//     nothing. go run stamps nothing, so checking the real line takes go build.
//   - info nil: the binary carries no metadata whatsoever.
//
// Nothing here reads the clock. vcs.time would be stable — it is the commit
// date, not the build date — but it is left out as noise the reader can get
// from the SHA.
func formatVersion(info *debug.BuildInfo) string {
	if info == nil {
		return binaryName + " unknown"
	}

	version := info.Main.Version

	// The pseudo-version already embeds the commit, a tag does not. Testing for
	// the abbreviation rather than appending blind keeps the SHA off the line
	// twice. The 7-character form is a prefix of the 12 the pseudo-version
	// carries, so Contains matches both.
	if rev := shortRevision(buildSetting(info, "vcs.revision")); rev != "" && !strings.Contains(version, rev) {
		version += " (" + rev + ")"
	}

	return binaryName + " " + version
}
