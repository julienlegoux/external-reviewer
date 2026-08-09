package cli

import (
	"fmt"
	"io"
)

// usageText documents the epic-1 subset of the CLI grammar: SPECS fixes a
// larger surface (--allow, --tier/--model, models/tiers/version) that the
// later epics add. Only what this binary accepts today is listed here, so
// the message never promises a flag that would itself be a usage error.
const usageText = `usage: external-reviewer <command> [flags] [args]

commands:
  review [--prompt <text>] <repo-path>
      Review the repository at <repo-path>. The task prompt comes from
      --prompt, or from stdin when --prompt is not given.

  help | --help | -h
      Print this message.
`

// printUsage writes the usage text to w, terminated by nothing further —
// callers decide the exit code around it.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}
