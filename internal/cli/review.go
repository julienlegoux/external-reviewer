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

	"github.com/julienlegoux/external-reviewer/internal/diag"
	"github.com/julienlegoux/external-reviewer/internal/reviewer"
)

// reviewRequest is the parsed, validated form of a `review` invocation: a
// repository to look at and the task to perform on it. Confinement
// (--allow) and everything past parsing (model resolution, the loop itself)
// arrive in later issues — this struct is the seam they build on.
type reviewRequest struct {
	RepoPath string
	Task     string
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

	resolution, err := reviewer.Resolver{
		Models:     registry,
		ProviderID: reviewer.DefaultProviderID,
		ModelID:    reviewer.DefaultModelID,
		Warn:       func(message string) { diag.WriteWarn(stderr, message) },
	}.Resolve(ctx)
	if err != nil {
		return err
	}
	diag.WriteModel(stderr, resolution.Model.Provider, resolution.Model.ID, resolution.AuthSource)

	conversation := reviewer.NewConversation(registry, resolution.Model, req.Task)
	turn, turnErr := conversation.Next(ctx)
	recordTurn(stderr, state, turn)
	if turnErr != nil {
		return turnErr
	}

	report := reviewer.FinalText(turn.Message)
	if report == "" {
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

// recordTurn folds a completed round trip into the run state the done line
// renders from and writes the turn's own line, plus a warn line for every
// non-fatal diagnostic kern-link attached to the message. It runs on the
// failure paths too: a turn that stopped with an error still consumed tokens,
// and its diagnostics are usually why it stopped. state.StopReason is set
// here to the turn's own reason on every call — runReview's switch
// unconditionally overwrites it with the CLI word on any termination the
// model's own turn does not explain (a failed turn, no_reviewer,
// interrupted), so the value set here is only ever what the done line
// actually renders on the success path.
func recordTurn(stderr io.Writer, state *diag.State, turn reviewer.Turn) {
	if turn.Message == nil {
		return
	}

	state.Turns++
	// Cached prompt tokens are prompt tokens: providers report them beside
	// Input rather than inside it, so summing the three is what makes the in=
	// figure comparable between a cold run and a warm one.
	state.InputTokens += int64(turn.Usage.Input + turn.Usage.CacheRead + turn.Usage.CacheWrite)
	state.OutputTokens += int64(turn.Usage.Output)
	state.Cost += turn.Usage.Cost.Total
	state.StopReason = modelStopReason(turn.StopReason)

	for _, diagnostic := range turn.Message.Diagnostics {
		diag.WriteWarn(stderr, diagnosticMessage(diagnostic))
	}
	diag.WriteTurn(stderr, state.Turns, turn.Tools, state.InputTokens, state.OutputTokens, state.Cost, turn.Elapsed)
}

// stopReasonUnspecified is the done line's named fallback for a successful
// turn whose assistant message carried no StopReason at all, so stop= can
// never render with nothing after it (SPECS § Interfaces).
const stopReasonUnspecified = "unspecified"

// usageStopReason is the CLI word every malformed invocation's done line
// carries — a named constant rather than the literal repeated at every usage
// call site in this file and run.go.
const usageStopReason = "usage"

// modelStopReason renders a turn's own StopReason for the done line's
// success path, spelled exactly as kern-link spells it — no translation
// table between the model's vocabulary and the CLI's.
func modelStopReason(reason ai.StopReason) string {
	if reason == "" {
		return stopReasonUnspecified
	}
	return string(reason)
}

// diagnosticMessage renders one AssistantMessageDiagnostic as a warn line's
// text. kern-link redacts these upstream, and nothing here re-formats or
// re-derives anything from the underlying error, which is what keeps a
// credential from being reintroduced by a diagnostic (SPECS § Security).
func diagnosticMessage(diagnostic ai.AssistantMessageDiagnostic) string {
	kind := diagnostic.Type
	if kind == "" {
		kind = "diagnostic"
	}
	if diagnostic.Error == nil || diagnostic.Error.Message == "" {
		return kind
	}
	return kind + ": " + diagnostic.Error.Message
}

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
func runReviewCommand(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, state *diag.State, registry ai.Models) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	prompt := fs.String("prompt", "", "the task prompt for the reviewer")

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
		data, err := io.ReadAll(stdin)
		if err != nil {
			diag.WriteError(stderr, fmt.Sprintf("reading task prompt from stdin: %v", err))
			state.StopReason = usageStopReason
			return fmt.Errorf("reading task prompt from stdin: %w", err)
		}
		task = string(data)
	}
	if task == "" {
		diag.WriteError(stderr, "empty task prompt: pass --prompt or provide one on stdin")
		state.StopReason = usageStopReason
		return errors.New("empty task prompt")
	}

	return runReview(ctx, registry, reviewRequest{RepoPath: repoPath, Task: task}, stdout, stderr, state)
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
