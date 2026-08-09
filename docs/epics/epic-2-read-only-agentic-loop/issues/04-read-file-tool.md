---
type: Issue
title: "Add the batched read_file tool"
description: "Add read_file taking an array of paths with offset and limit, returning each file under its own header with a bad path reported inline while the rest still return."
tags: [epic-2]
timestamp: 2026-08-09T07:50:00Z
epic: 2
issue: 04
slug: read-file-tool
size: M
status: open
gh_issue: 12
resource: https://github.com/julienlegoux/external-reviewer/issues/12
depends_on: [3]
---

# Add the batched read_file tool

## Summary

Reading is the reviewer's main verb, and turns are the scarce resource, so `read_file`
takes a **list** of paths: eight issue files in one turn instead of eight. The property
that makes the batch worth having is partial success — a batch where one path was a typo
must still return the other seven, with the failure reported beside them rather than
failing the call.

## Scope

- Tool `read_file`, parameters `paths` (array of repo-relative `/`-separated strings),
  `offset` (1-based line, default 1) and `limit` (lines, default 2000, capped at 5000).
- Each file returned under its own header naming the path and the line range, with each
  line prefixed `<n>\t`.
- Every path resolved through issue 01's helper: outside the root, outside the
  allow-list, or matching the sensitive-file floor each produce a distinct refusal naming
  the rule.
- **A failure on one path is reported inline, under that path's header, while every other
  path in the batch still returns its content.** The call itself is not an error.
- Truncation always announced, per file: `[truncated: lines 1–2000 of 8104]`.
- Content streamed to the cap rather than read whole and trimmed — the one performance
  property that is a correctness property (SPECS § Distribution & operations).
- A file that is not valid UTF-8, or is a directory, returns an inline explanatory
  message rather than raw bytes or a crash.
- Its `ai.Tool` declaration beside the implementation, `snake_case` schema fields, with a
  description that stands alone and states explicitly that a batch failure is partial
  rather than total.

## Out of scope

- `search` and `git_read` — issues 05 and 06.
- Any write path, any `os.OpenFile` — the forbidigo guard rejects them and the tool reads
  through `*os.Root` only.
- Binary-file detection beyond the UTF-8 check above; there is no MIME sniffing.
- Tolerating OS-native separators on the wire. Wire paths are `/`-separated on every
  platform (CONVENTIONS § Paths and platforms), so a backslash-separated path is refused
  like any other non-conforming path rather than normalised — hand-normalising it would
  make a legal Linux filename containing a backslash unreachable, and would make this
  tool accept what the other three do not.
- Caching file contents between calls.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box, table-driven, against `t.TempDir()`
      repositories.
- [ ] A batch of three valid paths returns three headed sections, in the order requested,
      with `<n>\t`-prefixed lines.
- [ ] **A batch of three paths where one does not exist returns the other two in full**
      and an inline message under the missing path's header; the tool call is not
      reported as an error.
- [ ] A batch mixing a valid path, a path outside the allow-list and a path matching the
      floor returns the valid content plus two refusals whose messages name *different*
      rules.
- [ ] `offset`/`limit` return exactly the requested window, 1-based, with the header
      naming the range.
- [ ] A file longer than the cap ends with `[truncated: lines 1–5000 of N]` carrying both
      real numbers; a file shorter than the cap carries no truncation line.
- [ ] A `limit` above the 5000-line cap is clamped, and the truncation line reflects the
      clamp.
- [ ] A directory path and a non-UTF-8 file each return an inline explanatory message,
      not raw bytes and not a panic.
- [ ] Paths in headers and in refusals are `/`-separated and repo-relative on both matrix
      OSes.
- [ ] End-to-end through issue 03's loop against the `faux` provider: a scripted
      `read_file` call over a batch containing one bad path leaves the run alive and the
      model's next turn is dispatched.
- [ ] `golangci-lint run` passes; CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/tools/read_file.go`, `internal/tools/read_file_test.go`
  (`package tools_test`)
- `internal/tools/registry.go` — registration
- `internal/confine/` — the resolution helper from issue 01, called, not re-implemented

Governing decisions:
[specs 09 — Read-only tool contract](../../../planning/specs/09-read-only-tool-contract.md),
[specs 10 — Repository confinement](../../../planning/specs/10-repository-confinement.md),
[SPECS § Reading the repository](../../../planning/SPECS.md),
[CONVENTIONS § Paths and platforms](../../../planning/CONVENTIONS.md).

## Dependencies

- Blocked by: [03 — Drive the multi-turn loop](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md).
- Blocks: [07 — Run a real review by hand](/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md).

## PR size note

Sized **M**: target ~400 changed lines — one tool over issue 01's resolver, with the
partial-success behaviour and the truncation line carrying most of the test weight. The
loop it plugs into already exists, so there is nothing to integrate. If it passes ~700,
something outside this tool's contract has been taken on.
