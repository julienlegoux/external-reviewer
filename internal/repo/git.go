package repo

import (
	"bytes"
	"context"
	"fmt"
	"io"
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

// runGit runs one git subcommand against repoPath and returns its stdout.
//
// This is the codebase's only process execution, and every constraint on it is
// here rather than at its call sites: git is named as a bare program resolved
// from PATH, its arguments are an argv slice that no shell ever parses, and
// its environment carries no GIT_* variable from this process. `-C repoPath`
// rather than cmd.Dir because it is git's own way of saying which repository
// it is being asked about, and it survives a repoPath the process's working
// directory could not.
//
// The subprocess reads the filesystem on its own authority, outside the
// *os.Root — which is exactly why nothing it reports is trusted: every path it
// names is re-resolved through the Scope before it is used (see enumerate.go),
// and its stdout is the only thing that leaves this function.
func runGit(ctx context.Context, repoPath string, args ...string) ([]byte, error) {
	// The single exec in this binary. exec.Command* is forbidden everywhere
	// else because a subprocess reads on its own authority, outside the root
	// handle (CONVENTIONS § Code style & formatting; ripgrep was rejected on
	// this exact point). git is the one program the confinement can survive,
	// because what it returns is names, and names are re-resolved.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...) //nolint:forbidigo // the one allowed exec, argv-only, no shell — see the comment above
	cmd.Env = gitEnvironment()

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// git's own stderr is discarded rather than forwarded: it is the OS's and
	// git's sentences, capitalised and platform-spelled, and this run states
	// its own diagnostics (CONVENTIONS § Error handling).
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), nil
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
