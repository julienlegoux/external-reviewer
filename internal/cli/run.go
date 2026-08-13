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
	"time"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// shippedBounds is the run ceiling every real invocation of Run carries, sized
// from Epic 2's MEASUREMENTS and re-read knowing issue 01 of Epic 3 had
// already closed the git_read gap that inflated the twelve-turn run — see
// MEASUREMENTS § Observations for Epic 3.
//
// MaxTurns sits in the SCOPE-fixed [25, 40] range: the three completed hand
// runs took 4, 6 and 12 turns, the 12 inflated by a path-scoped diff being
// unreachable through git_read, a gap MEASUREMENTS says would plausibly have
// cost 6-8 turns instead. Nothing observed justifies a single-digit cap.
//
// MaxElapsed sits in the SCOPE-fixed [15m, 25m] range: wall clock was
// 1m57s-3m59s across the three runs, but the final report-writing turn alone
// took 1m35s-2m5s in every one of them, so a deadline under roughly five
// minutes would kill a run mid-report and discard everything it had.
//
// MaxCost stays unset (0): the credential this binary authenticates with is a
// subscription, nothing is billed per token, and a cost ceiling is the least
// informative of the three on that credential. The field stays in Bounds
// because a tier that names an API-key provider makes it real again, and the
// check costs one comparison either way (reviewer.Bounds.check).
//
// No flag exposes any of the three on the command line (see the PR that
// landed this seam's values for the reasoning): they are a safety net sized
// well above every measured run, not a per-invocation tuning knob the
// review-epics/review-issues template has ever asked for, and the seam
// already lets a flag be added later — a normal PR, not a restructuring —
// the moment something actually needs one.
var shippedBounds = reviewer.Bounds{
	MaxTurns:   30,
	MaxElapsed: 20 * time.Minute,
}

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
	return runProcess(argv, stdin, stdout, stderr, nil, shippedBounds)
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
func runProcess(argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models, bounds reviewer.Bounds) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	defer handleSIGPIPE()()

	code, _ := run(ctx, argv, stdin, stdout, stderr, registry, bounds)
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
func run(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer, registry ai.Models, bounds reviewer.Bounds) (int, error) {
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
		err := runReviewCommand(ctx, rest, stdin, stdout, stderr, state, registry, bounds)
		return classify(err), err
	case "tiers":
		err := runTiersCommand(ctx, rest, stdout, stderr, state, registry)
		return classify(err), err
	case "models":
		err := runModelsCommand(ctx, rest, stdout, stderr, state, registry)
		return classify(err), err
	default:
		diag.WriteError(stderr, fmt.Sprintf("unknown command %q", cmd))
		state.StopReason = usageStopReason
		err := fmt.Errorf("unknown command %q", cmd)
		return classify(err), err
	}
}
