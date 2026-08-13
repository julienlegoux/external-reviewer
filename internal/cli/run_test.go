package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// TestRun_ShippedBounds_AreSizedFromMeasurements pins the run ceiling every
// real invocation carries as data, not only through its effect on a scripted
// run — a later edit that quietly unsets one of the three fields fails this
// test directly. The ranges are the ones issue 09 of Epic 3 derives from
// Epic 2's MEASUREMENTS, re-read after issue 01 closed the git_read gap that
// inflated the twelve-turn run.
func TestRun_ShippedBounds_AreSizedFromMeasurements(t *testing.T) {
	b := cli.ShippedBoundsForTest()
	if b.MaxTurns < 25 || b.MaxTurns > 40 {
		t.Errorf("MaxTurns = %d, want in [25, 40]", b.MaxTurns)
	}
	if b.MaxElapsed < 15*time.Minute || b.MaxElapsed > 25*time.Minute {
		t.Errorf("MaxElapsed = %s, want in [15m, 25m]", b.MaxElapsed)
	}
	if b.MaxCost != 0 {
		t.Errorf("MaxCost = %v, want 0 — the credential is a subscription, nothing is billed per token", b.MaxCost)
	}
}

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
	fields := fauxtest.ParseDoneLine(t, stderr.String())
	if fields.Stop != "usage" {
		t.Errorf("done stop = %q, want usage", fields.Stop)
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
			if fields := fauxtest.ParseDoneLine(t, stderr.String()); fields.Stop != "help" {
				t.Errorf("done stop = %q, want help", fields.Stop)
			}
		})
	}
}

// TestRunContext_CancelledContext_ExitsTwo drives the interruption path
// directly against the inner run seam (RunWithModelsForTest) with a
// pre-cancelled context, since delivering a real SIGINT is not portable to
// the Windows CI runner — Run's signal.NotifyContext wiring itself is
// exercised only by constructing Run in the other tests in this package.
//
// "Was this interrupted?" is decided once, in runReview's interruptedError
// check — not by a shortcut at the top of run() that used to fire before
// argv was even parsed. registry is a real offline faux registry (never the
// machine's credential store): auth resolves synchronously regardless of
// ctx, so the pre-cancelled context is first observed where kern-link's own
// stream.Result(ctx) returns ctx.Err() — the same path
// TestRun_CancelledMidStream_ExitsTwo exercises mid-flight, here hit before
// a single byte streams.
func TestRunContext_CancelledContext_ExitsTwo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout, stderr bytes.Buffer
	code := cli.RunWithModelsForTest(ctx, []string{"review", "--allow", ".", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
		registry(t, fauxtest.CredentialedAuth("OAuth"), nil, reviewer.DefaultModelID))

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
	fields := fauxtest.ParseDoneLine(t, stderr.String())
	if fields.Stop != "interrupted" {
		t.Errorf("done stop = %q, want interrupted", fields.Stop)
	}
	if fields.In != "0" || fields.Out != "0" {
		t.Errorf("done in/out = %q/%q, want 0/0 (a path that never reached a model)", fields.In, fields.Out)
	}
}

// TestRunReview_InterruptionDecidedOnce is the acceptance criterion's proof
// that "was this interrupted?" is decided in exactly one place: runReview's
// interruptedError check classifies every shape a cancelled run's terminal
// error can take, including a *ai.ModelsError — the type Resolver.Resolve
// returns for a broken credential — wrapping context.Canceled rather than a
// bare wrapped error.
//
// The decision point is here; producing the cause is internal/reviewer's job.
// A failing turn that ends under a cancelled context now leaves
// Conversation.Next already carrying context.Canceled (its
// TestNext_AbortedMessageUnderCancelledContextIsInterrupted pins that
// deterministically), which is what let internal/cli's own asInterrupted
// wrap — a second guard answering the same question one layer further out —
// be deleted rather than left overlapping it.
//
// The last row is the other half of the same boundary: a turn that ran out
// of this binary's own stream timeout is a dead connection, not an
// interruption, and must not borrow the word. It is asserted here rather
// than end to end because reaching the real timeout through Run would mean
// waiting out reviewer.DefaultStreamTimeout — ten minutes — while the bound
// itself is asserted in milliseconds against Conversation.StreamTimeout in
// internal/reviewer's TestNext_SilentProviderEndsAtTheStreamTimeout.
//
// This asserts interruptedError directly (via cli.InterruptedErrorForTest)
// rather than driving the classification through a full Run: reaching a
// *ai.ModelsError wrapping context.Canceled for real would require scripting
// kern-link's own auth-resolution internals rather than this package's
// seam, and runReview has exactly one call site for this question — the
// same one exercised end to end by TestRunContext_CancelledContext_ExitsTwo
// and TestRun_CancelledMidStream_ExitsTwo (roundtrip_test.go).
func TestRunReview_InterruptionDecidedOnce(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"a bare error wrapping context.Canceled", fmt.Errorf("streaming: %w", context.Canceled), true},
		{"a bare error wrapping context.DeadlineExceeded", fmt.Errorf("streaming: %w", context.DeadlineExceeded), true},
		{"a *ai.ModelsError wrapping context.Canceled", ai.NewModelsError(ai.ModelsErrorAuth, "resolving credentials", context.Canceled), true},
		{"a turn that stopped aborted under a cancelled context", fmt.Errorf("the reviewer's turn stopped with reason %q: %s: %w", ai.StopReasonAborted, "Request was aborted", context.Canceled), true},
		{"a stream timeout", fmt.Errorf("streaming the reviewer's turn: %w", reviewer.ErrStreamTimeout), false},
		{"an unrelated failure", errors.New("turn failed: stop reason error"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cli.InterruptedErrorForTest(tc.err); got != tc.want {
				t.Errorf("InterruptedErrorForTest(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
