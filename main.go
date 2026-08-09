// Command external-reviewer runs a bounded, read-only agent loop over an
// explicitly allowed slice of a repository, on a model outside the Anthropic
// family, and returns markdown on stdout.
package main

import (
	"os"

	"github.com/julienlegoux/external-reviewer/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
