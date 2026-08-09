---
type: Decision
title: "CLI surface & argv grammar"
description: "What the command line looks like, and whether a flag library is worth a dependency."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 02
slug: cli-surface-and-argv
status: decided
verdict: "stdlib flag, one FlagSet per subcommand; long-form flags only; repo path positional"
decided_via: triage
depends_on: [language-module-toolchain]
---

# Question

The argv grammar is a contract with a caller that is a *script*, not a person: the
`review-epics` / `review-issues` skills build the command line from a template
([scope](/scope/13-skills-integration-scope.md)). Renaming a flag after that template
ships means editing the skills repository, so the grammar is a one-way door even though
the parsing code behind it is not.

Scope has already fixed the pieces: a repository path and a task prompt (argument or
stdin) ([decision](/scope/09-report-output-contract.md)), `--tier light|standard|heavy`
([decision](/scope/04-reviewer-selection-vocabulary.md)), `--exclude-family anthropic`
by default ([decision](/scope/06-family-exclusion-rule.md)), and a tier-resolution
command ([Milestone 3](/SCOPE.md)). What is open is how they compose, and what parses
them.

# Options

- **stdlib `flag` with `flag.NewFlagSet` per subcommand.** Zero dependencies, matches
  `kern-link` (which parses argv by hand and carries no CLI library at all). No
  `--flag=value` after positionals, and no clustered short flags.
- **`spf13/cobra`.** Subcommands, completion, generated help — and a dependency tree far
  larger than the program, for a CLI whose entire surface is two commands.
- **`urfave/cli/v3`.** Lighter than cobra, still a dependency and still more machinery
  than two commands need.

# Recommendation

**stdlib `flag`, one `FlagSet` per subcommand**, with this grammar:

```
external-reviewer review [--tier light|standard|heavy] [--model provider/id]
                         [--exclude-family anthropic[,…]] [--prompt <text>] <repo-path>
external-reviewer tiers   [--exclude-family anthropic[,…]]
external-reviewer help | --help | -h | version
```

- `<repo-path>` is positional and required — it is the one argument that is always
  present and never optional, and a positional keeps the invocation readable in a
  transcript.
- The prompt comes from `--prompt` or, when that is absent, all of stdin. A review task
  is a paragraph, and a skill composing it in a Bash heredoc should not have to escape
  it into a flag value.
- `--model provider/id` bypasses tier resolution entirely. This is journey 4 — the
  author judging a model by hand before trusting it with a tier
  ([scope](/scope/10-core-user-journeys.md)) — and without it that journey requires
  editing the config file to try a model once.
- `--exclude-family` accepts a comma-separated list and defaults to `anthropic`, so the
  default matches the scope decision while staying a list rather than a boolean.

Every flag is long-form only: the caller is a script that never types twice, and short
flags are pure ambiguity surface. Unknown flags and unknown subcommands exit `2` with
usage on stderr, never `0`.

# Verdict

Accepted at triage as recommended: no CLI library, `review`/`tiers`/`help`/`version`, prompt from `--prompt` or stdin, `--model` for the by-hand journey, `--exclude-family` as a comma-separated list defaulting to `anthropic`.

**Refreshed after [decision 05](/specs/05-tier-assignment-schema.md).** One subcommand
added, none removed:

```
external-reviewer models [--provider <id>] [--refresh] [--all]
```

It lists the models reachable on this machine — the intersection of `kern-link`'s catalog
(refreshed for dynamic providers when `--refresh` is given), what credentials resolve,
and what the family rule allows — with price and context window per row. `--all` drops
the credential and family filters and shows the whole catalog. This is what makes journey
3 (first-time setup) and journey 4 (judging a model by hand) something other than trial
and error ([scope](/scope/10-core-user-journeys.md)).

The `--family` companion flag briefly considered alongside `--model` is **not** added:
decision 05 dropped the family override entirely.

**Refreshed after [decision 08](/specs/08-system-prompt-and-task-assembly.md).** The
prompt channel changes shape: stdin now carries a JSON request object
(`{"system": …, "task": …}`) rather than raw task text, with `--system <text>` and
`--prompt <text>` as the mutually-exclusive by-hand shorthand. `--system-file` was
considered and dropped. Grammar as it now stands:

```
external-reviewer review [--tier light|standard|heavy] [--model provider/id]
                         [--exclude-family anthropic[,…]]
                         [--system <text> --prompt <text>] <repo-path>
                         # …or the JSON request object on stdin
external-reviewer models  [--provider <id>] [--refresh] [--all]
external-reviewer tiers   [--exclude-family anthropic[,…]]
external-reviewer help | --help | -h | version
```

**Refreshed after [decision 10](/specs/10-repository-confinement.md).** `--allow` is added
and is **required** for `review` — one or more repo-relative subtrees the run may reach,
`.` for the whole repository. Omitting it is a usage error (exit `2`), not an implicit
run over everything. Final grammar:

```
external-reviewer review --allow <relpath> [--allow <relpath>…]
                         [--tier light|standard|heavy | --model provider/id]
                         [--exclude-family anthropic[,…]]
                         [--system <text> --prompt <text>] <repo-path>
                         # …or the JSON request object on stdin
external-reviewer models  [--provider <id>] [--refresh] [--all]
external-reviewer tiers   [--exclude-family anthropic[,…]]
external-reviewer help | --help | -h | version
```
