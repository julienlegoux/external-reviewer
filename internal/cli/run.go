// Package cli implements the external-reviewer command line, in-process and
// testable through Run.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// Run is the entire behavior of the external-reviewer binary. It takes argv
// (without the program name) and the process's standard streams, and
// returns the process exit code. main is a thin
// os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) over this
// function so every behavior is reachable from a test.
//
// No arguments, an unrecognised subcommand, an unrecognised flag, or a
// malformed `review` invocation are all usage errors: exit 2, stdout empty,
// the reason on stderr. help/--help/-h are not a failure path and exit 0
// with usage text on stdout.
//
// Run wires SIGINT to context cancellation with signal.NotifyContext, so an
// interrupted run terminates through the same path a failed one does: exit
// 2, a reason on stderr, an empty stdout, and a done line. Delivering a real
// SIGINT is not portable to the Windows CI runner, so this wiring is
// exercised only by constructing Run itself; the cancellation behavior it
// feeds into is exercised directly against the inner run, which takes the
// context explicitly (see export_test.go's RunWithModelsForTest).
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, _ := run(ctx, argv, stdin, stdout, stderr, nil)
	return code
}

// run is Run's inner seam. A defer here, rather than one at every return
// below, is what makes "no path can terminate without a done line" true
// regardless of how many branches this function grows: every return —
// success, a usage error, a reviewer that was never reached, one that
// failed, or a cancelled context — passes through it exactly once.
//
// registry is the provider registry a review resolves against. nil — the
// zero value every real invocation passes — means "build the real one,
// lazily, on first use" (see ensureModels); a test passes its own offline
// registry as an explicit argument instead of mutating a package-level
// variable (see export_test.go's RunForTest and RunWithModelsForTest).
func run(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) (int, error) {
	state := diag.NewState()
	defer func() { diag.WriteDone(stderr, state) }()

	if err := ctx.Err(); err != nil {
		state.StopReason = "interrupted"
		_, _ = fmt.Fprintln(stderr, "error: run interrupted")
		wrapped := fmt.Errorf("run interrupted: %w", err)
		return classify(wrapped), wrapped
	}

	if len(argv) == 0 {
		printUsage(stderr)
		state.StopReason = "usage"
		err := errors.New("no command given")
		return classify(err), err
	}

	cmd, rest := argv[0], argv[1:]
	switch cmd {
	case "help", "--help", "-h":
		printUsage(stdout)
		state.StopReason = "help"
		return 0, nil
	case "review":
		err := runReviewCommand(ctx, rest, stdin, stdout, stderr, state, registry)
		return classify(err), err
	default:
		_, _ = fmt.Fprintf(stderr, "error: unknown command %q\n", cmd)
		printUsage(stderr)
		state.StopReason = "usage"
		err := fmt.Errorf("unknown command %q", cmd)
		return classify(err), err
	}
}
