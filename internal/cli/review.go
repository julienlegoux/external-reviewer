package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// reviewRequest is the parsed, validated form of a `review` invocation: a
// repository to look at and the task to perform on it. Confinement
// (--allow) and everything past parsing (model resolution, the loop itself)
// arrive in later issues — this struct is the seam they build on.
type reviewRequest struct {
	RepoPath string
	Task     string
}

// runReviewCommand parses the `review` subcommand's flags and positional
// repository path out of args, resolves the task prompt from --prompt or
// stdin, and validates both before anything talks to a model. Every usage
// error exits 2 with a reason on stderr and nothing on stdout.
func runReviewCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(stderr)
	prompt := fs.String("prompt", "", "the task prompt for the reviewer")

	if err := fs.Parse(args); err != nil {
		// fs already wrote the reason and usage to stderr (SetOutput above).
		return 2
	}

	positional := fs.Args()
	if len(positional) == 0 {
		_, _ = fmt.Fprintln(stderr, "error: review requires a repository path")
		return 2
	}
	if len(positional) > 1 {
		_, _ = fmt.Fprintf(stderr, "error: unexpected extra arguments: %s\n", strings.Join(positional[1:], " "))
		return 2
	}
	repoPath := positional[0]

	info, err := os.Stat(repoPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: repository path %q: %v\n", repoPath, err)
		return 2
	}
	if !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "error: repository path %q is not a directory\n", repoPath)
		return 2
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
			return 2
		}
		task = string(data)
	}
	if task == "" {
		_, _ = fmt.Fprintln(stderr, "error: empty task prompt: pass --prompt or provide one on stdin")
		return 2
	}

	return runReview(reviewRequest{RepoPath: repoPath, Task: task}, stdout, stderr)
}

// runReview is the still-stubbed run path: resolving a model, confining the
// repository and making the round trip all arrive in later issues of this
// epic. For now, a request that parsed and validated cleanly exits 0.
func runReview(_ reviewRequest, _, _ io.Writer) int {
	return 0
}
