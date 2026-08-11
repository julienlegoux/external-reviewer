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
	"syscall"

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
// context explicitly (see export_test.go's RunWithModelsForTest).
//
// Whether a context was already cancelled before argv was even parsed is
// not decided here: help/empty-argv/an-unknown-command do no I/O that a
// cancelled context could interrupt, and runReview is the one place that
// decides "was this interrupted?" for the one command that does.
//
// Run also takes SIGPIPE off the runtime's default path (see
// handleSIGPIPE), because `external-reviewer review … | head -20` would
// otherwise kill the process outright and skip the done line entirely.
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runProcess(argv, stdin, stdout, stderr, nil)
}

// runProcess is Run's body with the registry left as an argument, so that
// export_test.go's RunForTest can drive the *same* process-level wiring
// against an offline registry instead of keeping a second copy of it.
//
// The duplication that used to sit in RunForTest is the reason this exists:
// a test-only replica of the entry point silently misses whatever the real
// entry point gains next. It had already missed handleSIGPIPE, which would
// have left the one test that exists to prove the process is not killed by
// SIGPIPE running against a process that never registered for it.
func runProcess(argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	defer handleSIGPIPE()()

	code, _ := run(ctx, argv, stdin, stdout, stderr, registry)
	return code
}

// handleSIGPIPE registers for SIGPIPE and returns the func that unregisters
// it. It is the difference between a run that ends and a run that is killed.
//
// os/signal's documented rule: with no Notify registered for SIGPIPE, a write
// to a broken pipe on file descriptor 1 or 2 raises SIGPIPE and the process
// dies by signal — a shell reports 141 — while the same write on any other
// descriptor merely returns EPIPE. Registering for it flattens that
// distinction: every such write returns EPIPE to Go, and the signal goes to
// the channel instead. That is exactly what the done-line seam needs, because
// a process killed by a signal runs no deferred function, so
// `external-reviewer review … | head -20` would exit 141 with the transcript
// cut off mid-run. With the registration in place the failed write comes back
// as an ordinary error, is classified in resolveAndReview, and terminates
// through run()'s single defer like every other failure.
//
// The channel is buffered and deliberately never read: nothing here wants to
// know that a pipe broke — the write's own error says so, with the context of
// what was being written. Dropped repeats are the point, not a leak.
//
// No //go:build divergence: syscall.SIGPIPE is defined on Windows too and
// signal.Notify accepts it there, where it is simply never delivered because
// Windows has no such signal. The one code path compiles and runs on both
// matrix OSes, and TestRun_ClosedStdoutPipe_NeverDiesBySignal exercises it on
// both, selecting its expectation at runtime.
func handleSIGPIPE() func() {
	broken := make(chan os.Signal, 1)
	signal.Notify(broken, syscall.SIGPIPE)
	return func() { signal.Stop(broken) }
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
		err := runReviewCommand(ctx, rest, stdin, stdout, stderr, state, registry)
		return classify(err), err
	default:
		diag.WriteError(stderr, fmt.Sprintf("unknown command %q", cmd))
		state.StopReason = usageStopReason
		err := fmt.Errorf("unknown command %q", cmd)
		return classify(err), err
	}
}
