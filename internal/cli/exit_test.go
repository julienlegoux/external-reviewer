package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
)

// doneLineRE parses the "done" line's key=value fields out of a run's stderr
// rather than matching it verbatim — elapsed time is real. Shared by every
// test in this package that asserts on a termination's done line.
var doneLineRE = regexp.MustCompile(
	`(?m)^done\s+turns=(\d+)\s+in=(\d+) out=(\d+)\s+\$([0-9.]+)\s+(\S+)\s+stop=(\S+)\s*$`)

// doneFields holds a parsed done line's key=value fields for assertions.
type doneFields struct {
	turns, in, out, cost, elapsed, stop string
}

// parseDoneLine finds and parses the done line in stderr, failing the test
// if none is present — every termination path must emit exactly one.
func parseDoneLine(t *testing.T, stderr string) doneFields {
	t.Helper()
	m := doneLineRE.FindStringSubmatch(stderr)
	if m == nil {
		t.Fatalf("stderr = %q, want a done line matching %s", stderr, doneLineRE)
	}
	return doneFields{turns: m[1], in: m[2], out: m[3], cost: m[4], elapsed: m[5], stop: m[6]}
}

// TestRun_NoReviewerReached_ExitsOne exercises the exit-1 path — a review
// that never reaches a model — through Run, with the wired reviewer stubbed
// (via SetReviewerForTest) to return cli.ErrNoReviewer wrapped two levels
// deep, since real model resolution arrives in a later issue. Classification
// must still land on exit 1 through the wrapping, proving errors.Is is what
// decides it rather than error text.
func TestRun_NoReviewerReached_ExitsOne(t *testing.T) {
	wrapped := fmt.Errorf("resolving model: %w", fmt.Errorf("checking auth: %w", cli.ErrNoReviewer))
	restore := cli.SetReviewerForTest(func(context.Context, string, string) error {
		return wrapped
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, want 1 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	fields := parseDoneLine(t, stderr.String())
	if fields.in != "0" || fields.out != "0" {
		t.Errorf("done in/out = %q/%q, want 0/0 (a path that never reached a model)", fields.in, fields.out)
	}
	if fields.stop != "no_reviewer" {
		t.Errorf("done stop = %q, want no_reviewer", fields.stop)
	}
}

// TestRun_ReachedAndFailed_ExitsTwo exercises the exit-2 "reached and
// unusable" path for a failure that is not a usage error — e.g. a failed
// turn — again through a stubbed reviewer, since issues 04/05 supply the
// real failure modes.
func TestRun_ReachedAndFailed_ExitsTwo(t *testing.T) {
	restore := cli.SetReviewerForTest(func(context.Context, string, string) error {
		return fmt.Errorf("turn failed: stop reason error")
	})
	defer restore()

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "failed" {
		t.Errorf("done stop = %q, want failed", fields.stop)
	}
}

// TestRun_Success_ExitsZero exercises the exit-0 path with the default
// (stub) reviewer, which every syntactically valid review reaches until
// issues 04/05 replace it.
func TestRun_Success_ExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "ok" {
		t.Errorf("done stop = %q, want ok", fields.stop)
	}
}
