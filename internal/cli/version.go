package cli

import (
	"fmt"
	"io"
	"runtime/debug"
)

// unknownBuildValue is what a field ReadBuildInfo did not fill in renders as.
// The line keeps its shape either way: a caller pasting it into a report
// should not have to guess whether a missing field means "not stamped" or
// "the command forgot to print it".
const unknownBuildValue = "unknown"

// versionLine renders the one line `version` prints, from what the toolchain
// stamped into the binary.
//
// specs 17 is the whole reason this exists and the whole reason it is this
// small: there are no tags, no release workflow and no CHANGELOG in v1 —
// `go install …@latest` resolves to the default branch's newest commit — so
// the VCS revision the toolchain stamps for free is the only thing that ties
// a report back to a build.
//
// ok is ReadBuildInfo's own second return, threaded in rather than called
// here so the rendering is testable without a build context. A binary built
// outside a module context reports what it has, and still exits 0: `version`
// failing would be a worse answer than an honest "unknown", since the caller
// asking is already trying to work out what it is holding.
func versionLine(info *debug.BuildInfo, ok bool) string {
	module, version, revision, dirty := unknownBuildValue, unknownBuildValue, unknownBuildValue, unknownBuildValue
	if ok && info != nil {
		if info.Main.Path != "" {
			module = info.Main.Path
		}
		if info.Main.Version != "" {
			version = info.Main.Version
		}
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				revision = setting.Value
			case "vcs.modified":
				dirty = setting.Value
			}
		}
	}
	return fmt.Sprintf("external-reviewer module=%s version=%s revision=%s dirty=%s\n",
		module, version, revision, dirty)
}

// printVersion writes the version line to w. Like printUsage, it writes to
// stdout and the caller decides the exit code around it — `version` is a
// question with an answer, not a review, so the answer goes where an answer
// goes.
func printVersion(w io.Writer) {
	_, _ = fmt.Fprint(w, versionLine(debug.ReadBuildInfo()))
}
