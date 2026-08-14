---
type: Decision
title: "Repository confinement & path safety"
description: "How access is bounded: one kernel-enforced root, plus a required allow-list of subtrees passed per invocation."
tags: [decision, specs]
timestamp: 2026-08-09T01:30:00Z
phase: specs
decision: 10
slug: repository-confinement
status: decided
verdict: "os.OpenRoot anchors one root; --allow is required and repeatable; git_read pathspec-restricted and show-path validated; deny-list demoted to a floor"
decided_via: discussion
depends_on: [read-only-tool-contract]
---

# Question

Scope states the requirement in a form that rules out the easy implementation: every tool
call is confined to the repository root passed on the command line, traversal outside it
refused, and *"never writes, never commits, never touches GitHub state" is a property of
the code that exists, not of a filter on a command string*
([decision](/scope/07-read-only-tool-set.md)). Confinement has to be enforced the same
way.

This is the classic path-traversal problem with a Windows twist the constraint makes
non-negotiable: `..` segments, absolute paths, symlinks pointing out of the tree, and on
Windows also junctions, `\\?\` prefixes, 8.3 short names, drive-relative paths like `C:x`,
and reserved device names (`NUL`, `COM1`). A hand-rolled `strings.HasPrefix` after
`filepath.Clean` misses most of that list, and misses it silently.

# Options

- **`os.OpenRoot` + `Root.FS()` for every filesystem operation.** The kernel-level answer
  — the OS holds a handle to the directory and refuses any name that escapes it. Standard
  library, and its documented behaviour covers Windows reserved device names and
  out-of-root symlinks explicitly.
- **`filepath.Abs` + `EvalSymlinks` + prefix comparison.** Portable-looking and wrong in
  the corners; every corner is a Windows corner, on the one platform scope names as the
  development machine.
- **Run in a sandbox / container.** Not available: the binary is `go install`ed onto a
  Windows workstation and invoked by a skill.

# Recommendation

**`os.OpenRoot(repoPath)` once at startup; every read goes through that `*os.Root`, and
no tool ever touches `os` or `filepath` for I/O directly.**

The API carries the whole requirement: methods "can only access files and directories
beneath a root directory", symlinks are followed but "may not reference a location
outside the root" and "must not be absolute", and on Windows "file names may not
reference Windows reserved device names such as NUL and COM1". That is the list a
hand-rolled check would have to reproduce, maintained by the standard library instead.

- `Root.ReadFile` backs `read_file`.
- `Root.FS()` yields an `fs.FS`, so `list` is `fs.Glob` and `search` is `fs.WalkDir` over
  the same confined filesystem — one confinement mechanism for all three tools rather
  than three.
  - **Amended 2026-08-12 ([drift](/DRIFT.md)):** the mechanism is unchanged but the
    *handle* is not an `fs.FS`. `Root.FS()` is never exported, because an `fs.FS` over
    the root reads on the root's authority alone — neither the allow-list nor the floor
    below is in its way — and `fs.WalkDir(fsys, ".")` cannot even start when the grant is
    a subtree rather than `.`. `Scope.ReadDir` and `Scope.Stat` back `list` and `search`
    instead, each resolving through all three rules.
- **Amended 2026-08-12 ([drift](/DRIFT.md)):** delegation to `*os.Root` is the rule for
  every escape the *kernel* can see, and nothing above re-implements that list. It is not
  the rule for the wire-path *vocabulary*: `Scope.Resolve` refuses a name that is not
  repo-relative and `/`-separated before the root is asked, because the allow-list has
  nothing to compare against otherwise, because `C:\Windows\win.ini` is a legal filename
  on Linux and the two matrix OSes must agree, and because `git show <rev>:<path>` must be
  decidable for a path with no file behind it.
- Only the read half of the API is ever called. `Create`, `OpenFile` with write flags,
  `Mkdir`, `Remove`, `Chmod` are absent from this codebase, which is what makes
  "never writes" a property of the code that exists.

Two rules layered on top:

- **`.git/` is not readable through `read_file`, `list` or `search`.** Repository history
  is reachable only through `git_read`, which is allowlisted
  ([decision](/specs/09-read-only-tool-contract.md)). Raw `.git` reads offer nothing a
  `git log` does not, and `.git/config` carries credential helper configuration and
  remote URLs that can contain tokens.
- **A deny-list on sensitive filenames** — `.env*`, `*.pem`, `*.key`, `id_rsa*`,
  `*.p12`, `*.pfx`, `.npmrc`, `.netrc`, `credentials*` — refused by name with an
  explanatory error, and excluded from `list` and `search` results
  ([decision](/specs/14-security-and-data-exposure.md)).

`os.OpenRoot` needs Go 1.24+; this project is on 1.26
([decision](/specs/01-language-module-toolchain.md)).

# Verdict

`os.OpenRoot` accepted as the enforcement mechanism, with the *policy* above replaced: a
deny-list of sensitive filenames is the wrong default shape. **Access is an allow-list,
passed as parameters.**

**`--allow <relpath>`, repeatable, required.** `review` without at least one `--allow` is
a usage error (exit `2`), not a run over the whole tree. Each value is repo-relative,
resolved through the root, and may be `.` to mean the entire repository — but that has to
be *said*. Nothing outside the union of allowed subtrees is readable by any tool.

The objection raised against requiring it — that pre-selecting what the reviewer sees
rebuilds the flaw the concept rejects, where *"the party under review decides what the
reviewer is allowed to see, so its blind spots are copied into the selection"*
([CONCEPT](/CONCEPT.md)) — was considered and does not hold. Granting access to
directories is not the same act as handing over chosen files: within an allowed subtree
the reviewer still globs, greps and reads whatever it decides is relevant, in whatever
order, and nobody has curated the evidence. What the caller controls is the *reach* of the
run, not its *path* through it.

**One root, not several.** Cross-repository review — an epic whose work spans this
repository and the lx skills repository — is two runs in v1. A multi-root form
(`--root name=path`) would put a root prefix on every path in every tool in both
directions, taxing the whole contract for a case that arrives once.

**`git_read` is confined too, which closes a real hole.** As previously written, the
allow-list (and the old deny-list) were both bypassable in one call:

```
git show HEAD:.env
```

`git show <rev>:<path>` reads any file in the repository's *history*, through the one tool
that never touches `*os.Root`. File-level confinement is worthless while that works. So:

- Any `<rev>:<path>` argument has its path validated against the allow-list and the floor
  below before the command is built; a refused path never reaches `git`. **Amended
  2026-08-12 ([drift](/DRIFT.md))** from "`show`'s" — `git diff HEAD:.env HEAD:docs/notes.md`
  prints both blobs' contents, so the object form reads files in `diff` as much as in
  `show`, and it is validated in every subcommand's arguments, in all three of git's
  spellings (`<rev>:<path>`, `:<path>`, `:<stage>:<path>`). Pathspecs are not an
  alternative here: `git show HEAD:.env -- docs` still prints `.env`, and
  `git diff <blob> <blob> -- docs` is a usage error.
- `log`, `diff` and `status` get the allowed subtrees appended as pathspecs
  (`-- <paths>`), so history is scoped the same way the working tree is. **Amended
  2026-08-12 ([drift](/DRIFT.md))** from "`status` is unaffected" — an unscoped
  `git status --porcelain` names every changed path in the repository, including the ones
  the allow-list refuses and the ones the floor exists to hide. It is the one subcommand
  that leaks path names without reading a file, so the standard this bullet sets was not
  met while it was exempt. **Amended 2026-08-13** — the appended pathspecs are the allowed
  subtrees *by default*; `git_read`'s `paths[]` parameter
  ([decision 09](/specs/09-read-only-tool-contract.md)) replaces them with entries the
  model names, each resolved through this same allow-list and floor before the command is
  built. It narrows and never widens, and it exists because owning the `--` separator, as
  this bullet requires, is also what made a path-scoped diff unreachable.

**The deny-list survives, demoted.** `.env*`, `*.pem`, `*.key`, `id_rsa*`, `*.p12`,
`*.pfx`, `.npmrc`, `.netrc`, `credentials*` and `.git/` remain refused *within* allowed
subtrees. It is no longer the boundary — the allow-list is — but it stays as a cheap,
non-configurable floor for the common case of `--allow .`, and `.git/` stays excluded
because history has a proper tool. A refusal message names which rule refused, so "outside
the allow-list" and "a denied filename" are distinguishable to the model and to a human
reading stderr.

**Unchanged:** one `*os.Root` opened at startup, every read through it, `Root.FS()`
backing `list` and `search`, and none of the write half of the API present in the codebase
— which is what makes "never writes" a property of the code that exists rather than a
promise. Paths reported by `git ls-files` are re-resolved through the root before use
([decision 11](/specs/11-tool-implementation-strategy.md)).
