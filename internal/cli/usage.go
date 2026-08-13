package cli

import (
	"fmt"
	"io"
)

// usageText documents the subset of the CLI grammar this binary accepts
// today: SPECS fixes a larger surface (the version command, the JSON request
// object and --system) that the remaining issues add. Only what is accepted
// is listed here, so the message never promises a flag that would itself be
// a usage error.
const usageText = `usage: external-reviewer <command> [flags] [args]

commands:
  review --allow <path> [--allow <path> ...]
         [--tier light|standard|heavy | --model <provider>/<id>]
         [--exclude-family <family>[,<family>...]]
         [--prompt <text>] <repo-path>
      Review the repository at <repo-path>. The task prompt comes from
      --prompt, or from stdin when --prompt is not given.

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

  tiers [--exclude-family <family>[,<family>...]]
      Report which tiers (light, standard, heavy) resolve to a reachable
      model on this machine right now: the config path that was looked at,
      each tier's assignment and where it came from, and — when a tier does
      not resolve — which rule refused it (unassigned, not in the catalog,
      excluded by family, provider unconfigured or a broken credential). The
      report is on stdout; warnings and diagnostics are on stderr.

      Exits 0 whenever the query was answered, including when no tier
      resolves. Exits 2 only when it could not be answered at all: a
      malformed invocation, malformed config, or a credential store that
      cannot be located.

  models [--provider <id>] [--refresh] [--all]
      List the models this machine can review with: the intersection of the
      catalog, what this machine's credentials reach, and what the family
      rule allows, with price (per million tokens) and context window per
      row. The report is on stdout; warnings and diagnostics are on stderr.

      --provider restricts the listing to one provider id; a name no
      provider matches is a usage error.

      --refresh asks dynamic providers to fetch their current model list
      first — opt-in, because it costs network calls.

      --all drops the credential and family filters and shows the whole
      catalog, marking each row the default view would otherwise have
      dropped and why.

      Exits 0 whenever the query was answered, including when nothing is
      left to show. Exits 2 only when it could not be answered at all: a
      malformed invocation, or a credential store that cannot be located.

  help | --help | -h
      Print this message.
`

// printUsage writes the usage text to w, terminated by nothing further —
// callers decide the exit code around it.
func printUsage(w io.Writer) {
	_, _ = fmt.Fprint(w, usageText)
}
