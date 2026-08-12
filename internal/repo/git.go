package repo

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// gitConfigEnvironment names the two variables that decide whether a
// machine's own git configuration reaches a subcommand. Pointed at the
// platform's null device they resolve to an empty configuration, so an alias,
// a pager, an `include`, or a global `core.excludesFile` cannot change what
// `ls-files` reports (specs 11 — Tool implementation strategy).
var gitConfigEnvironment = []string{
	"GIT_CONFIG_GLOBAL=" + os.DevNull,
	"GIT_CONFIG_SYSTEM=" + os.DevNull,
}

// Output is what one git invocation produced: its standard output, and its
// standard error for the one caller that has to explain a failure to the model
// rather than to a human (internal/tools' git_read — a non-zero git exit is a
// tool error carrying git's own sentence, since the reviewer is the only one
// who can act on it).
type Output struct {
	Stdout []byte
	Stderr []byte
}

// Git runs one git subcommand against repoPath and returns both its streams.
// It is exported for git_read, which is the only other place in this binary
// allowed to run git, so that there is one exec in the codebase rather than
// two: the argv discipline, the `-C` form and the environment neutralisation
// below are properties of *every* git this process runs, not of enumeration.
//
// The error is returned unwrapped, because its two shapes are the caller's to
// tell apart: exec.ErrNotFound (git is not installed) and *exec.ExitError (git
// ran and refused).
func Git(ctx context.Context, repoPath string, args ...string) (Output, error) {
	return runGitStreams(ctx, repoPath, args...)
}

// runGit runs one git subcommand against repoPath and returns its stdout,
// discarding git's own stderr — which is what enumeration wants: git's
// sentences are capitalised, full-stopped and platform-spelled, and this run
// states its own diagnostics (CONVENTIONS § Error handling).
func runGit(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	output, err := runGitStreams(ctx, repoPath, args...)
	if err != nil {
		return nil, fmt.Errorf("running git %s: %w", strings.Join(args, " "), err)
	}
	return output.Stdout, nil
}

// runGitStreams is the codebase's only process execution, and every constraint
// on it is here rather than at its call sites: git is named as a bare program
// resolved from PATH, its arguments are an argv slice that no shell ever
// parses, and its environment carries no GIT_* variable from this process.
// `-C repoPath` rather than cmd.Dir because it is git's own way of saying which
// repository it is being asked about, and it survives a repoPath the process's
// working directory could not.
//
// The subprocess reads the filesystem on its own authority, outside the
// *os.Root — which is exactly why nothing it reports is trusted: every path it
// names is re-resolved through the Scope before it is used (see enumerate.go),
// and every path a caller *sends* is resolved through the Scope before the
// command is built (see internal/tools/git_read.go).
func runGitStreams(ctx context.Context, repoPath string, args ...string) (Output, error) {
	// The single exec in this binary. exec.Command* is forbidden everywhere
	// else because a subprocess reads on its own authority, outside the root
	// handle (CONVENTIONS § Code style & formatting; ripgrep was rejected on
	// this exact point). git is the one program the confinement can survive,
	// because what it returns is names, and names are re-resolved.
	//
	// gosec's G204 fires here on principle — the arguments are not constants —
	// and it is answered rather than suppressed blindly: the program is the
	// literal "git", the arguments are an argv slice no shell ever sees, and
	// on the platform where argument splitting is a CommandLineToArgvW problem
	// that is the whole defence (specs 11). No caller passes model-supplied
	// text through here unexamined: enumeration's arguments are literals, and
	// git_read allowlists its subcommand and resolves every path it was given
	// through the Scope before the command is built.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...) //nolint:forbidigo,gosec // the one allowed exec: argv-only, no shell, see above
	cmd.Env = gitEnvironment()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	// git's own stderr is captured rather than forwarded: it is the OS's and
	// git's sentences, capitalised and platform-spelled, and this run states
	// its own diagnostics (CONVENTIONS § Error handling). The one caller that
	// surfaces it hands it to the model, which is the only reader that can act
	// on "fatal: invalid object name".
	cmd.Stderr = &stderr

	err := cmd.Run()
	return Output{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}, err
}

// gitEnvironment is the process environment with every GIT_* variable dropped
// and the two configuration variables set explicitly.
//
// Dropping the whole prefix rather than the two names is the point: GIT_DIR,
// GIT_WORK_TREE, GIT_INDEX_FILE and GIT_CEILING_DIRECTORIES each redirect a
// subcommand at a different repository than the one this run was pointed at,
// and GIT_CONFIG_COUNT injects configuration without naming a file. The
// remaining environment is left alone, because git needs PATH — and, on
// Windows, SystemRoot — to run at all.
//
// The comparison folds case: Windows environment variable names are
// case-insensitive, so `git_config_global` set in the parent would otherwise
// survive a prefix test that only knows about `GIT_`.
func gitEnvironment() []string {
	inherited := os.Environ()
	environment := make([]string, 0, len(inherited)+len(gitConfigEnvironment))
	for _, entry := range inherited {
		if strings.HasPrefix(strings.ToUpper(entry), "GIT_") {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, gitConfigEnvironment...)
}

// splitNUL splits the output of a `-z` git command into its records. `-z` is
// what makes this safe: git quotes a filename containing a non-ASCII byte, a
// quote or a newline under its default core.quotepath, and a NUL-separated
// stream is emitted raw instead — so a filename arrives spelled exactly as the
// filesystem holds it.
func splitNUL(output []byte) []string {
	records := make([]string, 0, bytes.Count(output, []byte{0}))
	for _, record := range bytes.Split(output, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		records = append(records, string(record))
	}
	return records
}
