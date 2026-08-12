---
type: Drift record
title: "Enumeration walks through the Scope's own ReadDir, not Root.FS()"
description: "SPECS says Root.FS() backs list and search; the fallback walk instead goes through Scope.ReadDir and Scope.Stat, because an exported fs.FS reads on the strength of the root alone — neither the allow-list nor the sensitive-file floor is in its way — and fs.WalkDir cannot even start at \".\" when a single subtree is granted."
tags: [epic-2, drift]
timestamp: 2026-08-12T08:00:00Z
epic: 2
issue: 02
---

# Enumeration walks through the Scope's own ReadDir, not Root.FS()

## Decided

[SPECS § Reading the repository](../../../planning/SPECS.md) names the mechanism by its
type: *"`Root.FS()` backs `list` and `search`, so one confinement mechanism serves all
three file tools."*
[specs 11 — Tool implementation strategy](../../../planning/specs/11-tool-implementation-strategy.md)
spells the walk out the same way — *"`fs.WalkDir(root.FS(), \".\")`"* for the in-process
matcher, and *"outside a git repository, enumeration falls back to `fs.WalkDir` over the
root with `.git/` skipped"*. Issue 02's Scope repeats it: *"`fs.WalkDir` over `Root.FS()`
with `.git/` skipped"*.

## Actual

`*os.Root` never leaves `internal/confine`, and neither does an `fs.FS` over it. The
`Scope` from issue 01 gained two read-half methods instead —
`Scope.ReadDir(wirePath) ([]fs.DirEntry, error)` and
`Scope.Stat(wirePath) (fs.FileInfo, error)` — each of which runs `Scope.Resolve` before
touching the root. `internal/repo`'s fallback walk is written against those: it starts at
`Scope.Allowed()` rather than at `"."`, and every child name it meets goes back through
`Resolve` before it is descended into or collected.

The confinement mechanism is unchanged — it is still the single `*os.Root` opened at
startup, and every read still goes through it. What differs is the *handle*: the walk
holds a `Scope`, which is that root plus the allow-list and the floor, rather than an
`fs.FS`, which is that root alone.

## Because

Three things, the first two of them verified against the code as it exists.

- **`fs.WalkDir(root.FS(), ".")` cannot start where the allow-list is not `.`.** The
  run's own gate refuses `"."` outright when the grant is `--allow docs`
  (`ErrOutsideAllowList`), which is correct: the repository root is not granted. Walking
  from `"."` anyway and filtering the results afterwards would list the names of every
  directory the invocation did not grant — including the names under them — which is the
  leak `git_read`'s pathspec confinement exists to prevent for `status`
  ([EPIC_2](../EPIC_2.md)). Starting at `Allowed()` is the only spelling that reads
  nothing outside the grant, and once the walk starts there, `fs.WalkDir` over the root's
  own FS is no longer the natural expression of it.
- **An `fs.FS` over the root has neither the allow-list nor the floor in it.** `os.Root`
  enforces one of the three rules — inside the root — and knows nothing of the other two.
  Any component holding the FS could `fs.WalkDir` into `secrets/` and `ReadFile` a
  `.env`, and the boundary this product sells would be back to a convention that every
  caller must remember. Issue 01 built `Scope` precisely so that *nothing outside
  `internal/confine` opens a file*; exporting `FS()` for the walk's convenience would
  reopen that as a matter of manners.
- **The floor is what prunes `.git/`, and it prunes it as a rule rather than as a special
  case.** Both SPECS and specs 11 describe the fallback as "`fs.WalkDir` with `.git/`
  skipped" — a hand-written exception in the walk. Going through `Resolve` deletes the
  exception: `.git` is already on the sensitive-file floor, so the same rule that refuses
  reading `.git/config` refuses descending into `.git`, and a `credentials/` directory is
  pruned by the same line without anyone having thought about it here.

## Alternatives tried

- **Export `Scope.FS()` and keep `fs.WalkDir(fsys, ".")`, filtering the results through
  `Resolve` afterwards.** Rejected on the leak above: filtering results is not the same as
  not reading, and the walk still traverses ungranted subtrees to produce the names it
  then discards.
- **Export a sub-FS per granted subtree (`fs.Sub(root.FS(), subtree)`).** This fixes the
  starting point and nothing else: the handle handed out still reads on the root's
  authority alone, so the floor inside that subtree becomes advice — the exact hole this
  epic's own `--allow .` criterion is written against.
- **Keep the walk inside `internal/confine` entirely, as `Scope.WalkFiles()`.** Viable,
  and rejected on layering: `list` and `search` need one enumeration that is *git's*
  answer first and a walk only as a fallback, and putting half of that mechanism in the
  confinement package would split the "what is in this repository" question across two
  packages. The primitives are in `confine`; the enumeration policy is in `repo`.

## Revisit when

`os.Root` grows a `ReadDir`, or an `fs.FS` implementation that accepts a filter — either
would let `Scope.ReadDir` shrink to a call rather than an open-list-close. Also revisit if
a later tool wants a genuine `fs.FS` for something the standard library only offers over
one (`fs.Glob`, `fs.WalkDir` proper): the answer would then be an `fs.FS` implemented
*by* `Scope`, gated on every `Open`, rather than the root's own.

Nothing here changes what SPECS was protecting — one confinement mechanism for all three
file tools — so the amendment SPECS needs is to the sentence's type, not its claim.

## Evidence

- `internal/confine/root.go` (`Stat`, `ReadDir`), `internal/repo/enumerate.go`
  (`walkFiles`, `walk`, `confined`).
- `TestScope_ReadDir_RefusesDirectoriesTheThreeRulesRefuse`
  (`internal/confine/enumerate_test.go`) is the first bullet as a test: a directory
  outside the allow-list and a directory on the floor are both refused by the same call a
  walk makes.
- `TestFiles_TheFallbackWalkStaysInsideTheAllowedSubtrees` and
  `TestFiles_OutsideAGitRepositoryWalksTheFilesystemInstead`
  (`internal/repo/fallback_test.go`) cover the walk's starting point and the `.git/`
  omission the floor produces.
- [Issue 02](../issues/02-enumeration-and-list-tool.md), whose Scope names `Root.FS()`;
  [issue 01's drift record](01-wire-path-validation-before-os-root.md), which is where
  `Scope` acquired its no-`FS()` shape.
