---
type: Decision
title: "Tool implementation strategy"
description: "Whether search and git_read shell out to external binaries or are implemented in-process."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 11
slug: tool-implementation-strategy
status: decided
verdict: "git ls-files enumerates, in-process RE2 matches, os.Root confines; git_read execs git with argv; no ripgrep, no shell"
decided_via: discussion
depends_on: [read-only-tool-contract, repository-confinement]
---

# Question

Scope calls the tools "native, in-process" and rules out a general shell
([decision](/scope/07-read-only-tool-set.md)) — but two of the four describe external
programs. `search` is specified as "ripgrep-style", and `git_read` is git by definition.
Whether either actually spawns a process is undecided, and it determines what the binary
requires to be installed on the machine and how tightly the confinement of
[decision 10](/specs/10-repository-confinement.md) holds.

# Options

**`search`**

- **In-process `regexp` over the confined `fs.FS`.** No external requirement, confinement
  is the same object the other tools use, and it is slower than ripgrep on a large tree.
- **Exec `rg`.** Fast and gitignore-aware for free, and it makes the binary silently
  degrade on any machine without ripgrep — a second undeclared dependency, which is the
  exact cost this project exists to remove from `opencode`.

**`git_read`**

- **Exec `git` with argv, allowlisted subcommand, no shell.** Real git output, byte for
  byte what the human sees, and it requires `git` on `PATH`.
- **`go-git`.** Pure Go, no external requirement, and its formatting differs from git's
  in ways the model will read as facts about the repository.

# Recommendation

**In-process `regexp` for `search`; exec `git` with argv for `git_read`. No shell in
either case.**

The two go opposite ways because the requirements are not symmetric. Ripgrep is optional
on a developer machine and its absence would be invisible until a review returned
nothing; git is definitionally present in the repository under review — a repo without
git is not a thing this tool is ever pointed at.

`search`: `regexp.Compile` (RE2 — linear time, no catastrophic backtracking on a
model-supplied pattern) over files enumerated by `fs.WalkDir(root.FS(), ".")`. Skipped by
default: `.git/`, the sensitive-file deny-list, anything the deny-list of
[decision 10](/specs/10-repository-confinement.md) covers, files over ~2 MB, and files
that fail a binary sniff (a NUL byte in the first 8 KB). Matches are capped by
`max_results` with the truncation marker the tool contract requires. This is slower than
ripgrep and irrelevantly so: the wall clock of a review is model latency, not grep.

`git_read`: `exec.CommandContext(ctx, "git", append([]string{"-C", repoPath, sub},
args...)...)` where `sub` is validated against the four-item allowlist *before* the
command is built. Never `exec.Command("sh", "-c", …)`, never `cmd.Args` assembled from a
joined string — the array-shaped `args` parameter
([decision](/specs/09-read-only-tool-contract.md)) exists so that no string is ever
parsed into arguments, on a platform where argument quoting is a `CommandLineToArgvW`
problem rather than a POSIX one. `cmd.Env` is reduced to the minimum, with
`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` neutralised so a machine-level `core.pager`,
alias or `include` cannot change what a subcommand does. Output is capped and truncation
announced, like every other tool.

`git` missing from `PATH` is a `git_read` tool error the model is told about, not a run
failure — the other three tools still work, and a review that read files without reading
history is a degraded review rather than no review.

# Verdict

**`search`: `git ls-files` enumerates, `regexp` matches, `os.Root` confines. No
ripgrep.**

Ripgrep was reconsidered on request and rejected on a stronger argument than the original
one. Speed was never the real issue — the trees this is pointed at are documentation
folders and single-service repositories, where an in-process walk finishes in
milliseconds and the wall clock of a review is model latency regardless. The decisive
objection is **confinement**: `rg` is a subprocess reading the filesystem on its own
authority, so it sits outside the `*os.Root` handle that
[decision 10](/specs/10-repository-confinement.md) makes the single enforcement point.
Adopting it would downgrade "the kernel refuses names that escape the root" to "we set
the working directory and validated the paths it printed" — for a speed gain against a
bottleneck that is not the bottleneck. A second undeclared `PATH` dependency is also the
precise cost this project exists to remove from `opencode`.

What ripgrep *would* have brought for free, and the original recommendation quietly threw
away, is **gitignore awareness**. An in-process walk over a repository containing
`node_modules/` or `vendor/` is slow, and worse, fills the reviewer's context with
vendored code it will read and reason about. That is fixed without `rg`:

- **Enumeration** is `git ls-files -z` for tracked files, plus
  `git ls-files -z --others --exclude-standard` for untracked-but-not-ignored ones. This
  is gitignore semantics exactly, produced by the implementation that defines them,
  through a dependency `git_read` already requires. `-z` avoids the quoting git applies to
  unusual filenames.
- **Matching** stays in-process: `regexp` (RE2 — linear time, no catastrophic
  backtracking on a model-supplied pattern) over content read through the confined root.
- **Confinement** stays the one `*os.Root`. Every path git reports is re-resolved through
  it, so a file git lists but the root refuses is skipped rather than trusted.
- Outside a git repository, enumeration falls back to `fs.WalkDir` over the root with
  `.git/` skipped — degraded, not broken.

`list` uses the same enumeration, so what the reviewer can glob and what it can grep never
disagree.

**`git_read`: exec `git` with argv, allowlist first.** Unchanged from the recommendation.
`exec.CommandContext(ctx, "git", "-C", repoPath, sub, args...)` with `sub` validated
against the four-item allowlist *before* the command is built; never a shell, never a
joined string parsed back into arguments — which matters most on Windows, where argument
splitting is a `CommandLineToArgvW` problem rather than a POSIX one.
`GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` are neutralised so a machine-level pager, alias or
`include` cannot change what a subcommand does.

**`git` is now a hard dependency, and that is a deliberate change.** The original
recommendation treated a missing `git` as a degraded run — three tools still working. With
enumeration built on `git ls-files`, its absence also costs correct ignore handling. It
remains non-fatal (the `fs.WalkDir` fallback keeps the run alive), but it is no longer
merely a `git_read` concern, and the `tiers`/`models` commands report `git`'s presence so
the degradation is visible rather than inferred from a thin review.
