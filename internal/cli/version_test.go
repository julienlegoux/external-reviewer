package cli_test

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
)

// TestRun_Version_PrintsTheBuildItCameFrom is specs 17's whole release
// story: `go install` and no tags, so the only way a report traces back to a
// build is what the toolchain stamps for free. The expectation is derived
// from debug.ReadBuildInfo() inside the test rather than hard-coded, because
// the values differ between the test binary, a `go install`ed one and CI —
// what is asserted is that the command reports what the runtime knows, not
// any particular revision.
func TestRun_Version_PrintsTheBuildItCameFrom(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"version"}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	line := strings.TrimRight(stdout.String(), "\n")
	if line == "" {
		t.Fatal("stdout is empty, want one line naming the build")
	}
	if strings.Contains(line, "\n") {
		t.Errorf("stdout = %q, want exactly one line", stdout.String())
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		// A binary built outside a module context reports what it has and
		// still exits 0, which the assertions above already cover.
		t.Log("debug.ReadBuildInfo reported nothing; the shape assertions above are the whole contract here")
		return
	}
	if !strings.Contains(line, info.Main.Path) {
		t.Errorf("version line = %q, want it to name the module path %q", line, info.Main.Path)
	}
	if !strings.Contains(line, info.Main.Version) && !strings.Contains(line, buildRevision(info)) {
		t.Errorf("version line = %q, want it to carry the version %q or the revision %q",
			line, info.Main.Version, buildRevision(info))
	}

	assertOnlyKnownPrefixedLines(t, stderr.String())
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "version" {
		t.Errorf("done stop = %q, want version", fields.Stop)
	}
}

// buildRevision is the vcs.revision setting the toolchain stamps, or the
// empty string when nothing stamped one — a test binary is often built
// without it, which is exactly why the assertion above accepts either the
// version or the revision.
func buildRevision(info *debug.BuildInfo) string {
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

// TestRun_Version_TakesNoArguments keeps `version` from quietly accepting
// something it does not act on: a caller that typed `version --json` has to
// learn that nothing consumed the flag, rather than getting the plain line
// and believing it asked for something else.
func TestRun_Version_TakesNoArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"version", "--json"}, strings.NewReader(""), &stdout, &stderr)

	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if n := len(errorLines(stderr.String())); n != 1 {
		t.Fatalf("stderr carries %d error: lines, want exactly 1: %q", n, stderr.String())
	}
	if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.Stop)
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
}
