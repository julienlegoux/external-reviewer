---
type: Reference
title: "Epic 2 — Hand-run measurements"
description: "What four hand-launched reviews on openai-codex/gpt-5.5 actually cost in turns, tokens, context, cost and wall clock, and what those numbers suggest — not decide — about Epic 3's loop caps."
tags: [epic-2, measurements]
timestamp: 2026-08-12T17:54:09Z
epic: 2
---

# Epic 2 — Hand-run measurements

Epic 2's second deliverable is not code. Nobody knew how many turns a review takes, how
much context it accumulates or how long it runs, and
[Epic 3](/epic-3-reviewer-selection-integration/EPIC_3.md)'s loop caps and
[SCOPE](../../planning/SCOPE.md)'s third success criterion both depend on numbers that
only exist once the loop runs against a real model. No automated test can stand in for
it: the `faux` provider scripts its responses, so it measures the harness rather than a
reviewer.

This records four hand-launched runs — three completed, one killed by the provider — on
the one credentialed family, `openai-codex/gpt-5.5`, against this repository. Everything
below is **observation**. The cap values stay unset; setting them is Epic 3's decision,
made from this data.

## How to read the figures

- **`in=`** on a `turn` line is cumulative across the run, and each turn's number
  includes cached prompt tokens (`Input + CacheRead + CacheWrite`,
  `internal/reviewer/loop.go`). Every turn resends the whole conversation, so the
  **difference between two consecutive `in=` figures is that turn's prompt** — which is
  the number that matters for context overflow. The run totals below are the cumulative
  figure from the `done` line; the *context* column is the largest single-turn prompt,
  derived from the deltas.
- **`$`** is `kern-link`'s catalog price times tokens. The credential here is OAuth on a
  Codex subscription, so no per-token charge was billed for any of these runs. Read the
  figure as a proxy for quota consumed, never as an invoice.
- Wall clock is the `done` line's own elapsed time, measured from process start.

## The runs

| Run | Surface | `--allow` | Turns | Tool calls | in | out | Cost | Peak context | Wall clock | Stop |
|---|---|---|---|---|---|---|---|---|---|---|
| **A0** | small (as A) | `docs/planning/specs` | 2 | 1 | 1,566 | 24 | $0.0086 | 1,566 | 15.3s | `failed` |
| **A** | small — one documentation directory, 21 files / 93.7 KiB | `docs/planning/specs` | 4 | 3 | 48,891 | 3,693 | $0.3483 | 29,393 | 1m57s | `stop` |
| **B** | medium — this repository's `docs/` and `internal/`, 135 files / 964.3 KiB | `docs`, `internal` | 6 | 9 | 357,713 | 4,322 | $1.4989 | **99,163** | 2m12s | `stop` |
| **C** | large — Epic 2's merged diff `a4e6db9^...develop`, 58 files / 8,699 diff lines / 358.7 KiB, with the whole 144-file / 981.7 KiB working tree also readable | `.` | 12 | 22 | 566,663 | 7,568 | $1.4130 | 98,023 | 3m59s | `stop` |

All three completed runs ended on the model's own stop reason (`stop=stop`), wrote a real
markdown report to stdout (7,426 / 6,522 / 5,346 bytes), and exited `0`. The reports are
substantive rather than nominal: run A opened with six internal contradictions between
specification documents, run B found the `SPECS.md` CLI grammar that `internal/cli` does
not yet implement, and run C returned five claimed defects in `git_read` itself. Full
stderr transcripts and the reports are in the pull request that introduced this document.

### Command lines

The binary is `go build`'s output from this branch, whose only difference from `develop`
at `a5b52d7` is this issue's own bookkeeping. Each was launched by hand from a shell, one
at a time, with the author watching the `turn` lines.

```
# A0 and A — the same invocation run twice, after A0 died on a provider error
external-reviewer review --allow docs/planning/specs \
  --prompt "Review the specification documents in this directory as an outside reviewer. Identify internal contradictions between the specs, decisions that are stated but never justified, and gaps an implementer would trip over. Cite the files and sections you rely on. Answer in markdown." \
  D:/Project/external-reviewer

# B
external-reviewer review --allow docs --allow internal \
  --prompt "You are an outside reviewer of this Go project. Check whether the code under internal/ actually implements what the planning documents under docs/planning/ decided, and whether the epic and issue documents under docs/epics/ describe the code that exists. Report the divergences you can evidence, citing file paths and line numbers. Answer in markdown." \
  D:/Project/external-reviewer

# C
external-reviewer review --allow . \
  --prompt "You are an outside reviewer. Review the complete merged diff of Epic 2 of this repository: the revision range is a4e6db9^...develop (58 files, about 7700 added lines). Read the diff itself through git history, not by reading the current files, and work through all of it rather than sampling. Report the defects, risks and inconsistencies the diff introduces, citing file and hunk. Answer in markdown." \
  D:/Project/external-reviewer
```

A0's prompt differed from A's by one word — a typo fix (`a implementer` → `an
implementer`) made between the failed attempt and the retry. Nothing else changed.

### Per-turn context growth

The delta between consecutive `in=` figures — the actual prompt size at each turn:

| Turn | A | B | C |
|---|---|---|---|
| 1 | 1,566 | 1,585 | 1,607 |
| 2 | 1,869 | 3,275 | 2,503 |
| 3 | 16,063 | 71,996 | 2,599 |
| 4 | 29,393 | 90,831 | 3,270 |
| 5 | — | 90,863 | 3,469 |
| 6 | — | 99,163 | 34,911 |
| 7 | — | — | 65,792 |
| 8 | — | — | 78,209 |
| 9 | — | — | 85,144 |
| 10 | — | — | 93,202 |
| 11 | — | — | 97,934 |
| 12 | — | — | 98,023 |

Two shapes, and the difference is the tool the run leaned on. B climbs in three large
steps, because `read_file` answers a batch of a dozen paths at once. C climbs almost
flat for five turns and then in ~30k steps, because `git_read` caps every answer at
2,000 lines (`MaxGitReadLines`) — the run's early turns cost nothing because their calls
were *refused* (see below), and its later ones each add one capped diff or one whole
file.

## Context overflow: not hit

**No run hit context overflow, and no run came near a provider error of any kind on
context.** The largest single-turn prompt observed anywhere is **99,163 tokens** (run B,
turn 6, the turn that wrote the report). The tested ceiling is therefore:

- **Surface in reach**: the entire repository — `--allow .`, 144 tracked files,
  981.7 KiB — with an 8,699-line, 358.7 KiB merged diff as the review target (run C).
- **Material actually pulled in**: 14 whole source files via `git show <rev>:<path>`,
  plus two 2,000-line diff chunks, plus `--stat` and `--name-only` summaries.
- **Peak prompt reached doing it**: 98,023 tokens.

That is the *tested* ceiling, not the model's. Nothing here establishes what happens
above ~100k, because neither run got there: in both B and C the model stopped reading
and wrote its report at around 98–99k, of its own accord, with no bound telling it to.

The reason the heaviest read in the pipeline did not overflow is that the tool caps
bound each answer before the context sees it — `MaxGitReadLines` 2,000,
`MaxListPaths` 200, `MaxReadLimit` 5,000, `MaxSearchMatches` 200. Measured cost of one
capped `git_read` answer over dense diff material: **~31k tokens** (run C, turn 5→6
delta 31,442 and turn 6→7 delta 30,881). Three such answers is a full context on their
own, which is the number Epic 3 should size a cap against rather than the surface size.

Truncation announced itself both times it bit, as the contract requires — run C, turn 5
asked for `git diff --unified=40 a4e6db9^...develop` (10,914 lines, capped at 2,000) and
turn 6 for the same range at `--unified=0` (8,292 lines, capped at 2,000).

## Tool coverage against a real model

All four tools were exercised by a real model's tool-calling, not only by `faux`:

| Tool | Where | Forms observed |
|---|---|---|
| `list` | A turn 1, B turns 1 and 4 | no arguments; `pattern="**/*"`; `pattern="*"` |
| `read_file` | A turns 2–3, B turns 2, 3 and 5 | batches of 5–13 paths; `limit=5000`; `limit=180 offset=300` |
| `search` | B turn 3 | `pattern` as a 9-branch alternation, `path="internal"`, `context=2`, `max_results=200` |
| `git_read` | C, every turn | `diff` with `--stat`, `--name-only`, `--unified=0/40/60/80`; `show` with the `<rev>:<path>` object form, 14 times |

`git_read`'s `log` and `status` subcommands were **not** reached by any run; nothing here
says how a real model uses them.

## Failure behaviour, observed rather than argued

- **A failing turn ends the run, and the `done` line still lands.** Run A0 died on turn 2
  on a provider-side error (`Codex error: An error occurred while processing your
  request`, request ID `5febbe1f-…`) — exit `2`, `stop=failed`, one `error:` line, empty
  stdout, `done` line present. The identical invocation retried immediately succeeded
  (run A), so this was transient rather than a defect in the request. **One in four hand
  runs failed this way**, which is the frequency Epic 3 should weigh when deciding
  whether the caller retries.
- **A failing tool is only a turn.** Run C's turn 2 asked for
  `diff --find-renames --find-copies --unified=80 a4e6db9^...develop -- internal`; the
  literal `--` is refused by `git_read` by design. The run stayed alive, the model read
  the refusal, and turn 3 came back with a different, legal request. The same refusal at
  turn 11 was followed by the model writing its report instead. This is the two-levels
  design working against a real model, not a scripted one.

## One defect this measurement exposed

Run C spent five of its twelve turns failing to scope a diff, and the cause is a real
gap rather than the model fumbling:

- `git_read` refuses a literal `--` (it appends the granted subtrees as pathspecs
  itself), so the model cannot write `diff <range> -- <path>`.
- Without `--`, the model's natural fallback is `diff <range> <path>` — run C, turn 4,
  three times. But `git_read` then appends its own `-- <subtrees>`, and everything before
  a `--` is a revision to git. Reproduced by hand:
  `git diff --unified=60 a4e6db9^...develop internal/confine -- .` →
  `fatal: bad revision 'internal/confine'`.

So **a path-scoped diff is unreachable through `git_read`**, and the only diff a reviewer
can obtain is the whole-repository one — which is exactly the read that hits the
2,000-line cap. This inflated run C's turn count and forced it to reconstruct the diff
file by file through `show`. It is out of scope here (this document changes no `.go`
file) and belongs in its own issue — filed at Epic 2's close as
[#60](https://github.com/julienlegoux/external-reviewer/issues/60), on Epic 3's
milestone, since Epic 3 sizes its caps from numbers this gap inflates.

## Observations for Epic 3

Input to a decision, **not** the decision. Epic 3 sets the values.

- **Turn cap — a plausible range is 25–40.** Observed: 4 (A), 6 (B), 12 (C). The heaviest
  run used 12, and only because the path-scoped-diff gap above forced it to read the diff
  the long way; with that fixed the same review is plausibly 6–8. A ceiling at 25–40
  leaves 2–3× headroom over anything observed while still stopping a runaway well before
  it costs an hour. Nothing observed here justifies a cap in single digits.
- **Wall-clock deadline — a plausible range is 15–25 minutes.** Observed: 1m57s, 2m12s,
  3m59s. But the distribution matters more than the total: reading turns took 2.4s–16.7s,
  while the final report-writing turn took **1m35s–2m5s** in all three runs. A deadline
  is therefore not safely set below about 5 minutes at any surface size — it would kill
  runs mid-report, discarding everything, since the report is written in one turn at the
  end. 15–25 minutes is 4–6× the heaviest run observed.
- **Cost ceiling, if one is wanted at all — $5–10.** Observed: $0.35 (A), $1.50 (B),
  $1.41 (C) at catalog prices, $3.27 for the whole exercise. Note that C reviewed a
  larger surface than B for slightly less, so cost tracks *turns and context* rather than
  surface size. On the current subscription credential nothing is billed per token, which
  is why [SPECS § Departures from SCOPE](../../planning/SPECS.md) puts success criterion
  3 on wall clock and quota headroom rather than spend — a cost bound is the least
  informative of the three here.
- **Context is not the binding constraint at this project's scale.** A whole-repository
  allow-list over ~1 MiB, reviewing an 8,699-line diff, peaked at 98k prompt tokens
  because the per-answer caps hold. Risk 3 stays unmitigated and untested above ~100k,
  but it is not what a review of a repository this size runs into first.
