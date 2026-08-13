package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/julienlegoux/kern-link/ai"

	"github.com/julienlegoux/external-reviewer/internal/confine"
	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/repo"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
	"github.com/julienlegoux/external-reviewer/internal/tools"
)

// reviewRequest is the parsed, validated form of a `review` invocation: a
// repository to look at, the task to perform on it, and the confinement every
// read goes through. Everything past parsing (the loop itself, the tools)
// arrives in later issues — this struct is the seam they build on.
//
// Scope is the *os.Root and the allow-list travelling as one value: a tool
// takes the Scope, and there is no other way for it to reach a file.
type reviewRequest struct {
	RepoPath string
	Task     string
	Scope    *confine.Scope
	// Bounds is the run ceiling the loop consults at the top of every turn.
	// It arrives here unset on every real invocation — no flag sets one yet
	// (see run.go) — and travels on the request rather than as a package
	// value so Epic 3 fills it in at the one place the request is built.
	Bounds reviewer.Bounds
}

// allowList is the repeatable --allow flag's value: repo-relative wire paths,
// accumulated in the order they were given. flag.Value rather than a
// comma-separated string, because a path may legitimately contain a comma and
// a subtree grant is not a place to invent an escaping rule.
type allowList []string

// String renders the flag's current value for flag's own diagnostics. The
// zero value is the empty list, which is the state that makes a review a
// usage error.
func (a *allowList) String() string {
	if a == nil {
		return ""
	}
	return strings.Join(*a, " ")
}

// Set appends one --allow occurrence. Validation — that the value names an
// existing directory inside the root — belongs to confine.OpenScope, which is
// the only thing holding the root it has to be resolved against.
func (a *allowList) Set(value string) error {
	*a = append(*a, value)
	return nil
}

// resolveAndReview runs the pre-flight that can end a run before a single
// request is sent, then takes the one turn Epic 1 ships and writes the
// reviewer's own markdown to stdout. Nothing about the credential exists at
// this layer: resolution returns a source label, never an AuthResult, so
// there is no credential value here to leak.
//
// registry reaches here as an explicit dependency threaded from run's inner
// seam (see run.go and ensureModels below) rather than through a
// package-level mutable — the only seam a test replaces is the argument it
// passes in.
//
// stdout is written exactly once, at the very end, from the completed final
// message — never from the text deltas as they arrive. That is what makes
// "stdout is empty on every failure" true for the failures that stream a
// paragraph of prose before falling over: every one of them fails before the
// single write, so nothing has been handed to the caller yet.
//
// The one failure that survives past that point is the write itself, and
// bytes already on stdout cannot be unwritten. So the invariant the shipped
// code actually holds is the three-part one below, not "stdout is empty on
// every failure" flat:
//
//   - Nothing reaches stdout until the reviewer's final message is complete,
//     so every failure before the write leaves stdout empty.
//   - The process never terminates by signal: a broken reader on stdout comes
//     back as an ordinary error (see handleSIGPIPE in run.go), so the run
//     still reaches the done line and still returns a real exit code.
//   - A write that fails after N bytes leaves those N bytes on stdout and
//     says so on stderr, naming the report as incomplete, so a caller reading
//     the transcript can tell a truncated report from a whole one. That is
//     the strongest thing available: the alternative would be an output file
//     the binary is forbidden to own (SPECS § Interfaces).
func resolveAndReview(ctx context.Context, registry ai.Models, req reviewRequest, stdout, stderr io.Writer, state *diag.State) error {
	registry, err := ensureModels(registry)
	if err != nil {
		return err
	}

	warn := func(message string) { diag.WriteWarn(stderr, message) }
	resolution, err := reviewer.Resolver{
		Models:     registry,
		ProviderID: reviewer.DefaultProviderID,
		ModelID:    reviewer.DefaultModelID,
		Warn:       warn,
	}.Resolve(ctx)
	if err != nil {
		return err
	}
	diag.WriteModel(stderr, resolution.Model.Provider, resolution.Model.ID, resolution.AuthSource)

	loop := &reviewer.Loop{
		Conversation: reviewer.NewConversation(registry, resolution.Model, req.Task),
		Tools: tools.NewRunRegistry(tools.Deps{
			Scope:    req.Scope,
			RepoPath: req.RepoPath,
			Files:    repo.NewEnumerator(req.Scope, req.RepoPath, warn),
			Warn:     warn,
		}),
		// Unset on every real invocation, and that is the decision rather
		// than an omission: SCOPE defers the turn, cost and wall-clock
		// ceilings until issue 07 has measured a real review, so Epic 3 sets
		// numbers on the way in instead of restructuring the loop (specs 07,
		// reviewer.Bounds).
		Bounds: req.Bounds,
		State:  state,
		Stderr: stderr,
	}
	report, err := loop.Run(ctx)
	if err != nil {
		return err
	}
	// An empty report is only ever a failure when nothing else explains it.
	// A bounds stop can legitimately have nothing to show — the report is
	// written in one final turn, so a run stopped ahead of it has no prose
	// at all — and SPECS says a bounded run is exit 0 with the report it
	// had, including none; it must not be reclassified as failed here.
	if report == "" && state.StopReason != reviewer.BoundsStopReason {
		return errors.New("the reviewer finished but its final message carried no text")
	}
	return writeReport(stdout, report)
}

// writeReport performs the run's one and only write to stdout and classifies
// what came back. Every outcome but a whole report is an ordinary exit-2
// failure: the error returned here is what runReview renders as the single
// error: line, so nothing about a failing stdout routes around the seam.
//
// The two failures are told apart because they are not the same fact. With
// nothing written, stdout is still empty and the caller has no report at all.
// With N bytes written, the caller holds a fragment that looks exactly like a
// report — that is the one the diagnostic has to name, with the byte counts,
// or a truncated review reads as a complete one.
//
// A short write reported with a nil error is io.Writer's contract being
// broken rather than a disk filling; os.File cannot produce one, since
// internal/poll loops until the buffer is drained. It is folded into the same
// classification anyway, because the alternative is returning nil for a
// report that was never fully delivered.
func writeReport(stdout io.Writer, report string) error {
	n, err := io.WriteString(stdout, report)
	if err == nil && n < len(report) {
		err = io.ErrShortWrite
	}
	switch {
	case err == nil:
		return nil
	case n == 0:
		return fmt.Errorf("writing the report to stdout: %w", err)
	default:
		return fmt.Errorf("writing the report to stdout: the report on stdout is incomplete, %d of %d bytes were written: %w",
			n, len(report), err)
	}
}

// usageStopReason is the CLI word every malformed invocation's done line
// carries — a named constant rather than the literal repeated at every usage
// call site in this file and run.go.
//
// The done line's other two sources moved to internal/reviewer with the
// accumulation this file used to own: the model's own stop reason and its
// "unspecified" fallback are set by the loop, one call frame below, because
// that is where the totals the bound check reads have to live (issue 03).
const usageStopReason = "usage"

// ensureModels returns registry unchanged when a caller supplied one — every
// test does, as an explicit argument — and builds the real one, over
// kern-link's own credential store, otherwise. A credential store that
// cannot even be located is a broken machine rather than an absent
// reviewer, so it propagates as an ordinary error — exit 2 with a reason —
// rather than the silent exit 1 an unconfigured provider gets.
func ensureModels(registry ai.Models) (ai.Models, error) {
	if registry != nil {
		return registry, nil
	}
	built, err := reviewer.DefaultModels()
	if err != nil {
		return nil, fmt.Errorf("building the model registry: %w", err)
	}
	return built, nil
}

// maxPromptBytes bounds the stdin read: the task prompt is an instruction,
// not a document — the reviewer reads repository content itself, through
// the tools Epic 2 wires, not through this read. 1 MiB is orders of
// magnitude above any legitimate instruction while still stopping an
// accidental multi-gigabyte pipe (`cat 8GB.bin | external-reviewer review
// /repo`) from buffering fully into memory — or reaching a third-party model
// in full — before validation ever runs (CONVENTIONS § Dependencies
// L155-156, the supply-chain and exposure surface).
const maxPromptBytes = 1 << 20 // 1 MiB

// errPromptTooLarge is the named sentinel a stdin body over maxPromptBytes
// returns. Classification inspects it with errors.Is — never by matching
// the rendered message text (CONVENTIONS § Error handling).
var errPromptTooLarge = errors.New("task prompt exceeds the maximum size")

// readPromptFromStdin reads the task prompt from stdin under ctx, bounded to
// maxPromptBytes+1 bytes — the +1 is what distinguishes "exactly at the
// bound" (returned whole) from "over it" (errPromptTooLarge) without ever
// reading past the bound itself.
//
// io.ReadAll(stdin) alone is not cancellable: the underlying Read blocks in
// the runtime with nothing available to interrupt it from another
// goroutine, short of closing stdin — which this function does not own
// (os.Stdin is the process's; a caller's io.Reader may not even support
// Close). So the read runs on its own goroutine, feeding a result over a
// channel buffered to 1; this function selects between that channel and
// ctx.Done(). When ctx wins, it returns ctx.Err() immediately, and the
// goroutine is deliberately abandoned: it is still blocked inside the real
// Read call (or will eventually unblock, e.g. when stdin reaches EOF at
// process exit), and its result lands in the buffered channel with no
// receiver rather than blocking forever on the send — a bounded leak for
// the remainder of the process's life, not an unbounded one, since run's
// caller is terminating through the done-line seam either way once this
// returns.
func readPromptFromStdin(ctx context.Context, stdin io.Reader) (string, error) {
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(io.LimitReader(stdin, maxPromptBytes+1))
		done <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-done:
		if r.err != nil {
			return "", r.err
		}
		if len(r.data) > maxPromptBytes {
			return "", fmt.Errorf("stdin carried more than %d bytes: %w", maxPromptBytes, errPromptTooLarge)
		}
		return string(r.data), nil
	}
}

// runReviewCommand parses the `review` subcommand's flags and positional
// repository path out of args, resolves the task prompt from --prompt or
// stdin, and validates both before anything talks to a model. It returns
// the error classify uses to pick the process exit code: nil is exit 0,
// ErrNoReviewer is exit 1, anything else — including every usage error
// below — is exit 2. Every usage error writes exactly one diag.WriteError
// line to stderr and nothing else; a reached-and-failed error gets one too,
// but a not-reached one (ErrNoReviewer) does not, since SPECS calls that
// path the silent-fallback case the caller needs no line about.
//
// fs.SetOutput(io.Discard) suppresses flag's own error-and-usage write —
// its default behaviour would otherwise put an unprefixed "Usage of
// review:" block on stderr ahead of the diag.WriteError line below, which
// is exactly the second, unprefixed line the acceptance criteria forbid.
func runReviewCommand(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, state *diag.State, registry ai.Models, bounds reviewer.Bounds) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prompt := fs.String("prompt", "", "the task prompt for the reviewer")
	var allow allowList
	fs.Var(&allow, "allow", "a repo-relative subtree the reviewer may read; repeatable, required")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// review --help / review -h are not a failure path: usage on
			// stdout, exit 0, matching top-level help (run.go).
			printUsage(stdout)
			state.StopReason = "help"
			return nil
		}
		diag.WriteError(stderr, err.Error())
		state.StopReason = usageStopReason
		return fmt.Errorf("parsing review flags: %w", err)
	}

	positional := fs.Args()
	if len(positional) == 0 {
		diag.WriteError(stderr, "review requires a repository path")
		state.StopReason = usageStopReason
		return errors.New("review requires a repository path")
	}
	if len(positional) > 1 {
		diag.WriteError(stderr, fmt.Sprintf("unexpected extra arguments: %s", strings.Join(positional[1:], " ")))
		state.StopReason = usageStopReason
		return fmt.Errorf("unexpected extra arguments: %s", strings.Join(positional[1:], " "))
	}
	// repoPath is an OS path from here on: filepath.Clean makes it canonical
	// for whichever platform this binary runs on, and os.Stat below takes it
	// native. wireRepoPath is the boundary conversion CONVENTIONS § Paths and
	// platforms names — every diagnostic below reaches stderr /-separated via
	// filepath.ToSlash, never the OS-native form, so a Windows run never puts
	// a backslash (or flag's %q double-escaping of one) on stderr.
	repoPath := filepath.Clean(positional[0])
	wireRepoPath := filepath.ToSlash(repoPath)

	info, err := os.Stat(repoPath)
	if err != nil {
		// The OS's own sentence (capitalised, full-stopped, and spelled
		// differently per platform) never reaches stderr; the message states
		// only what was attempted. err is still wrapped for errors.Is/As —
		// classification stays on the wrapped error, never on message text.
		diag.WriteError(stderr, fmt.Sprintf("repository path %q could not be accessed", wireRepoPath))
		state.StopReason = usageStopReason
		return fmt.Errorf("repository path %q could not be accessed: %w", wireRepoPath, err)
	}
	if !info.IsDir() {
		diag.WriteError(stderr, fmt.Sprintf("repository path %q is not a directory", wireRepoPath))
		state.StopReason = usageStopReason
		return fmt.Errorf("repository path %q is not a directory", wireRepoPath)
	}

	// Confinement is opened before the task prompt is resolved, so a run that
	// granted nothing — or granted something that is not a subtree — ends
	// without blocking on stdin and without reaching a model. The stat above
	// established nothing this depends on: OpenScope opens its own root from
	// repoPath and every read resolves through that handle on its own merits
	// (see internal/confine).
	if len(allow) == 0 {
		diag.WriteError(stderr, "review requires at least one --allow <path>: use --allow . to grant the whole repository")
		state.StopReason = usageStopReason
		return errors.New("review requires at least one --allow <path>")
	}
	scope, err := confine.OpenScope(repoPath, allow)
	if err != nil {
		diag.WriteError(stderr, err.Error())
		state.StopReason = usageStopReason
		return err
	}
	// One root for the run, released on every termination path this function
	// can take from here on — the success path included, since runReview is
	// called inside it rather than after it.
	defer func() { _ = scope.Close() }()

	promptSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "prompt" {
			promptSet = true
		}
	})

	var task string
	if promptSet {
		task = *prompt
	} else {
		data, err := readPromptFromStdin(ctx, stdin)
		if err != nil {
			switch {
			case interruptedError(err):
				// ctx ended (SIGINT via run.go's signal.NotifyContext, or a
				// test's own cancellation) while the read was still blocked:
				// this is an interruption, not a usage mistake, and the done
				// line has to say which of the two happened — the same
				// judgment runReview's own switch makes for a cancelled
				// model call.
				diag.WriteError(stderr, "run interrupted")
				state.StopReason = "interrupted"
				return err
			case errors.Is(err, errPromptTooLarge):
				diag.WriteError(stderr, err.Error())
				state.StopReason = usageStopReason
				return err
			default:
				diag.WriteError(stderr, fmt.Sprintf("reading task prompt from stdin: %v", err))
				state.StopReason = usageStopReason
				return fmt.Errorf("reading task prompt from stdin: %w", err)
			}
		}
		task = data
	}
	// strings.TrimSpace decides emptiness only. task itself — what reaches
	// reviewRequest.Task and eventually the model — is never rewritten, so a
	// caller's own leading/trailing whitespace around real text still
	// reaches the model verbatim (asserted in
	// TestRun_Review_PromptWhitespaceReachesModelVerbatim).
	if strings.TrimSpace(task) == "" {
		diag.WriteError(stderr, "empty task prompt: pass --prompt or provide one on stdin")
		state.StopReason = usageStopReason
		return errors.New("empty task prompt")
	}

	return runReview(ctx, registry, reviewRequest{RepoPath: repoPath, Task: task, Scope: scope, Bounds: bounds}, stdout, stderr, state)
}

// runReview calls the review seam and folds its outcome into state's stop
// reason. Cancellation (interruptedError) is checked before the generic
// failure case: a run a human interrupted is not a run that failed, and the
// transcript has to say which of the two happened. Both are exit 2 all the
// same.
func runReview(ctx context.Context, registry ai.Models, req reviewRequest, stdout, stderr io.Writer, state *diag.State) error {
	err := resolveAndReview(ctx, registry, req, stdout, stderr, state)
	switch {
	case err == nil:
		// state.StopReason already carries the model's own reason: recordTurn
		// set it from the successful turn, verbatim as kern-link spells it —
		// not a CLI word (SPECS § Interfaces).
	case errors.Is(err, ErrNoReviewer):
		state.StopReason = "no_reviewer"
	case interruptedError(err):
		state.StopReason = "interrupted"
		diag.WriteError(stderr, "run interrupted")
	default:
		state.StopReason = "failed"
		diag.WriteError(stderr, err.Error())
	}
	return err
}

// interruptedError reports whether err represents a run a human (or a
// deadline) interrupted rather than one that failed on its own — the same
// question at any wrap depth, whatever shape the terminal error takes:
// context.Canceled/context.DeadlineExceeded directly, or wrapped inside a
// *ai.ModelsError from Resolve, whose Unwrap exposes exactly that when the
// pre-flight call itself was cancelled, or carried out of a failing turn by
// reviewer.Conversation.Next, which attaches the cancellation cause on the
// path where the provider's own aborted message reaches kern-link's
// stream.Result before the cancellation does. This is the one place "was this
// interrupted?" is decided.
//
// reviewer.ErrStreamTimeout is deliberately not in this set. A provider that
// accepts the stream and then goes quiet is a dead connection, not a run
// anyone stopped, so it takes the "failed" branch below — SPECS § Interfaces'
// "reached and then unusable for any other reason" — and gets an error: line
// naming the timeout rather than the word "interrupted", which would send a
// reader looking for a Ctrl-C that never happened.
func interruptedError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
