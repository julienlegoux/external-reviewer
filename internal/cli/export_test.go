package cli

import (
	"context"
	"io"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// SetReviewerForTest swaps the whole review execution seam — pre-flight
// resolution and the round trip that follows it — for the duration of a
// test, returning a func that restores the previous seam. It lets a test
// drive the no-reviewer and reached-and-failed terminations directly, so the
// done-line and exit-code machinery can be exercised without a registry.
//
// The run state stays out of the substituted signature deliberately: a stub
// that could write turn counts and token totals would let a test assert on
// numbers no model produced.
func SetReviewerForTest(fn func(ctx context.Context, repoPath, task string, stdout, stderr io.Writer) error) func() {
	prev := performReview
	performReview = func(ctx context.Context, req reviewRequest, stdout, stderr io.Writer, _ *diag.State) error {
		return fn(ctx, req.RepoPath, req.Task, stdout, stderr)
	}
	return func() { performReview = prev }
}

// SetModelsForTest points model resolution at an offline registry for the
// duration of a test, returning a func that restores the previous one.
// Without it a test would resolve against every provider kern-link ships,
// over the machine's real credential store — the network and the bill this
// project's tests never touch.
func SetModelsForTest(registry ai.Models) func() {
	prev := models
	models = registry
	return func() { models = prev }
}

// RunContextForTest calls Run's inner seam directly with ctx, bypassing
// Run's own signal.NotifyContext wiring. It exists because delivering a
// real SIGINT is not portable to the Windows CI runner: a pre-cancelled
// context exercises the same interruption path without one.
func RunContextForTest(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	code, _ := run(ctx, argv, stdin, stdout, stderr)
	return code
}
