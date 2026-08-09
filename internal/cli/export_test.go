package cli

import (
	"context"
	"io"
)

// SetReviewerForTest swaps the review execution seam — the one issues 04
// and 05 replace with real model resolution and the round trip — for the
// duration of a test, returning a func that restores the previous seam. It
// lets this issue's black-box tests drive the no-reviewer and
// reached-and-failed terminations that prove the done-line and exit-code
// machinery handles them, without waiting on that later work.
func SetReviewerForTest(fn func(ctx context.Context, repoPath, task string) error) func() {
	prev := reviewer
	reviewer = fn
	return func() { reviewer = prev }
}

// RunContextForTest calls Run's inner seam directly with ctx, bypassing
// Run's own signal.NotifyContext wiring. It exists because delivering a
// real SIGINT is not portable to the Windows CI runner: a pre-cancelled
// context exercises the same interruption path without one.
func RunContextForTest(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	code, _ := run(ctx, argv, stdin, stdout, stderr)
	return code
}
