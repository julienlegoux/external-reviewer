package cli_test

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
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

// TestClassify_WrappedErrNoReviewer_IsExitOne is the one assertion that
// survives uniquely from the retired performReview seam: a sentinel wrapped
// two levels deep still resolves to exit 1, proving errors.Is is what
// decides classification rather than error text (issue 03 of Epic 1's
// explicit criterion). It is asserted against classify directly now that no
// stubbed reviewer seam exists to drive it through Run — the behavioral half
// of the exit-1 path (a real resolution that never reaches a model, landing
// on exit 1 with stop=no_reviewer) lives in preflight_test.go's
// TestRun_NotReached_ExitsOne, and the generic "reached and failed is exit 2"
// half lives in preflight_test.go's TestRun_BrokenCredential_ExitsTwo and
// roundtrip_test.go's TestRun_FailedTurn_ExitsTwo.
func TestClassify_WrappedErrNoReviewer_IsExitOne(t *testing.T) {
	wrapped := fmt.Errorf("resolving model: %w", fmt.Errorf("checking auth: %w", cli.ErrNoReviewer))
	if code := cli.ClassifyForTest(wrapped); code != 1 {
		t.Errorf("classify(wrapped ErrNoReviewer) = %d, want 1", code)
	}
}

// TestRun_Success_ExitsZero exercises the exit-0 path end to end: real
// pre-flight resolution and a real streamed round trip against an offline
// registry, terminating normally with the reviewer's answer on stdout.
func TestRun_Success_ExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.RunForTest([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

	if code != 0 {
		t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
	}
	if stdout.String() != scriptedAnswer {
		t.Errorf("stdout = %q, want the reviewer's answer %q", stdout.String(), scriptedAnswer)
	}
	fields := parseDoneLine(t, stderr.String())
	// "stop" is kern-link's own StopReasonStop — faux.TextMessage's default
	// when no AssistantMessageOptions.StopReason is scripted. The done line's
	// success path renders the model's own reason verbatim, not a CLI word:
	// see TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason for the
	// no-translation-table proof.
	if fields.stop != "stop" {
		t.Errorf("done stop = %q, want stop (the model's own reason)", fields.stop)
	}
}
