---
type: Drift record
title: "git_read scopes status too, and validates <rev>:<path> in every subcommand"
description: "specs 10 confines git_read by pathspec-scoping log and diff and validating show's <rev>:<path>, and states that status is unaffected; the implementation scopes status with the same pathspecs and validates the object form in every subcommand's arguments, because an unscoped status names every path in the repository and `git diff <blob> <blob>` prints two files' contents."
tags: [epic-2, drift]
timestamp: 2026-08-12T09:30:00Z
epic: 2
issue: 06
---

# git_read scopes status too, and validates `<rev>:<path>` in every subcommand

## Decided

[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md)
closes the `git show HEAD:.env` hole with exactly three measures, the third of which is a
statement that one subcommand needs none:

> - `show`'s `<rev>:<path>` form has its path validated against the allow-list and the
>   floor below before the command is built; a refused path never reaches `git`.
> - `log` and `diff` get the allowed subtrees appended as pathspecs (`-- <paths>`), so
>   history is scoped the same way the working tree is.
> - **`status` is unaffected.**

## Actual

`internal/tools/git_read.go` applies both measures more widely than specs 10 describes:

- **`status` is pathspec-scoped**, exactly as `log` and `diff` are. `git status -- docs`
  where `docs` is the granted subtree.
- **The `<rev>:<path>` form is validated in every subcommand's arguments**, not only
  `show`'s, and in all three of git's spellings for it: `<rev>:<path>`, the index form
  `:<path>`, and the staged index form `:<stage>:<path>`.

Both are what [issue 06](../issues/06-git-read-tool.md) specifies — its Scope names the
`status` pathspecs and its acceptance criteria assert them — so the code follows the issue
and the issue departs from the decision it cites.

## Because

Two facts measured against real git on the development machine, not reasoned from the
documentation.

- **An unscoped `status` hands over every path name in the repository**, including the
  ones the allow-list refuses and the ones the sensitive-file floor exists to hide, plus
  their modification state. It is the one of the four subcommands that leaks path names
  without reading a file:

  ```
  $ git -C <fixture> status --porcelain
   M docs/notes.md
   M secret/creds.txt
  $ git -C <fixture> status --porcelain -- docs
   M docs/notes.md
  ```

  specs 10's own standard for the other three — "history is scoped the same way the
  working tree is" — is not met while `status` reports the working tree unscoped.

- **`git diff <blob> <blob>` prints both blobs' contents**, so the `<rev>:<path>` form is
  a file-reading form in `diff` as much as in `show`:

  ```
  $ git -C <fixture> diff HEAD:.env HEAD:docs/notes.md
  diff --git a/.env b/docs/notes.md
  -KEY=1
  +notes
  ```

  Validating it only in `show` would have left the hole specs 10 closes open in its second
  doorway. Appending pathspecs does not close it either: `git diff <blob> <blob> -- docs`
  is a usage error (exit 129), which is why a call naming an object gets no pathspecs and
  is confined by path validation alone — and why mixing an object with a plain revision in
  one call is refused rather than run unscoped.

## Alternatives tried

- **Leave `status` unscoped, as specs 10 says.** Rejected on the measurement above: it
  makes the file-level confinement decorative for path *names*, which is the same class of
  hole — one call, no file opened — that specs 10 was amended to close.
- **Scope `show` with pathspecs instead of validating its paths.** Rejected: verified that
  `git show HEAD:.env -- docs` still prints `.env`. Pathspecs confine commit traversal, not
  object resolution, so the two measures are not interchangeable.

## Revisit when

specs 10 is next edited. The disposition this record proposes is **accepted**, with specs
10's third bullet amended from "`status` is unaffected" to `status` taking the same
pathspecs, and its `show` bullet widened to "any `<rev>:<path>` argument, in any
subcommand". Nothing about the decision's reasoning changes — only the enumeration of
where it applies, which was written before `git diff <blob> <blob>` and `status
--porcelain` were measured.

## Evidence

- `internal/tools/git_read.go` (`gitReadArgv`, `objectPath`, `objectRefusal`).
- `TestGitRead_ScopesStatusToTheGrantedSubtrees` asserts the `status` half separately from
  `log` and `diff` for exactly this reason, and `TestGitRead_RefusesABlobPairOnTheFloor`
  the `diff` half, both in `internal/tools/git_read_test.go`. Removing the appended
  pathspecs makes the three scoping tests fail with the ungranted subtree's commit and
  paths in the output — verified by mutation, not assumed.
- [Issue 06](../issues/06-git-read-tool.md), PR #55.
