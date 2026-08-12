package cli

import (
	"fmt"
	"io"
)

// usageText documents the subset of the CLI grammar this binary accepts
// today: SPECS fixes a larger surface (--tier/--model, models/tiers/version)
// that the later epics add. Only what is accepted is listed here, so the
// message never promises a flag that would itself be a usage error.
const usageText = `usage: external-reviewer <command> [flags] [args]

commands:
  review --allow <path> [--allow <path> ...] [--prompt <text>] <repo-path>
      Review the repository at <repo-path>. The task prompt comes from
      --prompt, or from stdin when --prompt is not given.

      --allow grants one repo-relative subtree the reviewer may read, and is
      repeatable. At least one is required: nothing outside the granted
      subtrees is readable. Use --allow . to grant the whole repository.

  help | --help | -h
      Print this message.
`

// printUsage writes the usage text to w, terminated by nothing further —
// callers decide the exit code around it.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}
