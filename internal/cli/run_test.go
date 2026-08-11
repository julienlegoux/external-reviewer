package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/cli"
	"github.com/julienlegoux/external-reviewer/internal/fauxtest"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
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
	code := cli.RunWithModelsForTest(ctx, []string{"review", "--prompt", "x", t.TempDir()}, strings.NewReader(""), &stdout, &stderr,
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
// interruptedError check classifies every shape a cancelled run's terminal
// error can take, including a *ai.ModelsError — the type Resolver.Resolve
// returns for a broken credential — wrapping context.Canceled rather than a
// bare wrapped error. Before this issue, a second check lived in
// internal/reviewer (Conversation.Next) as a belt-and-braces fallback for
// exactly this shape; deleting it was believed to change no observable
// behaviour, on the premise that kern-link v0.1.1 always preserves the
// cancellation cause through stream.Result(ctx)'s own error. That premise is
// only half true — see TestAsInterrupted_ReclassifiesFailingTurnWhenContextEnded
// below for the race that disproved it and asInterrupted, the narrow fix
// that keeps this still being the one place the question is decided (a
// second wrap, not a second decision point).
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

// TestAsInterrupted_ReclassifiesFailingTurnWhenContextEnded pins the fix for
// a real windows-latest `go test -race` failure on this issue's own PR (CI
// run 31459008705): the identical commit passed the same job moments
// earlier (run 31459006569), so this was never a bad assertion — it was a
// genuine race in kern-link's Stream.Result(ctx), which selects between its
// result channel and ctx.Done(). When a cancellation lands mid-stream,
// either can win: if ctx.Done() wins, Result returns ctx.Err() directly and
// Conversation.Next's error already wraps it — the path
// TestRunContext_CancelledContext_ExitsTwo and (usually)
// TestRun_CancelledMidStream_ExitsTwo exercise. But if the provider's own
// goroutine notices the cancellation first and finishes with a
// StopReasonAborted message before Result's select runs, Result returns
// that message with a nil error, and Conversation.Next's returned error —
// "the reviewer's turn stopped with reason %q: %s" — wraps nothing that
// errors.Is(_, context.Canceled) can find. That is exactly the failure CI
// hit: stop=failed instead of stop=interrupted.
//
// Rather than trying to force that exact goroutine interleaving from a
// black-box test — inherently non-deterministic, and the reason the bug
// shipped in the first place — this constructs both of asInterrupted's
// inputs directly, so the fix is pinned regardless of which race outcome
// the runtime happens to hit on any given run or platform.
func TestAsInterrupted_ReclassifiesFailingTurnWhenContextEnded(t *testing.T) {
	turnErr := fmt.Errorf("the reviewer's turn stopped with reason %q: %s", ai.StopReasonAborted, "Request was aborted")

	t.Run("context still live: turnErr passes through, not classified as interrupted", func(t *testing.T) {
		got := cli.AsInterruptedForTest(context.Background(), turnErr)
		if cli.InterruptedErrorForTest(got) {
			t.Errorf("AsInterruptedForTest(live ctx, %v) = %v, want it not classified as interrupted", turnErr, got)
		}
	})

	t.Run("context already ended: turnErr is reclassified as interrupted", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		got := cli.AsInterruptedForTest(ctx, turnErr)
		if !cli.InterruptedErrorForTest(got) {
			t.Errorf("AsInterruptedForTest(cancelled ctx, %v) = %v, want it classified as interrupted", turnErr, got)
		}
	})

	t.Run("nil turnErr: a complete message is never touched, cancelled context or not", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if got := cli.AsInterruptedForTest(ctx, nil); got != nil {
			t.Errorf("AsInterruptedForTest(cancelled ctx, nil) = %v, want nil", got)
		}
	})
}
