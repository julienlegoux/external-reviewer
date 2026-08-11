package cli

import (
	"context"
	"io"
	"os"
	"os/signal"

	"github.com/julienlegoux/kern-link/ai"
)

// RunForTest calls Run's full behavior — including its signal.NotifyContext
// wiring — against registry rather than the network, for every test that
// isn't specifically exercising cancellation. It is the constructor a test
// uses in place of mutating a package-level registry variable: registry is
// passed in as an explicit argument, once, at the call site.
func RunForTest(argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, _ := run(ctx, argv, stdin, stdout, stderr, registry)
	return code
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
