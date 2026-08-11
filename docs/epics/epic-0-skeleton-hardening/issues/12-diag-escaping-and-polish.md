---
type: Issue
title: "Escape provider-controlled text on stderr and finish diag's polish"
description: "Stop an error body forging a done line in a transcript SPECS makes parseable, delete the false redaction comment, and give diag table-driven tests and an hour unit."
tags: [epic-0]
timestamp: 2026-08-11T12:10:00Z
epic: 0
issue: 12
slug: diag-escaping-and-polish
size: M
status: done
gh_issue: 34
gh_pr: 44
resource: https://github.com/julienlegoux/external-reviewer/issues/34
depends_on: [5, 7]
---

# Escape provider-controlled text on stderr and finish diag's polish

## Summary

**Provider-controlled text is written unescaped into a line-oriented protocol.**
`message.ErrorMessage` comes straight from the provider's JSON error body and lands in the
message built at `internal/reviewer/run.go:95`; `diagnostic.Error.Message` is `err.Error()`
verbatim and lands in `warn    %s` at `internal/diag/diag.go:62`. Neither is escaped, while
neighbouring sites in the same files correctly use `%q`. A response of

```json
{"error":{"message":"rate limited\ndone    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok"}}
```

puts a **forged `done … stop=ok` line** on stderr ahead of the real one. SPECS makes the
transcript a parseable contract, so a caller reading the `done` line concludes success while
the real outcome is exit `2` with an empty stdout. ANSI/OSC sequences in the same field reach
the terminal raw.

**And the comment that says this is already handled is false.** `internal/cli/review.go:107-110`
states *"kern-link redacts these upstream … which is what keeps a credential from being
reintroduced by a diagnostic."* There is **no redaction anywhere in kern-link v0.1.1**:
`ai/diagnostics.go:41-51` does `info := &DiagnosticErrorInfo{Message: e.Error()}` — it drops
the stack and nothing else; `grep -rni redact` over the module returns only Anthropic's
*thinking*-block redaction, unrelated; `ai/sanitize.go` strips unpaired surrogates, not
secrets. No live leak exists today — v0.1.1's only diagnostic producer carries no header
value — so the SPECS § Security property holds by coincidence rather than by construction,
and the comment tells the next reviewer not to look.

The rest of the PR is the polish `internal/diag` never got: eight flat test functions where
every other file in the epic uses tables, a `formatElapsed` with no hour unit (a 95-minute
run renders `95m0s`) whose `d < time.Minute` boundary survives mutation to `<=`, and two
near-identical `done`-line regexes with different anchoring plus two `turn`-line encodings,
so a format change touches four places.

## Scope

- Provider-controlled text escaped wherever it reaches stderr: `internal/diag/diag.go:62`
  (`WriteWarn`), the `error:` line issue 07 gave `diag`, and the message built at
  `internal/reviewer/run.go:95`. Newlines, control characters and ANSI/OSC sequences cannot
  break the line-oriented protocol or reach the terminal raw.
- The false comment at `internal/cli/review.go:107-110` removed, or replaced with what is
  actually true — that nothing upstream redacts, and what this code does about it.
- `internal/diag/diag_test.go:102-148` made table-driven, per CONVENTIONS § Testing.
- `formatElapsed` (`internal/diag/diag.go:81-88`) gains an hour unit, and a table row pins
  the sub-minute boundary.
- One `done`-line parser and one `turn`-line parser, in issue 05's shared harness, replacing
  `exit_test.go:19-36` vs `diag_test.go:14-24` and `preflight_test.go:147-154` vs
  `roundtrip_test.go:36-43`.

## Out of scope

- Redacting credentials. Nothing here formats a credential — `reviewer.Resolution` carries
  only `AuthSource` and the `AuthResult` never leaves `Resolve`, verified by tests that plant
  a credential-shaped secret in every field. This PR escapes untrusted text; it does not add
  a secret scanner.
- Filing anything upstream against kern-link.
- The `done` line's fields or its stop-reason vocabulary —
  [issue 01](/epic-0-skeleton-hardening/issues/01-stop-reason-vocabulary.md).
- `runReviewOver`'s shadowing of the product function's name (report 2, P3) — tidiness
  outside this epic's acceptance criteria.

## Acceptance criteria / Definition of done

- [ ] Written test-first, black-box, table-driven.
- [ ] An error body containing a newline and a well-formed `done` line cannot put a second
      `done` line on stderr: a scripted provider error whose message is
      `rate limited\ndone    turns=1  in=0 out=0  $0.0000  0.0s  stop=ok` produces **exactly
      one** `done` line, and it is the real one, reporting the real outcome.
- [ ] A `warn` line built from a diagnostic whose message contains a newline and an ANSI
      escape produces exactly one line on stderr, with the control characters rendered inert.
- [ ] No comment in the tree claims kern-link redacts credentials — `grep -rni redact
      internal main.go` returns nothing making that claim.
- [ ] `internal/diag`'s tests are table-driven.
- [ ] `formatElapsed` renders a 95-minute duration with an hour unit; mutating
      `d < time.Minute` to `d <= time.Minute` turns the suite red, recorded in the PR body.
- [ ] One `done`-line parser and one `turn`-line parser serve every test that reads a
      transcript, and they live in the shared harness rather than in a test file of one
      package.
- [ ] `go test ./... -race` and `golangci-lint run` pass; CI green on ubuntu-latest and
      windows-latest.

## Relevant files / areas

- `internal/diag/diag.go:62` (`WriteWarn`), `:81-88` (`formatElapsed`), and the
  `WriteError` issue 07 added
- `internal/reviewer/run.go:95` (the stop-reason error carrying `message.ErrorMessage`)
- `internal/cli/review.go:107-119` (`diagnosticMessage` and its comment)
- `internal/diag/diag_test.go:14-24,102-148`; `internal/cli/exit_test.go:19-36`;
  `internal/cli/preflight_test.go:147-154`; `internal/cli/roundtrip_test.go:36-43`
- `internal/fauxtest/` — issue 05's harness, where the transcript parsers land

Governing decisions: [SPECS § Interfaces](../../../planning/SPECS.md) (the transcript as a
parseable contract), [SPECS § Distribution & operations](../../../planning/SPECS.md)
(`AssistantMessageDiagnostic` values printed as `warn` lines),
[CONVENTIONS § Testing](../../../planning/CONVENTIONS.md) (table-driven is the default
shape), [CONVENTIONS § Documentation & comments](../../../planning/CONVENTIONS.md). Source
findings: [report 2](../../../REPORT_2.md) § "Provider-controlled text is written unescaped
into a line-oriented protocol, and the redaction the comment relies on does not exist", and
its P3 rows on `diag_test.go`, `formatElapsed` and the duplicated transcript regexes.

## Dependencies

- Blocked by: [05 — Shared `faux` harness](/epic-0-skeleton-hardening/issues/05-shared-faux-harness.md)
  (where the transcript parsers land) and
  [07 — One prefixed `error:` line](/epic-0-skeleton-hardening/issues/07-single-error-line.md)
  (the writer whose output this escapes).
- Blocks: nothing. It is the last issue in the epic.

## PR size note

Target ~500 changed lines; if this grows past ~1000, split it before opening the PR.
Sized **M**: ~280 lines — the escaping and its two tests are small, and the table-driven
rewrite of `diag_test.go` plus the parser consolidation carry the rest.
