package cli_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// knownDiagPrefixes are the leading words internal/diag's Write* functions
// ever put on a line — the vocabulary assertOnlyKnownPrefixedLines checks
// every stderr line against.
var knownDiagPrefixes = map[string]bool{
	"turn":   true,
	"tool":   true,
	"model":  true,
	"warn":   true,
	"error:": true,
	"done":   true,
}

// assertOnlyKnownPrefixedLines fails the test if any non-blank line in
// stderr does not open with one of diag's known prefixes — the guard
// against a stray unprefixed line (flag's own usage dump, printUsage
// leaking onto stderr) reaching the transcript ahead of the done line.
func assertOnlyKnownPrefixedLines(t *testing.T, stderr string) {
	t.Helper()
	for _, line := range strings.Split(stderr, "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || !knownDiagPrefixes[fields[0]] {
			t.Errorf("stderr contains an unprefixed line: %q (full stderr: %q)", line, stderr)
		}
	}
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
	fields := fauxtest.ParseDoneLine(t, stderr.String())
	// "stop" is kern-link's own StopReasonStop — faux.TextMessage's default
	// when no AssistantMessageOptions.StopReason is scripted. The done line's
	// success path renders the model's own reason verbatim, not a CLI word:
	// see TestRun_SuccessfulTurn_DoneLineRendersModelsOwnStopReason for the
	// no-translation-table proof.
	if fields.Stop != "stop" {
		t.Errorf("done stop = %q, want stop (the model's own reason)", fields.Stop)
	}
}
