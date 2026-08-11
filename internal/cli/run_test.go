package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/cli"
)

// TestRun_EmptyArgv_UsageError asserts the empty-argv path's error: line
// text directly — not stderr.Len() != 0, which the leftover usage text this
// path used to print would have satisfied on its own even with no reason
// line at all.
func TestRun_EmptyArgv_UsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := cli.Run(nil, strings.NewReader(""), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "error:  no command given\n") {
		t.Errorf("stderr = %q, want an error:  no command given line", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.stop)
	}
}

// TestRun_Help_DoneLineStopIsHelp pins the CLI word for the one termination
// the model never even gets a chance to run for: help/--help/-h all exit 0
// with a done line whose stop= reads "help", not the model's vocabulary
// (there was no turn to have one).
func TestRun_Help_DoneLineStopIsHelp(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{"help", []string{"help"}},
		{"--help", []string{"--help"}},
		{"-h", []string{"-h"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := cli.Run(tc.argv, strings.NewReader(""), &stdout, &stderr)

			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr.String())
			}
			if fields := parseDoneLine(t, stderr.String()); fields.stop != "help" {
				t.Errorf("done stop = %q, want help", fields.stop)
			}
		})
	}
}

// TestRunContext_CancelledContext_ExitsTwo drives the interruption path
// directly against the inner run seam (RunContextForTest) with a
// pre-cancelled context, since delivering a real SIGINT is not portable to
// the Windows CI runner — Run's signal.NotifyContext wiring itself is
// exercised only by constructing Run in the other tests in this package.
//
// The reviewer seam is stubbed to observe ctx itself rather than left to
// resolve for real: "was this interrupted?" is decided once, in runReview,
// from whatever error performReview returns — not by a shortcut at the top
// of run() that used to fire before argv was even parsed. Stubbing also
// keeps this test hermetic: an unstubbed run would fall through to the
// machine's real credential store, present or absent, for a result this
// test has no business depending on.
func TestRunContext_CancelledContext_ExitsTwo(t *testing.T) {
	restore := cli.SetReviewerForTest(func(ctx context.Context, _, _ string, _, _ io.Writer) error {
		return fmt.Errorf("streaming the reviewer's turn: %w", ctx.Err())
	})
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer
	code := cli.RunContextForTest(ctx, []string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)

	if code != 2 {
		t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "interrupt") {
		t.Errorf("stderr = %q, want an interruption reason", stderr.String())
	}
	assertOnlyKnownPrefixedLines(t, stderr.String())
	fields := parseDoneLine(t, stderr.String())
	if fields.stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.stop)
	}
	if fields.in != "0" || fields.out != "0" {
		t.Errorf("done in/out = %q/%q, want 0/0 (a path that never reached a model)", fields.in, fields.out)
	}
}

// TestRunReview_InterruptionDecidedOnce is the acceptance criterion's proof
// that "was this interrupted?" is decided in exactly one place: runReview's
// errors.Is(err, context.Canceled) check classifies every shape a cancelled
// run's terminal error can take, including a *ai.ModelsError — the type
// Resolver.Resolve returns for a broken credential — wrapping
// context.Canceled rather than a bare wrapped error. Before this issue, a
// second check lived in internal/reviewer (Conversation.Next) as a
// belt-and-braces fallback for exactly this shape; deleting it changes no
// observable behaviour because kern-link v0.1.1 already preserves the
// cancellation cause through every path this binary reaches (the gap the
// belt-and-braces check existed for is real only against a newer kern-link
// API this project does not build against — see DRIFT.md #04).
func TestRunReview_InterruptionDecidedOnce(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"a bare error wrapping context.Canceled", fmt.Errorf("streaming: %w", context.Canceled)},
		{"a bare error wrapping context.DeadlineExceeded", fmt.Errorf("streaming: %w", context.DeadlineExceeded)},
		{"a *ai.ModelsError wrapping context.Canceled", ai.NewModelsError(ai.ModelsErrorAuth, "resolving credentials", context.Canceled)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restore := cli.SetReviewerForTest(func(context.Context, string, string, io.Writer, io.Writer) error {
				return tc.err
			})
			defer restore()

			var stdout, stderr bytes.Buffer
			code := cli.Run([]string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr)

			if code != 2 {
				t.Errorf("exit code = %d, want 2 (stderr: %q)", code, stderr.String())
			}
			assertOnlyKnownPrefixedLines(t, stderr.String())
			fields := parseDoneLine(t, stderr.String())
			if fields.stop != "interrupted" {
				t.Errorf("done stop = %q, want interrupted", fields.stop)
			}
		})
	}
}
