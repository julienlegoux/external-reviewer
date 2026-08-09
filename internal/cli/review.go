package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/julienlegoux/external-reviewer/internal/diag"
)

// reviewRequest is the parsed, validated form of a `review` invocation: a
// repository to look at and the task to perform on it. Confinement
// (--allow) and everything past parsing (model resolution, the loop itself)
// arrive in later issues — this struct is the seam they build on.
type reviewRequest struct {
	RepoPath string
	Task     string
}

// reviewer is the seam issues 04 (model resolution and auth preflight) and
// 05 (the round trip) replace. Until then every syntactically valid review
// "succeeds" trivially, which is enough to prove the done-line and
// exit-code machinery this issue builds without waiting on that work.
// export_test.go exposes SetReviewerForTest so this issue's own tests can
// drive the no-reviewer and reached-and-failed terminations directly.
var reviewer = func(_ context.Context, _, _ string) error {
	return nil
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

	return runReview(ctx, reviewRequest{RepoPath: repoPath, Task: task}, stderr, state)
}

// runReview calls the review seam and folds its outcome into state's stop
// reason. stdout is untouched here — the model's report only reaches it
// once issue 05 wires the round trip; this issue's stub never has anything
// to print.
func runReview(ctx context.Context, req reviewRequest, stderr io.Writer, state *diag.State) error {
	err := reviewer(ctx, req.RepoPath, req.Task)
	switch {
	case err == nil:
		state.StopReason = "ok"
	case errors.Is(err, ErrNoReviewer):
		state.StopReason = "no_reviewer"
	default:
		state.StopReason = "failed"
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
	}
	return err
}
