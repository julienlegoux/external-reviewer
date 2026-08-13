package cli

import (
	"fmt"
	"io"
)

// usageText documents the subset of the CLI grammar this binary accepts
// today: SPECS fixes a larger surface (the models and tiers commands) that
// the remaining issues add. Only what is accepted is listed here, so the
// message never promises a flag that would itself be a usage error.
const usageText = `usage: external-reviewer <command> [flags] [args]

commands:
  review --allow <path> [--allow <path> ...]
         [--tier light|standard|heavy | --model <provider>/<id>]
         [--exclude-family <family>[,<family>...]]
         [--system <text> --prompt <text>] <repo-path>
      Review the repository at <repo-path>. The system prompt and the task
      come from a JSON request object on stdin:

          {"system": "...", "task": "..."}

      Both fields are required and neither may be empty: the caller owns the
      system prompt and this binary supplies none. An unknown field is an
      error rather than a silently missing prompt.

      --system and --prompt are the by-hand shorthand for the same two
      values. They must be given together, and giving them means stdin is
      not read at all.

      --allow grants one repo-relative subtree the reviewer may read, and is
      repeatable. At least one is required: nothing outside the granted
      subtrees is readable. Use --allow . to grant the whole repository.

      --tier picks the reviewer by weight, resolved through the per-tier
      environment override and the machine-local assignment file. It defaults
      to standard.

      --model names one provider and model id and bypasses tiers entirely.
      It cannot be combined with --tier.

      --exclude-family lists the model families that may not review, and
      defaults to anthropic. A family the classifier cannot determine is
      never allowed to review, whatever this list says.

  version
      Print the module path, version and VCS revision this binary was built
      from, so a report can be traced back to a build.

  help | --help | -h
      Print this message.
`

// printUsage writes the usage text to w, terminated by nothing further —
// callers decide the exit code around it.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}
