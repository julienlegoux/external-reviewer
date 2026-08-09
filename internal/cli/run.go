// Package cli implements the external-reviewer command line, in-process and
// testable through Run.
package cli

import (
	"fmt"
	"io"
)

const usage = `usage: external-reviewer review --allow <relpath> [--allow <relpath>...] <repo-path>
`

// Run is the entire behavior of the external-reviewer binary. It takes argv
// (without the program name) and the process's standard streams, and returns
// the process exit code. main is a thin os.Exit(cli.Run(...)) over this
// function so every behavior is reachable from a test.
//
// Argument parsing beyond "no arguments is a usage error" is not implemented
// yet — the review grammar arrives in a later issue — so every invocation
// currently reports usage and exits 2.
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	_, _ = fmt.Fprint(stderr, usage)
	return 2
}
