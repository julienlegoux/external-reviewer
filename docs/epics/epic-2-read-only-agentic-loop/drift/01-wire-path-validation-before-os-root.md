---
type: Drift record
title: "Wire-path spellings are validated before os.Root sees them, not only by it"
description: "SPECS and issue 01 say escape rejection is delegated to *os.Root and never re-implemented; the resolver nonetheless rejects non-wire-path spellings syntactically first, because the allow-list test, the two-OS criterion and git_read's no-I/O validation each need a cleaned relative path that os.Root cannot supply."
tags: [epic-2, drift]
timestamp: 2026-08-12T00:00:00Z
epic: 2
issue: 01
---

# Wire-path spellings are validated before os.Root sees them, not only by it

## Decided

[SPECS § Reading the repository](../../../planning/SPECS.md) states the enforcement
without a layer above it: *"Enforcement is `os.OpenRoot(repoPath)` at startup, and
**every** read goes through that `*os.Root`. The standard library then carries the whole
requirement"*, and names the thing not to build — *"the list a hand-rolled
`filepath.Clean` + prefix check would have to reproduce, and where it would fail first on
the platform SCOPE names as the development machine"*.
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md)
rejects `filepath.Abs` + `EvalSymlinks` + prefix comparison in the same terms, and
[EPIC_2](../EPIC_2.md)'s Notes repeat it. Issue 01's Scope makes it an instruction for
this PR specifically: the resolver *"rejects the path if it escapes the root (delegated to
`*os.Root`, not re-implemented)"*.

## Actual

`internal/confine.Scope.Resolve` runs `cleanWirePath` before anything else, and that
function refuses four spellings on its own authority, all as `ErrOutsideRoot`:

- the empty path;
- a path containing a backslash;
- a path beginning with `/`;
- a path whose second byte is `:` (`C:\Windows\win.ini`, and the drive-relative `C:x`);

and then, after `path.Clean`, a path that is `..` or begins with `../`. That last one is a
`Clean` plus a prefix test — the shape SPECS names.

Everything the kernel alone can see stays with `*os.Root` and is *not* enumerated:
symlinks whose target leaves the tree, absolute symlinks, Windows reserved device names,
and a directory swapped for a link between the resolution and the open. Those arrive back
through `openError` and are reported as the same `ErrOutsideRoot`, so the two layers give
one answer.

## Because

Three requirements in this same issue cannot be met by delegation alone.

- **The allow-list test needs a cleaned, relative wire path.** `os.Root` answers "may this
  be opened", never "which granted subtree is this in". `Scope.allows` compares against
  `docs`, `internal` and the like, and a comparison against `docs/../../outside` or
  `/etc/passwd` is meaningless. Something has to produce the canonical form, and
  `path.Clean` is what produces it.
- **The two-OS criterion.** Issue 01 requires an absolute path — *"`/etc/passwd` and
  `C:\Windows\win.ini`"* — refused on both matrix OSes. `os.Root` refuses both on Windows
  (verified: each returns `path escapes from parent`). On Linux `C:\Windows\win.ini` is a
  legal single-segment filename; `os.Root` refuses it only as a missing file, and would
  resolve it happily if such a file existed. Delegating alone makes the two platforms
  disagree about a criterion that says they must not.
- **`Resolve` must answer without I/O.** Issue 06 validates `git show <rev>:<path>`
  through this resolver before building a command, for paths that exist only in history.
  `os.Root` cannot answer for a path with no file behind it, so the policy half of the
  resolver has to be decidable on the string.

What SPECS warns against is a hand-rolled check standing *in place of* `os.Root` — the
version that has to reproduce the symlink and device-name list and fails on Windows. This
is the opposite arrangement: a check on the wire *vocabulary*
([CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md) defines a wire path
as repo-relative and `/`-separated on every platform), which guarantees that what reaches
`os.Root` is always a relative slash-separated name, with every kernel-visible escape left
to `os.Root` and reported under the same rule.

## Alternatives tried

- **Delegate purely, and let the allow-list compare raw strings.** Rejected: `docs/../../outside`
  passes a `strings.HasPrefix(p, "docs/")` test, so the allow-list would grant a path
  `os.Root` then refuses — the boundary would be the root, not the allow-list, which
  inverts specs 10's verdict.
- **Delegate purely, and let `C:\Windows\win.ini` be an ordinary Linux filename.**
  Rejected: it fails issue 01's acceptance criterion, which names that exact string, and
  leaves the two matrix OSes answering differently for one input.
- **Make `Resolve` open the file, so `os.Root` decides everything.** Rejected: it breaks
  issue 06, whose paths exist only in history, and turns a policy question into a
  filesystem round trip on every tool call.

## Revisit when

`os.Root` grows an exported way to canonicalise a name against a root without opening it,
or a sentinel that distinguishes its escape refusal from other errors. Either would let
`cleanWirePath` shrink: the first removes the reason for the `Clean`-and-prefix test, the
second removes `openError`'s default branch, which today reports every non-absence,
non-permission refusal as `ErrOutsideRoot` because Go exports nothing finer.

Also revisit if issues 02–06 find the vocabulary check refusing a path a tool legitimately
needs to name. Nothing in this PR's tests does, but they are the resolver's own tests, not
four tools' worth of real traffic.

## Evidence

- `internal/confine/allow.go` (`cleanWirePath`), `internal/confine/root.go`
  (`Resolve`, `openError`).
- `TestScope_Resolve_RefusesPathsThatAreNotInsideTheRoot` covers the syntactic half and
  `TestScope_Open_RefusesSymlinksThatLeaveTheRoot` /
  `TestScope_Open_WindowsReservedDeviceNames` cover the delegated half, all in
  `internal/confine/root_test.go`. The symlink cases skip on the Windows development
  machine, which refuses to create a symlink without the privilege, and run for real on
  Linux — verified on the remote host `scripts/test-remote.sh` uses.
- The `os.Root` error shapes quoted above were measured on the Windows development machine
  before the resolver was written, not assumed.
- [Issue 01](../issues/01-allow-list-and-root-confinement.md).
