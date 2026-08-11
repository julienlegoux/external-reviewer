package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
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

// performReview is everything a syntactically valid review does once parsing
// is done: pre-flight resolution and the round trip that follows it.
// export_test.go exposes SetReviewerForTest so a test can replace the whole
// of it and drive a termination path directly.
var performReview = resolveAndReview

// models is the provider registry resolution runs against. It stays nil in a
// real process — reviewer.DefaultModels builds the real one, over kern-link's
// own credential store, on the first review — and is set by tests to an
// offline registry so no test can reach the network or spend money.
var models ai.Models

// resolveAndReview runs the pre-flight that can end a run before a single
// request is sent, then takes the one turn Epic 1 ships and writes the
// reviewer's own markdown to stdout. Nothing about the credential exists at
// this layer: resolution returns a source label, never an AuthResult, so
// there is no credential value here to leak.
//
// stdout is written exactly once, at the very end, from the completed final
// message — never from the text deltas as they arrive. That is what makes
// "stdout is empty on every failure" true for the failures that stream a
// paragraph of prose before falling over.
func resolveAndReview(ctx context.Context, req reviewRequest, stdout, stderr io.Writer, state *diag.State) error {
	registry, err := reviewModels()
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
	if _, err := io.WriteString(stdout, report); err != nil {
		return fmt.Errorf("writing the report to stdout: %w", err)
	}
	return nil
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

// reviewModels returns the injected registry when a test set one, and builds
// the real one otherwise. A credential store that cannot even be located is
// a broken machine rather than an absent reviewer, so it propagates as an
// ordinary error — exit 2 with a reason — rather than the silent exit 1 an
// unconfigured provider gets.
func reviewModels() (ai.Models, error) {
	if models != nil {
		return models, nil
	}
	registry, err := reviewer.DefaultModels()
	if err != nil {
		return nil, fmt.Errorf("building the model registry: %w", err)
	}
	return registry, nil
}

// runReviewCommand parses the `review` subcommand's flags and positional
// repository path out of args, resolves the task prompt from --prompt or
// stdin, and validates both before anything talks to a model. It returns
// the error classify uses to pick the process exit code: nil is exit 0,
// ErrNoReviewer is exit 1, anything else — including every usage error
// below — is exit 2. Every usage error also writes its reason to stderr
// here; a reached-and-failed error is written to stderr too, but a
// not-reached one (ErrNoReviewer) is not, since SPECS calls that path the
// silent-fallback case the caller needs no line about.
func runReviewCommand(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, state *diag.State) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(stderr)
	prompt := fs.String("prompt", "", "the task prompt for the reviewer")

	if err := fs.Parse(args); err != nil {
		// fs already wrote the reason and usage to stderr (SetOutput above).
		state.StopReason = "usage"
		return fmt.Errorf("parsing review flags: %w", err)
	}

	positional := fs.Args()
	if len(positional) == 0 {
		_, _ = fmt.Fprintln(stderr, "error: review requires a repository path")
		state.StopReason = "usage"
		return errors.New("review requires a repository path")
	}
	if len(positional) > 1 {
		_, _ = fmt.Fprintf(stderr, "error: unexpected extra arguments: %s\n", strings.Join(positional[1:], " "))
		state.StopReason = "usage"
		return fmt.Errorf("unexpected extra arguments: %s", strings.Join(positional[1:], " "))
	}
	repoPath := positional[0]

	info, err := os.Stat(repoPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: repository path %q: %v\n", repoPath, err)
		state.StopReason = "usage"
		return fmt.Errorf("checking repository path %q: %w", repoPath, err)
	}
	if !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "error: repository path %q is not a directory\n", repoPath)
		state.StopReason = "usage"
		return fmt.Errorf("repository path %q is not a directory", repoPath)
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
			_, _ = fmt.Fprintf(stderr, "error: reading task prompt from stdin: %v\n", err)
			state.StopReason = "usage"
			return fmt.Errorf("reading task prompt from stdin: %w", err)
		}
		task = string(data)
	}
	if task == "" {
		_, _ = fmt.Fprintln(stderr, "error: empty task prompt: pass --prompt or provide one on stdin")
		state.StopReason = "usage"
		return errors.New("empty task prompt")
	}

	return runReview(ctx, reviewRequest{RepoPath: repoPath, Task: task}, stdout, stderr, state)
}

// runReview calls the review seam and folds its outcome into state's stop
// reason. Cancellation is checked before the generic failure case: a run a
// human interrupted is not a run that failed, and the transcript has to say
// which of the two happened. Both are exit 2 all the same.
func runReview(ctx context.Context, req reviewRequest, stdout, stderr io.Writer, state *diag.State) error {
	err := performReview(ctx, req, stdout, stderr, state)
	switch {
	case err == nil:
		// state.StopReason already carries the model's own reason: recordTurn
		// set it from the successful turn, verbatim as kern-link spells it —
		// not a CLI word (SPECS § Interfaces).
	case errors.Is(err, ErrNoReviewer):
		state.StopReason = "no_reviewer"
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		state.StopReason = "interrupted"
		_, _ = fmt.Fprintln(stderr, "error: run interrupted")
	default:
		state.StopReason = "failed"
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
	}
	return err
}
