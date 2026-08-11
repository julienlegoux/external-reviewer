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
// one prefixed error: line on stderr (via diag.WriteError) and no usage
// text alongside it. help/--help/-h are not a failure path and exit 0 with
// usage text on stdout — review --help and review -h match them
// (runReviewCommand splits flag.ErrHelp from a genuine parse error).
//
// Run wires SIGINT to context cancellation with signal.NotifyContext, so an
// interrupted run terminates through the same path a failed one does: exit
// 2, a reason on stderr, an empty stdout, and a done line. Delivering a real
// SIGINT is not portable to the Windows CI runner, so this wiring is
// exercised only by constructing Run itself; the cancellation behavior it
// feeds into is exercised directly against the inner run, which takes the
// context explicitly (see export_test.go's RunContextForTest).
//
// Whether a context was already cancelled before argv was even parsed is
// not decided here: help/empty-argv/an-unknown-command do no I/O that a
// cancelled context could interrupt, and runReview is the one place that
// decides "was this interrupted?" for the one command that does.
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	code, _ := run(ctx, argv, stdin, stdout, stderr)
	return code
}

// run is Run's inner seam. A defer here, rather than one at every return
// below, is what makes "no path can terminate without a done line" true
// regardless of how many branches this function grows: every return —
// success, a usage error, a reviewer that was never reached, one that
// failed, or a cancelled context — passes through it exactly once.
func run(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	state := diag.NewState()
	defer func() { diag.WriteDone(stderr, state) }()

	if len(argv) == 0 {
		diag.WriteError(stderr, "no command given")
		state.StopReason = usageStopReason
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
		err := runReviewCommand(ctx, rest, stdin, stdout, stderr, state)
		return classify(err), err
	default:
		diag.WriteError(stderr, fmt.Sprintf("unknown command %q", cmd))
		state.StopReason = usageStopReason
		err := fmt.Errorf("unknown command %q", cmd)
		return classify(err), err
	}
}
