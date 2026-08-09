// Package cli implements the external-reviewer command line, in-process and
// testable through Run.
package cli

import (
	"fmt"
	"io"
)

// Run is the entire behavior of the external-reviewer binary. It takes argv
// (without the program name) and the process's standard streams, and returns
// the process exit code. main is a thin os.Exit(cli.Run(...)) over this
// function so every behavior is reachable from a test.
//
// No arguments, an unrecognised subcommand, an unrecognised flag, or a
// malformed `review` invocation are all usage errors: exit 2, stdout empty,
// the reason on stderr. help/--help/-h are not a failure path and exit 0
// with usage text on stdout.
func Run(argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(argv) == 0 {
		printUsage(stderr)
		return 2
	}

	cmd, rest := argv[0], argv[1:]
	switch cmd {
	case "help", "--help", "-h":
		printUsage(stdout)
		return 0
	case "review":
		return runReviewCommand(rest, stdin, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "error: unknown command %q\n", cmd)
		printUsage(stderr)
		return 2
	}
}
