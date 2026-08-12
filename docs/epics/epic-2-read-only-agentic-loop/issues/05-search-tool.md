---
type: Issue
title: "Add the search tool with surrounding context lines"
description: "Add search: an in-process RE2 regex over the confined enumeration, returning path:line:text with context lines, capped at 200 matches with the truncation announced."
tags: [epic-2]
timestamp: 2026-08-12T09:30:00Z
epic: 2
issue: 05
slug: search-tool
size: L
status: pr-open
gh_issue: 13
gh_pr: 53
resource: https://github.com/julienlegoux/external-reviewer/issues/13
depends_on: [3]
---

# Add the search tool with surrounding context lines

## Summary

`search` is how the reviewer finds things without knowing the layout, and it carries
`context` lines because a match without its surroundings usually forces a follow-up
`read_file` to judge it — the same reasoning that made `read_file` batched. Matching is
in-process `regexp` (RE2, so a model-supplied pattern cannot backtrack catastrophically)
over issue 02's enumeration, read through the confined root. No ripgrep: a subprocess
reading the filesystem on its own authority sits outside the root handle.

## Scope

- Tool `search`, parameters `pattern` (regex), `path` (subtree, default `.`), `glob`
  (file filter, optional), `max_results` (default 100, capped at 200) and `context`
  (lines before and after each match, default 2).
- Results as `path:line:text`, one match per line, with context lines rendered so they
  are distinguishable from matches; repo-relative `/`-separated paths.
- Candidate files come from issue 02's enumeration, so what the reviewer can glob and
  what it can grep never disagree — gitignored files are absent from both.
- Every candidate re-resolved through issue 01's helper before it is read: outside the
  allow-list and floor-matching files never appear in results, and the `path` parameter
  itself is refused with a rule-naming message when it points outside the allow-list.
- `regexp.Compile` failure returns a tool error saying the pattern was invalid — a
  message, not a run failure.
- Truncation always announced: `[truncated: showing 200 of 431 matches]`.
- Content streamed and matched line by line to the cap rather than read whole.
- Its `ai.Tool` declaration beside the implementation, `snake_case` schema fields, with a
  standalone description covering the path form, the context lines and what truncation
  means.

## Out of scope

- Ripgrep or any other subprocess matcher — rejected on confinement grounds, not speed
  ([specs 11](../../../planning/specs/11-tool-implementation-strategy.md)).
- Fuzzy or semantic search; PCRE features RE2 does not implement (backreferences,
  lookaround). A pattern using them fails to compile and returns a tool error, which is
  the intended behaviour.
- Parallel matching. Sequential tool execution, no concurrency
  (SPECS § Distribution & operations).
- Result ranking. Matches are returned in enumeration order.

## Acceptance criteria / Definition of done

- [ ] Written test-first and black-box, table-driven, against `t.TempDir()` git
      repositories.
- [ ] A pattern matching lines in several files returns `path:line:text` for each, in
      enumeration order, with `/`-separated repo-relative paths on both matrix OSes.
- [ ] `context: 2` returns two lines before and after each match, visually distinct from
      the match line; `context: 0` returns match lines only.
- [ ] Matches at the first and last line of a file return the available context without
      error or padding.
- [ ] `glob` restricts candidates (e.g. `**/*.go` excludes a matching `.md` file), and
      `path` restricts the subtree searched.
- [ ] A `path` outside the `--allow` subtrees is refused with a message naming the rule;
      a file inside the root but outside the allow-list never appears in results.
- [ ] A file matching the sensitive-file floor is never searched or reported, including
      with `--allow .`.
- [ ] A gitignored file containing the pattern does not appear in results.
- [ ] An invalid regex returns a tool error naming the pattern problem, and — asserted
      end-to-end through issue 03's loop — leaves the run alive so the model can retry
      with a corrected pattern.
- [ ] More matches than the cap ends with `[truncated: showing 200 of N matches]`
      carrying both real numbers; `max_results` above 200 is clamped.
- [ ] A pathological pattern (e.g. `(a+)+$`) against a long non-matching line completes
      promptly, demonstrating RE2 rather than a backtracking engine.
- [ ] `golangci-lint run` passes; CI green on ubuntu-latest and windows-latest.

## Relevant files / areas

No existing code for this yet. Expected paths:

- `internal/tools/search.go`, `internal/tools/search_test.go` (`package tools_test`)
- `internal/tools/registry.go` — registration
- `internal/repo/enumerate.go` — the candidate set from issue 02, called, not
  re-implemented

Governing decisions:
[specs 09 — Read-only tool contract](../../../planning/specs/09-read-only-tool-contract.md),
[specs 11 — Tool implementation strategy](../../../planning/specs/11-tool-implementation-strategy.md),
[SPECS § Reading the repository](../../../planning/SPECS.md).

## Dependencies

- Blocked by: [03 — Drive the multi-turn loop](/epic-2-read-only-agentic-loop/issues/03-multi-turn-loop-and-dispatch.md)
  (and through it [02](/epic-2-read-only-agentic-loop/issues/02-enumeration-and-list-tool.md),
  whose enumeration this consumes).
- Blocks: [07 — Run a real review by hand](/epic-2-read-only-agentic-loop/issues/07-hand-run-measurements.md).

## PR size note

Sized **L**: target ~650 changed lines — five parameters, streamed line-by-line matching,
context-window rendering at file edges, the enumeration and confinement filters, and a
test per parameter. It stays whole: `context` is why this tool exists rather than being a
`read_file` follow-up, and shipping matching without it is shipping the version the epic
rejected. If it passes ~1000, the context rendering and its edge cases are the split.
