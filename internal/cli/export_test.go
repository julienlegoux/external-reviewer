package cli

import (
	"context"
	"io"

	"github.com/julienlegoux/kern-link/ai"
)

// RunForTest calls Run's full behavior — including its signal.NotifyContext
// and SIGPIPE wiring — against registry rather than the network, for every
// test that isn't specifically exercising cancellation. It is the
// constructor a test uses in place of mutating a package-level registry
// variable: registry is passed in as an explicit argument, once, at the call
// site.
//
// It delegates to runProcess, which is Run's own body, rather than keeping a
// second copy of the wiring: a replica drifts the moment the real entry
// point gains anything, and this one had already missed handleSIGPIPE.
func RunForTest(argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	return runProcess(argv, stdin, stdout, stderr, registry)
}

// RunWithModelsForTest calls Run's inner seam directly with ctx and
// registry, bypassing Run's own signal.NotifyContext wiring. It exists
// because delivering a real SIGINT is not portable to the Windows CI
// runner: a pre-cancelled (or later-cancelled) context exercises the same
// interruption path without one.
func RunWithModelsForTest(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	code, _ := run(ctx, argv, stdin, stdout, stderr, registry)
	return code
}

// ClassifyForTest exposes classify's exit-code mapping directly, for the one
// assertion that needs the classification logic itself rather than a run
// through Run: a sentinel wrapped at any depth still resolves to the same
// exit code (issue 03 of Epic 1's explicit criterion).
func ClassifyForTest(err error) int { return classify(err) }

// InterruptedErrorForTest exposes interruptedError directly, for the
// classification-shape test that proves "was this interrupted?" is decided
// in exactly one place: every shape a cancelled run's terminal error can
// take — including a *ai.ModelsError from Resolve wrapping
// context.Canceled — must answer the same way, without going through a real
// (or stubbed) review execution to construct it.
func InterruptedErrorForTest(err error) bool { return interruptedError(err) }

// AsInterruptedForTest exposes asInterrupted directly, so the race it
// resolves — a failing turn's error winning the classification race against
// stream.Result observing ctx.Err() directly — can be pinned deterministically,
// by constructing both inputs by hand instead of racing a real stream against
// a real cancellation.
func AsInterruptedForTest(ctx context.Context, turnErr error) error {
	return asInterrupted(ctx, turnErr)
}
