package cli

import (
	"context"
	"io"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// RunForTest calls Run's full behavior — including its signal.NotifyContext
// and SIGPIPE wiring, and the run ceiling every real invocation carries —
// against registry rather than the network, for every test that isn't
// specifically exercising cancellation or the bounds seam itself. It is the
// constructor a test uses in place of mutating a package-level registry
// variable: registry is passed in as an explicit argument, once, at the call
// site.
//
// It delegates to runProcess, which is Run's own body, rather than keeping a
// second copy of the wiring: a replica drifts the moment the real entry
// point gains anything, and this one had already missed handleSIGPIPE. Using
// shippedBounds rather than the zero value is the same reasoning applied to
// the bounds seam: every scripted conversation this suite drives is a
// handful of turns, well under the shipped ceiling, so this is free coverage
// that the shipped caps do not interfere with an ordinary review.
func RunForTest(argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	return runProcess(argv, stdin, stdout, stderr, registry, shippedBounds)
}

// ShippedBoundsForTest exposes the run ceiling Run wires on every real
// invocation, so a test can assert the values directly — MaxTurns,
// MaxElapsed and MaxCost — rather than only their effect on a scripted run.
// A later edit that quietly unsets one of the three then fails this test
// directly instead of silently shipping an unbounded review.
func ShippedBoundsForTest() reviewer.Bounds { return shippedBounds }

// RunWithBoundsForTest calls Run's inner seam with a run ceiling in place.
// It exists because this epic ships the bounds seam with its values unset —
// no invocation can set one, so a test is the only caller that ever puts a
// number there, and "an exceeded bound returns what the run has at exit 0"
// is otherwise unassertable from outside the package.
func RunWithBoundsForTest(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models, bounds reviewer.Bounds) int {
	code, _ := run(ctx, argv, stdin, stdout, stderr, registry, bounds)
	return code
}

// RunWithModelsForTest calls Run's inner seam directly with ctx and
// registry, bypassing Run's own signal.NotifyContext wiring. It exists
// because delivering a real SIGINT is not portable to the Windows CI
// runner: a pre-cancelled (or later-cancelled) context exercises the same
// interruption path without one.
func RunWithModelsForTest(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	code, _ := run(ctx, argv, stdin, stdout, stderr, registry, reviewer.Bounds{})
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
